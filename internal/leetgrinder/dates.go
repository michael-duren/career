package leetgrinder

import (
	"time"
	_ "time/tzdata" // the runtime image has no zoneinfo
)

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
