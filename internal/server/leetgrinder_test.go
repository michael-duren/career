package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/config"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderAccess(t *testing.T) {
	s := &Server{config: config.Config{Username: "admin", JWTSecret: "test", PublicOrigin: "https://example.com"}}
	handler := s.RegisterRoutes()
	for _, path := range []string{"/leetgrinder", "/leetgrinder/day/1", "/leetgrinder/problem/two-sum", "/leetgrinder/export", "/leetgrinder/reviews", "/leetgrinder/settings", "/leetgrinder/about", "/leetgrinder/problems", "/leetgrinder/log?problem=two-sum"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 303 {
			t.Fatalf("%s got %d", path, w.Code)
		}
	}
	for _, path := range []string{"/bootcamp", "/bootcamp/day/1?from=old", "/bootcamp/problem/two-sum/attempts"} {
		r := httptest.NewRequest("GET", path, nil)
		r.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusPermanentRedirect || w.Header().Get("Location") != strings.Replace(path, "/bootcamp", "/leetgrinder", 1) {
			t.Fatalf("legacy redirect: %d %s", w.Code, w.Header().Get("Location"))
		}
	}
	for _, test := range []struct {
		path   string
		values url.Values
		origin string
		want   int
	}{
		{"/leetgrinder/problem/two-sum/attempts", url.Values{}, "https://evil.com", 403},
		{"/leetgrinder/settings/general", url.Values{"hours": {"2.0"}}, "https://evil.com", 403},
	} {
		r := httptest.NewRequest("POST", test.path, strings.NewReader(test.values.Encode()))
		r.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
		r.Header.Set("Origin", test.origin)
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != test.want {
			t.Fatalf("%s: %d %s", test.path, w.Code, w.Body.String())
		}
	}
}

func leetgrinderTestServer(t *testing.T) (*Server, *database.Store, func(method, path string, values url.Values) *httptest.ResponseRecorder) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL")
	}
	base, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { base.Close() })
	schema := "leetgrinder_http_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = base.DB.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { base.DB.Exec("DROP SCHEMA " + schema + " CASCADE") })
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	db, err := database.Open(dsn + sep + "search_path=" + schema)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = db.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	s := &Server{db: db, config: config.Config{Username: "admin", JWTSecret: "test", PublicOrigin: "https://example.com"}}
	handler := s.RegisterRoutes()
	request := func(method, path string, values url.Values) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(values.Encode()))
		r.AddCookie(&http.Cookie{Name: "session", Value: s.token()})
		r.Header.Set("Origin", "https://example.com")
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	return s, db, request
}

