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
	HealthRevision    int64                      `json:"-"`
	AccountID         string                     `json:"-"`
	CalendarID        string                     `json:"calendarId,omitempty"`
	Credentials       []byte                     `json:"-"`
	SelectedCalendars []string                   `json:"-"`
	Revision          string                     `json:"revision"`
	ReconnectRequired bool                       `json:"reconnectRequired"`
	Error             string                     `json:"error,omitempty"`
	LastRefresh       *time.Time                 `json:"lastRefresh,omitempty"`
	BusyFrom, BusyTo  *time.Time                 `json:"-"`
	Busy              []scheduler.GoogleInterval `json:"-"`
}

func (s *Store) SchedulerGoogle(ctx context.Context) (SchedulerGoogleConnection, error) {
	var c SchedulerGoogleConnection
	var selected, busy []byte
	err := s.DB.QueryRowContext(ctx, `SELECT account_id,calendar_id,credentials,selected_calendars,revision,reconnect_required,last_error,refreshed_at,busy_from,busy_to,busy,health_revision FROM scheduler_google WHERE id=1`).Scan(&c.AccountID, &c.CalendarID, &c.Credentials, &selected, &c.Revision, &c.ReconnectRequired, &c.Error, &c.LastRefresh, &c.BusyFrom, &c.BusyTo, &busy, &c.HealthRevision)
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
}

func (s *Store) SchedulerGoogleMappings(ctx context.Context, account string) ([]SchedulerGoogleMapping, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT account_id,session_id,calendar_id,event_id,planned_start FROM scheduler_google_mappings WHERE account_id=$1`, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []SchedulerGoogleMapping{}
	for rows.Next() {
		var m SchedulerGoogleMapping
		if err = rows.Scan(&m.AccountID, &m.SessionID, &m.CalendarID, &m.EventID, &m.PlannedStart); err != nil {
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
