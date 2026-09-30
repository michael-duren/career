-- The learner's own assessment of an attempt: whether they want to review
-- the problem again soon, and whether they reached the optimal approach or
-- took a simpler one for time. Existing attempts keep "not stated".
-- marked_at is when the first mark was raised, so a mark added by a later
-- correction flags from then on rather than from the attempt's day.
ALTER TABLE leetgrinder_attempts
    ADD COLUMN wants_review BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN approach TEXT NOT NULL DEFAULT '' CHECK (approach IN ('', 'optimal', 'suboptimal')),
    ADD COLUMN marked_at TIMESTAMPTZ;
UPDATE leetgrinder_attempts SET marked_at = created_at;
ALTER TABLE leetgrinder_attempts
    ALTER COLUMN marked_at SET DEFAULT clock_timestamp(),
    ALTER COLUMN marked_at SET NOT NULL;
