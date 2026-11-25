-- App-wide settings: which modules are on, their order in the phone tab
-- bar, the theme, and when first-run setup finished. A module missing from
-- modules is on. A missing row means a new install that has not been set up.
CREATE TABLE app_settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    modules JSONB NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(modules) = 'object'),
    module_order JSONB NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(module_order) = 'array'),
    theme TEXT NOT NULL DEFAULT 'dark' CHECK (theme IN ('dark', 'vellum', 'cobalt', 'concrete')),
    setup_completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Existing workspaces keep every module on and skip setup.
INSERT INTO app_settings (id, setup_completed_at)
SELECT 1, now()
WHERE EXISTS (SELECT 1 FROM goals) OR EXISTS (SELECT 1 FROM notes) OR EXISTS (SELECT 1 FROM documents)
   OR EXISTS (SELECT 1 FROM journal_weeks) OR EXISTS (SELECT 1 FROM personal_journal_entries)
   OR EXISTS (SELECT 1 FROM books) OR EXISTS (SELECT 1 FROM companies) OR EXISTS (SELECT 1 FROM connections)
   OR EXISTS (SELECT 1 FROM running_notes) OR EXISTS (SELECT 1 FROM leetgrinder_attempts);
