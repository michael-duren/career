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

const leetgrinderAttemptColumns = "id,problem_slug,outcome,minutes,assisted,notes,created_at,revision,source,is_review,time_complexity,space_complexity,code,code_language"

func scanLeetgrinderAttempt(row interface{ Scan(...any) error }) (leetgrinder.Attempt, error) {
	var a leetgrinder.Attempt
	err := row.Scan(&a.ID, &a.ProblemSlug, &a.Outcome, &a.Minutes, &a.Assisted, &a.Notes, &a.CreatedAt, &a.Revision, &a.Source, &a.IsReview, &a.TimeComplexity, &a.SpaceComplexity, &a.Code, &a.CodeLanguage)
	return a, err
}

func (s *Store) LeetgrinderState(ctx context.Context) (leetgrinder.State, error) {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return leetgrinder.State{Attempts: []leetgrinder.Attempt{}, CompletedDays: []int{}}, err
	}
	defer tx.Rollback()
	state, err := loadLeetgrinderState(ctx, tx)
	if err != nil {
		return state, err
	}
	return state, tx.Commit()
}

func loadLeetgrinderState(ctx context.Context, tx queryer) (leetgrinder.State, error) {
	state := leetgrinder.State{Attempts: []leetgrinder.Attempt{}, CompletedDays: []int{}}
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
	defer rows.Close()
	for rows.Next() {
		var day int
		if err = rows.Scan(&day); err != nil {
			return state, err
		}
		state.CompletedDays = append(state.CompletedDays, day)
	}
	return state, rows.Err()
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
	if a.NormalizeDetails() != nil {
		return leetgrinder.Attempt{}, ErrInvalid
	}
	if a.Source == "" {
		a.Source = "web"
	}
	if a.Source != "web" && a.Source != "extension" {
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
		_, err = tx.ExecContext(ctx, `INSERT INTO leetgrinder_attempts(id,problem_slug,outcome,minutes,assisted,notes,revision,source,is_review,time_complexity,space_complexity,code,code_language) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT(id) DO NOTHING`, a.ID, a.ProblemSlug, a.Outcome, a.Minutes, a.Assisted, a.Notes, uuid.NewString(), a.Source, a.IsReview, a.TimeComplexity, a.SpaceComplexity, a.Code, a.CodeLanguage)
		if err != nil {
			return leetgrinder.Attempt{}, err
		}
		saved, readErr := scanLeetgrinderAttempt(tx.QueryRowContext(ctx, "SELECT "+leetgrinderAttemptColumns+" FROM leetgrinder_attempts WHERE id=$1 FOR UPDATE", a.ID))
		if readErr != nil {
			return leetgrinder.Attempt{}, readErr
		}
		if saved.ProblemSlug != a.ProblemSlug || saved.Outcome != a.Outcome || saved.Minutes != a.Minutes || saved.Assisted != a.Assisted || saved.Notes != a.Notes || saved.Source != a.Source || saved.IsReview != a.IsReview ||
			saved.TimeComplexity != a.TimeComplexity || saved.SpaceComplexity != a.SpaceComplexity || saved.Code != a.Code || saved.CodeLanguage != a.CodeLanguage {
			return leetgrinder.Attempt{}, ErrConflict
		}
		return saved, tx.Commit()
	}
	// Corrections change only what the learner states; source, review flag,
	// and captured code stay as first recorded.
	saved, err := scanLeetgrinderAttempt(tx.QueryRowContext(ctx, `UPDATE leetgrinder_attempts SET outcome=$3,minutes=$4,assisted=$5,notes=$6,revision=$7,time_complexity=$9,space_complexity=$10 WHERE id=$1 AND revision=$2 AND problem_slug=$8 RETURNING `+leetgrinderAttemptColumns, a.ID, expectedRevision, a.Outcome, a.Minutes, a.Assisted, a.Notes, uuid.NewString(), a.ProblemSlug, a.TimeComplexity, a.SpaceComplexity))
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
