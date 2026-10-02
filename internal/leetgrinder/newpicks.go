package leetgrinder

import "time"

// NewPick is one frozen new-problem pick, drawn from a todo entry.
type NewPick struct {
	Slug string
	// SetID and SetTitle name the todo set the pick came from; both are
	// empty for an individual todo or a set deleted since.
	SetID, SetTitle string
}

// NewPickItem is one of today's new-problem picks.
type NewPickItem struct {
	Slot     int
	Problem  Problem
	SetTitle string
	// Done reports an attempt on the problem today.
	Done bool
}

// newPickItems lists date's frozen new picks when picking from todos is on.
// Turning it off hides picks already frozen; turning it back on shows them.
func (t Today) newPickItems() []NewPickItem {
	if !t.Settings.NewFromTodos {
		return nil
	}
	var items []NewPickItem
	for i, pick := range t.State.NewPlans[t.Date] {
		items = append(items, NewPickItem{Slot: i + 1, Problem: t.State.Problem(pick.Slug), SetTitle: pick.SetTitle, Done: t.AttemptedOn(pick.Slug)})
	}
	return items
}

// NextNewPick is the first of today's new picks without an attempt today.
func (t Today) NextNewPick() (NewPickItem, bool) {
	for _, p := range t.NewPicks {
		if !p.Done {
			return p, true
		}
	}
	return NewPickItem{}, false
}

// NewPicksShort reports that picking from todos is on and today has fewer
// picks than its new target: too few todos could be picked, or planning
// failed (see NewPicksFailed).
func (t Today) NewPicksShort() bool {
	return t.Settings.NewFromTodos && len(t.NewPicks) < t.Goal.New
}

// NewPicksWanted is how many more new picks date's frozen goal has room
// for, or 0 when picking from todos is off.
func NewPicksWanted(settings Settings, state State, date time.Time) int {
	if !settings.NewFromTodos {
		return 0
	}
	return max(0, state.GoalFor(date).New-len(state.NewPlans[date]))
}
