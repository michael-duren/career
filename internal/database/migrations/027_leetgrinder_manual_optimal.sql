-- A learner can correct a problem's optimal complexity, which is stored with
-- optimal_source = 'manual'. Model estimates only ever fill an empty optimum
-- (see saveLeetgrinderModelOptimal), so manual and curated values are kept.
ALTER TABLE leetgrinder_problems DROP CONSTRAINT leetgrinder_problems_optimal_source_check;
ALTER TABLE leetgrinder_problems ADD CONSTRAINT leetgrinder_problems_optimal_source_check
    CHECK (optimal_source IN ('', 'curated', 'model', 'manual'));
