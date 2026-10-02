package leetgrinder

import (
	"cmp"
	"fmt"
	"math/rand"
	"reflect"
	"slices"
	"sort"
	"testing"
	"time"
)

// The functions below are the previous implementations, which replayed every
// problem once per BuildCards and History call. They are kept on purpose as
// the oracle the single replay must match exactly: do not delete them or
// rewrite them in terms of the replay.

func oldBuildCards(state State, loc *time.Location) []Card {
	cards := make([]Card, 0)
	for slug, list := range attemptsBySlug(state.Attempts) {
		card := replaySlug(state.Problem(slug), list, loc, nil)
		if card.Reviews == 0 {
			continue
		}
		if f := flagFor(card.Last, card.Problem, state.Analyses, loc); f != nil {
			card.Flag = f
			if due := f.DueAt(loc); due.Before(card.Due) {
				card.Due = due
			}
		}
		cards = append(cards, card)
	}
	slices.SortFunc(cards, func(a, b Card) int {
		return cmp.Or(a.Due.Compare(b.Due), cmp.Compare(a.Problem.Slug, b.Problem.Slug))
	})
	return cards
}

func oldHistory(state State, loc *time.Location, extra ...time.Time) map[time.Time]*DayProgress {
	days := map[time.Time]*DayProgress{}
	day := func(date time.Time) *DayProgress {
		d := days[date]
		if d == nil {
			d = &DayProgress{Date: date, Goal: state.GoalFor(date), Kinds: map[string]string{}}
			days[date] = d
		}
		return d
	}
	for _, date := range extra {
		day(date)
	}
	type snapshot struct {
		date  time.Time
		after Card
	}
	bySlug := attemptsBySlug(state.Attempts)
	history := map[string][]snapshot{}
	for slug, list := range bySlug {
		problem := state.Problem(slug)
		var dates []time.Time
		var befores []Card
		final := replaySlug(problem, list, loc, func(date time.Time, before Card) {
			day(date).Kinds[slug] = classify(state, slug, problem, date, before, loc)
			dates, befores = append(dates, date), append(befores, before)
		})
		for i, date := range dates {
			after := final
			if i+1 < len(befores) {
				after = befores[i+1]
			}
			history[slug] = append(history[slug], snapshot{date, after})
		}
	}
	for date, d := range days {
		count := 0
		for slug, snaps := range history {
			if d.Kinds[slug] == KindReview {
				count++
				continue
			}
			i := sort.Search(len(snaps), func(i int) bool { return !snaps[i].date.Before(date) })
			if i > 0 && dueBy(snaps[i-1].after, state, date, loc) {
				count++
			}
		}
		d.Available = count
	}
	return days
}

func oldNewToday(settings Settings, state State, now time.Time) Today {
	loc := settings.Location()
	t := Today{Settings: settings, State: state, Now: now, Date: Date(now, loc)}
	t.Goal = state.GoalFor(t.Date)
	t.Cards = oldBuildCards(state, loc)
	t.history = oldHistory(state, loc, t.Date)
	t.Progress = *t.history[t.Date]
	t.Streaks = ComputeStreaks(t.history, t.Progress)
	var before State = state
	before.Attempts = nil
	for _, a := range state.Attempts {
		if Date(a.CreatedAt, loc).Before(t.Date) {
			before.Attempts = append(before.Attempts, a)
		}
	}
	cards := map[string]Card{}
	for _, c := range oldBuildCards(before, loc) {
		cards[c.Problem.Slug] = c
	}
	for i, slug := range state.Plans[t.Date] {
		item := ReviewItem{Slot: i + 1, Problem: state.Problem(slug), Card: cards[slug], Done: t.AttemptedOn(slug)}
		if item.Card.Reviews > 0 {
			item.Reason = ReviewReason(item.Card, now, loc)
		}
		t.Reviews = append(t.Reviews, item)
	}
	return t
}

