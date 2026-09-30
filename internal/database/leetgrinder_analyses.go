package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

// leetgrinderAnalysisInputHash is the SQL for the SHA-256 of an attempt's
// analysed inputs, for an attempts table aliased "a". None of the joined
// fields can contain a newline except the code, which comes last, so the
// encoding is unambiguous.
const leetgrinderAnalysisInputHash = `sha256(convert_to(a.code_language || chr(10) || a.time_complexity || chr(10) || a.space_complexity || chr(10) || a.code, 'UTF8'))`

// leetgrinderAnalysable matches attempts with code and a stated complexity.
const leetgrinderAnalysable = `a.code <> '' AND (a.time_complexity <> '' OR a.space_complexity <> '')`

// leetgrinderAnalysisLockKey serialises analysis workers across replicas.
const leetgrinderAnalysisLockKey = 724193621

// LeetgrinderAnalysisJob is an attempt waiting for analysis.
type LeetgrinderAnalysisJob struct {
	Attempt leetgrinder.Attempt
	// Hash identifies the analysed inputs; results are stored against it.
	Hash []byte
	// Tries counts earlier requests for the same inputs.
	Tries int
}

const leetgrinderAnalysisColumns = "an.attempt_id,an.status,an.tries,an.actual_time,an.actual_space,an.time_matches,an.space_matches,an.optimal,an.explanation,an.model,an.error,an.updated_at,an.code_sha256=" + leetgrinderAnalysisInputHash

func scanLeetgrinderAnalysis(row interface{ Scan(...any) error }) (leetgrinder.Analysis, error) {
	var a leetgrinder.Analysis
	var timeMatches, spaceMatches, optimal sql.NullBool
	err := row.Scan(&a.AttemptID, &a.Status, &a.Tries, &a.ActualTime, &a.ActualSpace, &timeMatches, &spaceMatches, &optimal, &a.Explanation, &a.Model, &a.Error, &a.UpdatedAt, &a.Current)
	a.TimeMatches, a.SpaceMatches, a.Optimal = nullBool(timeMatches), nullBool(spaceMatches), nullBool(optimal)
	return a, err
}

func nullBool(b sql.NullBool) *bool {
	if !b.Valid {
		return nil
	}
	v := b.Bool
	return &v
}

