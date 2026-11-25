package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
		Modules          map[string]bool `json:"modules"`
		ModuleOrder      []string        `json:"moduleOrder"`
		Theme            *string         `json:"theme"`
		SetupCompletedAt json.RawMessage `json:"setupCompletedAt"`
	}
	if !decode(w, r, 8000, &input) {
		return
	}
	patch := modules.Patch{Modules: input.Modules, Order: input.ModuleOrder, Theme: input.Theme}
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
func moduleOff(w http.ResponseWriter, r *http.Request, m modules.Module) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		respond(w, 404, map[string]string{"error": m.Label + " is turned off. Turn it on in Settings → Modules."})
		return
	}
	http.NotFound(w, r)
}

// moduleGuard returns 404 for pages and API routes of disabled modules.
// Unauthenticated API requests pass through so their handlers answer 401
// without revealing which modules are on. The Leetgrinder extension API
// authenticates with its own tokens, so it is guarded by requireModule after
// that check instead.
func (s *Server) moduleGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		m, ok := requestModule(path)
		api := strings.HasPrefix(path, "/api/")
		if !ok || strings.HasPrefix(path, "/api/leetgrinder/") || (api && !s.authenticated(r)) {
			next.ServeHTTP(w, r)
			return
		}
		s.guardModule(w, r, next, m)
	})
}

// guardModule serves next when m is on, 404 when it is off, and 503 when the
// settings cannot be loaded.
func (s *Server) guardModule(w http.ResponseWriter, r *http.Request, next http.Handler, m modules.Module) {
	settings, err := s.moduleSettings(r.Context())
	if err != nil {
		failure(w, err)
		return
	}
	if !settings.Enabled(m.ID) {
		moduleOff(w, r, m)
		return
	}
	next.ServeHTTP(w, r)
}

// requireModule returns 404 for every request while module id is off.
func (s *Server) requireModule(id string) func(http.Handler) http.Handler {
	m, _ := modules.Lookup(id)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.guardModule(w, r, next, m)
		})
	}
}
