package leetgrinder

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v4"
)

// Default FSRS parameters, minus the sub-day learning steps: only one attempt
// per local day counts, so minute-scale steps would only mean "due tomorrow".
var scheduler = func() *fsrs.FSRS {
	p := fsrs.DefaultParam()
	p.EnableShortTerm = false
	return fsrs.NewFSRS(p)
}()

// ReviewRating maps an attempt onto an FSRS rating.
func ReviewRating(a Attempt) fsrs.Rating {
	switch {
	case a.Outcome == "unfinished":
		return fsrs.Again
	case a.Outcome == "struggled" || a.Assisted:
		return fsrs.Hard
	case a.Minutes > ReviewSlotMinutes:
		return fsrs.Good
	default:
		return fsrs.Easy
	}
}

// Card is the spaced-repetition state of one attempted problem. It is derived
// by replaying attempts, never stored, so corrections flow through.
type Card struct {
	Problem Problem
	// Last is the attempt that counted most recently: the newest attempt.
	Last    Attempt
	Reviews int
	// Due is the effective due time: the FSRS due time, or the flag's due
	// time when that is earlier.
	Due time.Time
	// FSRSDue is when FSRS alone schedules the next review.
	FSRSDue time.Time
	// Flag is set when the analysis of Last found a mistake.
	Flag *Flag
	fsrs fsrs.Card
	loc  *time.Location
}

// Flag marks a problem whose latest attempt's analysis judged a stated
// complexity wrong or the solution not optimal. It makes the problem due the
// day after the analysis finished, and clears with the next attempt or a
// re-analysis without those verdicts.
type Flag struct {
	// Date is the local date the analysis finished.
	Date                  time.Time
	TimeWrong, SpaceWrong bool
	NotOptimal            bool
	// Actual and optimal complexities, for the not-optimal reason.
	ActualTime, ActualSpace   string
	OptimalTime, OptimalSpace string
}

// Flagged reports whether the card carries a complexity flag.
func (c Card) Flagged() bool { return c.Flag != nil }

// go-fsrs counts elapsed days between UTC calendar dates. Feeding it local
// wall-clock times labelled as UTC makes those days the learner's local days.
func wallClock(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), l.Hour(), l.Minute(), l.Second(), l.Nanosecond(), time.UTC)
}

func fromWallClock(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}

// Retrievability is the estimated recall probability at t.
func (c Card) Retrievability(t time.Time) float64 {
	loc := c.loc
	if loc == nil {
		loc = time.UTC
	}
	r, err := scheduler.Retrievability(c.fsrs, wallClock(t, loc))
	if err != nil {
		return 0
	}
	return r
}

// flagFor derives the flag of an attempt from its analysis, or nil.
func flagFor(a Attempt, problem Problem, analyses map[string]Analysis, loc *time.Location) *Flag {
	an, ok := analyses[a.ID]
	if !ok || !an.Done() {
		return nil
	}
	f := &Flag{Date: Date(an.UpdatedAt, loc), TimeWrong: isFalse(an.TimeMatches), SpaceWrong: isFalse(an.SpaceMatches), NotOptimal: isFalse(an.Optimal),
		ActualTime: an.ActualTime, ActualSpace: an.ActualSpace, OptimalTime: problem.OptimalTime, OptimalSpace: problem.OptimalSpace}
	if !f.TimeWrong && !f.SpaceWrong && !f.NotOptimal {
		return nil
	}
	return f
}

// DueAt is the start of the local day after the analysis.
func (f Flag) DueAt(loc *time.Location) time.Time {
	return time.Date(f.Date.Year(), f.Date.Month(), f.Date.Day()+1, 0, 0, 0, 0, loc)
}

// Reason explains the flag, e.g. "Time complexity judged wrong 2 days ago"
// or "Not optimal: O(n²) vs O(n)".
func (f Flag) Reason(now time.Time, loc *time.Location) string {
	when := daysAgo(f.Date, Date(now, loc))
	switch {
	case f.TimeWrong && f.SpaceWrong:
		return "Time and space complexity judged wrong " + when
	case f.TimeWrong:
		return "Time complexity judged wrong " + when
	case f.SpaceWrong:
		return "Space complexity judged wrong " + when
	case f.OptimalTime != "" && f.ActualTime != "" && f.ActualTime != f.OptimalTime:
		return fmt.Sprintf("Not optimal: %s vs %s", f.ActualTime, f.OptimalTime)
	case f.OptimalSpace != "" && f.ActualSpace != "" && f.ActualSpace != f.OptimalSpace:
		return fmt.Sprintf("Not optimal: space %s vs %s", f.ActualSpace, f.OptimalSpace)
	}
	return "Not optimal"
}

// Label is a short form for badges, e.g. "Flagged: time complexity judged wrong".
func (f Flag) Label() string {
	switch {
	case f.TimeWrong && f.SpaceWrong:
		return "Flagged: time and space complexity judged wrong"
	case f.TimeWrong:
		return "Flagged: time complexity judged wrong"
	case f.SpaceWrong:
		return "Flagged: space complexity judged wrong"
	}
	return "Flagged: not optimal"
}

