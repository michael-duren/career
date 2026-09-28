CREATE TABLE scheduler_google (
 id integer PRIMARY KEY CHECK (id = 1),
 account_id text NOT NULL DEFAULT '',
 calendar_id text NOT NULL DEFAULT '',
 credentials bytea,
 selected_calendars jsonb NOT NULL DEFAULT '[]',
 revision text NOT NULL,
 health_revision bigint NOT NULL DEFAULT 0,
 reconnect_required boolean NOT NULL DEFAULT false,
 last_error text NOT NULL DEFAULT '',
 refreshed_at timestamptz,
 busy_from timestamptz,
 busy_to timestamptz,
 busy jsonb NOT NULL DEFAULT '[]'
);
CREATE TABLE scheduler_google_oauth (
 state_hash text PRIMARY KEY,
 session_hash text NOT NULL,
 verifier text NOT NULL,
 expires_at timestamptz NOT NULL
);
CREATE TABLE scheduler_google_mappings (
 account_id text NOT NULL,
 session_id text NOT NULL,
 calendar_id text NOT NULL,
 event_id text NOT NULL,
 planned_start timestamptz NOT NULL,
 PRIMARY KEY(account_id, session_id)
);
CREATE TABLE scheduler_google_outbox (
 id integer PRIMARY KEY CHECK(id = 1),
 generation bigint NOT NULL DEFAULT 1,
 requested_at timestamptz NOT NULL DEFAULT now()
);
CREATE FUNCTION queue_scheduler_google() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 INSERT INTO scheduler_google_outbox(id) VALUES(1)
 ON CONFLICT(id) DO UPDATE SET generation=scheduler_google_outbox.generation+1,requested_at=now();
 RETURN NEW;
END;
$$;
CREATE TRIGGER scheduler_google_dirty AFTER INSERT OR UPDATE ON scheduler_state
FOR EACH ROW EXECUTE FUNCTION queue_scheduler_google();
CREATE TABLE scheduler_google_destinations (
 account_id text PRIMARY KEY,
 calendar_id text NOT NULL
);
