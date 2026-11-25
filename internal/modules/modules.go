// Package modules is the server copy of the module registry in
// src/lib/modules.ts. Every feature is a module the user can turn off: a
// disabled module's pages and API routes return 404, its MCP tools are
// hidden, its search results are dropped and its reminders are skipped. Its
// data is kept, so turning it back on restores everything.
package modules

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// Module describes what a module owns, so the server can hide all of it.
type Module struct {
	ID    string
	Label string
	// Pages are page path prefixes, matched on whole path segments.
	Pages []string
	// API are API path prefixes, matched on whole path segments.
	API []string
	// Kinds are the entry kinds it owns, for /api/entries/{kind}, search and
	// the generic MCP entry tools.
	Kinds []string
	// MCPTools are the MCP tools only this module uses.
	MCPTools  []string
	DependsOn []string
	// ToolsNeed lists MCP tools that also need another module, such as the
	// catch-up queue tools that need Companies while People stays on.
	ToolsNeed map[string]string
}

// Registry lists every module in default order. Keep the ids in sync with
// src/lib/modules.ts (TestRegistryMatchesFrontend checks this).
var Registry = []Module{
	{ID: "timeline", Label: "Timeline", Pages: []string{"/timeline"}, API: []string{"/api/goals"}, Kinds: []string{"goal"}},
	{ID: "schedule", Label: "Schedule", Pages: []string{"/weekly-scheduler"}, API: []string{"/api/scheduler"}, DependsOn: []string{"timeline"},
		MCPTools: []string{"get_weekly_schedule", "save_schedule_session", "remove_schedule_session", "save_schedule_rule", "end_schedule_rule", "record_schedule_actual", "update_schedule_settings"}},
	{ID: "leetgrinder", Label: "Leetgrinder", Pages: []string{"/leetgrinder", "/bootcamp"}, API: []string{"/api/leetgrinder"},
		MCPTools: []string{"get_leetgrinder_today", "list_leetgrinder_reviews", "get_leetgrinder_stats", "log_leetgrinder_attempt", "list_leetgrinder_todos", "create_leetgrinder_todo_set", "add_leetgrinder_todo_problem", "reorder_leetgrinder_set", "remove_leetgrinder_todo_problem", "remove_leetgrinder_todo_set"}},
	{ID: "companies", Label: "Companies", Pages: []string{"/companies", "/manage/companies"}, Kinds: []string{"company"}},
	{ID: "people", Label: "People", Pages: []string{"/connections"}, API: []string{"/api/connections"}, Kinds: []string{"connection"},
		MCPTools:  []string{"list_connection_companies", "add_companies_to_queue"},
		ToolsNeed: map[string]string{"list_connection_companies": "companies", "add_companies_to_queue": "companies"}},
	{ID: "bookshelf", Label: "Bookshelf", Pages: []string{"/books", "/manage/books"}, Kinds: []string{"book"}},
	{ID: "journal", Label: "Journal", Pages: []string{"/journal", "/logs", "/personal-journal"}, API: []string{"/api/summaries/progress"}, Kinds: []string{"week", "personal"}},
	{ID: "notes", Label: "Notes", Pages: []string{"/notes"}, Kinds: []string{"note"}},
	{ID: "audio", Label: "Audio thoughts", Pages: []string{"/audio-thoughts", "/running"}, API: []string{"/api/running"}, Kinds: []string{"run"}},
	{ID: "agents", Label: "Agent context", Pages: []string{"/agents"}, API: []string{"/api/agent-context"}, DependsOn: []string{"timeline"}},
}

// Themes are the accepted theme names (src/lib/theme.ts); the first is the default.
var Themes = []string{"dark", "vellum", "cobalt", "concrete"}

// Settings are the user's module switches, tab order, theme and setup state.
type Settings struct {
	// Modules holds the stored switch for every module. A module missing from
	// storage is on.
	Modules map[string]bool `json:"modules"`
	// Order lists every module id in the user's order.
	Order            []string   `json:"moduleOrder"`
	Theme            string     `json:"theme"`
	SetupCompletedAt *time.Time `json:"setupCompletedAt"`
}

// Lookup returns the module with id.
func Lookup(id string) (Module, bool) {
	for _, m := range Registry {
		if m.ID == id {
			return m, true
		}
	}
	return Module{}, false
}

// IDs returns every module id in registry order.
func IDs() []string {
	ids := make([]string, len(Registry))
	for i, m := range Registry {
		ids[i] = m.ID
	}
	return ids
}

// Normalize fills in missing switches (on) and appends ids missing from
// the order in registry order, dropping unknown and repeated ids.
func (s Settings) Normalize() Settings {
	out := Settings{Modules: map[string]bool{}, Theme: s.Theme, SetupCompletedAt: s.SetupCompletedAt}
	for _, id := range IDs() {
		on, ok := s.Modules[id]
		out.Modules[id] = on || !ok
	}
	for _, id := range s.Order {
		if _, ok := Lookup(id); ok && !slices.Contains(out.Order, id) {
			out.Order = append(out.Order, id)
		}
	}
	for _, id := range IDs() {
		if !slices.Contains(out.Order, id) {
			out.Order = append(out.Order, id)
		}
	}
	if !slices.Contains(Themes, out.Theme) {
		out.Theme = Themes[0]
	}
	return out
}

