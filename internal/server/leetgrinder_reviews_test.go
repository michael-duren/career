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

func TestLeetgrinderGeneralSettings(t *testing.T) {
	_, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
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
	if w := save(url.Values{"timezone": {"Asia/Tokyo"}, "hours": {"3.0"}}); w.Code != 303 || w.Header().Get("Location") != "/leetgrinder/settings?saved=general#general" {
		t.Fatalf("save: %d %s", w.Code, w.Body.String())
	}
	if settings, _ := db.LeetgrinderSettings(ctx); settings.Timezone != "Asia/Tokyo" || settings.DailyHours != 3 {
		t.Fatalf("saved settings: %+v", settings)
	}
	for _, test := range []struct {
		values url.Values
		want   int
		text   string
	}{
		{url.Values{"timezone": {"Local"}, "hours": {"2.0"}}, 400, "IANA"},
		{url.Values{"timezone": {"America/Chicago"}, "hours": {"4.5"}}, 400, "half-hour"},
		{url.Values{"timezone": {"America/Chicago"}, "hours": {"2.0"}, "revision": {uuid.NewString()}}, 409, "Reload settings"},
	} {
		w := save(test.values)
		if w.Code != test.want || !strings.Contains(w.Body.String(), test.text) || !strings.Contains(w.Body.String(), test.values.Get("timezone")) {
			t.Errorf("%v: %d %s", test.values, w.Code, w.Body.String())
		}
	}
	if settings, _ := db.LeetgrinderSettings(ctx); settings.Timezone != "Asia/Tokyo" {
		t.Fatal("rejected save changed the settings")
	}
	if w := request("POST", "/leetgrinder/settings/schedule", url.Values{"revision": {revision()}}); w.Code != 405 && w.Code != 404 {
		t.Fatalf("retired schedule form: %d", w.Code)
	}
}

func TestLeetgrinderReviews(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	chicago, _ := time.LoadLocation("America/Chicago")
	now := time.Date(2026, 10, 10, 20, 0, 0, 0, chicago)
	s.now = func() time.Time { return now }
	settings, _ := db.LeetgrinderSettings(ctx)
	if _, err := db.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error { v.DailyHours = 2.5; return nil }); err != nil {
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
	// 2.5 hours gives two review slots; the third due card is in the optional list.
	for _, want := range []string{"Struggled 20 days ago", `id="review-isomorphic-strings"`, `id="review-some-unseeded-problem"`, "Optional: 1 more due by the end of today.", `href="/leetgrinder/problem/two-sum"`} {
		if !strings.Contains(body, want) {
			t.Errorf("overview missing %q", want)
		}
	}
	id := regexp.MustCompile(`(?s)id="review-isomorphic-strings".*?name="id" value="([^"]+)"`).FindStringSubmatch(body)
	if id == nil {
		t.Fatal("review form missing")
	}
	w = request("POST", "/leetgrinder/problem/isomorphic-strings/attempts", url.Values{"id": {id[1]}, "outcome": {"solved"}, "minutes": {"20"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(n)"}, "review": {"true"}, "return": {"overview"}})
	if w.Code != 303 || w.Header().Get("Location") != "/leetgrinder#review-isomorphic-strings" {
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
	if body = w.Body.String(); w.Code != 200 || !strings.Contains(body, "Reviewed today") || !strings.Contains(body, "All cards") || !strings.Contains(body, "some-unseeded-problem") {
		t.Fatalf("review queue: %d", w.Code)
	}
	// An unknown return value falls back to the problem page.
	w = request("POST", "/leetgrinder/problem/two-sum/attempts", url.Values{"id": {uuid.NewString()}, "outcome": {"solved"}, "minutes": {"20"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(n)"}, "review": {"true"}, "return": {"https://evil.com"}})
	if w.Code != 303 || w.Header().Get("Location") != "/leetgrinder/problem/two-sum" {
		t.Fatalf("review return: %s", w.Header().Get("Location"))
	}
}
