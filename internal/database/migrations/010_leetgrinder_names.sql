-- Migration 009 retains its original SQL checksum for existing installations.
ALTER TABLE bootcamp_attempts RENAME TO leetgrinder_attempts;
ALTER TABLE bootcamp_completed_days RENAME TO leetgrinder_completed_days;
ALTER INDEX bootcamp_attempts_history RENAME TO leetgrinder_attempts_history;

DO $$
DECLARE
    item RECORD;
BEGIN
    FOR item IN
        SELECT conrelid::regclass AS table_name, conname
        FROM pg_constraint
        WHERE conrelid IN ('leetgrinder_attempts'::regclass, 'leetgrinder_completed_days'::regclass)
          AND conname LIKE 'bootcamp_%'
    LOOP
        EXECUTE format('ALTER TABLE %s RENAME CONSTRAINT %I TO %I',
            item.table_name, item.conname, replace(item.conname, 'bootcamp_', 'leetgrinder_'));
    END LOOP;
END $$;
