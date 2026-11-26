package server

import (
	"context"
	"testing"

	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/modules"
)

func TestCountsAPI(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	h := newOAuthHarness(t, db)
	counts := func() map[string]int {
		t.Helper()
		w := h.do("GET", "/api/counts", "", nil, true)
		var body struct{ Counts map[string]int }
		if w.Code != 200 || decodeJSON(w, &body) != nil {
			t.Fatal(w.Code, w.Body.String())
		}
		return body.Counts
	}
	if w := h.do("GET", "/api/counts", "", nil, false); w.Code != 401 {
		t.Fatal("anonymous read", w.Code)
	}
	before := counts()
	// Only modules that own entry kinds are counted.
	for _, id := range []string{"schedule", "leetgrinder", "agents"} {
		if _, ok := before[id]; ok {
			t.Errorf("%s has no entries but is counted", id)
		}
	}
	if _, err := db.Save(ctx, "company", database.Entity{"slug": "tracked/globex", "title": "Globex", "category": "Test", "type": "company", "url": "https://globex.example", "status": "applied", "featured": false, "priority": "high", "tags": []any{}, "body": ""}, nil); err != nil {
		t.Fatal(err)
	}
	if got := counts()["companies"]; got != before["companies"]+1 {
		t.Fatalf("companies %d after one save, want %d", got, before["companies"]+1)
	}
	// Turned-off modules are left out, including ones off through a dependency.
	if _, err := db.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"companies": false, "timeline": false}}); err != nil {
		t.Fatal(err)
	}
	after := counts()
	for _, id := range []string{"companies", "timeline"} {
		if _, ok := after[id]; ok {
			t.Errorf("%s is off but counted: %v", id, after)
		}
	}
	if _, ok := after["notes"]; !ok {
		t.Errorf("notes is on but missing: %v", after)
	}
}
