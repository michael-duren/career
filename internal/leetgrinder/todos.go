package leetgrinder

import "strings"

type TodoItem struct {
	ID      string
	SetID   string
	Problem Problem
	// SourceData is the exact imported problem detail snapshot for this todo.
	SourceData map[string]any
}

type TodoSet struct {
	ID          string
	Title       string
	Description string
	Metadata    map[string]any
	Items       []TodoItem
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
	Error      string
	Title      string
	Problems   string
	Problem    string
	SetID      string
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
