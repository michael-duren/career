package database

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func stepGoal(titles ...string) Entity {
	g := dependencyGoal(uuid.NewString())
	steps := []any{}
	for _, t := range titles {
		steps = append(steps, map[string]any{"id": uuid.NewString(), "title": t, "done": false})
	}
	g["steps"] = steps
	return g
}

func stepTitles(e Entity) []string {
	titles := []string{}
	for _, v := range e["steps"].([]any) {
		titles = append(titles, v.(map[string]any)["title"].(string))
	}
	return titles
}

func stepID(e Entity, i int) string {
	return e["steps"].([]any)[i].(map[string]any)["id"].(string)
}

func TestMoveGoalStep(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a, err := s.Save(ctx, "goal", stepGoal("a1", "a2", "a3"), nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Save(ctx, "goal", stepGoal("b1"), nil)
	if err != nil {
		t.Fatal(err)
	}
	aID, bID := a.Entry["id"].(string), b.Entry["id"].(string)

	// Reorder within one goal: a3 to the front.
	saved, err := s.MoveGoalStep(ctx, StepMove{StepID: stepID(a.Entry, 2), From: aID, FromRevision: a.Revision, To: aID, ToRevision: a.Revision, Index: 0})
	if err != nil || len(saved) != 1 {
		t.Fatal(saved, err)
	}
	if got := stepTitles(saved[0].Entry); len(got) != 3 || got[0] != "a3" || got[1] != "a1" || got[2] != "a2" {
		t.Fatal(got)
	}
	a = saved[0]

	// A stale revision is rejected and changes nothing.
	if _, err = s.MoveGoalStep(ctx, StepMove{StepID: stepID(a.Entry, 0), From: aID, FromRevision: "stale", To: bID, ToRevision: b.Revision}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale source: %v", err)
	}
	if _, err = s.MoveGoalStep(ctx, StepMove{StepID: stepID(a.Entry, 0), From: aID, FromRevision: a.Revision, To: bID, ToRevision: "stale"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale destination: %v", err)
	}

	// Move a1 (done) to goal b between nothing and b1; ID and done state survive.
	moving := a.Entry["steps"].([]any)[1].(map[string]any)
	moving["done"] = true
	if a, err = s.Save(ctx, "goal", a.Entry, &a.Revision); err != nil {
		t.Fatal(err)
	}
	saved, err = s.MoveGoalStep(ctx, StepMove{StepID: moving["id"].(string), From: aID, FromRevision: a.Revision, To: bID, ToRevision: b.Revision, Index: 0})
	if err != nil || len(saved) != 2 {
		t.Fatal(saved, err)
	}
	if got := stepTitles(saved[0].Entry); len(got) != 2 || got[0] != "a3" || got[1] != "a2" {
		t.Fatal("source", got)
	}
	moved := saved[1].Entry["steps"].([]any)[0].(map[string]any)
	if got := stepTitles(saved[1].Entry); len(got) != 2 || got[0] != "a1" || moved["id"] != moving["id"] || moved["done"] != true {
		t.Fatal("destination", saved[1].Entry["steps"])
	}
	a, b = saved[0], saved[1]
	if detail, _ := s.Detail(ctx, "goal", bID); detail.Revision != b.Revision {
		t.Fatal("returned revision is not the saved one")
	}

	// An index past the end appends.
	if saved, err = s.MoveGoalStep(ctx, StepMove{StepID: stepID(a.Entry, 0), From: aID, FromRevision: a.Revision, To: bID, ToRevision: b.Revision, Index: 99}); err != nil {
		t.Fatal(err)
	}
	if got := stepTitles(saved[1].Entry); len(got) != 3 || got[2] != "a3" {
		t.Fatal(got)
	}
	a, b = saved[0], saved[1]

	if _, err = s.MoveGoalStep(ctx, StepMove{StepID: uuid.NewString(), From: aID, FromRevision: a.Revision, To: bID, ToRevision: b.Revision}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing step: %v", err)
	}
	if _, err = s.MoveGoalStep(ctx, StepMove{StepID: stepID(a.Entry, 0), From: aID, FromRevision: a.Revision, To: uuid.NewString(), ToRevision: b.Revision}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing destination: %v", err)
	}
	if _, err = s.MoveGoalStep(ctx, StepMove{StepID: stepID(a.Entry, 0), From: aID, FromRevision: a.Revision, To: bID, ToRevision: b.Revision, Index: -1}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("negative index: %v", err)
	}
	// Failed moves leave both goals untouched.
	if detail, _ := s.Detail(ctx, "goal", aID); detail.Revision != a.Revision || len(stepTitles(detail.Entry)) != 1 {
		t.Fatal("failed move changed source", detail)
	}
}