// equivalenceZones cover DST in both hemispheres, a half-hour DST shift and
// fractional offsets, each with an instant near a clock change to cluster
// attempts around. Casey moved from +11 to +08 at 01:00 on 2010-03-05, so
// local dates went back a day.
var equivalenceZones = []struct {
	zone   string
	change time.Time
}{
	{"UTC", time.Date(2026, 3, 8, 8, 0, 0, 0, time.UTC)},
	{"America/Chicago", time.Date(2026, 11, 1, 7, 0, 0, 0, time.UTC)},
	{"America/New_York", time.Date(2026, 3, 8, 7, 0, 0, 0, time.UTC)},
	{"Europe/London", time.Date(2026, 10, 25, 1, 0, 0, 0, time.UTC)},
	{"Australia/Sydney", time.Date(2026, 4, 4, 16, 0, 0, 0, time.UTC)},
	{"Australia/Lord_Howe", time.Date(2026, 10, 3, 15, 30, 0, 0, time.UTC)},
	{"Asia/Kolkata", time.Date(2026, 6, 1, 18, 30, 0, 0, time.UTC)},
	{"Pacific/Auckland", time.Date(2026, 9, 26, 14, 0, 0, 0, time.UTC)},
	{"Antarctica/Casey", time.Date(2010, 3, 4, 15, 0, 0, 0, time.UTC)},
}

// randomState builds 60 days of attempts from base, clustered around local
// midnight and the clock change, with flags, analyses, goals and plans.
func randomState(rng *rand.Rand, loc *time.Location, base, change time.Time) State {
	slugs := []string{"two-sum", "binary-search", "isomorphic-strings", "ransom-note", "valid-anagram", "uncatalogued-problem"}
	state := State{Problems: testProblems, Analyses: map[string]Analysis{}, Plans: map[time.Time][]string{}, Goals: map[time.Time]DailyGoal{}}
	outcomes := []string{"solved", "solved", "solved", "struggled", "unfinished"}
	approaches := []string{"", "", ApproachOptimal, ApproachSuboptimal}
	verdict := func() *bool {
		switch rng.Intn(3) {
		case 0:
			return nil
		case 1:
			return boolPtr(true)
		}
		return boolPtr(false)
	}
	n := rng.Intn(40)
	for i := range n {
		var at time.Time
		switch rng.Intn(4) {
		case 0:
			// Near local midnight.
			d := base.AddDate(0, 0, rng.Intn(60))
			at = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc).Add(time.Duration(rng.Intn(240)-120) * time.Minute)
		case 1:
			at = change.Add(time.Duration(rng.Intn(300)-180) * time.Minute)
		default:
			at = base.Add(time.Duration(rng.Int63n(int64(60 * 24 * time.Hour))))
		}
		a := Attempt{
			ID: fmt.Sprintf("a%03d", i), ProblemSlug: slugs[rng.Intn(len(slugs))], Outcome: outcomes[rng.Intn(len(outcomes))],
			Minutes: 1 + rng.Intn(60), Assisted: rng.Intn(4) == 0, CreatedAt: at, WantsReview: rng.Intn(5) == 0, Approach: approaches[rng.Intn(len(approaches))],
		}
		if a.SelfFlagged() && rng.Intn(2) == 0 {
			a.MarkedAt = at.Add(time.Duration(rng.Intn(72)) * time.Hour)
		}
		state.Attempts = append(state.Attempts, a)
		if rng.Intn(3) == 0 {
			status := AnalysisDone
			if rng.Intn(5) == 0 {
				status = "pending"
			}
			state.Analyses[a.ID] = Analysis{AttemptID: a.ID, Status: status, Current: rng.Intn(6) != 0, TimeMatches: verdict(), SpaceMatches: verdict(), Optimal: verdict(),
				ActualTime: "O(n²)", ActualSpace: "O(n)", UpdatedAt: at.Add(time.Duration(rng.Intn(96)) * time.Hour)}
		}
	}
	// State keeps attempts newest first; equal times are allowed.
	slices.SortStableFunc(state.Attempts, func(a, b Attempt) int { return b.CreatedAt.Compare(a.CreatedAt) })
	for range rng.Intn(20) {
		date := Date(base.AddDate(0, 0, rng.Intn(62)), loc)
		state.Goals[date] = DailyGoal{New: rng.Intn(3), Review: rng.Intn(4)}
		var plan []string
		for range rng.Intn(4) {
			plan = append(plan, slugs[rng.Intn(len(slugs))])
		}
		if len(plan) > 0 {
			state.Plans[date] = plan
		}
	}
	return state
}

