CREATE TABLE bootcamp_attempts (
    id UUID PRIMARY KEY,
    problem_slug TEXT NOT NULL CHECK (problem_slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    outcome TEXT NOT NULL CHECK (outcome IN ('solved', 'struggled', 'unfinished')),
    minutes INTEGER NOT NULL CHECK (minutes BETWEEN 1 AND 240),
    assisted BOOLEAN NOT NULL,
    notes TEXT NOT NULL CHECK (char_length(notes) <= 2000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    revision UUID NOT NULL
);
CREATE INDEX bootcamp_attempts_history ON bootcamp_attempts (created_at DESC, id);

CREATE TABLE bootcamp_completed_days (
    day INTEGER PRIMARY KEY CHECK (day BETWEEN 1 AND 84)
);
