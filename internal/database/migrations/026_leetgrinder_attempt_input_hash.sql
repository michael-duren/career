-- Stores the SHA-256 of each attempt's analysed inputs (language, stated time,
-- stated space, code), so analysis queries compare it with
-- leetgrinder_analyses.code_sha256 instead of rehashing every attempt's code.
-- A generated column cannot hold it: convert_to is STABLE, not IMMUTABLE. A
-- trigger keeps it current on every insert and on updates that change an
-- input. The expression is the one analyses were stored against, so existing
-- analyses stay current.
ALTER TABLE leetgrinder_attempts ADD COLUMN input_sha256 BYTEA;

UPDATE leetgrinder_attempts
SET input_sha256 = sha256(convert_to(code_language || chr(10) || time_complexity || chr(10) || space_complexity || chr(10) || code, 'UTF8'));

ALTER TABLE leetgrinder_attempts
    ALTER COLUMN input_sha256 SET NOT NULL,
    ADD CONSTRAINT leetgrinder_attempts_input_sha256_check CHECK (octet_length(input_sha256) = 32);

CREATE OR REPLACE FUNCTION leetgrinder_attempts_input_sha256() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    -- An update that leaves the inputs alone (a notes or marked_at change)
    -- keeps the stored hash without rereading the code; any value written
    -- to input_sha256 directly is replaced either way.
    IF TG_OP = 'UPDATE'
       AND NEW.code_language = OLD.code_language AND NEW.time_complexity = OLD.time_complexity
       AND NEW.space_complexity = OLD.space_complexity AND NEW.code = OLD.code THEN
        NEW.input_sha256 := OLD.input_sha256;
        RETURN NEW;
    END IF;
    NEW.input_sha256 := sha256(convert_to(NEW.code_language || chr(10) || NEW.time_complexity || chr(10) || NEW.space_complexity || chr(10) || NEW.code, 'UTF8'));
    RETURN NEW;
END;
$$;

CREATE TRIGGER leetgrinder_attempts_input_sha256
    BEFORE INSERT OR UPDATE ON leetgrinder_attempts
    FOR EACH ROW EXECUTE FUNCTION leetgrinder_attempts_input_sha256();