func TestSingleReplayMatchesPerCallReplays(t *testing.T) {
	rng := rand.New(rand.NewSource(20261002))
	unordered := 0
	for _, z := range equivalenceZones {
		loc, err := time.LoadLocation(z.zone)
		if err != nil {
			t.Fatal(err)
		}
		settings := Settings{Timezone: z.zone}
		for _, base := range []time.Time{z.change.AddDate(0, 0, -30), z.change.AddDate(0, 0, -5)} {
			for i := range 150 {
				state := randomState(rng, loc, base, z.change)
				for _, r := range replayAll(state, loc) {
					if !r.ordered {
						unordered++
					}
				}
				label := fmt.Sprintf("%s base %s #%d", z.zone, base.Format(time.DateOnly), i)
				if got, want := BuildCards(state, loc), oldBuildCards(state, loc); !reflect.DeepEqual(got, want) {
					t.Fatalf("%s: cards differ\ngot  %+v\nwant %+v", label, got, want)
				}
				extra := Date(base.AddDate(0, 0, rng.Intn(70)), loc)
				if got, want := History(state, loc, extra), oldHistory(state, loc, extra); !reflect.DeepEqual(got, want) {
					t.Fatalf("%s: history differs", label)
				}
				for range 4 {
					now := base.Add(time.Duration(rng.Int63n(int64(70 * 24 * time.Hour))))
					if rng.Intn(2) == 0 {
						now = z.change.Add(time.Duration(rng.Intn(72*60)-36*60) * time.Minute)
					}
					date := Date(now, loc)
					// Plan today from the current picks, as the store does.
					if rng.Intn(2) == 0 {
						replay := ReplayAttempts(state, loc)
						plan := PlanReviews(replay.Cards(), date, loc, state.GoalFor(date).Review, state.Plans[date])
						if want := PlanReviews(oldBuildCards(state, loc), date, loc, state.GoalFor(date).Review, state.Plans[date]); !slices.Equal(plan, want) {
							t.Fatalf("%s: plan %v, want %v", label, plan, want)
						}
						if len(plan) > 0 {
							state.Plans[date] = plan
						}
						assertSameToday(t, label, NewTodayFrom(settings, state, now, replay), oldNewToday(settings, state, now))
					}
					assertSameToday(t, label, NewToday(settings, state, now), oldNewToday(settings, state, now))
				}
			}
		}
	}
	// Casey's clock change must have exercised the unordered path.
	if unordered == 0 {
		t.Fatal("no history had local dates going back")
	}
}

// A review pick's reason uses the card as it stood before today. When local
// dates go back, that is not a prefix of the replay, and the attempts before
// today are replayed on their own.
func TestTodayBeforeCardWhenLocalDatesGoBack(t *testing.T) {
	settings := Settings{Timezone: "Antarctica/Casey"}
	state := State{Problems: testProblems, Attempts: []Attempt{
		// 23:30 on 03-04 at +08, after 00:30 on 03-05 at +11.
		attempt("two-sum", "solved", 10, false, time.Date(2010, 3, 4, 15, 30, 0, 0, time.UTC)),
		attempt("two-sum", "unfinished", 30, false, time.Date(2010, 3, 4, 13, 30, 0, 0, time.UTC)),
		attempt("two-sum", "solved", 10, false, time.Date(2010, 3, 1, 2, 0, 0, 0, time.UTC)),
	}, Plans: map[time.Time][]string{day("2010-03-05"): {"two-sum"}}}
	now := time.Date(2010, 3, 5, 4, 0, 0, 0, time.UTC)
	got, want := NewToday(settings, state, now), oldNewToday(settings, state, now)
	if got.Reviews[0].Card.Reviews != 2 || got.Reviews[0].Card.Last.ID != state.Attempts[0].ID {
		t.Fatalf("before card: %+v", got.Reviews[0].Card)
	}
	assertSameToday(t, "Casey", got, want)
}

