package server

import (
	"context"
	"encoding/json"
	"log"
	"slices"

	"github.com/michael-duren/career-strategy/internal/modules"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpEntryTools take an MCP kind name and work on any module's entries.
var mcpEntryTools = []string{"list_career_entries", "search_career_context", "read_career_entry", "create_career_entry", "update_career_entry", "delete_career_entry"}

// mcpModuleFilter hides the tools of disabled modules from tools/list and
// refuses calls to them, including entry tools called with a disabled
// module's kind. If the module settings cannot be loaded, both fail.
func (s *Server) mcpModuleFilter(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		switch method {
		case "tools/list":
			result, err := next(ctx, method, req)
			list, ok := result.(*mcp.ListToolsResult)
			if err != nil || !ok {
				return result, err
			}
			settings, err := s.moduleSettings(ctx)
			if err != nil {
				log.Printf("mcp: %v", err)
				return nil, errMCPStorage
			}
			filtered := *list
			filtered.Tools = slices.DeleteFunc(slices.Clone(list.Tools), func(t *mcp.Tool) bool {
				_, off := settings.ToolOff(t.Name)
				return off
			})
			return &filtered, nil
		case "tools/call":
			call, ok := req.(*mcp.CallToolRequest)
			if !ok || call.Params == nil {
				break
			}
			settings, err := s.moduleSettings(ctx)
			if err != nil {
				log.Printf("mcp: %v", err)
				return mcpToolError(errMCPStorage.Error()), nil
			}
			if m, off := mcpToolModuleOff(settings, call.Params.Name, call.Params.Arguments); off {
				return mcpToolError(settings.OffReason(m.ID) + " in Settings → Modules, so its tools and entries are unavailable."), nil
			}
		}
		return next(ctx, method, req)
	}
}

func mcpToolError(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

// mcpToolModuleOff reports the disabled module a tool call needs, if any.
// Entry tools with arguments that do not parse are left to the tool, which
// rejects them before reading anything.
func mcpToolModuleOff(settings modules.Settings, name string, arguments json.RawMessage) (modules.Module, bool) {
	if m, off := settings.ToolOff(name); off {
		return m, true
	}
	if !slices.Contains(mcpEntryTools, name) {
		return modules.Module{}, false
	}
	var args struct {
		Kind string `json:"kind"`
	}
	if json.Unmarshal(arguments, &args) != nil {
		return modules.Module{}, false
	}
	m, owned := modules.ForKind(mcpKinds[args.Kind])
	return m, owned && !settings.Enabled(m.ID)
}
