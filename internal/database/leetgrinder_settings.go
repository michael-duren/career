package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

const leetgrinderSettingsColumns = "start_date,timezone,daily_hours::float8,ntfy_url,ntfy_topic,ntfy_token_ciphertext,notifications,analysis_enabled,revision"

func scanLeetgrinderSettings(row interface{ Scan(...any) error }) (leetgrinder.Settings, error) {
	var s leetgrinder.Settings
	var start sql.NullTime
	var notifications []byte
	if err := row.Scan(&start, &s.Timezone, &s.DailyHours, &s.NtfyURL, &s.NtfyTopic, &s.NtfyTokenCiphertext, &notifications, &s.AnalysisEnabled, &s.Revision); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return s, ErrNotFound
		}
		return s, err
	}
	if start.Valid {
		date := leetgrinder.Date(start.Time, time.UTC)
		s.StartDate = &date
	}
	if err := json.Unmarshal(notifications, &s.Notifications); err != nil {
		return s, fmt.Errorf("leetgrinder notification settings: %w", err)
	}
	if s.Notifications == nil {
		s.Notifications = map[string]leetgrinder.NotificationPref{}
	}
	return s, nil
}

func (s *Store) LeetgrinderSettings(ctx context.Context) (leetgrinder.Settings, error) {
	return scanLeetgrinderSettings(s.DB.QueryRowContext(ctx, "SELECT "+leetgrinderSettingsColumns+" FROM leetgrinder_settings WHERE id=1"))
}

// UpdateLeetgrinderSettings applies change to the current settings when
// expectedRevision still matches, validates the result, and stores it with a
// new revision. It returns ErrConflict for a stale revision and wraps
// ErrInvalid for invalid settings. Errors from change are returned as-is.
func (s *Store) UpdateLeetgrinderSettings(ctx context.Context, expectedRevision string, change func(*leetgrinder.Settings) error) (leetgrinder.Settings, error) {
	if _, err := uuid.Parse(expectedRevision); err != nil {
		return leetgrinder.Settings{}, ErrInvalid
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return leetgrinder.Settings{}, err
	}
	defer tx.Rollback()
	current, err := scanLeetgrinderSettings(tx.QueryRowContext(ctx, "SELECT "+leetgrinderSettingsColumns+" FROM leetgrinder_settings WHERE id=1 FOR UPDATE"))
	if err != nil {
		return leetgrinder.Settings{}, err
	}
	if current.Revision != expectedRevision {
		return leetgrinder.Settings{}, ErrConflict
	}
	next := current
	if err = change(&next); err != nil {
		return leetgrinder.Settings{}, err
	}
	if err = next.Validate(); err != nil {
		return leetgrinder.Settings{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	notifications, err := json.Marshal(next.Notifications)
	if err != nil {
		return leetgrinder.Settings{}, err
	}
	var start any
	if next.StartDate != nil {
		start = next.StartDate.Format(time.DateOnly)
	}
	var token any
	if len(next.NtfyTokenCiphertext) > 0 {
		token = next.NtfyTokenCiphertext
	}
	saved, err := scanLeetgrinderSettings(tx.QueryRowContext(ctx, `UPDATE leetgrinder_settings SET start_date=$1,timezone=$2,daily_hours=$3,ntfy_url=$4,ntfy_topic=$5,ntfy_token_ciphertext=$6,notifications=$7,analysis_enabled=$8,revision=$9 WHERE id=1 RETURNING `+leetgrinderSettingsColumns,
		start, next.Timezone, next.DailyHours, next.NtfyURL, next.NtfyTopic, token, notifications, next.AnalysisEnabled, uuid.NewString()))
	if err != nil {
		return leetgrinder.Settings{}, err
	}
	return saved, tx.Commit()
}

// LeetgrinderNtfyTokenStored reports whether an encrypted ntfy token exists.
func (s *Store) LeetgrinderNtfyTokenStored(ctx context.Context) (bool, error) {
	var stored bool
	err := s.DB.QueryRowContext(ctx, "SELECT ntfy_token_ciphertext IS NOT NULL FROM leetgrinder_settings WHERE id=1").Scan(&stored)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return stored, err
}

// LeetgrinderToday loads settings and history and returns today's view for
// now. On first access for a local date, it plans that date's reviews and
// freezes them; later accesses only add picks when more slots opened up.
func (s *Store) LeetgrinderToday(ctx context.Context, now time.Time) (leetgrinder.Today, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return leetgrinder.Today{}, err
	}
	defer tx.Rollback()
	// Serializes planners so concurrent first visits agree on one plan.
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(724193611)"); err != nil {
		return leetgrinder.Today{}, err
	}
	settings, err := scanLeetgrinderSettings(tx.QueryRowContext(ctx, "SELECT "+leetgrinderSettingsColumns+" FROM leetgrinder_settings WHERE id=1"))
	if err != nil {
		return leetgrinder.Today{}, err
	}
	state, err := loadLeetgrinderState(ctx, tx)
	if err != nil {
		return leetgrinder.Today{}, err
	}
	loc := settings.Location()
	date := leetgrinder.Date(now, loc)
	day := date.Format(time.DateOnly)
	rows, err := tx.QueryContext(ctx, "SELECT problem_slug FROM leetgrinder_review_plan WHERE plan_date=$1 ORDER BY slot", day)
	if err != nil {
		return leetgrinder.Today{}, err
	}
	var existing []string
	for rows.Next() {
		var slug string
		if err = rows.Scan(&slug); err != nil {
			rows.Close()
			return leetgrinder.Today{}, err
		}
		existing = append(existing, slug)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return leetgrinder.Today{}, err
	}
	session := 0
	if schedule, ok := settings.Schedule(); ok {
		session = schedule.SessionForDate(date)
	}
	plan := leetgrinder.PlanReviews(leetgrinder.BuildCards(state.Attempts, loc), date, loc, session, settings.DailyHours, existing)
	for slot := len(existing); slot < len(plan); slot++ {
		if _, err = tx.ExecContext(ctx, "INSERT INTO leetgrinder_review_plan(plan_date,problem_slug,slot) VALUES($1,$2,$3)", day, plan[slot], slot+1); err != nil {
			return leetgrinder.Today{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return leetgrinder.Today{}, err
	}
	return leetgrinder.NewToday(settings, state, plan, now), nil
}
