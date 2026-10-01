-- Keys are todo item IDs. Values are arrays of imported source snapshots.
-- Entries remain when a todo is removed, so copied metadata stays with the problem.
ALTER TABLE leetgrinder_problems ADD COLUMN import_metadata JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE leetgrinder_problems DROP CONSTRAINT leetgrinder_problems_metadata_source_check;
ALTER TABLE leetgrinder_problems ADD CONSTRAINT leetgrinder_problems_metadata_source_check
    CHECK (metadata_source IN ('', 'seed', 'extension', 'leetcode', 'mcp'));

CREATE TABLE leetgrinder_todo_sets (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL CHECK (char_length(title) BETWEEN 1 AND 120),
    description TEXT NOT NULL DEFAULT '' CHECK (char_length(description) <= 2000),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (octet_length(metadata::text) <= 16384),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE TABLE leetgrinder_todo_items (
    id UUID PRIMARY KEY,
    set_id UUID REFERENCES leetgrinder_todo_sets(id) ON DELETE CASCADE,
    problem_slug TEXT NOT NULL REFERENCES leetgrinder_problems(slug),
    source_data JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (octet_length(source_data::text) <= 16384),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (set_id, problem_slug)
);
CREATE UNIQUE INDEX leetgrinder_todo_standalone_slug ON leetgrinder_todo_items(problem_slug) WHERE set_id IS NULL;
CREATE INDEX leetgrinder_todo_set_items ON leetgrinder_todo_items(set_id, created_at, id);
