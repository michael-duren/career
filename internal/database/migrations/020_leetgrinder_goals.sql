-- Leetgrinder daily goal: new problems plus reviews each day, replacing the
-- curriculum schedule. Take "Export attempt history" before this migration.
ALTER TABLE leetgrinder_settings
    ADD COLUMN goal_new SMALLINT NOT NULL DEFAULT 2 CHECK (goal_new BETWEEN 0 AND 10),
    ADD COLUMN goal_review SMALLINT NOT NULL DEFAULT 1 CHECK (goal_review BETWEEN 0 AND 10),
    DROP COLUMN start_date,
    DROP COLUMN daily_hours;
-- 0 + 0 is rejected in Go.

-- Each day's targets, frozen on first access that day like the review plan.
-- Days without a row (before this feature) count with the defaults, 2 + 1.
CREATE TABLE leetgrinder_daily_goal (
    local_date DATE PRIMARY KEY,
    goal_new SMALLINT NOT NULL CHECK (goal_new BETWEEN 0 AND 10),
    goal_review SMALLINT NOT NULL CHECK (goal_review BETWEEN 0 AND 10)
);

-- Reminder kinds: missing_work becomes goal_incomplete, late_escalation
-- becomes streak_at_risk (threshold 1 day), behind_schedule is dropped.
-- Old log rows keep their old kind names.
UPDATE leetgrinder_settings SET notifications =
    (notifications - 'missing_work' - 'late_escalation' - 'behind_schedule')
    || CASE WHEN notifications ? 'missing_work'
        THEN jsonb_build_object('goal_incomplete', notifications->'missing_work') ELSE '{}'::jsonb END
    || CASE WHEN notifications ? 'late_escalation'
        THEN jsonb_build_object('streak_at_risk', (notifications->'late_escalation') || '{"threshold": 1}'::jsonb) ELSE '{}'::jsonb END;

DROP TABLE leetgrinder_completed_days;

-- An attempt is a review when its problem has an attempt on an earlier
-- local day, in the settings time zone.
UPDATE leetgrinder_attempts a SET is_review = EXISTS (
    SELECT 1 FROM leetgrinder_attempts b
    WHERE b.problem_slug = a.problem_slug
      AND (b.created_at AT TIME ZONE s.timezone)::date < (a.created_at AT TIME ZONE s.timezone)::date)
FROM leetgrinder_settings s WHERE s.id = 1;
