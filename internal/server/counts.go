package server

import (
	"net/http"

	"github.com/michael-duren/career-strategy/internal/modules"
)

// readCounts answers GET /api/counts with how much each usable module has
// saved: entries for modules that own entry kinds (Journal sums its kinds) and
// attempts for Leetgrinder; a module with none saved counts 0. Schedule and
// Agents save nothing countable and are left out. Turned-off modules are left
// out too, unless ?all=1 asks for them (Settings → Modules shows "paused · N
// kept" for them).
func (s *Server) readCounts(w http.ResponseWriter, r *http.Request) {
	settings, err := s.moduleSettings(r.Context())
	if err != nil {
		failure(w, err)
		return
	}
	byKind, err := s.db.Counts(r.Context())
	if err != nil {
		failure(w, err)
		return
	}
	all := r.URL.Query().Get("all") == "1"
	effective := settings.Effective()
	counts := map[string]int{}
	for _, m := range modules.Registry {
		if !all && !effective[m.ID] {
			continue
		}
		switch {
		case m.ID == "leetgrinder":
			n, err := s.db.LeetgrinderAttemptCount(r.Context())
			if err != nil {
				failure(w, err)
				return
			}
			counts[m.ID] = n
		case len(m.Kinds) > 0:
			total := 0
			for _, kind := range m.Kinds {
				total += byKind[kind]
			}
			counts[m.ID] = total
		}
	}
	respond(w, 200, map[string]any{"counts": counts})
}
