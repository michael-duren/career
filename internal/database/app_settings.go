package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/michael-duren/career-strategy/internal/modules"
)

func scanAppSettings(row interface{ Scan(...any) error }) (modules.Settings, error) {
	var s modules.Settings
	var raw, order []byte
	var completed sql.NullTime
	if err := row.Scan(&raw, &order, &s.Theme, &completed); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return modules.Settings{}.Normalize(), nil
		}
		return s, err
	}
	if err := json.Unmarshal(raw, &s.Modules); err != nil {
		return s, fmt.Errorf("app settings modules: %w", err)
	}
	if err := json.Unmarshal(order, &s.Order); err != nil {
		return s, fmt.Errorf("app settings module order: %w", err)
	}
	if completed.Valid {
		t := completed.Time.UTC()
		s.SetupCompletedAt = &t
	}
	return s.Normalize(), nil
}

// settingsError is a client-facing validation message that matches ErrInvalid.
type settingsError string

func (e settingsError) Error() string        { return string(e) }
func (e settingsError) Is(target error) bool { return target == ErrInvalid }

const appSettingsQuery = "SELECT modules,module_order,theme,setup_completed_at FROM app_settings WHERE id=1"

// AppSettings returns the module switches, order, theme and setup state.
// A new install without a settings row has every module on and setup not
// completed.
func (s *Store) AppSettings(ctx context.Context) (modules.Settings, error) {
	return scanAppSettings(s.DB.QueryRowContext(ctx, appSettingsQuery))
}

// UpdateAppSettings applies p to the current settings and stores the result.
// Validation errors wrap ErrInvalid.
func (s *Store) UpdateAppSettings(ctx context.Context, p modules.Patch) (modules.Settings, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return modules.Settings{}, err
	}
	defer tx.Rollback()
	// Serialize writers even when the row does not exist yet.
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(724193602)"); err != nil {
		return modules.Settings{}, err
	}
	current, err := scanAppSettings(tx.QueryRowContext(ctx, appSettingsQuery))
	if err != nil {
		return modules.Settings{}, err
	}
	next, err := current.Apply(p)
	if err != nil {
		return modules.Settings{}, settingsError(err.Error())
	}
	raw, _ := json.Marshal(next.Modules)
	order, _ := json.Marshal(next.Order)
	var completed any
	if next.SetupCompletedAt != nil {
		completed = next.SetupCompletedAt.UTC().Truncate(time.Microsecond)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO app_settings (id,modules,module_order,theme,setup_completed_at,updated_at) VALUES (1,$1,$2,$3,$4,now())
		ON CONFLICT (id) DO UPDATE SET modules=EXCLUDED.modules,module_order=EXCLUDED.module_order,theme=EXCLUDED.theme,setup_completed_at=EXCLUDED.setup_completed_at,updated_at=now()`, raw, order, next.Theme, completed); err != nil {
		return modules.Settings{}, err
	}
	if err = tx.Commit(); err != nil {
		return modules.Settings{}, err
	}
	return next, nil
}
