package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/michael-duren/career-strategy/internal/config"
	"github.com/michael-duren/career-strategy/internal/modules"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSettingsAPI(t *testing.T) {
	h := newOAuthHarness(t, testDB(t))
	json := map[string]string{"Origin": testOrigin, "Content-Type": "application/json"}
	if w := h.do("GET", "/api/settings", "", nil, false); w.Code != 401 {
		t.Fatal("anonymous read", w.Code)
	}
	w := h.do("GET", "/api/settings", "", nil, true)
	var got settingsView
	if w.Code != 200 || decodeJSON(w, &got) != nil || got.Theme != "dark" || len(got.Order) != len(modules.Registry) || !got.Effective["schedule"] {
		t.Fatal(w.Code, w.Body.String())
	}
	for name, body := range map[string]string{
		"unknown module": `{"modules":{"bogus":true}}`,
		"bad theme":      `{"theme":"neon"}`,
		"bad order":      `{"moduleOrder":["notes","notes"]}`,
		"bad setup time": `{"setupCompletedAt":"yesterday"}`,
		"unknown field":  `{"colour":"red"}`,
		"null module":    `{"modules":{"notes":null}}`,
	} {
		if w := h.do("PATCH", "/api/settings", body, json, true); w.Code != 400 {
			t.Errorf("%s: %d %s", name, w.Code, w.Body.String())
		}
	}
	// A module whose dependency is off has a locked switch.
	if w := h.do("PATCH", "/api/settings", `{"modules":{"timeline":false,"agents":false}}`, json, true); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := h.do("PATCH", "/api/settings", `{"modules":{"agents":true}}`, json, true); w.Code != 400 || !strings.Contains(w.Body.String(), "Agent context needs Timeline") {
		t.Fatal("locked module turned on:", w.Code, w.Body.String())
	}
	if w := h.do("PATCH", "/api/settings", `{"modules":{"timeline":true,"agents":true}}`, json, true); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := h.do("PATCH", "/api/settings", `{"theme":"cobalt"}`, map[string]string{"Origin": "https://evil.example", "Content-Type": "application/json"}, true); w.Code != 403 {
		t.Fatal("cross-origin patch", w.Code)
	}
	w = h.do("PATCH", "/api/settings", `{"modules":{"timeline":false},"moduleOrder":["notes"],"theme":"concrete","setupCompletedAt":"2026-11-20T10:00:00Z"}`, json, true)
	if w.Code != 200 || decodeJSON(w, &got) != nil {
		t.Fatal(w.Code, w.Body.String())
	}
	if got.Modules["timeline"] || !got.Modules["schedule"] || got.Effective["schedule"] || got.Effective["agents"] || got.Order[0] != "notes" || got.Theme != "concrete" || got.SetupCompletedAt == nil {
		t.Fatalf("%+v", got)
	}
	if w = h.do("PATCH", "/api/settings", `{"setupCompletedAt":null}`, json, true); w.Code != 200 || decodeJSON(w, &got) != nil || got.SetupCompletedAt != nil || got.Theme != "concrete" {
		t.Fatal(w.Code, w.Body.String())
	}
}

func decodeJSON(w *httptest.ResponseRecorder, v any) error { return json.Unmarshal(w.Body.Bytes(), v) }

