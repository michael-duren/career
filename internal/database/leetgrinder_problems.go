package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

const leetgrinderProblemColumns = "slug,COALESCE(number,0),title,difficulty,array_to_json(topics)::text,optimal_time,optimal_space,optimal_note,optimal_source,metadata_source,(fetched_at IS NOT NULL AND title=''),fetch_attempts"

func scanLeetgrinderProblem(row interface{ Scan(...any) error }) (leetgrinder.Problem, error) {
	var p leetgrinder.Problem
	var topics string
	if err := row.Scan(&p.Slug, &p.Number, &p.Title, &p.Difficulty, &topics, &p.OptimalTime, &p.OptimalSpace, &p.OptimalNote, &p.OptimalSource, &p.MetadataSource, &p.NotFound, &p.FetchAttempts); err != nil {
		return p, err
	}
	if err := json.Unmarshal([]byte(topics), &p.Topics); err != nil {
		return p, err
	}
	return p, nil
}

func loadLeetgrinderProblems(ctx context.Context, q queryer) (map[string]leetgrinder.Problem, error) {
	rows, err := q.QueryContext(ctx, "SELECT "+leetgrinderProblemColumns+" FROM leetgrinder_problems")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	problems := map[string]leetgrinder.Problem{}
	for rows.Next() {
		p, err := scanLeetgrinderProblem(rows)
		if err != nil {
			return nil, err
		}
		problems[p.Slug] = p
	}
	return problems, rows.Err()
}

// LeetgrinderProblem returns the catalog row for slug, or a bare problem
// when the slug is not in the catalog.
func (s *Store) LeetgrinderProblem(ctx context.Context, slug string) (leetgrinder.Problem, error) {
	p, err := scanLeetgrinderProblem(s.DB.QueryRowContext(ctx, "SELECT "+leetgrinderProblemColumns+" FROM leetgrinder_problems WHERE slug=$1", slug))
	if errors.Is(err, sql.ErrNoRows) {
		return leetgrinder.Problem{Slug: slug}, nil
	}
	return p, err
}

// seedLeetgrinderProblems inserts the curated catalog and a bare row for
// every attempted slug. Existing rows are left alone, so it runs after every
// migration.
func seedLeetgrinderProblems(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO leetgrinder_problems(slug,number,title,difficulty,optimal_time,optimal_space,optimal_note,optimal_source,metadata_source)
SELECT x.slug, NULLIF(x.number,0), x.title, x.difficulty, x."optimalTime", x."optimalSpace", COALESCE(x."optimalNote",''), 'curated', 'seed'
FROM jsonb_to_recordset($1::jsonb) AS x(slug text, number int, title text, difficulty text, "optimalTime" text, "optimalSpace" text, "optimalNote" text)
ON CONFLICT (slug) DO NOTHING`, string(leetgrinder.CatalogSeedJSON())); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO leetgrinder_problems(slug) SELECT DISTINCT problem_slug FROM leetgrinder_attempts
WHERE char_length(problem_slug) <= 100 ON CONFLICT (slug) DO NOTHING`)
	return err
}

// ensureLeetgrinderProblem adds a bare catalog row for slug if it has none.
func ensureLeetgrinderProblem(ctx context.Context, tx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, slug string) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO leetgrinder_problems(slug) VALUES($1) ON CONFLICT (slug) DO NOTHING", slug)
	return err
}

// EnsureLeetgrinderProblem adds a bare catalog row for slug, so the metadata
// fetcher picks it up, and returns the row.
func (s *Store) EnsureLeetgrinderProblem(ctx context.Context, slug string) (leetgrinder.Problem, error) {
	if !leetgrinder.ValidSlug(slug) {
		return leetgrinder.Problem{}, ErrInvalid
	}
	if err := ensureLeetgrinderProblem(ctx, s.DB, slug); err != nil {
		return leetgrinder.Problem{}, err
	}
	return s.LeetgrinderProblem(ctx, slug)
}