func daysAgo(from, to time.Time) string {
	switch days := DaysBetween(from, to); days {
	case 0:
		return "today"
	case 1:
		return "yesterday"
	default:
		return fmt.Sprintf("%d days ago", days)
	}
}

// replaySlug replays one problem's attempts, sorted oldest first, through
// FSRS. Only the last attempt on each local day counts. visit, when set,
// sees each local day with the card as it stood before that day.
func replaySlug(problem Problem, list []Attempt, loc *time.Location, visit func(date time.Time, before Card)) Card {
	card := Card{Problem: problem, loc: loc}
	for i, a := range list {
		date := Date(a.CreatedAt, loc)
		if i+1 < len(list) && Date(list[i+1].CreatedAt, loc).Equal(date) {
			continue
		}
		if visit != nil {
			visit(date, card)
		}
		state := card.fsrs
		if card.Reviews == 0 {
			state = fsrs.NewCard(wallClock(a.CreatedAt, loc))
		}
		info, err := scheduler.Next(state, wallClock(a.CreatedAt, loc), ReviewRating(a))
		if err != nil {
			continue
		}
		card.fsrs, card.Last = info.Card, a
		card.FSRSDue = fromWallClock(info.Card.Due, loc)
		card.Due = card.FSRSDue
		card.Reviews++
	}
	return card
}

// attemptsBySlug groups attempts by problem, each list oldest first.
func attemptsBySlug(attempts []Attempt) map[string][]Attempt {
	bySlug := map[string][]Attempt{}
	for _, a := range attempts {
		bySlug[a.ProblemSlug] = append(bySlug[a.ProblemSlug], a)
	}
	for _, list := range bySlug {
		slices.SortFunc(list, func(a, b Attempt) int {
			return cmp.Or(a.CreatedAt.Compare(b.CreatedAt), cmp.Compare(a.ID, b.ID))
		})
	}
	return bySlug
}

// BuildCards replays state's attempts through FSRS, one card per attempted
// problem, and applies complexity flags. Cards come back ordered by due time.
func BuildCards(state State, loc *time.Location) []Card {
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

// DueCards lists cards due by the end of date, flagged first, then lowest
// estimated recall, then the most overdue. Cards in skip are left out.
func DueCards(cards []Card, date time.Time, loc *time.Location, skip map[string]bool) []Card {
	end := EndOfDate(date, loc)
	type candidate struct {
		card Card
		r    float64
	}
	var candidates []candidate
	for _, c := range cards {
		if !skip[c.Problem.Slug] && c.Due.Before(end) {
			candidates = append(candidates, candidate{c, c.Retrievability(end)})
		}
	}
	flagRank := func(c Card) int {
		if c.Flagged() {
			return 0
		}
		return 1
	}
	slices.SortFunc(candidates, func(a, b candidate) int {
		return cmp.Or(cmp.Compare(flagRank(a.card), flagRank(b.card)), cmp.Compare(a.r, b.r), a.card.Due.Compare(b.card.Due), cmp.Compare(a.card.Problem.Slug, b.card.Problem.Slug))
	})
	out := make([]Card, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, c.card)
	}
	return out
}

// PlanReviews extends a frozen plan for date to slots picks. Existing picks
// are kept in order; new picks come from DueCards, leaving out problems first
// attempted on date.
func PlanReviews(cards []Card, date time.Time, loc *time.Location, slots int, existing []string) []string {
	plan := slices.Clone(existing)
	if len(plan) >= slots {
		return plan
	}
	skip := map[string]bool{}
	for _, slug := range plan {
		skip[slug] = true
	}
	for _, c := range cards {
		if c.firstAttemptOn(date) {
			skip[c.Problem.Slug] = true
		}
	}
	for _, c := range DueCards(cards, date, loc, skip) {
		if len(plan) == slots {
			break
		}
		plan = append(plan, c.Problem.Slug)
	}
	return plan
}

// firstAttemptOn reports whether the card's first counted attempt was on date.
func (c Card) firstAttemptOn(date time.Time) bool {
	return c.Reviews == 1 && Date(c.Last.CreatedAt, c.loc).Equal(date)
}

// ReviewReason explains a card in plain text: its flag, or its last counted
// attempt and recall, e.g. "Struggled 9 days ago · recall estimate 62%".
func ReviewReason(c Card, now time.Time, loc *time.Location) string {
	if c.Flag != nil {
		return c.Flag.Reason(now, loc)
	}
	label := OutcomeLabel(c.Last.Outcome)
	if c.Last.Outcome == "solved" && c.Last.Assisted {
		label = "Solved with help"
	}
	return fmt.Sprintf("%s %s · recall estimate %d%%", label, daysAgo(Date(c.Last.CreatedAt, loc), Date(now, loc)), int(c.Retrievability(now)*100+0.5))
}
