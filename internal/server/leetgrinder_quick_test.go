package server

import (
	"context"
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

// seedStruggles saves a struggled attempt on each slug, backdated by days.
func seedStruggles(t *testing.T, db *database.Store, at time.Time, slugs ...string) {
	t.Helper()
	ctx := context.Background()
	for _, slug := range slugs {
		a := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: slug, Outcome: "struggled", Minutes: 30, TimeComplexity: "O(n)", SpaceComplexity: "O(n)", Notes: "remember the hash map", Code: "def f(): pass", CodeLanguage: "python3"}
		if _, err := db.SaveLeetgrinderAttempt(ctx, a, ""); err != nil {
			t.Fatal(err)
		}
		if _, err := db.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1,marked_at=$1 WHERE id=$2", at, a.ID); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLeetgrinderQuickReviewFlow(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	// Saved attempts get the database's time, so the server runs on the
	// real clock.
	settings, _ := db.LeetgrinderSettings(ctx)
	if _, err := db.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error { v.Timezone = "UTC"; return nil }); err != nil {
		t.Fatal(err)
	}
	draw := 0.99 // never a spot check
	s.rand = func() float64 { return draw }
	seedStruggles(t, db, time.Now().AddDate(0, 0, -20), "two-sum", "valid-anagram", "binary-search")

	reviews := request("GET", "/leetgrinder/reviews", nil).Body.String()
	for _, want := range []string{`href="/leetgrinder/problem/two-sum/quick"`, "Clear backlog (3)", "Full suggested"} {
		if !strings.Contains(reviews, want) {
			t.Errorf("reviews page missing %q", want)
		}
	}
	w := request("GET", "/leetgrinder/problem/two-sum/quick", nil)
	body := w.Body.String()
	if w.Code != 200 || !strings.Contains(body, `name="step" value="reveal"`) || strings.Contains(body, "def f()") || strings.Contains(body, "remember the hash map") {
		t.Fatalf("answer step must hide the solution: %d", w.Code)
	}
	id := regexp.MustCompile(`name="id" value="([^"]+)"`).FindStringSubmatch(body)[1]
	answer := url.Values{"id": {id}, "approach": {"hash map of complements"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(1)"}}

	// Complexity is required before Reveal.
	missing := url.Values{"id": {id}, "step": {"reveal"}, "timeComplexity": {"O(n)"}}
	if w = request("POST", "/leetgrinder/problem/two-sum/quick", missing); w.Code != 400 || !strings.Contains(w.Body.String(), "Choose the time and space complexity") {
		t.Fatalf("missing complexity: %d", w.Code)
	}
	reveal := url.Values{"step": {"reveal"}}
	for k, v := range answer {
		reveal[k] = v
	}
	w = request("POST", "/leetgrinder/problem/two-sum/quick", reveal)
	body = w.Body.String()
	for _, want := range []string{"remember the hash map", "def", "matches the optimum", "optimum is O(n)", `name="step" value="grade"`, `<option value="space" selected`, "neetcode.io/problems/"} {
		if !strings.Contains(body, want) {
			t.Errorf("reveal missing %q", want)
		}
	}
	if state, _ := db.LeetgrinderState(ctx); len(state.Attempts) != 3 {
		t.Fatal("reveal must not save")
	}

	grade := func(slug string, values url.Values) (int, string) {
		values.Set("step", "grade")
		if values.Get("minutes") == "" {
			values.Set("minutes", "3")
		}
		w := request("POST", "/leetgrinder/problem/"+slug+"/quick", values)
		return w.Code, w.Header().Get("Location")
	}
	// Recalled: the problem leaves the due and flagged lists in this request.
	if code, loc := grade("two-sum", url.Values{"id": {id}, "grade": {"recalled"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(n)"}, "approach": {"hash map"}}); code != 303 || loc != "/leetgrinder/reviews#review-two-sum" {
		t.Fatalf("recalled: %d %s", code, loc)
	}
	today, err := db.LeetgrinderToday(ctx, s.clock())
	if err != nil {
		t.Fatal(err)
	}
	card, _ := today.Card("two-sum")
	if today.Due(card) || card.Flagged() || today.QuickReviews != 1 || today.Progress.Reviews() != 0 {
		t.Fatalf("recalled card due %v flagged %v quick %d", today.Due(card), card.Flagged(), today.QuickReviews)
	}
	if !strings.Contains(request("GET", "/leetgrinder", nil).Body.String(), "Quick reviews: <strong>1</strong>") {
		t.Error("dashboard quick counter")
	}
	history := request("GET", "/leetgrinder/problem/two-sum", nil).Body.String()
	if !strings.Contains(history, "Quick review · Recalled") || !strings.Contains(history, "Recalled approach: ") || !strings.Contains(history, "hash map") {
		t.Error("history must show the quick review")
	}
	// Retrying the same id with the same values is idempotent, though the
	// problem is no longer due; other values conflict.
	if code, _ := grade("two-sum", url.Values{"id": {id}, "grade": {"recalled"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(n)"}, "approach": {"hash map"}}); code != 303 {
		t.Fatalf("retry: %d", code)
	}
	if code, _ := grade("two-sum", url.Values{"id": {id}, "grade": {"forgot"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(n)"}, "approach": {"hash map"}}); code != 409 {
		t.Fatalf("retry with other values: %d", code)
	}
	if state, _ := db.LeetgrinderState(ctx); state.Attempts[0].Outcome != "recalled" || len(state.Attempts) != 4 {
		t.Fatal("retries must not log again")
	}
	if code, _ := grade("two-sum", url.Values{"id": {uuid.NewString()}, "grade": {"recalled"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(n)"}}); code != 409 {
		t.Fatalf("new quick review of a problem no longer due: %d", code)
	}

	// Partial needs what was missed and keeps the problem flagged.
	if code, _ := grade("valid-anagram", url.Values{"id": {uuid.NewString()}, "grade": {"partial"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(1)"}}); code != 400 {
		t.Fatalf("partial without missed: %d", code)
	}
	if code, _ := grade("valid-anagram", url.Values{"id": {uuid.NewString()}, "grade": {"partial"}, "missed": {"space"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(1)"}}); code != 303 {
		t.Fatalf("partial: %d", code)
	}
	// Forgot blocks quick mode until a full attempt.
	if code, _ := grade("binary-search", url.Values{"id": {uuid.NewString()}, "grade": {"forgot"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(n)"}}); code != 303 {
		t.Fatalf("forgot: %d", code)
	}
	w = request("GET", "/leetgrinder/problem/binary-search/quick", nil)
	if !strings.Contains(w.Body.String(), "needs a full re-solve") || strings.Contains(w.Body.String(), `name="step"`) {
		t.Fatal("forgot problem must block quick review")
	}
	if code, _ := grade("binary-search", url.Values{"id": {uuid.NewString()}, "grade": {"recalled"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(n)"}}); code != 409 {
		t.Fatalf("quick review after forgot: %d", code)
	}
	reviews = request("GET", "/leetgrinder/reviews", nil).Body.String()
	for _, want := range []string{"Space complexity missed in quick review today", "Forgot in quick review today · full re-solve required", "Full re-solve required", "Quick reviews today"} {
		if !strings.Contains(reviews, want) {
			t.Errorf("reviews page missing %q", want)
		}
	}
	// A full attempt lifts the block.
	if w = request("POST", "/leetgrinder/problem/binary-search/attempts", url.Values{"id": {uuid.NewString()}, "outcome": {"struggled"}, "minutes": {"30"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(n)"}}); w.Code != 303 {
		t.Fatalf("full attempt: %d", w.Code)
	}
	if w = request("GET", "/leetgrinder/problem/binary-search/quick", nil); strings.Contains(w.Body.String(), "needs a full re-solve") {
		t.Fatal("full attempt must lift the block")
	}
	// Quick reviews cannot be corrected.
	state, _ := db.LeetgrinderState(ctx)
	for _, a := range state.Attempts {
		if a.Quick() {
			if _, err := db.SaveLeetgrinderAttempt(ctx, a, a.Revision); err != database.ErrInvalid {
				t.Fatalf("correction of a quick review: %v", err)
			}
			break
		}
	}
}

func TestLeetgrinderQuickSpotCheckStaysSecret(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	settings, _ := db.LeetgrinderSettings(ctx)
	if _, err := db.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error { v.Timezone = "UTC"; return nil }); err != nil {
		t.Fatal(err)
	}
	s.rand = func() float64 { return 0 } // always a spot check at a rate above 0
	seedStruggles(t, db, time.Now().AddDate(0, 0, -20), "two-sum")
	w := request("POST", "/leetgrinder/problem/two-sum/quick", url.Values{"id": {uuid.NewString()}, "step": {"grade"}, "grade": {"recalled"}, "minutes": {"3"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(n)"}})
	if w.Code != 303 {
		t.Fatalf("recalled: %d", w.Code)
	}
	state, _ := db.LeetgrinderState(ctx)
	if !state.Attempts[0].SpotCheck {
		t.Fatal("not picked for a spot check")
	}
	// Nothing names it yet: not the pages, the export or the APIs.
	for _, path := range []string{"/leetgrinder", "/leetgrinder/reviews", "/leetgrinder/problem/two-sum", "/leetgrinder/problems", "/leetgrinder/export"} {
		if body := request("GET", path, nil).Body.String(); strings.Contains(strings.ToLower(body), "spot check of") || strings.Contains(body, "spotCheck\":true") || strings.Contains(body, ">Spot check<") {
			t.Errorf("%s reveals the spot check", path)
		}
	}
	today, _ := db.LeetgrinderToday(ctx, s.clock())
	card, _ := today.Card("two-sum")
	if today.Due(card) || card.Flagged() {
		t.Fatal("spot check must leave the due and flagged lists")
	}
	// When it comes due it is a labelled full re-solve.
	s.now = func() time.Time { return card.Due.Add(time.Hour) }
	reviews := request("GET", "/leetgrinder/reviews", nil).Body.String()
	if !strings.Contains(reviews, ">Spot check<") || !strings.Contains(reviews, "Spot check of quick review from ") {
		t.Fatal("due spot check not labelled")
	}
	if w := request("GET", "/leetgrinder/problem/two-sum/quick", nil); !strings.Contains(w.Body.String(), "spot check of an earlier quick review") {
		t.Fatal("due spot check must block quick review")
	}
}

func TestLeetgrinderClearBacklog(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	settings, _ := db.LeetgrinderSettings(ctx)
	if _, err := db.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error { v.Timezone = "UTC"; return nil }); err != nil {
		t.Fatal(err)
	}
	s.rand = func() float64 { return 0.99 }
	seedStruggles(t, db, time.Now().AddDate(0, 0, -20), "two-sum", "valid-anagram")
	w := request("GET", "/leetgrinder/reviews/backlog", nil)
	if w.Code != 303 || !strings.HasPrefix(w.Header().Get("Location"), "/leetgrinder/reviews/backlog?batch=") {
		t.Fatalf("start: %d %s", w.Code, w.Header().Get("Location"))
	}
	start := w.Header().Get("Location")
	batch, _ := url.QueryUnescape(strings.TrimPrefix(start, "/leetgrinder/reviews/backlog?batch="))
	body := request("GET", start, nil).Body.String()
	if !strings.Contains(body, "Flagged: 2 → 2 · Due: 2 → 2") || !strings.Contains(body, "2 left in this batch") {
		t.Fatalf("start counts: %s", body)
	}
	grade := func(slug, outcome, missed string) {
		t.Helper()
		values := url.Values{"id": {uuid.NewString()}, "step": {"grade"}, "grade": {outcome}, "missed": {missed}, "minutes": {"3"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"O(n)"}, "batch": {batch}}
		w := request("POST", "/leetgrinder/problem/"+slug+"/quick", values)
		if w.Code != 303 || w.Header().Get("Location") != start {
			t.Fatalf("%s: %d %s", slug, w.Code, w.Header().Get("Location"))
		}
	}
	grade("two-sum", "recalled", "")
	body = request("GET", start, nil).Body.String()
	if !strings.Contains(body, "Flagged: 2 → 1 · Due: 2 → 1") || !strings.Contains(body, "1 left in this batch") {
		t.Fatalf("live counts: %s", body)
	}
	grade("valid-anagram", "partial", "detail")
	body = request("GET", start, nil).Body.String()
	if !strings.Contains(body, "Batch finished") || !strings.Contains(body, "Flagged: 2 → 1 · Due: 2 → 0") || !strings.Contains(body, "A key detail missed") || strings.Contains(body, "two-sum\">Two Sum") {
		t.Fatalf("summary: %s", body)
	}
}

func TestLeetgrinderSpotCheckRateSetting(t *testing.T) {
	_, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	settings, _ := db.LeetgrinderSettings(ctx)
	if settings.SpotCheckRate != leetgrinder.DefaultSpotCheckRate {
		t.Fatalf("default rate %d", settings.SpotCheckRate)
	}
	values := url.Values{"revision": {settings.Revision}, "timezone": {settings.Timezone}, "goalNew": {"2"}, "goalReview": {"1"}, "spotCheckRate": {"25"}}
	if w := request("POST", "/leetgrinder/settings/general", values); w.Code != 303 {
		t.Fatalf("save: %d %s", w.Code, w.Body.String())
	}
	settings, _ = db.LeetgrinderSettings(ctx)
	if settings.SpotCheckRate != 25 {
		t.Fatalf("rate %d", settings.SpotCheckRate)
	}
	values.Set("revision", settings.Revision)
	values.Set("spotCheckRate", "26")
	if w := request("POST", "/leetgrinder/settings/general", values); w.Code != 400 || !strings.Contains(w.Body.String(), "spot-check rate from 0 to 25%") {
		t.Fatalf("out of range: %d", w.Code)
	}
	// A form without the field keeps the saved rate.
	values.Del("spotCheckRate")
	if w := request("POST", "/leetgrinder/settings/general", values); w.Code != 303 {
		t.Fatalf("save without rate: %d", w.Code)
	}
	if settings, _ = db.LeetgrinderSettings(ctx); settings.SpotCheckRate != 25 {
		t.Fatalf("rate after save without it: %d", settings.SpotCheckRate)
	}
	var exported struct {
		Settings struct {
			SpotCheckRate int `json:"spotCheckRate"`
		} `json:"settings"`
	}
	if w := request("GET", "/leetgrinder/export", nil); json.Unmarshal(w.Body.Bytes(), &exported) != nil || exported.Settings.SpotCheckRate != 25 {
		t.Fatal("export rate")
	}
	if !strings.Contains(request("GET", "/leetgrinder/settings", nil).Body.String(), `<option value="25" selected>25%</option>`) {
		t.Fatal("settings page rate")
	}
}
