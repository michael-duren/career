package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
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
	if w := save(url.Values{"timezone": {"Asia/Tokyo"}, "goalNew": {"3"}, "goalReview": {"0"}, "confirm": {"1"}}); w.Code != 303 || w.Header().Get("Location") != "/leetgrinder/settings?saved=general#general" {
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
	for slug, kind := range map[string]string{"never-tried": "new", "two-sum": "review", "valid-anagram": "review"} {
		if w := api("GET", "/api/leetgrinder/problem/"+slug, nil); !strings.Contains(w.Body.String(), `"todayKind":"`+kind+`"`) {
			t.Errorf("%s: %s", slug, w.Body.String())
		}
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
	if w = api("GET", "/api/leetgrinder/problem/"+pick, nil); !strings.Contains(w.Body.String(), `"todaysPick":true`) || !strings.Contains(w.Body.String(), `"attemptedToday":true`) || !strings.Contains(w.Body.String(), `"todayKind":"review"`) {
		t.Fatalf("pick status: %s", w.Body.String())
	}
}

func TestLeetgrinderTimezoneChangePreview(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	settings, _ := db.LeetgrinderSettings(ctx)
	if _, err := db.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error { v.Timezone = "UTC"; return nil }); err != nil {
		t.Fatal(err)
	}
	// Two new problems on Oct 8 at 20:00 UTC and two on Oct 9 at 10:00 UTC:
	// two goal-met days in UTC, but all four land on Oct 9 in Tokyo.
	for i, slug := range []string{"two-sum", "valid-anagram", "binary-search", "isomorphic-strings"} {
		id := uuid.NewString()
		if _, err := db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: id, ProblemSlug: slug, Outcome: "solved", Minutes: 20, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"}, ""); err != nil {
			t.Fatal(err)
		}
		at := time.Date(2026, 10, 8, 20, 0, 0, 0, time.UTC)
		if i >= 2 {
			at = time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
		}
		if _, err := db.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1 WHERE id=$2", at, id); err != nil {
			t.Fatal(err)
		}
	}
	current := func() leetgrinder.Settings {
		v, err := db.LeetgrinderSettings(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	form := func(zone string, extra ...string) url.Values {
		v := url.Values{"timezone": {zone}, "goalNew": {"2"}, "goalReview": {"1"}, "revision": {current().Revision}}
		for _, e := range extra {
			v.Set(e, "1")
		}
		return v
	}
	before := current()

	w := request("POST", "/leetgrinder/settings/general", form("Asia/Tokyo"))
	body := w.Body.String()
	if w.Code != 200 || !strings.Contains(body, "Confirm the time zone change") || !strings.Contains(body, "recounts past days") || !strings.Contains(body, `name="confirm" value="1"`) {
		t.Fatalf("no preview: %d %s", w.Code, body)
	}
	row := func(label string) string {
		m := regexp.MustCompile(`(?s)` + label + `</th>\s*<td>([^<]*)</td>\s*<td>([^<]*)</td>`).FindStringSubmatch(body)
		if m == nil {
			t.Fatalf("no %q row in %s", label, body)
		}
		return m[1] + " -> " + m[2]
	}
	for label, want := range map[string]string{"Current streak": "2 days -> 1 day", "Longest streak": "2 days -> 1 day", "Goal-met days, last 8 weeks": "2 -> 1"} {
		if got := row(label); got != want {
			t.Errorf("%s: got %q, want %q", label, got, want)
		}
	}
	if after := current(); after.Timezone != "UTC" || after.Revision != before.Revision {
		t.Fatalf("preview saved: %+v", after)
	}
	for _, date := range []string{"2026-10-10", "2026-10-09"} {
		d, _ := time.Parse(time.DateOnly, date)
		if _, ok, _ := db.LeetgrinderDailyGoal(ctx, d); ok {
			t.Errorf("preview froze a goal for %s", date)
		}
	}

	// A stale revision on the previewing form is a conflict, not a preview.
	stale := form("Asia/Tokyo")
	stale.Set("revision", uuid.NewString())
	if w := request("POST", "/leetgrinder/settings/general", stale); w.Code != 409 {
		t.Fatalf("stale preview: %d", w.Code)
	}
	stale.Set("confirm", "1")
	if w := request("POST", "/leetgrinder/settings/general", stale); w.Code != 409 || current().Timezone != "UTC" {
		t.Fatalf("stale confirm: %d", w.Code)
	}

	// Saving only the goal in the same zone does not ask.
	same := form("UTC")
	same.Set("goalNew", "3")
	if w := request("POST", "/leetgrinder/settings/general", same); w.Code != 303 || current().Goal.New != 3 {
		t.Fatalf("same zone: %d %s", w.Code, w.Body.String())
	}

	// Confirming saves the zone.
	if w := request("POST", "/leetgrinder/settings/general", form("Asia/Tokyo", "confirm")); w.Code != 303 || current().Timezone != "Asia/Tokyo" {
		t.Fatalf("confirm: %d %s", w.Code, w.Body.String())
	}

	// An invalid zone still gets the usual rejection, not a preview.
	if w := request("POST", "/leetgrinder/settings/general", form("Nowhere/Land")); w.Code != 400 || strings.Contains(w.Body.String(), "Confirm the time zone change") {
		t.Fatalf("invalid zone: %d", w.Code)
	}
}

// zoneFixture saves UTC settings with frozen goals and a frozen pick keyed by
// old dates, and attempts that move across midnight in Tokyo. now is Oct 10
// 20:00 UTC, which is already Oct 11 in Tokyo.
func zoneFixture(t *testing.T) (db interface {
	LeetgrinderSettings(context.Context) (leetgrinder.Settings, error)
	LeetgrinderToday(context.Context, time.Time) (leetgrinder.Today, error)
	LeetgrinderDailyGoal(context.Context, time.Time) (leetgrinder.DailyGoal, bool, error)
}, now time.Time, post func(url.Values) *httptest.ResponseRecorder) {
	s, store, request := leetgrinderTestServer(t)
	ctx := context.Background()
	now = time.Date(2026, 10, 10, 20, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	settings, _ := store.LeetgrinderSettings(ctx)
	if _, err := store.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error { v.Timezone = "UTC"; return nil }); err != nil {
		t.Fatal(err)
	}
	add := func(slug string, at time.Time) {
		id := uuid.NewString()
		if _, err := store.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: id, ProblemSlug: slug, Outcome: "struggled", Minutes: 20, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"}, ""); err != nil {
			t.Fatal(err)
		}
		if _, err := store.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1 WHERE id=$2", at, id); err != nil {
			t.Fatal(err)
		}
	}
	utc := func(d, h int) time.Time { return time.Date(2026, 10, d, h, 0, 0, 0, time.UTC) }
	add("two-sum", utc(1, 12))
	add("valid-anagram", utc(1, 13))
	add("binary-search", utc(8, 22))
	add("isomorphic-strings", utc(9, 20))
	add("ransom-note", utc(9, 21))
	for _, g := range []string{"INSERT INTO leetgrinder_daily_goal(local_date,goal_new,goal_review) VALUES('2026-10-08',1,0)", "INSERT INTO leetgrinder_daily_goal(local_date,goal_new,goal_review) VALUES('2026-10-09',2,1)", "INSERT INTO leetgrinder_review_plan(plan_date,problem_slug,slot) VALUES('2026-10-09','two-sum',1)"} {
		if _, err := store.DB.Exec(g); err != nil {
			t.Fatal(err)
		}
	}
	post = func(v url.Values) *httptest.ResponseRecorder {
		cur, _ := store.LeetgrinderSettings(ctx)
		v.Set("revision", cur.Revision)
		return request("POST", "/leetgrinder/settings/general", v)
	}
	return store, now, post
}

