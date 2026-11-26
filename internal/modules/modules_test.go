package modules

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestEffectiveAppliesDependencies(t *testing.T) {
	s := Settings{Modules: map[string]bool{"timeline": false}}
	eff := s.Effective()
	if eff["timeline"] || eff["schedule"] || eff["agents"] || !eff["leetgrinder"] || !eff["notes"] {
		t.Fatal(eff)
	}
	if s.Enabled("schedule") || !s.Enabled("not-a-module") {
		t.Fatal("Enabled")
	}
	// The stored schedule switch is kept, so turning Timeline on restores it.
	if n := s.Normalize(); !n.Modules["schedule"] || len(n.Order) != len(Registry) || n.Theme != "dark" {
		t.Fatal(n)
	}
	if got := s.DisabledKinds(); !slices.Equal(got, []string{"goal"}) {
		t.Fatal(got)
	}
}

func TestApplyEnforcesDependencies(t *testing.T) {
	off := Settings{Modules: map[string]bool{"timeline": false, "schedule": false}}
	if _, err := off.Apply(Patch{Modules: map[string]bool{"schedule": true}}); err == nil || err.Error() != "Schedule needs Timeline" {
		t.Fatal("turned on a locked module:", err)
	}
	next, err := off.Apply(Patch{Modules: map[string]bool{"schedule": true, "timeline": true}})
	if err != nil || !next.Enabled("schedule") {
		t.Fatal(next, err)
	}
	// Turning a dependency off is allowed; the dependent keeps its switch.
	next, err = next.Apply(Patch{Modules: map[string]bool{"timeline": false}})
	if err != nil || !next.Modules["schedule"] || next.Enabled("schedule") {
		t.Fatal(next, err)
	}
	// Re-sending a locked module's stored "on" is not a change.
	if _, err = next.Apply(Patch{Modules: map[string]bool{"schedule": true}}); err != nil {
		t.Fatal(err)
	}
}

func TestApplyValidates(t *testing.T) {
	s := Settings{}.Normalize()
	bad := "neon"
	for name, p := range map[string]Patch{
		"unknown module": {Modules: map[string]bool{"bogus": true}},
		"unknown order":  {Order: []string{"bogus"}},
		"repeated order": {Order: []string{"notes", "notes"}},
		"theme":          {Theme: &bad},
	} {
		if _, err := s.Apply(p); err == nil {
			t.Error(name, "accepted")
		}
	}
	theme, at := "cobalt", time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	next, err := s.Apply(Patch{Order: []string{"notes", "journal"}, Theme: &theme, SetupCompletedAt: &at})
	if err != nil || next.Order[0] != "notes" || next.Order[1] != "journal" || len(next.Order) != len(Registry) || next.Theme != "cobalt" || !next.SetupCompletedAt.Equal(at) {
		t.Fatal(next, err)
	}
	if next, _ = next.Apply(Patch{ClearSetup: true}); next.SetupCompletedAt != nil {
		t.Fatal("setup not cleared")
	}
}

func TestOffReason(t *testing.T) {
	s := Settings{Modules: map[string]bool{"timeline": false, "notes": false}}
	if got := s.OffReason("schedule"); got != "Schedule needs Timeline, which is turned off" {
		t.Fatal(got)
	}
	if got := s.OffReason("notes"); got != "Notes is turned off" {
		t.Fatal(got)
	}
}

func TestToolOff(t *testing.T) {
	s := Settings{Modules: map[string]bool{"companies": false}}
	if m, off := s.ToolOff("add_companies_to_queue"); !off || m.ID != "companies" {
		t.Fatal("queue tool needs Companies", m, off)
	}
	if _, off := s.ToolOff("get_leetgrinder_today"); off {
		t.Fatal("leetgrinder tool off")
	}
	s = Settings{Modules: map[string]bool{"people": false}}
	if m, off := s.ToolOff("list_connection_companies"); !off || m.ID != "people" {
		t.Fatal("owner off", m, off)
	}
	if _, off := s.ToolOff("read_career_entry"); off {
		t.Fatal("entry tools are not owned")
	}
}

