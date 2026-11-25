package timefmt

import (
	"testing"
	"time"
	_ "time/tzdata" // CI images may lack zoneinfo
)

func at(hour, minute int) time.Time { return time.Date(2026, 11, 24, hour, minute, 0, 0, time.UTC) }

func TestTimes(t *testing.T) {
	tests := []struct {
		name          string
		in            time.Time
		full, compact string
	}{
		{"morning", at(7, 15), "7:15 AM", "7:15 AM"},
		{"evening", at(17, 45), "5:45 PM", "5:45 PM"},
		{"midnight", at(0, 0), "12:00 AM", "12 AM"},
		{"after midnight", at(0, 30), "12:30 AM", "12:30 AM"},
		{"noon", at(12, 0), "12:00 PM", "12 PM"},
		{"after noon", at(12, 30), "12:30 PM", "12:30 PM"},
		{"on the hour", at(9, 0), "9:00 AM", "9 AM"},
		{"single-digit minutes", at(9, 5), "9:05 AM", "9:05 AM"},
		{"last minute", at(23, 59), "11:59 PM", "11:59 PM"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Time(tt.in); got != tt.full {
				t.Errorf("Time = %q, want %q", got, tt.full)
			}
			if got := CompactTime(tt.in); got != tt.compact {
				t.Errorf("CompactTime = %q, want %q", got, tt.compact)
			}
		})
	}
}

func TestTimeUsesGivenLocation(t *testing.T) {
	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	if got := Time(at(17, 20).In(chicago)); got != "11:20 AM" {
		t.Errorf("Time = %q, want 11:20 AM", got)
	}
}

func TestRanges(t *testing.T) {
	if got := Range(at(7, 15), at(8, 45)); got != "7:15 AM – 8:45 AM" {
		t.Errorf("Range = %q", got)
	}
	if got := CompactRange(at(9, 0), at(16, 0)); got != "9 AM – 4 PM" {
		t.Errorf("CompactRange = %q", got)
	}
	if got := CompactRange(at(9, 0), at(16, 30)); got != "9 AM – 4:30 PM" {
		t.Errorf("CompactRange = %q", got)
	}
}

func TestDates(t *testing.T) {
	day := at(10, 0)
	if got := Date(day); got != "Nov 24" {
		t.Errorf("Date = %q", got)
	}
	if got := Date(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)); got != "Jan 5" {
		t.Errorf("Date = %q", got)
	}
	if got := WeekdayDate(day); got != "Tue 24 Nov" {
		t.Errorf("WeekdayDate = %q", got)
	}
	if got := NumericDate(time.Date(2030, 3, 7, 0, 0, 0, 0, time.UTC)); got != "07.03.30" {
		t.Errorf("NumericDate = %q", got)
	}
	if got := NumericDate(day); got != "24.11.26" {
		t.Errorf("NumericDate = %q", got)
	}
}
