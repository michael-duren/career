package leetgrinder

import (
	"reflect"
	"testing"
	"time"
)

func boolPtr(b bool) *bool { return &b }

func TestClassifyNewReviewPractice(t *testing.T) {
	chicago, _ := time.LoadLocation("America/Chicago")
	at := func(day, hour int) time.Time { return time.Date(2026, 10, day, hour, 0, 0, 0, chicago) }
	state := State{Problems: testProblems, Attempts: []Attempt{
		// Two attempts on a new problem the same day count once.
		attempt("two-sum", "unfinished", 25, false, at(1, 9)),
		attempt("two-sum", "struggled", 30, false, at(1, 20)),
		// 23:30 and 00:30 local are different days: the second is a review,
		// due the next day after an Again rating.
		attempt("binary-search", "unfinished", 25, false, at(1, 23).Add(30*time.Minute)),
		attempt("binary-search", "solved", 20, false, at(2, 0).Add(30*time.Minute)),
		// An easy solve is not due tomorrow: re-solving it is practice.
		attempt("valid-anagram", "solved", 5, false, at(1, 10)),
		attempt("valid-anagram", "solved", 5, false, at(2, 10)),
	}}
	history := History(state, chicago)
	day1, day2 := history[Date(at(1, 12), chicago)], history[Date(at(2, 12), chicago)]
	if !reflect.DeepEqual(day1.Kinds, map[string]string{"two-sum": KindNew, "binary-search": KindNew, "valid-anagram": KindNew}) {
		t.Fatalf("day 1: %v", day1.Kinds)
	}
	if !reflect.DeepEqual(day2.Kinds, map[string]string{"binary-search": KindReview, "valid-anagram": KindPractice}) {
		t.Fatalf("day 2: %v", day2.Kinds)
	}
	// Day 1 has no frozen goal, so the default 2 + 1 applies. Nothing was due,
	// so 3 new problems meet it.
	if day1.Goal != DefaultGoal || day1.New() != 3 || !day1.Met() || day1.Bonus() != 1 || day1.Remaining() != 0 {
		t.Fatalf("day 1 progress: new %d met %v bonus %d", day1.New(), day1.Met(), day1.Bonus())
	}
	// In UTC the 23:30 and 00:30 attempts fall on the same day.
	utc := History(state, time.UTC)
	if kinds := utc[Date(at(2, 12), time.UTC)].Kinds; kinds["binary-search"] != KindNew {
		t.Fatalf("UTC days: %v", kinds)
	}

	// Today's pick counts toward the goal even when it was not due.
	state.Plans = map[time.Time][]string{Date(at(2, 12), chicago): {"valid-anagram"}}
	if k := History(state, chicago)[Date(at(2, 12), chicago)].Kinds["valid-anagram"]; k != KindReview {
		t.Fatalf("pick kind: %s", k)
	}
	// A frozen goal replaces the default.
	state.Goals = map[time.Time]DailyGoal{Date(at(2, 12), chicago): {New: 0, Review: 2}}
	day2 = History(state, chicago)[Date(at(2, 12), chicago)]
	if !day2.Met() || day2.Bonus() != 0 {
		t.Fatalf("frozen goal: met %v bonus %d", day2.Met(), day2.Bonus())
	}
}

