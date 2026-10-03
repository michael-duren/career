package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/scheduler"
	"time"
)

type SchedulerGoogleConnection struct {
	AvailabilityRevision int64                      `json:"-"`
	HealthRevision       int64                      `json:"-"`
	AccountID            string                     `json:"-"`
	CalendarID           string                     `json:"calendarId,omitempty"`
	Credentials          []byte                     `json:"-"`
	SelectedCalendars    []string                   `json:"-"`
	Revision             string                     `json:"revision"`
	ReconnectRequired    bool                       `json:"reconnectRequired"`
	Error                string                     `json:"error,omitempty"`
	LastRefresh          *time.Time                 `json:"lastRefresh,omitempty"`
	BusyFrom, BusyTo     *time.Time                 `json:"-"`
	Busy                 []scheduler.GoogleInterval `json:"-"`
}

func (s *Store) SchedulerGoogle(ctx context.Context) (SchedulerGoogleConnection, error) {
	var c SchedulerGoogleConnection
	var selected, busy []byte
	err := s.DB.QueryRowContext(ctx, `SELECT account_id,calendar_id,credentials,selected_calendars,revision,reconnect_required,last_error,refreshed_at,busy_from,busy_to,busy,health_revision,availability_revision FROM scheduler_google WHERE id=1`).Scan(&c.AccountID, &c.CalendarID, &c.Credentials, &selected, &c.Revision, &c.ReconnectRequired, &c.Error, &c.LastRefresh, &c.BusyFrom, &c.BusyTo, &busy, &c.HealthRevision, &c.AvailabilityRevision)
	if errors.Is(err, sql.ErrNoRows) {
		c.Revision = "0"
		c.SelectedCalendars = []string{}
		c.Busy = []scheduler.GoogleInterval{}
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err = json.Unmarshal(selected, &c.SelectedCalendars); err != nil {
		return c, err
	}
	err = json.Unmarshal(busy, &c.Busy)
	return c, err
}
func (s *Store) SaveSchedulerGoogle(ctx context.Context, c SchedulerGoogleConnection, revision string) (SchedulerGoogleConnection, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return c, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(724193620)`); err != nil {
		return c, err
	}
	var current string
	err = tx.QueryRowContext(ctx, `SELECT revision FROM scheduler_google WHERE id=1`).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		current = "0"
	} else if err != nil {
		return c, err
	}
	if current != revision {
		return c, ErrConflict
	}
	c.Revision = uuid.NewString()
	if c.SelectedCalendars == nil {
		c.SelectedCalendars = []string{}
	}
	if c.Busy == nil {
		c.Busy = []scheduler.GoogleInterval{}
	}
	selected, _ := json.Marshal(c.SelectedCalendars)
	busy, _ := json.Marshal(c.Busy)
	_, err = tx.ExecContext(ctx, `INSERT INTO scheduler_google(id,account_id,calendar_id,credentials,selected_calendars,revision,reconnect_required,last_error,refreshed_at,busy_from,busy_to,busy) VALUES(1,$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT(id) DO UPDATE SET account_id=$1,calendar_id=$2,credentials=$3,selected_calendars=$4,revision=$5,reconnect_required=$6,last_error=$7,refreshed_at=$8,busy_from=$9,busy_to=$10,busy=$11`, c.AccountID, c.CalendarID, c.Credentials, selected, c.Revision, c.ReconnectRequired, c.Error, c.LastRefresh, c.BusyFrom, c.BusyTo, busy)
	if err != nil {
		return c, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO scheduler_google_outbox(id) VALUES(1) ON CONFLICT(id) DO UPDATE SET generation=scheduler_google_outbox.generation+1,requested_at=now()`)
	if err != nil {
		return c, err
	}
	if c.AccountID != "" && c.CalendarID != "" {
		if _, err = tx.ExecContext(ctx, `INSERT INTO scheduler_google_destinations(account_id,calendar_id) VALUES($1,$2) ON CONFLICT(account_id) DO UPDATE SET calendar_id=$2`, c.AccountID, c.CalendarID); err != nil {
			return c, err
		}
	}
	return c, tx.Commit()
}
func (s *Store) StoreSchedulerOAuth(ctx context.Context, state, session, verifier string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM scheduler_google_oauth WHERE expires_at<now()`)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO scheduler_google_oauth(state_hash,session_hash,verifier,expires_at) VALUES($1,$2,$3,now()+interval '10 minutes')`, state, session, verifier)
	return err
}
func (s *Store) ConsumeSchedulerOAuth(ctx context.Context, state, session string) (string, error) {
	var verifier string
	err := s.DB.QueryRowContext(ctx, `DELETE FROM scheduler_google_oauth WHERE state_hash=$1 AND session_hash=$2 AND expires_at>now() RETURNING verifier`, state, session).Scan(&verifier)
	return verifier, err
}

