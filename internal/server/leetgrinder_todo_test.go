package server

import (
	"context"
	"net/url"
	"strings"
	"testing"
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
