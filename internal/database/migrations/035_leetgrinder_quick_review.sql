-- Quick reviews: a short recall check without code. mode is 'full' for an
-- attempt at solving the problem and 'quick' for a quick review, whose
-- outcome is the learner's own grade: recalled, partial or forgot. missed
-- says what a partial quick review got wrong. spot_check marks a recalled
-- quick review picked for a full re-solve at its next review.
ALTER TABLE leetgrinder_attempts
    ADD COLUMN mode TEXT NOT NULL DEFAULT 'full' CHECK (mode IN ('full', 'quick')),
    ADD COLUMN missed TEXT NOT NULL DEFAULT '' CHECK (missed IN ('', 'time', 'space', 'both', 'detail')),
    ADD COLUMN spot_check BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE leetgrinder_attempts DROP CONSTRAINT leetgrinder_attempts_outcome_check;
ALTER TABLE leetgrinder_attempts ADD CONSTRAINT leetgrinder_attempts_outcome_check CHECK (
    (mode = 'full' AND outcome IN ('solved', 'struggled', 'unfinished') AND missed = '' AND NOT spot_check)
    OR (mode = 'quick' AND outcome IN ('recalled', 'partial', 'forgot')
        AND (missed = '' OR outcome = 'partial') AND (NOT spot_check OR outcome = 'recalled')));

-- The share of recalled quick reviews picked for a spot check, in percent.
ALTER TABLE leetgrinder_settings
    ADD COLUMN spot_check_rate SMALLINT NOT NULL DEFAULT 10 CHECK (spot_check_rate BETWEEN 0 AND 25);
