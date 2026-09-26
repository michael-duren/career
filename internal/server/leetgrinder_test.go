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

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/config"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderAccess(t *testing.T) {
	s := &Server{config: config.Config{Username: "admin", JWTSecret: "test", PublicOrigin: "https://example.com"}}
	handler := s.RegisterRoutes()
	for _, path := range []string{"/leetgrinder", "/leetgrinder/day/1", "/leetgrinder/problem/two-sum", "/leetgrinder/export"} {
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
		{"/leetgrinder/day/1/complete", url.Values{"completed": {"yes"}}, "https://example.com", 400},
		{"/leetgrinder/day/85/complete", url.Values{"completed": {"true"}}, "https://example.com", 404},
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

func TestLeetgrinderWorkflow(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL")
	}
	base, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer base.Close()
	schema := "leetgrinder_http_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = base.DB.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	defer base.DB.Exec("DROP SCHEMA " + schema + " CASCADE")
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	db, err := database.Open(dsn + sep + "search_path=" + schema)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
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
	for _, path := range []string{"/leetgrinder", "/leetgrinder/day/1", "/leetgrinder/day/84", "/leetgrinder/problem/two-sum"} {
		w := request("GET", path, nil)
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Header().Get("Content-Type"), "text/html") || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("wrong response headers")
		}
	}
	for _, path := range []string{"/leetgrinder/day/0", "/leetgrinder/day/85", "/leetgrinder/problem/not-a-problem"} {
		if w := request("GET", path, nil); w.Code != 404 {
			t.Fatalf("bad path %s: %d", path, w.Code)
		}
	}
	values := url.Values{"id": {uuid.NewString()}, "outcome": {"solved"}, "minutes": {"25"}, "assisted": {"true"}, "notes": {"needed a hint <script>"}, "returnDay": {"1"}}
	path := "/leetgrinder/problem/two-sum/attempts"
	for i := 0; i < 2; i++ {
		if w := request("POST", path, values); w.Code != 303 {
			t.Fatalf("save: %d %s", w.Code, w.Body.String())
		}
	}
	state, err := db.LeetgrinderState(context.Background())
	if err != nil || len(state.Attempts) != 1 {
		t.Fatalf("retry duplicated attempt: %+v %v", state, err)
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
	if w := request("POST", "/leetgrinder/day/1/complete", url.Values{"completed": {"true"}}); w.Code != 303 || w.Header().Get("Location") != "/leetgrinder/day/2" {
		t.Fatalf("finish day: %d %s", w.Code, w.Header().Get("Location"))
	}
	w := request("GET", "/leetgrinder/export", nil)
	var exported leetgrinder.State
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &exported) != nil {
		t.Fatal("bad export")
	}
	got := leetgrinder.Summarize(exported)
	if got.Completed != 1 || got.Solved != 1 || got.Independent != 1 || got.NextDay != 2 {
		t.Fatalf("wrong progress: %+v", got)
	}
	if w = request("POST", "/leetgrinder/day/1/complete", url.Values{"completed": {"false"}}); w.Code != 303 {
		t.Fatal("reopen failed")
	}
	state, err = db.LeetgrinderState(context.Background())
	if err != nil || len(state.CompletedDays) != 0 || len(state.Attempts) != 1 {
		t.Fatal("reopening changed attempts")
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
	if w.Code != 503 || !strings.Contains(w.Body.String(), "needed a hint &lt;script&gt;") {
		t.Fatalf("storage failure lost draft: %d %s", w.Code, w.Body.String())
	}
}
