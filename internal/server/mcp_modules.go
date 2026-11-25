package server

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/michael-duren/career-strategy/internal/modules"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpEntryTools take an MCP kind name and work on any module's entries.
var mcpEntryTools = []string{"list_career_entries", "search_career_context", "read_career_entry", "create_career_entry", "update_career_entry", "delete_career_entry"}

// mcpModuleFilter hides the tools of disabled modules from tools/list and
// refuses calls to them, including entry tools called with a disabled
// module's kind.
func (s *Server) mcpModuleFilter(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		switch method {
		case "tools/list":
			result, err := next(ctx, method, req)
			list, ok := result.(*mcp.ListToolsResult)
			if err != nil || !ok {
				return result, err
			}
			settings := s.moduleSettings(ctx)
			filtered := *list
			filtered.Tools = slices.DeleteFunc(slices.Clone(list.Tools), func(t *mcp.Tool) bool {
				m, owned := modules.ForTool(t.Name)
				return owned && !settings.Enabled(m.ID)
			})
			return &filtered, nil
		case "tools/call":
			call, ok := req.(*mcp.CallToolRequest)
			if !ok || call.Params == nil {
				break
			}
			if m, off := s.mcpToolModuleOff(ctx, call.Params.Name, call.Params.Arguments); off {
				return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: m.Label + " is turned off in Settings → Modules, so its tools and entries are unavailable."}}}, nil
			}
		}
		return next(ctx, method, req)
	}
}

// mcpToolModuleOff reports the disabled module a tool call needs, if any.
func (s *Server) mcpToolModuleOff(ctx context.Context, name string, arguments json.RawMessage) (modules.Module, bool) {
	m, owned := modules.ForTool(name)
	if !owned && slices.Contains(mcpEntryTools, name) {
		var args struct {
			Kind string `json:"kind"`
		}
		_ = json.Unmarshal(arguments, &args)
		m, owned = modules.ForKind(mcpKinds[args.Kind])
	}
	if !owned {
		return modules.Module{}, false
	}
	return m, !s.moduleSettings(ctx).Enabled(m.ID)
}
