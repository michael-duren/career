package leetgrinder

import (
	"strings"
	"time"
)

type TodoItem struct {
	ID      string
	SetID   string
	Problem Problem
	// SourceData is the exact imported problem detail snapshot for this todo.
	SourceData map[string]any
	// DoneAt is the latest solved or struggled attempt that completes this
	// entry, zero while it is still to do.
	DoneAt time.Time
}

func (i TodoItem) Done() bool { return !i.DoneAt.IsZero() }

type TodoSet struct {
	ID          string
	Title       string
	Description string
	Metadata    map[string]any
	Items       []TodoItem
}

// Remaining counts the set's problems still to do.
func (s TodoSet) Remaining() int {
	n := 0
	for _, item := range s.Items {
		if !item.Done() {
			n++
		}
	}
	return n
}

// TodoProblemInput is a problem copied into a todo set. ImportMetadata holds
// source-specific fields that do not have a catalog column.
type TodoProblemInput struct {
	Slug           string
	Reference      string
	Number         int
	Title          string
	Difficulty     string
	Topics         []string
	ImportMetadata map[string]any
}

type TodosPage struct {
	Sets       []TodoSet
	Standalone []TodoItem
	// Location formats done dates in the settings time zone.
	Location *time.Location
	Error    string
	Title    string
	Problems string
	Problem  string
	SetID    string
}

// ParseTodoRefs accepts one LeetCode link or slug per line, or comma-separated entries.
func ParseTodoRefs(input string) ([]string, bool) {
	if strings.TrimSpace(input) == "" {
		return []string{}, true
	}
	parts := strings.FieldsFunc(input, func(r rune) bool { return r == '\n' || r == ',' || r == '\r' })
	if len(parts) > 200 {
		return nil, false
	}
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		slug, ok := NormalizeProblemRef(part)
		if !ok {
			return nil, false
		}
		if !seen[slug] {
			out = append(out, slug)
			seen[slug] = true
		}
	}
	return out, true
}