// Effective reports whether each module is usable: switched on and every
// module it depends on usable too.
func (s Settings) Effective() map[string]bool {
	s = s.Normalize()
	out := map[string]bool{}
	var usable func(id string, seen []string) bool
	usable = func(id string, seen []string) bool {
		if v, ok := out[id]; ok {
			return v
		}
		m, _ := Lookup(id)
		on := s.Modules[id] && !slices.Contains(seen, id)
		for _, dep := range m.DependsOn {
			on = on && usable(dep, append(seen, id))
		}
		out[id] = on
		return on
	}
	for _, id := range IDs() {
		usable(id, nil)
	}
	return out
}

// Enabled reports whether module id is usable. Unknown ids are not modules
// and are always enabled.
func (s Settings) Enabled(id string) bool {
	if _, ok := Lookup(id); !ok {
		return true
	}
	return s.Effective()[id]
}

// Patch is a partial settings change. Nil fields are left as they are; an
// empty non-nil Order resets the order to registry order.
type Patch struct {
	Modules          map[string]bool
	Order            []string
	Theme            *string
	SetupCompletedAt *time.Time
	// ClearSetup clears the setup time and wins over SetupCompletedAt.
	ClearSetup bool
}

// Apply validates p against the registry and returns the changed settings.
// Newly turning a module on while a module it depends on stays off is
// rejected, since its switch is locked until the dependency is on. A module
// whose switch is already on (locked by an off dependency) may be sent as on
// again.
func (s Settings) Apply(p Patch) (Settings, error) {
	s = s.Normalize()
	next := Settings{Modules: map[string]bool{}, Order: slices.Clone(s.Order), Theme: s.Theme, SetupCompletedAt: s.SetupCompletedAt}
	for id, on := range s.Modules {
		next.Modules[id] = on
	}
	for id, on := range p.Modules {
		if _, ok := Lookup(id); !ok {
			return s, fmt.Errorf("unknown module %q", id)
		}
		next.Modules[id] = on
	}
	effective := next.Effective()
	for id, on := range p.Modules {
		if !on || effective[id] || s.Modules[id] {
			continue
		}
		m, _ := Lookup(id)
		var off []string
		for _, dep := range m.DependsOn {
			if !effective[dep] {
				dl, _ := Lookup(dep)
				off = append(off, dl.Label)
			}
		}
		return s, fmt.Errorf("%s needs %s", m.Label, strings.Join(off, " and "))
	}
	if p.Order != nil {
		seen := map[string]bool{}
		for _, id := range p.Order {
			if _, ok := Lookup(id); !ok {
				return s, fmt.Errorf("unknown module %q in moduleOrder", id)
			}
			if seen[id] {
				return s, fmt.Errorf("module %q repeated in moduleOrder", id)
			}
			seen[id] = true
		}
		next.Order = p.Order
	}
	if p.Theme != nil {
		if !slices.Contains(Themes, *p.Theme) {
			return s, fmt.Errorf("theme must be one of %s", strings.Join(Themes, ", "))
		}
		next.Theme = *p.Theme
	}
	if p.ClearSetup {
		next.SetupCompletedAt = nil
	} else if p.SetupCompletedAt != nil {
		next.SetupCompletedAt = p.SetupCompletedAt
	}
	return next.Normalize(), nil
}

// hasPrefix reports whether path is prefix or below it.
func hasPrefix(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

// ForPath returns the module that owns a page or API path.
func ForPath(path string) (Module, bool) {
	for _, m := range Registry {
		for _, p := range append(slices.Clone(m.Pages), m.API...) {
			if hasPrefix(path, p) {
				return m, true
			}
		}
	}
	return Module{}, false
}

// ForKind returns the module that owns an entry kind.
func ForKind(kind string) (Module, bool) {
	for _, m := range Registry {
		if slices.Contains(m.Kinds, kind) {
			return m, true
		}
	}
	return Module{}, false
}

// ForTool returns the module that owns an MCP tool.
func ForTool(name string) (Module, bool) {
	for _, m := range Registry {
		if slices.Contains(m.MCPTools, name) {
			return m, true
		}
	}
	return Module{}, false
}

// ToolOff returns the module that keeps an MCP tool unavailable: its owner
// when that is off, else a module the tool also needs.
func (s Settings) ToolOff(name string) (Module, bool) {
	m, owned := ForTool(name)
	if !owned {
		return Module{}, false
	}
	effective := s.Effective()
	if !effective[m.ID] {
		return m, true
	}
	if need, ok := m.ToolsNeed[name]; ok && !effective[need] {
		n, _ := Lookup(need)
		return n, true
	}
	return Module{}, false
}

// OffReason says why module id is unusable: "Schedule needs Timeline, which
// is turned off" when a module it depends on is off, else "<Label> is turned
// off".
func (s Settings) OffReason(id string) string {
	m, _ := Lookup(id)
	s = s.Normalize()
	if s.Modules[id] {
		effective := s.Effective()
		var off []string
		for _, dep := range m.DependsOn {
			if !effective[dep] {
				d, _ := Lookup(dep)
				off = append(off, d.Label)
			}
		}
		if len(off) > 0 {
			return m.Label + " needs " + strings.Join(off, " and ") + ", which is turned off"
		}
	}
	return m.Label + " is turned off"
}

// DisabledKinds returns the entry kinds of modules that are not usable.
func (s Settings) DisabledKinds() []string {
	effective := s.Effective()
	var kinds []string
	for _, m := range Registry {
		if !effective[m.ID] {
			kinds = append(kinds, m.Kinds...)
		}
	}
	return kinds
}
