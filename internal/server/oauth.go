package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

// The server is its own single-owner OAuth 2.1 authorization server for MCP
// clients. Every artifact (client ID, consent, code, access and refresh token)
// is an HMAC-signed claim set keyed from JWT_SECRET, so all replicas agree
// without shared session state. Only single-use IDs are stored in PostgreSQL.
// Rotating JWT_SECRET revokes every MCP client and token.

const (
	mcpScope        = "career:read"
	mcpWriteScope   = "career:write"
	accessTokenTTL  = time.Hour
	refreshTokenTTL = 30 * 24 * time.Hour
	codeTTL         = 5 * time.Minute
	consentTTL      = 10 * time.Minute
)

var pkceVerifierRE = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)
var pkceChallengeRE = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

var errInvalidSignature = errors.New("invalid or expired token")

func (s *Server) mcpResource() string { return s.config.PublicOrigin + "/api/mcp" }

func (s *Server) oauthKey(purpose string) []byte {
	mac := hmac.New(sha256.New, []byte(s.config.JWTSecret))
	mac.Write([]byte("career-mcp-oauth:" + purpose))
	return mac.Sum(nil)
}

// seal signs claims for one purpose. Keys differ per purpose, so a token of
// one kind can never be accepted as another.
func (s *Server) seal(purpose string, claims any) string {
	b, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, s.oauthKey(purpose))
	mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// open verifies a sealed token and rejects it once its exp (if any) has passed.
func (s *Server) open(purpose, token string, claims any) error {
	payload, sig, ok := strings.Cut(token, ".")
	if !ok || len(token) > 8192 {
		return errInvalidSignature
	}
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil {
		return errInvalidSignature
	}
	mac := hmac.New(sha256.New, s.oauthKey(purpose))
	mac.Write([]byte(payload))
	if !hmac.Equal(got, mac.Sum(nil)) {
		return errInvalidSignature
	}
	b, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return errInvalidSignature
	}
	var expiry struct {
		Exp int64 `json:"exp"`
	}
	if json.Unmarshal(b, &expiry) != nil || (expiry.Exp != 0 && time.Now().Unix() >= expiry.Exp) {
		return errInvalidSignature
	}
	if json.Unmarshal(b, claims) != nil {
		return errInvalidSignature
	}
	return nil
}

func newJTI() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// clientHash binds codes and tokens to a client without embedding its long ID.
func clientHash(clientID string) string {
	sum := sha256.Sum256([]byte(clientID))
	return hex.EncodeToString(sum[:16])
}

type oauthClient struct {
	// ID makes each registration distinct, even with identical metadata.
	ID           string   `json:"i"`
	Name         string   `json:"n"`
	RedirectURIs []string `json:"r"`
}

type authorizeRequest struct {
	ClientID    string `json:"c"`
	RedirectURI string `json:"u"`
	// Explicit records whether the client sent redirect_uri; only then must
	// the token request repeat it (RFC 6749 §4.1.3).
	Explicit  bool   `json:"x,omitempty"`
	State     string `json:"s,omitempty"`
	Challenge string `json:"p"`
	Exp       int64  `json:"exp"`
}

type grantClaims struct {
	JTI         string `json:"jti"`
	Client      string `json:"cid"`
	RedirectURI string `json:"u,omitempty"`
	Challenge   string `json:"p,omitempty"`
	Scope       string `json:"scope"`
	Audience    string `json:"aud"`
	Exp         int64  `json:"exp"`
}

func (s *Server) registerOAuth(r interface {
	Get(string, http.HandlerFunc)
	Post(string, http.HandlerFunc)
	Handle(string, http.Handler)
}) {
	metadata := auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
		Resource:               s.mcpResource(),
		AuthorizationServers:   []string{s.config.PublicOrigin},
		ScopesSupported:        []string{mcpScope, mcpWriteScope},
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Career Strategy",
	})
	r.Handle("/.well-known/oauth-protected-resource", metadata)
	r.Handle("/.well-known/oauth-protected-resource/api/mcp", metadata)
	r.Get("/.well-known/oauth-authorization-server", s.authorizationServerMetadata)
	r.Post("/oauth/register", s.registerClient)
	r.Get("/oauth/authorize", s.authorize)
	r.Post("/oauth/authorize", s.approve)
	r.Post("/oauth/token", s.tokenEndpoint)
}

