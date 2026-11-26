package database

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/linkedin"
)

func queuePerson(t *testing.T, s *Store, name string, extra Entity) Result {
	t.Helper()
	e := Entity{"id": uuid.NewString(), "name": name, "role": "", "companyName": "", "email": "", "queued": false, "notes": "", "tags": []any{}}
	for k, v := range extra {
		e[k] = v
	}
	r, err := s.Save(context.Background(), "connection", e, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestConnectionStageField(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	plain := queuePerson(t, s, "Plain", nil)
	if _, ok := plain.Entry["stage"]; ok {
		t.Fatalf("new connection has a stage: %v", plain.Entry)
	}
	if _, err := s.Save(ctx, "connection", Entity{"id": uuid.NewString(), "name": "Bad", "role": "", "companyName": "", "email": "", "queued": false, "notes": "", "tags": []any{}, "stage": "waiting"}, nil); err == nil {
		t.Fatal("accepted an unknown stage")
	}
	staged := queuePerson(t, s, "Staged", Entity{"companyName": "Acme", "stage": "follow_up", "snoozedUntil": "2026-12-01"})
	if staged.Entry["stage"] != "follow_up" || staged.Entry["snoozedUntil"] != "2026-12-01" {
		t.Fatalf("stage not saved: %v", staged.Entry)
	}
	// list_connection_companies shows each sampled person's stage.
	page, err := s.ConnectionCompanies(ctx, ConnectionCompanyFilter{Limit: 50, People: 5, Query: "acme"})
	if err != nil || len(page.Companies) != 1 || len(page.Companies[0].People) != 1 || page.Companies[0].People[0].Stage != "follow_up" {
		t.Fatalf("company people %+v %v", page, err)
	}
	// An empty stage, as MCP's update tool sends to clear it, leaves the queue.
	staged.Entry["stage"], staged.Entry["snoozedUntil"] = "", ""
	cleared, err := s.Save(ctx, "connection", staged.Entry, &staged.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cleared.Entry["stage"]; ok {
		t.Fatalf("empty stage kept: %v", cleared.Entry)
	}
	if _, ok := cleared.Entry["snoozedUntil"]; ok {
		t.Fatalf("empty snooze kept: %v", cleared.Entry)
	}
	// Clearing only the stage also ends the snooze, so a later re-queue is not hidden.
	again := queuePerson(t, s, "Again", Entity{"stage": "messaged", "snoozedUntil": "2026-12-01"})
	again.Entry["stage"] = ""
	if again, err = s.Save(ctx, "connection", again.Entry, &again.Revision); err != nil || again.Entry["snoozedUntil"] != nil {
		t.Fatalf("snooze outlived the stage: %v %v", again.Entry, err)
	}
	base := Entity{"id": uuid.NewString(), "name": "V", "role": "", "companyName": "", "email": "", "queued": false, "notes": "", "tags": []any{}}
	for _, extra := range []Entity{{"stage": float64(5)}, {"stage": nil}, {"snoozedUntil": "2026-12-01"}} {
		e := Entity{}
		for k, v := range base {
			e[k] = v
		}
		for k, v := range extra {
			e[k] = v
		}
		if err := Validate("connection", e); err == nil {
			t.Fatalf("Validate accepted %v", extra)
		}
	}
}

func TestConnectionQueueActions(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	// Sunday 9 PM in Chicago is already Monday in UTC; the week must follow
	// the scheduler's zone.
	if err := s.SchedulerInitializeTimeZone(ctx, "America/Chicago"); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 30, 3, 0, 0, 0, time.UTC)
	ada := queuePerson(t, s, "Ada", Entity{"lastContactedOn": "2026-09-11"})
	bob := queuePerson(t, s, "Bob", nil)
	cy := queuePerson(t, s, "Cy", nil)
	id := func(r Result) string { return r.Entry["id"].(string) }

	q, err := s.ConnectionQueue(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if q.Today != "2026-11-29" || q.Week != (CatchUpWeek{Start: "2026-11-23", End: "2026-11-29", Sent: 0, Target: 2}) || len(q.Connections) != 0 {
		t.Fatalf("empty queue %+v", q)
	}

	if _, err = s.SetConnectionStage(ctx, id(ada), ada.Revision, "waiting"); !errors.Is(err, ErrInvalid) {
		t.Fatal("unknown stage", err)
	}
	if _, err = s.SetConnectionStage(ctx, id(ada), "stale", "to_contact"); !errors.Is(err, ErrConflict) {
		t.Fatal("stale revision", err)
	}
	if _, err = s.SetConnectionStage(ctx, uuid.NewString(), "x", "to_contact"); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing connection", err)
	}
	if _, err = s.SetConnectionStage(ctx, "not-a-uuid", "x", "to_contact"); !errors.Is(err, ErrInvalid) {
		t.Fatal("bad id", err)
	}
	if ada, err = s.SetConnectionStage(ctx, id(ada), ada.Revision, "to_contact"); err != nil || ada.Entry["stage"] != "to_contact" {
		t.Fatal(ada, err)
	}
	if bob, err = s.SetConnectionStage(ctx, id(bob), bob.Revision, "referral_ask"); err != nil {
		t.Fatal(err)
	}
	if cy, err = s.SetConnectionStage(ctx, id(cy), cy.Revision, "to_contact"); err != nil {
		t.Fatal(err)
	}
	q, err = s.ConnectionQueue(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Connections) != 3 || q.Connections[0]["name"] != "Ada" || q.Connections[1]["name"] != "Cy" || q.Connections[2]["name"] != "Bob" || q.Revisions[id(bob)] != bob.Revision {
		t.Fatalf("queue order %+v", q)
	}

	// Snooze: after today, within a year, only for queued people.
	plain := queuePerson(t, s, "Plain", nil)
	if _, err = s.SnoozeConnection(ctx, id(plain), plain.Revision, "2026-12-06", now); !errors.Is(err, ErrInvalid) {
		t.Fatal("snoozed someone not queued", err)
	}
	for _, until := range []string{"2026-11-29", "2027-11-30", "12/06/2026"} {
		if _, err = s.SnoozeConnection(ctx, id(cy), cy.Revision, until, now); !errors.Is(err, ErrInvalid) {
			t.Fatal("snooze until", until, err)
		}
	}
	for _, until := range []string{"2026-11-30", "2027-11-29", "2026-12-06"} {
		if cy, err = s.SnoozeConnection(ctx, id(cy), cy.Revision, until, now); err != nil || cy.Entry["snoozedUntil"] != until {
			t.Fatal(until, cy, err)
		}
	}
	// Snoozed people stay in the listing; clients hide them by date.
	if q, err = s.ConnectionQueue(ctx, now); err != nil || len(q.Connections) != 3 || q.Connections[1]["snoozedUntil"] != "2026-12-06" {
		t.Fatalf("snoozed listing %+v %v", q, err)
	}
	// Moving clears the snooze.
	if cy, err = s.SetConnectionStage(ctx, id(cy), cy.Revision, "messaged"); err != nil || cy.Entry["snoozedUntil"] != nil {
		t.Fatal(cy, err)
	}

	// Sent: last contact becomes today, the stage advances, the week counts it.
	if _, _, err = s.MarkConnectionSent(ctx, id(plain), plain.Revision, now); !errors.Is(err, ErrInvalid) {
		t.Fatal("sent to someone not queued", err)
	}
	if _, _, err = s.MarkConnectionSent(ctx, id(ada), "stale", now); !errors.Is(err, ErrConflict) {
		t.Fatal("stale sent", err)
	}
	if _, _, err = s.MarkConnectionSent(ctx, uuid.NewString(), "x", now); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing sent", err)
	}
	firstRev := ada.Revision
	ada, week, err := s.MarkConnectionSent(ctx, id(ada), ada.Revision, now)
	if err != nil {
		t.Fatal(err)
	}
	if ada.Entry["stage"] != "messaged" || ada.Entry["lastContactedOn"] != "2026-11-29" || week.Sent != 1 || week.Start != "2026-11-23" {
		t.Fatalf("after sent %v %+v", ada.Entry, week)
	}
	// A repeated click with the old revision is a conflict, not a second send.
	if _, _, err = s.MarkConnectionSent(ctx, id(ada), firstRev, now); !errors.Is(err, ErrConflict) {
		t.Fatal("double send", err)
	}
	// The last stage's send takes the person out of the queue.
	bob, week, err = s.MarkConnectionSent(ctx, id(bob), bob.Revision, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := bob.Entry["stage"]; ok || week.Sent != 2 {
		t.Fatalf("after referral ask %v %+v", bob.Entry, week)
	}
	// A send never moves last contact backwards.
	cy.Entry["lastContactedOn"] = "2026-12-25"
	if cy, err = s.Save(ctx, "connection", cy.Entry, &cy.Revision); err != nil {
		t.Fatal(err)
	}
	if cy, _, err = s.MarkConnectionSent(ctx, id(cy), cy.Revision, now); err != nil || cy.Entry["lastContactedOn"] != "2026-12-25" || cy.Entry["stage"] != "follow_up" {
		t.Fatal(cy, err)
	}

	// Next week starts a new count; deleting a person keeps their sends.
	q, err = s.ConnectionQueue(ctx, now.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if q.Week.Start != "2026-11-30" || q.Week.Sent != 0 {
		t.Fatalf("next week %+v", q.Week)
	}
	// A send on the week's first day counts in that week, not the one before.
	monday := now.Add(24 * time.Hour)
	if ada, week, err = s.MarkConnectionSent(ctx, id(ada), ada.Revision, monday); err != nil || week.Start != "2026-11-30" || week.Sent != 1 {
		t.Fatalf("monday send %+v %v", week, err)
	}
	if err = s.Delete(ctx, "connection", id(bob), &bob.Revision); err != nil {
		t.Fatal(err)
	}
	if q, err = s.ConnectionQueue(ctx, now); err != nil || q.Week.Sent != 3 {
		t.Fatalf("after delete %+v %v", q.Week, err)
	}

	// Remove.
	if ada, err = s.SetConnectionStage(ctx, id(ada), ada.Revision, ""); err != nil || ada.Entry["stage"] != nil {
		t.Fatal(ada, err)
	}
	if _, err = s.SetConnectionStage(ctx, id(ada), ada.Revision, ""); !errors.Is(err, ErrInvalid) {
		t.Fatal("removed someone not queued", err)
	}

	// Weekly target.
	for _, target := range []int{0, 51} {
		if _, err = s.SetCatchUpTarget(ctx, target, now); !errors.Is(err, ErrInvalid) {
			t.Fatal("target", target, err)
		}
	}
	if week, err = s.SetCatchUpTarget(ctx, 5, now); err != nil || week.Target != 5 || week.Sent != 3 {
		t.Fatal(week, err)
	}
	if q, err = s.ConnectionQueue(ctx, now); err != nil || q.Week.Target != 5 {
		t.Fatal(q.Week, err)
	}
}

func TestConnectionStageSurvivesReimportAndArchive(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	rows := []linkedin.Connection{{Name: "Ada", Role: "SRE", Company: "Example", URL: "https://www.linkedin.com/in/ada"}}
	if _, err := s.ImportConnections(ctx, rows, nil); err != nil {
		t.Fatal(err)
	}
	page, err := s.List(ctx, "connection", Filter{Limit: 10})
	if err != nil || len(page.Entries) != 1 {
		t.Fatal(page, err)
	}
	ada := page.Entries[0]
	if _, ok := ada.Entry["stage"]; ok {
		t.Fatalf("imported with a stage: %v", ada.Entry)
	}
	id := ada.Entry["id"].(string)
	r, err := s.SetConnectionStage(ctx, id, ada.Revision, "follow_up")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SnoozeConnection(ctx, id, r.Revision, "2026-12-03", time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	rows[0].Role, rows[0].Company = "Staff SRE", "Elsewhere"
	got, err := s.ImportConnections(ctx, rows, map[string]string{"https://www.linkedin.com/in/ada": "2026-10-01"})
	if err != nil || got.Updated != 1 {
		t.Fatal(got, err)
	}
	after, err := s.Detail(ctx, "connection", id)
	if err != nil {
		t.Fatal(err)
	}
	if after.Entry["role"] != "Staff SRE" || after.Entry["stage"] != "follow_up" || after.Entry["snoozedUntil"] == nil {
		t.Fatalf("re-import lost the stage: %v", after.Entry)
	}

	var out bytes.Buffer
	if err = s.Export(ctx, &out); err != nil {
		t.Fatal(err)
	}
	var workspace map[string]any
	if err = json.Unmarshal(out.Bytes(), &workspace); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"notes", "weeks", "books", "companies", "documents"} {
		if workspace[key] == nil {
			workspace[key] = []any{}
		}
	}
	raw, _ := json.Marshal(workspace)
	restored := testStore(t)
	if _, err = restored.Import(ctx, bytes.NewReader(raw), Source{Store: "fixture", Key: "stages", Checksum: uuid.NewString(), Archive: "test"}, false); err != nil {
		t.Fatal(err)
	}
	back, err := restored.Detail(ctx, "connection", id)
	if err != nil || back.Entry["stage"] != "follow_up" || back.Entry["snoozedUntil"] != after.Entry["snoozedUntil"] {
		t.Fatalf("archive lost the stage: %v %v", back.Entry, err)
	}
}

// The frontend's stage list must match the server's.
func TestConnectionStagesMatchFrontend(t *testing.T) {
	src, err := os.ReadFile("../../src/lib/connections.ts")
	if err != nil {
		t.Fatal(err)
	}
	want := "CONNECTION_STAGES = ['" + strings.ReplaceAll(ConnectionStages, "|", "', '") + "'] as const"
	if !strings.Contains(string(src), want) {
		t.Fatalf("src/lib/connections.ts does not declare %s", want)
	}
}
