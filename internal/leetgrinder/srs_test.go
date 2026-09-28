package leetgrinder

import (
	"reflect"
	"strings"
	"testing"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v4"
)

func attempt(slug, outcome string, minutes int, assisted bool, at time.Time) Attempt {
	return Attempt{ID: slug + at.Format(time.RFC3339), ProblemSlug: slug, Outcome: outcome, Minutes: minutes, Assisted: assisted, CreatedAt: at}
}

func TestReviewRatingMap(t *testing.T) {
	for _, test := range []struct {
		outcome  string
		minutes  int
		assisted bool
		want     fsrs.Rating
	}{
		{"unfinished", 10, false, fsrs.Again},
		{"unfinished", 40, true, fsrs.Again},
		{"struggled", 10, false, fsrs.Hard},
		{"solved", 10, true, fsrs.Hard},
		{"solved", 26, false, fsrs.Good},
		{"solved", 25, false, fsrs.Easy},
	} {
		if got := ReviewRating(Attempt{Outcome: test.outcome, Minutes: test.minutes, Assisted: test.assisted}); got != test.want {
			t.Errorf("%+v: %s", test, got)
		}
	}
}

func TestCardsReplayAttempts(t *testing.T) {
	chicago, _ := time.LoadLocation("America/Chicago")
	morning := time.Date(2026, 10, 1, 9, 0, 0, 0, chicago)
	// Same local day: only the later struggle counts, not the quick solve.
	collapsed := BuildCards([]Attempt{
		attempt("two-sum", "struggled", 30, false, morning.Add(10*time.Hour)),
		attempt("two-sum", "solved", 10, false, morning),
	}, chicago)
	struggledOnly := BuildCards([]Attempt{attempt("two-sum", "struggled", 30, false, morning.Add(10*time.Hour))}, chicago)
	if len(collapsed) != 1 || collapsed[0].Reviews != 1 || collapsed[0].Last.Outcome != "struggled" || !collapsed[0].Due.Equal(struggledOnly[0].Due) {
		t.Fatalf("same-day attempts not collapsed: %+v", collapsed)
	}
	// 23:30 and 00:30 local are different days even though they are an hour apart.
	split := BuildCards([]Attempt{
		attempt("two-sum", "struggled", 30, false, time.Date(2026, 10, 1, 23, 30, 0, 0, chicago)),
		attempt("two-sum", "solved", 20, false, time.Date(2026, 10, 2, 0, 30, 0, 0, chicago)),
	}, chicago)
	if split[0].Reviews != 2 {
		t.Fatalf("local days merged: %d", split[0].Reviews)
	}
	// Mon 20:00 and Tue 18:00 in Chicago share a UTC date; FSRS must still see
	// one elapsed day, exactly as it does for the same local times in UTC.
	chicagoDays := BuildCards([]Attempt{
		attempt("two-sum", "struggled", 30, false, time.Date(2026, 10, 5, 20, 0, 0, 0, chicago)),
		attempt("two-sum", "solved", 20, false, time.Date(2026, 10, 6, 18, 0, 0, 0, chicago)),
	}, chicago)
	utcDays := BuildCards([]Attempt{
		attempt("two-sum", "struggled", 30, false, time.Date(2026, 10, 5, 20, 0, 0, 0, time.UTC)),
		attempt("two-sum", "solved", 20, false, time.Date(2026, 10, 6, 18, 0, 0, 0, time.UTC)),
	}, time.UTC)
	if got, want := Date(chicagoDays[0].Due, chicago), Date(utcDays[0].Due, time.UTC); !got.Equal(want) {
		t.Fatalf("local due %s, want %s", got, want)
	}
	// A correction from unfinished to easy solved pushes the due date out.
	before := BuildCards([]Attempt{attempt("two-sum", "unfinished", 25, false, morning)}, chicago)
	after := BuildCards([]Attempt{attempt("two-sum", "solved", 12, false, morning)}, chicago)
	if !after[0].Due.After(before[0].Due) || before[0].Due.Sub(morning) > 48*time.Hour {
		t.Fatalf("correction did not move due: %s -> %s", before[0].Due, after[0].Due)
	}
	if r := after[0].Retrievability(after[0].Due.Add(30 * 24 * time.Hour)); r <= 0 || r >= after[0].Retrievability(morning.Add(24*time.Hour)) {
		t.Fatalf("recall should decay: %v", r)
	}
	if cards := BuildCards([]Attempt{attempt("not-in-curriculum", "solved", 5, false, morning)}, chicago); len(cards) != 0 {
		t.Fatal("unknown slug became a card")
	}
}

