CREATE TABLE running_notes (
 id TEXT PRIMARY KEY CHECK(length(id) BETWEEN 1 AND 200 AND id ~ '^[a-zA-Z0-9_/-]+$'),
 title TEXT NOT NULL CHECK(length(title) BETWEEN 1 AND 200), run_date DATE NOT NULL,
 started_at TIMESTAMPTZ NOT NULL, distance_km NUMERIC CHECK(distance_km BETWEEN 0 AND 500),
 duration_min NUMERIC CHECK(duration_min BETWEEN 0 AND 1440), tags TEXT[] NOT NULL CHECK(cardinality(tags)<=30),
 body TEXT NOT NULL CHECK(length(body)<=100000), revision TEXT NOT NULL,
 position BIGINT NOT NULL CHECK(position>=0), updated_at TIMESTAMPTZ
);
CREATE TABLE running_note_clips (
 note_id TEXT NOT NULL REFERENCES running_notes(id) ON DELETE CASCADE,
 id UUID NOT NULL UNIQUE, position INTEGER NOT NULL CHECK(position>=0), recorded_at TIMESTAMPTZ NOT NULL,
 duration_ms INTEGER NOT NULL CHECK(duration_ms BETWEEN 0 AND 1800000), mime_type TEXT NOT NULL,
 audio BYTEA NOT NULL CHECK(octet_length(audio) BETWEEN 1 AND 15728640),
 transcript TEXT CHECK(length(transcript)<=50000),
 transcript_status TEXT NOT NULL CHECK(transcript_status IN ('pending','transcribing','done','failed')),
 transcript_error TEXT, model TEXT, attempts INTEGER NOT NULL DEFAULT 0,
 claimed_at TIMESTAMPTZ, next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 body_merged BOOLEAN NOT NULL DEFAULT false,
 PRIMARY KEY(note_id,id), UNIQUE(note_id,position)
);
CREATE INDEX running_notes_run_date_idx ON running_notes(run_date DESC,id);
CREATE INDEX running_note_clips_pending_idx ON running_note_clips(next_attempt_at) WHERE transcript_status IN ('pending','transcribing');
