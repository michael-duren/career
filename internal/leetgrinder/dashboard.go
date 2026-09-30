package leetgrinder

import (
	"strings"
	"time"
)

// Started reports whether the learner has logged anything yet.
func (p OverviewPage) Started() bool { return len(p.Today.State.Attempts) > 0 }

// Recent returns the newest n attempts; State keeps attempts newest first.
func (p OverviewPage) Recent(n int) []Attempt {
	if len(p.Today.State.Attempts) < n {
		return p.Today.State.Attempts
	}
	return p.Today.State.Attempts[:n]
}

// ActiveStreak counts consecutive local days with at least one attempt,
// ending today, or yesterday when nothing is logged yet today.
func (p OverviewPage) ActiveStreak() int {
	loc := p.Today.Settings.Location()
	days := map[time.Time]bool{}
	for _, a := range p.Today.State.Attempts {
		days[Date(a.CreatedAt, loc)] = true
	}
	day := p.Today.Date
	if !days[day] {
		day = day.AddDate(0, 0, -1)
	}
	n := 0
	for days[day] {
		n++
		day = day.AddDate(0, 0, -1)
	}
	return n
}

// ReviewsLeft counts today's planned reviews that have no attempt yet.
func (p OverviewPage) ReviewsLeft() int { return len(p.Today.MissingReviews()) }

// DueSoon is the start of the optional due list, at most n cards.
func (p OverviewPage) DueSoon(n int) []Card {
	if len(p.Due) < n {
		return p.Due
	}
	return p.Due[:n]
}

// DayUnit is the unit label for a count of days.
func DayUnit(n int) string {
	if n == 1 {
		return "day"
	}
	return "days"
}

// IndependentRate is the share of solved problems solved without help.
func (p OverviewPage) IndependentRate() string {
	progress := Summarize(p.Today.State)
	if progress.Solved == 0 {
		return "—"
	}
	return Percent(float64(progress.Independent) / float64(progress.Solved))
}

// DifficultyClass maps a LeetCode difficulty to its badge class.
func DifficultyClass(difficulty string) string { return "diff diff-" + strings.ToLower(difficulty) }
