ALTER TABLE leetgrinder_attempts
 ADD COLUMN claim TEXT NOT NULL DEFAULT '' CHECK (char_length(claim) <= 2000),
 ADD COLUMN invariant TEXT NOT NULL DEFAULT '' CHECK (char_length(invariant) <= 2000),
 ADD COLUMN correctness_initially TEXT NOT NULL DEFAULT '' CHECK (char_length(correctness_initially) <= 2000),
 ADD COLUMN after_step TEXT NOT NULL DEFAULT '' CHECK (char_length(after_step) <= 2000),
 ADD COLUMN therefore TEXT NOT NULL DEFAULT '' CHECK (char_length(therefore) <= 2000),
 ADD COLUMN termination TEXT NOT NULL DEFAULT '' CHECK (char_length(termination) <= 2000);
ALTER TABLE leetgrinder_analyses ADD COLUMN correctness_feedback TEXT NOT NULL DEFAULT '' CHECK (char_length(correctness_feedback) <= 6000);

CREATE OR REPLACE FUNCTION leetgrinder_attempts_input_sha256() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE inputs TEXT;
BEGIN
 IF TG_OP = 'UPDATE' AND NEW.code_language = OLD.code_language AND NEW.time_complexity = OLD.time_complexity AND NEW.space_complexity = OLD.space_complexity AND NEW.code = OLD.code AND NEW.claim = OLD.claim AND NEW.invariant = OLD.invariant AND NEW.correctness_initially = OLD.correctness_initially AND NEW.after_step = OLD.after_step AND NEW.therefore = OLD.therefore AND NEW.termination = OLD.termination THEN
 NEW.input_sha256 := OLD.input_sha256; RETURN NEW;
 END IF;
 inputs := NEW.code_language || chr(10) || NEW.time_complexity || chr(10) || NEW.space_complexity || chr(10) || NEW.code;
 IF NEW.claim <> '' OR NEW.invariant <> '' OR NEW.correctness_initially <> '' OR NEW.after_step <> '' OR NEW.therefore <> '' OR NEW.termination <> '' THEN
 inputs := inputs || chr(10) || json_build_array(NEW.claim, NEW.invariant, NEW.correctness_initially, NEW.after_step, NEW.therefore, NEW.termination)::text;
 END IF;
 NEW.input_sha256 := sha256(convert_to(inputs, 'UTF8'));
 RETURN NEW;
END;
$$;
