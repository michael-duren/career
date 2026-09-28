ALTER TABLE scheduler_state ADD COLUMN reconciled_at timestamptz;
DROP TRIGGER scheduler_google_dirty ON scheduler_state;
CREATE TRIGGER scheduler_google_dirty AFTER INSERT OR UPDATE OF document ON scheduler_state
FOR EACH ROW EXECUTE FUNCTION queue_scheduler_google();
