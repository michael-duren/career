package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/scheduler"
)

// ConnectionStages are the catch-up queue stages in order, separated by "|"
// for allowed() and split by NextStage.
const ConnectionStages = "to_contact|messaged|follow_up|referral_ask"

// DefaultCatchUpTarget is the weekly message target until the user sets one.
const DefaultCatchUpTarget = 2

// MaxSnoozeDays bounds how far ahead a snooze may hide someone.
const MaxSnoozeDays = 365

// CatchUpWeek is the weekly message counter: sends dated Monday to Sunday of
// the current week in the scheduler's time zone.
type CatchUpWeek struct {
	Start  string `json:"start"`
	End    string `json:"end"`
	Sent   int    `json:"sent"`
	Target int    `json:"target"`
}

// ConnectionQueue is everyone in the catch-up queue, snoozed people included
// (compare snoozedUntil with Today), ordered by stage then name.
type ConnectionQueue struct {
	Today       string            `json:"today"`
	Week        CatchUpWeek       `json:"week"`
	Connections []Entity          `json:"connections"`
	Revisions   map[string]string `json:"revisions"`
}

// StageList is ConnectionStages for messages: "to_contact, messaged, …".
func StageList() string { return strings.ReplaceAll(ConnectionStages, "|", ", ") }

// ValidStage reports whether stage is a catch-up queue stage.
func ValidStage(stage string) bool { return allowed(stage, ConnectionStages) }

// NextStage returns the stage after stage, or "" after the last one.
func NextStage(stage string) string {
	stages := strings.Split(ConnectionStages, "|")
	for i, s := range stages[:len(stages)-1] {
		if s == stage {
			return stages[i+1]
		}
	}
	return ""
}

// catchUpToday is the current date in the scheduler's time zone, which the
// browser sets on first use, so "today" and "this week" match the user's.
// Until then the scheduler's default zone, UTC, applies. A stored zone that
// no longer loads is an error rather than a silently wrong date.
func catchUpToday(ctx context.Context, q queryer, now time.Time) (string, error) {
	d, err := schedulerLoad(ctx, q)
	if err != nil {
		return "", err
	}
	loc, err := time.LoadLocation(d.Settings.TimeZone)
	if err != nil {
		return "", fmt.Errorf("load scheduler time zone %q: %w", d.Settings.TimeZone, err)
	}
	return now.In(loc).Format("2006-01-02"), nil
}

func catchUpWeek(ctx context.Context, q queryer, today string) (CatchUpWeek, error) {
	w := CatchUpWeek{Start: scheduler.Monday(today), Target: DefaultCatchUpTarget}
	w.End = scheduler.DateAdd(w.Start, 6)
	err := q.QueryRowContext(ctx, "SELECT weekly_target FROM catch_up_settings WHERE id=1").Scan(&w.Target)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return w, err
	}
	err = q.QueryRowContext(ctx, "SELECT count(*) FROM connection_sends WHERE sent_on BETWEEN $1 AND $2", w.Start, w.End).Scan(&w.Sent)
	return w, err
}

