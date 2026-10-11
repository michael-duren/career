package server

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/michael-duren/career-strategy/internal/scheduler"
)

func TestMCPWeeklyScheduleCRUD(t *testing.T) {
	db := testDB(t)
	h := newOAuthHarness(t, db)
	token := func(scope string) string {
		clientID := h.register(testCallback)
		verifier := strings.Repeat("ws"+scope, 10)
		code := h.approve(h.consent(clientID, testCallback, pkce(verifier)), scope).Query().Get("code")
		status, tokens := h.token(url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {testCallback}, "client_id": {clientID}, "code_verifier": {verifier}})
		if status != 200 {
			t.Fatal(status, tokens)
		}
		return tokens["access_token"].(string)
	}
	call := connectMCP(t, h, token("write"), 25)
	today := time.Now().UTC().Format(time.DateOnly)
	mon := scheduler.Monday(scheduler.DateAdd(today, 14))
	wed := scheduler.DateAdd(mon, 2)

	goal, failed := call("create_career_entry", map[string]any{"kind": "goal", "entry": map[string]any{"title": "Ship", "startDate": scheduler.DateAdd(today, -30), "endDate": scheduler.DateAdd(today, 90), "status": "planned", "steps": []any{map[string]any{"title": "Draft"}}}})
	if failed {
		t.Fatal(goal)
	}
	goalID := goal["id"].(string)

	week, failed := call("get_weekly_schedule", map[string]any{"date": wed})
	if failed || week["week"] != mon || week["revision"] == "" {
		t.Fatal(week)
	}
	sessions := func(week map[string]any) []map[string]any {
		out := []map[string]any{}
		for _, raw := range week["sessions"].([]any) {
			out = append(out, raw.(map[string]any))
		}
		return out
	}

	read := connectMCP(t, h, token("read"), 25)
	if out, failed := read("save_schedule_session", map[string]any{"revision": week["revision"], "date": mon, "start": mon + "T10:00", "end": mon + "T11:00", "goalId": goalID}); !failed || !strings.Contains(out["error"].(string), "read-only") {
		t.Fatal("read-only token saved a session", out)
	}

	// Create, then edit, a one-off goal session; local times use the schedule zone.
	created, failed := call("save_schedule_session", map[string]any{"revision": week["revision"], "date": mon, "start": mon + "T10:00", "end": mon + "T11:00", "goalId": goalID})
	if failed || len(sessions(created)) != 1 {
		t.Fatal(created)
	}
	session := sessions(created)[0]
	sessionID := session["id"].(string)
	if session["assignment"].(map[string]any)["title"] != "Ship" || session["plan"].(map[string]any)["start"] != mon+"T10:00:00Z" {
		t.Fatal("session", session)
	}
	if out, failed := call("save_schedule_session", map[string]any{"revision": week["revision"], "date": mon, "start": mon + "T12:00", "end": mon + "T13:00", "goalId": goalID}); !failed || !strings.Contains(out["error"].(string), "changed since") {
		t.Fatal("stale revision accepted", out)
	}
	edited, failed := call("save_schedule_session", map[string]any{"revision": created["revision"], "id": sessionID, "date": mon, "start": mon + "T12:00:00Z", "end": mon + "T13:00:00Z", "goalId": goalID, "newStepTitle": "Review"})
	if failed || len(sessions(edited)) != 1 {
		t.Fatal(edited)
	}
	if got := sessions(edited)[0]; got["id"] != sessionID || got["plan"].(map[string]any)["start"] != mon+"T12:00:00Z" || got["assignment"].(map[string]any)["title"] != "Review" {
		t.Fatal("edited session", got)
	}

	// Fixed commitments need a title; overlapping plans are refused.
	if out, failed := call("save_schedule_session", map[string]any{"revision": edited["revision"], "date": mon, "start": mon + "T14:00", "end": mon + "T15:00"}); !failed || !strings.Contains(out["error"].(string), "title") {
		t.Fatal("untitled commitment", out)
	}
	if out, failed := call("save_schedule_session", map[string]any{"revision": edited["revision"], "date": mon, "start": mon + "T12:30", "end": mon + "T13:30", "title": "Lunch"}); !failed {
		t.Fatal("overlap accepted", out)
	}
	if out, failed := call("save_schedule_session", map[string]any{"revision": edited["revision"], "date": mon, "start": "noon", "end": mon + "T15:00", "title": "Lunch"}); !failed || !strings.Contains(out["error"].(string), "start must be") {
		t.Fatal("bad time accepted", out)
	}
	lunch, failed := call("save_schedule_session", map[string]any{"revision": edited["revision"], "date": mon, "start": mon + "T14:00", "end": mon + "T15:00", "title": "Lunch"})
	if failed || len(sessions(lunch)) != 2 {
		t.Fatal(lunch)
	}
	var lunchID string
	for _, s := range sessions(lunch) {
		if s["id"] != sessionID {
			lunchID = s["id"].(string)
		}
	}
	removed, failed := call("remove_schedule_session", map[string]any{"revision": lunch["revision"], "id": lunchID})
	if failed {
		t.Fatal(removed)
	}
	for _, s := range sessions(removed) {
		if s["id"] == lunchID && s["state"] != "canceled" {
			t.Fatal("not canceled", s)
		}
	}

	// Recurring rules: create, change from a later date, then end.
	ruled, failed := call("save_schedule_rule", map[string]any{"revision": removed["revision"], "weekday": 3, "localStart": "09:00", "durationMinutes": 60, "effectiveFrom": mon, "goalId": goalID})
	if failed || len(ruled["rules"].([]any)) != 1 {
		t.Fatal(ruled)
	}
	ruleID := ruled["rules"].([]any)[0].(map[string]any)["id"].(string)
	occurrences := 0
	for _, s := range sessions(ruled) {
		if s["ruleId"] == ruleID && s["date"] == wed {
			occurrences++
		}
	}
	if occurrences != 1 {
		t.Fatal("rule occurrence", ruled)
	}
	changed, failed := call("save_schedule_rule", map[string]any{"revision": ruled["revision"], "id": ruleID, "weekday": 3, "localStart": "10:00", "durationMinutes": 30, "effectiveFrom": scheduler.DateAdd(mon, 7), "goalId": goalID})
	if failed {
		t.Fatal(changed)
	}
	var newRuleID string
	for _, raw := range changed["rules"].([]any) {
		rule := raw.(map[string]any)
		if rule["id"] == ruleID && rule["effectiveTo"] != scheduler.DateAdd(mon, 6) {
			t.Fatal("old rule not split", rule)
		}
		if rule["id"] != ruleID {
			newRuleID = rule["id"].(string)
		}
	}
	if newRuleID == "" || changed["week"] != scheduler.DateAdd(mon, 7) {
		t.Fatal("changed rule", changed)
	}
	if out, failed := call("end_schedule_rule", map[string]any{"revision": changed["revision"], "id": newRuleID, "from": scheduler.DateAdd(today, -1)}); !failed || !strings.Contains(out["error"].(string), "today or later") {
		t.Fatal("ended rule in the past", out)
	}
	ended, failed := call("end_schedule_rule", map[string]any{"revision": changed["revision"], "id": newRuleID, "from": scheduler.DateAdd(mon, 14)})
	if failed || ended["week"] != scheduler.DateAdd(mon, 14) {
		t.Fatal(ended)
	}
	for _, s := range sessions(ended) {
		if s["ruleId"] == newRuleID {
			t.Fatal("ended rule still has occurrences", s)
		}
	}
	for _, raw := range ended["rules"].([]any) {
		if rule := raw.(map[string]any); rule["id"] == newRuleID && rule["effectiveTo"] != scheduler.DateAdd(mon, 13) {
			t.Fatal("rule not ended", rule)
		}
	}
	if out, failed := call("end_schedule_rule", map[string]any{"revision": ended["revision"], "id": ruleID, "from": scheduler.DateAdd(mon, 28)}); !failed || !strings.Contains(out["error"].(string), "already ended") {
		t.Fatal("replaced rule extended", out)
	}

	// Actual work: skip a plan, and log unplanned past work.
	skipped, failed := call("record_schedule_actual", map[string]any{"revision": ended["revision"], "sessionId": sessionID, "status": "skipped"})
	if failed || skipped["week"] != mon {
		t.Fatal(skipped)
	}
	for _, s := range sessions(skipped) {
		if s["id"] == sessionID && s["actual"].(map[string]any)["status"] != "skipped" {
			t.Fatal("not skipped", s)
		}
	}
	if out, failed := call("record_schedule_actual", map[string]any{"revision": skipped["revision"], "status": "assumed"}); !failed || !strings.Contains(out["error"].(string), "sessionId") {
		t.Fatal("assumed without session", out)
	}
	yesterday := scheduler.DateAdd(today, -1)
	logged, failed := call("record_schedule_actual", map[string]any{"revision": skipped["revision"], "status": "explicit", "start": yesterday + "T09:00", "end": yesterday + "T10:00", "goalId": goalID})
	if failed || logged["week"] != scheduler.Monday(yesterday) {
		t.Fatal(logged)
	}
	found := false
	for _, s := range sessions(logged) {
		if actual, _ := s["actual"].(map[string]any); actual != nil && actual["date"] == yesterday && s["plan"] == nil {
			found = true
		}
	}
	if !found {
		t.Fatal("unplanned actual missing", logged)
	}

	// Settings.
	if out, failed := call("update_schedule_settings", map[string]any{"revision": logged["revision"], "timeZone": "Not/AZone", "defaultDay": map[string]any{"start": "06:00", "end": "22:00"}}); !failed || !strings.Contains(out["error"].(string), "time zone") {
		t.Fatal("invalid zone accepted", out)
	}
	settled, failed := call("update_schedule_settings", map[string]any{"revision": logged["revision"], "timeZone": "America/Chicago", "defaultDay": map[string]any{"start": "06:00", "end": "22:00"}, "weekdays": map[string]any{"6": map[string]any{"start": "08:00", "end": "12:00"}}})
	if failed {
		t.Fatal(settled)
	}
	settings := settled["settings"].(map[string]any)
	if settings["timeZone"] != "America/Chicago" || settings["defaultDay"].(map[string]any)["end"] != "22:00" || settings["weekdays"].(map[string]any)["6"] == nil {
		t.Fatal("settings", settings)
	}
}