func (s *Server) authorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	o := s.config.PublicOrigin
	respond(w, 200, map[string]any{
		"issuer":                                         o,
		"authorization_endpoint":                         o + "/oauth/authorize",
		"token_endpoint":                                 o + "/oauth/token",
		"registration_endpoint":                          o + "/oauth/register",
		"scopes_supported":                               []string{mcpScope},
		"response_types_supported":                       []string{"code"},
		"grant_types_supported":                          []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":               []string{"S256"},
		"token_endpoint_auth_methods_supported":          []string{"none"},
		"authorization_response_iss_parameter_supported": true,
	})
}

func oauthError(w http.ResponseWriter, status int, code, description string) {
	respond(w, status, map[string]string{"error": code, "error_description": description})
}

// validRedirectURI accepts HTTPS callbacks and RFC 8252 loopback callbacks.
func validRedirectURI(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || !u.IsAbs() || u.Host == "" || u.User != nil || u.Fragment != "" || len(raw) > 512 {
		return false
	}
	return u.Scheme == "https" || (u.Scheme == "http" && loopback(u.Hostname()))
}

func loopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// redirectMatches compares exactly, except that loopback callbacks may use any
// port (RFC 8252 §7.3), since native clients pick a free port at runtime.
func redirectMatches(registered, requested string) bool {
	if registered == requested {
		return true
	}
	a, errA := url.Parse(registered)
	b, errB := url.Parse(requested)
	if errA != nil || errB != nil || a.Scheme != "http" || b.Scheme != "http" || !loopback(a.Hostname()) {
		return false
	}
	return a.Hostname() == b.Hostname() && a.EscapedPath() == b.EscapedPath() && a.RawQuery == b.RawQuery
}

