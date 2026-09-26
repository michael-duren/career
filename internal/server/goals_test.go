package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/database"
)

func TestMoveGoalStepRoute(t *testing.T) {
	db := testDB(t)
	h := newOAuthHarness(t, db)
	ctx := context.Background()
	stepID := uuid.NewString()
	save := func(title string, steps ...any) database.Result {
		g, err := database.PrepareSave("goal", database.Entity{"id": uuid.NewString(), "title": title, "status": "planned", "dependsOn": []any{}, "startDate": "2026-10-01", "endDate": "2026-10-02", "color": "#abcdef", "createdAt": "2026-10-01T00:00:00Z", "updatedAt": "2026-10-01T00:00:00Z", "notes": []any{}, "steps": steps, "metadata": map[string]any{}})
		if err != nil {
			t.Fatal(err)
		}
		r, err := db.Save(ctx, "goal", g, nil)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	a := save("A", map[string]any{"id": stepID, "title": "mini", "done": true})
	b := save("B")
	aID, bID := a.Entry["id"].(string), b.Entry["id"].(string)
	body := func(fromRev, to, toRev string) string {
		raw, _ := json.Marshal(map[string]any{"stepId": stepID, "from": aID, "fromRevision": fromRev, "to": to, "toRevision": toRev, "index": 0})
		return string(raw)
	}
	headers := map[string]string{"Origin": testOrigin, "Content-Type": "application/json"}
	move := func(payload string, hdr map[string]string, session bool) (int, map[string]any) {
		w := h.do("POST", "/api/goals/steps/move", payload, hdr, session)
		var out map[string]any
		json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}

	valid := body(a.Revision, bID, b.Revision)
	if code, _ := move(valid, headers, false); code != 401 {
		t.Fatal("unauthenticated", code)
	}
	if code, _ := move(valid, map[string]string{"Origin": "https://evil.example", "Content-Type": "application/json"}, true); code != 403 {
		t.Fatal("foreign origin", code)
	}
	if code, _ := move(valid, map[string]string{"Origin": testOrigin, "Content-Type": "text/plain"}, true); code != 415 {
		t.Fatal("content type", code)
	}
	if code, _ := move(`{"stepId":"x","extra":1}`, headers, true); code != 400 {
		t.Fatal("unknown field", code)
	}
	if code, out := move(body(a.Revision, aID, "other"), headers, true); code != 400 {
		t.Fatal("same goal, differing revisions", code, out)
	}
	if code, _ := move(body("stale", bID, b.Revision), headers, true); code != 409 {
		t.Fatal("stale revision", code)
	}
	if code, _ := move(body(a.Revision, uuid.NewString(), b.Revision), headers, true); code != 404 {
		t.Fatal("missing destination", code)
	}

	code, out := move(valid, headers, true)
	goals, _ := out["goals"].([]any)
	revisions, _ := out["revisions"].(map[string]any)
	if code != 200 || len(goals) != 2 || len(revisions) != 2 {
		t.Fatal(code, out)
	}
	for _, raw := range goals {
		g := raw.(map[string]any)
		steps := g["steps"].([]any)
		saved, err := db.Detail(ctx, "goal", g["id"].(string))
		if err != nil || revisions[g["id"].(string)] != saved.Revision {
			t.Fatal("revision mismatch", g["id"], err)
		}
		if g["id"] == aID && len(steps) != 0 || g["id"] == bID && (len(steps) != 1 || steps[0].(map[string]any)["id"] != stepID || steps[0].(map[string]any)["done"] != true) {
			t.Fatal("unexpected steps", g)
		}
	}
	// The step is no longer in the source goal.
	if code, _ := move(body(revisions[aID].(string), bID, revisions[bID].(string)), headers, true); code != 404 {
		t.Fatal("moved step still in source", code)
	}
}
