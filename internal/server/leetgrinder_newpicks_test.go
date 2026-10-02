package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLeetgrinderNewPicksFromTodos(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	if _, err := db.CreateLeetgrinderTodoSet(ctx, "Blind 75", []string{"two-sum", "valid-anagram", "group-anagrams"}); err != nil {
		t.Fatal(err)
	}
	settings, err := db.LeetgrinderSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Off by default: the box is unticked and the dashboard shows no picks.
	if strings.Contains(request("GET", "/leetgrinder/settings", nil).Body.String(), `name="newFromTodos" value="true" checked`) {
		t.Fatal("picking from todos on by default")
	}
	tomorrow := now.AddDate(0, 0, 1)
	s.now = func() time.Time { return tomorrow }
	if body := request("GET", "/leetgrinder", nil).Body.String(); strings.Contains(body, "Next new problem") {
		t.Fatal("picks while off")
	}
	// The settings form turns it on; a later, unplanned day picks from the
	// oldest todos.
	s.now = func() time.Time { return now }
	w := request("POST", "/leetgrinder/settings/general", url.Values{"timezone": {"UTC"}, "goalNew": {"2"}, "goalReview": {"1"}, "newFromTodos": {"true"}, "confirm": {"1"}, "revision": {settings.Revision}})
	if w.Code != 303 {
		t.Fatalf("save: %d %s", w.Code, w.Body.String())
	}
	if settings, _ = db.LeetgrinderSettings(ctx); !settings.NewFromTodos {
		t.Fatal("toggle not saved")
	}
	if !strings.Contains(request("GET", "/leetgrinder/settings", nil).Body.String(), `name="newFromTodos" value="true" checked`) {
		t.Fatal("toggle not shown on")
	}
	s.now = func() time.Time { return tomorrow.AddDate(0, 0, 1) }
	body := request("GET", "/leetgrinder", nil).Body.String()
	for _, want := range []string{"<strong>0/2</strong> new", `Next new problem: <a href="/leetgrinder/problem/two-sum">Two Sum</a> from Blind 75`, "Valid Anagram"} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard missing %q", want)
		}
	}
	if n := strings.Count(body, "· Blind 75</span>"); n != 2 {
		t.Errorf("%d picks for a goal of two", n)
	}
	// The extension's today API lists the same picks.
	_, plain, err := db.CreateLeetgrinderToken(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/api/leetgrinder/today", nil)
	r.Header.Set("Authorization", "Bearer "+plain)
	rec := httptest.NewRecorder()
	s.RegisterRoutes().ServeHTTP(rec, r)
	var out struct {
		NewPicks []struct {
			Slug, Title, Set string
			Done             bool
		} `json:"newPicks"`
	}
	if err = json.Unmarshal(rec.Body.Bytes(), &out); err != nil || rec.Code != 200 {
		t.Fatalf("today API: %d %s", rec.Code, rec.Body.String())
	}
	want := []struct {
		Slug, Title, Set string
		Done             bool
	}{{"two-sum", "Two Sum", "Blind 75", false}, {"valid-anagram", "Valid Anagram", "Blind 75", false}}
	if !reflect.DeepEqual(out.NewPicks, want) {
		t.Fatalf("API picks %+v", out.NewPicks)
	}
	// A time zone change keeps the option through the confirm step.
	settings, _ = db.LeetgrinderSettings(ctx)
	w = request("POST", "/leetgrinder/settings/general", url.Values{"timezone": {"Asia/Tokyo"}, "goalNew": {"2"}, "goalReview": {"1"}, "newFromTodos": {"true"}, "revision": {settings.Revision}})
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Confirm the time zone change") || !strings.Contains(w.Body.String(), `<input type="hidden" name="newFromTodos" value="true">`) {
		t.Fatalf("confirm step drops the option: %d %s", w.Code, w.Body.String())
	}
	// The export keeps the setting.
	var exported struct {
		Settings struct {
			NewFromTodos bool `json:"newFromTodos"`
		} `json:"settings"`
	}
	if w = request("GET", "/leetgrinder/export", nil); json.Unmarshal(w.Body.Bytes(), &exported) != nil || !exported.Settings.NewFromTodos {
		t.Fatalf("export: %s", w.Body.String())
	}
	// Unticking the box turns it off.
	w = request("POST", "/leetgrinder/settings/general", url.Values{"timezone": {"UTC"}, "goalNew": {"2"}, "goalReview": {"1"}, "revision": {settings.Revision}})
	if settings, _ = db.LeetgrinderSettings(ctx); w.Code != 303 || settings.NewFromTodos {
		t.Fatalf("turn off: %d %+v", w.Code, settings)
	}
}
