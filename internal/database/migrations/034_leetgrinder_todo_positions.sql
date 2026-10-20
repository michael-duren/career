-- Set entries keep an explicit order: position is the entry's 0-based index
-- in its set. Existing entries keep their creation order. Individual entries
-- have no position and stay in creation order. The uniqueness check is
-- deferred so a reorder can rewrite every position in one statement.
ALTER TABLE leetgrinder_todo_items ADD COLUMN position INTEGER;
UPDATE leetgrinder_todo_items i SET position=r.n FROM (
  SELECT id,row_number() OVER (PARTITION BY set_id ORDER BY created_at,id)-1 AS n
  FROM leetgrinder_todo_items WHERE set_id IS NOT NULL
) r WHERE i.id=r.id;
ALTER TABLE leetgrinder_todo_items ADD CONSTRAINT leetgrinder_todo_items_position_check
    CHECK ((set_id IS NULL) = (position IS NULL) AND position >= 0);
ALTER TABLE leetgrinder_todo_items ADD CONSTRAINT leetgrinder_todo_items_set_position
    UNIQUE (set_id, position) DEFERRABLE INITIALLY DEFERRED;
DROP INDEX leetgrinder_todo_set_items;
