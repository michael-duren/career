package leetgrinder

import (
	"errors"
	"sort"
	"time"
)

// DailyGoal is a day's targets: new problems and goal-counting reviews.
type DailyGoal struct {
	New    int `json:"new"`
	Review int `json:"review"`
}

// DefaultGoal applies to days with no frozen goal, including every day
// before daily goals existed.
var DefaultGoal = DailyGoal{New: 2, Review: 1}

func (g DailyGoal) Validate() error {
	if g.New < 0 || g.New > MaxGoal || g.Review < 0 || g.Review > MaxGoal {
		return errors.New("daily targets must be 0 to 10")
	}
	if g.New == 0 && g.Review == 0 {
		return errors.New("set at least one daily target above 0")
	}
	return nil
}

// Attempt kinds, as a day counts them.
const (
	// KindNew is the first local day with an attempt on the problem.
	KindNew = "new"
	// KindReview is a later day on a problem that was due by the end of the
	// day, flagged, or that day's review pick. It counts toward the goal.
	KindReview = "review"
	// KindPractice is a later day on a problem that was not due.
	KindPractice = "practice"
)

// DayProgress is what one local day's attempts did for its goal. Several
// attempts on one problem in a day count once.
type DayProgress struct {
	Date time.Time
	Goal DailyGoal
	// Kinds maps each problem attempted that day to its kind.
	Kinds map[string]string
}

func (d DayProgress) count(kind string) int {
	n := 0
	for _, k := range d.Kinds {
		if k == kind {
			n++
		}
	}
	return n
}

func (d DayProgress) New() int      { return d.count(KindNew) }
func (d DayProgress) Reviews() int  { return d.count(KindReview) }
func (d DayProgress) Practice() int { return d.count(KindPractice) }

// Met reports whether the day reached both targets.
func (d DayProgress) Met() bool { return d.New() >= d.Goal.New && d.Reviews() >= d.Goal.Review }

// NewLeft and ReviewsLeft are what the goal still needs.
func (d DayProgress) NewLeft() int     { return max(0, d.Goal.New-d.New()) }
func (d DayProgress) ReviewsLeft() int { return max(0, d.Goal.Review-d.Reviews()) }

// Remaining is the work left to meet the goal.
func (d DayProgress) Remaining() int { return d.NewLeft() + d.ReviewsLeft() }

// Bonus counts work beyond the goal: extra new problems, extra reviews,
// and practice.
func (d DayProgress) Bonus() int {
	return max(0, d.New()-d.Goal.New) + max(0, d.Reviews()-d.Goal.Review) + d.Practice()
}

// History classifies every attempt by local day in loc, using the frozen
// goals and review plans in state. The result has an entry for every day
// with an attempt.
func History(state State, loc *time.Location) map[time.Time]*DayProgress {
	days := map[time.Time]*DayProgress{}
	for slug, list := range attemptsBySlug(state.Attempts) {
		problem := state.Problem(slug)
		replaySlug(problem, list, loc, func(date time.Time, before Card) {
			day := days[date]
			if day == nil {
				day = &DayProgress{Date: date, Goal: state.GoalFor(date), Kinds: map[string]string{}}
				days[date] = day
			}
			day.Kinds[slug] = classify(state, slug, problem, date, before, loc)
		})
	}
	return days
}

// classify decides the kind of a problem's attempts on date, given its card
// as it stood before that day.
func classify(state State, slug string, problem Problem, date time.Time, before Card, loc *time.Location) string {
	if before.Reviews == 0 {
		return KindNew
	}
	if before.FSRSDue.Before(EndOfDate(date, loc)) {
		return KindReview
	}
	if f := flagFor(before.Last, problem, state.Analyses, loc); f != nil && f.Date.Before(date) {
		return KindReview
	}
	for _, pick := range state.Plans[date] {
		if pick == slug {
			return KindReview
		}
	}
	return KindPractice
}

// Streaks are counts of consecutive goal-met days.
type Streaks struct {
	// Current ends today, or yesterday while today's goal is not met yet.
	Current int
	Longest int
	// Active counts consecutive days with any attempt, the same way.
	Active int
}

// ComputeStreaks walks history up to today. today is today's progress,
// whose goal is today's frozen goal.
func ComputeStreaks(history map[time.Time]*DayProgress, today DayProgress) Streaks {
	met := func(date time.Time) bool {
		if date.Equal(today.Date) {
			return today.Met()
		}
		d, ok := history[date]
		return ok && d.Met()
	}
	active := func(date time.Time) bool {
		d, ok := history[date]
		return ok && len(d.Kinds) > 0
	}
	run := func(ok func(time.Time) bool) int {
		day := today.Date
		if !ok(day) {
			day = day.AddDate(0, 0, -1)
		}
		n := 0
		for ok(day) {
			n++
			day = day.AddDate(0, 0, -1)
		}
		return n
	}
	s := Streaks{Current: run(met), Active: run(active)}
	dates := make([]time.Time, 0, len(history))
	for d := range history {
		if !d.After(today.Date) {
			dates = append(dates, d)
		}
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	length := 0
	var prev time.Time
	for _, d := range dates {
		if !met(d) {
			length = 0
			continue
		}
		if length > 0 && DaysBetween(prev, d) == 1 {
			length++
		} else {
			length = 1
		}
		prev = d
		s.Longest = max(s.Longest, length)
	}
	s.Longest = max(s.Longest, s.Current)
	return s
}

// GoalFor is date's frozen goal, or DefaultGoal when none was frozen.
func (s State) GoalFor(date time.Time) DailyGoal {
	if g, ok := s.Goals[date]; ok {
		return g
	}
	return DefaultGoal
}

// GoalMetDays lists the last n local dates ending today with whether each
// met its goal, oldest first.
func GoalMetDays(history map[time.Time]*DayProgress, today DayProgress, n int) []CalendarDay {
	out := make([]CalendarDay, 0, n)
	for i := n - 1; i >= 0; i-- {
		date := today.Date.AddDate(0, 0, -i)
		day := CalendarDay{Date: date}
		if date.Equal(today.Date) {
			day.Met, day.Active = today.Met(), len(today.Kinds) > 0
		} else if d, ok := history[date]; ok {
			day.Met, day.Active = d.Met(), len(d.Kinds) > 0
		}
		out = append(out, day)
	}
	return out
}

// CalendarDay is one cell of the goal calendar.
type CalendarDay struct {
	Date        time.Time
	Met, Active bool
}