func TestReviewSlots(t *testing.T) {
	// Day 1 has required reading; day 2 only an optional refresher.
	for _, test := range []struct {
		session int
		hours   float64
		want    int
	}{
		{1, 2, 0}, {2, 2, 1}, {0, 2, 1}, {2, 2.5, 2}, {2, 3, 3}, {1, 3.5, 3}, {2, 4, 5},
	} {
		if got := ReviewSlots(test.session, test.hours); got != test.want {
			t.Errorf("session %d at %.1fh: %d, want %d", test.session, test.hours, got, test.want)
		}
	}
}

func TestPlanReviews(t *testing.T) {
	loc := time.UTC
	date := day("2026-11-01")
	old := date.AddDate(0, 0, -30)
	cards := BuildCards([]Attempt{
		attempt("two-sum", "solved", 10, false, old),                // week 1, easy: highest recall
		attempt("binary-search", "unfinished", 25, false, old),      // week 2, again
		attempt("isomorphic-strings", "unfinished", 25, false, old), // week 1, again: ties binary-search
		attempt("ransom-note", "struggled", 25, false, old),         // week 1, hard (day 2 assignment)
		attempt("valid-anagram", "solved", 10, false, date),         // attempted today: not due
	}, loc)
	// Session 2 (optional reading only) at 3 hours: 1 base + 2 extra slots.
	plan := PlanReviews(cards, date, loc, 2, 3, nil)
	want := []string{"isomorphic-strings", "binary-search", "two-sum"}
	if !reflect.DeepEqual(plan, want) {
		t.Fatalf("plan = %v, want %v (ransom-note is assigned today)", plan, want)
	}
	// The plan is frozen: new state never reorders or removes existing picks.
	if got := PlanReviews(nil, date, loc, 2, 2, want); !reflect.DeepEqual(got, want) {
		t.Fatalf("frozen plan changed: %v", got)
	}
	// Fewer hours never drop picks; more hours add to the end.
	if got := PlanReviews(cards, date, loc, 2, 2, want[:1]); !reflect.DeepEqual(got, want[:1]) {
		t.Fatalf("plan without new slots: %v", got)
	}
	if got := PlanReviews(cards, date, loc, 2, 4, []string{"two-sum"}); !reflect.DeepEqual(got, []string{"two-sum", "isomorphic-strings", "binary-search"}) {
		t.Fatalf("topped-up plan: %v", got)
	}
	// Reading day at 2 hours has no slots.
	if got := PlanReviews(cards, date, loc, 1, 2, nil); len(got) != 0 {
		t.Fatalf("reading day planned %v", got)
	}
	// Nothing due in the future is picked.
	if got := PlanReviews(cards, old, loc, 0, 4, nil); len(got) != 0 {
		t.Fatalf("planned undue cards: %v", got)
	}
}

func TestTodayReviewsAndMissingWork(t *testing.T) {
	loc := time.UTC
	start := day("2026-10-01")
	now := time.Date(2026, 10, 20, 18, 0, 0, 0, loc)
	settings := DefaultSettings()
	settings.Timezone, settings.StartDate = "UTC", &start
	state := State{CompletedDays: []int{1, 2}, Attempts: []Attempt{
		attempt("two-sum", "solved", 12, false, now.Add(-time.Hour)),
		attempt("two-sum", "struggled", 30, false, now.AddDate(0, 0, -9)),
		attempt("binary-search", "unfinished", 25, true, now.AddDate(0, 0, -3)),
	}}
	today := NewToday(settings, state, []string{"two-sum", "binary-search"}, now)
	if today.Session != 20 || today.Status() != "18 sessions behind" {
		t.Fatalf("session %d, status %q", today.Session, today.Status())
	}
	if len(today.Reviews) != 2 || !today.Reviews[0].Done || today.Reviews[1].Done || today.Reviews[0].Slot != 1 {
		t.Fatalf("reviews: %+v", today.Reviews)
	}
	if reason := today.Reviews[0].Reason; !strings.HasPrefix(reason, "Struggled 9 days ago · recall estimate ") || !strings.HasSuffix(reason, "%") {
		t.Fatalf("reason %q describes today's attempt or is malformed", reason)
	}
	missing := today.Missing()
	if missing.Session != 20 || len(missing.Reviews) != 1 || missing.Reviews[0].Problem.Slug != "binary-search" || missing.Empty() {
		t.Fatalf("missing: %+v", missing)
	}
	state.CompletedDays = append(state.CompletedDays, 20)
	state.Attempts = append(state.Attempts, attempt("binary-search", "solved", 20, false, now))
	if m := NewToday(settings, state, []string{"two-sum", "binary-search"}, now).Missing(); !m.Empty() {
		t.Fatalf("finished day still missing %+v", m)
	}
	if unset := NewToday(DefaultSettings(), state, nil, now); unset.Session != 0 || unset.Status() != "" || !unset.Missing().Empty() {
		t.Fatal("unset schedule reported work")
	}
}
