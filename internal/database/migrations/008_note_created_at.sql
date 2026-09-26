-- Notes only tracked their last edit. Existing notes take that edit as their creation time.
ALTER TABLE notes ADD COLUMN created_at TIMESTAMPTZ;
UPDATE notes SET created_at=updated_at;
