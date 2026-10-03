package server

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestMCPLeetgrinderTools(t *testing.T) {
	db := testDB(t)
	h := newOAuthHarness(t, db)
	// Saved attempts get the database's time, so this test runs on the real
	// clock.
	now := time.Now().UTC()
	date := now.Format(time.DateOnly)
	weekOf := leetgrinder.Date(now, time.UTC).AddDate(0, 0, -((int(now.Weekday())+6)%7)).Format(time.DateOnly)
	ctx := context.Background()
	settings, err := db.LeetgrinderSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error {
		v.Timezone, v.Goal = "UTC", leetgrinder.DailyGoal{New: 1, Review: 1}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// Two problems struggled 20 days ago: both due, one picked for today.
	for _, slug := range []string{"two-sum", "valid-anagram"} {
		a := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: slug, Outcome: "struggled", Minutes: 30, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"}
		if _, err = db.SaveLeetgrinderAttempt(ctx, a, ""); err != nil {
			t.Fatal(err)
		}
		if _, err = db.DB.Exec("UPDATE leetgrinder_attempts SET created_at=$1 WHERE id=$2", now.AddDate(0, 0, -20), a.ID); err != nil {
			t.Fatal(err)
		}
	}
	token := func(scope string) string {
		clientID := h.register(testCallback)
		verifier := strings.Repeat("lg"+scope, 10)
		code := h.approve(h.consent(clientID, testCallback, pkce(verifier)), scope).Query().Get("code")
		status, tokens := h.token(url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {testCallback}, "client_id": {clientID}, "code_verifier": {verifier}})
		if status != 200 {
			t.Fatal(status, tokens)
		}
		return tokens["access_token"].(string)
	}
	call := connectMCP(t, h, token("write"), 18)

	today, failed := call("get_leetgrinder_today", nil)
	if failed || today["date"] != date || today["met"] != false || today["alsoDue"] != float64(1) {
		t.Fatal(today)
	}
	picks := today["reviewPicks"].([]any)
	if len(picks) != 1 || picks[0].(map[string]any)["reason"] == "" || today["goal"].(map[string]any)["review"] != float64(1) {
		t.Fatal("picks", today)
	}
	picked := picks[0].(map[string]any)["slug"].(string)

	reviews, failed := call("list_leetgrinder_reviews", map[string]any{"days": 3})
	if failed || reviews["total"] != float64(2) {
		t.Fatal(reviews)
	}
	first := reviews["reviews"].([]any)[0].(map[string]any)
	if first["slug"] != picked || first["todaysPick"] != true || first["due"] != true || first["flagged"] != true || first["lastOutcome"] != "struggled" {
		t.Fatal("review order", reviews)
	}
	// days 0 lists only today's; limit caps the list, not the total.
	if only, failed := call("list_leetgrinder_reviews", map[string]any{"days": 0, "limit": 1}); failed || only["days"] != float64(0) || only["total"] != float64(2) || len(only["reviews"].([]any)) != 1 {
		t.Fatal(only)
	}
	for _, bad := range []map[string]any{{"days": 31}, {"days": -1}, {"limit": 101}, {"limit": -1}} {
		if out, failed := call("list_leetgrinder_reviews", bad); !failed || !strings.Contains(out["error"].(string), "days must be") {
			t.Fatal(bad, out)
		}
	}

	// Logging from chat: NeetCode links resolve, complexity is required for
	// a solve, and the attempt counts like any other.
	if out, failed := call("log_leetgrinder_attempt", map[string]any{"problem": "group-anagrams", "outcome": "solved", "minutes": 20}); !failed || !strings.Contains(out["error"].(string), "complexity") {
		t.Fatal(out)
	}
	if out, failed := call("log_leetgrinder_attempt", map[string]any{"problem": "https://example.com/x", "outcome": "solved", "minutes": 20}); !failed || !strings.Contains(out["error"].(string), "invalid") {
		t.Fatal(out)
	}
	id := uuid.NewString()
	args := map[string]any{"id": id, "problem": "https://neetcode.io/problems/two-integer-sum", "outcome": "solved", "minutes": 12, "timeComplexity": "O(n)", "spaceComplexity": "O(n)", "code": "def f(): pass", "codeLanguage": "python3"}
	logged, failed := call("log_leetgrinder_attempt", args)
	if failed || logged["problemSlug"] != "two-sum" || logged["id"] != id || logged["kind"] != "review" || logged["warning"] != nil || logged["historyUrl"] != testOrigin+"/leetgrinder/problem/two-sum" {
		t.Fatal(logged)
	}
	// A problem never seen before is logged with a warning, in case of a typo.
	fresh, failed := call("log_leetgrinder_attempt", map[string]any{"problem": "some-made-up-problem", "outcome": "unfinished", "minutes": 5})
	if failed || fresh["kind"] != "new" || !strings.Contains(fresh["warning"].(string), "some-made-up-problem") {
		t.Fatal(fresh)
	}
	// The same id with the same values is not logged twice; changed values are refused.
	if again, failed := call("log_leetgrinder_attempt", args); failed || again["id"] != id {
		t.Fatal(again)
	}
	args["minutes"] = 13
	if out, failed := call("log_leetgrinder_attempt", args); !failed || !strings.Contains(out["error"].(string), "different values") {
		t.Fatal(out)
	}
	var count int
	var source string
	if err = db.DB.QueryRow("SELECT count(*), max(source) FROM leetgrinder_attempts WHERE problem_slug='two-sum' AND created_at > $1", now.AddDate(0, 0, -1)).Scan(&count, &source); err != nil || count != 1 || source != "mcp" {
		t.Fatalf("saved %d attempts from %q: %v", count, source, err)
	}

	stats, failed := call("get_leetgrinder_stats", map[string]any{"weeks": 4})
	if failed {
		t.Fatal(stats)
	}
	if topics := stats["topics"].([]any); len(topics) == 0 {
		t.Fatal("topics", stats)
	}
	weeks := stats["weeklyTrends"].([]any)
	if len(weeks) != 4 {
		t.Fatal("weeks", stats)
	}
	this := weeks[3].(map[string]any)
	if this["weekOf"] != weekOf || this["problemsSolved"] != float64(1) || this["independentRate"] != float64(1) {
		t.Fatal("this week", this)
	}
	if out, failed := call("get_leetgrinder_stats", map[string]any{"weeks": 27}); !failed {
		t.Fatal(out)
	}

	// A read-only connection reads but cannot log.
	read := connectMCP(t, h, token("read"), 18)
	if out, failed := read("get_leetgrinder_today", nil); failed {
		t.Fatal(out)
	}
	var before, after int
	if err = db.DB.QueryRow("SELECT count(*) FROM leetgrinder_attempts").Scan(&before); err != nil {
		t.Fatal(err)
	}
	if out, failed := read("log_leetgrinder_attempt", map[string]any{"problem": "two-sum", "outcome": "unfinished", "minutes": 5}); !failed || !strings.Contains(strings.ToLower(out["error"].(string)), "edit") {
		t.Fatal(out)
	}
	if err = db.DB.QueryRow("SELECT count(*) FROM leetgrinder_attempts").Scan(&after); err != nil || after != before {
		t.Fatalf("read-only call saved an attempt: %d -> %d %v", before, after, err)
	}
}

