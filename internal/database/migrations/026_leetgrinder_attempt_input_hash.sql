-- Stores the SHA-256 of each attempt's analysed inputs (language, stated time,
-- stated space, code), so analysis queries compare it with
-- leetgrinder_analyses.code_sha256 instead of rehashing every attempt's code.
-- A generated column cannot hold it: convert_to is STABLE, not IMMUTABLE. A
-- trigger keeps it current on every insert and update instead. The expression
-- is the one analyses were stored against, so existing analyses stay current.
ALTER TABLE leetgrinder_attempts ADD COLUMN input_sha256 BYTEA;

UPDATE leetgrinder_attempts
SET input_sha256 = sha256(convert_to(code_language || chr(10) || time_complexity || chr(10) || space_complexity || chr(10) || code, 'UTF8'));

ALTER TABLE leetgrinder_attempts
    ALTER COLUMN input_sha256 SET NOT NULL,
    ADD CONSTRAINT leetgrinder_attempts_input_sha256_check CHECK (octet_length(input_sha256) = 32);

CREATE OR REPLACE FUNCTION leetgrinder_attempts_input_sha256() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.input_sha256 := sha256(convert_to(NEW.code_language || chr(10) || NEW.time_complexity || chr(10) || NEW.space_complexity || chr(10) || NEW.code, 'UTF8'));
    RETURN NEW;
END;
$$;

CREATE TRIGGER leetgrinder_attempts_input_sha256
    BEFORE INSERT OR UPDATE ON leetgrinder_attempts
    FOR EACH ROW EXECUTE FUNCTION leetgrinder_attempts_input_sha256();
