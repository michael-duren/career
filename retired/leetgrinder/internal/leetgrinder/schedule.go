package leetgrinder

import (
	"fmt"
	"time"
	_ "time/tzdata" // the runtime image has no zoneinfo
)

const SessionCount = 84

// Date returns the calendar date of t in loc as midnight UTC. Every date in
// this package uses that form, so day arithmetic never crosses a DST change.
func Date(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// DaysBetween counts calendar days from one date to another.
func DaysBetween(from, to time.Time) int {
	return int(Date(to, time.UTC).Sub(Date(from, time.UTC)).Hours() / 24)
}

// EndOfDate is the first instant after date in loc.
func EndOfDate(date time.Time, loc *time.Location) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day()+1, 0, 0, 0, 0, loc)
}

// Schedule maps the 84 sessions onto consecutive calendar days.
type Schedule struct {
	Start time.Time
	Loc   *time.Location
}

func NewSchedule(start time.Time, loc *time.Location) Schedule {
	return Schedule{Start: Date(start, time.UTC), Loc: loc}
}

// ScheduleEndingOn is the schedule whose last session falls on end.
func ScheduleEndingOn(end time.Time, loc *time.Location) Schedule {
	return NewSchedule(Date(end, time.UTC).AddDate(0, 0, -(SessionCount-1)), loc)
}

func (s Schedule) End() time.Time                { return s.Start.AddDate(0, 0, SessionCount-1) }
func (s Schedule) Today(now time.Time) time.Time { return Date(now, s.Loc) }

// SessionForDate is the session scheduled on date, or 0 outside the schedule.
func (s Schedule) SessionForDate(date time.Time) int {
	n := DaysBetween(s.Start, date) + 1
	if n < 1 || n > SessionCount {
		return 0
	}
	return n
}

// ExpectedSessions is how many sessions should be finished by the end of today.
func (s Schedule) ExpectedSessions(now time.Time) int {
	return max(0, min(SessionCount, DaysBetween(s.Start, s.Today(now))+1))
}

func (s Schedule) Delta(completed int, now time.Time) int {
	return completed - s.ExpectedSessions(now)
}

func (s Schedule) Status(completed int, now time.Time) string {
	if days := DaysBetween(s.Today(now), s.Start); days > 0 {
		if days == 1 {
			return "Starts tomorrow"
		}
		return fmt.Sprintf("Starts in %d days", days)
	}
	switch delta := s.Delta(completed, now); {
	case delta > 0:
		return sessionCount(delta) + " ahead"
	case delta < 0:
		return sessionCount(-delta) + " behind"
	default:
		return "On track"
	}
}

func sessionCount(n int) string {
	if n == 1 {
		return "1 session"
	}
	return fmt.Sprintf("%d sessions", n)
}
