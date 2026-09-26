package database

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

var leetgrinderSlug = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

const leetgrinderAttemptColumns = "id,problem_slug,outcome,minutes,assisted,notes,created_at,revision"

func scanLeetgrinderAttempt(row interface{ Scan(...any) error }) (leetgrinder.Attempt, error) {
	var a leetgrinder.Attempt
	err := row.Scan(&a.ID, &a.ProblemSlug, &a.Outcome, &a.Minutes, &a.Assisted, &a.Notes, &a.CreatedAt, &a.Revision)
	return a, err
}

func (s *Store) LeetgrinderState(ctx context.Context) (leetgrinder.State, error) {
	state := leetgrinder.State{Attempts: []leetgrinder.Attempt{}, CompletedDays: []int{}}
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return state, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, "SELECT "+leetgrinderAttemptColumns+" FROM leetgrinder_attempts ORDER BY created_at DESC,id")
	if err != nil {
		return state, err
	}
	for rows.Next() {
		a, scanErr := scanLeetgrinderAttempt(rows)
		if scanErr != nil {
			rows.Close()
			return state, scanErr
		}
		state.Attempts = append(state.Attempts, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return state, err
	}
	rows, err = tx.QueryContext(ctx, "SELECT day FROM leetgrinder_completed_days ORDER BY day")
	if err != nil {
		return state, err
	}
	for rows.Next() {
		var day int
		if err = rows.Scan(&day); err != nil {
			rows.Close()
			return state, err
		}
		state.CompletedDays = append(state.CompletedDays, day)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return state, err
	}
	return state, tx.Commit()
}

func (s *Store) SaveLeetgrinderAttempt(ctx context.Context, a leetgrinder.Attempt, expectedRevision string) (leetgrinder.Attempt, error) {
	id, err := uuid.Parse(a.ID)
	if err != nil || !leetgrinderSlug.MatchString(a.ProblemSlug) || a.Minutes < 1 || a.Minutes > 240 || !utf8.ValidString(a.Notes) || strings.ContainsRune(a.Notes, 0) || utf8.RuneCountInString(a.Notes) > 2000 {
		return leetgrinder.Attempt{}, ErrInvalid
	}
	switch a.Outcome {
	case "solved", "struggled", "unfinished":
	default:
		return leetgrinder.Attempt{}, ErrInvalid
	}
	a.ID = id.String()
	if expectedRevision != "" {
		if _, err = uuid.Parse(expectedRevision); err != nil {
			return leetgrinder.Attempt{}, ErrInvalid
		}
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return leetgrinder.Attempt{}, err
	}
	defer tx.Rollback()
	if expectedRevision == "" {
		_, err = tx.ExecContext(ctx, `INSERT INTO leetgrinder_attempts(id,problem_slug,outcome,minutes,assisted,notes,revision) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(id) DO NOTHING`, a.ID, a.ProblemSlug, a.Outcome, a.Minutes, a.Assisted, a.Notes, uuid.NewString())
		if err != nil {
			return leetgrinder.Attempt{}, err
		}
		saved, readErr := scanLeetgrinderAttempt(tx.QueryRowContext(ctx, "SELECT "+leetgrinderAttemptColumns+" FROM leetgrinder_attempts WHERE id=$1 FOR UPDATE", a.ID))
		if readErr != nil {
			return leetgrinder.Attempt{}, readErr
		}
		if saved.ProblemSlug != a.ProblemSlug || saved.Outcome != a.Outcome || saved.Minutes != a.Minutes || saved.Assisted != a.Assisted || saved.Notes != a.Notes {
			return leetgrinder.Attempt{}, ErrConflict
		}
		return saved, tx.Commit()
	}
	saved, err := scanLeetgrinderAttempt(tx.QueryRowContext(ctx, `UPDATE leetgrinder_attempts SET outcome=$3,minutes=$4,assisted=$5,notes=$6,revision=$7 WHERE id=$1 AND revision=$2 AND problem_slug=$8 RETURNING `+leetgrinderAttemptColumns, a.ID, expectedRevision, a.Outcome, a.Minutes, a.Assisted, a.Notes, uuid.NewString(), a.ProblemSlug))
	if errors.Is(err, sql.ErrNoRows) {
		return leetgrinder.Attempt{}, ErrConflict
	}
	if err != nil {
		return leetgrinder.Attempt{}, err
	}
	return saved, tx.Commit()
}

func (s *Store) SetLeetgrinderDay(ctx context.Context, day int, completed bool) error {
	if day < 1 || day > 84 {
		return ErrInvalid
	}
	query := "DELETE FROM leetgrinder_completed_days WHERE day=$1"
	if completed {
		query = "INSERT INTO leetgrinder_completed_days(day) VALUES($1) ON CONFLICT(day) DO NOTHING"
	}
	_, err := s.DB.ExecContext(ctx, query, day)
	return err
}
