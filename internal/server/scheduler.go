package server

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/scheduler"
	"net/http"
	"time"
)

func (s *Server) registerScheduler(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(s.private)
		r.Get("/api/scheduler/week", s.schedulerWeek)
		r.Post("/api/scheduler/mutate", s.schedulerMutate)
	})
}
func (s *Server) schedulerNow() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}
func (s *Server) schedulerWeek(w http.ResponseWriter, r *http.Request) {
	week := r.URL.Query().Get("week")
	if zone := r.URL.Query().Get("timeZone"); zone != "" {
		if err := s.db.SchedulerInitializeTimeZone(r.Context(), zone); err != nil {
			failure(w, err)
			return
		}
	}
	busy, availabilityErr := s.schedulerAvailability(r.Context(), week)
	out, err := s.db.SchedulerWeek(r.Context(), week, s.schedulerNow(), busy)
	if err != nil {
		failure(w, err)
		return
	}
	if availabilityErr != nil {
		out.Warnings = append(out.Warnings, "Google availability is stale or unavailable; refresh before planning")
	}
	respond(w, 200, out)
}
func (s *Server) schedulerMutate(w http.ResponseWriter, r *http.Request) {
	if !s.mutation(w, r) {
		return
	}
	var input scheduler.Mutation
	if !decode(w, r, 150000, &input) {
		return
	}
	var busy []scheduler.Busy
	var err error
	if input.Action != "actual" && input.Action != "cancel" {
		busy, err = s.schedulerPlanningAvailability(r.Context(), input.Week)
		if err != nil {
			respond(w, 503, map[string]string{"error": "Google availability check failed; your draft is kept. Retry or disconnect Google."})
			return
		}
	}
	out, err := s.db.SchedulerMutate(r.Context(), input, s.schedulerNow(), busy)
	if err != nil {
		var conflict *scheduler.Conflict
		if errors.Is(err, database.ErrConflict) || errors.As(err, &conflict) {
			ids := []string{}
			if conflict != nil {
				ids = conflict.IDs
			}
			respond(w, 409, map[string]any{"error": err.Error(), "current": out, "conflictIds": ids})
			return
		}
		failure(w, err)
		return
	}
	respond(w, 200, out)
}
