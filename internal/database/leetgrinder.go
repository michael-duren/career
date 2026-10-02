package database

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

const leetgrinderAttemptColumns = "id,problem_slug,outcome,minutes,assisted,notes,created_at,revision,source,is_review,time_complexity,space_complexity,code,code_language,wants_review,approach,marked_at"

func scanLeetgrinderAttempt(row interface{ Scan(...any) error }) (leetgrinder.Attempt, error) {
	var a leetgrinder.Attempt
	err := row.Scan(&a.ID, &a.ProblemSlug, &a.Outcome, &a.Minutes, &a.Assisted, &a.Notes, &a.CreatedAt, &a.Revision, &a.Source, &a.IsReview, &a.TimeComplexity, &a.SpaceComplexity, &a.Code, &a.CodeLanguage, &a.WantsReview, &a.Approach, &a.MarkedAt)
	return a, err
}

func (s *Store) LeetgrinderState(ctx context.Context) (leetgrinder.State, error) {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return leetgrinder.State{Attempts: []leetgrinder.Attempt{}}, err
	}
	defer tx.Rollback()
	state, err := loadLeetgrinderState(ctx, tx, true)
	if err != nil {
		return state, err
	}
	return state, tx.Commit()
}

// LeetgrinderStateWithoutCode is LeetgrinderState with every attempt's captured
// code left empty, for pages that never show it.
func (s *Store) LeetgrinderStateWithoutCode(ctx context.Context) (leetgrinder.State, error) {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return leetgrinder.State{Attempts: []leetgrinder.Attempt{}}, err
	}
	defer tx.Rollback()
	state, err := loadLeetgrinderState(ctx, tx, false)
	if err != nil {
		return state, err
	}
	return state, tx.Commit()
}

// LeetgrinderProblemState is LeetgrinderState with captured code kept only on
// slug's attempts, which is all a problem page shows.
func (s *Store) LeetgrinderProblemState(ctx context.Context, slug string) (leetgrinder.State, error) {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return leetgrinder.State{Attempts: []leetgrinder.Attempt{}}, err
	}
	defer tx.Rollback()
	state, err := loadLeetgrinderState(ctx, tx, false)
	if err != nil {
		return state, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT id,code FROM leetgrinder_attempts WHERE problem_slug=$1 AND code<>''", slug)
	if err != nil {
		return state, err
	}
	code := map[string]string{}
	for rows.Next() {
		var id, c string
		if err = rows.Scan(&id, &c); err != nil {
			rows.Close()
			return state, err
		}
		code[id] = c
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return state, err
	}
	for i := range state.Attempts {
		if a := &state.Attempts[i]; a.ProblemSlug == slug {
			a.Code = code[a.ID]
		}
	}
	return state, tx.Commit()
}

// loadLeetgrinderState reads everything; withCode false leaves captured code
// out, for paths that only count and schedule attempts.
func loadLeetgrinderState(ctx context.Context, tx queryer, withCode bool) (leetgrinder.State, error) {
	state := leetgrinder.State{Attempts: []leetgrinder.Attempt{}}
	columns := leetgrinderAttemptColumns
	if !withCode {
		columns = strings.Replace(columns, ",code,", ",''::text,", 1)
	}
	rows, err := tx.QueryContext(ctx, "SELECT "+columns+" FROM leetgrinder_attempts ORDER BY created_at DESC,id")
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
	if state.Analyses, err = loadLeetgrinderAnalyses(ctx, tx); err != nil {
		return state, err
	}
	if state.Problems, err = loadLeetgrinderProblems(ctx, tx); err != nil {
		return state, err
	}
	if state.Plans, err = loadLeetgrinderPlans(ctx, tx); err != nil {
		return state, err
	}
	state.Goals, err = loadLeetgrinderGoals(ctx, tx)
	return state, err
}

func loadLeetgrinderPlans(ctx context.Context, tx queryer) (map[time.Time][]string, error) {
	rows, err := tx.QueryContext(ctx, "SELECT plan_date,problem_slug FROM leetgrinder_review_plan ORDER BY plan_date,slot")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	plans := map[time.Time][]string{}
	for rows.Next() {
		var date time.Time
		var slug string
		if err = rows.Scan(&date, &slug); err != nil {
			return nil, err
		}
		d := leetgrinder.Date(date, time.UTC)
		plans[d] = append(plans[d], slug)
	}
	return plans, rows.Err()
}

func loadLeetgrinderGoals(ctx context.Context, tx queryer) (map[time.Time]leetgrinder.DailyGoal, error) {
	rows, err := tx.QueryContext(ctx, "SELECT local_date,goal_new,goal_review FROM leetgrinder_daily_goal")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	goals := map[time.Time]leetgrinder.DailyGoal{}
	for rows.Next() {
		var date time.Time
		var g leetgrinder.DailyGoal
		if err = rows.Scan(&date, &g.New, &g.Review); err != nil {
			return nil, err
		}
		goals[leetgrinder.Date(date, time.UTC)] = g
	}
	return goals, rows.Err()
}

