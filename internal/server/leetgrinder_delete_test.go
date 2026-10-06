package server

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderDeleteAttempt(t *testing.T) {
	_, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	var ids []string
	for _, notes := range []string{"Keep this attempt", "Accidental attempt"} {
		a, err := db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "unfinished", Minutes: 5, Notes: notes}, "")
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, a.ID)
	}
	if _, err := db.DB.ExecContext(ctx, `INSERT INTO leetgrinder_analyses(attempt_id,code_sha256,status) VALUES($1,$2,'pending')`, ids[1], make([]byte, 32)); err != nil {
		t.Fatal(err)
	}
	path := "/leetgrinder/problem/two-sum/attempts/" + ids[1] + "/delete"
	page := request("GET", "/leetgrinder/problem/two-sum", nil)
	if page.Code != 200 || !strings.Contains(page.Body.String(), `action="`+path+`"`) || !strings.Contains(page.Body.String(), "Delete this attempt") {
		t.Fatalf("missing deletion UI: %d", page.Code)
	}
	for _, bad := range []string{
		"/leetgrinder/problem/other-problem/attempts/" + ids[1] + "/delete",
		"/leetgrinder/problem/two-sum/attempts/not-a-uuid/delete",
		"/leetgrinder/problem/two-sum/attempts/" + uuid.NewString() + "/delete",
	} {
		if w := request("POST", bad, url.Values{"confirm": {"yes"}}); w.Code != 404 {
			t.Fatalf("invalid target %s: %d", bad, w.Code)
		}
	}
	if w := request("GET", path, nil); w.Code != 404 {
		t.Fatalf("GET delete: %d", w.Code)
	}
	if w := request("POST", path, url.Values{"confirm": {"yes"}}); w.Code != 303 || w.Header().Get("Location") != "/leetgrinder/problem/two-sum" {
		t.Fatalf("delete: %d %s", w.Code, w.Body.String())
	}
	state, err := db.LeetgrinderState(ctx)
	if err != nil || len(state.Attempts) != 1 || state.Attempts[0].ID != ids[0] || len(state.Analyses) != 0 {
		t.Fatalf("remaining history: %+v %v", state, err)
	}
	page = request("GET", "/leetgrinder/problem/two-sum", nil)
	if page.Code != 200 || strings.Contains(page.Body.String(), "Accidental attempt") || !strings.Contains(page.Body.String(), "Keep this attempt") {
		t.Fatal("history did not refresh")
	}
	if w := request("POST", path, url.Values{"confirm": {"yes"}}); w.Code != 404 {
		t.Fatalf("already deleted: %d", w.Code)
	}
	last := "/leetgrinder/problem/two-sum/attempts/" + ids[0] + "/delete"
	if w := request("POST", last, nil); w.Code != 400 {
		t.Fatalf("missing confirmation: %d", w.Code)
	}
	if w := request("POST", last, url.Values{"confirm": {"yes"}}); w.Code != 303 {
		t.Fatalf("delete last: %d", w.Code)
	}
	if w := request("GET", "/leetgrinder/problem/two-sum", nil); w.Code != 200 || !strings.Contains(w.Body.String(), "No attempts recorded yet.") {
		t.Fatal("missing empty history")
	}
}
