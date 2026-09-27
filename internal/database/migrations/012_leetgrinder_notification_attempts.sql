-- Counts sends per notification kind and local date so failed sends retry a bounded number of times.
ALTER TABLE leetgrinder_notification_log
    ADD COLUMN attempts SMALLINT NOT NULL DEFAULT 1 CHECK (attempts >= 0);
CREATE INDEX leetgrinder_notification_log_sent_at ON leetgrinder_notification_log (sent_at DESC);
