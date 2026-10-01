package server

import (
	"context"
	"net/url"
	"strings"
	"testing"

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
	if w := request("POST", "/leetgrinder/todos/items", url.Values{"problem": {"three-sum"}}); w.Code != 303 {
		t.Fatalf("add standalone: %d %s", w.Code, w.Body.String())
	}
	if w := request("POST", "/leetgrinder/todos/items", url.Values{"problem": {"merge-intervals"}, "setID": {sets[0].ID}}); w.Code != 303 {
		t.Fatalf("add to set: %d %s", w.Code, w.Body.String())
	}
	page := request("GET", "/leetgrinder/todos", nil)
	if page.Code != 200 || !strings.Contains(page.Body.String(), "Interview prep") || !strings.Contains(page.Body.String(), "three-sum") || !strings.Contains(page.Body.String(), "merge-intervals") {
		t.Fatalf("todo page: %d %s", page.Code, page.Body.String())
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
