package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/config"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderAnalysisSettings(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	s.config.AnalysisModel, s.config.AnalysisDailyLimit = "claude-sonnet-5", 50

	body := request("GET", "/leetgrinder/settings", nil).Body.String()
	for _, want := range []string{`id="analysis"`, "Not configured", "ANTHROPIC_API_KEY", "claude-sonnet-5", "0 of 50", "0 queued", `name="enabled" value="true" checked`, "sent to Anthropic only while analysis is on"} {
		if !strings.Contains(body, want) {
			t.Errorf("settings lack %q", want)
		}
	}
	const key = "sk-ant-render-never-0123456789"
	s.config.AnthropicAPIKey = config.Secret(key)
	body = request("GET", "/leetgrinder/settings", nil).Body.String()
	if strings.Contains(body, key) || !strings.Contains(body, "Configured") || strings.Contains(body, "Not configured") {
		t.Fatal("key rendered or status missing")
	}

	settings, _ := db.LeetgrinderSettings(ctx)
	w := request("POST", "/leetgrinder/settings/analysis", url.Values{"revision": {settings.Revision}})
	if w.Code != 303 || w.Header().Get("Location") != "/leetgrinder/settings?saved=analysis#analysis" {
		t.Fatalf("turn off: %d %s", w.Code, w.Body.String())
	}
	if settings, _ = db.LeetgrinderSettings(ctx); settings.AnalysisEnabled {
		t.Fatal("analysis still on")
	}
	if body = request("GET", "/leetgrinder/settings", nil).Body.String(); strings.Contains(body, `name="enabled" value="true" checked`) {
		t.Fatal("toggle still checked")
	}
	if w = request("POST", "/leetgrinder/settings/analysis", url.Values{"revision": {settings.Revision}, "enabled": {"yes"}}); w.Code != 400 {
		t.Fatalf("bad value: %d", w.Code)
	}
	if w = request("POST", "/leetgrinder/settings/analysis", url.Values{"revision": {"00000000-0000-4000-8000-000000000000"}, "enabled": {"true"}}); w.Code != 409 {
		t.Fatalf("stale revision: %d", w.Code)
	}
	if w = request("POST", "/leetgrinder/settings/analysis", url.Values{"revision": {settings.Revision}, "enabled": {"true"}}); w.Code != 303 {
		t.Fatalf("turn on: %d", w.Code)
	}
	if settings, _ = db.LeetgrinderSettings(ctx); !settings.AnalysisEnabled {
		t.Fatal("analysis still off")
	}
}

