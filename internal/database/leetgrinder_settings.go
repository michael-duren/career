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

const leetgrinderSettingsColumns = "timezone,goal_new,goal_review,ntfy_url,ntfy_topic,ntfy_token_ciphertext,notifications,analysis_enabled,revision"

func scanLeetgrinderSettings(row interface{ Scan(...any) error }) (leetgrinder.Settings, error) {
	var s leetgrinder.Settings
	var notifications []byte
	if err := row.Scan(&s.Timezone, &s.Goal.New, &s.Goal.Review, &s.NtfyURL, &s.NtfyTopic, &s.NtfyTokenCiphertext, &notifications, &s.AnalysisEnabled, &s.Revision); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return s, ErrNotFound
		}
		return s, err
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
	var token any
	if len(next.NtfyTokenCiphertext) > 0 {
		token = next.NtfyTokenCiphertext
	}
	saved, err := scanLeetgrinderSettings(tx.QueryRowContext(ctx, `UPDATE leetgrinder_settings SET timezone=$1,goal_new=$2,goal_review=$3,ntfy_url=$4,ntfy_topic=$5,ntfy_token_ciphertext=$6,notifications=$7,analysis_enabled=$8,revision=$9 WHERE id=1 RETURNING `+leetgrinderSettingsColumns,
		next.Timezone, next.Goal.New, next.Goal.Review, next.NtfyURL, next.NtfyTopic, token, notifications, next.AnalysisEnabled, uuid.NewString()))
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
// now. On first access for a local date it freezes that date's goal from
// settings and plans its review picks; later accesses only add picks when
// the frozen review target has more slots than the plan.
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
	loc := settings.Location()
	date := leetgrinder.Date(now, loc)
	day := date.Format(time.DateOnly)
	if _, err = tx.ExecContext(ctx, "INSERT INTO leetgrinder_daily_goal(local_date,goal_new,goal_review) VALUES($1,$2,$3) ON CONFLICT (local_date) DO NOTHING", day, settings.Goal.New, settings.Goal.Review); err != nil {
		return leetgrinder.Today{}, err
	}
	state, err := loadLeetgrinderState(ctx, tx)
	if err != nil {
		return leetgrinder.Today{}, err
	}
	existing := state.Plans[date]
	plan := leetgrinder.PlanReviews(leetgrinder.BuildCards(state, loc), date, loc, state.GoalFor(date).Review, existing)
	for slot := len(existing); slot < len(plan); slot++ {
		if _, err = tx.ExecContext(ctx, "INSERT INTO leetgrinder_review_plan(plan_date,problem_slug,slot) VALUES($1,$2,$3)", day, plan[slot], slot+1); err != nil {
			return leetgrinder.Today{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return leetgrinder.Today{}, err
	}
	if len(plan) > 0 {
		state.Plans[date] = plan
	}
	return leetgrinder.NewToday(settings, state, now), nil
}

// LeetgrinderDailyGoal returns the goal frozen for a local date, if any.
func (s *Store) LeetgrinderDailyGoal(ctx context.Context, date time.Time) (leetgrinder.DailyGoal, bool, error) {
	var g leetgrinder.DailyGoal
	err := s.DB.QueryRowContext(ctx, "SELECT goal_new,goal_review FROM leetgrinder_daily_goal WHERE local_date=$1", date.Format(time.DateOnly)).Scan(&g.New, &g.Review)
	if errors.Is(err, sql.ErrNoRows) {
		return g, false, nil
	}
	return g, err == nil, err
}
