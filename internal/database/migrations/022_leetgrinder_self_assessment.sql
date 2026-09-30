-- The learner's own assessment of an attempt: whether they want to review
-- the problem again soon, and whether they reached the optimal approach or
-- took a simpler one for time. Existing attempts keep "not stated".
ALTER TABLE leetgrinder_attempts
    ADD COLUMN wants_review BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN approach TEXT NOT NULL DEFAULT '' CHECK (approach IN ('', 'optimal', 'suboptimal'));
