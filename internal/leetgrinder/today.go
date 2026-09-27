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
	// Session is today's scheduled session, or 0 when unset or out of range.
	Session int
	Reviews []ReviewItem
}

// NewToday combines saved state with the frozen review plan for now's local date.
func NewToday(settings Settings, state State, plan []string, now time.Time) Today {
	loc := settings.Location()
	t := Today{Settings: settings, State: state, Now: now, Date: Date(now, loc)}
	if schedule, ok := settings.Schedule(); ok {
		t.Session = schedule.SessionForDate(t.Date)
	}
	// Reasons describe each card as it stood before today, so they stay
	// stable after today's review is logged.
	var before []Attempt
	for _, a := range state.Attempts {
		if Date(a.CreatedAt, loc).Before(t.Date) {
			before = append(before, a)
		}
	}
	cards := map[string]Card{}
	for _, c := range BuildCards(before, loc) {
		cards[c.Problem.Slug] = c
	}
	for i, slug := range plan {
		item := ReviewItem{Slot: i + 1, Card: cards[slug], Done: t.AttemptedOn(slug)}
		item.Problem, _ = FindProblem(slug)
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

func (t Today) Schedule() (Schedule, bool) { return t.Settings.Schedule() }

// Status is the on/off-track label, or "" when no schedule is set.
func (t Today) Status() string {
	s, ok := t.Schedule()
	if !ok {
		return ""
	}
	return s.Status(Summarize(t.State).Completed, t.Now)
}

// MissingWork lists what today still needs: the scheduled session if it is
// unfinished, and any planned review without an attempt today.
type MissingWork struct {
	Session int
	Reviews []ReviewItem
}

func (m MissingWork) Empty() bool { return m.Session == 0 && len(m.Reviews) == 0 }

func (t Today) Missing() MissingWork {
	var m MissingWork
	if t.Session > 0 && !t.State.DayCompleted(t.Session) {
		m.Session = t.Session
	}
	for _, r := range t.Reviews {
		if !r.Done {
			m.Reviews = append(m.Reviews, r)
		}
	}
	return m
}

// Backlog counts cards due by the end of today that today's plan left out.
func (t Today) Backlog(cards []Card) int {
	planned := map[string]bool{}
	for _, r := range t.Reviews {
		planned[r.Problem.Slug] = true
	}
	end := EndOfDate(t.Date, t.Settings.Location())
	n := 0
	for _, c := range cards {
		if !planned[c.Problem.Slug] && c.Due.Before(end) {
			n++
		}
	}
	return n
}
