package database

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/scheduler"
	"sync"
	"testing"
	"time"
)

func TestSchedulerConcurrentReservationsAndStaleDraft(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC)
	w, err := s.SchedulerWeek(ctx, "2030-01-07", now, nil)
	if err != nil {
		t.Fatal(err)
	}
	session := scheduler.Session{Date: "2030-01-07", Assignment: scheduler.Assignment{Title: "Work"}, Plan: &scheduler.Plan{Start: now.Add(9 * time.Hour), End: now.Add(10 * time.Hour)}}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := s.SchedulerMutate(ctx, scheduler.Mutation{Revision: w.Revision, Week: w.Week, Action: "session", Session: &session}, now, nil)
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	success, conflicts := 0, 0
	for e := range results {
		if e == nil {
			success++
		} else if errors.Is(e, ErrConflict) {
			conflicts++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatalf("success=%d conflicts=%d", success, conflicts)
	}
	fresh, err := s.SchedulerWeek(ctx, w.Week, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(fresh.Sessions) != 1 {
		t.Fatalf("sessions=%d", len(fresh.Sessions))
	}
	_, err = s.SchedulerMutate(ctx, scheduler.Mutation{Revision: fresh.Revision, Week: w.Week, Action: "session", Session: &session}, now, nil)
	var reservation *scheduler.Conflict
	if !errors.As(err, &reservation) {
		t.Fatalf("overlapping fresh save: %v", err)
	}
}
func TestSchedulerIdleTickDoesNotRotateRevisionOrDirtyOutbox(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC)
	w, err := s.SchedulerWeek(ctx, "2030-01-07", now, nil)
	if err != nil {
		t.Fatal(err)
	}
	var generationBefore int64
	if err = s.DB.QueryRowContext(ctx, "SELECT generation FROM scheduler_google_outbox WHERE id=1").Scan(&generationBefore); err != nil {
		t.Fatal(err)
	}
	// An idle worker tick a minute later, with nothing else having changed,
	// must not rotate the revision (open scheduler tabs would spuriously
	// conflict) or dirty the Google export outbox (a full resync every tick).
	if err = s.SchedulerTick(ctx, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	var generationAfter int64
	if err = s.DB.QueryRowContext(ctx, "SELECT generation FROM scheduler_google_outbox WHERE id=1").Scan(&generationAfter); err != nil {
		t.Fatal(err)
	}
	if generationAfter != generationBefore {
		t.Fatalf("idle tick dirtied the Google outbox: %d -> %d", generationBefore, generationAfter)
	}
	session := scheduler.Session{Date: "2030-01-07", Assignment: scheduler.Assignment{Title: "Work"}, Plan: &scheduler.Plan{Start: now.Add(9 * time.Hour), End: now.Add(10 * time.Hour)}}
	if _, err = s.SchedulerMutate(ctx, scheduler.Mutation{Revision: w.Revision, Week: w.Week, Action: "session", Session: &session}, now.Add(time.Minute), nil); err != nil {
		t.Fatalf("idle tick stale-revisioned an open draft: %v", err)
	}
}
func TestImportDoesNotFabricateAlreadySkippedOccurrence(t *testing.T) {
	s := imported(t)
	ctx := context.Background()
	// A document snapshot as it would look right after the anti-fabrication
	// guard skipped a newly-eligible-but-already-past occurrence: the rule
	// and an eligible goal exist, but no session was ever generated for the
	// Monday the rule matches, and the document is reconciled through that
	// same date. 2020-01-06 is a Monday, far enough in the past that this
	// doesn't depend on wall-clock time at test-run time.
	hours := 1.0
	doc := scheduler.Document{
		Revision:    uuid.NewString(),
		Settings:    scheduler.Settings{TimeZone: "UTC", Initialized: true, DefaultDay: scheduler.DayInterval{Start: "05:00", End: "20:30"}, Weekdays: map[string]scheduler.DayInterval{}, Dates: map[string]scheduler.DayInterval{}},
		Goals:       map[string]scheduler.Goal{"g": {ID: "g", Title: "G", StartDate: "2020-01-01", EndDate: "2030-01-01", DailyHours: &hours, Status: "planned"}},
		Sessions:    map[string]scheduler.Session{},
		Rules:       map[string]scheduler.Rule{"r": {ID: "r", Weekday: 1, LocalStart: "09:00", DurationMinutes: 60, EffectiveFrom: "2020-01-06", Assignment: scheduler.Assignment{GoalID: "g"}}},
		ClosedWeeks: map[string][]scheduler.Goal{},
		LastDate:    "2020-01-06",
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.ExecContext(ctx, "INSERT INTO scheduler_state(id,document,reconciled_at) VALUES(1,$1::jsonb,$2)", raw, time.Date(2020, 1, 6, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	var archive bytes.Buffer
	if err = s.Export(ctx, &archive); err != nil {
		t.Fatal(err)
	}
	dest := testStore(t)
	if _, err = dest.Import(ctx, bytes.NewReader(archive.Bytes()), Source{}, false); err != nil {
		t.Fatal(err)
	}
	restored, err := dest.SchedulerDocument(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := restored.Sessions["r:2020-01-06"]; ok {
		t.Fatal("import fabricated an occurrence the export had already skipped")
	}
}
func TestSchedulerBackupRoundtripAndInitialTimeZone(t *testing.T) {
	s := imported(t)
	ctx := context.Background()
	if err := s.SchedulerInitializeTimeZone(ctx, "America/Chicago"); err != nil {
		t.Fatal(err)
	}
	if err := s.SchedulerInitializeTimeZone(ctx, "Asia/Tokyo"); err != nil {
		t.Fatal(err)
	}
	doc, err := s.SchedulerDocument(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Settings.TimeZone != "America/Chicago" {
		t.Fatal("browser changed persisted zone")
	}
	var archive bytes.Buffer
	if err = s.Export(ctx, &archive); err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err = json.Unmarshal(archive.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if raw["scheduler"] == nil {
		t.Fatal("schedule missing from export")
	}
	dest := testStore(t)
	if _, err = dest.Import(ctx, bytes.NewReader(archive.Bytes()), Source{}, false); err != nil {
		t.Fatal(err)
	}
	restored, err := dest.SchedulerDocument(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Settings.TimeZone != "America/Chicago" {
		t.Fatal("schedule not restored")
	}
}

func TestSchedulerAcceptedActualsSurviveExportImport(t *testing.T) {
	s := imported(t)
	ctx := context.Background()
	week := "2026-09-14"
	goalID := "22222222-2222-4222-8222-222222222222"
	before := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
	w, err := s.SchedulerWeek(ctx, week, before, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan := scheduler.Session{Date: week, Assignment: scheduler.Assignment{GoalID: goalID}, Plan: &scheduler.Plan{Start: before.Add(time.Hour), End: before.Add(2 * time.Hour)}}
	w, err = s.SchedulerMutate(ctx, scheduler.Mutation{Action: "session", Week: week, Revision: w.Revision, Session: &plan}, before, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Sessions) != 1 {
		t.Fatalf("planned sessions: %+v", w.Sessions)
	}
	plannedID := w.Sessions[0].ID
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	w, err = s.SchedulerMutate(ctx, scheduler.Mutation{Action: "actual", Week: week, Revision: w.Revision, ID: plannedID, Actual: &scheduler.Actual{Status: "skipped", Date: "wrong-date"}}, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	forged := scheduler.Session{RuleID: "missing", OccurrenceDate: "bad", Date: "bad", Exception: true, Attention: "stale", ConflictIDs: []string{"other"}, Assignment: scheduler.Assignment{GoalID: goalID}}
	explicit := scheduler.Actual{Status: "explicit", Date: week, Start: before.Add(3 * time.Hour), End: before.Add(4 * time.Hour)}
	w, err = s.SchedulerMutate(ctx, scheduler.Mutation{Action: "actual", Week: week, Revision: w.Revision, Session: &forged, Actual: &explicit}, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Sessions) != 2 {
		t.Fatalf("actual sessions: %+v", w.Sessions)
	}
	var archive bytes.Buffer
	if err = s.Export(ctx, &archive); err != nil {
		t.Fatal(err)
	}
	dest := testStore(t)
	if _, err = dest.Import(ctx, bytes.NewReader(archive.Bytes()), Source{}, false); err != nil {
		t.Fatalf("accepted actuals rejected on import: %v", err)
	}
	restored, err := dest.SchedulerDocument(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Sessions) != 2 || restored.Sessions[plannedID].Actual == nil || restored.Sessions[plannedID].Actual.Status != "skipped" || restored.Sessions[plannedID].Actual.Date != week {
		t.Fatalf("skipped actual lost: %+v", restored.Sessions)
	}
	var foundExplicit bool
	for _, session := range restored.Sessions {
		if session.Actual != nil && session.Actual.Status == "explicit" {
			foundExplicit = true
			if session.Date != week || session.RuleID != "" || session.OccurrenceDate != "" || session.Exception || session.Attention != "" || len(session.ConflictIDs) != 0 {
				t.Fatalf("forged identity survived roundtrip: %+v", session)
			}
		}
	}
	if !foundExplicit {
		t.Fatal("explicit actual lost")
	}
}
