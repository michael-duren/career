package server

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type createTodoSetInput struct {
	Title          string                    `json:"title" jsonschema:"name of the new problem set, 1-120 characters"`
	Description    string                    `json:"description,omitempty" jsonschema:"optional set description, up to 2000 characters"`
	Metadata       map[string]any            `json:"metadata,omitempty" jsonschema:"optional source-specific set data to preserve"`
	Problems       []string                  `json:"problems,omitempty" jsonschema:"LeetCode or NeetCode links, or LeetCode slugs, without metadata"`
	ProblemDetails []todoProblemDetailsInput `json:"problemDetails,omitempty" jsonschema:"problems with catalog fields and arbitrary source metadata; use to copy a complete set"`
}

type todoProblemDetailsInput struct {
	Problem    string         `json:"problem" jsonschema:"LeetCode or NeetCode problem link, or LeetCode slug"`
	Number     int            `json:"number,omitempty" jsonschema:"optional LeetCode frontend number"`
	Title      string         `json:"title,omitempty" jsonschema:"optional problem title"`
	Difficulty string         `json:"difficulty,omitempty" jsonschema:"Easy, Medium, or Hard"`
	Topics     []string       `json:"topics,omitempty" jsonschema:"LeetCode topic slugs such as dynamic-programming"`
	Metadata   map[string]any `json:"metadata,omitempty" jsonschema:"other source-specific problem data to preserve on the problem"`
}

type addTodoProblemInput struct {
	todoProblemDetailsInput
	SetID    string `json:"setID,omitempty" jsonschema:"optional problem set ID; omit for the individual todo list"`
	Position *int   `json:"position,omitempty" jsonschema:"optional 0-based index to place the problem at in the set; omit to add it last"`
	BeforeID string `json:"beforeId,omitempty" jsonschema:"optional set entry ID to place the problem before"`
	AfterID  string `json:"afterId,omitempty" jsonschema:"optional set entry ID to place the problem after"`
}

type listTodosInput struct {
	SetID   string `json:"setID,omitempty" jsonschema:"optional set ID to list only that set; individual problems are then left out"`
	Fields  string `json:"fields,omitempty" jsonschema:"full (default) or compact; compact problems carry only id, title, number, difficulty, done, position, pattern and neetcode150"`
	Done    *bool  `json:"done,omitempty" jsonschema:"optional: only problems with this done status"`
	Pattern string `json:"pattern,omitempty" jsonschema:"optional: only problems whose metadata.pattern matches, ignoring case"`
	Limit   int    `json:"limit,omitempty" jsonschema:"optional: at most this many problems per listed set and in individualProblems"`
	Cursor  string `json:"cursor,omitempty" jsonschema:"nextCursor from an earlier call with the same setID and filters"`
}

type reorderTodoSetInput struct {
	SetID      string          `json:"setID" jsonschema:"problem set ID"`
	OrderedIDs []string        `json:"orderedIds,omitempty" jsonschema:"entry IDs to put first, in this order; unlisted entries keep their relative order after them; repeated IDs are ignored"`
	Moves      []todoMoveInput `json:"moves,omitempty" jsonschema:"targeted moves applied in turn"`
	OrderBy    []todoSortInput `json:"orderBy,omitempty" jsonschema:"sort keys applied in turn; ties keep the current order"`
}

type todoMoveInput struct {
	ID       string `json:"id" jsonschema:"entry ID to move"`
	BeforeID string `json:"beforeId,omitempty" jsonschema:"place the entry before this entry"`
	AfterID  string `json:"afterId,omitempty" jsonschema:"place the entry after this entry"`
	ToIndex  *int   `json:"toIndex,omitempty" jsonschema:"place the entry at this 0-based index, counted with the entry taken out"`
}

type todoSortInput struct {
	Field      string `json:"field" jsonschema:"title, number, difficulty, done, slug, or metadata.<key> such as metadata.pattern"`
	Values     []any  `json:"values,omitempty" jsonschema:"optional ranking: entries with these values come first in this order, then the rest"`
	Descending bool   `json:"descending,omitempty" jsonschema:"without values, sort largest first; entries lacking the field still go last"`
}

// Limits on one reorder request.
const (
	maxTodoReorderIDs    = 2 * leetgrinder.MaxTodoSetProblems
	maxTodoSortKeys      = 10
	maxTodoSortValues    = 200
	todoReorderPreviewed = 20
)