func TestFlagMakesProblemDueAndCounts(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, loc)
	solve := attempt("two-sum", "solved", 5, false, now.AddDate(0, 0, -3))
	state := State{Problems: testProblems, Attempts: []Attempt{solve}, Analyses: map[string]Analysis{
		solve.ID: {AttemptID: solve.ID, Status: AnalysisDone, Current: true, TimeMatches: boolPtr(false), SpaceMatches: boolPtr(true), Optimal: boolPtr(true), ActualTime: "O(n²)", UpdatedAt: now.AddDate(0, 0, -3).Add(time.Hour)},
	}}
	card := BuildCards(state, loc)[0]
	if card.Flag == nil || !card.Due.Equal(time.Date(2026, 10, 8, 0, 0, 0, 0, loc)) || !card.FSRSDue.After(now) {
		t.Fatalf("flagged card: flag %+v due %s fsrs %s", card.Flag, card.Due, card.FSRSDue)
	}
	if got := card.Flag.Reason(now, loc); got != "Time complexity judged wrong 3 days ago" {
		t.Fatalf("reason %q", got)
	}
	if got := ReviewReason(card, now, loc); got != "Time complexity judged wrong 3 days ago" {
		t.Fatalf("review reason %q", got)
	}
	// A re-solve the day after the analysis counts toward the goal.
	later := attempt("two-sum", "solved", 5, false, now.AddDate(0, 0, -2))
	state.Attempts = []Attempt{later, solve}
	if k := History(state, loc)[Date(later.CreatedAt, loc)].Kinds["two-sum"]; k != KindReview {
		t.Fatalf("flagged re-solve kind %s", k)
	}
	// The next attempt clears the flag: only the latest attempt counts.
	if c := BuildCards(state, loc)[0]; c.Flag != nil {
		t.Fatal("flag survived a newer attempt")
	}
	// A re-analysis that clears the verdict clears the flag.
	state.Attempts = []Attempt{solve}
	a := state.Analyses[solve.ID]
	a.TimeMatches = boolPtr(true)
	state.Analyses[solve.ID] = a
	if c := BuildCards(state, loc)[0]; c.Flag != nil {
		t.Fatal("flag survived a clean analysis")
	}
	// A pending re-analysis has no verdict.
	a.TimeMatches, a.Status = boolPtr(false), AnalysisPending
	state.Analyses[solve.ID] = a
	if c := BuildCards(state, loc)[0]; c.Flag != nil {
		t.Fatal("pending analysis flagged")
	}
	// Not optimal reads the actual against the optimum.
	a.Status, a.TimeMatches, a.Optimal = AnalysisDone, boolPtr(true), boolPtr(false)
	state.Analyses[solve.ID] = a
	c := BuildCards(state, loc)[0]
	if c.Flag == nil || c.Flag.Reason(now, loc) != "Not optimal: O(n²) vs O(n)" || c.Flag.Label() != "Flagged: not optimal" {
		t.Fatalf("not optimal flag %+v", c.Flag)
	}
}

func TestLearnerMarksFlagProblem(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, loc)
	solve := attempt("two-sum", "solved", 5, false, now.AddDate(0, 0, -3))
	state := State{Problems: testProblems, Attempts: []Attempt{solve}}
	if c := BuildCards(state, loc)[0]; c.Flag != nil {
		t.Fatalf("unmarked attempt flagged: %+v", c.Flag)
	}
	optimal := solve
	optimal.Approach = ApproachOptimal
	state.Attempts = []Attempt{optimal}
	if c := BuildCards(state, loc)[0]; c.Flag != nil {
		t.Fatalf("optimal approach flagged: %+v", c.Flag)
	}
	for _, test := range []struct {
		name          string
		mark          func(*Attempt)
		reason, label string
	}{
		{"wants review", func(a *Attempt) { a.WantsReview = true }, "Marked for review 3 days ago", "Flagged: marked for review"},
		{"suboptimal", func(a *Attempt) { a.Approach = ApproachSuboptimal }, "Took a simpler approach 3 days ago", "Flagged: simpler approach taken"},
		{"both", func(a *Attempt) { a.WantsReview, a.Approach = true, ApproachSuboptimal }, "Took a simpler approach 3 days ago", "Flagged: simpler approach taken"},
	} {
		marked := solve
		test.mark(&marked)
		state.Attempts = []Attempt{marked}
		card := BuildCards(state, loc)[0]
		// Due the day after the attempt, well before FSRS would schedule it.
		if card.Flag == nil || !card.Due.Equal(time.Date(2026, 10, 8, 0, 0, 0, 0, loc)) || !card.FSRSDue.After(now) {
			t.Fatalf("%s: flag %+v due %s", test.name, card.Flag, card.Due)
		}
		if got := card.Flag.Reason(now, loc); got != test.reason {
			t.Errorf("%s: reason %q", test.name, got)
		}
		if got := card.Flag.Label(); got != test.label {
			t.Errorf("%s: label %q", test.name, got)
		}
		// A re-solve the next day counts as a review.
		later := attempt("two-sum", "solved", 5, false, now.AddDate(0, 0, -2))
		state.Attempts = []Attempt{later, marked}
		if k := History(state, loc)[Date(later.CreatedAt, loc)].Kinds["two-sum"]; k != KindReview {
			t.Errorf("%s: re-solve kind %s", test.name, k)
		}
		if c := BuildCards(state, loc)[0]; c.Flag != nil {
			t.Errorf("%s: flag survived a newer attempt", test.name)
		}
	}
	// The analysis's verdict reads first; the earlier date raises the flag.
	marked := solve
	marked.WantsReview = true
	state.Attempts = []Attempt{marked}
	state.Analyses = map[string]Analysis{marked.ID: {AttemptID: marked.ID, Status: AnalysisDone, Current: true, TimeMatches: boolPtr(false), SpaceMatches: boolPtr(true), Optimal: boolPtr(true), ActualTime: "O(n²)", UpdatedAt: now.AddDate(0, 0, -1)}}
	c := BuildCards(state, loc)[0]
	if c.Flag == nil || c.Flag.Reason(now, loc) != "Time complexity judged wrong 3 days ago" || !c.Due.Equal(time.Date(2026, 10, 8, 0, 0, 0, 0, loc)) {
		t.Fatalf("combined flag %+v due %s", c.Flag, c.Due)
	}
}