func TestDisabledModulesReturn404(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	root := t.TempDir()
	for _, page := range []string{"notes", "weekly-scheduler", "timeline", "books", "journal", "static-shells/note"} {
		if err := os.MkdirAll(filepath.Join(root, page), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, page, "index.html"), []byte(page), 0644); err != nil {
			t.Fatal(err)
		}
	}
	s := &Server{db: db, config: config.Config{StaticDir: root, Username: "admin", JWTSecret: "test-secret", PublicOrigin: testOrigin}, attempts: map[string]attempt{}}
	handler := s.RegisterRoutes()
	_, token, err := db.CreateLeetgrinderToken(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	do := func(method, path string, auth string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader("{}"))
		r.Header.Set("Origin", testOrigin)
		r.Header.Set("Content-Type", "application/json")
		switch auth {
		case "session":
			r.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
		case "bearer":
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	set := func(change map[string]bool) {
		t.Helper()
		if _, err := db.UpdateAppSettings(ctx, modules.Patch{Modules: change}); err != nil {
			t.Fatal(err)
		}
	}
	search := func() []string {
		t.Helper()
		w := do("GET", "/api/search?q=caf%C3%A9", "session")
		var out struct {
			Results []struct{ Kind string } `json:"results"`
		}
		if w.Code != 200 || decodeJSON(w, &out) != nil {
			t.Fatal(w.Code, w.Body.String())
		}
		kinds := []string{}
		for _, r := range out.Results {
			kinds = append(kinds, r.Kind)
		}
		return kinds
	}
	paths := []string{"/journal", "/notes", "/notes/", "/notes/systems/nested-note", "/api/entries/note", "/weekly-scheduler", "/api/scheduler/week?week=2026-11-23", "/timeline", "/api/goals", "/api/summaries/progress", "/leetgrinder", "/leetgrinder/problem/two-sum", "/api/agent-context"}
	for _, path := range paths {
		if w := do("GET", path, "session"); w.Code != 200 {
			t.Fatalf("%s while on: %d %s", path, w.Code, w.Body.String())
		}
	}
	if !slices.Contains(search(), "note") {
		t.Fatal("search misses notes while on")
	}
	if w := do("GET", "/api/leetgrinder/today", "bearer"); w.Code != 200 {
		t.Fatal("extension while on", w.Code, w.Body.String())
	}

	// Writes reach their handlers while on (any answer but 404) and are
	// refused before them while off.
	writes := [][3]string{
		{"POST", "/api/entries/note", "session"}, {"DELETE", "/api/entries/note", "session"}, {"POST", "/api/scheduler/mutate", "session"},
		{"POST", "/leetgrinder/problem/two-sum/attempts", "session"}, {"POST", "/leetgrinder/settings/tokens", "session"},
		{"POST", "/api/leetgrinder/attempts", "bearer"}, {"PUT", "/api/leetgrinder/problem/two-sum", "bearer"}, {"GET", "/api/leetgrinder/problem/two-sum", "bearer"},
	}
	for _, call := range writes {
		if w := do(call[0], call[1], call[2]); w.Code == 404 {
			t.Errorf("%s %s while on: 404 %s", call[0], call[1], w.Body.String())
		}
	}

	// Timeline off takes Schedule and Agents down with it; Notes off hides its entries.
	set(map[string]bool{"notes": false, "timeline": false, "leetgrinder": false, "journal": false})
	for _, path := range append(paths, "//notes", "/books/../notes", "/notes/../books") {
		if w := do("GET", path, "session"); w.Code != 404 {
			t.Errorf("%s while off: %d", path, w.Code)
		}
	}
	for _, call := range writes {
		if w := do(call[0], call[1], call[2]); w.Code != 404 {
			t.Errorf("%s %s while off: %d %s", call[0], call[1], w.Code, w.Body.String())
		}
	}
	if w := do("GET", "/api/entries/note", "session"); !strings.Contains(w.Body.String(), "Notes is turned off") {
		t.Error("API 404 should say the module is off:", w.Body.String())
	}
	if w := do("GET", "/api/scheduler/week?week=2026-11-23", "session"); !strings.Contains(w.Body.String(), "Schedule needs Timeline, which is turned off") {
		t.Error(w.Body.String())
	}
	if w := do("GET", "/api/leetgrinder/today", "bearer"); w.Code != 404 || !strings.Contains(w.Body.String(), "Leetgrinder is turned off") {
		t.Error("extension while off:", w.Code, w.Body.String())
	}
	// Unauthenticated requests still get 401 and learn nothing about modules.
	if w := do("GET", "/api/entries/note", ""); w.Code != 401 {
		t.Error("anonymous API:", w.Code)
	}
	if w := do("GET", "/api/leetgrinder/today", "session"); w.Code != 404 {
		t.Error("cookie request to an extension route while off:", w.Code)
	}
	if w := do("GET", "/api/leetgrinder/today", ""); w.Code != 401 {
		t.Error("anonymous extension:", w.Code)
	}
	if w := do("GET", "/notes", ""); w.Code != http.StatusSeeOther {
		t.Error("anonymous page:", w.Code)
	}
	// Other modules and module-less routes are unaffected.
	if w := do("GET", "/api/entries/week", "session"); w.Code != 404 {
		t.Error("journal entries while Journal is off:", w.Code)
	}
	for _, path := range []string{"/books", "/api/entries/book", "/api/entries/document", "/api/entries/company", "/api/settings"} {
		if w := do("GET", path, "session"); w.Code != 200 {
			t.Errorf("%s: %d", path, w.Code)
		}
	}
	if kinds := search(); slices.Contains(kinds, "note") {
		t.Fatal("search returned notes while off", kinds)
	}
	if w := do("GET", "/api/search?q=caf%C3%A9&kind=note", "session"); w.Code != 200 || strings.Contains(w.Body.String(), `"kind":"note"`) {
		t.Fatal("kind search of a disabled module", w.Code, w.Body.String())
	}

	// Turning them back on restores everything, data included.
	set(map[string]bool{"notes": true, "timeline": true, "leetgrinder": true, "journal": true})
	for _, path := range paths {
		if w := do("GET", path, "session"); w.Code != 200 {
			t.Fatalf("%s after turning on: %d", path, w.Code)
		}
	}
	if w := do("GET", "/api/entries/note", "session"); !strings.Contains(w.Body.String(), "systems/nested-note") {
		t.Fatal("note data lost", w.Body.String())
	}
	if !slices.Contains(search(), "note") {
		t.Fatal("search misses notes after turning on")
	}
}

func TestMCPHidesDisabledModules(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	h := newOAuthHarness(t, db)
	clientID := h.register(testCallback)
	verifier := strings.Repeat("m7", 30)
	code := h.approve(h.consent(clientID, testCallback, pkce(verifier)), "read").Query().Get("code")
	status, tokens := h.token(url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {testCallback}, "client_id": {clientID}, "code_verifier": {verifier}})
	if status != 200 {
		t.Fatal(status, tokens)
	}
	srv := httptest.NewServer(h.handler)
	t.Cleanup(srv.Close)
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: srv.URL + "/api/mcp", HTTPClient: &http.Client{Transport: bearerTransport{tokens["access_token"].(string)}}, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	toolNames := func() []string {
		t.Helper()
		list, err := session.ListTools(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		names := []string{}
		for _, tool := range list.Tools {
			names = append(names, tool.Name)
		}
		return names
	}
	call := func(name string, args map[string]any) (string, bool) {
		t.Helper()
		res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatal(name, err)
		}
		b, _ := json.Marshal(res.StructuredContent)
		if res.IsError {
			return res.Content[0].(*mcp.TextContent).Text, true
		}
		return string(b), false
	}

	all := toolNames()
	// Every tool the registry assigns to a module exists.
	for _, m := range modules.Registry {
		for _, tool := range m.MCPTools {
			if !slices.Contains(all, tool) {
				t.Errorf("%s lists unknown tool %s", m.ID, tool)
			}
		}
	}

	if _, err := db.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"leetgrinder": false, "timeline": false, "notes": false}}); err != nil {
		t.Fatal(err)
	}
	hidden := []string{}
	for _, id := range []string{"leetgrinder", "schedule"} {
		m, _ := modules.Lookup(id)
		hidden = append(hidden, m.MCPTools...)
	}
	names := toolNames()
	if len(names) != len(all)-len(hidden) {
		t.Fatalf("%d tools listed, want %d: %v", len(names), len(all)-len(hidden), names)
	}
	for _, tool := range hidden {
		if slices.Contains(names, tool) {
			t.Errorf("%s listed while its module is off", tool)
		}
	}
	if out, isErr := call("get_leetgrinder_today", nil); !isErr || !strings.Contains(out, "Leetgrinder is turned off") {
		t.Fatal("leetgrinder tool while off:", out)
	}
	if out, isErr := call("get_weekly_schedule", map[string]any{}); !isErr || !strings.Contains(out, "Schedule needs Timeline") {
		t.Fatal("schedule tool while timeline is off:", out)
	}
	for tool, args := range map[string]map[string]any{
		"read_career_entry":   {"kind": "note", "id": "systems/nested-note"},
		"list_career_entries": {"kind": "note"},
		"create_career_entry": {"kind": "note", "entry": map[string]any{"title": "Blocked"}},
		"update_career_entry": {"kind": "note", "id": "systems/nested-note", "revision": "x", "fields": map[string]any{"title": "Blocked"}},
		"delete_career_entry": {"kind": "note", "id": "systems/nested-note", "revision": "x"},
	} {
		if out, isErr := call(tool, args); !isErr || !strings.Contains(out, "Notes is turned off") {
			t.Errorf("%s with notes off: %s", tool, out)
		}
	}
	if out, isErr := call("search_career_context", map[string]any{"query": "café"}); isErr || strings.Contains(out, `"kind":"note"`) {
		t.Fatal("search returned notes while off:", out)
	}
	if out, isErr := call("get_career_overview", nil); isErr || strings.Contains(out, `"note":`) || !strings.Contains(out, `"goals":[]`) {
		t.Fatal("overview while notes and timeline are off:", out)
	}
	if _, isErr := call("list_career_entries", map[string]any{"kind": "page"}); isErr {
		t.Fatal("pages are not a module")
	}
	if out, _ := call("get_career_overview", nil); strings.Contains(out, `"note"`) || !strings.Contains(out, `"page"`) {
		t.Fatal("overview kinds while notes are off:", out)
	}

	// People stays on without Companies, but its company queue tools need Companies.
	if _, err := db.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"companies": false}}); err != nil {
		t.Fatal(err)
	}
	if names := toolNames(); slices.Contains(names, "add_companies_to_queue") || slices.Contains(names, "list_connection_companies") {
		t.Fatal("queue tools listed while Companies is off", names)
	}
	if out, isErr := call("list_connection_companies", map[string]any{}); !isErr || !strings.Contains(out, "Companies is turned off") {
		t.Fatal(out)
	}
	if _, isErr := call("list_career_entries", map[string]any{"kind": "connection"}); isErr {
		t.Fatal("connections stay available while Companies is off")
	}
	if _, err := db.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"companies": true}}); err != nil {
		t.Fatal(err)
	}

	if _, err := db.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"leetgrinder": true, "timeline": true, "notes": true}}); err != nil {
		t.Fatal(err)
	}
	if got := toolNames(); len(got) != len(all) {
		t.Fatalf("tools after turning on: %v", got)
	}
	if out, isErr := call("read_career_entry", map[string]any{"kind": "note", "id": "systems/nested-note"}); isErr {
		t.Fatal(out)
	}
}

