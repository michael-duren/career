-- Stated time and space complexity, and the code LeetCode judged, per attempt.
-- Existing attempts keep empty values.
ALTER TABLE leetgrinder_attempts
    ADD COLUMN time_complexity TEXT NOT NULL DEFAULT '' CHECK (char_length(time_complexity) <= 40),
    ADD COLUMN space_complexity TEXT NOT NULL DEFAULT '' CHECK (char_length(space_complexity) <= 40),
    ADD COLUMN code TEXT NOT NULL DEFAULT '' CHECK (octet_length(code) <= 65536),
    ADD COLUMN code_language TEXT NOT NULL DEFAULT '' CHECK (char_length(code_language) <= 32);