func loadLeetgrinderAnalyses(ctx context.Context, tx queryer) (map[string]leetgrinder.Analysis, error) {
	rows, err := tx.QueryContext(ctx, "SELECT "+leetgrinderAnalysisColumns+" FROM leetgrinder_analyses an JOIN leetgrinder_attempts a ON a.id=an.attempt_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]leetgrinder.Analysis{}
	for rows.Next() {
		a, err := scanLeetgrinderAnalysis(rows)
		if err != nil {
			return nil, err
		}
		out[a.AttemptID] = a
	}
	return out, rows.Err()
}

// LeetgrinderAnalysisBackoff is how long a failed request for the given
// number of tries waits before the next one.
func LeetgrinderAnalysisBackoff(tries int) time.Duration {
	return time.Duration(tries*tries) * time.Minute
}

// NextLeetgrinderAnalysis returns the newest attempt that needs analysis at
// now: it has code and a stated complexity, and has no analysis for its
// current inputs, or a pending one whose retry backoff has passed. Failed
// analyses wait for a re-analyse request or changed inputs.
func (s *Store) NextLeetgrinderAnalysis(ctx context.Context, now time.Time) (LeetgrinderAnalysisJob, bool, error) {
	var job LeetgrinderAnalysisJob
	var current bool
	a := &job.Attempt
	err := s.DB.QueryRowContext(ctx, `SELECT a.id,a.problem_slug,a.outcome,a.created_at,a.time_complexity,a.space_complexity,a.code,a.code_language,h.hash,COALESCE(an.tries,0),COALESCE(an.code_sha256=h.hash,false)
FROM leetgrinder_attempts a
CROSS JOIN LATERAL (SELECT `+leetgrinderAnalysisInputHash+` AS hash) h
LEFT JOIN leetgrinder_analyses an ON an.attempt_id=a.id
WHERE `+leetgrinderAnalysable+`
  AND (an.attempt_id IS NULL OR an.code_sha256<>h.hash
       OR (an.status='pending' AND an.tries<$2 AND an.updated_at <= $1::timestamptz - make_interval(mins => an.tries*an.tries)))
ORDER BY a.created_at DESC, a.id
LIMIT 1`, now, leetgrinder.AnalysisMaxTries).Scan(&a.ID, &a.ProblemSlug, &a.Outcome, &a.CreatedAt, &a.TimeComplexity, &a.SpaceComplexity, &a.Code, &a.CodeLanguage, &job.Hash, &job.Tries, &current)
	if errors.Is(err, sql.ErrNoRows) {
		return job, false, nil
	}
	if err != nil {
		return job, false, err
	}
	if !current {
		// Tries count requests for the current inputs only.
		job.Tries = 0
	}
	return job, true, nil
}

// FinishLeetgrinderAnalysis stores the outcome of a request for the job's
// inputs. status is done, pending (to retry after the backoff), or failed.
func (s *Store) FinishLeetgrinderAnalysis(ctx context.Context, job LeetgrinderAnalysisJob, status string, tries int, result leetgrinder.AnalysisResult, model, detail string, now time.Time) error {
	switch status {
	case leetgrinder.AnalysisDone, leetgrinder.AnalysisPending, leetgrinder.AnalysisFailed:
	default:
		return ErrInvalid
	}
	var optimal any
	if status == leetgrinder.AnalysisDone {
		optimal = result.Optimal
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO leetgrinder_analyses(attempt_id,code_sha256,status,tries,actual_time,actual_space,time_matches,space_matches,optimal,explanation,model,error,updated_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (attempt_id) DO UPDATE SET code_sha256=EXCLUDED.code_sha256,status=EXCLUDED.status,tries=EXCLUDED.tries,actual_time=EXCLUDED.actual_time,actual_space=EXCLUDED.actual_space,
time_matches=EXCLUDED.time_matches,space_matches=EXCLUDED.space_matches,optimal=EXCLUDED.optimal,explanation=EXCLUDED.explanation,model=EXCLUDED.model,error=EXCLUDED.error,updated_at=EXCLUDED.updated_at`,
		job.Attempt.ID, job.Hash, status, tries, result.ActualTime, result.ActualSpace, result.TimeMatches, result.SpaceMatches, optimal,
		leetgrinder.CleanAnalysisText(result.Explanation, leetgrinder.MaxAnalysisExplanation), leetgrinder.CleanAnalysisText(model, 100), leetgrinder.CleanAnalysisText(detail, leetgrinder.MaxAnalysisError), now)
	if err != nil {
		return err
	}
	if status == leetgrinder.AnalysisDone {
		if err = saveLeetgrinderModelOptimal(ctx, tx, job.Attempt.ProblemSlug, result); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// RequeueLeetgrinderAnalysis resets an attempt's analysis to pending with no
// tries, for its current inputs. It returns ErrNotFound when the attempt does
// not exist for slug or cannot be analysed.
func (s *Store) RequeueLeetgrinderAnalysis(ctx context.Context, slug, attemptID string, now time.Time) error {
	if _, err := uuid.Parse(attemptID); err != nil {
		return ErrNotFound
	}
	res, err := s.DB.ExecContext(ctx, `INSERT INTO leetgrinder_analyses(attempt_id,code_sha256,status,tries,updated_at)
SELECT a.id,`+leetgrinderAnalysisInputHash+`,'pending',0,$3 FROM leetgrinder_attempts a WHERE a.id=$1 AND a.problem_slug=$2 AND `+leetgrinderAnalysable+`
ON CONFLICT (attempt_id) DO UPDATE SET code_sha256=EXCLUDED.code_sha256,status='pending',tries=0,actual_time='',actual_space='',time_matches=NULL,space_matches=NULL,optimal=NULL,explanation='',model='',error='',updated_at=EXCLUDED.updated_at`,
		attemptID, slug, now)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		if err != nil {
			return err
		}
		return ErrNotFound
	}
	return nil
}

// ReserveLeetgrinderAnalysisRequest counts one request against the daily
// limit for the local date. It returns false, and counts nothing, once
// limit requests were already made that day.
func (s *Store) ReserveLeetgrinderAnalysisRequest(ctx context.Context, date time.Time, limit int) (bool, error) {
	if limit <= 0 {
		return false, nil
	}
	var n int
	err := s.DB.QueryRowContext(ctx, `INSERT INTO leetgrinder_analysis_usage(usage_date,requests) VALUES($1,1)
ON CONFLICT (usage_date) DO UPDATE SET requests=leetgrinder_analysis_usage.requests+1 WHERE leetgrinder_analysis_usage.requests < $2
RETURNING requests`, date.Format(time.DateOnly), limit).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// LeetgrinderAnalysisUsage returns the requests made on the local date.
func (s *Store) LeetgrinderAnalysisUsage(ctx context.Context, date time.Time) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, "SELECT requests FROM leetgrinder_analysis_usage WHERE usage_date=$1", date.Format(time.DateOnly)).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return n, err
}

// LeetgrinderAnalysisQueueCounts counts analysable attempts by state.
// Queued includes attempts waiting for a retry.
func (s *Store) LeetgrinderAnalysisQueueCounts(ctx context.Context) (leetgrinder.AnalysisQueue, error) {
	var q leetgrinder.AnalysisQueue
	err := s.DB.QueryRowContext(ctx, `SELECT
count(*) FILTER (WHERE an.attempt_id IS NULL OR an.code_sha256<>h.hash OR an.status='pending'),
count(*) FILTER (WHERE an.code_sha256=h.hash AND an.status='failed'),
count(*) FILTER (WHERE an.code_sha256=h.hash AND an.status='done')
FROM leetgrinder_attempts a
CROSS JOIN LATERAL (SELECT `+leetgrinderAnalysisInputHash+` AS hash) h
LEFT JOIN leetgrinder_analyses an ON an.attempt_id=a.id
WHERE `+leetgrinderAnalysable).Scan(&q.Queued, &q.Failed, &q.Done)
	return q, err
}

// LeetgrinderAnalysisPause returns the stored pause; its zero value means
// requests are not held back.
func (s *Store) LeetgrinderAnalysisPause(ctx context.Context) (leetgrinder.AnalysisPause, error) {
	var p leetgrinder.AnalysisPause
	err := s.DB.QueryRowContext(ctx, "SELECT paused_until,reason FROM leetgrinder_analysis_pause WHERE id=1").Scan(&p.Until, &p.Reason)
	if errors.Is(err, sql.ErrNoRows) {
		return leetgrinder.AnalysisPause{}, nil
	}
	return p, err
}

// PauseLeetgrinderAnalysis holds requests back until until, recording the
// (already redacted) reason.
func (s *Store) PauseLeetgrinderAnalysis(ctx context.Context, until time.Time, reason string) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO leetgrinder_analysis_pause(id,paused_until,reason) VALUES(1,$1,$2)
ON CONFLICT (id) DO UPDATE SET paused_until=EXCLUDED.paused_until,reason=EXCLUDED.reason`, until, leetgrinder.CleanAnalysisText(reason, leetgrinder.MaxAnalysisError))
	return err
}

// WithLeetgrinderAnalysisLock runs fn while holding a session advisory lock,
// so only one analysis request is in flight across server replicas. It
// returns false without running fn when another worker holds the lock.
func (s *Store) WithLeetgrinderAnalysisLock(ctx context.Context, fn func(context.Context) error) (bool, error) {
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	var locked bool
	if err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", leetgrinderAnalysisLockKey).Scan(&locked); err != nil || !locked {
		return false, err
	}
	defer func() {
		unlock, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if _, err := conn.ExecContext(unlock, "SELECT pg_advisory_unlock($1)", leetgrinderAnalysisLockKey); err != nil {
			// A connection that may still hold the lock must not return to the pool.
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		}
	}()
	return true, fn(ctx)
}
