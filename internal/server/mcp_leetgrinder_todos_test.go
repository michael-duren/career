package server

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
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
	call := connectMCP(t, h, tokens["access_token"].(string), 26)
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
	if _, err := db.SaveLeetgrinderAttempt(context.Background(), leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "clone-graph", Outcome: "solved", Minutes: 15, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"}, ""); err != nil {
		t.Fatal(err)
	}
	listed, failed := call("list_leetgrinder_todos", nil)
	if failed {
		t.Fatal(listed)
	}
	sets := listed["sets"].([]any)
	listedSet := sets[0].(map[string]any)
	problems := listedSet["problems"].([]any)
	if len(sets) != 1 || len(problems) != 3 || len(listed["individualProblems"].([]any)) != 1 {
		t.Fatal(listed)
	}
	if listedSet["problemCount"] != float64(3) || listedSet["remainingCount"] != float64(2) || problems[1].(map[string]any)["done"] != true || problems[0].(map[string]any)["done"] != false {
		t.Fatal("set progress", listedSet)
	}
	if out, failed := call("add_leetgrinder_todo_problem", map[string]any{"problem": "https://example.com/problems/bad/"}); !failed || !strings.Contains(out["error"].(string), "invalid") {
		t.Fatal(out)
	}
	neet, failed := call("add_leetgrinder_todo_problem", map[string]any{"problem": "https://neetcode.io/problems/valid-sudoku"})
	if failed || neet["problemSlug"] != "valid-sudoku" {
		t.Fatal("neetcode link", neet)
	}
	if out, failed := call("remove_leetgrinder_todo_problem", map[string]any{"id": neet["id"]}); failed || out["removed"] != true {
		t.Fatal(out)
	}
	if out, failed := call("add_leetgrinder_todo_problem", map[string]any{"problem": "https://neetcode.io/problems/no-such-neetcode-problem"}); !failed || !strings.Contains(out["error"].(string), "unknown NeetCode problem") {
		t.Fatal("unknown neetcode link", out)
	}
	if out, failed := call("create_leetgrinder_todo_set", map[string]any{"title": "Mixed", "problems": []string{"https://neetcode.io/problems/no-such-neetcode-problem"}}); !failed || !strings.Contains(out["error"].(string), "unknown NeetCode problem") {
		t.Fatal("unknown neetcode link in set", out)
	}
	if out, failed := call("add_leetgrinder_todo_problem", map[string]any{"problem": "https://example.com/" + strings.Repeat("x", 500)}); !failed || len(out["error"].(string)) > 200 {
		t.Fatal("long rejected ref should be truncated", out)
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

func TestMCPLeetgrinderTodoOrder(t *testing.T) {
	db := testDB(t)
	h := newOAuthHarness(t, db)
	clientID := h.register(testCallback)
	verifier := strings.Repeat("order", 10)
	code := h.approve(h.consent(clientID, testCallback, pkce(verifier)), "write").Query().Get("code")
	status, tokens := h.token(url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {testCallback}, "client_id": {clientID}, "code_verifier": {verifier}})
	if status != 200 {
		t.Fatal(status, tokens)
	}
	call := connectMCP(t, h, tokens["access_token"].(string), 26)
	detail := func(slug, pattern string, neetcode150 bool, index int) map[string]any {
		return map[string]any{"problem": slug, "title": slug, "metadata": map[string]any{"pattern": pattern, "neetcode150": neetcode150, "positionInPattern": index}}
	}
	set, failed := call("create_leetgrinder_todo_set", map[string]any{"title": "NeetCode", "problemDetails": []any{
		detail("two-sum", "Arrays & Hashing", true, 1),
		detail("number-of-islands", "Graphs", true, 1),
		detail("clone-graph", "Graphs", false, 2),
		detail("invert-binary-tree", "Trees", true, 2),
		detail("maximum-depth-of-binary-tree", "Trees", true, 1),
	}})
	if failed {
		t.Fatal(set)
	}
	setID := set["id"].(string)
	order := func() []string {
		t.Helper()
		listed, failed := call("list_leetgrinder_todos", map[string]any{"setID": setID})
		if failed {
			t.Fatal(listed)
		}
		var slugs []string
		for i, p := range listed["sets"].([]any)[0].(map[string]any)["problems"].([]any) {
			problem := p.(map[string]any)
			if problem["position"] != float64(i) {
				t.Fatal("positions not dense", listed)
			}
			slugs = append(slugs, problem["slug"].(string))
		}
		return slugs
	}
	ids := map[string]string{}
	listed, _ := call("list_leetgrinder_todos", nil)
	for _, p := range listed["sets"].([]any)[0].(map[string]any)["problems"].([]any) {
		problem := p.(map[string]any)
		ids[problem["slug"].(string)] = problem["id"].(string)
	}

	sorted, failed := call("reorder_leetgrinder_set", map[string]any{"setID": setID, "orderBy": []any{
		map[string]any{"field": "metadata.pattern", "values": []any{"Trees", "Graphs"}},
		map[string]any{"field": "metadata.neetcode150", "values": []any{true}},
		map[string]any{"field": "metadata.positionInPattern"},
	}})
	if failed || sorted["moved"] != float64(5) || len(sorted["problems"].([]any)) != 5 {
		t.Fatal(sorted)
	}
	want := []string{"maximum-depth-of-binary-tree", "invert-binary-tree", "number-of-islands", "clone-graph", "two-sum"}
	if got := order(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatal("orderBy", got)
	}

	if out, failed := call("reorder_leetgrinder_set", map[string]any{"setID": setID, "orderedIds": []string{ids["two-sum"], ids["clone-graph"], ids["two-sum"]}}); failed || out["moved"] != float64(5) {
		t.Fatal(out)
	}
	want = []string{"two-sum", "clone-graph", "maximum-depth-of-binary-tree", "invert-binary-tree", "number-of-islands"}
	if got := order(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatal("orderedIds", got)
	}

	if out, failed := call("reorder_leetgrinder_set", map[string]any{"setID": setID, "moves": []any{map[string]any{"id": ids["number-of-islands"], "beforeId": ids["two-sum"]}}}); failed || out["problems"].([]any)[0].(map[string]any)["id"] != ids["number-of-islands"] {
		t.Fatal(out)
	}

	other, _ := call("create_leetgrinder_todo_set", map[string]any{"title": "Other", "problems": []string{"lru-cache"}})
	otherListed, _ := call("list_leetgrinder_todos", map[string]any{"setID": other["id"]})
	otherID := otherListed["sets"].([]any)[0].(map[string]any)["problems"].([]any)[0].(map[string]any)["id"].(string)
	before := order()
	out, failed := call("reorder_leetgrinder_set", map[string]any{"setID": setID, "orderedIds": []string{ids["clone-graph"], otherID, "not-an-id"}})
	if !failed || !strings.Contains(out["error"].(string), otherID) || !strings.Contains(out["error"].(string), "not-an-id") {
		t.Fatal("foreign IDs not rejected by name", out)
	}
	if got := order(); strings.Join(got, ",") != strings.Join(before, ",") {
		t.Fatal("rejected reorder changed the set", got)
	}
	for _, bad := range []map[string]any{
		{"setID": setID},
		{"setID": setID, "orderedIds": []string{ids["two-sum"]}, "moves": []any{map[string]any{"id": ids["two-sum"], "toIndex": 0}}},
		{"setID": setID, "moves": []any{map[string]any{"id": ids["two-sum"]}}},
		{"setID": setID, "orderBy": []any{map[string]any{"field": "sourceProblem"}}},
	} {
		if out, failed := call("reorder_leetgrinder_set", bad); !failed {
			t.Fatal("bad reorder accepted", bad, out)
		}
	}

	added, failed := call("add_leetgrinder_todo_problem", map[string]any{"problem": "valid-anagram", "setID": setID, "afterId": ids["number-of-islands"]})
	if failed || added["position"] != float64(1) {
		t.Fatal(added)
	}
	appended, failed := call("add_leetgrinder_todo_problem", map[string]any{"problem": "group-anagrams", "setID": setID})
	if failed || appended["position"] != float64(6) {
		t.Fatal(appended)
	}
	if moved, failed := call("add_leetgrinder_todo_problem", map[string]any{"problem": "group-anagrams", "setID": setID, "position": 0}); failed || moved["id"] != appended["id"] || moved["position"] != float64(0) {
		t.Fatal("re-adding with a position should move the entry", moved)
	}
	if out, failed := call("add_leetgrinder_todo_problem", map[string]any{"problem": "valid-anagram", "position": 0}); !failed || !strings.Contains(out["error"].(string), "setID") {
		t.Fatal(out)
	}
	if out, failed := call("remove_leetgrinder_todo_problem", map[string]any{"id": added["id"]}); failed {
		t.Fatal(out)
	}
	want = []string{"group-anagrams", "number-of-islands", "two-sum", "clone-graph", "maximum-depth-of-binary-tree", "invert-binary-tree"}
	if got := order(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatal("after add and remove", got)
	}

	if _, err := db.SaveLeetgrinderAttempt(context.Background(), leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "clone-graph", Outcome: "solved", Minutes: 15, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"}, ""); err != nil {
		t.Fatal(err)
	}
	page, failed := call("list_leetgrinder_todos", map[string]any{"setID": setID, "fields": "compact", "pattern": "graphs", "done": false})
	pageSet := page["sets"].([]any)[0].(map[string]any)
	if failed || pageSet["matchingCount"] != float64(1) || pageSet["problemCount"] != float64(6) || pageSet["remainingCount"] != float64(5) {
		t.Fatal(page)
	}
	compact := pageSet["problems"].([]any)[0].(map[string]any)
	if compact["title"] != "Number of Islands" || compact["pattern"] != "Graphs" || compact["neetcode150"] != true || compact["sourceProblem"] != nil || compact["metadata"] != nil {
		t.Fatal("compact entry", compact)
	}
	if _, ok := page["individualProblems"]; ok {
		t.Fatal("setID listing should leave out individual problems", page)
	}
	var paged []string
	cursor := ""
	for range 4 {
		page, failed = call("list_leetgrinder_todos", map[string]any{"setID": setID, "limit": 4, "cursor": cursor})
		if failed {
			t.Fatal(page)
		}
		pageSet = page["sets"].([]any)[0].(map[string]any)
		for _, p := range pageSet["problems"].([]any) {
			paged = append(paged, p.(map[string]any)["slug"].(string))
		}
		next, ok := pageSet["nextCursor"].(string)
		if !ok {
			break
		}
		cursor = next
	}
	if strings.Join(paged, ",") != strings.Join(want, ",") {
		t.Fatal("paging", paged)
	}
	if out, failed := call("list_leetgrinder_todos", map[string]any{"cursor": "-1"}); !failed {
		t.Fatal("bad cursor accepted", out)
	}
}
