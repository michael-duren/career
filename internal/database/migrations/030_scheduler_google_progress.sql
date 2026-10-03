ALTER TABLE scheduler_google_mappings
 ADD COLUMN desired_fingerprint text NOT NULL DEFAULT '',
 ADD COLUMN synced_fingerprint text NOT NULL DEFAULT '',
 ADD COLUMN desired_generation bigint NOT NULL DEFAULT 0;

CREATE TABLE scheduler_google_reconciliation (
 account_id text NOT NULL,
 calendar_id text NOT NULL,
 after_session text NOT NULL DEFAULT '',
 PRIMARY KEY (account_id, calendar_id)
);

ALTER TABLE scheduler_google ADD COLUMN availability_revision bigint NOT NULL DEFAULT 0;