// Logging can be the day's first access: the attempt counts as a review and
// the day is planned with it as today's pick.
func TestMCPLeetgrinderLogFirst(t *testing.T) {
	db := testDB(t)
	h := newOAuthHarness(t, db)
	ctx := context.Background()
	settings, err := db.LeetgrinderSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.UpdateLeetgrinderSettings(ctx, settings.Revision, func(v *leetgrinder.Settings) error { v.Timezone = "UTC"; return nil }); err != nil {
		t.Fatal(err)
	}
	a := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "struggled", Minutes: 30, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"}
	if _, err = db.SaveLeetgrinderAttempt(ctx, a, ""); err != nil {
		t.Fatal(err)
	}
	if _, err = db.DB.Exec("UPDATE leetgrinder_attempts SET created_at=now()-interval '20 days' WHERE id=$1", a.ID); err != nil {
		t.Fatal(err)
	}
	clientID := h.register(testCallback)
	verifier := strings.Repeat("first", 10)
	code := h.approve(h.consent(clientID, testCallback, pkce(verifier)), "write").Query().Get("code")
	status, tokens := h.token(url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {testCallback}, "client_id": {clientID}, "code_verifier": {verifier}})
	if status != 200 {
		t.Fatal(status, tokens)
	}
	call := connectMCP(t, h, tokens["access_token"].(string), 18)
	logged, failed := call("log_leetgrinder_attempt", map[string]any{"problem": "two-sum", "outcome": "unfinished", "minutes": 10})
	if failed || logged["kind"] != "review" {
		t.Fatal(logged)
	}
	// Today is planned, with the review as its pick.
	var picks int
	if err = db.DB.QueryRow("SELECT count(*) FROM leetgrinder_review_plan WHERE problem_slug='two-sum'").Scan(&picks); err != nil || picks != 1 {
		t.Fatalf("picks %d: %v", picks, err)
	}
}