func TestOwnership(t *testing.T) {
	for path, want := range map[string]string{
		"/notes": "notes", "/notes/a/b": "notes", "/manage/books": "bookshelf", "/api/scheduler/week": "schedule",
		"/api/connections/photo": "people", "/leetgrinder/problem/two-sum": "leetgrinder", "/journal/new": "journal",
		"/api/agent-context": "agents", "/running": "audio", "/api/summaries/progress": "journal",
	} {
		if m, ok := ForPath(path); !ok || m.ID != want {
			t.Errorf("%s: got %q", path, m.ID)
		}
	}
	for _, path := range []string{"/", "/notesy", "/api/search", "/api/settings", "/documents/x"} {
		if m, ok := ForPath(path); ok {
			t.Errorf("%s owned by %s", path, m.ID)
		}
	}
	if m, ok := ForKind("week"); !ok || m.ID != "journal" {
		t.Error("week")
	}
	if _, ok := ForKind("document"); ok {
		t.Error("documents are not a module")
	}
	if m, ok := ForTool("get_weekly_schedule"); !ok || m.ID != "schedule" {
		t.Error("tool")
	}
}

// TestRegistryMatchesFrontend keeps src/lib/modules.ts and this registry in
// sync: same ids in the same order, themes, labels, short labels, routes, dependencies,
// entry kinds, MCP tools, tools needing other modules and API prefixes.
func TestRegistryMatchesFrontend(t *testing.T) {
	src, err := os.ReadFile("../../src/lib/modules.ts")
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	quoted := regexp.MustCompile(`'([^']*)'`)
	list := func(s string) []string {
		out := []string{}
		for _, m := range quoted.FindAllStringSubmatch(s, -1) {
			out = append(out, m[1])
		}
		return out
	}
	ids := regexp.MustCompile(`MODULE_IDS = \[([^\]]*)\]`).FindStringSubmatch(text)
	if ids == nil || !slices.Equal(list(ids[1]), IDs()) {
		t.Fatalf("MODULE_IDS %v, Go %v", ids, IDs())
	}
	themeSrc, err := os.ReadFile("../../src/lib/theme.ts")
	if err != nil {
		t.Fatal(err)
	}
	themes := regexp.MustCompile(`THEMES = \[([^\]]*)\]`).FindStringSubmatch(string(themeSrc))
	if themes == nil || !slices.Equal(list(themes[1]), Themes) {
		t.Fatalf("THEMES %v, Go %v", themes, Themes)
	}
	_, defs, _ := strings.Cut(text, "export const MODULES")
	defs, _, _ = strings.Cut(defs, "\n];")
	blocks := strings.Split(defs, "{ id: '")[1:]
	if len(blocks) != len(Registry) {
		t.Fatalf("%d frontend modules, %d Go", len(blocks), len(Registry))
	}
	field := func(block, name string) []string {
		m := regexp.MustCompile(name + `: \[([^\]]*)\]`).FindStringSubmatch(block)
		if m == nil {
			t.Fatalf("no %s in %q", name, block)
		}
		return list(m[1])
	}
	for i, m := range Registry {
		block := blocks[i]
		if !strings.HasPrefix(block, m.ID+"'") {
			t.Fatalf("module %d: %q", i, block[:20])
		}
		for name, want := range map[string][]string{"dependsOn": m.DependsOn, "kinds": m.Kinds, "mcpTools": m.MCPTools, "apiPrefixes": m.API} {
			if got := field(block, name); !slices.Equal(got, append([]string{}, want...)) {
				t.Errorf("%s %s: frontend %v, Go %v", m.ID, name, got, want)
			}
		}
		if label := regexp.MustCompile(`label: '([^']*)'`).FindStringSubmatch(block); label == nil || label[1] != m.Label {
			t.Errorf("%s label: frontend %v, Go %q", m.ID, label, m.Label)
		}
		if short := regexp.MustCompile(`short: '([^']*)'`).FindStringSubmatch(block); short == nil || short[1] != m.Short {
			t.Errorf("%s short: frontend %v, Go %q", m.ID, short, m.Short)
		}
		toolsNeed := map[string]string{}
		if tn := regexp.MustCompile(`toolsNeed: \{([^}]*)\}`).FindStringSubmatch(block); tn != nil {
			for _, pair := range regexp.MustCompile(`(\w+): '(\w+)'`).FindAllStringSubmatch(tn[1], -1) {
				toolsNeed[pair[1]] = pair[2]
			}
		}
		if len(toolsNeed) != len(m.ToolsNeed) {
			t.Errorf("%s toolsNeed: frontend %v, Go %v", m.ID, toolsNeed, m.ToolsNeed)
		}
		for tool, need := range m.ToolsNeed {
			if toolsNeed[tool] != need {
				t.Errorf("%s toolsNeed %s: frontend %q, Go %q", m.ID, tool, toolsNeed[tool], need)
			}
		}
		if got := field(block, "routes"); !slices.Equal(got, m.Pages) {
			t.Errorf("%s routes: frontend %v, Go %v", m.ID, got, m.Pages)
		}
	}
}