func TestAgentContextLeavesOutDisabledModules(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	s := &Server{db: db, config: config.Config{Username: "admin", JWTSecret: "test-secret", PublicOrigin: testOrigin}}
	handler := s.RegisterRoutes()
	get := func(path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		r.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	var before struct {
		Journals   map[string][]any `json:"journals"`
		References map[string][]any `json:"references"`
	}
	if w := get("/api/agent-context"); w.Code != 200 || decodeJSON(w, &before) != nil || len(before.Journals["work"]) == 0 || len(before.References["notes"]) == 0 {
		t.Fatal("fixture should have work journal and notes", w.Code, w.Body.String())
	}
	if len(before.References["books"]) == 0 || len(before.References["companies"]) == 0 {
		t.Fatal("fixture should have books and companies", before.References)
	}
	if _, err := db.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"journal": false, "audio": false, "notes": false, "bookshelf": false, "companies": false}}); err != nil {
		t.Fatal(err)
	}
	var after struct {
		Journals   map[string][]any `json:"journals"`
		References map[string][]any `json:"references"`
	}
	if w := get("/api/agent-context"); w.Code != 200 || decodeJSON(w, &after) != nil || len(after.Journals["work"]) != 0 || len(after.Journals["personal"]) != 0 || len(after.References["notes"]) != 0 || len(after.References["books"]) != 0 || len(after.References["companies"]) != 0 || len(after.References["pages"]) == 0 {
		t.Fatal("disabled modules leaked into agent context", w.Body.String())
	}
	for _, journal := range []string{"work", "personal", "running"} {
		if w := get("/api/agent-context?journal=" + journal); w.Code != 404 {
			t.Errorf("journal=%s while off: %d", journal, w.Code)
		}
	}
	if w := get("/api/agent-context?format=markdown"); w.Code != 200 || !strings.Contains(w.Body.String(), "# Work journal\n\nNo entries yet.") || !strings.Contains(w.Body.String(), "# Personal journal\n\nNo entries yet.") {
		t.Fatal("markdown leaked journal entries", w.Body.String())
	}
	// The running view follows Audio, the work view follows Journal.
	if _, err := db.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"audio": true}}); err != nil {
		t.Fatal(err)
	}
	if w := get("/api/agent-context?journal=running"); w.Code != 200 {
		t.Error("running view with Audio on and Journal off:", w.Code)
	}
	if w := get("/api/agent-context?journal=work"); w.Code != 404 {
		t.Error("work view with Journal off:", w.Code)
	}
	if _, err := db.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"audio": false}}); err != nil {
		t.Fatal(err)
	}
	// The journal's Markdown view needs Journal, not Agents.
	if _, err := db.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"journal": true, "agents": false}}); err != nil {
		t.Fatal(err)
	}
	if w := get("/api/agent-context?format=markdown&journal=work"); w.Code != 200 {
		t.Error("journal view with Agents off:", w.Code, w.Body.String())
	}
	if w := get("/api/agent-context"); w.Code != 404 {
		t.Error("agent context with Agents off:", w.Code)
	}
}

