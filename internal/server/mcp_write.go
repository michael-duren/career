package server

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpWritableKinds are the entry kinds MCP clients with career:write may edit.
var mcpWritableKinds = map[string]bool{"goal": true, "company": true, "note": true}

// managedFields are set by the server and cannot be written through MCP.
var managedFields = map[string]bool{"id": true, "slug": true, "createdAt": true, "updatedAt": true}

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

var errNeedsWrite = errors.New("this connection is read-only; reconnect the connector and choose \"Allow read and edit\" to change saved data")

func canWrite(req *mcp.CallToolRequest) bool {
	return req != nil && req.Extra != nil && req.Extra.TokenInfo != nil && slices.Contains(req.Extra.TokenInfo.Scopes, mcpWriteScope)
}

func writableKind(name string) (string, error) {
	if !mcpWritableKinds[name] {
		return "", fmt.Errorf("%w: kind must be goal, company or note", database.ErrInvalid)
	}
	return mcpKinds[name], nil
}

func slugify(title string) string {
	slug := strings.Trim(slugRE.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if len(slug) > 80 {
		slug = strings.Trim(slug[:80], "-")
	}
	return slug
}

// fillChildren gives new goal steps/notes and note todos the IDs and defaults
// the website would create, so clients only need to send titles and bodies.
func fillChildren(kind string, e database.Entity) {
	keys := map[string][]string{"goal": {"steps", "notes"}, "note": {"todos"}}[kind]
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, key := range keys {
		items, _ := e[key].([]any)
		for _, raw := range items {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if id, _ := item["id"].(string); id == "" {
				item["id"] = uuid.NewString()
			}
			if key == "notes" {
				if _, ok := item["createdAt"]; !ok {
					item["createdAt"] = now
				}
			} else if _, ok := item["done"]; !ok {
				item["done"] = false
			}
		}
	}
}

func setDefault(e database.Entity, key string, value any) {
	if _, ok := e[key]; !ok {
		e[key] = value
	}
}

// newEntry applies creation defaults and assigns the entry's key.
func newEntry(kind string, input map[string]any) (database.Entity, error) {
	e := database.Entity{}
	for k, v := range input {
		if managedFields[k] {
			continue
		}
		e[k] = v
	}
	title, _ := e["title"].(string)
	setDefault(e, "tags", []any{})
	switch kind {
	case "goal":
		now := time.Now().UTC().Format(time.RFC3339Nano)
		e["id"], e["createdAt"], e["updatedAt"] = uuid.NewString(), now, now
		setDefault(e, "color", "#67e8f9")
		setDefault(e, "steps", []any{})
		setDefault(e, "notes", []any{})
		setDefault(e, "metadata", map[string]any{})
		delete(e, "tags")
	case "note":
		topic, _ := e["topic"].(string)
		e["id"] = slugify(topic + " " + title)
		setDefault(e, "description", "")
		setDefault(e, "body", "")
	case "company":
		e["slug"] = slugify(title)
		setDefault(e, "type", "company")
		setDefault(e, "featured", false)
		setDefault(e, "priority", "medium")
		setDefault(e, "status", "not_started")
		setDefault(e, "body", "")
	}
	if e["slug"] == "" || e["id"] == "" {
		return nil, fmt.Errorf("%w: title must contain letters or digits", database.ErrInvalid)
	}
	fillChildren(kind, e)
	return e, nil
}

type createInput struct {
	Kind  string         `json:"kind" jsonschema:"goal, company or note"`
	Entry map[string]any `json:"entry" jsonschema:"fields of the new entry; IDs, slugs and timestamps are generated"`
}

type updateInput struct {
	Kind     string         `json:"kind" jsonschema:"goal, company or note"`
	ID       string         `json:"id" jsonschema:"entry ID or slug"`
	Revision string         `json:"revision" jsonschema:"revision returned by read_career_entry; the save fails if the entry changed since"`
	Fields   map[string]any `json:"fields" jsonschema:"only the fields to change; arrays such as steps, notes, todos, tags and dependsOn replace the whole array; null removes an optional field"`
}

func writeSchema[T any]() any {
	schema := inputSchema[T]()
	schema.Properties["kind"].Enum = []any{"company", "goal", "note"}
	return schema
}

var createAnnotations = &mcp.ToolAnnotations{Title: "Create career entry", DestructiveHint: new(bool), OpenWorldHint: new(bool)}
var updateAnnotations = &mcp.ToolAnnotations{Title: "Update career entry", OpenWorldHint: new(bool)}

const fieldGuide = "Fields — goal: title, status (planned|active|done|dropped), startDate, endDate (YYYY-MM-DD), color (#rrggbb), dailyHours, dependsOn (goal IDs), steps [{title, done}], notes [{body}], metadata {string: string}. " +
	"company: title, category, url, status (not_started|applied|interviewing|offer|rejected|passed), priority (high|medium|low), featured, tags, body (markdown). " +
	"note: title, topic, description, tags, body (markdown), todos [{title, done}]."

func (s *Server) addWriteTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_career_entry",
		Description: "Create a goal, company or note. Confirm with the user first. " + fieldGuide + " Returns the new ID and revision.",
		InputSchema: writeSchema[createInput](),
		Annotations: createAnnotations,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in createInput) (*mcp.CallToolResult, any, error) {
		if !canWrite(req) {
			return nil, nil, errNeedsWrite
		}
		kind, err := writableKind(in.Kind)
		if err != nil {
			return nil, nil, err
		}
		entry, err := newEntry(kind, in.Entry)
		if err != nil {
			return nil, nil, err
		}
		return s.saveFromMCP(ctx, in.Kind, kind, entry, nil)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_career_entry",
		Description: "Change fields of a goal, company or note. Confirm the change with the user first. Call read_career_entry first and pass its revision; send only changed fields. Existing steps, notes and todos keep their IDs; new ones may omit IDs. " + fieldGuide,
		InputSchema: writeSchema[updateInput](),
		Annotations: updateAnnotations,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in updateInput) (*mcp.CallToolResult, any, error) {
		if !canWrite(req) {
			return nil, nil, errNeedsWrite
		}
		kind, err := writableKind(in.Kind)
		if err != nil {
			return nil, nil, err
		}
		if !database.ValidID(kind, in.ID) || in.Revision == "" {
			return nil, nil, fmt.Errorf("%w: valid id and revision are required", database.ErrInvalid)
		}
		if len(in.Fields) == 0 {
			return nil, nil, fmt.Errorf("%w: no fields to change", database.ErrInvalid)
		}
		current, err := s.db.Detail(ctx, kind, in.ID)
		if err != nil {
			return nil, nil, toolError(err)
		}
		entry := current.Entry
		for k, v := range in.Fields {
			if managedFields[k] {
				return nil, nil, fmt.Errorf("%w: %s cannot be changed", database.ErrInvalid, k)
			}
			if v == nil {
				delete(entry, k)
			} else {
				entry[k] = v
			}
		}
		fillChildren(kind, entry)
		return s.saveFromMCP(ctx, in.Kind, kind, entry, &in.Revision)
	})
}

// saveFromMCP validates like the website and saves with an optimistic
// revision check, so a stale read can never overwrite newer edits.
func (s *Server) saveFromMCP(ctx context.Context, name, kind string, entry database.Entity, revision *string) (*mcp.CallToolResult, any, error) {
	prepared, err := database.PrepareSave(kind, entry)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", database.ErrInvalid, err)
	}
	saved, err := s.db.Save(ctx, kind, prepared, revision)
	switch {
	case errors.Is(err, database.ErrConflict) && revision == nil:
		return nil, nil, fmt.Errorf("%w: an entry with this ID already exists; update it instead or choose another title", database.ErrInvalid)
	case errors.Is(err, database.ErrConflict):
		return nil, nil, fmt.Errorf("%w: the entry changed since it was read; read it again, reapply the change, and retry", database.ErrInvalid)
	case err != nil:
		return nil, nil, toolError(err)
	}
	key := "id"
	if kind == "company" {
		key = "slug"
	}
	id, _ := saved.Entry[key].(string)
	return nil, map[string]any{"kind": name, "id": id, "revision": saved.Revision, "updatedAt": saved.Entry["updatedAt"]}, nil
}
