package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/config"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/crypto/bcrypt"
)

const testOrigin = "https://example.com"
const testCallback = "https://client.example/callback"

type oauthHarness struct {
	t       *testing.T
	handler http.Handler
	session *http.Cookie
}

func newOAuthHarness(t *testing.T, db *database.Store) *oauthHarness {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.MinCost)
	s := &Server{db: db, config: config.Config{Username: "admin", PasswordHash: string(hash), JWTSecret: "test-secret", PublicOrigin: testOrigin}, attempts: map[string]attempt{}}
	h := &oauthHarness{t: t, handler: s.RegisterRoutes()}
	w := h.do("POST", "/api/auth/login", `{"username":"admin","password":"password"}`, map[string]string{"Origin": testOrigin, "Content-Type": "application/json"}, false)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	h.session = w.Result().Cookies()[0]
	return h
}

func (h *oauthHarness) do(method, target, body string, headers map[string]string, withSession bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	if withSession {
		r.AddCookie(h.session)
	}
	w := httptest.NewRecorder()
	h.handler.ServeHTTP(w, r)
	return w
}

func (h *oauthHarness) register(uris ...string) string {
	body, _ := json.Marshal(map[string]any{"redirect_uris": uris, "client_name": "Test <client>\u202e", "token_endpoint_auth_method": "client_secret_post"})
	w := h.do("POST", "/oauth/register", string(body), map[string]string{"Content-Type": "application/json"}, false)
	if w.Code != 201 {
		h.t.Fatal(w.Code, w.Body.String())
	}
	var out struct {
		ClientID   string `json:"client_id"`
		AuthMethod string `json:"token_endpoint_auth_method"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out.AuthMethod != "none" {
		h.t.Fatal("public client expected", out.AuthMethod)
	}
	return out.ClientID
}

func pkce(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func authorizeURL(clientID, redirect, challenge string) string {
	return "/oauth/authorize?" + url.Values{"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {redirect}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}, "state": {"xyz"}, "resource": {testOrigin + "/api/mcp"}}.Encode()
}

// consent loads the consent page and returns its signed request field.
func (h *oauthHarness) consent(clientID, redirect, challenge string) string {
	w := h.do("GET", authorizeURL(clientID, redirect, challenge), "", nil, true)
	if w.Code != 200 {
		h.t.Fatal(w.Code, w.Body.String())
	}
	page := w.Body.String()
	if !strings.Contains(page, "Test &lt;client&gt;") || strings.Contains(page, "\u202e") || w.Header().Get("X-Frame-Options") != "DENY" {
		h.t.Fatal("consent page must escape client name and deny framing")
	}
	_, rest, _ := strings.Cut(page, `name="request" value="`)
	request, _, _ := strings.Cut(rest, `"`)
	return request
}

func (h *oauthHarness) approve(request, decision string) *url.URL {
	form := url.Values{"request": {request}, "decision": {decision}}.Encode()
	w := h.do("POST", "/oauth/authorize", form, map[string]string{"Origin": testOrigin, "Content-Type": "application/x-www-form-urlencoded"}, true)
	if w.Code != http.StatusFound {
		h.t.Fatal(w.Code, w.Body.String())
	}
	u, _ := url.Parse(w.Header().Get("Location"))
	return u
}

func (h *oauthHarness) token(form url.Values) (int, map[string]any) {
	w := h.do("POST", "/oauth/token", form.Encode(), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, false)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestOAuthDiscoveryAndAuthorize(t *testing.T) {
	h := newOAuthHarness(t, nil)
	w := h.do("GET", "/.well-known/oauth-protected-resource/api/mcp", "", nil, false)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"resource":"https://example.com/api/mcp"`) || !strings.Contains(w.Body.String(), `"authorization_servers":["https://example.com"]`) {
		t.Fatal(w.Code, w.Body.String())
	}
	w = h.do("GET", "/.well-known/oauth-authorization-server", "", nil, false)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"registration_endpoint":"https://example.com/oauth/register"`) || !strings.Contains(w.Body.String(), `"S256"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	w = h.do("POST", "/api/mcp", `{}`, map[string]string{"Content-Type": "application/json"}, false)
	if w.Code != 401 || !strings.Contains(w.Header().Get("WWW-Authenticate"), `resource_metadata="https://example.com/.well-known/oauth-protected-resource/api/mcp"`) {
		t.Fatal(w.Code, w.Header())
	}
	// The website session cookie never authenticates MCP.
	if w = h.do("POST", "/api/mcp", `{}`, map[string]string{"Content-Type": "application/json"}, true); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w = h.do("POST", "/api/mcp", `{}`, map[string]string{"Authorization": "Bearer " + h.session.Value}, false); w.Code != 401 {
		t.Fatal("website JWT accepted as MCP token", w.Code)
	}

	for _, bad := range []string{"http://evil.example/cb", "javascript:alert(1)", "https://x.example/cb#frag"} {
		body, _ := json.Marshal(map[string]any{"redirect_uris": []string{bad}})
		if w = h.do("POST", "/oauth/register", string(body), nil, false); w.Code != 400 {
			t.Fatal(bad, w.Code)
		}
	}
	clientID := h.register(testCallback, "http://127.0.0.1:3000/callback")
	challenge := pkce(strings.Repeat("v", 43))

	// Signed-out owners go through the website login and come back.
	w = h.do("GET", authorizeURL(clientID, testCallback, challenge), "", nil, false)
	if w.Code != http.StatusSeeOther || !strings.HasPrefix(w.Header().Get("Location"), "/login?redirect=%2Foauth%2Fauthorize%3F") {
		t.Fatal(w.Code, w.Header().Get("Location"))
	}
	// Unregistered or tampered clients and callbacks never redirect.
	if w = h.do("GET", authorizeURL(clientID+"x", testCallback, challenge), "", nil, true); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w = h.do("GET", authorizeURL(clientID, "https://evil.example/callback", challenge), "", nil, true); w.Code != 400 {
		t.Fatal(w.Code)
	}
	// Loopback callbacks may use any port.
	h.consent(clientID, "http://127.0.0.1:49152/callback", challenge)
	// Pre-consent errors never redirect, so registered clients cannot use
	// the authorize endpoint as an open redirect.
	w = h.do("GET", strings.Replace(authorizeURL(clientID, testCallback, challenge), "code_challenge_method=S256", "code_challenge_method=plain", 1), "", nil, true)
	if w.Code != 400 || w.Header().Get("Location") != "" || !strings.Contains(w.Body.String(), "PKCE") {
		t.Fatal(w.Code, w.Header().Get("Location"))
	}

	request := h.consent(clientID, testCallback, challenge)
	form := url.Values{"request": {request}, "decision": {"approve"}}.Encode()
	if w = h.do("POST", "/oauth/authorize", form, map[string]string{"Origin": "https://evil.example", "Content-Type": "application/x-www-form-urlencoded"}, true); w.Code != 403 {
		t.Fatal("cross-origin approval accepted", w.Code)
	}
	if u := h.approve(request, "deny"); u.Query().Get("error") != "access_denied" || u.Query().Get("state") != "xyz" || u.Query().Get("iss") != testOrigin {
		t.Fatal(u)
	}
	u := h.approve(request, "read")
	if u.Host != "client.example" || u.Query().Get("code") == "" || u.Query().Get("state") != "xyz" {
		t.Fatal(u)
	}
}

func testDB(t *testing.T) *database.Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL")
	}
	base, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "mcp_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = base.DB.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	db, err := database.Open(dsn + sep + "search_path=" + schema)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close(); base.DB.Exec("DROP SCHEMA " + schema + " CASCADE"); base.Close() })
	if err = db.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile("../../tests/fixtures/migration/workspace-v2.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Import(context.Background(), bytes.NewReader(fixture), database.Source{Store: "fixture", Key: "content", Checksum: uuid.NewString(), Archive: "test"}, false); err != nil {
		t.Fatal(err)
	}
	return db
}

type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r)
}

func TestMCPOverOAuth(t *testing.T) {
	db := testDB(t)
	run, err := database.PrepareSave("run", database.Entity{"id": "private-run", "title": "Private run", "runDate": "2026-09-19", "startedAt": "2026-09-19T10:00:00Z", "tags": []any{}, "body": "café thoughts"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Save(context.Background(), "run", run, nil); err != nil {
		t.Fatal(err)
	}
	h := newOAuthHarness(t, db)
	clientID := h.register(testCallback)

	// A client that omits redirect_uri at authorize may omit it at the token
	// endpoint too (RFC 6749 §4.1.3).
	implicitVerifier := strings.Repeat("c3", 30)
	implicitAuthorize := strings.Replace(authorizeURL(clientID, testCallback, pkce(implicitVerifier)), "redirect_uri="+url.QueryEscape(testCallback)+"&", "", 1)
	w := h.do("GET", implicitAuthorize, "", nil, true)
	_, rest, _ := strings.Cut(w.Body.String(), `name="request" value="`)
	implicitRequest, _, _ := strings.Cut(rest, `"`)
	implicitCode := h.approve(implicitRequest, "read").Query().Get("code")
	if status, out := h.token(url.Values{"grant_type": {"authorization_code"}, "code": {implicitCode}, "client_id": {clientID}, "code_verifier": {implicitVerifier}}); status != 200 {
		t.Fatal("omitted redirect_uri rejected", status, out)
	}

	verifier := strings.Repeat("a1", 30)
	code := h.approve(h.consent(clientID, testCallback, pkce(verifier)), "read").Query().Get("code")

	exchange := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {testCallback}, "client_id": {clientID}, "code_verifier": {verifier}}
	bad := url.Values{}
	for k, v := range exchange {
		bad[k] = v
	}
	bad.Set("code_verifier", strings.Repeat("b", 43))
	if status, out := h.token(bad); status != 400 || out["error"] != "invalid_grant" {
		t.Fatal("wrong PKCE verifier accepted", status, out)
	}
	other := h.register(testCallback)
	bad.Set("code_verifier", verifier)
	bad.Set("client_id", other)
	if status, out := h.token(bad); status != 400 || out["error"] != "invalid_grant" {
		t.Fatal("code accepted for another client", status, out)
	}
	status, tokens := h.token(exchange)
	if status != 200 || tokens["token_type"] != "Bearer" || tokens["scope"] != mcpScope {
		t.Fatal(status, tokens)
	}
	if status, out := h.token(exchange); status != 400 || out["error"] != "invalid_grant" {
		t.Fatal("authorization code replayed", status, out)
	}
	refresh := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tokens["refresh_token"].(string)}, "client_id": {clientID}}
	status, rotated := h.token(refresh)
	if status != 200 || rotated["refresh_token"] == tokens["refresh_token"] {
		t.Fatal(status, rotated)
	}
	if status, out := h.token(refresh); status != 400 || out["error"] != "invalid_grant" {
		t.Fatal("rotated refresh token reused", status, out)
	}
	// Refresh tokens are not access tokens.
	if w = h.do("POST", "/api/mcp", `{}`, map[string]string{"Authorization": "Bearer " + rotated["refresh_token"].(string)}, false); w.Code != 401 {
		t.Fatal(w.Code)
	}

	call := connectMCP(t, h, rotated["access_token"].(string), 6)
	overview, _ := call("get_career_overview", nil)
	goals, _ := overview["goals"].([]any)
	counts, _ := overview["counts"].(map[string]any)
	if len(goals) != 1 || goals[0].(map[string]any)["title"] != "Learn" || counts["note"] != float64(1) || counts["page"] != float64(2) {
		t.Fatal(overview)
	}
	list, _ := call("list_career_entries", map[string]any{"kind": "page", "limit": 1})
	if entries, _ := list["entries"].([]any); len(entries) != 1 || list["nextOffset"] != float64(1) {
		t.Fatal(list)
	}
	search, _ := call("search_career_context", map[string]any{"query": "café"})
	matches, _ := search["matches"].([]any)
	if len(matches) == 0 || matches[0].(map[string]any)["kind"] != "note" {
		t.Fatal(search)
	}
	for _, match := range matches {
		if match.(map[string]any)["kind"] == "audio_thought" {
			t.Fatal("audio thoughts must be opt-in", search)
		}
	}
	search, _ = call("search_career_context", map[string]any{"query": "café", "kind": "audio_thought"})
	if matches, _ = search["matches"].([]any); len(matches) != 1 || matches[0].(map[string]any)["id"] != "private-run" {
		t.Fatal(search)
	}
	if out, isErr := call("search_career_context", map[string]any{"query": "Learn", "kind": "goal"}); !isErr || !strings.Contains(out["error"].(string), "get_career_overview") {
		t.Fatal(out)
	}
	read, _ := call("read_career_entry", map[string]any{"kind": "note", "id": "systems/nested-note", "length": 10})
	text, _ := read["text"].(string)
	revision, _ := read["revision"].(string)
	if len([]rune(text)) != 10 || revision == "" || read["nextOffset"] != float64(10) || read["totalLength"].(float64) <= 10 {
		t.Fatal(read)
	}
	full, _ := call("read_career_entry", map[string]any{"kind": "note", "id": "systems/nested-note", "length": 20000})
	if body, _ := full["text"].(string); !strings.Contains(body, "<script>") || full["nextOffset"] != nil {
		t.Fatal("entry text should be complete and unescaped", full)
	}
	if out, isErr := call("read_career_entry", map[string]any{"kind": "note", "id": "missing"}); !isErr || !strings.Contains(out["error"].(string), "not found") {
		t.Fatal(out)
	}
	if out, isErr := call("list_career_entries", map[string]any{"kind": "secrets"}); !isErr {
		t.Fatal(out)
	}
	if out, isErr := call("update_career_entry", map[string]any{"kind": "note", "id": "systems/nested-note", "revision": revision, "fields": map[string]any{"title": "x"}}); !isErr || !strings.Contains(out["error"].(string), "read-only") {
		t.Fatal("read-only token wrote", out)
	}
}

