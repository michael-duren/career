package leetgrinder

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestScheduleBoundaries(t *testing.T) {
	chicago, _ := time.LoadLocation("America/Chicago")
	s := NewSchedule(day("2026-10-01"), chicago)
	if got := s.End().Format(time.DateOnly); got != "2026-12-23" {
		t.Fatalf("end = %s", got)
	}
	at := func(local string) time.Time {
		v, err := time.ParseInLocation("2006-01-02 15:04", local, chicago)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	for _, test := range []struct {
		now       string
		expected  int
		session   int
		completed int
		status    string
	}{
		{"2026-09-29 12:00", 0, 0, 0, "Starts in 2 days"},
		{"2026-09-30 23:59", 0, 0, 0, "Starts tomorrow"},
		{"2026-10-01 00:00", 1, 1, 0, "1 session behind"},
		{"2026-10-01 00:00", 1, 1, 1, "On track"},
		{"2026-10-03 08:00", 3, 3, 5, "2 sessions ahead"},
		{"2026-10-10 08:00", 10, 10, 6, "4 sessions behind"},
		{"2026-12-23 23:59", 84, 84, 84, "On track"},
		{"2026-12-24 00:00", 84, 0, 80, "4 sessions behind"},
		{"2027-03-01 00:00", 84, 0, 84, "On track"},
	} {
		now := at(test.now)
		if got := s.ExpectedSessions(now); got != test.expected {
			t.Errorf("%s: expected %d, want %d", test.now, got, test.expected)
		}
		if got := s.SessionForDate(Date(now, chicago)); got != test.session {
			t.Errorf("%s: session %d, want %d", test.now, got, test.session)
		}
		if got := s.Status(test.completed, now); got != test.status {
			t.Errorf("%s: status %q, want %q", test.now, got, test.status)
		}
		if got := s.Delta(test.completed, now); got != test.completed-test.expected {
			t.Errorf("%s: delta %d", test.now, got)
		}
	}
}

func TestScheduleUsesLocalDateAcrossTimezonesAndDST(t *testing.T) {
	chicago, _ := time.LoadLocation("America/Chicago")
	tokyo, _ := time.LoadLocation("Asia/Tokyo")
	// 03:30 UTC on 2 Oct is still 1 Oct in Chicago but already 2 Oct in Tokyo.
	now := time.Date(2026, 10, 2, 3, 30, 0, 0, time.UTC)
	if got := NewSchedule(day("2026-10-01"), chicago).ExpectedSessions(now); got != 1 {
		t.Fatalf("chicago expected %d", got)
	}
	if got := NewSchedule(day("2026-10-01"), tokyo).ExpectedSessions(now); got != 2 {
		t.Fatalf("tokyo expected %d", got)
	}
	// US DST ends 1 Nov 2026; the 25-hour day must not shift sessions.
	s := NewSchedule(day("2026-10-31"), chicago)
	late := time.Date(2026, 11, 2, 23, 30, 0, 0, chicago)
	if got := s.ExpectedSessions(late); got != 3 {
		t.Fatalf("across DST expected %d", got)
	}
	if got := EndOfDate(day("2026-11-01"), chicago); !got.Equal(time.Date(2026, 11, 2, 6, 0, 0, 0, time.UTC)) {
		t.Fatalf("end of DST day = %s", got.UTC())
	}
}

func TestEditingEitherScheduleDate(t *testing.T) {
	loc := time.UTC
	fromStart := NewSchedule(day("2026-01-01"), loc)
	fromEnd := ScheduleEndingOn(fromStart.End(), loc)
	if !fromEnd.Start.Equal(fromStart.Start) {
		t.Fatalf("end round trip: %s", fromEnd.Start)
	}
	// Leap years change nothing: the schedule is always 84 consecutive days.
	leap := ScheduleEndingOn(day("2028-03-15"), loc)
	if got := leap.Start.Format(time.DateOnly); got != "2027-12-23" {
		t.Fatalf("leap start %s", got)
	}
	if DaysBetween(leap.Start, leap.End()) != SessionCount-1 {
		t.Fatal("schedule is not 84 days")
	}
}