func TestModuleGuardFailsClosed(t *testing.T) {
	db := testDB(t)
	if _, err := db.DB.Exec(`INSERT INTO app_settings(id,modules) VALUES(1,'{"notes":"yes"}') ON CONFLICT (id) DO UPDATE SET modules=EXCLUDED.modules`); err != nil {
		t.Fatal(err)
	}
	s := &Server{db: db, config: config.Config{Username: "admin", JWTSecret: "test-secret", PublicOrigin: testOrigin}}
	handler := s.RegisterRoutes()
	for _, path := range []string{"/api/entries/note", "/api/search?q=caf%C3%A9", "/api/agent-context"} {
		r := httptest.NewRequest("GET", path, nil)
		r.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 503 {
			t.Errorf("%s with unreadable settings: %d", path, w.Code)
		}
	}
}

func TestSchedulerCyclePausesWhileScheduleIsOff(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	ticks := 0
	s := &Server{db: db, config: config.Config{}, schedulerTick: func(ctx context.Context, now time.Time) error { ticks++; return db.SchedulerTick(ctx, now) }}
	now := time.Date(2030, 1, 7, 9, 0, 0, 0, time.UTC)
	var last time.Time
	for name, change := range map[string]map[string]bool{
		"schedule off": {"schedule": false},
		"timeline off": {"schedule": true, "timeline": false},
	} {
		if _, err := db.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"timeline": true, "schedule": true}}); err != nil {
			t.Fatal(err)
		}
		if _, err := db.UpdateAppSettings(ctx, modules.Patch{Modules: change}); err != nil {
			t.Fatal(err)
		}
		if s.schedulerCycle(ctx, now, &last) || ticks != 0 {
			t.Errorf("%s: scheduler ran (%d ticks)", name, ticks)
		}
	}
	if _, err := db.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"timeline": true, "schedule": true}}); err != nil {
		t.Fatal(err)
	}
	if !s.schedulerCycle(ctx, now, &last) || ticks != 1 {
		t.Fatal("scheduler did not run with Schedule on", ticks)
	}
	if _, err := db.DB.Exec(`UPDATE app_settings SET modules='{"schedule":"yes"}'`); err != nil {
		t.Fatal(err)
	}
	if s.schedulerCycle(ctx, now, &last) || ticks != 1 {
		t.Fatal("scheduler ran with unreadable settings", ticks)
	}
}

func TestRequireModuleRejectsUnknownIDs(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("a typo in a module id must fail at wiring time")
		}
	}()
	(&Server{}).requireModule("leetgrindr")
}