func TestLeetgrinderAnalysisDisplayAndReanalyse(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	s.config.AnthropicAPIKey = config.Secret("sk-ant-test")
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	attempt, err := db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "solved", Minutes: 20, Source: "extension", TimeComplexity: "O(n²)", SpaceComplexity: "O(1)", Code: "for i in a:\n  for j in a: pass", CodeLanguage: "python3"}, "")
	if err != nil {
		t.Fatal(err)
	}
	history := func() string { return request("GET", "/leetgrinder/problem/two-sum", nil).Body.String() }
	if body := history(); !strings.Contains(body, "Claude's assessment") || !strings.Contains(body, "Queued for analysis") || !strings.Contains(body, `id="attempt-`+attempt.ID+`"`) {
		t.Fatalf("queued card missing:\n%s", body)
	}

	job, ok, err := db.NextLeetgrinderAnalysis(ctx, now)
	if err != nil || !ok {
		t.Fatalf("job %v %v", ok, err)
	}
	f, tr := false, true
	result := leetgrinder.AnalysisResult{ActualTime: "O(n)", ActualSpace: "O(n)", TimeMatches: &f, SpaceMatches: &tr, Optimal: true, Explanation: "Uses a hash map. <script>alert(1)</script>"}
	if err = db.FinishLeetgrinderAnalysis(ctx, job, leetgrinder.AnalysisDone, 1, result, "claude-sonnet-5", "", now); err != nil {
		t.Fatal(err)
	}
	body := history()
	for _, want := range []string{"✗</span> your O(n²) does not match", "✓</span> matches your O(1)", ">O(n)</span>", "<dt>Optimal</dt>", "Assessed by claude-sonnet-5", "&lt;script&gt;", `action="/leetgrinder/problem/two-sum/attempts/` + attempt.ID + `/analysis"`, "Re-analyse"} {
		if !strings.Contains(body, want) {
			t.Errorf("history lacks %q", want)
		}
	}
	if strings.Contains(body, "<script>alert") {
		t.Fatal("explanation not escaped")
	}
	for _, path := range []string{"/leetgrinder/problems", "/leetgrinder"} {
		if b := request("GET", path, nil).Body.String(); !strings.Contains(b, "Complexity off") {
			t.Errorf("%s lacks the complexity badge", path)
		}
	}

	// Re-analyse: session and same-origin form required.
	target := "/leetgrinder/problem/two-sum/attempts/" + attempt.ID + "/analysis"
	raw := func(origin string, session bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", target, strings.NewReader(""))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if session {
			r.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
		}
		w := httptest.NewRecorder()
		s.RegisterRoutes().ServeHTTP(w, r)
		return w
	}
	// Without a session the request goes to sign-in, not the handler.
	if w := raw("https://example.com", false); w.Code != 401 && (w.Code != 303 || strings.Contains(w.Header().Get("Location"), "/leetgrinder")) {
		t.Errorf("no session: %d %s", w.Code, w.Header().Get("Location"))
	}
	for name, w := range map[string]*httptest.ResponseRecorder{
		"cross origin": raw("https://evil.example", true),
		"no origin":    raw("", true),
	} {
		if w.Code != 403 {
			t.Errorf("%s: %d", name, w.Code)
		}
	}
	if state, _ := db.LeetgrinderState(ctx); !state.Analyses[attempt.ID].Done() {
		t.Fatal("rejected request changed the analysis")
	}
	w := request("POST", target, url.Values{})
	if w.Code != 303 || w.Header().Get("Location") != "/leetgrinder/problem/two-sum#attempt-"+attempt.ID {
		t.Fatalf("re-analyse: %d %s", w.Code, w.Header().Get("Location"))
	}
	state, _ := db.LeetgrinderState(ctx)
	if a := state.Analyses[attempt.ID]; a.Status != leetgrinder.AnalysisPending || a.Tries != 0 || a.ActualTime != "" {
		t.Fatalf("requeued %+v", a)
	}
	if b := request("GET", "/leetgrinder/problems", nil).Body.String(); strings.Contains(b, "Complexity off") {
		t.Error("badge kept after re-analyse")
	}
	for _, bad := range []string{"/leetgrinder/problem/valid-anagram/attempts/" + attempt.ID + "/analysis", "/leetgrinder/problem/two-sum/attempts/not-a-uuid/analysis", "/leetgrinder/problem/two-sum/attempts/" + uuid.NewString() + "/analysis"} {
		if w = request("POST", bad, url.Values{}); w.Code != 404 {
			t.Errorf("%s: %d", bad, w.Code)
		}
	}

	// A failed analysis shows its error; with analysis off, queued attempts say so.
	job, _, _ = db.NextLeetgrinderAnalysis(ctx, now)
	if err = db.FinishLeetgrinderAnalysis(ctx, job, leetgrinder.AnalysisFailed, 4, leetgrinder.AnalysisResult{}, "claude-sonnet-5", "Anthropic API returned 500 api_error", now); err != nil {
		t.Fatal(err)
	}
	if body = history(); !strings.Contains(body, "Analysis failed: Anthropic API returned 500 api_error") || !strings.Contains(body, "Re-analyse") {
		t.Error("failed card missing")
	}
	s.config.AnthropicAPIKey = ""
	if _, err = db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "solved", Minutes: 9, Source: "extension", TimeComplexity: "O(n)", SpaceComplexity: "O(n)", Code: "pass", CodeLanguage: "python3"}, ""); err != nil {
		t.Fatal(err)
	}
	if body = history(); !strings.Contains(body, "Analysis is off.") {
		t.Error("analysis-off note missing")
	}
}
