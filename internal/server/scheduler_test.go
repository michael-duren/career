package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/michael-duren/career-strategy/internal/scheduler"
)

func mustSchedulerTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestSchedulerActualRouteStoresCanonicalUnplannedSession(t *testing.T) {
	db := testDB(t)
	h := newOAuthHarness(t, db)
	read := h.do("GET", "/api/scheduler/week?week=2026-09-14", "", nil, true)
	if read.Code != 200 {
		t.Fatalf("week: %d %s", read.Code, read.Body.String())
	}
	var week scheduler.Week
	if err := json.Unmarshal(read.Body.Bytes(), &week); err != nil {
		t.Fatal(err)
	}
	input := scheduler.Mutation{
		Action: "actual", Week: week.Week, Revision: week.Revision,
		Session: &scheduler.Session{ID: "forged", RuleID: "missing-rule", OccurrenceDate: "bad-date", Date: "bad-date", Exception: true, State: "attention", Attention: "stale", ConflictIDs: []string{"old"}, Assignment: scheduler.Assignment{GoalID: "22222222-2222-4222-8222-222222222222"}},
		Actual:  &scheduler.Actual{Status: "explicit", Date: "2026-09-14", Start: mustSchedulerTime(t, "2026-09-14T09:00:00Z"), End: mustSchedulerTime(t, "2026-09-14T10:00:00Z")},
	}
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	write := h.do("POST", "/api/scheduler/mutate", string(raw), map[string]string{"Origin": testOrigin, "Content-Type": "application/json"}, true)
	if write.Code != 200 {
		t.Fatalf("actual: %d %s", write.Code, write.Body.String())
	}
	var result scheduler.Week
	if err := json.Unmarshal(write.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Sessions) != 1 {
		t.Fatalf("sessions: %+v", result.Sessions)
	}
	s := result.Sessions[0]
	if s.ID == "forged" || s.RuleID != "" || s.OccurrenceDate != "" || s.Date != "2026-09-14" || s.Exception || s.Attention != "" || len(s.ConflictIDs) != 0 || s.Actual == nil {
		t.Fatalf("noncanonical API result: %+v", s)
	}
	doc, err := db.SchedulerDocument(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Validate(); err != nil {
		t.Fatalf("API accepted invalid stored record: %v", err)
	}
	input.Revision = result.Revision
	input.Actual.Date = "invalid"
	raw, err = json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	write = h.do("POST", "/api/scheduler/mutate", string(raw), map[string]string{"Origin": testOrigin, "Content-Type": "application/json"}, true)
	if write.Code != 400 {
		t.Fatalf("malformed actual date: %d %s", write.Code, write.Body.String())
	}
}

func TestSchedulerActualRouteDerivesOvernightDates(t *testing.T) {
	db := testDB(t)
	h := newOAuthHarness(t, db)
	headers := map[string]string{"Origin": testOrigin, "Content-Type": "application/json"}
	read := h.do("GET", "/api/scheduler/week?week=2026-09-14&timeZone=America%2FChicago", "", nil, true)
	if read.Code != 200 {
		t.Fatalf("week: %d %s", read.Code, read.Body.String())
	}
	var week scheduler.Week
	if err := json.Unmarshal(read.Body.Bytes(), &week); err != nil {
		t.Fatal(err)
	}
	settings := week.Settings
	settings.DefaultDay = scheduler.DayInterval{Start: "09:00", End: "02:00", NextDay: true}
	settings.Dates = map[string]scheduler.DayInterval{"2026-09-20": {Start: "09:00", End: "03:00", NextDay: true}}
	settingsRaw, _ := json.Marshal(scheduler.Mutation{Action: "settings", Week: week.Week, Revision: week.Revision, Settings: &settings})
	write := h.do("POST", "/api/scheduler/mutate", string(settingsRaw), headers, true)
	if write.Code != 200 {
		t.Fatalf("settings: %d %s", write.Code, write.Body.String())
	}
	if err := json.Unmarshal(write.Body.Bytes(), &week); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, week, date, start, end, want string }{
		{"Sunday into Monday", "2026-09-21", "2026-09-28", "2026-09-28T06:00:00Z", "2026-09-28T06:30:00Z", "2026-09-27"},
		{"date override into Monday", "2026-09-14", "2026-09-21", "2026-09-21T07:30:00Z", "2026-09-21T08:00:00Z", "2026-09-20"},
		{"outside configured hours", "2026-09-28", "2026-09-27", "2026-09-28T09:00:00Z", "2026-09-28T09:30:00Z", "2026-09-28"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			read := h.do("GET", "/api/scheduler/week?week="+tc.week, "", nil, true)
			if read.Code != 200 {
				t.Fatalf("read: %d %s", read.Code, read.Body.String())
			}
			var current scheduler.Week
			if err := json.Unmarshal(read.Body.Bytes(), &current); err != nil {
				t.Fatal(err)
			}
			input := scheduler.Mutation{Action: "actual", Week: tc.week, Revision: current.Revision, Session: &scheduler.Session{Assignment: scheduler.Assignment{GoalID: "22222222-2222-4222-8222-222222222222"}}, Actual: &scheduler.Actual{Status: "explicit", Date: tc.date, Start: mustSchedulerTime(t, tc.start), End: mustSchedulerTime(t, tc.end)}}
			raw, _ := json.Marshal(input)
			write := h.do("POST", "/api/scheduler/mutate", string(raw), headers, true)
			if write.Code != 200 {
				t.Fatalf("actual: %d %s", write.Code, write.Body.String())
			}
			var result scheduler.Week
			if err := json.Unmarshal(write.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			found := false
			for _, session := range result.Sessions {
				if session.Actual != nil && session.Actual.Start.Equal(input.Actual.Start) {
					found = true
					if session.Actual.Date != tc.want || session.Date != tc.want {
						t.Fatalf("canonical dates: session=%s actual=%s want=%s", session.Date, session.Actual.Date, tc.want)
					}
				}
			}
			if !found {
				t.Fatalf("canonical actual missing from response: %+v", result.Sessions)
			}
			doc, err := db.SchedulerDocument(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if err := doc.Validate(); err != nil {
				t.Fatalf("accepted API actual fails import validation: %v", err)
			}
		})
	}
}