func todoProblemFromMCP(in todoProblemDetailsInput) (leetgrinder.TodoProblemInput, error) {
	slug, err := leetgrinder.ResolveProblemRef(in.Problem)
	if err != nil {
		return leetgrinder.TodoProblemInput{}, refInputError(in.Problem, err)
	}
	return leetgrinder.TodoProblemInput{Slug: slug, Reference: in.Problem, Number: in.Number, Title: in.Title, Difficulty: in.Difficulty, Topics: in.Topics, ImportMetadata: in.Metadata}, nil
}

// refInputError explains a refused problem reference to the MCP client.
func refInputError(ref string, err error) error {
	if errors.Is(err, leetgrinder.ErrUnknownNeetCodeProblem) {
		return invalidInput("unknown NeetCode problem %q: no LeetCode match is known; use its LeetCode link or slug", leetgrinder.EchoRef(ref))
	}
	return invalidInput("invalid LeetCode or NeetCode problem link, or LeetCode slug: %q", leetgrinder.EchoRef(ref))
}

type removeTodoInput struct {
	ID string `json:"id" jsonschema:"todo item or set ID from list_leetgrinder_todos"`
}

func (s *Server) addLeetgrinderTodoTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_leetgrinder_todos", Description: "List saved Leetgrinder problem sets and individual todo problems, including IDs for editing. Set problems are in the set's saved order, with their 0-based position. Problem entries include a LeetCode url and neetcodeUrl when a match is known. A problem is done once a solved or struggled attempt is logged: set problems stay listed with done=true and count against remainingCount; individual problems leave the list once done. " +
			"Large sets: pass setID to list one set, fields=compact to drop sourceProblem and full metadata, done and pattern to filter, and limit with cursor to page; problemCount and remainingCount always cover the whole set, matchingCount counts the filtered problems, and nextCursor is set while more remain.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listTodosInput) (*mcp.CallToolResult, any, error) {
		view, err := todoListViewFromMCP(in)
		if err != nil {
			return nil, nil, err
		}
		var sets []leetgrinder.TodoSet
		var standalone []leetgrinder.TodoItem
		if in.SetID != "" {
			set, err := s.db.LeetgrinderTodoSet(ctx, in.SetID)
			if err != nil {
				return nil, nil, todoToolError(err)
			}
			sets = []leetgrinder.TodoSet{set}
		} else {
			if sets, err = s.db.LeetgrinderTodoSets(ctx); err != nil {
				return nil, nil, toolError(err)
			}
			if standalone, err = s.db.LeetgrinderTodoItems(ctx, ""); err != nil {
				return nil, nil, toolError(err)
			}
		}
		listedSets := make([]map[string]any, 0, len(sets))
		for _, set := range sets {
			listed := map[string]any{"id": set.ID, "title": set.Title, "description": set.Description, "metadata": set.Metadata, "problemCount": len(set.Items), "remainingCount": set.Remaining()}
			view.list(listed, set.Items, true)
			listedSets = append(listedSets, listed)
		}
		out := map[string]any{"sets": listedSets}
		if in.SetID == "" {
			individual := map[string]any{}
			view.list(individual, standalone, false)
			out["individualProblems"] = individual["problems"]
			if n, ok := individual["matchingCount"]; ok {
				out["individualMatchingCount"] = n
			}
			if cursor, ok := individual["nextCursor"]; ok {
				out["individualNextCursor"] = cursor
			}
		}
		return nil, out, nil
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "create_leetgrinder_todo_set", Description: "Create a named Leetgrinder todo set with up to 1000 problems, kept in the order given. Use problemDetails to copy title, number, difficulty, topics and arbitrary metadata onto each catalog problem; use problems for plain links or slugs (leetcode.com, leetcode.cn or neetcode.io problem links, or LeetCode slugs; a NeetCode link is mapped to its LeetCode slug, and an unknown NeetCode problem is rejected). Set description and metadata are also preserved. Requires edit access.",
		Annotations: &mcp.ToolAnnotations{Title: "Create Leetgrinder problem set", DestructiveHint: new(bool), OpenWorldHint: new(bool)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in createTodoSetInput) (*mcp.CallToolResult, any, error) {
		if !canWrite(req) {
			return nil, nil, errNeedsWrite
		}
		if len(in.Problems)+len(in.ProblemDetails) > leetgrinder.MaxTodoSetProblems {
			return nil, nil, invalidInput("provide at most %d problems", leetgrinder.MaxTodoSetProblems)
		}
		problems := make([]leetgrinder.TodoProblemInput, 0, len(in.Problems)+len(in.ProblemDetails))
		seen := map[string]int{}
		for _, ref := range in.Problems {
			slug, err := leetgrinder.ResolveProblemRef(ref)
			if err != nil {
				return nil, nil, refInputError(ref, err)
			}
			if _, exists := seen[slug]; !exists {
				seen[slug] = len(problems)
				problems = append(problems, leetgrinder.TodoProblemInput{Slug: slug, Reference: ref})
			}
		}
		detailed := map[string]bool{}
		for _, input := range in.ProblemDetails {
			problem, err := todoProblemFromMCP(input)
			if err != nil {
				return nil, nil, err
			}
			if detailed[problem.Slug] {
				return nil, nil, invalidInput("duplicate problem detail: %s", problem.Slug)
			}
			detailed[problem.Slug] = true
			if index, exists := seen[problem.Slug]; exists {
				problems[index] = problem
			} else {
				seen[problem.Slug] = len(problems)
				problems = append(problems, problem)
			}
		}
		set, err := s.db.CreateLeetgrinderTodoSetDetailed(ctx, leetgrinder.TodoSet{Title: in.Title, Description: in.Description, Metadata: in.Metadata}, problems)
		if err != nil {
			return nil, nil, todoToolError(err)
		}
		return nil, map[string]any{"id": set.ID, "title": set.Title, "problemCount": len(problems)}, nil
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "add_leetgrinder_todo_problem", Description: "Add a LeetCode problem to the individual list or a set. The problem may be a leetcode.com, leetcode.cn or neetcode.io problem link, or a LeetCode slug; a NeetCode link is mapped to its LeetCode slug, and an unknown NeetCode problem is rejected. Optional title, number, difficulty, topics, and arbitrary metadata are saved on the catalog problem. A new set entry goes last unless position, beforeId or afterId places it; these also move an entry already in the set. Repeated additions keep one todo entry; adding a done individual problem again queues it for another pass. Requires edit access.",
		Annotations: &mcp.ToolAnnotations{Title: "Add Leetgrinder todo problem", DestructiveHint: new(bool), IdempotentHint: true, OpenWorldHint: new(bool)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in addTodoProblemInput) (*mcp.CallToolResult, any, error) {
		if !canWrite(req) {
			return nil, nil, errNeedsWrite
		}
		place := leetgrinder.TodoMove{BeforeID: in.BeforeID, AfterID: in.AfterID, ToIndex: in.Position}
		if placed := btoi(in.Position != nil) + btoi(in.BeforeID != "") + btoi(in.AfterID != ""); placed > 1 {
			return nil, nil, invalidInput("pass at most one of position, beforeId and afterId")
		} else if placed == 1 && in.SetID == "" {
			return nil, nil, invalidInput("position, beforeId and afterId need a setID; individual problems stay in the order added")
		}
		problem, err := todoProblemFromMCP(in.todoProblemDetailsInput)
		if err != nil {
			return nil, nil, err
		}
		item, err := s.db.AddLeetgrinderTodoProblemAt(ctx, in.SetID, problem, place)
		if err != nil {
			return nil, nil, todoToolError(err)
		}
		out := map[string]any{"id": item.ID, "setID": item.SetID, "problemSlug": problem.Slug}
		if item.SetID != "" {
			out["position"] = item.Position
		}
		return nil, out, nil
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "reorder_leetgrinder_set", Description: "Change the saved order of a Leetgrinder problem set; done status, attempt history and metadata stay as they are. Pass exactly one of: orderedIds, entry IDs to put first in that order (the rest keep their relative order after them, so a subset can be moved to the front without listing every ID; repeated IDs are ignored); moves, targeted moves applied in turn, each with id and one of beforeId, afterId or toIndex; or orderBy, sort keys applied in turn with ties keeping the current order, " +
			`for example [{"field":"metadata.pattern","values":["Trees","Graphs"]},{"field":"metadata.neetcode150","values":[true]},{"field":"metadata.positionInPattern"}]. ` +
			"IDs not in the set are rejected by name, and the whole reorder applies or none of it does. Returns how many entries moved and the first 20 entries in the new order. Requires edit access.",
		Annotations: &mcp.ToolAnnotations{Title: "Reorder Leetgrinder problem set", DestructiveHint: new(bool), IdempotentHint: true, OpenWorldHint: new(bool)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in reorderTodoSetInput) (*mcp.CallToolResult, any, error) {
		if !canWrite(req) {
			return nil, nil, errNeedsWrite
		}
		order, err := todoOrderFromMCP(in)
		if err != nil {
			return nil, nil, err
		}
		items, moved, err := s.db.ReorderLeetgrinderTodoSet(ctx, in.SetID, order)
		if err != nil {
			return nil, nil, todoToolError(err)
		}
		preview := make([]map[string]any, 0, todoReorderPreviewed)
		for _, item := range items[:min(len(items), todoReorderPreviewed)] {
			preview = append(preview, map[string]any{"id": item.ID, "title": item.Problem.Title, "done": item.Done(), "position": item.Position})
		}
		return nil, map[string]any{"setID": in.SetID, "moved": moved, "problemCount": len(items), "problems": preview}, nil
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "remove_leetgrinder_todo_problem", Description: "Remove one todo entry by its ID from list_leetgrinder_todos. Attempt history remains. Requires edit access.",
		Annotations: &mcp.ToolAnnotations{Title: "Remove Leetgrinder todo problem", DestructiveHint: &deleteIsDestructive, OpenWorldHint: new(bool)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in removeTodoInput) (*mcp.CallToolResult, any, error) {
		if !canWrite(req) {
			return nil, nil, errNeedsWrite
		}
		if err := s.db.DeleteLeetgrinderTodoItem(ctx, in.ID); err != nil {
			return nil, nil, todoToolError(err)
		}
		return nil, map[string]any{"removed": true, "id": in.ID}, nil
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "remove_leetgrinder_todo_set", Description: "Remove a named todo set and its todo entries by set ID. Attempt history remains. Requires edit access.",
		Annotations: &mcp.ToolAnnotations{Title: "Remove Leetgrinder problem set", DestructiveHint: &deleteIsDestructive, OpenWorldHint: new(bool)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in removeTodoInput) (*mcp.CallToolResult, any, error) {
		if !canWrite(req) {
			return nil, nil, errNeedsWrite
		}
		if err := s.db.DeleteLeetgrinderTodoSet(ctx, in.ID); err != nil {
			return nil, nil, todoToolError(err)
		}
		return nil, map[string]any{"removed": true, "id": in.ID}, nil
	})
}

// todoListView is how list_leetgrinder_todos filters, pages and shapes
// problems.
type todoListView struct {
	compact  bool
	done     *bool
	pattern  string
	offset   int
	limit    int
	filtered bool
}

func todoListViewFromMCP(in listTodosInput) (todoListView, error) {
	v := todoListView{done: in.Done, pattern: strings.TrimSpace(in.Pattern), limit: in.Limit}
	switch in.Fields {
	case "", "full":
	case "compact":
		v.compact = true
	default:
		return v, invalidInput("fields must be full or compact")
	}
	if in.Limit < 0 {
		return v, invalidInput("limit must be positive")
	}
	if in.Cursor != "" {
		offset, err := strconv.Atoi(in.Cursor)
		if err != nil || offset < 0 {
			return v, invalidInput("invalid cursor; pass nextCursor from an earlier call")
		}
		v.offset = offset
	}
	v.filtered = v.done != nil || v.pattern != "" || v.limit > 0 || v.offset > 0
	return v, nil
}

// list sets out's problems, and with any filter or page its matchingCount and
// nextCursor.
func (v todoListView) list(out map[string]any, items []leetgrinder.TodoItem, inSet bool) {
	matching := make([]leetgrinder.TodoItem, 0, len(items))
	for _, item := range items {
		pattern, _ := item.Field("metadata.pattern")
		patternText, _ := pattern.(string)
		if (v.done == nil || *v.done == item.Done()) && (v.pattern == "" || strings.EqualFold(strings.TrimSpace(patternText), v.pattern)) {
			matching = append(matching, item)
		}
	}
	page := matching[min(v.offset, len(matching)):]
	if v.limit > 0 && len(page) > v.limit {
		page = page[:v.limit]
		out["nextCursor"] = strconv.Itoa(v.offset + v.limit)
	}
	if v.filtered {
		out["matchingCount"] = len(matching)
	}
	if v.compact {
		out["problems"] = compactTodoResults(page, inSet)
	} else {
		out["problems"] = todoResults(page, inSet)
	}
}

func todoResults(items []leetgrinder.TodoItem, inSet bool) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		metadata, _ := item.SourceData["metadata"].(map[string]any)
		if metadata == nil {
			metadata = map[string]any{}
		}
		result := mcpLeetgrinderProblem(item.Problem)
		result["id"], result["metadata"], result["sourceProblem"], result["done"] = item.ID, metadata, item.SourceData, item.Done()
		if item.Done() {
			result["doneAt"] = item.DoneAt
		}
		if inSet {
			result["position"] = item.Position
		}
		out = append(out, result)
	}
	return out
}