func (s *Server) registerClient(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RedirectURIs []string `json:"redirect_uris"`
		ClientName   string   `json:"client_name"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	// Unknown RFC 7591 metadata is allowed and ignored; the response states
	// what was actually registered.
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		oauthError(w, 400, "invalid_client_metadata", "Request body must be JSON client metadata.")
		return
	}
	if len(input.RedirectURIs) == 0 || len(input.RedirectURIs) > 10 {
		oauthError(w, 400, "invalid_redirect_uri", "Provide 1–10 redirect_uris.")
		return
	}
	for _, uri := range input.RedirectURIs {
		if !validRedirectURI(uri) {
			oauthError(w, 400, "invalid_redirect_uri", "Redirect URIs must be HTTPS or loopback HTTP without fragments.")
			return
		}
	}
	name := cleanClientName(input.ClientName)
	clientID := s.seal("client", oauthClient{ID: newJTI(), Name: name, RedirectURIs: input.RedirectURIs})
	respond(w, 201, map[string]any{
		"client_id":                  clientID,
		"client_id_issued_at":        time.Now().Unix(),
		"client_name":                name,
		"redirect_uris":              input.RedirectURIs,
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
		"scope":                      mcpScope,
	})
}

// cleanClientName strips control and bidirectional-override characters so a
// self-reported name cannot disguise itself on the consent page.
func cleanClientName(raw string) string {
	name := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r) {
			return -1
		}
		return r
	}, raw)
	runes := []rune(strings.TrimSpace(name))
	return string(runes[:min(len(runes), 100)])
}

func (s *Server) client(clientID string) (oauthClient, bool) {
	var c oauthClient
	return c, clientID != "" && s.open("client", clientID, &c) == nil
}

func (s *Server) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	client, ok := s.client(q.Get("client_id"))
	if !ok {
		authorizePage(w, 400, consentView{Error: "This application is not registered. Remove and re-add the connector."})
		return
	}
	redirectURI := q.Get("redirect_uri")
	if redirectURI == "" && len(client.RedirectURIs) == 1 {
		redirectURI = client.RedirectURIs[0]
	}
	registered := false
	for _, uri := range client.RedirectURIs {
		registered = registered || redirectMatches(uri, redirectURI)
	}
	if !registered {
		authorizePage(w, 400, consentView{Error: "The redirect URI does not match this application's registration."})
		return
	}
	// Anyone can register a client, so errors before consent stay on this site
	// rather than bouncing the owner to an arbitrary callback (RFC 9700 §4.11.2).
	fail := func(_, description string) {
		authorizePage(w, 400, consentView{Error: description})
	}
	switch {
	case q.Get("response_type") != "code":
		fail("unsupported_response_type", "Only the authorization code flow is supported.")
	case q.Get("code_challenge_method") != "S256" || !pkceChallengeRE.MatchString(q.Get("code_challenge")):
		fail("invalid_request", "PKCE with code_challenge_method=S256 is required.")
	case q.Has("resource") && q.Get("resource") != s.mcpResource():
		fail("invalid_target", "Unknown resource.")
	case !s.authenticated(r):
		// pageAccess normally redirects to login first; keep the check here so
		// consent can never be rendered without the owner's session.
		http.Redirect(w, r, "/login?redirect="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
	default:
		request := s.seal("consent", authorizeRequest{ClientID: q.Get("client_id"), RedirectURI: redirectURI, Explicit: q.Get("redirect_uri") != "", State: q.Get("state"), Challenge: q.Get("code_challenge"), Exp: time.Now().Add(consentTTL).Unix()})
		host := redirectURI
		if u, err := url.Parse(redirectURI); err == nil {
			host = u.Host
		}
		authorizePage(w, 200, consentView{Client: client.Name, Host: host, Request: request})
	}
}

func (s *Server) approve(w http.ResponseWriter, r *http.Request) {
	// The consent form posts from this origin with the owner's session cookie.
	if r.Header.Get("Origin") != s.config.PublicOrigin || !s.authenticated(r) {
		authorizePage(w, 403, consentView{Error: "Sign in and approve from this site."})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var req authorizeRequest
	if err := s.open("consent", r.PostFormValue("request"), &req); err != nil {
		authorizePage(w, 400, consentView{Error: "This approval expired. Start the connection again from your MCP client."})
		return
	}
	// The owner picks the access level; the client's requested scope is not
	// trusted to decide whether edits are allowed.
	scope := mcpScope
	switch r.PostFormValue("decision") {
	case "read":
	case "write":
		scope += " " + mcpWriteScope
	default:
		s.redirectResult(w, r, req.RedirectURI, url.Values{"error": {"access_denied"}, "state": {req.State}})
		return
	}
	bound := ""
	if req.Explicit {
		bound = req.RedirectURI
	}
	code := s.seal("code", grantClaims{JTI: newJTI(), Client: clientHash(req.ClientID), RedirectURI: bound, Challenge: req.Challenge, Scope: scope, Audience: s.mcpResource(), Exp: time.Now().Add(codeTTL).Unix()})
	s.redirectResult(w, r, req.RedirectURI, url.Values{"code": {code}, "state": {req.State}})
}

func (s *Server) redirectResult(w http.ResponseWriter, r *http.Request, redirectURI string, values url.Values) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		authorizePage(w, 400, consentView{Error: "Invalid redirect URI."})
		return
	}
	q := u.Query()
	for k, v := range values {
		if len(v) > 0 && v[0] != "" {
			q.Set(k, v[0])
		}
	}
	q.Set("iss", s.config.PublicOrigin)
	u.RawQuery = q.Encode()
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (s *Server) tokenEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Pragma", "no-cache")
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		oauthError(w, 400, "invalid_request", "Send an application/x-www-form-urlencoded body.")
		return
	}
	clientID := r.PostForm.Get("client_id")
	if _, ok := s.client(clientID); !ok {
		oauthError(w, 401, "invalid_client", "Unknown client.")
		return
	}
	var grant grantClaims
	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		if s.open("code", r.PostForm.Get("code"), &grant) != nil || grant.Client != clientHash(clientID) || (grant.RedirectURI != "" && grant.RedirectURI != r.PostForm.Get("redirect_uri")) {
			oauthError(w, 400, "invalid_grant", "Authorization code is invalid, expired, or issued to another client.")
			return
		}
		verifier := r.PostForm.Get("code_verifier")
		sum := sha256.Sum256([]byte(verifier))
		if !pkceVerifierRE.MatchString(verifier) || base64.RawURLEncoding.EncodeToString(sum[:]) != grant.Challenge {
			oauthError(w, 400, "invalid_grant", "PKCE verification failed.")
			return
		}
	case "refresh_token":
		if s.open("refresh", r.PostForm.Get("refresh_token"), &grant) != nil || grant.Client != clientHash(clientID) {
			oauthError(w, 400, "invalid_grant", "Refresh token is invalid, expired, or issued to another client.")
			return
		}
	default:
		oauthError(w, 400, "unsupported_grant_type", "Use authorization_code or refresh_token.")
		return
	}
	if resource := r.PostForm.Get("resource"); resource != "" && resource != grant.Audience {
		oauthError(w, 400, "invalid_target", "Unknown resource.")
		return
	}
	// Codes and refresh tokens are single use: a replay fails on every replica.
	fresh, err := s.db.ConsumeToken(r.Context(), grant.JTI, time.Unix(grant.Exp, 0))
	if err != nil {
		log.Printf("oauth: consume token: %v", err)
		oauthError(w, 503, "temporarily_unavailable", "Retry shortly.")
		return
	}
	if !fresh {
		oauthError(w, 400, "invalid_grant", "This grant was already used. Reconnect the client.")
		return
	}
	now := time.Now()
	access := s.seal("access", grantClaims{JTI: newJTI(), Client: grant.Client, Scope: grant.Scope, Audience: grant.Audience, Exp: now.Add(accessTokenTTL).Unix()})
	refresh := s.seal("refresh", grantClaims{JTI: newJTI(), Client: grant.Client, Scope: grant.Scope, Audience: grant.Audience, Exp: now.Add(refreshTokenTTL).Unix()})
	respond(w, 200, map[string]any{"access_token": access, "token_type": "Bearer", "expires_in": int(accessTokenTTL.Seconds()), "refresh_token": refresh, "scope": grant.Scope})
}

// verifyAccessToken is the bearer verifier for the MCP endpoint.
func (s *Server) verifyAccessToken(_ context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
	var grant grantClaims
	if s.open("access", token, &grant) != nil || grant.Audience != s.mcpResource() {
		return nil, auth.ErrInvalidToken
	}
	return &auth.TokenInfo{Scopes: strings.Fields(grant.Scope), Expiration: time.Unix(grant.Exp, 0), UserID: s.config.Username, Extra: map[string]any{"client": grant.Client}}, nil
}

type consentView struct {
	Client, Host, Request, Error string
}

var consentTemplate = template.Must(template.New("consent").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Connect to Career Strategy</title>
<style>
body{font-family:system-ui,sans-serif;background:#0f172a;color:#e2e8f0;display:grid;place-items:center;min-height:100vh;margin:0;padding:16px;box-sizing:border-box}
main{max-width:28rem;background:#1e293b;border-radius:12px;padding:24px}
h1{font-size:1.25rem;margin-top:0}code{background:#0f172a;padding:2px 6px;border-radius:4px}
.actions{display:flex;flex-wrap:wrap;gap:12px;margin-top:20px}button{flex:1;padding:10px;border-radius:8px;border:0;font-size:1rem;cursor:pointer}
.approve{background:#38bdf8;color:#0f172a;font-weight:600}.deny{background:#334155;color:#e2e8f0}
</style></head><body><main>
{{if .Error}}<h1>Cannot connect</h1><p>{{.Error}}</p>{{else}}
<h1>Allow this application to read your career workspace?</h1>
{{if .Client}}<p>It calls itself <strong>{{.Client}}</strong>. Names are self-reported; check where you return below.</p>{{end}}
<p>It will be able to read your goals, work and personal journals, notes, pages, books, companies, and connections.</p>
<p><strong>Read and edit</strong> also lets it create and change goals, companies, and notes. It cannot delete anything.</p>
<p>After approval you return to <code>{{.Host}}</code>. Only allow it if you started this connection there.</p>
<form method="post" action="/oauth/authorize"><input type="hidden" name="request" value="{{.Request}}">
<div class="actions"><button class="deny" name="decision" value="deny">Deny</button><button class="approve" name="decision" value="read">Allow read only</button><button class="approve" name="decision" value="write">Allow read and edit</button></div>
</form>{{end}}
</main></body></html>`))

func authorizePage(w http.ResponseWriter, status int, view consentView) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'")
	w.WriteHeader(status)
	_ = consentTemplate.Execute(w, view)
}
