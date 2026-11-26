-- App-wide settings: which modules are on, their order in the phone tab
-- bar, the theme, and when first-run setup finished. A module missing from
-- modules is on. A missing row means a new install that has not been set up.
CREATE TABLE app_settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    modules JSONB NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(modules) = 'object'),
    module_order JSONB NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(module_order) = 'array'),
    -- Keep in sync with modules.Themes; a new theme needs a migration.
    theme TEXT NOT NULL DEFAULT 'dark' CHECK (theme IN ('dark', 'vellum', 'cobalt', 'concrete')),
    setup_completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Existing installs keep every module on and skip setup. An install that
-- applied earlier migrations in an earlier transaction already existed; a
-- new database runs every migration in this one transaction, where now() is
-- the same for every row.
INSERT INTO app_settings (id, setup_completed_at)
SELECT 1, now()
WHERE EXISTS (SELECT 1 FROM schema_migrations WHERE applied_at < now());
