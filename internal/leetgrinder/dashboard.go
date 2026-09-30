package leetgrinder

import "strings"

// Started reports whether the learner has logged anything yet.
func (p OverviewPage) Started() bool { return len(p.Today.State.Attempts) > 0 }

// Recent returns the newest n attempts; State keeps attempts newest first.
func (p OverviewPage) Recent(n int) []Attempt {
	if len(p.Today.State.Attempts) < n {
		return p.Today.State.Attempts
	}
	return p.Today.State.Attempts[:n]
}

// DueSoon is the start of today's optional due list, at most n cards, and
// the list's full length.
func DueSoon(today Today, n int) ([]Card, int) {
	due := today.DueOptional()
	return due[:min(n, len(due))], len(due)
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

// KindLabel names an attempt kind for display.
func KindLabel(kind string) string {
	switch kind {
	case KindNew:
		return "New"
	case KindReview:
		return "Review"
	case KindPractice:
		return "Practice"
	}
	return ""
}