func compactTodoResults(items []leetgrinder.TodoItem, inSet bool) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result := map[string]any{"id": item.ID, "title": item.Problem.Title, "difficulty": item.Problem.Difficulty, "done": item.Done()}
		if item.Problem.Number > 0 {
			result["number"] = item.Problem.Number
		}
		if inSet {
			result["position"] = item.Position
		}
		if pattern, ok := item.Field("metadata.pattern"); ok {
			result["pattern"] = pattern
		}
		if neetcode150, ok := item.Field("metadata.neetcode150"); ok {
			result["neetcode150"] = neetcode150
		}
		out = append(out, result)
	}
	return out
}

// todoOrderFromMCP checks a reorder request and returns the order it asks for.
func todoOrderFromMCP(in reorderTodoSetInput) (func([]leetgrinder.TodoItem) ([]string, error), error) {
	if in.SetID == "" {
		return nil, invalidInput("setID is required")
	}
	if btoi(len(in.OrderedIDs) > 0)+btoi(len(in.Moves) > 0)+btoi(len(in.OrderBy) > 0) != 1 {
		return nil, invalidInput("pass exactly one of orderedIds, moves and orderBy")
	}
	if len(in.OrderedIDs) > maxTodoReorderIDs || len(in.Moves) > maxTodoReorderIDs {
		return nil, invalidInput("pass at most %d IDs or moves", maxTodoReorderIDs)
	}
	switch {
	case len(in.OrderedIDs) > 0:
		return func(items []leetgrinder.TodoItem) ([]string, error) {
			return leetgrinder.OrderTodosFront(in.SetID, items, in.OrderedIDs)
		}, nil
	case len(in.Moves) > 0:
		moves := make([]leetgrinder.TodoMove, len(in.Moves))
		for i, m := range in.Moves {
			moves[i] = leetgrinder.TodoMove{ID: m.ID, BeforeID: m.BeforeID, AfterID: m.AfterID, ToIndex: m.ToIndex}
		}
		return func(items []leetgrinder.TodoItem) ([]string, error) {
			return leetgrinder.MoveTodos(in.SetID, items, moves)
		}, nil
	}
	if len(in.OrderBy) > maxTodoSortKeys {
		return nil, invalidInput("pass at most %d orderBy keys", maxTodoSortKeys)
	}
	keys := make([]leetgrinder.TodoSortKey, len(in.OrderBy))
	for i, key := range in.OrderBy {
		if !leetgrinder.TodoSortFieldValid(key.Field) {
			return nil, invalidInput("unknown orderBy field %q; use title, number, difficulty, done, slug or metadata.<key>", leetgrinder.EchoRef(key.Field))
		}
		if len(key.Values) > maxTodoSortValues {
			return nil, invalidInput("pass at most %d values per orderBy key", maxTodoSortValues)
		}
		keys[i] = leetgrinder.TodoSortKey{Field: key.Field, Values: key.Values, Descending: key.Descending}
	}
	return func(items []leetgrinder.TodoItem) ([]string, error) {
		return leetgrinder.SortTodos(items, keys), nil
	}, nil
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func todoToolError(err error) error {
	var order leetgrinder.TodoOrderError
	if errors.As(err, &order) {
		return invalidInput("%s; refresh the todo list", order.Error())
	}
	if errors.Is(err, leetgrinder.ErrInvalidTodoMove) {
		return invalidInput("%s", err.Error())
	}
	if errors.Is(err, database.ErrInvalid) {
		return invalidInput("invalid todo input")
	}
	if errors.Is(err, database.ErrNotFound) {
		return invalidInput("todo item or set not found; refresh the todo list")
	}
	return toolError(err)
}