type toolCall func(name string, args map[string]any) (map[string]any, bool)

// connectMCP opens a Go MCP client session with the bearer token and returns
// a helper that decodes structured results or the tool error text.
func connectMCP(t *testing.T, h *oauthHarness, token string, wantTools int) toolCall {
	t.Helper()
	srv := httptest.NewServer(h.handler)
	t.Cleanup(srv.Close)
	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL + "/api/mcp", HTTPClient: &http.Client{Transport: bearerTransport{token}}, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	tools, err := session.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != wantTools {
		t.Fatal(err, tools)
	}
	return func(name string, args map[string]any) (map[string]any, bool) {
		t.Helper()
		res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatal(name, err)
		}
		if res.IsError {
			return map[string]any{"error": res.Content[0].(*mcp.TextContent).Text}, true
		}
		var out map[string]any
		b, _ := json.Marshal(res.StructuredContent)
		_ = json.Unmarshal(b, &out)
		return out, false
	}
}

func TestMCPWriteTools(t *testing.T) {
	db := testDB(t)
	h := newOAuthHarness(t, db)
	clientID := h.register(testCallback)
	verifier := strings.Repeat("w9", 30)
	code := h.approve(h.consent(clientID, testCallback, pkce(verifier)), "write").Query().Get("code")
	status, tokens := h.token(url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {testCallback}, "client_id": {clientID}, "code_verifier": {verifier}})
	if status != 200 || tokens["scope"] != "career:read career:write" {
		t.Fatal(status, tokens)
	}
	call := connectMCP(t, h, tokens["access_token"].(string), 6)
	ctx := context.Background()

	company, isErr := call("create_career_entry", map[string]any{"kind": "company", "entry": map[string]any{"title": "Duck Corp!", "category": "Infra", "url": "https://duck.example", "slug": "ignored"}})
	if isErr || company["id"] != "duck-corp" || company["revision"] == "" {
		t.Fatal(company)
	}
	if out, isErr := call("create_career_entry", map[string]any{"kind": "company", "entry": map[string]any{"title": "Duck Corp", "category": "Infra", "url": "https://duck.example"}}); !isErr || !strings.Contains(out["error"].(string), "already exists") {
		t.Fatal("duplicate company created", out)
	}
	updated, isErr := call("update_career_entry", map[string]any{"kind": "company", "id": "duck-corp", "revision": company["revision"], "fields": map[string]any{"status": "applied", "body": "Applied via referral."}})
	if isErr || updated["revision"] == company["revision"] {
		t.Fatal(updated)
	}
	saved, err := db.Detail(ctx, "company", "duck-corp")
	if err != nil || saved.Entry["status"] != "applied" || saved.Entry["body"] != "Applied via referral." || saved.Entry["title"] != "Duck Corp!" {
		t.Fatal(saved, err)
	}
	// A stale revision never overwrites newer edits.
	if out, isErr := call("update_career_entry", map[string]any{"kind": "company", "id": "duck-corp", "revision": company["revision"], "fields": map[string]any{"status": "offer"}}); !isErr || !strings.Contains(out["error"].(string), "changed since") {
		t.Fatal("stale revision accepted", out)
	}
	if out, isErr := call("update_career_entry", map[string]any{"kind": "company", "id": "duck-corp", "revision": updated["revision"], "fields": map[string]any{"status": "hired"}}); !isErr || !strings.Contains(out["error"].(string), "invalid company enum") {
		t.Fatal("invalid status accepted", out)
	}
	if out, isErr := call("update_career_entry", map[string]any{"kind": "company", "id": "duck-corp", "revision": updated["revision"], "fields": map[string]any{"slug": "other"}}); !isErr || !strings.Contains(out["error"].(string), "cannot be changed") {
		t.Fatal("managed field changed", out)
	}
	if out, isErr := call("update_career_entry", map[string]any{"kind": "page", "id": "x", "revision": "r", "fields": map[string]any{"title": "x"}}); !isErr {
		t.Fatal("page kind writable", out)
	}

	goal, isErr := call("create_career_entry", map[string]any{"kind": "goal", "entry": map[string]any{"title": "Ship MCP writes", "startDate": "2026-10-01", "endDate": "2026-10-31", "steps": []any{map[string]any{"title": "Write tests"}}, "notes": []any{map[string]any{"body": "Started"}}}})
	if isErr {
		t.Fatal(goal)
	}
	goalID, _ := goal["id"].(string)
	g, err := db.Detail(ctx, "goal", goalID)
	steps, _ := g.Entry["steps"].([]any)
	if err != nil || g.Entry["status"] != "planned" || len(steps) != 1 || steps[0].(map[string]any)["done"] != false {
		t.Fatal(g, err)
	}
	step := steps[0].(map[string]any)
	step["done"] = true
	if out, isErr := call("update_career_entry", map[string]any{"kind": "goal", "id": goalID, "revision": g.Revision, "fields": map[string]any{"status": "active", "steps": []any{step, map[string]any{"title": "Deploy"}}}}); isErr {
		t.Fatal(out)
	}
	g, _ = db.Detail(ctx, "goal", goalID)
	steps, _ = g.Entry["steps"].([]any)
	if g.Entry["status"] != "active" || len(steps) != 2 || steps[0].(map[string]any)["id"] != step["id"] || steps[0].(map[string]any)["done"] != true {
		t.Fatal(g.Entry)
	}

	note, isErr := call("create_career_entry", map[string]any{"kind": "note", "entry": map[string]any{"title": "Interview prep", "topic": "Career", "todos": []any{map[string]any{"title": "Mock interview"}}}})
	if isErr || note["id"] != "career-interview-prep" {
		t.Fatal(note)
	}
	n, err := db.Detail(ctx, "note", "career-interview-prep")
	if todos, _ := n.Entry["todos"].([]any); err != nil || len(todos) != 1 {
		t.Fatal(n, err)
	}
}
