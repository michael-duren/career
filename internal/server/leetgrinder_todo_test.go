package server

import (
	"context"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderTodos(t *testing.T) {
	_, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	if w := request("POST", "/leetgrinder/todos/sets", url.Values{"title": {"Interview prep"}, "problems": {"https://leetcode.com/problems/two-sum/\nvalid-parentheses"}}); w.Code != 303 {
		t.Fatalf("create set: %d %s", w.Code, w.Body.String())
	}
	sets, err := db.LeetgrinderTodoSets(ctx)
	if err != nil || len(sets) != 1 || len(sets[0].Items) != 2 {
		t.Fatalf("saved set: %+v %v", sets, err)
	}
	// Trailing separators, blank lines and space-separated slugs are fine.
	messy := request("POST", "/leetgrinder/todos/sets", url.Values{"title": {"Messy"}, "problems": {"two-sum, three-sum, \n   \nvalid-anagram contains-duplicate\t\n"}})
	all, err := db.LeetgrinderTodoSets(ctx)
	if err != nil || len(all) != 2 {
		t.Fatalf("sets: %+v %v", all, err)
	}
	var messySet leetgrinder.TodoSet
	for _, set := range all {
		if set.Title == "Messy" {
			messySet = set
		}
	}
	// Creating a set opens it.
	if messy.Code != 303 || messy.Header().Get("Location") != "/leetgrinder/todos/sets/"+messySet.ID || len(messySet.Items) != 4 {
		t.Fatalf("messy list: %d %s %+v", messy.Code, messy.Header().Get("Location"), messySet.Items)
	}
	if err := db.DeleteLeetgrinderTodoSet(ctx, messySet.ID); err != nil {
		t.Fatal(err)
	}
	if bad := request("POST", "/leetgrinder/todos/sets", url.Values{"title": {"Bad"}, "problems": {"two-sum\nnot a/problem"}}); bad.Code != 400 || !strings.Contains(bad.Body.String(), "is not a LeetCode or NeetCode link or a LeetCode slug.") {
		t.Fatalf("bad entry: %d %s", bad.Code, bad.Body.String())
	}
	if bad := request("POST", "/leetgrinder/todos/sets", url.Values{"title": {"Bad"}, "problems": {"two-sum\nhttps://neetcode.io/problems/no-such-neetcode-problem"}}); bad.Code != 400 || !strings.Contains(bad.Body.String(), "no known LeetCode match") {
		t.Fatalf("unknown NeetCode entry: %d %s", bad.Code, bad.Body.String())
	}
	if bad := request("POST", "/leetgrinder/todos/items", url.Values{"problem": {"https://neetcode.io/problems/no-such-neetcode-problem"}}); bad.Code != 400 || !strings.Contains(bad.Body.String(), "no known LeetCode match") {
		t.Fatalf("unknown NeetCode item: %d %s", bad.Code, bad.Body.String())
	}
	if w := request("POST", "/leetgrinder/todos/sets", url.Values{"title": {"NeetCode"}, "problems": {"https://neetcode.io/problems/two-integer-sum validate-parentheses https://leetcode.cn/problems/valid-sudoku/"}}); w.Code != 303 {
		t.Fatalf("NeetCode set: %d %s", w.Code, w.Body.String())
	}
	var neetSet leetgrinder.TodoSet
	all, err = db.LeetgrinderTodoSets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, set := range all {
		if set.Title == "NeetCode" {
			neetSet = set
		}
	}
	if neetSet.ID == "" {
		t.Fatalf("NeetCode set not saved: %+v", all)
	}
	var slugs []string
	for _, item := range neetSet.Items {
		slugs = append(slugs, item.Problem.Slug)
	}
	if !reflect.DeepEqual(slugs, []string{"two-sum", "validate-parentheses", "valid-sudoku"}) {
		t.Fatalf("NeetCode set slugs: %v", slugs)
	}
	if err := db.DeleteLeetgrinderTodoSet(ctx, neetSet.ID); err != nil {
		t.Fatal(err)
	}
	if w := request("POST", "/leetgrinder/todos/items", url.Values{"problem": {"three-sum"}}); w.Code != 303 {
		t.Fatalf("add standalone: %d %s", w.Code, w.Body.String())
	}
	if w := request("POST", "/leetgrinder/todos/items", url.Values{"problem": {"merge-intervals"}, "setID": {sets[0].ID}}); w.Code != 303 {
		t.Fatalf("add to set: %d %s", w.Code, w.Body.String())
	}
	page := request("GET", "/leetgrinder/todos", nil)
	if page.Code != 200 || !strings.Contains(page.Body.String(), "Interview prep") || !strings.Contains(page.Body.String(), "three-sum") || !strings.Contains(page.Body.String(), "0 of 3 done") {
		t.Fatalf("todo page: %d %s", page.Code, page.Body.String())
	}
	if setPage := request("GET", "/leetgrinder/todos/sets/"+sets[0].ID, nil); setPage.Code != 200 || !strings.Contains(setPage.Body.String(), "merge-intervals") {
		t.Fatalf("set page: %d", setPage.Code)
	}
	items, err := db.LeetgrinderTodoItems(ctx, "")
	if err != nil || len(items) != 1 {
		t.Fatalf("standalone items: %+v %v", items, err)
	}
	if w := request("POST", "/leetgrinder/todos/items/"+items[0].ID+"/delete", nil); w.Code != 303 {
		t.Fatalf("delete item: %d %s", w.Code, w.Body.String())
	}
	if w := request("POST", "/leetgrinder/todos/sets/"+sets[0].ID+"/delete", nil); w.Code != 303 {
		t.Fatalf("delete set: %d %s", w.Code, w.Body.String())
	}
	sets, err = db.LeetgrinderTodoSets(ctx)
	if err != nil || len(sets) != 0 {
		t.Fatalf("sets after delete: %+v %v", sets, err)
	}
	items, err = db.LeetgrinderTodoItems(ctx, "")
	if err != nil || len(items) != 0 {
		t.Fatalf("items after delete: %+v %v", items, err)
	}
	if p, err := db.LeetgrinderProblem(ctx, "two-sum"); err != nil || !p.InCatalog {
		t.Fatalf("deleting todos must retain catalog: %+v %v", p, err)
	}
	if w := request("POST", "/leetgrinder/todos/sets", url.Values{"title": {"Start later"}}); w.Code != 303 {
		t.Fatalf("empty set: %d %s", w.Code, w.Body.String())
	}
	sets, err = db.LeetgrinderTodoSets(ctx)
	if err != nil || len(sets) != 1 || len(sets[0].Items) != 0 {
		t.Fatalf("empty set was not saved: %+v %v", sets, err)
	}
}

func TestTodoPagesShowDetailsAndKeepFormsSeparate(t *testing.T) {
	_, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	if _, err := db.AddLeetgrinderTodoProblem(ctx, "", leetgrinder.TodoProblemInput{Slug: "two-sum", Number: 1, Title: "Two Sum", Difficulty: "Easy", Topics: []string{"array", "hash-table"}}); err != nil {
		t.Fatal(err)
	}
	main := request("GET", "/leetgrinder/todos", nil)
	if main.Code != 200 {
		t.Fatalf("todos: %d", main.Code)
	}
	for _, want := range []string{"Two Sum", "Easy", "Array", "Hash Table", "Add a problem", "Add a problem set"} {
		if !strings.Contains(main.Body.String(), want) {
			t.Errorf("todo list missing %q", want)
		}
	}
	if strings.Contains(main.Body.String(), `action="/leetgrinder/todos/items"`) || strings.Contains(main.Body.String(), `action="/leetgrinder/todos/sets"`) {
		t.Fatal("entry forms still occupy the todo list")
	}
	for _, path := range []string{"/leetgrinder/todos/add", "/leetgrinder/todos/sets/new"} {
		page := request("GET", path, nil)
		if page.Code != 200 || !strings.Contains(page.Body.String(), "<form") {
			t.Errorf("entry page %s: %d", path, page.Code)
		}
	}
	invalid := request("POST", "/leetgrinder/todos/items", url.Values{"problem": {"https://example.com/bad"}})
	if invalid.Code != 400 || !strings.Contains(invalid.Body.String(), `action="/leetgrinder/todos/items"`) {
		t.Fatalf("invalid problem must stay on its form: %d", invalid.Code)
	}
	invalidSet := request("POST", "/leetgrinder/todos/sets", url.Values{"title": {""}})
	if invalidSet.Code != 400 || !strings.Contains(invalidSet.Body.String(), `action="/leetgrinder/todos/sets"`) {
		t.Fatalf("invalid set must stay on its form: %d", invalidSet.Code)
	}
}

func TestDashboardShowsNextFiveDistinctTodoProblems(t *testing.T) {
	_, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	for _, slug := range []string{"two-sum", "valid-parentheses", "three-sum", "merge-intervals", "clone-graph", "number-of-islands"} {
		if _, err := db.AddLeetgrinderTodoItem(ctx, "", slug); err != nil {
			t.Fatal(err)
		}
	}
	set, err := db.CreateLeetgrinderTodoSet(ctx, "Repeated", []string{"two-sum"})
	if err != nil || set.ID == "" {
		t.Fatalf("set: %+v %v", set, err)
	}
	page := request("GET", "/leetgrinder", nil)
	if page.Code != 200 {
		t.Fatalf("dashboard: %d", page.Code)
	}
	body := page.Body.String()
	start := strings.Index(body, `id="next-todos-title"`)
	if start < 0 {
		t.Fatal("dashboard missing next todos section")
	}
	section := body[start:]
	for _, slug := range []string{"two-sum", "valid-parentheses", "three-sum", "merge-intervals", "clone-graph"} {
		if !strings.Contains(section, "/leetgrinder/problem/"+slug) {
			t.Errorf("next five missing %s", slug)
		}
	}
	if strings.Contains(section, "/leetgrinder/problem/number-of-islands") || strings.Count(section, "/leetgrinder/problem/two-sum") != 1 {
		t.Fatal("dashboard did not show five distinct next problems")
	}
}

func TestDoneTodosLeaveTheQueueAndCountInSets(t *testing.T) {
	_, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	attempt := func(slug, outcome string) {
		t.Helper()
		a := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: slug, Outcome: outcome, Minutes: 20}
		if outcome != "unfinished" {
			a.TimeComplexity, a.SpaceComplexity = "O(n)", "O(1)"
		}
		if _, err := db.SaveLeetgrinderAttempt(ctx, a, ""); err != nil {
			t.Fatal(err)
		}
	}
	attempt("two-sum", "solved")
	if _, err := db.CreateLeetgrinderTodoSet(ctx, "Blind 75", []string{"two-sum", "valid-parentheses", "three-sum"}); err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{"merge-intervals", "clone-graph"} {
		if _, err := db.AddLeetgrinderTodoItem(ctx, "", slug); err != nil {
			t.Fatal(err)
		}
	}
	attempt("valid-parentheses", "struggled")
	attempt("three-sum", "unfinished")
	attempt("merge-intervals", "solved")

	sets, err := db.LeetgrinderTodoSets(ctx)
	if err != nil || len(sets) != 1 || len(sets[0].Items) != 3 || sets[0].Remaining() != 1 {
		t.Fatalf("set keeps done problems and counts the rest: %+v %v", sets, err)
	}
	for _, item := range sets[0].Items {
		if want := item.Problem.Slug != "three-sum"; item.Done() != want {
			t.Errorf("%s done=%v, want %v", item.Problem.Slug, item.Done(), want)
		}
	}
	standalone, err := db.LeetgrinderTodoItems(ctx, "")
	if err != nil || len(standalone) != 1 || standalone[0].Problem.Slug != "clone-graph" {
		t.Fatalf("done individual problem must leave the list: %+v %v", standalone, err)
	}

	page := request("GET", "/leetgrinder/todos", nil).Body.String()
	if !strings.Contains(page, "2 of 3 done") || !strings.Contains(page, "1 remaining") || strings.Contains(page, "/leetgrinder/problem/merge-intervals") {
		t.Fatalf("todo page progress: %s", page)
	}
	dashboard := request("GET", "/leetgrinder", nil).Body.String()
	section := dashboard[strings.Index(dashboard, `id="next-todos-title"`):]
	section = section[:strings.Index(section, "</section>")]
	for _, slug := range []string{"two-sum", "valid-parentheses", "merge-intervals"} {
		if strings.Contains(section, "/leetgrinder/problem/"+slug+`"`) {
			t.Errorf("dashboard queue shows done %s", slug)
		}
	}
	if !strings.Contains(section, "/leetgrinder/problem/three-sum") || !strings.Contains(section, "/leetgrinder/problem/clone-graph") {
		t.Fatal("dashboard queue lost open problems")
	}
	if !strings.Contains(section, `<span class="todo-card-set">Blind 75</span>`) || !strings.Contains(section, `<span class="todo-card-set">Individual</span>`) {
		t.Fatal("dashboard queue does not name each problem's set")
	}

	if _, err = db.AddLeetgrinderTodoItem(ctx, "", "merge-intervals"); err != nil {
		t.Fatal(err)
	}
	standalone, err = db.LeetgrinderTodoItems(ctx, "")
	if err != nil || len(standalone) != 2 || standalone[1].Problem.Slug != "merge-intervals" {
		t.Fatalf("adding a done problem again must queue it: %+v %v", standalone, err)
	}

	attempt("clone-graph", "unfinished")
	if _, err = db.AddLeetgrinderTodoItem(ctx, "", "clone-graph"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.AddLeetgrinderTodoItem(ctx, "", "two-sum"); err != nil {
		t.Fatal(err)
	}
	standalone, err = db.LeetgrinderTodoItems(ctx, "")
	if err != nil || len(standalone) != 3 || standalone[0].Problem.Slug != "clone-graph" || standalone[2].Problem.Slug != "two-sum" || standalone[2].Done() {
		t.Fatalf("open entries keep their place and earlier solves do not complete new entries: %+v %v", standalone, err)
	}

	attempt("three-sum", "solved")
	if page = request("GET", "/leetgrinder/todos", nil).Body.String(); !strings.Contains(page, "3 of 3 done") || !strings.Contains(page, "all finished") {
		t.Fatal("finished set must say so")
	}
}

func TestTodoSetPageSummarizesAndSearches(t *testing.T) {
	_, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	set, err := db.CreateLeetgrinderTodoSetDetailed(ctx, leetgrinder.TodoSet{Title: "Graphs and arrays"}, []leetgrinder.TodoProblemInput{
		{Slug: "two-sum", Number: 1, Title: "Two Sum", Difficulty: "Easy", Topics: []string{"array", "hash-table"}},
		{Slug: "number-of-islands", Number: 200, Title: "Number of Islands", Difficulty: "Medium", Topics: []string{"graph", "array"}},
		{Slug: "word-ladder", Number: 127, Title: "Word Ladder", Difficulty: "Hard", Topics: []string{"graph"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.SaveLeetgrinderAttempt(ctx, leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "solved", Minutes: 10, TimeComplexity: "O(n)", SpaceComplexity: "O(n)"}, ""); err != nil {
		t.Fatal(err)
	}
	base := "/leetgrinder/todos/sets/" + set.ID
	page := request("GET", base, nil)
	body := page.Body.String()
	if page.Code != 200 {
		t.Fatalf("set page: %d", page.Code)
	}
	for _, want := range []string{"1 of 3 done", "2 remaining", "Easy</span> 1/1", "Medium</span> 0/1", "Hard</span> 0/1", "Array</span> 1/2", "Graph</span> 0/2", `class="todo-check"`, `role="search"`} {
		if !strings.Contains(body, want) {
			t.Errorf("set page missing %q", want)
		}
	}
	if strings.Count(body, `class="todo-check"`) != 1 || strings.Count(body, `class="todo-open"`) != 2 {
		t.Error("each done problem needs one check and each open problem none")
	}
	search := func(query string) string {
		t.Helper()
		w := request("GET", base+"?"+query, nil)
		if w.Code != 200 {
			t.Fatalf("search %s: %d", query, w.Code)
		}
		body := w.Body.String()
		start := strings.Index(body, `role="status"`)
		if start < 0 {
			t.Fatalf("search %s: no result count", query)
		}
		return body[start:]
	}
	if _, err = db.AddLeetgrinderTodoItem(ctx, set.ID, "custom-untagged-problem"); err != nil {
		t.Fatal(err)
	}
	if body = request("GET", base, nil).Body.String(); !strings.Contains(body, "Unknown</span> 0/1") {
		t.Error("summary must count problems without a difficulty")
	}
	for query, want := range map[string]string{
		"q=island":                    "number-of-islands",
		"q=127":                       "word-ladder",
		"topic=graph&difficulty=Hard": "word-ladder",
		"status=done":                 "two-sum",
		"status=todo&topic=array":     "number-of-islands",
		"topic=untagged":              "custom-untagged-problem",
	} {
		results := search(query)
		for _, slug := range []string{"two-sum", "number-of-islands", "word-ladder", "custom-untagged-problem"} {
			if shown := strings.Contains(results, "/leetgrinder/problem/"+slug+`"`); shown != (slug == want) {
				t.Errorf("%s: %s shown=%v", query, slug, shown)
			}
		}
	}
	if results := search("q=nothing-matches"); !strings.Contains(results, "Showing 0 of 4") {
		t.Error("empty search must say nothing matched")
	}
	for id, want := range map[string]int{uuid.NewString(): 404, "not-a-set": 404, "urn:uuid:" + set.ID: 200} {
		if w := request("GET", "/leetgrinder/todos/sets/"+id, nil); w.Code != want {
			t.Errorf("set %s: %d, want %d", id, w.Code, want)
		}
	}
	if add := request("GET", "/leetgrinder/todos/add?setID="+set.ID, nil).Body.String(); !strings.Contains(add, `<a href="`+base+`">Graphs and arrays</a>`) {
		t.Error("add page must link back to its set")
	}

	if w := request("POST", "/leetgrinder/todos/items", url.Values{"problem": {"clone-graph"}, "setID": {set.ID}}); w.Code != 303 || w.Header().Get("Location") != base {
		t.Fatalf("adding to a set returns to it: %d %s", w.Code, w.Header().Get("Location"))
	}
	set, err = db.LeetgrinderTodoSet(ctx, set.ID)
	if err != nil || len(set.Items) != 5 {
		t.Fatalf("set after add: %+v %v", set, err)
	}
	if w := request("POST", "/leetgrinder/todos/items/"+set.Items[4].ID+"/delete", url.Values{"setID": {set.ID}}); w.Code != 303 || w.Header().Get("Location") != base {
		t.Fatalf("removing from a set returns to it: %d %s", w.Code, w.Header().Get("Location"))
	}
}

func TestTodoImportMetadataDoesNotGrowOnReAdd(t *testing.T) {
	_, db, _ := leetgrinderTestServer(t)
	ctx := context.Background()
	input := []leetgrinder.TodoProblemInput{{Slug: "two-sum", Title: "Two Sum", ImportMetadata: map[string]any{"sourceID": "a"}}}
	snapshots := func() int {
		p, err := db.LeetgrinderProblem(ctx, "two-sum")
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, versions := range p.ImportMetadata {
			n += len(versions.([]any))
		}
		return n
	}
	for i := 0; i < 3; i++ {
		set, err := db.CreateLeetgrinderTodoSetDetailed(ctx, leetgrinder.TodoSet{Title: "Cycle"}, input)
		if err != nil {
			t.Fatal(err)
		}
		if err = db.DeleteLeetgrinderTodoSet(ctx, set.ID); err != nil {
			t.Fatal(err)
		}
	}
	if n := snapshots(); n != 1 {
		t.Fatalf("identical snapshots archived %d times", n)
	}
	// A different snapshot is still kept.
	input[0].ImportMetadata = map[string]any{"sourceID": "b"}
	if _, err := db.CreateLeetgrinderTodoSetDetailed(ctx, leetgrinder.TodoSet{Title: "Other"}, input); err != nil {
		t.Fatal(err)
	}
	if n := snapshots(); n != 2 {
		t.Fatalf("changed snapshot not kept: %d", n)
	}
	// The archive stops at its size limit; saving the todo still works.
	input[0].ImportMetadata = map[string]any{"sourceID": "c"}
	if _, err := db.DB.ExecContext(ctx, `UPDATE leetgrinder_problems SET import_metadata=jsonb_build_object('pad', jsonb_build_array(repeat('x', 65500))) WHERE slug='two-sum'`); err != nil {
		t.Fatal(err)
	}
	full, err := db.CreateLeetgrinderTodoSetDetailed(ctx, leetgrinder.TodoSet{Title: "Full"}, input)
	if err != nil {
		t.Fatal(err)
	}
	var source string
	if err := db.DB.QueryRowContext(ctx, "SELECT source_data::text FROM leetgrinder_todo_items WHERE set_id=$1", full.ID).Scan(&source); err != nil || !strings.Contains(source, `"sourceID": "c"`) {
		t.Fatalf("the todo's own source_data was not saved: %q %v", source, err)
	}
	if p, _ := db.LeetgrinderProblem(ctx, "two-sum"); len(p.ImportMetadata) != 1 {
		t.Fatalf("archive grew past its limit: %v", p.ImportMetadata)
	}
}
