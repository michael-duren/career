package leetcode

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

// fakeLeetCode answers GraphQL requests with handler's reply for the slug.
type fakeLeetCode struct {
	*httptest.Server
	mu    sync.Mutex
	slugs []string
	reply func(slug string) (int, string)
}

func newFake(t *testing.T, reply func(slug string) (int, string)) *fakeLeetCode {
	f := &fakeLeetCode{reply: reply}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Query     string            `json:"query"`
			Variables map[string]string `json:"variables"`
		}
		_ = json.Unmarshal(body, &req)
		slug := req.Variables["titleSlug"]
		f.mu.Lock()
		f.slugs = append(f.slugs, slug)
		reply := f.reply
		f.mu.Unlock()
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" || !strings.Contains(req.Query, "questionFrontendId") {
			w.WriteHeader(400)
			return
		}
		status, text := reply(slug)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(text))
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeLeetCode) take() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.slugs
	f.slugs = nil
	return out
}

const twoSum = `{"data":{"question":{"questionFrontendId":"1","title":"Two Sum","difficulty":"Easy","topicTags":[{"slug":"array","name":"Array"},{"slug":"hash-table","name":"Hash Table"}]}}}`

func TestFetch(t *testing.T) {
	f := newFake(t, func(slug string) (int, string) {
		switch slug {
		case "two-sum":
			return 200, twoSum
		case "missing":
			return 200, `{"data":{"question":null}}`
		case "bad-tags":
			return 200, `{"data":{"question":{"questionFrontendId":"2","title":"X","difficulty":"Easy","topicTags":[{"slug":"Not A Slug","name":"x"}]}}}`
		case "blocked":
			return 403, `<html>Just a moment...</html>`
		case "garbage":
			return 200, `<html></html>`
		}
		return 500, ""
	})
	c := &Client{URL: f.URL}
	ctx := context.Background()
	m, err := c.Fetch(ctx, "two-sum")
	if err != nil || m == nil || m.Number != 1 || m.Title != "Two Sum" || m.Difficulty != "Easy" || strings.Join(m.TopicSlugs(), ",") != "array,hash-table" {
		t.Fatalf("two-sum: %+v %v", m, err)
	}
	if m, err = c.Fetch(ctx, "missing"); m != nil || err != nil {
		t.Fatalf("missing problem: %+v %v", m, err)
	}
	for _, slug := range []string{"bad-tags", "blocked", "garbage", "server-error"} {
		if m, err := c.Fetch(ctx, slug); err == nil || m != nil {
			t.Errorf("%s: %+v %v", slug, m, err)
		} else if strings.Contains(err.Error(), "Just a moment") {
			t.Errorf("%s: error quotes the body", slug)
		}
	}
}

func TestFetchTimesOut(t *testing.T) {
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer slow.Close()
	defer close(release)
	c := &Client{URL: slow.URL}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := c.Fetch(ctx, "two-sum"); err == nil || time.Since(start) > 5*time.Second {
		t.Fatalf("slow fetch: %v after %s", err, time.Since(start))
	}
}

func testStore(t *testing.T) *database.Store {
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
	schema := "leetgrinder_fetch_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	return db
}

func TestWorker(t *testing.T) {
	db := testStore(t)
	ctx := context.Background()
	failing := true
	f := newFake(t, func(slug string) (int, string) {
		switch {
		case slug == "flaky" && failing:
			return 502, ""
		case slug == "no-such-problem":
			return 200, `{"data":{"question":null}}`
		case slug == "two-sum":
			return 200, twoSum
		}
		return 200, `{"data":{"question":{"questionFrontendId":"9","title":"Flaky","difficulty":"Hard","topicTags":[]}}}`
	})
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	w := &Worker{Store: db, Client: &Client{URL: f.URL}, Now: func() time.Time { return now }}
	step := func() []string {
		t.Helper()
		if err := w.Step(ctx); err != nil {
			t.Fatal(err)
		}
		return f.take()
	}
	// Seeded problems without attempts are never fetched.
	if got := step(); len(got) != 0 {
		t.Fatalf("fetched with nothing to do: %v", got)
	}
	for _, slug := range []string{"no-such-problem", "flaky"} {
		if _, err := db.EnsureLeetgrinderProblem(ctx, slug); err != nil {
			t.Fatal(err)
		}
		now = now.Add(time.Second)
	}
	// Newest first, one per step.
	if got := step(); len(got) != 1 || got[0] != "flaky" {
		t.Fatalf("first fetch: %v", got)
	}
	// The failure backs off 1, 10, then 60 minutes, then gives up.
	if got := step(); len(got) != 1 || got[0] != "no-such-problem" {
		t.Fatalf("second fetch: %v", got)
	}
	if p, _ := db.LeetgrinderProblem(ctx, "no-such-problem"); !p.NotFound || p.Fetching() {
		t.Fatalf("not-found problem: %+v", p)
	}
	for i, wait := range []time.Duration{time.Minute, 10 * time.Minute, time.Hour} {
		now = now.Add(wait - time.Second)
		if got := step(); len(got) != 0 {
			t.Fatalf("retry %d before its backoff: %v", i+1, got)
		}
		now = now.Add(time.Second)
		if got := step(); len(got) != 1 {
			t.Fatalf("retry %d: %v", i+1, got)
		}
	}
	now = now.Add(24 * time.Hour)
	if got := step(); len(got) != 0 {
		t.Fatalf("fetched past the attempt cap: %v", got)
	}
	if p, _ := db.LeetgrinderProblem(ctx, "flaky"); p.Known() || p.Fetching() || p.FetchAttempts != leetgrinder.MaxFetchAttempts {
		t.Fatalf("gave-up problem: %+v", p)
	}
	// A seeded problem with an attempt but no tags gets its tags; curated
	// optimal values stay.
	if _, err := db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "unfinished", Minutes: 5}, ""); err != nil {
		t.Fatal(err)
	}
	if got := step(); len(got) != 1 || got[0] != "two-sum" {
		t.Fatalf("seed tag fetch: %v", got)
	}
	p, _ := db.LeetgrinderProblem(ctx, "two-sum")
	if strings.Join(p.Topics, ",") != "array,hash-table" || p.OptimalSource != "curated" || p.OptimalTime != "O(n)" || p.MetadataSource != "leetcode" {
		t.Fatalf("two-sum after fetch: %+v", p)
	}
	if got := step(); len(got) != 0 {
		t.Fatalf("refetched: %v", got)
	}
}
