package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderGeneralSettings(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	revision := func() string {
		settings, err := db.LeetgrinderSettings(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return settings.Revision
	}
	save := func(values url.Values) *httptest.ResponseRecorder {
		if values.Get("revision") == "" {
			values.Set("revision", revision())
		}
		return request("POST", "/leetgrinder/settings/general", values)
	}
	if w := request("GET", "/leetgrinder/settings", nil); !strings.Contains(w.Body.String(), "Changes apply from today.") {
		t.Fatal("unfrozen day not explained")
	}
	if w := save(url.Values{"timezone": {"Asia/Tokyo"}, "goalNew": {"3"}, "goalReview": {"0"}}); w.Code != 303 || w.Header().Get("Location") != "/leetgrinder/settings?saved=general#general" {
		t.Fatalf("save: %d %s", w.Code, w.Body.String())
	}
	if settings, _ := db.LeetgrinderSettings(ctx); settings.Timezone != "Asia/Tokyo" || settings.Goal != (leetgrinder.DailyGoal{New: 3}) {
		t.Fatalf("saved settings: %+v", settings)
	}
	for _, test := range []struct {
		values url.Values
		want   int
		text   string
	}{
		{url.Values{"timezone": {"Local"}, "goalNew": {"2"}, "goalReview": {"1"}}, 400, "IANA"},
		{url.Values{"timezone": {"America/Chicago"}, "goalNew": {"0"}, "goalReview": {"0"}}, 400, "at least one above 0"},
		{url.Values{"timezone": {"America/Chicago"}, "goalNew": {"11"}, "goalReview": {"1"}}, 400, "0 to 10"},
		{url.Values{"timezone": {"America/Chicago"}, "goalNew": {"x"}, "goalReview": {"1"}}, 400, "0 to 10"},
		{url.Values{"timezone": {"America/Chicago"}, "goalNew": {"2"}, "goalReview": {"1"}, "revision": {uuid.NewString()}}, 409, "Reload settings"},
	} {
		w := save(test.values)
		if w.Code != test.want || !strings.Contains(w.Body.String(), test.text) || !strings.Contains(w.Body.String(), test.values.Get("timezone")) {
			t.Errorf("%v: %d %s", test.values, w.Code, w.Body.String())
		}
	}
	// Once today's goal is frozen, the page says changes apply from tomorrow.
	request("GET", "/leetgrinder", nil)
	if w := request("GET", "/leetgrinder/settings", nil); !strings.Contains(w.Body.String(), "Today's goal is already set to 3 new + 0 reviews, so changes apply from tomorrow.") {
		t.Fatal("frozen day not explained")
	}
}

func TestLeetgrinderGoalDashboardAndReviews(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	chicago, _ := time.LoadLocation("America/Chicago")
	now := time.Date(2026, 10, 10, 20, 0, 0, 0, chicago)
	s.now = func() time.Time { return now }
	settings, _ := db.LeetgrinderSettings(ctx)
	if _, err := db.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error { v.Goal = leetgrinder.DailyGoal{New: 1, Review: 2}; return nil }); err != nil {
		t.Fatal(err)
	}
	old := now.AddDate(0, 0, -20)
	for _, slug := range []string{"isomorphic-strings", "two-sum", "some-unseeded-problem"} {
		if _, err := db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: slug, Outcome: "struggled", Minutes: 30, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"}, ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1", old); err != nil {
		t.Fatal(err)
	}
	w := request("GET", "/leetgrinder", nil)
	body := w.Body.String()
	// Two review picks; the third due card is in the optional list.
	for _, want := range []string{"TODAY'S GOAL · 1 new + 2 reviews", "3 to go", "<strong>0/1</strong> new", "<strong>0/2</strong> review", "Struggled 20 days ago", "Optional: 1 more due by the end of today.", `action="/leetgrinder/log"`} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard missing %q", want)
		}
	}
	picks := regexp.MustCompile(`id="review-([a-z0-9-]+)"`).FindAllStringSubmatch(body, -1)
	if len(picks) != 2 {
		t.Fatalf("picks: %v", picks)
	}
	id := regexp.MustCompile(`(?s)id="review-` + picks[0][1] + `".*?name="id" value="([^"]+)"`).FindStringSubmatch(body)
	w = request("POST", "/leetgrinder/problem/"+picks[0][1]+"/attempts", url.Values{"id": {id[1]}, "outcome": {"solved"}, "minutes": {"20"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(n)"}, "review": {"true"}, "return": {"overview"}})
	if w.Code != 303 || w.Header().Get("Location") != "/leetgrinder#review-"+picks[0][1] {
		t.Fatalf("review save: %d %s", w.Code, w.Header().Get("Location"))
	}
	if _, err := db.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1 WHERE id=$2", now.Add(-time.Hour), id[1]); err != nil {
		t.Fatal(err)
	}
	// A new problem and a re-solve of the optional due card finish the goal.
	for _, slug := range []string{"design-a-thing", "two-sum", "isomorphic-strings", "some-unseeded-problem"} {
		if slug == picks[0][1] {
			continue
		}
		if w = request("POST", "/leetgrinder/problem/"+slug+"/attempts", url.Values{"id": {uuid.NewString()}, "outcome": {"unfinished"}, "minutes": {"30"}}); w.Code != 303 {
			t.Fatalf("%s: %d", slug, w.Code)
		}
	}
	// Attempts are saved at the real time; move the new ones into the test's today.
	if _, err := db.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1 WHERE created_at > $2", now.Add(-30*time.Minute), old); err != nil {
		t.Fatal(err)
	}
	body = request("GET", "/leetgrinder", nil).Body.String()
	for _, want := range []string{"Goal met", "<strong>1/1</strong> new", "<strong>3/2</strong> review", "<strong>1</strong> bonus", "<strong>1</strong> day"} {
		if !strings.Contains(body, want) {
			t.Errorf("met dashboard missing %q", want)
		}
	}
	w = request("GET", "/leetgrinder/reviews", nil)
	if body = w.Body.String(); w.Code != 200 || !strings.Contains(body, "Reviewed today") || !strings.Contains(body, "Today's pick") || !strings.Contains(body, "some-unseeded-problem") || !strings.Contains(body, "3/2") {
		t.Fatalf("review queue: %d", w.Code)
	}
	w = request("GET", "/leetgrinder/export", nil)
	var exported struct {
		DailyGoals []struct {
			Date        string
			New, Review int
		} `json:"dailyGoals"`
	}
	if json.Unmarshal(w.Body.Bytes(), &exported) != nil || len(exported.DailyGoals) != 1 || exported.DailyGoals[0].Date != "2026-10-10" || exported.DailyGoals[0].Review != 2 {
		t.Fatalf("export goals: %s", w.Body.String())
	}
}

