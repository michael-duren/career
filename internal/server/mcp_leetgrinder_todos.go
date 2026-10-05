package server

import (
	"context"
	"errors"

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
	SetID string `json:"setID,omitempty" jsonschema:"optional problem set ID; omit for the individual todo list"`
}

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
		Name: "list_leetgrinder_todos", Description: "List saved Leetgrinder problem sets and individual todo problems, including IDs for editing. Problem entries include a LeetCode url and neetcodeUrl when a match is known. A problem is done once a solved or struggled attempt is logged: set problems stay listed with done=true and count against remainingCount; individual problems leave the list once done.",
		Annotations: readOnly,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		sets, err := s.db.LeetgrinderTodoSets(ctx)
		if err != nil {
			return nil, nil, toolError(err)
		}
		standalone, err := s.db.LeetgrinderTodoItems(ctx, "")
		if err != nil {
			return nil, nil, toolError(err)
		}
		listedSets := make([]map[string]any, 0, len(sets))
		for _, set := range sets {
			listedSets = append(listedSets, map[string]any{"id": set.ID, "title": set.Title, "description": set.Description, "metadata": set.Metadata, "problemCount": len(set.Items), "remainingCount": set.Remaining(), "problems": todoResults(set.Items)})
		}
		return nil, map[string]any{"sets": listedSets, "individualProblems": todoResults(standalone)}, nil
	})
	mcp.AddTool(server, &mcp.Tool{
		Name: "create_leetgrinder_todo_set", Description: "Create a named Leetgrinder todo set with up to 200 problems. Use problemDetails to copy title, number, difficulty, topics and arbitrary metadata onto each catalog problem; use problems for plain links or slugs (leetcode.com, leetcode.cn or neetcode.io problem links, or LeetCode slugs; a NeetCode link is mapped to its LeetCode slug, and an unknown NeetCode problem is rejected). Set description and metadata are also preserved. Requires edit access.",
		Annotations: &mcp.ToolAnnotations{Title: "Create Leetgrinder problem set", DestructiveHint: new(bool), OpenWorldHint: new(bool)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in createTodoSetInput) (*mcp.CallToolResult, any, error) {
		if !canWrite(req) {
			return nil, nil, errNeedsWrite
		}
		if len(in.Problems)+len(in.ProblemDetails) > 200 {
			return nil, nil, invalidInput("provide at most 200 problems")
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
		Name: "add_leetgrinder_todo_problem", Description: "Add a LeetCode problem to the individual list or a set. The problem may be a leetcode.com, leetcode.cn or neetcode.io problem link, or a LeetCode slug; a NeetCode link is mapped to its LeetCode slug, and an unknown NeetCode problem is rejected. Optional title, number, difficulty, topics, and arbitrary metadata are saved on the catalog problem. Repeated additions keep one todo entry; adding a done individual problem again queues it for another pass. Requires edit access.",
		Annotations: &mcp.ToolAnnotations{Title: "Add Leetgrinder todo problem", DestructiveHint: new(bool), IdempotentHint: true, OpenWorldHint: new(bool)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in addTodoProblemInput) (*mcp.CallToolResult, any, error) {
		if !canWrite(req) {
			return nil, nil, errNeedsWrite
		}
		problem, err := todoProblemFromMCP(in.todoProblemDetailsInput)
		if err != nil {
			return nil, nil, err
		}
		item, err := s.db.AddLeetgrinderTodoProblem(ctx, in.SetID, problem)
		if err != nil {
			return nil, nil, todoToolError(err)
		}
		return nil, map[string]any{"id": item.ID, "setID": item.SetID, "problemSlug": problem.Slug}, nil
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

func todoResults(items []leetgrinder.TodoItem) []map[string]any {
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
		out = append(out, result)
	}
	return out
}

func todoToolError(err error) error {
	if errors.Is(err, database.ErrInvalid) {
		return invalidInput("invalid todo input")
	}
	if errors.Is(err, database.ErrNotFound) {
		return invalidInput("todo item or set not found; refresh the todo list")
	}
	return toolError(err)
}
