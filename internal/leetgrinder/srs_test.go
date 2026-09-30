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

// cards builds cards for attempts over the test catalog.
func cards(attempts []Attempt, loc *time.Location) []Card {
	return BuildCards(State{Attempts: attempts, Problems: testProblems}, loc)
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
	collapsed := cards([]Attempt{
		attempt("two-sum", "struggled", 30, false, morning.Add(10*time.Hour)),
		attempt("two-sum", "solved", 10, false, morning),
	}, chicago)
	struggledOnly := cards([]Attempt{attempt("two-sum", "struggled", 30, false, morning.Add(10*time.Hour))}, chicago)
	if len(collapsed) != 1 || collapsed[0].Reviews != 1 || collapsed[0].Last.Outcome != "struggled" || !collapsed[0].Due.Equal(struggledOnly[0].Due) {
		t.Fatalf("same-day attempts not collapsed: %+v", collapsed)
	}
	// 23:30 and 00:30 local are different days even though they are an hour apart.
	split := cards([]Attempt{
		attempt("two-sum", "struggled", 30, false, time.Date(2026, 10, 1, 23, 30, 0, 0, chicago)),
		attempt("two-sum", "solved", 20, false, time.Date(2026, 10, 2, 0, 30, 0, 0, chicago)),
	}, chicago)
	if split[0].Reviews != 2 {
		t.Fatalf("local days merged: %d", split[0].Reviews)
	}
	// Mon 20:00 and Tue 18:00 in Chicago share a UTC date; FSRS must still see
	// one elapsed day, exactly as it does for the same local times in UTC.
	chicagoDays := cards([]Attempt{
		attempt("two-sum", "struggled", 30, false, time.Date(2026, 10, 5, 20, 0, 0, 0, chicago)),
		attempt("two-sum", "solved", 20, false, time.Date(2026, 10, 6, 18, 0, 0, 0, chicago)),
	}, chicago)
	utcDays := cards([]Attempt{
		attempt("two-sum", "struggled", 30, false, time.Date(2026, 10, 5, 20, 0, 0, 0, time.UTC)),
		attempt("two-sum", "solved", 20, false, time.Date(2026, 10, 6, 18, 0, 0, 0, time.UTC)),
	}, time.UTC)
	if got, want := Date(chicagoDays[0].Due, chicago), Date(utcDays[0].Due, time.UTC); !got.Equal(want) {
		t.Fatalf("local due %s, want %s", got, want)
	}
	// A correction from unfinished to easy solved pushes the due date out.
	before := cards([]Attempt{attempt("two-sum", "unfinished", 25, false, morning)}, chicago)
	after := cards([]Attempt{attempt("two-sum", "solved", 12, false, morning)}, chicago)
	if !after[0].Due.After(before[0].Due) || before[0].Due.Sub(morning) > 48*time.Hour {
		t.Fatalf("correction did not move due: %s -> %s", before[0].Due, after[0].Due)
	}
	if r := after[0].Retrievability(after[0].Due.Add(30 * 24 * time.Hour)); r <= 0 || r >= after[0].Retrievability(morning.Add(24*time.Hour)) {
		t.Fatalf("recall should decay: %v", r)
	}
	// Any problem gets a card; one missing from the catalog has just its slug.
	if cards := cards([]Attempt{attempt("not-in-catalog", "solved", 5, false, morning)}, chicago); len(cards) != 1 || cards[0].Problem.Slug != "not-in-catalog" || cards[0].Problem.Known() {
		t.Fatalf("unknown slug card: %+v", cards)
	}
	if cards := cards([]Attempt{attempt("two-sum", "solved", 5, false, morning)}, chicago); cards[0].Problem.Title != "Two Sum" {
		t.Fatalf("card problem not from catalog: %+v", cards[0].Problem)
	}
}

