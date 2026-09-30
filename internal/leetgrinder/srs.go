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
	// Last is the attempt that counted most recently.
	Last    Attempt
	Reviews int
	Due     time.Time
	fsrs    fsrs.Card
	loc     *time.Location
}

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

// BuildCards replays attempts through FSRS, one card per attempted problem,
// taking each card's problem from problems. Only the last attempt on each
// local day counts. Cards come back ordered by due time.
func BuildCards(attempts []Attempt, problems map[string]Problem, loc *time.Location) []Card {
	bySlug := map[string][]Attempt{}
	for _, a := range attempts {
		bySlug[a.ProblemSlug] = append(bySlug[a.ProblemSlug], a)
	}
	cards := make([]Card, 0, len(bySlug))
	for slug, list := range bySlug {
		slices.SortFunc(list, func(a, b Attempt) int {
			return cmp.Or(a.CreatedAt.Compare(b.CreatedAt), cmp.Compare(a.ID, b.ID))
		})
		problem, ok := problems[slug]
		if !ok {
			problem = Problem{Slug: slug}
		}
		card := Card{Problem: problem, loc: loc}
		for i, a := range list {
			if i+1 < len(list) && Date(list[i+1].CreatedAt, loc).Equal(Date(a.CreatedAt, loc)) {
				continue
			}
			state := card.fsrs
			if card.Reviews == 0 {
				state = fsrs.NewCard(wallClock(a.CreatedAt, loc))
			}
			info, err := scheduler.Next(state, wallClock(a.CreatedAt, loc), ReviewRating(a))
			if err != nil {
				continue
			}
			card.fsrs, card.Last, card.Due = info.Card, a, fromWallClock(info.Card.Due, loc)
			card.Reviews++
		}
		if card.Reviews > 0 {
			cards = append(cards, card)
		}
	}
	slices.SortFunc(cards, func(a, b Card) int {
		return cmp.Or(a.Due.Compare(b.Due), cmp.Compare(a.Problem.Slug, b.Problem.Slug))
	})
	return cards
}

// ReviewSlots is how many reviews a day gets: one, plus the extra slots
// from daily time.
func ReviewSlots(hours float64) int { return 1 + ExtraReviewSlots(hours) }

// DueCards lists cards due by the end of date, lowest estimated recall
// first, then the most overdue. Cards in skip are left out.
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
	slices.SortFunc(candidates, func(a, b candidate) int {
		return cmp.Or(cmp.Compare(a.r, b.r), a.card.Due.Compare(b.card.Due), cmp.Compare(a.card.Problem.Slug, b.card.Problem.Slug))
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

// ReviewReason explains a pick in plain text, e.g.
// "Struggled 9 days ago · recall estimate 62%".
func ReviewReason(c Card, now time.Time, loc *time.Location) string {
	label := OutcomeLabel(c.Last.Outcome)
	if c.Last.Outcome == "solved" && c.Last.Assisted {
		label = "Solved with help"
	}
	var when string
	switch days := DaysBetween(Date(c.Last.CreatedAt, loc), Date(now, loc)); days {
	case 0:
		when = "today"
	case 1:
		when = "yesterday"
	default:
		when = fmt.Sprintf("%d days ago", days)
	}
	return fmt.Sprintf("%s %s · recall estimate %d%%", label, when, int(c.Retrievability(now)*100+0.5))
}
