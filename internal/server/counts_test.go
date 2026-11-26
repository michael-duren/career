package server

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
	"github.com/michael-duren/career-strategy/internal/modules"
)

func TestCountsAPI(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	h := newOAuthHarness(t, db)
	countsAt := func(path string) map[string]int {
		t.Helper()
		w := h.do("GET", path, "", nil, true)
		var body struct{ Counts map[string]int }
		if w.Code != 200 || decodeJSON(w, &body) != nil {
			t.Fatal(w.Code, w.Body.String())
		}
		return body.Counts
	}
	counts := func() map[string]int { return countsAt("/api/counts") }
	if w := h.do("GET", "/api/counts", "", nil, false); w.Code != 401 {
		t.Fatal("anonymous read", w.Code)
	}
	before := counts()
	// Schedule and Agents save nothing countable; Leetgrinder counts its attempts.
	for _, id := range []string{"schedule", "agents"} {
		if _, ok := before[id]; ok {
			t.Errorf("%s has no entries but is counted", id)
		}
	}
	if _, err := db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "solved", Minutes: 10, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"}, ""); err != nil {
		t.Fatal(err)
	}
	if got := counts()["leetgrinder"]; got != before["leetgrinder"]+1 {
		t.Fatalf("leetgrinder %d after one attempt, want %d", got, before["leetgrinder"]+1)
	}
	if _, err := db.Save(ctx, "company", database.Entity{"slug": "tracked/globex", "title": "Globex", "category": "Test", "type": "company", "url": "https://globex.example", "status": "applied", "featured": false, "priority": "high", "tags": []any{}, "body": ""}, nil); err != nil {
		t.Fatal(err)
	}
	if got := counts()["companies"]; got != before["companies"]+1 {
		t.Fatalf("companies %d after one save, want %d", got, before["companies"]+1)
	}
	// Turned-off modules are left out. (No module that owns entry kinds depends on another
	// today, so a dependency case can't be shown here; readCounts uses Effective() regardless.)
	if _, err := db.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"companies": false, "timeline": false}}); err != nil {
		t.Fatal(err)
	}
	after := counts()
	for _, id := range []string{"companies", "timeline"} {
		if _, ok := after[id]; ok {
			t.Errorf("%s is off but counted: %v", id, after)
		}
	}
	if n, ok := after["notes"]; !ok || n != before["notes"] {
		t.Errorf("notes is on but missing: %v", after)
	}
	// ?all=1 keeps them, for "paused · N kept" in Settings → Modules.
	all := countsAt("/api/counts?all=1")
	if all["companies"] != before["companies"]+1 || all["timeline"] != before["timeline"] {
		t.Errorf("?all=1 counts %v, want companies %d and timeline %d", all, before["companies"]+1, before["timeline"])
	}
}
