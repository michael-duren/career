package database

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/scheduler"
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

func TestRuleChecksKnownReservationBeyondGenerationWindow(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Date(2026, 1, 5, 8, 0, 0, 0, time.UTC)
	week := "2026-01-05"
	reservationWeek := "2026-07-06"
	w, err := s.SchedulerWeek(ctx, reservationWeek, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	reservation := scheduler.Session{Date: reservationWeek, Assignment: scheduler.Assignment{Title: "One-off"}, Plan: &scheduler.Plan{Start: time.Date(2026, 7, 6, 9, 0, 0, 0, time.UTC), End: time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)}}
	w, err = s.SchedulerMutate(ctx, scheduler.Mutation{Revision: w.Revision, Week: reservationWeek, Action: "session", Session: &reservation}, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	w, err = s.SchedulerWeek(ctx, week, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	makeRule := func(start string) scheduler.Mutation {
		return scheduler.Mutation{Revision: w.Revision, Week: week, Action: "rule", Rule: &scheduler.Rule{Weekday: 1, LocalStart: start, DurationMinutes: 60, EffectiveFrom: week, Assignment: scheduler.Assignment{Title: "Weekly"}}}
	}
	_, err = s.SchedulerMutate(ctx, makeRule("09:00"), now, nil)
	var conflict *scheduler.Conflict
	if !errors.As(err, &conflict) || !strings.Contains(err.Error(), reservationWeek) {
		t.Fatalf("overlapping recurrence should name future date %s; got %v", reservationWeek, err)
	}
	_, err = s.SchedulerMutate(ctx, makeRule("10:00"), now, nil)
	if err != nil {
		t.Fatalf("adjacent future reservation should allow recurrence: %v", err)
	}
}

func TestRuleChecksDatedBoundaryBeyondGenerationWindow(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Date(2026, 1, 5, 8, 0, 0, 0, time.UTC)
	week := "2026-01-05"
	w, err := s.SchedulerWeek(ctx, week, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	settings := scheduler.New().Settings
	settings.Dates["2026-07-06"] = scheduler.DayInterval{Start: "12:00", End: "20:30"}
	w, err = s.SchedulerMutate(ctx, scheduler.Mutation{Revision: w.Revision, Week: week, Action: "settings", Settings: &settings}, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.SchedulerMutate(ctx, scheduler.Mutation{Revision: w.Revision, Week: week, Action: "rule", Rule: &scheduler.Rule{Weekday: 1, LocalStart: "09:00", DurationMinutes: 60, EffectiveFrom: week, Assignment: scheduler.Assignment{Title: "Weekly"}}}, now, nil)
	if err == nil || !strings.Contains(err.Error(), "2026-07-06") {
		t.Fatalf("recurrence outside a dated day boundary should name 2026-07-06; got %v", err)
	}
	doc, err := s.SchedulerDocument(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Rules) != 0 || len(doc.Sessions) != 0 {
		t.Fatalf("rejected rule changed saved schedule: rules=%d sessions=%d", len(doc.Rules), len(doc.Sessions))
	}
}

func TestRuleChecksOvernightReservationOnSchedulingDate(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Date(2026, 1, 5, 8, 0, 0, 0, time.UTC)
	week := "2026-01-05"
	reservationDate := "2026-07-06"
	w, err := s.SchedulerWeek(ctx, week, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	settings := scheduler.New().Settings
	settings.DefaultDay = scheduler.DayInterval{Start: "09:00", End: "02:00", NextDay: true}
	w, err = s.SchedulerMutate(ctx, scheduler.Mutation{Revision: w.Revision, Week: week, Action: "settings", Settings: &settings}, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	w, err = s.SchedulerWeek(ctx, reservationDate, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	reservation := scheduler.Session{Date: reservationDate, Assignment: scheduler.Assignment{Title: "Overnight one-off"}, Plan: &scheduler.Plan{Start: time.Date(2026, 7, 7, 0, 30, 0, 0, time.UTC), End: time.Date(2026, 7, 7, 1, 30, 0, 0, time.UTC)}}
	w, err = s.SchedulerMutate(ctx, scheduler.Mutation{Revision: w.Revision, Week: reservationDate, Action: "session", Session: &reservation}, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	w, err = s.SchedulerWeek(ctx, week, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.SchedulerMutate(ctx, scheduler.Mutation{Revision: w.Revision, Week: week, Action: "rule", Rule: &scheduler.Rule{Weekday: 1, LocalStart: "23:45", DurationMinutes: 60, EffectiveFrom: week, Assignment: scheduler.Assignment{Title: "Overnight weekly"}}}, now, nil)
	var conflict *scheduler.Conflict
	if !errors.As(err, &conflict) || !strings.Contains(err.Error(), reservationDate) {
		t.Fatalf("Monday overnight recurrence should collide with Tuesday 00:30 one-off on scheduling date %s; got %v", reservationDate, err)
	}
}

func TestRuleChecksMatchingOccurrenceInsideDistantBusySpan(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Date(2026, 1, 5, 8, 0, 0, 0, time.UTC)
	week := "2026-01-05"
	w, err := s.SchedulerWeek(ctx, week, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	busy := []scheduler.Busy{{ID: "multi-day", Start: time.Date(2026, 7, 6, 8, 0, 0, 0, time.UTC), End: time.Date(2026, 7, 10, 17, 0, 0, 0, time.UTC)}}
	_, err = s.SchedulerMutate(ctx, scheduler.Mutation{Revision: w.Revision, Week: week, Action: "rule", Rule: &scheduler.Rule{Weekday: 3, LocalStart: "09:00", DurationMinutes: 60, EffectiveFrom: week, Assignment: scheduler.Assignment{Title: "Weekly"}}}, now, busy)
	var conflict *scheduler.Conflict
	if !errors.As(err, &conflict) || !strings.Contains(err.Error(), "2026-07-08") {
		t.Fatalf("recurrence inside a distant multi-day busy reservation should name 2026-07-08; got %v", err)
	}
}

func TestRuleChecksKnownFutureRuleWhenProbeOrdersItAfterNewRule(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Date(2026, 1, 5, 8, 0, 0, 0, time.UTC)
	week := "2026-01-05"
	farDate := "2026-07-06"
	doc := scheduler.New()
	doc.Settings.Dates[farDate] = scheduler.DayInterval{Start: "08:00", End: "20:30"}
	doc.Rules["z-existing"] = scheduler.Rule{ID: "z-existing", Weekday: 1, LocalStart: "09:00", DurationMinutes: 60, EffectiveFrom: farDate, Assignment: scheduler.Assignment{Title: "Existing future weekly"}}
	doc.Sessions["one-off"] = scheduler.Session{ID: "one-off", Date: farDate, Assignment: scheduler.Assignment{Title: "Adjacent one-off"}, State: "accepted", Plan: &scheduler.Plan{Start: time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC), End: time.Date(2026, 7, 6, 13, 0, 0, 0, time.UTC)}}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.ExecContext(ctx, "INSERT INTO scheduler_state(id,document,reconciled_at) VALUES(1,$1::jsonb,$2)", raw, now); err != nil {
		t.Fatal(err)
	}
	w, err := s.SchedulerWeek(ctx, week, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.SchedulerMutate(ctx, scheduler.Mutation{Revision: w.Revision, Week: week, Action: "rule", Rule: &scheduler.Rule{Weekday: 1, LocalStart: "09:00", DurationMinutes: 60, EffectiveFrom: week, Assignment: scheduler.Assignment{Title: "New weekly"}}}, now, nil)
	var conflict *scheduler.Conflict
	if !errors.As(err, &conflict) || !strings.Contains(err.Error(), farDate) {
		t.Fatalf("new recurrence should detect an existing unmaterialized future rule on %s; got %v", farDate, err)
	}
}

func TestRuleChecksPreservedFutureDateException(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	now := time.Date(2026, 1, 5, 8, 0, 0, 0, time.UTC)
	week := "2026-01-05"
	doc := scheduler.New()
	doc.Rules["existing"] = scheduler.Rule{ID: "existing", Weekday: 1, LocalStart: "08:00", DurationMinutes: 60, EffectiveFrom: week, Assignment: scheduler.Assignment{Title: "Existing weekly"}}
	plan := &scheduler.Plan{Start: time.Date(2026, 7, 6, 9, 0, 0, 0, time.UTC), End: time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)}
	doc.Sessions["existing:2026-07-06"] = scheduler.Session{ID: "existing:2026-07-06", RuleID: "existing", OccurrenceDate: "2026-07-06", Date: "2026-07-06", Exception: true, Assignment: scheduler.Assignment{Title: "Preserved exception"}, State: "accepted", Plan: plan}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB.ExecContext(ctx, "INSERT INTO scheduler_state(id,document,reconciled_at) VALUES(1,$1::jsonb,$2)", raw, now); err != nil {
		t.Fatal(err)
	}
	w, err := s.SchedulerWeek(ctx, week, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.SchedulerDocument(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.SchedulerMutate(ctx, scheduler.Mutation{Revision: w.Revision, Week: week, Action: "rule", Rule: &scheduler.Rule{Weekday: 1, LocalStart: "09:00", DurationMinutes: 60, EffectiveFrom: week, Assignment: scheduler.Assignment{Title: "New weekly"}}}, now, nil)
	var conflict *scheduler.Conflict
	if !errors.As(err, &conflict) || !strings.Contains(err.Error(), "2026-07-06") {
		t.Fatalf("recurrence should conflict with preserved exception occurrence; got %v", err)
	}
	after, err := s.SchedulerDocument(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := after.Sessions["existing:2026-07-06"]
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		t.Fatal(err)
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeJSON, afterJSON) || !got.Exception || got.RuleID != "existing" || !got.Plan.Start.Equal(plan.Start) || !got.Plan.End.Equal(plan.End) {
		t.Fatalf("rejected recurrence rewrote future exception/history: rules=%d sessions=%d exception=%+v", len(after.Rules), len(after.Sessions), got)
	}
	for id, session := range after.Sessions {
		if session.Date > scheduler.DateAdd(week, 62) && id != "existing:2026-07-06" {
			t.Fatalf("validation materialized intervening future occurrence %s", id)
		}
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

func TestFutureRuleValidationPreservesRecordedActualException(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	raw := bytes.Replace(fixture(t), []byte(`"endDate": "2026-09-14"`), []byte(`"endDate": "2026-12-31"`), 1)
	if _, err := s.Import(ctx, bytes.NewReader(raw), Source{}, false); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.UTC)
	week := "2026-09-14"
	w, err := s.SchedulerWeek(ctx, week, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	rule := scheduler.Rule{Weekday: 1, LocalStart: "09:00", DurationMinutes: 60, EffectiveFrom: week, Assignment: scheduler.Assignment{GoalID: "22222222-2222-4222-8222-222222222222"}}
	w, err = s.SchedulerMutate(ctx, scheduler.Mutation{Week: week, Revision: w.Revision, Action: "rule", Rule: &rule}, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Sessions) != 1 {
		t.Fatalf("initial occurrences=%+v", w.Sessions)
	}
	original := w.Sessions[0]
	actual := scheduler.Actual{Status: "explicit", Date: week, Start: now.Add(-2 * time.Hour), End: now.Add(-time.Hour)}
	w, err = s.SchedulerMutate(ctx, scheduler.Mutation{Week: week, Revision: w.Revision, Action: "actual", ID: original.ID, Actual: &actual}, now, nil)
	if err != nil {
		t.Fatal(err)
	}
	rule.LocalStart = "10:00"
	w, err = s.SchedulerMutate(ctx, scheduler.Mutation{Week: week, Revision: w.Revision, Action: "rule", ID: original.RuleID, EffectiveFrom: week, Rule: &rule}, now, nil)
	if err != nil {
		t.Fatalf("future template edit rejected preserved actual exception: %v", err)
	}
	doc, err := s.SchedulerDocument(ctx)
	if err != nil {
		t.Fatal(err)
	}
	saved, ok := doc.Sessions[original.ID]
	if !ok || saved.Actual == nil || *saved.Actual != actual || saved.Plan == nil || *saved.Plan != *original.Plan || !saved.Exception || saved.State != "attention" || saved.RuleID == original.RuleID {
		t.Fatalf("template edit changed recorded history: %+v", saved)
	}
	if len(w.Sessions) != 1 || len(w.Goals) != 1 || w.Goals[0].ActualHours != 1 {
		t.Fatalf("actual duplicated or hidden: sessions=%+v goals=%+v", w.Sessions, w.Goals)
	}
	future, err := s.SchedulerWeek(ctx, "2026-09-21", now, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(future.Sessions) != 1 || future.Sessions[0].Plan.Start.Hour() != 10 || future.Sessions[0].Actual != nil || future.Sessions[0].RuleID != saved.RuleID {
		t.Fatalf("new template did not generate future plan: %+v", future.Sessions)
	}
}
