ALTER TABLE goals ADD COLUMN status TEXT NOT NULL DEFAULT 'planned' CHECK (status IN ('planned','active','done','dropped'));
UPDATE goals SET status = CASE
 WHEN end_date < CURRENT_DATE AND NOT EXISTS (SELECT 1 FROM goal_steps WHERE goal_id=goals.id AND NOT done) THEN 'done'
 WHEN start_date <= CURRENT_DATE THEN 'active' ELSE 'planned' END;
CREATE TABLE goal_dependencies (
 goal_id UUID REFERENCES goals(id) ON DELETE CASCADE,
 depends_on_id UUID REFERENCES goals(id) ON DELETE CASCADE,
 position INTEGER NOT NULL CHECK(position >= 0),
 PRIMARY KEY(goal_id, depends_on_id), CHECK(goal_id <> depends_on_id)
);
CREATE INDEX goal_dependencies_depends_on_idx ON goal_dependencies(depends_on_id);
