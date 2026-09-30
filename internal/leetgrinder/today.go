package leetgrinder

import "time"

// ReviewItem is one planned review for a date.
type ReviewItem struct {
	Slot    int
	Problem Problem
	Card    Card
	Reason  string
	Done    bool
}

// Today is the day-level view shared by pages and notifications.
type Today struct {
	Settings Settings
	State    State
	Now      time.Time
	Date     time.Time
	Reviews  []ReviewItem
}

// NewToday combines saved state with the frozen review plan for now's local date.
func NewToday(settings Settings, state State, plan []string, now time.Time) Today {
	loc := settings.Location()
	t := Today{Settings: settings, State: state, Now: now, Date: Date(now, loc)}
	// Reasons describe each card as it stood before today, so they stay
	// stable after today's review is logged.
	var before []Attempt
	for _, a := range state.Attempts {
		if Date(a.CreatedAt, loc).Before(t.Date) {
			before = append(before, a)
		}
	}
	cards := map[string]Card{}
	for _, c := range BuildCards(before, state.Problems, loc) {
		cards[c.Problem.Slug] = c
	}
	for i, slug := range plan {
		item := ReviewItem{Slot: i + 1, Problem: state.Problem(slug), Card: cards[slug], Done: t.AttemptedOn(slug)}
		if item.Card.Reviews > 0 {
			item.Reason = ReviewReason(item.Card, now, loc)
		}
		t.Reviews = append(t.Reviews, item)
	}
	return t
}

// AttemptedOn reports whether slug has an attempt on today's local date.
func (t Today) AttemptedOn(slug string) bool {
	loc := t.Settings.Location()
	for _, a := range t.State.Attempts {
		if a.ProblemSlug == slug && Date(a.CreatedAt, loc).Equal(t.Date) {
			return true
		}
	}
	return false
}

// Cards replays every attempt into review cards.
func (t Today) Cards() []Card {
	return BuildCards(t.State.Attempts, t.State.Problems, t.Settings.Location())
}

// MissingReviews lists today's planned reviews without an attempt today.
func (t Today) MissingReviews() []ReviewItem {
	var m []ReviewItem
	for _, r := range t.Reviews {
		if !r.Done {
			m = append(m, r)
		}
	}
	return m
}

// DueOptional lists cards due by the end of today that are neither in
// today's plan nor attempted today, in review order.
func (t Today) DueOptional(cards []Card) []Card {
	skip := map[string]bool{}
	for _, r := range t.Reviews {
		skip[r.Problem.Slug] = true
	}
	for _, c := range cards {
		if t.AttemptedOn(c.Problem.Slug) {
			skip[c.Problem.Slug] = true
		}
	}
	return DueCards(cards, t.Date, t.Settings.Location(), skip)
}

// Backlog counts cards due by the end of today that today's plan left out
// and that have no attempt today.
func (t Today) Backlog(cards []Card) int { return len(t.DueOptional(cards)) }
