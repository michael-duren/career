package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
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
	// The extension's today API lists the same picks, with an attempt today
	// marking its pick done.
	attempt := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "unfinished", Minutes: 20}
	if _, err = db.SaveLeetgrinderAttempt(ctx, attempt, ""); err != nil {
		t.Fatal(err)
	}
	if _, err = db.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1 WHERE id=$2", tomorrow.AddDate(0, 0, 1).Add(-time.Hour), attempt.ID); err != nil {
		t.Fatal(err)
	}
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
	}{{"two-sum", "Two Sum", "Blind 75", true}, {"valid-anagram", "Valid Anagram", "Blind 75", false}}
	if !reflect.DeepEqual(out.NewPicks, want) {
		t.Fatalf("API picks %+v", out.NewPicks)
	}
	exported := func() bool {
		t.Helper()
		var out struct {
			Settings struct {
				NewFromTodos *bool `json:"newFromTodos"`
			} `json:"settings"`
		}
		if w := request("GET", "/leetgrinder/export", nil); json.Unmarshal(w.Body.Bytes(), &out) != nil || out.Settings.NewFromTodos == nil {
			t.Fatalf("export: %s", w.Body.String())
		}
		return *out.Settings.NewFromTodos
	}
	if !exported() {
		t.Fatal("export dropped the option")
	}
	// A time zone change keeps the option, on or off, through the confirm
	// step: the confirm form's hidden fields are what gets saved.
	hidden := regexp.MustCompile(`<input type="hidden" name="([A-Za-z]+)" value="([^"]*)">`)
	for i, zone := range []string{"Asia/Tokyo", "Europe/Paris", "UTC"} {
		on := i%2 == 0
		settings, _ = db.LeetgrinderSettings(ctx)
		draft := url.Values{"timezone": {zone}, "goalNew": {"2"}, "goalReview": {"1"}, "revision": {settings.Revision}}
		if on {
			draft.Set("newFromTodos", "true")
		}
		w = request("POST", "/leetgrinder/settings/general", draft)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "Confirm the time zone change") {
			t.Fatalf("%s: no confirm step: %d", zone, w.Code)
		}
		confirm := url.Values{}
		for _, m := range hidden.FindAllStringSubmatch(w.Body.String(), -1) {
			confirm.Set(m[1], m[2])
		}
		if confirm.Get("confirm") != "1" || (confirm.Get("newFromTodos") == "true") != on {
			t.Fatalf("%s: confirm form %v, option %v", zone, confirm, on)
		}
		if w = request("POST", "/leetgrinder/settings/general", confirm); w.Code != 303 {
			t.Fatalf("%s: confirm: %d", zone, w.Code)
		}
		if settings, _ = db.LeetgrinderSettings(ctx); settings.Timezone != zone || settings.NewFromTodos != on {
			t.Fatalf("%s: saved %+v, option %v", zone, settings, on)
		}
		if exported() != on {
			t.Fatalf("%s: export disagrees with option %v", zone, on)
		}
	}
}