func TestPlanPutsFlaggedFirst(t *testing.T) {
	loc := time.UTC
	date := day("2026-11-01")
	old := date.AddDate(0, 0, -30)
	recent := attempt("two-sum", "solved", 10, false, date.AddDate(0, 0, -2))
	state := State{Problems: testProblems, Attempts: []Attempt{
		recent,
		attempt("binary-search", "unfinished", 25, false, old),
	}, Analyses: map[string]Analysis{recent.ID: {Status: AnalysisDone, Current: true, SpaceMatches: boolPtr(false), UpdatedAt: date.AddDate(0, 0, -2)}}}
	all := BuildCards(state, loc)
	if got := PlanReviews(all, date, loc, 1, nil); !reflect.DeepEqual(got, []string{"two-sum"}) {
		t.Fatalf("plan %v, want the flagged problem first", got)
	}
}

func TestStreaks(t *testing.T) {
	today := day("2026-10-10")
	var attempts []Attempt
	// Goal {1 new, 0 reviews} is met by one new problem a day.
	goals := map[time.Time]DailyGoal{}
	solveOn := func(slug string, d time.Time) {
		attempts = append(attempts, attempt(slug, "solved", 10, false, d.Add(12*time.Hour)))
		goals[d] = DailyGoal{New: 1}
	}
	// Met on 1st–3rd (longest 3), missed 4th, met 7th–9th and today.
	for i, d := range []int{1, 2, 3, 7, 8, 9} {
		solveOn(string(rune('a'+i))+"-problem", today.AddDate(0, 0, d-10))
	}
	settings := DefaultSettings()
	settings.Timezone = "UTC"
	state := State{Problems: testProblems, Attempts: attempts, Goals: goals}
	state.Goals[today] = DailyGoal{New: 1}
	now := today.Add(9 * time.Hour)

	// Before anything today, the streak through yesterday is still alive.
	got := NewToday(settings, state, now).Streaks
	if got.Current != 3 || got.Longest != 3 || got.Active != 3 {
		t.Fatalf("morning streaks %+v", got)
	}
	state.Attempts = append(state.Attempts, attempt("today-problem", "unfinished", 30, false, now))
	got = NewToday(settings, state, now).Streaks
	if got.Current != 4 || got.Longest != 4 || got.Active != 4 {
		t.Fatalf("after today's work %+v", got)
	}
	// A missed day breaks it: two days later with nothing logged.
	later := NewToday(settings, state, now.AddDate(0, 0, 2)).Streaks
	if later.Current != 0 || later.Longest != 4 || later.Active != 0 {
		t.Fatalf("after a missed day %+v", later)
	}
	// Pre-feature days use the default 2 + 1, which one solve does not meet.
	state.Goals = nil
	if got = NewToday(settings, state, now).Streaks; got.Current != 0 || got.Longest != 0 || got.Active != 4 {
		t.Fatalf("default goal streaks %+v", got)
	}
	cal := NewToday(settings, State{Problems: testProblems, Attempts: state.Attempts, Goals: goals}, now).Calendar(5)
	if len(cal) != 5 || !cal[0].Date.Equal(today.AddDate(0, 0, -4)) || cal[0].Met || cal[0].Active || !cal[1].Met || !cal[4].Met || !cal[4].Active {
		t.Fatalf("calendar %+v", cal)
	}
}