type SchedulerGoogleMapping struct {
	AccountID, SessionID, CalendarID, EventID string
	PlannedStart                              time.Time
	DesiredFingerprint, SyncedFingerprint     string
	DesiredGeneration                         int64
}

func (s *Store) SchedulerGoogleMappings(ctx context.Context, account string) ([]SchedulerGoogleMapping, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT account_id,session_id,calendar_id,event_id,planned_start,desired_fingerprint,synced_fingerprint,desired_generation FROM scheduler_google_mappings WHERE account_id=$1`, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []SchedulerGoogleMapping{}
	for rows.Next() {
		var m SchedulerGoogleMapping
		if err = rows.Scan(&m.AccountID, &m.SessionID, &m.CalendarID, &m.EventID, &m.PlannedStart, &m.DesiredFingerprint, &m.SyncedFingerprint, &m.DesiredGeneration); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}
func (s *Store) SaveSchedulerGoogleMapping(ctx context.Context, m SchedulerGoogleMapping) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO scheduler_google_mappings(account_id,session_id,calendar_id,event_id,planned_start) VALUES($1,$2,$3,$4,$5) ON CONFLICT(account_id,session_id) DO UPDATE SET calendar_id=$3,event_id=$4,planned_start=$5`, m.AccountID, m.SessionID, m.CalendarID, m.EventID, m.PlannedStart)
	return err
}
func (s *Store) DeleteSchedulerGoogleMapping(ctx context.Context, account, session string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM scheduler_google_mappings WHERE account_id=$1 AND session_id=$2`, account, session)
	return err
}
func (s *Store) SchedulerGoogleOutbox(ctx context.Context) (int64, error) {
	var generation int64
	err := s.DB.QueryRowContext(ctx, `SELECT generation FROM scheduler_google_outbox WHERE id=1`).Scan(&generation)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return generation, err
}
func (s *Store) CompleteSchedulerGoogleOutbox(ctx context.Context, generation int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM scheduler_google_outbox WHERE id=1 AND generation=$1`, generation)
	return err
}

func (s *Store) UpdateSchedulerGoogleHealth(ctx context.Context, c *SchedulerGoogleConnection) error {
	busy, _ := json.Marshal(c.Busy)
	if c.Busy == nil {
		busy = []byte("[]")
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE scheduler_google SET credentials=$1,reconnect_required=$2,last_error=$3,refreshed_at=$4,busy_from=$5,busy_to=$6,busy=$7,health_revision=health_revision+1 WHERE id=1 AND revision=$8 AND health_revision=$9`, c.Credentials, c.ReconnectRequired, c.Error, c.LastRefresh, c.BusyFrom, c.BusyTo, busy, c.Revision, c.HealthRevision)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return ErrConflict
	}
	if err == nil {
		c.HealthRevision++
	}
	return err
}

func (s *Store) SchedulerGoogleDestination(ctx context.Context, account string) (string, error) {
	var id string
	err := s.DB.QueryRowContext(ctx, `SELECT calendar_id FROM scheduler_google_destinations WHERE account_id=$1`, account).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// PrepareSchedulerGoogleMapping records desired work before contacting Google.
// Identity changes invalidate prior successful progress; plan changes retain it
// so an unchanged retry can compare the exact successful payload.
func (s *Store) PrepareSchedulerGoogleMapping(ctx context.Context, m SchedulerGoogleMapping) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO scheduler_google_mappings(account_id,session_id,calendar_id,event_id,planned_start,desired_fingerprint,desired_generation) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(account_id,session_id) DO UPDATE SET calendar_id=$3,event_id=$4,planned_start=$5,desired_fingerprint=$6,desired_generation=$7,synced_fingerprint=CASE WHEN scheduler_google_mappings.calendar_id=$3 AND scheduler_google_mappings.event_id=$4 THEN scheduler_google_mappings.synced_fingerprint ELSE '' END`, m.AccountID, m.SessionID, m.CalendarID, m.EventID, m.PlannedStart, m.DesiredFingerprint, m.DesiredGeneration)
	return err
}
func (s *Store) CompleteSchedulerGoogleMapping(ctx context.Context, m SchedulerGoogleMapping) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE scheduler_google_mappings SET synced_fingerprint=$1 WHERE account_id=$2 AND session_id=$3 AND calendar_id=$4 AND event_id=$5 AND desired_fingerprint=$1 AND desired_generation=$6`, m.DesiredFingerprint, m.AccountID, m.SessionID, m.CalendarID, m.EventID, m.DesiredGeneration)
	return err
}
func (s *Store) SchedulerGoogleReconciliationCursor(ctx context.Context, account, calendar string) (string, error) {
	var cursor string
	err := s.DB.QueryRowContext(ctx, `SELECT after_session FROM scheduler_google_reconciliation WHERE account_id=$1 AND calendar_id=$2`, account, calendar).Scan(&cursor)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return cursor, err
}
func (s *Store) SaveSchedulerGoogleReconciliationCursor(ctx context.Context, account, calendar, cursor string) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO scheduler_google_reconciliation(account_id,calendar_id,after_session) VALUES($1,$2,$3) ON CONFLICT(account_id,calendar_id) DO UPDATE SET after_session=$3`, account, calendar, cursor)
	return err
}

