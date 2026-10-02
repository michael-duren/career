package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

// A topic name sent to the extension API shows on the problems page, its
// filter, the stats page and a todo set, and a later save without a name
// keeps it.
func TestLeetgrinderTopicNamesThroughAPI(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	handler := s.RegisterRoutes()
	w := request("POST", "/leetgrinder/settings/tokens", url.Values{"name": {"ext"}})
	match := createdToken.FindStringSubmatch(w.Body.String())
	if match == nil {
		t.Fatalf("create token: %d", w.Code)
	}
	put := func(topics []map[string]string) {
		t.Helper()
		b, _ := json.Marshal(map[string]any{"number": 104, "title": "Maximum Depth of Binary Tree", "difficulty": "Easy", "topics": topics})
		r := httptest.NewRequest("PUT", "/api/leetgrinder/problem/maximum-depth", strings.NewReader(string(b)))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer "+match[1])
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		if rec.Code != 204 {
			t.Fatalf("PUT: %d %s", rec.Code, rec.Body.String())
		}
	}
	put([]map[string]string{{"slug": "xx-walk", "name": "XX-Walk Search"}, {"slug": "tree", "name": "Tree"}})
	put([]map[string]string{{"slug": "xx-walk", "name": ""}, {"slug": "tree", "name": "Tree"}})

	if _, err := db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "maximum-depth", Outcome: "unfinished", Minutes: 5}, ""); err != nil {
		t.Fatal(err)
	}
	set, err := db.CreateLeetgrinderTodoSetDetailed(ctx, leetgrinder.TodoSet{Title: "Trees"}, []leetgrinder.TodoProblemInput{{Slug: "maximum-depth"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/leetgrinder/problems", "/leetgrinder/problems?q=XX-Walk", "/leetgrinder/stats?all=1", "/leetgrinder/todos/sets/" + set.ID, "/leetgrinder/problem/maximum-depth"} {
		page := request("GET", path, nil)
		body := page.Body.String()
		if page.Code != 200 || !strings.Contains(body, "XX-Walk Search") || strings.Contains(body, "Xx Walk") {
			t.Errorf("%s: %d, stored topic name missing or slug label shown", path, page.Code)
		}
	}
}