func TestTodayKindsAndRemaining(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 10, 10, 18, 0, 0, 0, loc)
	settings := DefaultSettings()
	settings.Timezone = "UTC"
	state := State{Problems: testProblems, Attempts: []Attempt{
		attempt("two-sum", "solved", 10, false, now.Add(-time.Hour)),
		attempt("binary-search", "unfinished", 25, false, now.Add(-2*time.Hour)),
		attempt("binary-search", "unfinished", 25, false, now.AddDate(0, 0, -5)),
		attempt("ransom-note", "unfinished", 25, false, now.AddDate(0, 0, -5)),
	}, Plans: map[time.Time][]string{Date(now, loc): {"binary-search", "ransom-note"}}, Goals: map[time.Time]DailyGoal{Date(now, loc): {New: 2, Review: 2}}}
	today := NewToday(settings, state, now)
	if today.Kind("two-sum") != KindNew || today.Kind("binary-search") != KindReview || today.Kind("valid-anagram") != "" {
		t.Fatalf("kinds %v", today.Progress.Kinds)
	}
	if today.Progress.NewLeft() != 1 || today.Progress.ReviewsLeft() != 1 || today.Progress.Remaining() != 2 || today.Progress.Met() {
		t.Fatalf("remaining %+v", today.Progress)
	}
	if !today.Picked("ransom-note") || today.Picked("two-sum") || len(today.MissingReviews()) != 1 {
		t.Fatal("picks")
	}
}

func TestGoalWithNothingDue(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 10, 10, 18, 0, 0, 0, loc)
	settings := DefaultSettings()
	settings.Timezone = "UTC"
	// Two new problems; the only earlier problem was solved easily yesterday
	// and is not due, so a re-solve of it is practice.
	state := State{Problems: testProblems, Attempts: []Attempt{
		attempt("two-sum", "solved", 10, false, now.Add(-time.Hour)),
		attempt("ransom-note", "solved", 10, false, now.Add(-2*time.Hour)),
		attempt("valid-anagram", "solved", 5, false, now.Add(-3*time.Hour)),
		attempt("valid-anagram", "solved", 5, false, now.AddDate(0, 0, -1)),
	}}
	today := NewToday(settings, state, now)
	p := today.Progress
	if p.Available != 0 || p.ReviewTarget() != 0 || !p.Met() || p.Remaining() != 0 || p.Bonus() != 1 || today.Kind("valid-anagram") != KindPractice {
		t.Fatalf("nothing-due day: available %d met %v remaining %d bonus %d", p.Available, p.Met(), p.Remaining(), p.Bonus())
	}
	if today.Streaks.Current != 1 {
		t.Fatalf("streak %d", today.Streaks.Current)
	}
	// With one card due, the target is 1 again until it is reviewed.
	state.Attempts = append(state.Attempts, attempt("binary-search", "unfinished", 25, false, now.AddDate(0, 0, -10)))
	p = NewToday(settings, state, now).Progress
	if p.Available != 1 || p.ReviewTarget() != 1 || p.Met() || p.ReviewsLeft() != 1 {
		t.Fatalf("one due: available %d met %v", p.Available, p.Met())
	}
	// Recent attempts label a re-solve by its kind.
	if k := NewToday(settings, state, now).KindOf(state.Attempts[2]); k != KindPractice {
		t.Fatalf("KindOf %s", k)
	}
}

func TestZeroTargetsNeedAnAttempt(t *testing.T) {
	now := time.Date(2026, 10, 10, 18, 0, 0, 0, time.UTC)
	settings := DefaultSettings()
	settings.Timezone = "UTC"
	date := Date(now, time.UTC)
	state := State{Problems: testProblems, Goals: map[time.Time]DailyGoal{date: {New: 0, Review: 1}}}
	today := NewToday(settings, state, now)
	if today.Progress.Met() || today.Progress.Remaining() != 1 || today.Streaks.Current != 0 {
		t.Fatalf("empty day: met %v remaining %d", today.Progress.Met(), today.Progress.Remaining())
	}
	state.Attempts = []Attempt{attempt("two-sum", "unfinished", 10, false, now.Add(-time.Hour))}
	if today = NewToday(settings, state, now); !today.Progress.Met() || today.Progress.Remaining() != 0 || today.Streaks.Current != 1 || today.Progress.Target() != (DailyGoal{}) {
		t.Fatalf("one attempt: met %v target %+v", today.Progress.Met(), today.Progress.Target())
	}
}