func TestLeetgrinderTodayAPIAndKinds(t *testing.T) {
	// Saved attempts get the real time, so this test runs on the real clock.
	s, db, _ := leetgrinderTestServer(t)
	ctx := context.Background()
	now := time.Now()
	settings, _ := db.LeetgrinderSettings(ctx)
	if _, err := db.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error { v.Timezone = "UTC"; return nil }); err != nil {
		t.Fatal(err)
	}
	handler := s.RegisterRoutes()
	_, plain, err := db.CreateLeetgrinderToken(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	api := func(method, path string, body any) *httptest.ResponseRecorder {
		var payload string
		if body != nil {
			b, _ := json.Marshal(body)
			payload = string(b)
		}
		r := httptest.NewRequest(method, path, strings.NewReader(payload))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer "+plain)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	for _, slug := range []string{"two-sum", "valid-anagram", "binary-search"} {
		if _, err := db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: slug, Outcome: "unfinished", Minutes: 25}, ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1", now.AddDate(0, 0, -20)); err != nil {
		t.Fatal(err)
	}
	var today struct {
		Date      string `json:"date"`
		Goal      struct{ New, Review int }
		Done      struct{ New, Review, Bonus int }
		Met       bool
		Remaining int
		Streak    int
		Picks     []struct {
			Slug, Title, Difficulty, Reason string
			Recall                          float64
			Done                            bool
		}
		Due      []struct{ Slug string }
		DueCount int
	}
	w := api("GET", "/api/leetgrinder/today", nil)
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &today) != nil {
		t.Fatalf("today: %d %s", w.Code, w.Body.String())
	}
	if today.Date != now.UTC().Format(time.DateOnly) || today.Goal.New != 2 || today.Goal.Review != 1 || today.Met || today.Remaining != 3 || len(today.Picks) != 1 || today.Picks[0].Title == "" || today.Picks[0].Recall <= 0 || today.Picks[0].Reason == "" || today.DueCount != 2 || len(today.Due) != 2 {
		t.Fatalf("today: %s", w.Body.String())
	}
	pick := today.Picks[0].Slug
	// Saving the pick reports a review; a first attempt reports new; a
	// re-solve of something not due today would be practice.
	for _, test := range []struct{ slug, kind string }{{pick, "review"}, {"lru-cache", "new"}, {"lru-cache", "new"}} {
		body := map[string]any{"id": uuid.NewString(), "problemSlug": test.slug, "outcome": "unfinished", "minutes": 20, "isReview": false}
		w = api("POST", "/api/leetgrinder/attempts", body)
		var saved struct {
			Kind    string              `json:"kind"`
			Attempt leetgrinder.Attempt `json:"attempt"`
		}
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &saved) != nil || saved.Kind != test.kind {
			t.Fatalf("%s kind: %s", test.slug, w.Body.String())
		}
		if saved.Attempt.IsReview != (test.kind == "review") {
			t.Fatalf("%s is_review %v", test.slug, saved.Attempt.IsReview)
		}
	}
	w = api("GET", "/api/leetgrinder/today", nil)
	today.Picks = nil
	if json.Unmarshal(w.Body.Bytes(), &today) != nil || today.Done.New != 1 || today.Done.Review != 1 || today.Remaining != 1 || !today.Picks[0].Done {
		t.Fatalf("after attempts: %s", w.Body.String())
	}
	if w = api("GET", "/api/leetgrinder/problem/"+pick, nil); !strings.Contains(w.Body.String(), `"todaysPick":true`) || !strings.Contains(w.Body.String(), `"attemptedToday":true`) {
		t.Fatalf("pick status: %s", w.Body.String())
	}
}
