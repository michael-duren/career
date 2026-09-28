package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/scheduler"
	"time"
)

func schedulerLoad(ctx context.Context, q queryer) (scheduler.Document, error) {
	d := scheduler.New()
	var raw []byte
	e := q.QueryRowContext(ctx, "SELECT document FROM scheduler_state WHERE id=1").Scan(&raw)
	if errors.Is(e, sql.ErrNoRows) {
		return d, nil
	}
	if e != nil {
		return d, e
	}
	e = json.Unmarshal(raw, &d)
	return d, e
}
func schedulerGoals(ctx context.Context, q queryer) ([]scheduler.Goal, error) {
	rows, e := q.QueryContext(ctx, "SELECT id FROM goals ORDER BY id")
	if e != nil {
		return nil, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return nil, e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	goals := []scheduler.Goal{}
	for _, id := range ids {
		r, e := readOne(ctx, q, "goal", id)
		if e != nil {
			return nil, e
		}
		raw, _ := json.Marshal(r.Entry)
		var g scheduler.Goal
		if e = json.Unmarshal(raw, &g); e != nil {
			return nil, e
		}
		steps, _ := r.Entry["steps"].([]any)
		for i, rawStep := range steps {
			step, _ := rawStep.(map[string]any)
			done, _ := step["done"].(bool)
			if i < len(g.Steps) {
				g.Steps[i].Completed = done
			}
		}
		goals = append(goals, g)
	}
	return goals, nil
}
func schedulerSave(ctx context.Context, tx *sql.Tx, d scheduler.Document) error {
	raw, e := json.Marshal(d)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO scheduler_state(id,document) VALUES(1,$1::jsonb) ON CONFLICT(id) DO UPDATE SET document=EXCLUDED.document", raw)
	return e
}
func schedulerReconcileTx(ctx context.Context, tx *sql.Tx, now time.Time) error {
	if e := lockGoals(ctx, tx); e != nil {
		return e
	}
	d, e := schedulerLoad(ctx, tx)
	if e != nil {
		return e
	}
	before, e := json.Marshal(d)
	if e != nil {
		return e
	}
	goals, e := schedulerGoals(ctx, tx)
	if e != nil {
		return e
	}
	loc, _ := time.LoadLocation(d.Settings.TimeZone)
	week := scheduler.Monday(now.In(loc).Format("2006-01-02"))
	sinceDate := d.LastDate
	sinceInstant := d.LastReconciledAt
	from, to := schedulerRange(d, week, sinceDate)
	d.Generate(from, to, now, d.Busy, sinceInstant)
	d.Reconcile(goals, now)
	d.Revalidate(now, d.Busy)
	after, e := json.Marshal(d)
	if e != nil {
		return e
	}
	if string(before) == string(after) {
		return nil
	}
	d.Revision = uuid.NewString()
	return schedulerSave(ctx, tx, d)
}
func (s *Store) SchedulerDocument(ctx context.Context) (scheduler.Document, error) {
	return schedulerLoad(ctx, s.DB)
}

// schedulerRange bounds the occurrence-generation window. from starts at the
// earliest active rule, but never earlier than sinceDate: dates before that
// watermark were already reconciled, so re-scanning them on every call would
// make generation cost grow without bound as rules and history accumulate.
func schedulerRange(d scheduler.Document, w string, sinceDate string) (string, string) {
	from := w
	to := scheduler.DateAdd(w, 62)
	for _, r := range d.Rules {
		if r.EffectiveFrom < from {
			from = r.EffectiveFrom
		}
	}
	if sinceDate != "" && from < sinceDate {
		from = sinceDate
	}
	return from, to
}
func (s *Store) schedulerUpdate(ctx context.Context, w string, m *scheduler.Mutation, now time.Time, busy []scheduler.Busy) (scheduler.Week, error) {
	if !scheduler.ValidDate(w) || scheduler.Monday(w) != w {
		return scheduler.Week{}, fmt.Errorf("%w: week must be a Monday date", ErrInvalid)
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return scheduler.Week{}, e
	}
	defer tx.Rollback()
	if e = lockGoals(ctx, tx); e != nil {
		return scheduler.Week{}, e
	}
	d, e := schedulerLoad(ctx, tx)
	if e != nil {
		return scheduler.Week{}, e
	}
	before, e := json.Marshal(d)
	if e != nil {
		return scheduler.Week{}, e
	}
	if busy == nil {
		busy = d.Busy
	} else {
		d.Busy = busy
	}
	goals, e := schedulerGoals(ctx, tx)
	if e != nil {
		return scheduler.Week{}, e
	}
	sinceDate := d.LastDate
	sinceInstant := d.LastReconciledAt
	d.Reconcile(goals, now)
	if m != nil && m.Revision != d.Revision {
		return d.Week(w, now, busy), ErrConflict
	}
	originalRules := map[string]bool{}
	for id := range d.Rules {
		originalRules[id] = true
	}
	if m != nil {
		baseline, be := json.Marshal(d)
		if be != nil {
			return scheduler.Week{}, be
		}
		if e = d.Apply(*m, now, busy); e != nil {
			if ue := json.Unmarshal(baseline, &d); ue != nil {
				return scheduler.Week{}, ue
			}
			var c *scheduler.Conflict
			if errors.As(e, &c) {
				return d.Week(w, now, busy), e
			}
			return d.Week(w, now, busy), fmt.Errorf("%w: %s", ErrInvalid, e)
		}
	}
	from, to := schedulerRange(d, w, sinceDate)
	d.Generate(from, to, now, busy, sinceInstant)
	if m != nil && m.Action == "rule" {
		dates := []string{}
		ids := []string{}
		for _, session := range d.Sessions {
			if session.RuleID != "" && !originalRules[session.RuleID] && session.State == "attention" && session.Plan != nil {
				dates = append(dates, session.Date)
				ids = append(ids, session.ConflictIDs...)
			}
		}
		if len(dates) > 0 {
			var original scheduler.Document
			if ue := json.Unmarshal(before, &original); ue != nil {
				return scheduler.Week{}, ue
			}
			return original.Week(w, now, busy), &scheduler.Conflict{Message: fmt.Sprintf("Recurring plan needs placement on %v", dates), IDs: ids}
		}
	}
	after, e := json.Marshal(d)
	if e != nil {
		return scheduler.Week{}, e
	}
	if string(before) != string(after) {
		d.Revision = uuid.NewString()
		if e = schedulerSave(ctx, tx, d); e != nil {
			return scheduler.Week{}, e
		}
	}
	out := d.Week(w, now, busy)
	return out, tx.Commit()
}
func (s *Store) SchedulerWeek(ctx context.Context, week string, now time.Time, busy []scheduler.Busy) (scheduler.Week, error) {
	return s.schedulerUpdate(ctx, week, nil, now, busy)
}
func (s *Store) SchedulerMutate(ctx context.Context, m scheduler.Mutation, now time.Time, busy []scheduler.Busy) (scheduler.Week, error) {
	return s.schedulerUpdate(ctx, m.Week, &m, now, busy)
}
func (s *Store) SchedulerTick(ctx context.Context, now time.Time) error {
	d, e := s.SchedulerDocument(ctx)
	if e != nil {
		return e
	}
	loc, e := time.LoadLocation(d.Settings.TimeZone)
	if e != nil {
		return e
	}
	_, e = s.SchedulerWeek(ctx, scheduler.Monday(now.In(loc).Format("2006-01-02")), now, nil)
	return e
}

// SchedulerInitializeTimeZone applies the browser zone once, before the user's
// first scheduler view; subsequent browser changes never alter saved settings.
func (s *Store) SchedulerInitializeTimeZone(ctx context.Context, zone string) error {
	if _, e := time.LoadLocation(zone); e != nil {
		return fmt.Errorf("%w: invalid IANA time zone", ErrInvalid)
	}
	tx, e := s.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if e = lockGoals(ctx, tx); e != nil {
		return e
	}
	d, e := schedulerLoad(ctx, tx)
	if e != nil {
		return e
	}
	if !d.Settings.Initialized {
		d.Settings.TimeZone = zone
		d.Settings.Initialized = true
		d.Revision = uuid.NewString()
		if e = schedulerSave(ctx, tx, d); e != nil {
			return e
		}
	}
	return tx.Commit()
}