func TestPlanReviews(t *testing.T) {
	loc := time.UTC
	date := day("2026-11-01")
	old := date.AddDate(0, 0, -30)
	all := cards([]Attempt{
		attempt("two-sum", "solved", 10, false, old),                // easy: highest recall
		attempt("binary-search", "unfinished", 25, false, old),      // again
		attempt("isomorphic-strings", "unfinished", 25, false, old), // again: ties binary-search
		attempt("ransom-note", "struggled", 25, false, old),         // hard
		attempt("valid-anagram", "solved", 10, false, date),         // first attempted today: never picked
	}, loc)
	plan := PlanReviews(all, date, loc, 3, nil)
	want := []string{"binary-search", "isomorphic-strings", "ransom-note"}
	if !reflect.DeepEqual(plan, want) {
		t.Fatalf("plan = %v, want %v", plan, want)
	}
	// The plan is frozen: new state never reorders or removes existing picks.
	if got := PlanReviews(nil, date, loc, 1, want); !reflect.DeepEqual(got, want) {
		t.Fatalf("frozen plan changed: %v", got)
	}
	// Fewer slots never drop picks; more slots add to the end.
	if got := PlanReviews(all, date, loc, 1, want[:1]); !reflect.DeepEqual(got, want[:1]) {
		t.Fatalf("plan without new slots: %v", got)
	}
	if got := PlanReviews(all, date, loc, 3, []string{"two-sum"}); !reflect.DeepEqual(got, []string{"two-sum", "binary-search", "isomorphic-strings"}) {
		t.Fatalf("topped-up plan: %v", got)
	}
	// Nothing due in the future is picked.
	if got := PlanReviews(all, old, loc, 5, nil); len(got) != 0 {
		t.Fatalf("planned undue cards: %v", got)
	}
}

func TestTodayReviewsAndDueList(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 10, 20, 18, 0, 0, 0, loc)
	settings := DefaultSettings()
	settings.Timezone = "UTC"
	state := State{Problems: testProblems, Attempts: []Attempt{
		attempt("two-sum", "solved", 12, false, now.Add(-time.Hour)),
		attempt("two-sum", "struggled", 30, false, now.AddDate(0, 0, -9)),
		attempt("binary-search", "unfinished", 25, true, now.AddDate(0, 0, -3)),
		attempt("ransom-note", "unfinished", 25, true, now.AddDate(0, 0, -5)),
		attempt("valid-anagram", "unfinished", 25, true, now.AddDate(0, 0, -5)),
	}}
	state.Plans = map[time.Time][]string{Date(now, loc): {"two-sum", "binary-search"}}
	today := NewToday(settings, state, now)
	if len(today.Reviews) != 2 || !today.Reviews[0].Done || today.Reviews[1].Done || today.Reviews[0].Slot != 1 || today.Reviews[0].Problem.Title != "Two Sum" {
		t.Fatalf("reviews: %+v", today.Reviews)
	}
	if reason := today.Reviews[0].Reason; !strings.HasPrefix(reason, "Struggled 9 days ago · recall estimate ") || !strings.HasSuffix(reason, "%") {
		t.Fatalf("reason %q describes today's attempt or is malformed", reason)
	}
	if missing := today.MissingReviews(); len(missing) != 1 || missing[0].Problem.Slug != "binary-search" {
		t.Fatalf("missing: %+v", missing)
	}
	due := today.DueOptional()
	if len(due) != 2 || today.Backlog() != 2 {
		t.Fatalf("due list: %+v", due)
	}
	for _, c := range due {
		if c.Problem.Slug == "two-sum" || c.Problem.Slug == "binary-search" {
			t.Fatalf("due list has a planned or attempted card: %s", c.Problem.Slug)
		}
	}
	state.Attempts = append(state.Attempts, attempt("binary-search", "solved", 20, false, now))
	if m := NewToday(settings, state, now).MissingReviews(); len(m) != 0 {
		t.Fatalf("finished reviews still missing %+v", m)
	}
}
