-- Optional daily new-problem picks drawn from todos. new_from_todos turns the
-- picks on; leetgrinder_new_plan freezes each date's picks the way
-- leetgrinder_review_plan freezes review picks. set_id names the todo set a
-- pick came from, and is cleared when that set is deleted.
ALTER TABLE leetgrinder_settings ADD COLUMN new_from_todos BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE leetgrinder_new_plan (
    plan_date DATE NOT NULL,
    problem_slug TEXT NOT NULL,
    slot SMALLINT NOT NULL,
    set_id UUID REFERENCES leetgrinder_todo_sets(id) ON DELETE SET NULL,
    PRIMARY KEY (plan_date, slot),
    UNIQUE (plan_date, problem_slug)
);
