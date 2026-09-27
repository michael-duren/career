-- LLM complexity analyses of captured attempt code. Derived data: never exported.
-- code_sha256 hashes the analysed inputs (language, stated time and space, code),
-- so a changed input queues the attempt again.
CREATE TABLE leetgrinder_analyses (
    attempt_id UUID PRIMARY KEY REFERENCES leetgrinder_attempts(id) ON DELETE CASCADE,
    code_sha256 BYTEA NOT NULL CHECK (octet_length(code_sha256) = 32),
    status TEXT NOT NULL CHECK (status IN ('pending', 'done', 'failed')),
    tries INTEGER NOT NULL DEFAULT 0 CHECK (tries >= 0),
    actual_time TEXT NOT NULL DEFAULT '' CHECK (char_length(actual_time) <= 40),
    actual_space TEXT NOT NULL DEFAULT '' CHECK (char_length(actual_space) <= 40),
    time_matches BOOLEAN,
    space_matches BOOLEAN,
    optimal BOOLEAN,
    explanation TEXT NOT NULL DEFAULT '' CHECK (char_length(explanation) <= 2000),
    model TEXT NOT NULL DEFAULT '' CHECK (char_length(model) <= 100),
    error TEXT NOT NULL DEFAULT '' CHECK (char_length(error) <= 300),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Requests sent per local date, for the daily cap.
CREATE TABLE leetgrinder_analysis_usage (
    usage_date DATE PRIMARY KEY,
    requests INTEGER NOT NULL DEFAULT 0 CHECK (requests >= 0)
);

-- Set after an error that would hit every attempt (bad key, billing, model),
-- so every worker holds requests back until it passes.
CREATE TABLE leetgrinder_analysis_pause (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    paused_until TIMESTAMPTZ NOT NULL,
    reason TEXT NOT NULL DEFAULT '' CHECK (char_length(reason) <= 300)
);

-- Analysis is on by default; it also needs ANTHROPIC_API_KEY in the environment.
ALTER TABLE leetgrinder_settings ADD COLUMN analysis_enabled BOOLEAN NOT NULL DEFAULT true;