// ConnectionQueue returns the catch-up queue and this week's counter.
func (s *Store) ConnectionQueue(ctx context.Context, now time.Time) (ConnectionQueue, error) {
	out := ConnectionQueue{Connections: []Entity{}, Revisions: map[string]string{}}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	if out.Today, err = catchUpToday(ctx, tx, now); err != nil {
		return out, err
	}
	if out.Week, err = catchUpWeek(ctx, tx, out.Today); err != nil {
		return out, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT "+projection(models["connection"], true)+",revision FROM connections WHERE stage IS NOT NULL ORDER BY array_position($1::text[],stage),name,id", strings.Split(ConnectionStages, "|"))
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		var rev string
		if err = rows.Scan(&raw, &rev); err != nil {
			return out, err
		}
		r, err := scanResult(raw, rev)
		if err != nil {
			return out, err
		}
		out.Connections = append(out.Connections, r.Entry)
		out.Revisions[r.Entry["id"].(string)] = rev
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	return out, tx.Commit()
}

// updateQueued applies set (SQL assignments using $3.. for args) to one
// connection at revision, bumps its revision and returns the saved row.
// A missing connection is ErrNotFound and a stale revision ErrConflict.
func updateQueued(ctx context.Context, tx *sql.Tx, id, revision, set string, args ...any) (Result, error) {
	if !ValidID("connection", id) {
		return Result{}, fmt.Errorf("%w: invalid connection id", ErrInvalid)
	}
	r, err := tx.ExecContext(ctx, "UPDATE connections SET "+set+",revision=gen_random_uuid()::text,updated_at=now() WHERE id=$1 AND revision=$2", append([]any{id, revision}, args...)...)
	if err != nil {
		return Result{}, err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return Result{}, err
	}
	if n != 1 {
		if _, err = readOne(ctx, tx, "connection", id); err != nil {
			return Result{}, err
		}
		return Result{}, ErrConflict
	}
	if err = bump(ctx, tx, "connection"); err != nil {
		return Result{}, err
	}
	return readOne(ctx, tx, "connection", id)
}

// requireQueued returns ErrNotFound for a missing connection and ErrInvalid
// for one that is not in the queue.
func requireQueued(ctx context.Context, tx *sql.Tx, id string) error {
	if !ValidID("connection", id) {
		return fmt.Errorf("%w: invalid connection id", ErrInvalid)
	}
	var queued bool
	err := tx.QueryRowContext(ctx, "SELECT stage IS NOT NULL FROM connections WHERE id=$1", id).Scan(&queued)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err == nil && !queued {
		err = fmt.Errorf("%w: this person is not in the catch-up queue", ErrInvalid)
	}
	return err
}

// SetConnectionStage adds a connection to the queue or moves it to stage, and
// clears any snooze. An empty stage removes a queued connection from the
// queue; removing someone who is not queued is ErrInvalid.
func (s *Store) SetConnectionStage(ctx context.Context, id, revision, stage string) (Result, error) {
	if stage != "" && !ValidStage(stage) {
		return Result{}, fmt.Errorf("%w: stage must be one of %s", ErrInvalid, StageList())
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback()
	if stage == "" {
		if err = requireQueued(ctx, tx, id); err != nil {
			return Result{}, err
		}
	}
	r, err := updateQueued(ctx, tx, id, revision, "stage=$3,snoozed_until=NULL", sql.NullString{String: stage, Valid: stage != ""})
	if err != nil {
		return r, err
	}
	return r, tx.Commit()
}

// MarkConnectionSent records a message to a queued connection: last contact
// becomes today, the person moves to the next stage (leaving the queue after
// the referral ask), any snooze is cleared, and the send counts toward this
// week's target.
func (s *Store) MarkConnectionSent(ctx context.Context, id, revision string, now time.Time) (Result, CatchUpWeek, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, CatchUpWeek{}, err
	}
	defer tx.Rollback()
	if !ValidID("connection", id) {
		return Result{}, CatchUpWeek{}, fmt.Errorf("%w: invalid connection id", ErrInvalid)
	}
	var stage sql.NullString
	err = tx.QueryRowContext(ctx, "SELECT stage FROM connections WHERE id=$1 AND revision=$2 FOR UPDATE", id, revision).Scan(&stage)
	if errors.Is(err, sql.ErrNoRows) {
		if _, err = readOne(ctx, tx, "connection", id); err == nil {
			err = ErrConflict
		}
		return Result{}, CatchUpWeek{}, err
	}
	if err != nil {
		return Result{}, CatchUpWeek{}, err
	}
	if !stage.Valid {
		return Result{}, CatchUpWeek{}, fmt.Errorf("%w: this person is not in the catch-up queue", ErrInvalid)
	}
	today, err := catchUpToday(ctx, tx, now)
	if err != nil {
		return Result{}, CatchUpWeek{}, err
	}
	next := NextStage(stage.String)
	r, err := updateQueued(ctx, tx, id, revision, "stage=$3,snoozed_until=NULL,last_contacted_on=GREATEST(last_contacted_on,$4::date)", sql.NullString{String: next, Valid: next != ""}, today)
	if err != nil {
		return r, CatchUpWeek{}, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO connection_sends(id,connection_id,stage,sent_on) VALUES($1,$2,$3,$4)", uuid.NewString(), id, stage.String, today); err != nil {
		return r, CatchUpWeek{}, err
	}
	week, err := catchUpWeek(ctx, tx, today)
	if err != nil {
		return r, week, err
	}
	return r, week, tx.Commit()
}

// SnoozeConnection hides a queued connection until the given date, which must
// be after today and at most MaxSnoozeDays ahead.
func (s *Store) SnoozeConnection(ctx context.Context, id, revision, until string, now time.Time) (Result, error) {
	if !validDate(until) {
		return Result{}, fmt.Errorf("%w: until must be a YYYY-MM-DD date", ErrInvalid)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback()
	today, err := catchUpToday(ctx, tx, now)
	if err != nil {
		return Result{}, err
	}
	if until <= today || until > scheduler.DateAdd(today, MaxSnoozeDays) {
		return Result{}, fmt.Errorf("%w: until must be after today and at most %d days ahead", ErrInvalid, MaxSnoozeDays)
	}
	if err = requireQueued(ctx, tx, id); err != nil {
		return Result{}, err
	}
	r, err := updateQueued(ctx, tx, id, revision, "snoozed_until=$3", until)
	if err != nil {
		return r, err
	}
	return r, tx.Commit()
}

// SetCatchUpTarget saves the weekly message target and returns this week's
// counter with it.
func (s *Store) SetCatchUpTarget(ctx context.Context, target int, now time.Time) (CatchUpWeek, error) {
	if target < 1 || target > 50 {
		return CatchUpWeek{}, fmt.Errorf("%w: weeklyTarget must be 1-50", ErrInvalid)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return CatchUpWeek{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "INSERT INTO catch_up_settings(id,weekly_target) VALUES(1,$1) ON CONFLICT(id) DO UPDATE SET weekly_target=EXCLUDED.weekly_target,updated_at=now()", target); err != nil {
		return CatchUpWeek{}, err
	}
	today, err := catchUpToday(ctx, tx, now)
	if err != nil {
		return CatchUpWeek{}, err
	}
	week, err := catchUpWeek(ctx, tx, today)
	if err != nil {
		return week, err
	}
	return week, tx.Commit()
}
