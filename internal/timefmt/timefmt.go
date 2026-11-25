// Package timefmt is the shared display formatting for templ pages, matching
// src/lib/format.ts (docs/redesign/design-system.md, Formatting). Times are
// 12-hour with no leading zero and a space before AM/PM; ranges use a spaced
// en dash. Output is for on-screen labels only, never for parsing back.
//
// Every function formats the time.Time as given; convert with t.In(loc)
// first when the page shows a user's local time.
package timefmt

import "time"

// Time renders "7:15 AM" or "12:00 PM".
func Time(t time.Time) string { return t.Format("3:04 PM") }

// CompactTime drops ":00" on the hour ("9 AM") for tight places.
func CompactTime(t time.Time) string {
	if t.Minute() == 0 {
		return t.Format("3 PM")
	}
	return Time(t)
}

// Range renders "7:15 AM – 8:45 AM".
func Range(start, end time.Time) string { return Time(start) + " – " + Time(end) }

// CompactRange renders "9 AM – 4 PM".
func CompactRange(start, end time.Time) string {
	return CompactTime(start) + " – " + CompactTime(end)
}

// Date renders "Nov 24".
func Date(t time.Time) string { return t.Format("Jan 2") }

// WeekdayDate renders "Tue 24 Nov".
func WeekdayDate(t time.Time) string { return t.Format("Mon 2 Jan") }

// NumericDate renders "24.11.26"; only for the large Home date.
func NumericDate(t time.Time) string { return t.Format("02.01.06") }
