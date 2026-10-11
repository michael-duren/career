ALTER TABLE leetgrinder_analyses ADD COLUMN code_feedback TEXT NOT NULL DEFAULT '' CHECK (char_length(code_feedback) <= 3000);
