package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/modules"
)

func TestConnectionQueueRoutes(t *testing.T) {
	db := testDB(t)
	h := newOAuthHarness(t, db)
	ctx := context.Background()
	person, err := db.Save(ctx, "connection", database.Entity{"id": uuid.NewString(), "name": "Ada", "role": "", "companyName": "", "email": "", "queued": false, "notes": "", "tags": []any{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	id := person.Entry["id"].(string)
	headers := map[string]string{"Origin": testOrigin, "Content-Type": "application/json"}
	call := func(method, path string, body any) (int, map[string]any) {
		t.Helper()
		raw := ""
		if body != nil {
			b, _ := json.Marshal(body)
			raw = string(b)
		}
		w := h.do(method, path, raw, headers, true)
		var out map[string]any
		json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}

	if w := h.do("GET", "/api/connections/queue", "", nil, false); w.Code != 401 {
		t.Fatal("unauthenticated", w.Code)
	}
	if w := h.do("POST", "/api/connections/queue", `{}`, map[string]string{"Origin": "https://evil.example", "Content-Type": "application/json"}, true); w.Code != 403 {
		t.Fatal("foreign origin", w.Code)
	}
	if code, out := call("POST", "/api/connections/queue", map[string]any{"id": id, "revision": person.Revision, "stage": "to_contact", "until": "2026-12-01"}); code != 400 {
		t.Fatal("unknown field", code, out)
	}
	if code, _ := call("POST", "/api/connections/queue", map[string]any{"id": id, "stage": "to_contact"}); code != 400 {
		t.Fatal("missing revision", code)
	}
	if code, _ := call("POST", "/api/connections/queue", map[string]any{"id": id, "revision": person.Revision, "stage": ""}); code != 400 {
		t.Fatal("empty stage", code)
	}
	if code, _ := call("POST", "/api/connections/queue/sent", map[string]any{"id": id, "revision": person.Revision}); code != 400 {
		t.Fatal("sent while not queued", code)
	}
	if code, _ := call("POST", "/api/connections/queue", map[string]any{"id": id, "revision": "stale", "stage": "to_contact"}); code != 409 {
		t.Fatal("stale", code)
	}
	if code, _ := call("POST", "/api/connections/queue", map[string]any{"id": uuid.NewString(), "revision": "x", "stage": "to_contact"}); code != 404 {
		t.Fatal("missing person", code)
	}

	code, out := call("POST", "/api/connections/queue", map[string]any{"id": id, "revision": person.Revision, "stage": "to_contact"})
	if code != 200 || out["entry"].(map[string]any)["stage"] != "to_contact" {
		t.Fatal(code, out)
	}
	rev := out["revision"].(string)
	until := time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")
	if code, out = call("POST", "/api/connections/queue/snooze", map[string]any{"id": id, "revision": rev, "until": until}); code != 200 || out["entry"].(map[string]any)["snoozedUntil"] != until {
		t.Fatal(code, out)
	}
	rev = out["revision"].(string)
	if code, out = call("POST", "/api/connections/queue/sent", map[string]any{"id": id, "revision": rev}); code != 200 || out["entry"].(map[string]any)["stage"] != "messaged" || out["week"].(map[string]any)["sent"] != float64(1) {
		t.Fatal(code, out)
	}
	rev = out["revision"].(string)
	if code, out = call("PATCH", "/api/connections/queue/settings", map[string]any{"weeklyTarget": 4}); code != 200 || out["week"].(map[string]any)["target"] != float64(4) {
		t.Fatal(code, out)
	}
	if code, _ = call("PATCH", "/api/connections/queue/settings", map[string]any{"weeklyTarget": 0}); code != 400 {
		t.Fatal("target 0", code)
	}
	for _, path := range []string{"/api/connections/queue", "/api/connections/queue/"} {
		code, out = call("GET", path, nil)
		if code != 200 || len(out["connections"].([]any)) != 1 || out["revisions"].(map[string]any)[id] != rev || out["week"].(map[string]any)["target"] != float64(4) {
			t.Fatal(path, code, out)
		}
	}
	if code, out = call("DELETE", "/api/connections/queue", map[string]any{"id": id, "revision": rev}); code != 200 || out["entry"].(map[string]any)["stage"] != nil {
		t.Fatal(code, out)
	}
	rev = out["revision"].(string)
	if code, out = call("GET", "/api/connections/queue", nil); code != 200 || len(out["connections"].([]any)) != 0 {
		t.Fatal(code, out)
	}

	// The queue needs Companies; the rest of People keeps working without it.
	if _, err = db.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"companies": false}}); err != nil {
		t.Fatal(err)
	}
	routes := [][2]string{{"GET", "/api/connections/queue"}, {"POST", "/api/connections/queue"}, {"DELETE", "/api/connections/queue"}, {"POST", "/api/connections/queue/sent"}, {"POST", "/api/connections/queue/snooze"}, {"PATCH", "/api/connections/queue/settings"}}
	allOff := func(why string) {
		t.Helper()
		for _, route := range routes {
			// An empty body must still get the module 404, not a 400.
			if w := h.do(route[0], route[1], "", headers, true); w.Code != 404 {
				t.Fatal(why, route, w.Code, w.Body.String())
			}
		}
	}
	allOff("companies off")
	if code, _ = call("GET", "/api/connections", nil); code != 200 {
		t.Fatal("people with companies off", code)
	}
	if _, err = db.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"companies": true, "people": false}}); err != nil {
		t.Fatal(err)
	}
	allOff("people off")
	if _, err = db.UpdateAppSettings(ctx, modules.Patch{Modules: map[string]bool{"people": true}}); err != nil {
		t.Fatal(err)
	}
	if code, _ = call("GET", "/api/connections/queue", nil); code != 200 {
		t.Fatal("back on", code)
	}
	if code, _ = call("DELETE", "/api/connections/queue", map[string]any{"id": id, "revision": rev}); code != 400 {
		t.Fatal("remove someone not queued", code)
	}
}
