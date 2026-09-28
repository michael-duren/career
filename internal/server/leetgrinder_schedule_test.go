package server

import (
	"context"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderScheduleAndReviews(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	chicago, _ := time.LoadLocation("America/Chicago")
	now := time.Date(2026, 10, 10, 20, 0, 0, 0, chicago)
	s.now = func() time.Time { return now }

	w := request("GET", "/leetgrinder", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Set your schedule") {
		t.Fatalf("unset overview: %d", w.Code)
	}
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
		return request("POST", "/leetgrinder/settings/schedule", values)
	}
	schedule := func() (string, string) {
		settings, _ := db.LeetgrinderSettings(ctx)
		f := leetgrinder.NewScheduleForm(settings)
		return f.Start, f.End
	}

	// Setting only the start derives the end.
	if w = save(url.Values{"start": {"2026-10-01"}, "timezone": {"America/Chicago"}, "hours": {"3.0"}}); w.Code != 303 {
		t.Fatalf("save start: %d %s", w.Code, w.Body.String())
	}
	if start, end := schedule(); start != "2026-10-01" || end != "2026-12-23" {
		t.Fatalf("start edit: %s %s", start, end)
	}
	// Changing only the end moves the start.
	if w = save(url.Values{"start": {"2026-10-01"}, "end": {"2026-12-30"}, "timezone": {"America/Chicago"}, "hours": {"3.0"}}); w.Code != 303 {
		t.Fatalf("save end: %d %s", w.Code, w.Body.String())
	}
	if start, end := schedule(); start != "2026-10-08" || end != "2026-12-30" {
		t.Fatalf("end edit: %s %s", start, end)
	}
	// Changing only the start moves the end.
	if w = save(url.Values{"start": {"2026-10-01"}, "end": {"2026-12-30"}, "timezone": {"America/Chicago"}, "hours": {"3.0"}}); w.Code != 303 {
		t.Fatalf("save start again: %d", w.Code)
	}
	if start, end := schedule(); start != "2026-10-01" || end != "2026-12-23" {
		t.Fatalf("second start edit: %s %s", start, end)
	}
	for _, test := range []struct {
		values url.Values
		want   int
		text   string
	}{
		{url.Values{"start": {"2026-10-05"}, "end": {"2026-10-20"}, "timezone": {"America/Chicago"}, "hours": {"2.0"}}, 400, "not both"},
		{url.Values{"start": {"10/05/2026"}, "timezone": {"America/Chicago"}, "hours": {"2.0"}}, 400, "YYYY-MM-DD"},
		{url.Values{"start": {"2026-10-01"}, "timezone": {"Local"}, "hours": {"2.0"}}, 400, "IANA"},
		{url.Values{"start": {"2026-10-01"}, "timezone": {"America/Chicago"}, "hours": {"4.5"}}, 400, "half-hour"},
		{url.Values{"start": {"2026-10-01"}, "timezone": {"America/Chicago"}, "hours": {"2.0"}, "revision": {uuid.NewString()}}, 409, "Reload settings"},
	} {
		w = save(test.values)
		if w.Code != test.want || !strings.Contains(w.Body.String(), test.text) || !strings.Contains(w.Body.String(), test.values.Get("timezone")) {
			t.Errorf("%v: %d %s", test.values, w.Code, w.Body.String())
		}
	}
	if start, _ := schedule(); start != "2026-10-01" {
		t.Fatal("rejected save changed the schedule")
	}

	// Day 10 is today; nothing is finished, so the overview reports 10 behind.
	old := now.AddDate(0, 0, -20)
	for _, slug := range []string{"isomorphic-strings", "two-sum"} {
		if _, err := db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: slug, Outcome: "struggled", Minutes: 30, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"}, ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1", old); err != nil {
		t.Fatal(err)
	}
	w = request("GET", "/leetgrinder", nil)
	body := w.Body.String()
	for _, want := range []string{"10 sessions behind", "Today is Day 10 of 84.", "Struggled 20 days ago", `id="review-isomorphic-strings"`, `id="review-two-sum"`} {
		if !strings.Contains(body, want) {
			t.Errorf("overview missing %q", want)
		}
	}
	w = request("GET", "/leetgrinder/day/10", nil)
	if !strings.Contains(w.Body.String(), `id="review-isomorphic-strings"`) {
		t.Fatal("today's session page has no review section")
	}
	if w = request("GET", "/leetgrinder/day/30", nil); strings.Contains(w.Body.String(), `id="reviews-title"`) {
		t.Fatal("unrelated session page shows today's reviews")
	}
	id := regexp.MustCompile(`(?s)id="review-two-sum".*?name="id" value="([^"]+)"`).FindStringSubmatch(body)
	if id == nil {
		t.Fatal("review form missing")
	}
	w = request("POST", "/leetgrinder/problem/two-sum/attempts", url.Values{"id": {id[1]}, "outcome": {"solved"}, "minutes": {"20"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(n)"}, "review": {"true"}, "return": {"overview"}, "returnDay": {"0"}})
	if w.Code != 303 || w.Header().Get("Location") != "/leetgrinder#review-two-sum" {
		t.Fatalf("review save: %d %s", w.Code, w.Header().Get("Location"))
	}
	if _, err := db.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1 WHERE id=$2", now.Add(-time.Hour), id[1]); err != nil {
		t.Fatal(err)
	}
	state, err := db.LeetgrinderState(ctx)
	if err != nil || !state.Attempts[0].IsReview || state.Attempts[0].Source != "web" {
		t.Fatalf("review attempt: %+v %v", state.Attempts[0], err)
	}
	w = request("GET", "/leetgrinder/reviews", nil)
	if body = w.Body.String(); w.Code != 200 || !strings.Contains(body, "Reviewed today") || !strings.Contains(body, "All cards") {
		t.Fatalf("review queue: %d", w.Code)
	}
	w = request("POST", "/leetgrinder/problem/isomorphic-strings/attempts", url.Values{"id": {uuid.NewString()}, "outcome": {"solved"}, "minutes": {"20"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(n)"}, "review": {"true"}, "return": {"https://evil.com"}, "returnDay": {"10"}})
	if w.Code != 303 || w.Header().Get("Location") != "/leetgrinder/day/10#review-isomorphic-strings" {
		t.Fatalf("review return: %s", w.Header().Get("Location"))
	}

	// Clearing both dates removes the schedule.
	if w = save(url.Values{"timezone": {"UTC"}, "hours": {"2.0"}}); w.Code != 303 {
		t.Fatalf("clear: %d", w.Code)
	}
	if start, _ := schedule(); start != "" {
		t.Fatal("schedule not cleared")
	}
}
