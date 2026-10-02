package leetgrinder

import (
	"cmp"
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
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

// TodoTally counts a set's problems in one difficulty or topic.
type TodoTally struct {
	Key, Label  string
	Total, Done int
}

// TodoSummary describes a whole set, whatever the search shows.
type TodoSummary struct {
	Total, Done int
	// Difficulties lists Easy, Medium, Hard and Unknown, leaving out any
	// with no problems.
	Difficulties []TodoTally
	// Topics lists topic tags, most common first.
	Topics []TodoTally
}

func (s TodoSummary) Remaining() int { return s.Total - s.Done }

// Summary tallies the set's progress by difficulty and topic.
func (s TodoSet) Summary() TodoSummary {
	sum := TodoSummary{Total: len(s.Items)}
	difficulties := []TodoTally{{Key: "Easy", Label: "Easy"}, {Key: "Medium", Label: "Medium"}, {Key: "Hard", Label: "Hard"}, {Key: "", Label: "Unknown"}}
	topics := map[string]*TodoTally{}
	count := func(t *TodoTally, done bool) {
		t.Total++
		if done {
			t.Done++
		}
	}
	for _, item := range s.Items {
		done := item.Done()
		if done {
			sum.Done++
		}
		for i := range difficulties {
			if difficulties[i].Key == item.Problem.Difficulty {
				count(&difficulties[i], done)
			}
		}
		for _, topic := range item.Problem.Topics {
			if topics[topic] == nil {
				topics[topic] = &TodoTally{Key: topic, Label: TopicLabel(topic)}
			}
			count(topics[topic], done)
		}
	}
	for _, d := range difficulties {
		if d.Total > 0 {
			sum.Difficulties = append(sum.Difficulties, d)
		}
	}
	for _, t := range topics {
		sum.Topics = append(sum.Topics, *t)
	}
	slices.SortFunc(sum.Topics, func(a, b TodoTally) int {
		return cmp.Or(cmp.Compare(b.Total, a.Total), cmp.Compare(a.Label, b.Label))
	})
	return sum
}

// TodoFilter holds a set page's search form; zero values match everything.
type TodoFilter struct {
	Query      string
	Difficulty string
	Topic      string
	// Status is "todo", "done" or "".
	Status string
}

// ParseTodoFilter reads the search form, ignoring values it does not know.
func ParseTodoFilter(q url.Values) TodoFilter {
	p := ParseProblemFilter(q)
	f := TodoFilter{Query: p.Query, Difficulty: p.Difficulty, Topic: p.Topic}
	if s := q.Get("status"); s == "todo" || s == "done" {
		f.Status = s
	}
	return f
}

// Active reports whether any filter narrows the list.
func (f TodoFilter) Active() bool {
	return f.Query != "" || f.Difficulty != "" || f.Topic != "" || f.Status != ""
}

// FilterTodos returns the items the filter matches, in their saved order.
func FilterTodos(items []TodoItem, f TodoFilter) []TodoItem {
	out := []TodoItem{}
	for _, item := range items {
		if matchProblem(item.Problem, f.Query, f.Difficulty, f.Topic) && (f.Status == "" || (f.Status == "done") == item.Done()) {
			out = append(out, item)
		}
	}
	return out
}

// TodoSetPage is one set with its summary and the problems the search shows.
type TodoSetPage struct {
	Set      TodoSet
	Summary  TodoSummary
	Items    []TodoItem
	Filter   TodoFilter
	Topics   []FilterOption
	Location *time.Location
}

func NewTodoSetPage(set TodoSet, f TodoFilter, loc *time.Location) TodoSetPage {
	problems := make([]Problem, len(set.Items))
	for i, item := range set.Items {
		problems[i] = item.Problem
	}
	return TodoSetPage{Set: set, Summary: set.Summary(), Items: FilterTodos(set.Items, f), Filter: f, Topics: topicOptions(problems), Location: loc}
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

// MaxTodoRefs is how many problems one pasted list may hold.
const MaxTodoRefs = 200

// ErrTooManyTodoRefs is returned for a list longer than MaxTodoRefs.
var ErrTooManyTodoRefs = errors.New("too many todo problems")

// InvalidTodoRefError carries the first entry that is not a LeetCode problem.
type InvalidTodoRefError struct{ Entry string }

func (e InvalidTodoRefError) Error() string { return "invalid todo problem " + strconv.Quote(e.Entry) }

// ParseTodoRefs accepts LeetCode links or slugs separated by new lines,
// commas, spaces or tabs. Empty entries are ignored. The error names the first
// entry that is not a LeetCode problem.
func ParseTodoRefs(input string) ([]string, error) {
	parts := strings.FieldsFunc(input, func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
	if len(parts) > MaxTodoRefs {
		return nil, ErrTooManyTodoRefs
	}
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		slug, ok := NormalizeProblemRef(part)
		if !ok {
			// Keep the echoed entry short, cutting on a rune boundary.
			if r := []rune(part); len(r) > 60 {
				part = string(r[:60]) + "..."
			}
			return nil, InvalidTodoRefError{Entry: part}
		}
		if !seen[slug] {
			out = append(out, slug)
			seen[slug] = true
		}
	}
	return out, nil
}
