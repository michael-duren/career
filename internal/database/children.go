package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

func writeChildren(ctx context.Context, tx *sql.Tx, kind, id string, e Entity) error {
	exec := func(q string, a ...any) error { _, err := tx.ExecContext(ctx, q, a...); return err }
	switch kind {
	case "week":
		if err := exec("DELETE FROM journal_week_metrics WHERE week_slug=$1", id); err != nil {
			return err
		}
		hours := e["hours"].(map[string]any)
		targets, _ := e["targets"].(map[string]any)
		keys := map[string]bool{}
		for k := range hours {
			keys[k] = true
		}
		for k := range targets {
			keys[k] = true
		}
		for k := range keys {
			if err := exec("INSERT INTO journal_week_metrics(week_slug,track,hours,target) VALUES($1,$2,$3,$4)", id, k, hours[k], targets[k]); err != nil {
				return err
			}
		}
	case "note":
		if err := exec("DELETE FROM note_todos WHERE note_id=$1", id); err != nil {
			return err
		}
		todos, _ := e["todos"].([]any)
		for i, v := range todos {
			o := v.(map[string]any)
			if err := exec("INSERT INTO note_todos(note_id,id,position,title,done) VALUES($1,$2,$3,$4,$5)", id, o["id"], i, o["title"], o["done"]); err != nil {
				return err
			}
		}
	case "goal":
		for _, t := range []string{"goal_notes", "goal_steps", "goal_metadata"} {
			if err := exec("DELETE FROM "+t+" WHERE goal_id=$1", id); err != nil {
				return err
			}
		}
		for i, v := range e["notes"].([]any) {
			o := v.(map[string]any)
			if err := exec("INSERT INTO goal_notes(goal_id,id,position,body,created_at) VALUES($1,$2,$3,$4,$5)", id, o["id"], i, o["body"], o["createdAt"]); err != nil {
				return err
			}
		}
		for i, v := range e["steps"].([]any) {
			o := v.(map[string]any)
			if err := exec("INSERT INTO goal_steps(goal_id,id,position,title,done) VALUES($1,$2,$3,$4,$5)", id, o["id"], i, o["title"], o["done"]); err != nil {
				return err
			}
		}
		for k, v := range e["metadata"].(map[string]any) {
			if err := exec("INSERT INTO goal_metadata(goal_id,key,value) VALUES($1,$2,$3)", id, k, v); err != nil {
				return err
			}
		}
	}
	return nil
}
func readChildren(ctx context.Context, q queryer, kind string, e Entity) error {
	id := e[models[kind].Key]
	switch kind {
	case "week":
		e["hours"] = map[string]any{}
		if e["_targetsPresent"] == true {
			e["targets"] = map[string]any{}
		}
		delete(e, "_targetsPresent")
		rows, err := q.QueryContext(ctx, "SELECT track,hours::double precision,target::double precision FROM journal_week_metrics WHERE week_slug=$1 ORDER BY track", id)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var k string
			var h, t sql.NullFloat64
			if err := rows.Scan(&k, &h, &t); err != nil {
				return err
			}
			if h.Valid {
				e["hours"].(map[string]any)[k] = h.Float64
			}
			if t.Valid {
				e["targets"].(map[string]any)[k] = t.Float64
			}
		}
		return rows.Err()
	case "note":
		todos, err := childJSON(ctx, q, "SELECT json_build_object('id',id,'title',title,'done',done) FROM note_todos WHERE note_id=$1 ORDER BY position", id)
		if err != nil {
			return err
		}
		// Omit when empty so exports of notes without todos stay unchanged.
		if len(todos) > 0 {
			e["todos"] = todos
		}
	case "goal":
		for key, query := range map[string]string{"notes": "SELECT json_build_object('id',id,'body',body,'createdAt',to_char(created_at AT TIME ZONE 'UTC','YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"')) FROM goal_notes WHERE goal_id=$1 ORDER BY position", "steps": "SELECT json_build_object('id',id,'title',title,'done',done) FROM goal_steps WHERE goal_id=$1 ORDER BY position"} {
			a, err := childJSON(ctx, q, query, id)
			if err != nil {
				return err
			}
			e[key] = a
		}
		rows, err := q.QueryContext(ctx, "SELECT key,value FROM goal_metadata WHERE goal_id=$1 ORDER BY key", id)
		if err != nil {
			return err
		}
		defer rows.Close()
		m := map[string]any{}
		for rows.Next() {
			var k, v string
			if err := rows.Scan(&k, &v); err != nil {
				return err
			}
			m[k] = v
		}
		e["metadata"] = m
		return rows.Err()
	}
	return nil
}
func childJSON(ctx context.Context, q queryer, query string, id any) ([]any, error) {
	rows, err := q.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	a := []any{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("child decode: %w", err)
		}
		a = append(a, v)
	}
	return a, rows.Err()
}
