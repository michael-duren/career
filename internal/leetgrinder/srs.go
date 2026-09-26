package leetgrinder

import (
	"cmp"
	"fmt"
	"slices"
	"sync"
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

type problemPlace struct {
	Problem Problem
	Day     int
}

// problemIndex maps each curriculum slug to its first assigning session.
var problemIndex = sync.OnceValue(func() map[string]problemPlace {
	index := map[string]problemPlace{}
	for _, week := range Curriculum() {
		for _, day := range week.Days {
			for _, p := range slices.Concat(day.Core, day.Optional) {
				if _, ok := index[p.Slug]; !ok {
					index[p.Slug] = problemPlace{p, day.Number}
				}
			}
		}
	}
	return index
})

// ProblemDay is the first session that assigns slug, or 0 outside the curriculum.
func ProblemDay(slug string) int { return problemIndex()[slug].Day }

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
	Day     int
	// Last is the attempt that counted most recently.
	Last    Attempt
	Reviews int
	Due     time.Time
	fsrs    fsrs.Card
}

func (c Card) Week() int { return WeekNumber(c.Day) }

// Retrievability is the estimated recall probability at t.
func (c Card) Retrievability(t time.Time) float64 {
	r, err := scheduler.Retrievability(c.fsrs, t)
	if err != nil {
		return 0
	}
	return r
}

// BuildCards replays attempts on curriculum problems through FSRS. Only the
// last attempt on each local day counts. Cards come back ordered by due time.
func BuildCards(attempts []Attempt, loc *time.Location) []Card {
	bySlug := map[string][]Attempt{}
	for _, a := range attempts {
		if _, ok := problemIndex()[a.ProblemSlug]; ok {
			bySlug[a.ProblemSlug] = append(bySlug[a.ProblemSlug], a)
		}
	}
	cards := make([]Card, 0, len(bySlug))
	for slug, list := range bySlug {
		slices.SortFunc(list, func(a, b Attempt) int {
			return cmp.Or(a.CreatedAt.Compare(b.CreatedAt), cmp.Compare(a.ID, b.ID))
		})
		place := problemIndex()[slug]
		card := Card{Problem: place.Problem, Day: place.Day}
		for i, a := range list {
			if i+1 < len(list) && Date(list[i+1].CreatedAt, loc).Equal(Date(a.CreatedAt, loc)) {
				continue
			}
			state := card.fsrs
			if card.Reviews == 0 {
				state = fsrs.NewCard(a.CreatedAt)
			}
			info, err := scheduler.Next(state, a.CreatedAt, ReviewRating(a))
			if err != nil {
				continue
			}
			card.fsrs, card.Last, card.Due = info.Card, a, info.Card.Due
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

// ReviewSlots is how many reviews date gets. Sessions with required reading
// spend the base review time reading; extra daily hours add slots.
func ReviewSlots(session int, hours float64) int {
	base := 1
	if day, ok := FindDay(session); ok {
		for _, r := range day.Readings {
			if !r.Optional {
				base = 0
				break
			}
		}
	}
	return base + ExtraReviewSlots(hours)
}

// PlanReviews extends a frozen plan for date. Existing picks are kept in
// order; new picks fill any remaining slots, lowest recall first, preferring
// earlier curriculum weeks, then the most overdue.
func PlanReviews(cards []Card, date time.Time, loc *time.Location, session int, hours float64, existing []string) []string {
	plan := slices.Clone(existing)
	slots := ReviewSlots(session, hours)
	if len(plan) >= slots {
		return plan
	}
	skip := map[string]bool{}
	for _, slug := range plan {
		skip[slug] = true
	}
	if day, ok := FindDay(session); ok {
		for _, p := range slices.Concat(day.Core, day.Optional) {
			skip[p.Slug] = true
		}
	}
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
		return cmp.Or(cmp.Compare(a.r, b.r), cmp.Compare(a.card.Week(), b.card.Week()), a.card.Due.Compare(b.card.Due), cmp.Compare(a.card.Problem.Slug, b.card.Problem.Slug))
	})
	for _, c := range candidates {
		if len(plan) == slots {
			break
		}
		plan = append(plan, c.card.Problem.Slug)
	}
	return plan
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
