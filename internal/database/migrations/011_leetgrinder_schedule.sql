-- Leetgrinder schedule, spaced repetition, notifications, and extension tokens.
-- Later features share this migration so parallel work never races on numbers.
CREATE TABLE leetgrinder_settings (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    start_date DATE,
    timezone TEXT NOT NULL DEFAULT 'America/Chicago',
    daily_hours NUMERIC(2,1) NOT NULL DEFAULT 2.0 CHECK (daily_hours BETWEEN 2.0 AND 4.0),
    ntfy_url TEXT NOT NULL DEFAULT 'https://ntfy.sh',
    ntfy_topic TEXT NOT NULL DEFAULT '',
    ntfy_token_ciphertext BYTEA,
    notifications JSONB NOT NULL DEFAULT '{}',
    revision UUID NOT NULL
);
INSERT INTO leetgrinder_settings(id, revision) VALUES (1, gen_random_uuid());

CREATE TABLE leetgrinder_api_tokens (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    token_hash BYTEA NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
);

CREATE TABLE leetgrinder_review_plan (
    plan_date DATE NOT NULL,
    problem_slug TEXT NOT NULL,
    slot SMALLINT NOT NULL,
    PRIMARY KEY (plan_date, slot),
    UNIQUE (plan_date, problem_slug)
);

CREATE TABLE leetgrinder_notification_log (
    kind TEXT NOT NULL,
    local_date DATE NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    status TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (kind, local_date)
);

ALTER TABLE leetgrinder_attempts
    ADD COLUMN source TEXT NOT NULL DEFAULT 'web' CHECK (source IN ('web', 'extension')),
    ADD COLUMN is_review BOOLEAN NOT NULL DEFAULT false;
