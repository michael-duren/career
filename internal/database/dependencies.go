package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// All graph mutations take the same lock before touching rows. READ COMMITTED
// then sees the preceding writer's committed edges, preventing concurrent cycles.
func lockGoals(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(724193603)")
	return err
}
func writeDependencies(ctx context.Context, tx *sql.Tx, id string, deps []any) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM goal_dependencies WHERE goal_id=$1", id); err != nil {
		return err
	}
	for position, dep := range deps {
		var raw []byte
		var path []string
		if dep == id {
			return fmt.Errorf("%w: dependency cycle: %s → %s", ErrInvalid, id, id)
		}
		var exists bool
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM goals WHERE id=$1)", dep).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: prerequisite %s does not exist", ErrInvalid, dep)
		}
		err := tx.QueryRowContext(ctx, `WITH RECURSIVE reach(id,path) AS (
    SELECT $1::uuid, ARRAY[$1::text]
    UNION ALL SELECT d.depends_on_id, r.path || d.depends_on_id::text FROM reach r JOIN goal_dependencies d ON d.goal_id=r.id WHERE NOT d.depends_on_id::text=ANY(r.path)
  ) SELECT array_to_json(path) FROM reach WHERE id=$2::uuid LIMIT 1`, dep, id).Scan(&raw)
		if err == nil {
			json.Unmarshal(raw, &path)
			return fmt.Errorf("%w: dependency cycle: %s → %s", ErrInvalid, id, strings.Join(path, " → "))
		}
		if err != sql.ErrNoRows {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO goal_dependencies(goal_id,depends_on_id,position) VALUES($1,$2,$3)", id, dep, position); err != nil {
			return err
		}
	}
	return nil
}