func TestLeetgrinderWorkflow(t *testing.T) {
	_, db, request := leetgrinderTestServer(t)
	for _, path := range []string{"/leetgrinder", "/leetgrinder/problem/two-sum", "/leetgrinder/problem/any-problem-at-all", "/leetgrinder/reviews", "/leetgrinder/settings", "/leetgrinder/problems"} {
		w := request("GET", path, nil)
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Header().Get("Content-Type"), "text/html") || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("wrong response headers")
		}
	}
	for _, path := range []string{"/leetgrinder/problem/Not_A_Slug", "/leetgrinder/problem/" + strings.Repeat("a", 101), "/leetgrinder/lesson-player.js"} {
		if w := request("GET", path, nil); w.Code != 404 {
			t.Fatalf("bad path %s: %d", path, w.Code)
		}
	}
	values := url.Values{"id": {uuid.NewString()}, "outcome": {"solved"}, "minutes": {"25"}, "assisted": {"true"}, "notes": {"needed a hint <script>"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"other"}, "spaceComplexityOther": {" O(n) "}}
	path := "/leetgrinder/problem/two-sum/attempts"
	for i := 0; i < 2; i++ {
		if w := request("POST", path, values); w.Code != 303 || w.Header().Get("Location") != "/leetgrinder/problem/two-sum" {
			t.Fatalf("save: %d %s", w.Code, w.Body.String())
		}
	}
	state, err := db.LeetgrinderState(context.Background())
	if err != nil || len(state.Attempts) != 1 {
		t.Fatalf("retry duplicated attempt: %+v %v", state, err)
	}
	reused := url.Values{"id": values["id"], "outcome": {"struggled"}, "minutes": {"30"}, "notes": {"reused form"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(n)"}}
	w := request("POST", path, reused)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "reused form") || strings.Contains(w.Body.String(), `value="`+values.Get("id")+`"><input type="hidden" name="revision" value="">`) {
		t.Fatalf("reused create identifier was not replaced: %d %s", w.Code, w.Body.String())
	}
	values.Set("revision", state.Attempts[0].Revision)
	values.Set("assisted", "false")
	if w := request("POST", path, values); w.Code != 303 {
		t.Fatalf("edit: %d %s", w.Code, w.Body.String())
	}
	if w := request("POST", path, values); w.Code != 409 || !strings.Contains(w.Body.String(), "needed a hint &lt;script&gt;") {
		t.Fatalf("stale draft not retained: %d %s", w.Code, w.Body.String())
	}
	values.Set("minutes", "bad")
	if w := request("POST", path, values); w.Code != 400 || !strings.Contains(w.Body.String(), "needed a hint &lt;script&gt;") {
		t.Fatal("invalid draft not retained")
	}
	// Any problem can be logged from the web form.
	other := url.Values{"id": {uuid.NewString()}, "outcome": {"unfinished"}, "minutes": {"30"}}
	if w := request("POST", "/leetgrinder/problem/design-a-thing/attempts", other); w.Code != 303 || w.Header().Get("Location") != "/leetgrinder/problem/design-a-thing" {
		t.Fatalf("any-problem save: %d %s", w.Code, w.Body.String())
	}
	if w := request("GET", "/leetgrinder/problem/design-a-thing", nil); !strings.Contains(w.Body.String(), "<h1>design-a-thing</h1>") || !strings.Contains(w.Body.String(), "Fetching details from LeetCode") || !strings.Contains(w.Body.String(), "Unfinished") {
		t.Fatalf("unknown problem page: %s", w.Body.String())
	}
	w = request("GET", "/leetgrinder/export", nil)
	var exported struct {
		Attempts []leetgrinder.Attempt `json:"attempts"`
		Problems []struct {
			Slug          string   `json:"slug"`
			Number        int      `json:"number"`
			Title         string   `json:"title"`
			Topics        []string `json:"topics"`
			OptimalSource string   `json:"optimalSource"`
		} `json:"problems"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &exported) != nil || strings.Contains(w.Body.String(), "completedDays") {
		t.Fatalf("bad export: %s", w.Body.String())
	}
	if len(exported.Attempts) != 2 || len(exported.Problems) != 2 || exported.Problems[0].Slug != "design-a-thing" || exported.Problems[1].Title != "Two Sum" || exported.Problems[1].OptimalSource != "curated" || exported.Problems[0].Topics == nil {
		t.Fatalf("export: %+v", exported)
	}
	unicodeValues := url.Values{"id": {uuid.NewString()}, "outcome": {"unfinished"}, "minutes": {"25"}, "notes": {strings.Repeat("🙂", 2000)}}
	if w = request("POST", path, unicodeValues); w.Code != 303 {
		t.Fatalf("valid Unicode notes rejected: %d %s", w.Code, w.Body.String())
	}
	db.Close()
	values.Set("revision", "")
	values.Set("id", uuid.NewString())
	values.Set("minutes", "25")
	w = request("POST", path, values)
	if w.Code != 503 {
		t.Fatalf("storage failure: %d %s", w.Code, w.Body.String())
	}
}

func TestLeetgrinderRetiredPages(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	if _, err := db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "unfinished", Minutes: 25}, ""); err != nil {
		t.Fatal(err)
	}
	// A month later the failed attempt is due, so loading today would plan it.
	s.now = func() time.Time { return time.Now().AddDate(0, 1, 0) }
	for _, path := range []string{"/leetgrinder/day/1", "/leetgrinder/day/84", "/leetgrinder/day/85", "/leetgrinder/about"} {
		w := request("GET", path, nil)
		if w.Code != http.StatusGone || !strings.Contains(w.Body.String(), `href="/leetgrinder"`) {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	if w := request("POST", "/leetgrinder/day/1/complete", url.Values{"completed": {"true"}}); w.Code != http.StatusGone {
		t.Fatalf("retired complete: %d", w.Code)
	}
	var planned int
	if err := db.DB.QueryRow("SELECT count(*) FROM leetgrinder_review_plan").Scan(&planned); err != nil || planned != 0 {
		t.Fatalf("retired page planned reviews: %d %v", planned, err)
	}
}

func TestLeetgrinderLogForm(t *testing.T) {
	_, db, request := leetgrinderTestServer(t)
	for ref, want := range map[string]string{
		"https://leetcode.com/problems/lru-cache/description/": "/leetgrinder/problem/lru-cache",
		"Two-Sum": "/leetgrinder/problem/two-sum",
	} {
		w := request("GET", "/leetgrinder/log?"+url.Values{"problem": {ref}}.Encode(), nil)
		if w.Code != 303 || w.Header().Get("Location") != want {
			t.Errorf("%s: %d %s", ref, w.Code, w.Header().Get("Location"))
		}
	}
	w := request("GET", "/leetgrinder/log?"+url.Values{"problem": {"https://example.com/<b>"}}.Encode(), nil)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "Enter a LeetCode problem link") || !strings.Contains(w.Body.String(), "https://example.com/&lt;b&gt;") {
		t.Fatalf("bad reference: %d", w.Code)
	}
	// Opening a new problem's page adds no catalog row, so a GET never
	// queues a LeetCode fetch; saving an attempt does.
	if w = request("GET", "/leetgrinder/problem/brand-new-problem", nil); w.Code != 200 || !strings.Contains(w.Body.String(), "Fetching details") {
		t.Fatal(w.Code)
	}
	var rows int
	if err := db.DB.QueryRow("SELECT count(*) FROM leetgrinder_problems WHERE slug='brand-new-problem'").Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("GET added a catalog row: %d %v", rows, err)
	}
}