// UpdateSchedulerGoogleAvailability only replaces availability. Export health
// writes and token refreshes cannot masquerade as a completed FreeBusy fetch.
func (s *Store) UpdateSchedulerGoogleAvailability(ctx context.Context, c *SchedulerGoogleConnection) error {
	busy, err := json.Marshal(c.Busy)
	if err != nil {
		return err
	}
	if c.Busy == nil {
		busy = []byte("[]")
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE scheduler_google SET busy=$1,busy_from=$2,busy_to=$3,refreshed_at=$4,last_error='',availability_revision=availability_revision+1,health_revision=health_revision+1 WHERE id=1 AND revision=$5 AND health_revision=$6`, busy, c.BusyFrom, c.BusyTo, c.LastRefresh, c.Revision, c.HealthRevision)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return ErrConflict
	}
	if err == nil {
		c.HealthRevision++
		c.AvailabilityRevision++
	}
	return err
}

// RecordSchedulerGoogleError preserves newer availability and credential writes.
// Only the same connection and credential generation can be marked revoked.
func (s *Store) RecordSchedulerGoogleError(ctx context.Context, c *SchedulerGoogleConnection) error {
	result, err := s.DB.ExecContext(ctx, `UPDATE scheduler_google SET last_error=$1,reconnect_required=reconnect_required OR $2,health_revision=health_revision+1 WHERE id=1 AND revision=$3 AND credentials=$4`, c.Error, c.ReconnectRequired, c.Revision, c.Credentials)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return ErrConflict
	}
	return err
}

func (s *Store) SchedulerGooglePendingCursor(ctx context.Context, account, calendar string) (string, error) {
	var cursor string
	err := s.DB.QueryRowContext(ctx, `SELECT pending_after_session FROM scheduler_google_reconciliation WHERE account_id=$1 AND calendar_id=$2`, account, calendar).Scan(&cursor)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return cursor, err
}
func (s *Store) SaveSchedulerGooglePendingCursor(ctx context.Context, account, calendar, cursor string) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO scheduler_google_reconciliation(account_id,calendar_id,pending_after_session) VALUES($1,$2,$3) ON CONFLICT(account_id,calendar_id) DO UPDATE SET pending_after_session=$3`, account, calendar, cursor)
	return err
}

// Reconciliation invalidates successful progress before a forced write, so a
// crash or provider failure becomes pending ordinary work. This short local
// transaction queues work without including any provider request.
func (s *Store) QueueSchedulerGoogleMapping(ctx context.Context, m SchedulerGoogleMapping) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE scheduler_google_mappings SET synced_fingerprint='' WHERE account_id=$1 AND session_id=$2 AND calendar_id=$3 AND event_id=$4`, m.AccountID, m.SessionID, m.CalendarID, m.EventID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO scheduler_google_outbox(id) VALUES(1) ON CONFLICT(id) DO NOTHING`); err != nil {
		return err
	}
	return tx.Commit()
}
