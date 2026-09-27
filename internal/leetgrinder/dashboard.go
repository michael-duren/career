package leetgrinder

import (
	"strings"
	"time"
)

// UpcomingSession is a session after the current one, with its scheduled
// date when a schedule is set.
type UpcomingSession struct {
	Day  Day
	Date time.Time
}

func (u UpcomingSession) Scheduled() bool { return !u.Date.IsZero() }

// Overdue reports that the session's scheduled date is before today.
func (u UpcomingSession) Overdue(today time.Time) bool { return u.Scheduled() && u.Date.Before(today) }

// Upcoming lists up to n unfinished sessions after the next unfinished one.
func (p OverviewPage) Upcoming(n int) []UpcomingSession {
	next := Summarize(p.Today.State).NextDay
	if next == 0 {
		return nil
	}
	schedule, scheduled := p.Today.Schedule()
	var result []UpcomingSession
	for number := next + 1; number <= SessionCount && len(result) < n; number++ {
		if p.Today.State.DayCompleted(number) {
			continue
		}
		day, _ := FindDay(number)
		u := UpcomingSession{Day: day}
		if scheduled {
			u.Date = schedule.Start.AddDate(0, 0, number-1)
		}
		result = append(result, u)
	}
	return result
}

// NextSession is the earliest unfinished session; ok is false once all are done.
func (p OverviewPage) NextSession() (Day, bool) {
	return FindDay(Summarize(p.Today.State).NextDay)
}

// Started reports whether the learner has logged anything yet.
func (p OverviewPage) Started() bool {
	return len(p.Today.State.Attempts) > 0 || len(p.Today.State.CompletedDays) > 0
}

// Recent returns the newest n attempts; State keeps attempts newest first.
func (p OverviewPage) Recent(n int) []Attempt {
	if len(p.Today.State.Attempts) < n {
		return p.Today.State.Attempts
	}
	return p.Today.State.Attempts[:n]
}

// Streak counts consecutive local days with at least one attempt, ending
// today, or yesterday when nothing is logged yet today.
func (p OverviewPage) Streak() int {
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
func (p OverviewPage) ReviewsLeft() int {
	return len(p.Today.Missing().Reviews)
}

// IndependentRate is the share of solved problems solved without help.
func (p OverviewPage) IndependentRate() string {
	progress := Summarize(p.Today.State)
	if progress.Solved == 0 {
		return "—"
	}
	return Percent(float64(progress.Independent) / float64(progress.Solved))
}

// RequiredReading totals the minutes of non-optional reading for a session.
func (d Day) RequiredReading() int {
	n := 0
	for _, r := range d.Readings {
		if !r.Optional {
			n += r.Minutes
		}
	}
	return n
}

// DifficultyClass maps a LeetCode difficulty to its badge class.
func DifficultyClass(difficulty string) string { return "diff diff-" + strings.ToLower(difficulty) }
