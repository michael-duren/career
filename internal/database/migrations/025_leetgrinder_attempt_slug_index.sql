-- Serves the problem_slug lookups: the todo done_at subquery, the is_review
-- EXISTS on insert, NextLeetgrinderFetch and the todo conflict check.
-- Plain CREATE INDEX: the migration runner wraps every file in a transaction,
-- which CREATE INDEX CONCURRENTLY cannot run in.
CREATE INDEX IF NOT EXISTS leetgrinder_attempts_problem_slug_created_at_idx
    ON leetgrinder_attempts (problem_slug, created_at) INCLUDE (outcome);