// previewFigures reads the three figures of a table row from a preview page.
func previewFigures(t *testing.T, body string) (before, after [3]int) {
	t.Helper()
	for i, label := range []string{"Current streak", "Longest streak", "Goal-met days, last 8 weeks"} {
		m := regexp.MustCompile(`(?s)` + label + `</th>\s*<td>(\d+)[^<]*</td>\s*<td>(\d+)[^<]*</td>`).FindStringSubmatch(body)
		if m == nil {
			t.Fatalf("no %q row in %s", label, body)
		}
		before[i], _ = strconv.Atoi(m[1])
		after[i], _ = strconv.Atoi(m[2])
	}
	return
}

func todayFigures(t *testing.T, today leetgrinder.Today) [3]int {
	t.Helper()
	met := 0
	for _, d := range today.Calendar(leetgrinder.ZoneWeeks * 7) {
		if d.Met {
			met++
		}
	}
	return [3]int{today.Streaks.Current, today.Streaks.Longest, met}
}

func TestLeetgrinderTimezonePreviewMatchesSavedResult(t *testing.T) {
	ctx := context.Background()
	values := func() url.Values {
		return url.Values{"timezone": {"Asia/Tokyo"}, "goalNew": {"1"}, "goalReview": {"2"}}
	}
	day := func(s string) time.Time { d, _ := time.Parse(time.DateOnly, s); return d }

	// After: the dashboard once the change is confirmed matches the preview.
	db, now, post := zoneFixture(t)
	w := post(values())
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Confirm the time zone change") {
		t.Fatalf("preview: %d", w.Code)
	}
	b0, after := previewFigures(t, w.Body.String())
	t.Logf("figures %v -> %v", b0, after)
	for _, d := range []string{"2026-10-10", "2026-10-11"} {
		if _, ok, _ := db.LeetgrinderDailyGoal(ctx, day(d)); ok {
			t.Errorf("preview froze %s", d)
		}
	}
	confirm := values()
	confirm.Set("confirm", "1")
	if w := post(confirm); w.Code != 303 {
		t.Fatalf("confirm: %d", w.Code)
	}
	today, err := db.LeetgrinderToday(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if got := todayFigures(t, today); got != after {
		t.Errorf("after the save the dashboard shows %v, the preview said %v", got, after)
	}

	// Before: the dashboard in the saved zone matches the preview's left side.
	db, now, post = zoneFixture(t)
	w = post(values())
	before, _ := previewFigures(t, w.Body.String())
	if today, err = db.LeetgrinderToday(ctx, now); err != nil {
		t.Fatal(err)
	}
	if got := todayFigures(t, today); got != before {
		t.Errorf("before the change the dashboard shows %v, the preview said %v", got, before)
	}
}

func TestLeetgrinderTimezoneDraftEscapedAndUnreadable(t *testing.T) {
	_, _, post := zoneFixture(t)
	w := post(url.Values{"timezone": {`Foo"><b>x`}, "goalNew": {"2"}, "goalReview": {"1"}})
	if w.Code != 400 || strings.Contains(w.Body.String(), `"><b>x`) || !strings.Contains(w.Body.String(), "Foo&#34;&gt;&lt;b&gt;x") {
		t.Fatalf("zone not escaped: %d", w.Code)
	}
}
