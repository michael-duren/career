package leetgrinder

import (
	"errors"
	"slices"
	"testing"
)

func orderItems(ids ...string) []TodoItem {
	items := make([]TodoItem, len(ids))
	for i, id := range ids {
		items[i] = TodoItem{ID: id, Position: i}
	}
	return items
}

func TestOrderTodosFront(t *testing.T) {
	items := orderItems("a", "b", "c", "d")
	got, err := OrderTodosFront("set", items, []string{"c", "a", "c"})
	if err != nil || !slices.Equal(got, []string{"c", "a", "b", "d"}) {
		t.Fatal(got, err)
	}
	_, err = OrderTodosFront("set", items, []string{"a", "x", "y", "x"})
	var unknown TodoOrderError
	if !errors.As(err, &unknown) || !slices.Equal(unknown.IDs, []string{"x", "y"}) {
		t.Fatal("unknown IDs not named", err)
	}
}

func TestMoveTodos(t *testing.T) {
	items := orderItems("a", "b", "c", "d")
	index := func(i int) *int { return &i }
	for _, tc := range []struct {
		moves []TodoMove
		want  []string
	}{
		{[]TodoMove{{ID: "d", BeforeID: "a"}}, []string{"d", "a", "b", "c"}},
		{[]TodoMove{{ID: "a", AfterID: "c"}}, []string{"b", "c", "a", "d"}},
		{[]TodoMove{{ID: "a", ToIndex: index(3)}}, []string{"b", "c", "d", "a"}},
		{[]TodoMove{{ID: "b", ToIndex: index(99)}}, []string{"a", "c", "d", "b"}},
		{[]TodoMove{{ID: "d", ToIndex: index(0)}, {ID: "c", AfterID: "d"}}, []string{"d", "c", "a", "b"}},
	} {
		got, err := MoveTodos("set", items, tc.moves)
		if err != nil || !slices.Equal(got, tc.want) {
			t.Errorf("%+v: got %v %v, want %v", tc.moves, got, err, tc.want)
		}
	}
	for _, bad := range []TodoMove{{ID: "a"}, {ID: "a", BeforeID: "b", AfterID: "c"}, {ID: "a", BeforeID: "a"}, {BeforeID: "a"}} {
		if _, err := MoveTodos("set", items, []TodoMove{bad}); !errors.Is(err, ErrInvalidTodoMove) {
			t.Errorf("%+v accepted: %v", bad, err)
		}
	}
	var unknown TodoOrderError
	if _, err := MoveTodos("set", items, []TodoMove{{ID: "a", BeforeID: "z"}}); !errors.As(err, &unknown) || unknown.IDs[0] != "z" {
		t.Fatal("unknown target not named", err)
	}
}

func TestSortTodos(t *testing.T) {
	item := func(id string, metadata map[string]any) TodoItem {
		return TodoItem{ID: id, SourceData: map[string]any{"metadata": metadata}}
	}
	items := []TodoItem{
		item("arrays", map[string]any{"pattern": "Arrays", "neetcode150": true, "positionInPattern": 1.0}),
		item("graph-2", map[string]any{"pattern": "Graphs", "neetcode150": false, "positionInPattern": 1.0}),
		item("trees-2", map[string]any{"pattern": "Trees", "neetcode150": true, "positionInPattern": 2.0}),
		item("graph-1", map[string]any{"pattern": "Graphs", "neetcode150": true, "positionInPattern": 3.0}),
		item("trees-1", map[string]any{"pattern": "Trees", "neetcode150": true, "positionInPattern": 1.0}),
		item("bare", nil),
	}
	got := SortTodos(items, []TodoSortKey{
		{Field: "metadata.pattern", Values: []any{"Trees", "Graphs"}},
		{Field: "metadata.neetcode150", Values: []any{true}},
		{Field: "metadata.positionInPattern"},
	})
	want := []string{"trees-1", "trees-2", "graph-1", "graph-2", "arrays", "bare"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	got = SortTodos(items, []TodoSortKey{{Field: "metadata.positionInPattern", Descending: true}})
	if got[0] != "graph-1" || got[len(got)-1] != "bare" {
		t.Fatal("descending sort should keep entries without the field last", got)
	}
}
