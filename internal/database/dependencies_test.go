package database

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"reflect"
	"sync"
	"testing"
)

func dependencyGoal(id string) Entity {
	return Entity{"id": id, "title": id, "status": "planned", "dependsOn": []any{}, "startDate": "2026-01-01", "endDate": "2026-01-03", "color": "#abcdef", "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z", "notes": []any{}, "steps": []any{}, "metadata": map[string]any{}}
}
func TestDependencies(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a, err := s.Save(ctx, "goal", dependencyGoal(uuid.NewString()), nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Save(ctx, "goal", dependencyGoal(uuid.NewString()), nil)
	if err != nil {
		t.Fatal(err)
	}
	b.Entry["dependsOn"] = []any{a.Entry["id"]}
	old := b.Revision
	b, err = s.Save(ctx, "goal", b.Entry, &b.Revision)
	if err != nil || old == b.Revision {
		t.Fatalf("edge revision: %v", err)
	}
	a.Entry["dependsOn"] = []any{b.Entry["id"]}
	if _, err = s.Save(ctx, "goal", a.Entry, &a.Revision); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cycle: %v", err)
	}
	a.Entry["dependsOn"] = []any{a.Entry["id"]}
	if _, err = s.Save(ctx, "goal", a.Entry, &a.Revision); !errors.Is(err, ErrInvalid) {
		t.Fatalf("self cycle: %v", err)
	}
	if _, err = s.DB.Exec("INSERT INTO goal_dependencies(goal_id,depends_on_id,position) VALUES($1,$1,0)", a.Entry["id"]); err == nil {
		t.Fatal("self CHECK accepted")
	}
	if err = s.Delete(ctx, "goal", a.Entry["id"].(string), &a.Revision); err != nil {
		t.Fatal(err)
	}
	after, err := s.Detail(ctx, "goal", b.Entry["id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Entry["dependsOn"].([]any)) != 0 || after.Revision == b.Revision {
		t.Fatal("cascade must clear edges and invalidate dependent revision")
	}
	if _, err = s.Save(ctx, "goal", b.Entry, &b.Revision); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale dependent save: %v", err)
	}
}
func TestConcurrentDependencyCycle(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a, _ := s.Save(ctx, "goal", dependencyGoal(uuid.NewString()), nil)
	b, _ := s.Save(ctx, "goal", dependencyGoal(uuid.NewString()), nil)
	a.Entry["dependsOn"] = []any{b.Entry["id"]}
	b.Entry["dependsOn"] = []any{a.Entry["id"]}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, g := range []Result{a, b} {
		wg.Add(1)
		go func(g Result) { defer wg.Done(); _, err := s.Save(ctx, "goal", g.Entry, &g.Revision); results <- err }(g)
	}
	wg.Wait()
	close(results)
	failures := 0
	for err := range results {
		if err != nil {
			if !errors.Is(err, ErrInvalid) {
				t.Fatal(err)
			}
			failures++
		}
	}
	if failures != 1 {
		t.Fatalf("expected one cycle rejection, got %d", failures)
	}
}
func TestDependencyTransfer(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a := dependencyGoal(uuid.NewString())
	b := dependencyGoal(uuid.NewString())
	c := dependencyGoal(uuid.NewString())
	a["status"] = "active"
	b["status"] = "dropped"
	a["dependsOn"] = []any{c["id"], b["id"]}
	data := map[string]any{"version": 2, "notes": []any{}, "weeks": []any{}, "books": []any{}, "companies": []any{}, "documents": []any{}, "goals": []any{a, b, c}}
	raw, _ := json.Marshal(data)
	if _, err := s.Import(ctx, bytes.NewReader(raw), Source{}, false); err != nil {
		t.Fatal(err)
	}
	var exported bytes.Buffer
	if err := s.Export(ctx, &exported); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	json.Unmarshal(exported.Bytes(), &got)
	var want map[string]any
	json.Unmarshal(raw, &want)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("round trip differs: %s", exported.String())
	}
	second := testStore(t)
	if _, err := second.Import(ctx, bytes.NewReader(exported.Bytes()), Source{}, false); err != nil {
		t.Fatal(err)
	}
}
func TestDependencyBackfill(t *testing.T) {
	s := testStore(t)
	if _, err := s.DB.Exec("DROP TABLE goal_dependencies; ALTER TABLE goals DROP COLUMN status; DELETE FROM schema_migrations WHERE version=3"); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		offset, end int
		unchecked   bool
		want        string
	}{{-5, -1, false, "done"}, {-5, -1, true, "active"}, {0, 1, false, "active"}, {1, 2, false, "planned"}} {
		id := uuid.NewString()
		_, err := s.DB.Exec("INSERT INTO goals(id,title,start_date,end_date,color,created_at,updated_at,revision,position) VALUES($1,$2,CURRENT_DATE+$3::int,CURRENT_DATE+$4::int,'#abcdef',now(),now(),'r',nextval('entity_position'))", id, item.want, item.offset, item.end)
		if err != nil {
			t.Fatal(err)
		}
		if item.unchecked {
			if _, err = s.DB.Exec("INSERT INTO goal_steps(goal_id,id,position,title,done) VALUES($1,$2,0,'step',false)", id, uuid.NewString()); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	var wrong int
	if err := s.DB.QueryRow("SELECT count(*) FROM goals WHERE status<>title").Scan(&wrong); err != nil || wrong != 0 {
		t.Fatalf("backfill wrong: %d %v", wrong, err)
	}
}

func TestMissingGoalPayload(t *testing.T) {
	if _, err := PrepareSave("goal", nil); err == nil {
		t.Fatal("nil goal accepted")
	}
}
