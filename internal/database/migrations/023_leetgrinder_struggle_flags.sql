-- Struggles (struggled, unfinished, or solved with help) now flag their
-- problem for review. Date existing struggles' flags from this migration,
-- not from the attempt, so past days keep their goals, kinds and streaks;
-- attempts already flagged by a mark keep that mark's date.
UPDATE leetgrinder_attempts SET marked_at = clock_timestamp()
WHERE (outcome <> 'solved' OR assisted) AND NOT (wants_review OR approach = 'suboptimal');