// upsertLeetgrinderMetadata stores LeetCode metadata for slug. Empty values
// never replace known ones, and the optimal columns are never touched.
func upsertLeetgrinderMetadata(ctx context.Context, tx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, slug string, m leetgrinder.ProblemMetadata, source string, now time.Time) error {
	var number any
	if m.Number > 0 {
		number = m.Number
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO leetgrinder_problems(slug,number,title,difficulty,topics,metadata_source,fetched_at) VALUES($1,$2,$3,$4,$5::text[],$6,$7)
ON CONFLICT (slug) DO UPDATE SET
	number=COALESCE(EXCLUDED.number,leetgrinder_problems.number),
	title=COALESCE(NULLIF(EXCLUDED.title,''),leetgrinder_problems.title),
	difficulty=COALESCE(NULLIF(EXCLUDED.difficulty,''),leetgrinder_problems.difficulty),
	topics=CASE WHEN cardinality(EXCLUDED.topics)>0 THEN EXCLUDED.topics ELSE leetgrinder_problems.topics END,
	metadata_source=EXCLUDED.metadata_source,
	fetched_at=EXCLUDED.fetched_at`, slug, number, m.Title, m.Difficulty, m.TopicSlugs(), source, now)
	return err
}

// SaveLeetgrinderMetadata validates and stores metadata the extension read
// from LeetCode. A title is required: a fetched row without one means
// LeetCode has no such problem.
func (s *Store) SaveLeetgrinderMetadata(ctx context.Context, slug string, m leetgrinder.ProblemMetadata, now time.Time) error {
	if !leetgrinder.ValidSlug(slug) || m.Normalize() != nil || m.Title == "" {
		return ErrInvalid
	}
	return upsertLeetgrinderMetadata(ctx, s.DB, slug, m, "extension", now)
}

// saveLeetgrinderModelOptimal stores Claude's estimate of a problem's
// optimum, only when the problem has none yet.
func saveLeetgrinderModelOptimal(ctx context.Context, tx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, slug string, r leetgrinder.AnalysisResult) error {
	if r.OptimalTime == "" || r.OptimalSpace == "" {
		return nil
	}
	if err := ensureLeetgrinderProblem(ctx, tx, slug); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE leetgrinder_problems SET optimal_time=$2,optimal_space=$3,optimal_note=$4,optimal_source='model'
WHERE slug=$1 AND optimal_source='' AND optimal_time='' AND optimal_space=''`, slug, r.OptimalTime, r.OptimalSpace, leetgrinder.CleanAnalysisText(r.OptimalNote, leetgrinder.MaxOptimalNote))
	return err
}

// NextLeetgrinderFetch picks one slug whose metadata the server should ask
// LeetCode for: a row with no title, or a seeded row of an attempted problem
// that has no topic tags yet, that LeetCode has not answered for, that is
// under the attempt cap and past its backoff. Newest rows go first.
func (s *Store) NextLeetgrinderFetch(ctx context.Context, now time.Time) (string, bool, error) {
	var slug string
	err := s.DB.QueryRowContext(ctx, `SELECT p.slug FROM leetgrinder_problems p
WHERE p.fetched_at IS NULL AND p.fetch_attempts < $1 AND p.fetch_after <= $2
  AND (p.title='' OR (p.metadata_source='seed' AND cardinality(p.topics)=0 AND EXISTS (SELECT 1 FROM leetgrinder_attempts a WHERE a.problem_slug=p.slug)))
ORDER BY p.title='' DESC, p.created_at DESC, p.slug LIMIT 1`, leetgrinder.MaxFetchAttempts, now).Scan(&slug)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return slug, err == nil, err
}

// FinishLeetgrinderFetch records a fetch. A nil m with found false means
// LeetCode answered that the problem does not exist; err means the fetch
// failed and is retried after 1, 10, then 60 minutes.
func (s *Store) FinishLeetgrinderFetch(ctx context.Context, slug string, m *leetgrinder.ProblemMetadata, fetchErr error, now time.Time) error {
	if fetchErr != nil {
		_, err := s.DB.ExecContext(ctx, `UPDATE leetgrinder_problems SET fetch_attempts=fetch_attempts+1,
fetch_after=$2::timestamptz + CASE fetch_attempts+1 WHEN 1 THEN interval '1 minute' WHEN 2 THEN interval '10 minutes' ELSE interval '60 minutes' END
WHERE slug=$1`, slug, now)
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "UPDATE leetgrinder_problems SET fetch_attempts=fetch_attempts+1, fetched_at=$2 WHERE slug=$1", slug, now); err != nil {
		return err
	}
	if m != nil {
		if m.Normalize() != nil {
			return ErrInvalid
		}
		if err = upsertLeetgrinderMetadata(ctx, tx, slug, *m, "leetcode", now); err != nil {
			return err
		}
	}
	return tx.Commit()
}
