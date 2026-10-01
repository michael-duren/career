package server

import (
	"context"
	"net/url"
	"strings"
	"testing"
)

func TestMCPLeetgrinderTodos(t *testing.T) {
	db := testDB(t)
	h := newOAuthHarness(t, db)
	clientID := h.register(testCallback)
	verifier := strings.Repeat("todo", 12)
	code := h.approve(h.consent(clientID, testCallback, pkce(verifier)), "write").Query().Get("code")
	status, tokens := h.token(url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {testCallback}, "client_id": {clientID}, "code_verifier": {verifier}})
	if status != 200 {
		t.Fatal(status, tokens)
	}
	call := connectMCP(t, h, tokens["access_token"].(string), 14)
	set, failed := call("create_leetgrinder_todo_set", map[string]any{"title": "Graphs", "problems": []string{"https://leetcode.com/problems/number-of-islands/", "clone-graph"}})
	if failed || set["problemCount"] != float64(2) {
		t.Fatal(set)
	}
	setID := set["id"].(string)
	item, failed := call("add_leetgrinder_todo_problem", map[string]any{"problem": "two-sum", "setID": setID})
	if failed || item["problemSlug"] != "two-sum" {
		t.Fatal(item)
	}
	if duplicate, failed := call("add_leetgrinder_todo_problem", map[string]any{"problem": "two-sum", "setID": setID}); failed || duplicate["id"] != item["id"] {
		t.Fatal("duplicate entry", duplicate)
	}
	standalone, failed := call("add_leetgrinder_todo_problem", map[string]any{"problem": "valid-parentheses"})
	if failed {
		t.Fatal(standalone)
	}
	listed, failed := call("list_leetgrinder_todos", nil)
	if failed {
		t.Fatal(listed)
	}
	sets := listed["sets"].([]any)
	problems := sets[0].(map[string]any)["problems"].([]any)
	if len(sets) != 1 || len(problems) != 3 || len(listed["individualProblems"].([]any)) != 1 {
		t.Fatal(listed)
	}
	if out, failed := call("add_leetgrinder_todo_problem", map[string]any{"problem": "https://example.com/problems/bad/"}); !failed || !strings.Contains(out["error"].(string), "invalid") {
		t.Fatal(out)
	}
	if out, failed := call("remove_leetgrinder_todo_problem", map[string]any{"id": standalone["id"]}); failed || out["removed"] != true {
		t.Fatal(out)
	}
	if out, failed := call("remove_leetgrinder_todo_set", map[string]any{"id": setID}); failed || out["removed"] != true {
		t.Fatal(out)
	}
	listed, failed = call("list_leetgrinder_todos", nil)
	if failed || len(listed["sets"].([]any)) != 0 || len(listed["individualProblems"].([]any)) != 0 {
		t.Fatal(listed)
	}
	imported, failed := call("create_leetgrinder_todo_set", map[string]any{
		"title": "Dynamic programming", "description": "Work through these in order", "metadata": map[string]any{"source": "study-plan", "week": 2},
		"problemDetails": []any{map[string]any{
			"problem": "unique-paths", "number": 62, "title": "Unique Paths", "difficulty": "Medium",
			"topics": []string{"dynamic-programming", "math"}, "metadata": map[string]any{"reason": "grid recurrence", "sourceID": "dp-4"},
		}},
	})
	if failed || imported["problemCount"] != float64(1) {
		t.Fatal(imported)
	}
	listed, failed = call("list_leetgrinder_todos", nil)
	if failed {
		t.Fatal(listed)
	}
	importedSet := listed["sets"].([]any)[0].(map[string]any)
	importedProblem := importedSet["problems"].([]any)[0].(map[string]any)
	if importedSet["description"] != "Work through these in order" || importedSet["metadata"].(map[string]any)["source"] != "study-plan" ||
		importedProblem["title"] != "Unique Paths" || importedProblem["number"] != float64(62) ||
		importedProblem["metadata"].(map[string]any)["sourceID"] != "dp-4" {
		t.Fatal("imported metadata missing", listed)
	}
	if out, failed := call("remove_leetgrinder_todo_set", map[string]any{"id": imported["id"]}); failed || out["removed"] != true {
		t.Fatal(out)
	}
	problem, err := db.LeetgrinderProblem(context.Background(), "unique-paths")
	if err != nil || problem.Title != "Unique Paths" || problem.Number != 62 || len(problem.ImportMetadata) != 1 {
		t.Fatal("problem metadata removed with todo", problem, err)
	}
	for _, raw := range problem.ImportMetadata {
		versions := raw.([]any)
		if versions[0].(map[string]any)["metadata"].(map[string]any)["sourceID"] != "dp-4" {
			t.Fatal("catalog source snapshot missing", problem.ImportMetadata)
		}
	}
	if out, failed := call("create_leetgrinder_todo_set", map[string]any{
		"title": "Invalid import", "problemDetails": []any{
			map[string]any{"problem": "custom-first-problem", "title": "First"},
			map[string]any{"problem": "custom-second-problem", "difficulty": "Extreme"},
		},
	}); !failed || !strings.Contains(out["error"].(string), "invalid") {
		t.Fatal(out)
	}
	if p, err := db.LeetgrinderProblem(context.Background(), "custom-first-problem"); err != nil || p.InCatalog {
		t.Fatal("invalid bulk import saved a problem", p, err)
	}
	for _, sourceID := range []string{"first", "second"} {
		out, failed := call("create_leetgrinder_todo_set", map[string]any{
			"title": sourceID, "problemDetails": []any{map[string]any{
				"problem": "two-sum", "title": "Source " + sourceID, "topics": []string{"source-topic"},
				"metadata": map[string]any{"sourceID": sourceID},
			}},
		})
		if failed {
			t.Fatal(out)
		}
	}
	listed, failed = call("list_leetgrinder_todos", nil)
	if failed || len(listed["sets"].([]any)) != 2 {
		t.Fatal(listed)
	}
	for i, sourceID := range []string{"first", "second"} {
		entry := listed["sets"].([]any)[i].(map[string]any)["problems"].([]any)[0].(map[string]any)
		if entry["title"] != "Two Sum" || entry["sourceProblem"].(map[string]any)["title"] != "Source "+sourceID ||
			entry["metadata"].(map[string]any)["sourceID"] != sourceID {
			t.Fatal("source set metadata changed", entry)
		}
	}
	if p, err := db.LeetgrinderProblem(context.Background(), "two-sum"); err != nil || len(p.ImportMetadata) != 2 {
		t.Fatal("catalog did not keep both source snapshots", p, err)
	}
	firstVersion, failed := call("add_leetgrinder_todo_problem", map[string]any{"problem": "source-version-problem", "metadata": map[string]any{"foo": 1, "bar": 2}})
	if failed {
		t.Fatal(firstVersion)
	}
	for _, metadata := range []map[string]any{{"foo": 1}, {"foo": 1}} {
		version, failed := call("add_leetgrinder_todo_problem", map[string]any{"problem": "source-version-problem", "metadata": metadata})
		if failed || version["id"] != firstVersion["id"] {
			t.Fatal(version)
		}
	}
	versioned, err := db.LeetgrinderProblem(context.Background(), "source-version-problem")
	if err != nil || len(versioned.ImportMetadata[firstVersion["id"].(string)].([]any)) != 2 {
		t.Fatal("source versions lost or duplicate retry archived", versioned, err)
	}
}