func (s *Store) SaveLeetgrinderAttempt(ctx context.Context, a leetgrinder.Attempt, expectedRevision string) (leetgrinder.Attempt, error) {
	return s.SaveLeetgrinderAttemptWithProblem(ctx, a, expectedRevision, nil, time.Time{})
}

// SaveLeetgrinderAttemptWithProblem saves an attempt, adds its problem to the
// catalog, and, when meta is set, stores the problem's LeetCode metadata in
// the same transaction.
func (s *Store) SaveLeetgrinderAttemptWithProblem(ctx context.Context, a leetgrinder.Attempt, expectedRevision string, meta *leetgrinder.ProblemMetadata, now time.Time) (leetgrinder.Attempt, error) {
	id, err := uuid.Parse(a.ID)
	if err != nil || !leetgrinder.ValidSlug(a.ProblemSlug) || a.Minutes < 1 || a.Minutes > 240 || !utf8.ValidString(a.Notes) || strings.ContainsRune(a.Notes, 0) || utf8.RuneCountInString(a.Notes) > 2000 {
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
	if meta != nil && (meta.Normalize() != nil || meta.Title == "") {
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
	if err = ensureLeetgrinderProblem(ctx, tx, a.ProblemSlug); err != nil {
		return leetgrinder.Attempt{}, err
	}
	if meta != nil {
		if err = upsertLeetgrinderMetadata(ctx, tx, a.ProblemSlug, *meta, "extension", now); err != nil {
			return leetgrinder.Attempt{}, err
		}
	}
	if expectedRevision == "" {
		// is_review is decided here: the problem has an attempt on an earlier
		// local day in the settings time zone.
		_, err = tx.ExecContext(ctx, `INSERT INTO leetgrinder_attempts(id,problem_slug,outcome,minutes,assisted,notes,revision,source,is_review,time_complexity,space_complexity,code,code_language,wants_review,approach)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,EXISTS (SELECT 1 FROM leetgrinder_attempts b, leetgrinder_settings s WHERE s.id=1 AND b.problem_slug=$2
	AND (b.created_at AT TIME ZONE s.timezone)::date < (clock_timestamp() AT TIME ZONE s.timezone)::date),$9,$10,$11,$12,$13,$14) ON CONFLICT(id) DO NOTHING`,
			a.ID, a.ProblemSlug, a.Outcome, a.Minutes, a.Assisted, a.Notes, uuid.NewString(), a.Source, a.TimeComplexity, a.SpaceComplexity, a.Code, a.CodeLanguage, a.WantsReview, a.Approach)
		if err != nil {
			return leetgrinder.Attempt{}, err
		}
		saved, readErr := scanLeetgrinderAttempt(tx.QueryRowContext(ctx, "SELECT "+leetgrinderAttemptColumns+" FROM leetgrinder_attempts WHERE id=$1 FOR UPDATE", a.ID))
		if readErr != nil {
			return leetgrinder.Attempt{}, readErr
		}
		if saved.ProblemSlug != a.ProblemSlug || saved.Outcome != a.Outcome || saved.Minutes != a.Minutes || saved.Assisted != a.Assisted || saved.Notes != a.Notes || saved.Source != a.Source ||
			saved.TimeComplexity != a.TimeComplexity || saved.SpaceComplexity != a.SpaceComplexity || saved.Code != a.Code || saved.CodeLanguage != a.CodeLanguage ||
			saved.WantsReview != a.WantsReview || saved.Approach != a.Approach {
			return leetgrinder.Attempt{}, ErrConflict
		}
		return saved, tx.Commit()
	}
	// Corrections change only what the learner states; source, review flag,
	// and captured code stay as first recorded. marked_at moves to now only
	// when the correction makes the attempt flag its problem (a struggle or
	// a mark, see Attempt.SelfFlagged) and it did not before.
	saved, err := scanLeetgrinderAttempt(tx.QueryRowContext(ctx, `UPDATE leetgrinder_attempts SET outcome=$3,minutes=$4,assisted=$5,notes=$6,revision=$7,time_complexity=$9,space_complexity=$10,wants_review=$11,approach=$12,
	marked_at=CASE WHEN ($3<>'solved' OR $5 OR $11 OR $12='suboptimal') AND NOT (outcome<>'solved' OR assisted OR wants_review OR approach='suboptimal') THEN clock_timestamp() ELSE marked_at END WHERE id=$1 AND revision=$2 AND problem_slug=$8 RETURNING `+leetgrinderAttemptColumns, a.ID, expectedRevision, a.Outcome, a.Minutes, a.Assisted, a.Notes, uuid.NewString(), a.ProblemSlug, a.TimeComplexity, a.SpaceComplexity, a.WantsReview, a.Approach))
	if errors.Is(err, sql.ErrNoRows) {
		return leetgrinder.Attempt{}, ErrConflict
	}
	if err != nil {
		return leetgrinder.Attempt{}, err
	}
	return saved, tx.Commit()
}
