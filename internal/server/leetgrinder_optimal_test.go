package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderEditOptimal(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	slug := "mystery-problem"
	if _, err := db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: slug, Outcome: "solved", Minutes: 20, Source: "extension", TimeComplexity: "O(n)", SpaceComplexity: "O(n)"}, ""); err != nil {
		t.Fatal(err)
	}
	page := func() string { return request("GET", "/leetgrinder/problem/"+slug, nil).Body.String() }
	edit := "/leetgrinder/problem/" + slug + "/optimal"
	revision := func() string {
		p, err := db.LeetgrinderProblem(ctx, slug)
		if err != nil {
			t.Fatal(err)
		}
		return p.OptimalRevision()
	}
	problem := func() leetgrinder.Problem { p, _ := db.LeetgrinderProblem(ctx, slug); return p }
	form := func(time, space, note string) url.Values {
		return url.Values{"timeComplexity": {time}, "spaceComplexity": {space}, "note": {note}, "revision": {revision()}}
	}

	// No value yet: the edit link shows, Re-estimate does not.
	if b := page(); !strings.Contains(b, "Edit optimal complexity") || strings.Contains(b, "Re-estimate") {
		t.Fatalf("empty panel:\n%s", b)
	}
	if b := request("GET", edit, nil).Body.String(); !strings.Contains(b, `name="revision"`) || !strings.Contains(b, "No optimal complexity is known yet") {
		t.Fatalf("edit page:\n%s", b)
	}

	// Validation errors keep the draft and save nothing.
	for name, c := range map[string]struct {
		values url.Values
		want   string
		status int
	}{
		"missing time":  {form("", "O(1)", ""), "Choose both", 400},
		"missing space": {form("O(1)", "", ""), "Choose both", 400},
		"other empty":   {form("other", "O(1)", "n"), "Choose both", 400},
		"other not big-O": {func() url.Values {
			v := form("other", "O(1)", "my note")
			v.Set("timeComplexityOther", "fast")
			return v
		}(), "big-O notation", 400},
		"long note":    {form("O(n)", "O(1)", strings.Repeat("x", 301)), "300 characters", 400},
		"control char": {form("O(n)", "O(1)", "a\x01b"), "control characters", 400},
		"stale":        {url.Values{"timeComplexity": {"O(n)"}, "spaceComplexity": {"O(1)"}, "revision": {"stale"}}, "changed since you opened", 409},
	} {
		w := request("POST", edit, c.values)
		if w.Code != c.status || !strings.Contains(w.Body.String(), c.want) {
			t.Errorf("%s: %d\n%s", name, w.Code, w.Body.String())
		}
		if p := problem(); p.OptimalSource != "" || p.HasOptimal() {
			t.Fatalf("%s saved %+v", name, p)
		}
	}
	if b := request("POST", edit, form("O(n)", "other", "kept note")).Body.String(); !strings.Contains(b, `value="kept note"`) {
		t.Errorf("draft not kept:\n%s", b)
	}

	// A model estimate shows as Claude's estimate with Re-estimate; a manual
	// edit then shows as the learner's value and blocks later estimates.
	if _, err := db.DB.ExecContext(ctx, `UPDATE leetgrinder_problems SET optimal_time='O(n²)',optimal_space='O(1)',optimal_source='model' WHERE slug=$1`, slug); err != nil {
		t.Fatal(err)
	}
	b := page()
	if !strings.Contains(b, "Claude&#39;s estimate") && !strings.Contains(b, "Claude's estimate") || !strings.Contains(b, "Re-estimate") {
		t.Fatalf("model panel:\n%s", b)
	}
	w := request("POST", edit, form("O(n)", "other", ""))
	if w.Code != 400 {
		t.Fatalf("space other without text: %d", w.Code)
	}
	w = request("POST", edit, form("O(n log n)", "O(1)", "  sort first  "))
	if w.Code != 303 || w.Header().Get("Location") != "/leetgrinder/problem/"+slug+"#optimal" {
		t.Fatalf("save: %d %s", w.Code, w.Header().Get("Location"))
	}
	if p := problem(); p.OptimalSource != "manual" || p.OptimalTime != "O(n log n)" || p.OptimalSpace != "O(1)" || p.OptimalNote != "sort first" {
		t.Fatalf("saved %+v", p)
	}
	b = page()
	if !strings.Contains(b, "Your value") || strings.Contains(b, "Re-estimate") || !strings.Contains(b, "O(n log n)") {
		t.Fatalf("manual panel:\n%s", b)
	}
	if b = request("GET", "/leetgrinder/problems", nil).Body.String(); !strings.Contains(b, "Your value") {
		t.Errorf("problems table lacks the label")
	}
	if err := db.FinishLeetgrinderAnalysis(ctx, mustJob(t, s, slug), leetgrinder.AnalysisDone, 1, leetgrinder.AnalysisResult{ActualTime: "O(n)", ActualSpace: "O(n)", Optimal: true, OptimalTime: "O(1)", OptimalSpace: "O(1)"}, "m", "", s.clock()); err != nil {
		t.Fatal(err)
	}
	if p := problem(); p.OptimalSource != "manual" || p.OptimalTime != "O(n log n)" {
		t.Fatalf("analysis overwrote the manual value: %+v", p)
	}

	// A form opened before the edit is refused.
	if w = request("POST", edit, url.Values{"timeComplexity": {"O(1)"}, "spaceComplexity": {"O(1)"}, "revision": {"stale"}}); w.Code != 409 {
		t.Errorf("stale edit: %d", w.Code)
	}
	// Re-estimate refuses a manual value; the curated label shows for seeds.
	re := "/leetgrinder/problem/" + slug + "/optimal/reestimate"
	if w = request("POST", re, url.Values{"revision": {revision()}}); w.Code != 409 || !strings.Contains(w.Body.String(), "curated or manual value is kept") {
		t.Errorf("re-estimate manual: %d", w.Code)
	}
	if p := problem(); p.OptimalSource != "manual" {
		t.Fatalf("manual cleared: %+v", p)
	}
	if b = request("GET", "/leetgrinder/problem/two-sum", nil).Body.String(); strings.Contains(b, "Re-estimate") {
		t.Error("curated problem offers Re-estimate")
	}

	// Re-estimate clears a model value, with a stale revision refused.
	if _, err := db.DB.ExecContext(ctx, `UPDATE leetgrinder_problems SET optimal_time='O(n²)',optimal_space='O(1)',optimal_note='x',optimal_source='model' WHERE slug=$1`, slug); err != nil {
		t.Fatal(err)
	}
	if w = request("POST", re, url.Values{"revision": {"stale"}}); w.Code != 409 || problem().OptimalSource != "model" {
		t.Fatalf("stale re-estimate: %d", w.Code)
	}
	if w = request("POST", re, url.Values{"revision": {revision()}}); w.Code != 303 {
		t.Fatalf("re-estimate: %d", w.Code)
	}
	if p := problem(); p.OptimalSource != "" || p.HasOptimal() || p.OptimalNote != "" {
		t.Fatalf("not cleared: %+v", p)
	}

	// Same-origin, form-encoded, signed-in requests only.
	for _, target := range []string{edit, re} {
		raw := func(origin, contentType string, session bool) (int, string) {
			r := httptest.NewRequest("POST", target, strings.NewReader("timeComplexity=O(1)&spaceComplexity=O(1)&revision="+revision()))
			r.Header.Set("Content-Type", contentType)
			if origin != "" {
				r.Header.Set("Origin", origin)
			}
			if session {
				r.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
			}
			w := httptest.NewRecorder()
			s.RegisterRoutes().ServeHTTP(w, r)
			return w.Code, w.Header().Get("Location")
		}
		form := "application/x-www-form-urlencoded"
		if code, _ := raw("https://evil.example", form, true); code != 403 {
			t.Errorf("%s cross origin: %d", target, code)
		}
		if code, _ := raw("", form, true); code != 403 {
			t.Errorf("%s no origin: %d", target, code)
		}
		if code, _ := raw("https://example.com", "application/json", true); code != 415 {
			t.Errorf("%s json: %d", target, code)
		}
		// Without a session the request goes to sign-in, never the handler.
		if code, loc := raw("https://example.com", form, false); code != 401 && (code != 303 || strings.HasPrefix(loc, "/leetgrinder/problem/")) {
			t.Errorf("%s without a session: %d %s", target, code, loc)
		}
	}
	if p := problem(); p.HasOptimal() {
		t.Fatalf("rejected request saved %+v", p)
	}
	if w = request("POST", "/leetgrinder/problem/Bad_Slug/optimal", form("O(1)", "O(1)", "")); w.Code != 404 {
		t.Errorf("bad slug: %d", w.Code)
	}
}

// mustJob queues an analysable attempt's job for slug.
func mustJob(t *testing.T, s *Server, slug string) database.LeetgrinderAnalysisJob {
	t.Helper()
	ctx := context.Background()
	if _, err := s.db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: slug, Outcome: "solved", Minutes: 20, Source: "extension", TimeComplexity: "O(n)", SpaceComplexity: "O(n)", Code: "pass", CodeLanguage: "python3"}, ""); err != nil {
		t.Fatal(err)
	}
	job, ok, err := s.db.NextLeetgrinderAnalysis(ctx, s.clock())
	if err != nil || !ok {
		t.Fatalf("job %v %v", ok, err)
	}
	return job
}