// A replay made in another zone, or no replay, is not trusted.
func TestNewTodayFromReplaysForOtherZone(t *testing.T) {
	settings := Settings{Timezone: "America/Chicago"}
	loc := settings.Location()
	state := randomState(rand.New(rand.NewSource(7)), loc, time.Date(2026, 10, 2, 0, 0, 0, 0, loc), time.Date(2026, 11, 1, 7, 0, 0, 0, time.UTC))
	now := time.Date(2026, 11, 1, 3, 30, 0, 0, time.UTC)
	want := oldNewToday(settings, state, now)
	assertSameToday(t, "UTC replay", NewTodayFrom(settings, state, now, ReplayAttempts(state, time.UTC)), want)
	assertSameToday(t, "no replay", NewTodayFrom(settings, state, now, Replay{}), want)
}

func assertSameToday(t *testing.T, label string, got, want Today) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s at %s: today differs\ngot  %+v\nwant %+v", label, want.Now, got.Reviews, want.Reviews)
	}
	// Spot-check the derived views the pages use.
	if !reflect.DeepEqual(got.DueOptional(), want.DueOptional()) || !reflect.DeepEqual(got.Calendar(14), want.Calendar(14)) || got.Streaks != want.Streaks {
		t.Fatalf("%s: derived views differ", label)
	}
	for _, a := range want.State.Attempts {
		if got.KindOf(a) != want.KindOf(a) {
			t.Fatalf("%s: kind of %s differs", label, a.ID)
		}
	}
}

// benchmarkState is a year of daily practice over 150 problems.
func benchmarkState() (Settings, State, time.Time) {
	settings := Settings{Timezone: "America/Chicago"}
	loc := settings.Location()
	rng := rand.New(rand.NewSource(1))
	start := time.Date(2025, 10, 1, 8, 0, 0, 0, loc)
	state := State{Problems: map[string]Problem{}, Analyses: map[string]Analysis{}, Plans: map[time.Time][]string{}, Goals: map[time.Time]DailyGoal{}}
	outcomes := []string{"solved", "solved", "struggled", "unfinished"}
	id := 0
	for d := range 365 {
		for range 3 {
			slug := fmt.Sprintf("problem-%03d", rng.Intn(150))
			at := start.AddDate(0, 0, d).Add(time.Duration(rng.Intn(14*60)) * time.Minute)
			state.Attempts = append(state.Attempts, Attempt{ID: fmt.Sprint(id), ProblemSlug: slug, Outcome: outcomes[rng.Intn(len(outcomes))], Minutes: 10 + rng.Intn(40), CreatedAt: at})
			id++
		}
		date := Date(start.AddDate(0, 0, d), loc)
		state.Goals[date] = DailyGoal{New: 2, Review: 2}
		state.Plans[date] = []string{fmt.Sprintf("problem-%03d", rng.Intn(150)), fmt.Sprintf("problem-%03d", rng.Intn(150))}
	}
	slices.SortFunc(state.Attempts, func(a, b Attempt) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return settings, state, start.AddDate(0, 0, 364).Add(10 * time.Hour)
}

// The benchmarks model LeetgrinderToday: plan reviews from the cards, then
// build today's view.
func BenchmarkTodayPerCallReplays(b *testing.B) {
	settings, state, now := benchmarkState()
	loc := settings.Location()
	date := Date(now, loc)
	b.ResetTimer()
	for range b.N {
		PlanReviews(oldBuildCards(state, loc), date, loc, state.GoalFor(date).Review, nil)
		oldNewToday(settings, state, now)
	}
}

func BenchmarkTodaySingleReplay(b *testing.B) {
	settings, state, now := benchmarkState()
	loc := settings.Location()
	date := Date(now, loc)
	b.ResetTimer()
	for range b.N {
		replay := ReplayAttempts(state, loc)
		PlanReviews(replay.Cards(), date, loc, state.GoalFor(date).Review, nil)
		NewTodayFrom(settings, state, now, replay)
	}
}
