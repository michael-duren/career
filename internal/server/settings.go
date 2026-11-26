package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	pathpkg "path"
	"slices"
	"strings"
	"time"

	"github.com/michael-duren/career-strategy/internal/modules"
)

// settingsView is the GET/PATCH /api/settings response: the stored settings
// plus whether each module is usable after its dependencies.
type settingsView struct {
	modules.Settings
	Effective map[string]bool `json:"effective"`
}

func newSettingsView(s modules.Settings) settingsView {
	return settingsView{Settings: s, Effective: s.Effective()}
}

func (s *Server) readSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.db.AppSettings(r.Context())
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, newSettingsView(settings))
}

func (s *Server) patchSettings(w http.ResponseWriter, r *http.Request) {
	if !s.mutation(w, r) {
		return
	}
	var input struct {
		Modules          map[string]*bool `json:"modules"`
		ModuleOrder      []string        `json:"moduleOrder"`
		Theme            *string         `json:"theme"`
		SetupCompletedAt json.RawMessage `json:"setupCompletedAt"`
	}
	if !decode(w, r, 8000, &input) {
		return
	}
	patch := modules.Patch{Order: input.ModuleOrder, Theme: input.Theme}
	if input.Modules != nil {
		patch.Modules = map[string]bool{}
		for id, on := range input.Modules {
			if on == nil {
				respond(w, 400, map[string]string{"error": "modules values must be true or false"})
				return
			}
			patch.Modules[id] = *on
		}
	}
	switch raw := string(input.SetupCompletedAt); {
	case raw == "":
	case raw == "null":
		patch.ClearSetup = true
	default:
		var value string
		at, err := time.Time{}, json.Unmarshal(input.SetupCompletedAt, &value)
		if err == nil {
			at, err = time.Parse(time.RFC3339, value)
		}
		if err != nil {
			respond(w, 400, map[string]string{"error": "setupCompletedAt must be an RFC 3339 time or null"})
			return
		}
		patch.SetupCompletedAt = &at
	}
	settings, err := s.db.UpdateAppSettings(r.Context(), patch)
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, newSettingsView(settings))
}

// moduleSettings loads the module settings. Without a database (handler
// tests that need none) every module is on. A load failure is returned so
// callers fail closed instead of showing a module that may be off.
func (s *Server) moduleSettings(ctx context.Context) (modules.Settings, error) {
	if s.db == nil {
		return modules.Settings{}.Normalize(), nil
	}
	settings, err := s.db.AppSettings(ctx)
	if err != nil {
		return settings, fmt.Errorf("load module settings: %w", err)
	}
	return settings, nil
}

// requestModule returns the module that owns a request path.
func requestModule(path string) (modules.Module, bool) {
	if kind, ok := strings.CutPrefix(path, "/api/entries/"); ok {
		kind, _, _ = strings.Cut(kind, "/")
		return modules.ForKind(kind)
	}
	return modules.ForPath(path)
}

// moduleOff answers a request for a disabled module with 404.
func moduleOff(w http.ResponseWriter, r *http.Request, settings modules.Settings, m modules.Module) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		respond(w, 404, map[string]string{"error": settings.OffReason(m.ID) + ". Turn it on in Settings → Modules (or PATCH /api/settings)."})
		return
	}
	http.NotFound(w, r)
}

// moduleGuard returns 404 for pages and API routes of disabled modules.
// Unauthenticated API requests pass through so their handlers answer 401
// without revealing which modules are on. Leetgrinder extension requests
// carry their own bearer token, so they are guarded by requireModule after
// the token check instead.
func (s *Server) moduleGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := pathpkg.Clean("/" + r.URL.Path)
		api := strings.HasPrefix(path, "/api/")
		// Every /api/leetgrinder/ route is in the bearer-token group behind
		// requireModule; a session route added there must not rely on this.
		extension := strings.HasPrefix(path, "/api/leetgrinder/") && r.Header.Get("Authorization") != ""
		// Journal and audio Markdown views of agent context are checked by the
		// handler against their own modules, so they work with Agents off.
		journalView := path == "/api/agent-context" && r.URL.Query().Get("journal") != ""
		if extension || journalView || (api && !s.authenticated(r)) {
			next.ServeHTTP(w, r)
			return
		}
		// Check the module of the cleaned path (so //notes or /x/../notes
		// cannot slip past) and of the raw path the router matches.
		var owners []modules.Module
		for _, p := range []string{path, r.URL.Path} {
			if m, ok := requestModule(p); ok && !slices.ContainsFunc(owners, func(o modules.Module) bool { return o.ID == m.ID }) {
				owners = append(owners, m)
			}
		}
		if len(owners) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		settings, err := s.moduleSettings(r.Context())
		if err != nil {
			failure(w, err)
			return
		}
		for _, m := range owners {
			if !settings.Enabled(m.ID) {
				moduleOff(w, r, settings, m)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// guardModule serves next when m is on, 404 when it is off, and 503 when the
// settings cannot be loaded (fail closed).
func (s *Server) guardModule(w http.ResponseWriter, r *http.Request, next http.Handler, m modules.Module) {
	settings, err := s.moduleSettings(r.Context())
	if err != nil {
		failure(w, err)
		return
	}
	if !settings.Enabled(m.ID) {
		moduleOff(w, r, settings, m)
		return
	}
	next.ServeHTTP(w, r)
}

// requireModule returns 404 for every request while module id is unusable
// (off, or a dependency off) and 503 when settings cannot be loaded.
func (s *Server) requireModule(id string) func(http.Handler) http.Handler {
	m, ok := modules.Lookup(id)
	if !ok {
		panic("requireModule: unknown module " + id)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.guardModule(w, r, next, m)
		})
	}
}
