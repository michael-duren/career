-- Leetgrinder tracks any LeetCode problem. The catalog caches each problem's
-- metadata and optimal complexity. The curated rows for the retired
-- curriculum, and a bare row for every attempted slug, are seeded in Go
-- right after migrations (see SeedLeetgrinderProblems).
CREATE TABLE leetgrinder_problems (
    slug TEXT PRIMARY KEY CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(slug) <= 100),
    number INTEGER CHECK (number BETWEEN 1 AND 100000),
    title TEXT NOT NULL DEFAULT '' CHECK (char_length(title) <= 200),
    difficulty TEXT NOT NULL DEFAULT '' CHECK (difficulty IN ('', 'Easy', 'Medium', 'Hard')),
    topics TEXT[] NOT NULL DEFAULT '{}' CHECK (cardinality(topics) <= 20),
    optimal_time TEXT NOT NULL DEFAULT '' CHECK (char_length(optimal_time) <= 40),
    optimal_space TEXT NOT NULL DEFAULT '' CHECK (char_length(optimal_space) <= 40),
    optimal_note TEXT NOT NULL DEFAULT '' CHECK (char_length(optimal_note) <= 400),
    optimal_source TEXT NOT NULL DEFAULT '' CHECK (optimal_source IN ('', 'curated', 'model')),
    metadata_source TEXT NOT NULL DEFAULT '' CHECK (metadata_source IN ('', 'seed', 'extension', 'leetcode')),
    -- Set when LeetCode answered, with or without the problem; a fetched row
    -- with no title is a slug LeetCode does not know.
    fetched_at TIMESTAMPTZ,
    fetch_attempts SMALLINT NOT NULL DEFAULT 0 CHECK (fetch_attempts >= 0),
    -- The earliest time the server may ask LeetCode again.
    fetch_after TIMESTAMPTZ NOT NULL DEFAULT '-infinity',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
