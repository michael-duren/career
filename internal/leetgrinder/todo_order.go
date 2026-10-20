package leetgrinder

import (
	"cmp"
	"errors"
	"slices"
	"strings"
)

// MaxTodoSetProblems is how many problems one set may be created with.
const MaxTodoSetProblems = 1000

// TodoOrderError names entry IDs a reorder refused because the set lacks
// them, either because they do not exist or belong to another set.
type TodoOrderError struct {
	SetID string
	IDs   []string
}

func (e TodoOrderError) Error() string {
	ids := make([]string, len(e.IDs))
	for i, id := range e.IDs {
		ids[i] = EchoRef(id)
	}
	return "not entries of set " + e.SetID + ": " + strings.Join(ids, ", ")
}

// ErrInvalidTodoMove is returned for a move without exactly one target, or
// one placed relative to itself.
var ErrInvalidTodoMove = errors.New("each move needs an id and exactly one of beforeId, afterId or toIndex, and may not name its own entry")

// TodoMove moves one entry before or after another entry, or to a 0-based
// index counted after the entry is taken out. Exactly one target is set.
type TodoMove struct {
	ID       string
	BeforeID string
	AfterID  string
	ToIndex  *int
}

// TodoSortKey sorts entries by one field. Field is title, number, difficulty,
// done, slug, or metadata.<key> for imported problem metadata. With Values,
// entries whose field equals a listed value come first, in the listed order,
// and the rest follow in their current order. Without Values, entries sort
// by the field's value, and entries lacking it go last.
type TodoSortKey struct {
	Field      string
	Values     []any
	Descending bool
}

// OrderTodosFront moves the listed entries to the front in the given order;
// the others keep their relative order after them. Repeated IDs are ignored.
func OrderTodosFront(setID string, items []TodoItem, ids []string) ([]string, error) {
	index := todoIndex(items)
	if err := unknownTodoIDs(setID, index, ids); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	listed := map[string]bool{}
	for _, id := range ids {
		if !listed[id] {
			listed[id] = true
			out = append(out, id)
		}
	}
	for _, item := range items {
		if !listed[item.ID] {
			out = append(out, item.ID)
		}
	}
	return out, nil
}

// MoveTodos applies each move in turn to the current order.
func MoveTodos(setID string, items []TodoItem, moves []TodoMove) ([]string, error) {
	index := todoIndex(items)
	named := []string{}
	for _, m := range moves {
		targets := 0
		for _, set := range []bool{m.BeforeID != "", m.AfterID != "", m.ToIndex != nil} {
			if set {
				targets++
			}
		}
		if m.ID == "" || targets != 1 || m.ID == m.BeforeID || m.ID == m.AfterID {
			return nil, ErrInvalidTodoMove
		}
		named = append(named, m.ID, m.BeforeID, m.AfterID)
	}
	if err := unknownTodoIDs(setID, index, named); err != nil {
		return nil, err
	}
	order := make([]string, len(items))
	for i, item := range items {
		order[i] = item.ID
	}
	for _, m := range moves {
		i := slices.Index(order, m.ID)
		order = slices.Delete(order, i, i+1)
		at := 0
		switch {
		case m.BeforeID != "":
			at = slices.Index(order, m.BeforeID)
		case m.AfterID != "":
			at = slices.Index(order, m.AfterID) + 1
		default:
			at = min(max(*m.ToIndex, 0), len(order))
		}
		order = slices.Insert(order, at, m.ID)
	}
	return order, nil
}

// SortTodos stably sorts entries by each key in turn.
func SortTodos(items []TodoItem, keys []TodoSortKey) []string {
	sorted := slices.Clone(items)
	slices.SortStableFunc(sorted, func(a, b TodoItem) int {
		for _, key := range keys {
			if c := key.compare(a, b); c != 0 {
				return c
			}
		}
		return 0
	})
	out := make([]string, len(sorted))
	for i, item := range sorted {
		out[i] = item.ID
	}
	return out
}

func (k TodoSortKey) compare(a, b TodoItem) int {
	av, aok := a.Field(k.Field)
	bv, bok := b.Field(k.Field)
	if len(k.Values) > 0 {
		return cmp.Compare(k.rank(av, aok), k.rank(bv, bok))
	}
	switch {
	case !aok && !bok:
		return 0
	case !aok:
		return 1
	case !bok:
		return -1
	}
	c := compareTodoValues(av, bv)
	if k.Descending {
		return -c
	}
	return c
}

// rank is the value's index in Values, or len(Values) when it is not listed.
func (k TodoSortKey) rank(v any, ok bool) int {
	if ok {
		for i, want := range k.Values {
			if scalarEqual(v, want) {
				return i
			}
		}
	}
	return len(k.Values)
}

// Field returns a field of the entry for sorting and filtering: title,
// number, difficulty, done, slug, or metadata.<key> from the imported
// problem metadata. A missing or empty field reports false.
func (i TodoItem) Field(name string) (any, bool) {
	switch name {
	case "title":
		return i.Problem.Title, i.Problem.Title != ""
	case "number":
		return float64(i.Problem.Number), i.Problem.Number != 0
	case "difficulty":
		return i.Problem.Difficulty, i.Problem.Difficulty != ""
	case "done":
		return i.Done(), true
	case "slug":
		return i.Problem.Slug, true
	}
	key, ok := strings.CutPrefix(name, "metadata.")
	if !ok {
		return nil, false
	}
	metadata, _ := i.SourceData["metadata"].(map[string]any)
	v, ok := metadata[key]
	return v, ok && v != nil
}

// TodoSortFieldValid reports whether Field knows the name.
func TodoSortFieldValid(name string) bool {
	switch name {
	case "title", "number", "difficulty", "done", "slug":
		return true
	}
	key, ok := strings.CutPrefix(name, "metadata.")
	return ok && key != ""
}

// compareTodoValues orders numbers, then strings, then booleans with true
// first, then anything else, which compares equal.
func compareTodoValues(a, b any) int {
	if c := cmp.Compare(todoValueKind(a), todoValueKind(b)); c != 0 {
		return c
	}
	switch av := a.(type) {
	case float64:
		return cmp.Compare(av, b.(float64))
	case string:
		return cmp.Compare(strings.ToLower(av), strings.ToLower(b.(string)))
	case bool:
		if av == b.(bool) {
			return 0
		} else if av {
			return -1
		}
		return 1
	}
	return 0
}

func todoValueKind(v any) int {
	switch v.(type) {
	case float64:
		return 0
	case string:
		return 1
	case bool:
		return 2
	}
	return 3
}

// scalarEqual compares JSON scalars; objects and arrays never match.
func scalarEqual(a, b any) bool {
	switch a.(type) {
	case float64, string, bool:
		return a == b
	}
	return false
}

func todoIndex(items []TodoItem) map[string]bool {
	index := make(map[string]bool, len(items))
	for _, item := range items {
		index[item.ID] = true
	}
	return index
}

// unknownTodoIDs names the non-empty IDs the set lacks, each once.
func unknownTodoIDs(setID string, index map[string]bool, ids []string) error {
	var unknown []string
	for _, id := range ids {
		if id != "" && !index[id] && !slices.Contains(unknown, id) {
			unknown = append(unknown, id)
		}
	}
	if len(unknown) > 0 {
		return TodoOrderError{SetID: setID, IDs: unknown}
	}
	return nil
}
