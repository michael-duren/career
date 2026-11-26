package server

import (
	"net/http"

	"github.com/michael-duren/career-strategy/internal/modules"
)

// readCounts answers GET /api/counts with the number of saved entries per
// usable module that owns entry kinds (the phone More page shows them); a
// module with none saved counts 0. Turned-off modules and modules that own
// no entry kinds (Schedule, Leetgrinder, Agents) are left out.
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
	effective := settings.Effective()
	counts := map[string]int{}
	for _, m := range modules.Registry {
		if !effective[m.ID] || len(m.Kinds) == 0 {
			continue
		}
		total := 0
		for _, kind := range m.Kinds {
			total += byKind[kind]
		}
		counts[m.ID] = total
	}
	respond(w, 200, map[string]any{"counts": counts})
}
