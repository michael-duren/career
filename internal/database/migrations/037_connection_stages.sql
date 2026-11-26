-- Catch-up queue: a connection's stage says where it is in the outreach
-- sequence. NULL means the person is not in the queue. snoozed_until hides a
-- queued person until that date. Existing rows start outside the queue, so
-- the old queued flag and cadences keep working unchanged.
ALTER TABLE connections
    ADD COLUMN stage TEXT CHECK (stage IN ('to_contact', 'messaged', 'follow_up', 'referral_ask')),
    ADD COLUMN snoozed_until DATE,
    ADD CONSTRAINT connections_snooze_needs_stage CHECK (snoozed_until IS NULL OR stage IS NOT NULL);
CREATE INDEX connections_stage_idx ON connections(stage, name) WHERE stage IS NOT NULL;

-- One row per "sent" action, for the weekly message counter. A deleted
-- connection keeps its sends so the week's count does not drop.
CREATE TABLE connection_sends (
    id UUID PRIMARY KEY,
    connection_id UUID REFERENCES connections(id) ON DELETE SET NULL,
    stage TEXT NOT NULL CHECK (stage IN ('to_contact', 'messaged', 'follow_up', 'referral_ask')),
    sent_on DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX connection_sends_sent_on_idx ON connection_sends(sent_on);

-- The weekly message target. A missing row means the default of 2.
CREATE TABLE catch_up_settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    weekly_target SMALLINT NOT NULL DEFAULT 2 CHECK (weekly_target BETWEEN 1 AND 50),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
