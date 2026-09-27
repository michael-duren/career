package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func (s *Server) registerLeetgrinderAnalysis(r chi.Router) {
	r.Post("/leetgrinder/settings/analysis", s.leetgrinderSaveAnalysis)
	r.Post("/leetgrinder/problem/{slug}/attempts/{id}/analysis", s.leetgrinderReanalyse)
}

// analysisOn reports whether the analysis worker sends requests: a key is
// configured and analysis is turned on in settings.
func (s *Server) analysisOn(settings leetgrinder.Settings) bool {
	return s.config.AnthropicAPIKey.Reveal() != "" && settings.AnalysisEnabled
}

// analysisPanel builds the settings section. It reports only whether a key
// is configured, never the key.
func (s *Server) analysisPanel(ctx context.Context, settings leetgrinder.Settings) leetgrinder.AnalysisPanel {
	p := leetgrinder.AnalysisPanel{
		KeyConfigured: s.config.AnthropicAPIKey.Reveal() != "",
		Enabled:       settings.AnalysisEnabled,
		Model:         s.config.AnalysisModel,
		Limit:         s.config.AnalysisDailyLimit,
		Revision:      settings.Revision,
	}
	var err error
	p.Used, err = s.db.LeetgrinderAnalysisUsage(ctx, leetgrinder.Date(s.clock(), settings.Location()))
	p.UsageError = err != nil
	p.Queue, err = s.db.LeetgrinderAnalysisQueueCounts(ctx)
	p.QueueError = err != nil
	return p
}

func (s *Server) leetgrinderSaveAnalysis(w http.ResponseWriter, r *http.Request) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	enabled := r.PostForm.Get("enabled")
	if enabled != "" && enabled != "true" {
		s.rejectNotify(w, r, 400, "Choose whether to analyse captured code.", leetgrinder.NotifyPanel{})
		return
	}
	s.saveNotifySettings(w, r, r.PostForm.Get("revision"), "analysis", leetgrinder.NotifyPanel{}, func(settings *leetgrinder.Settings) error {
		settings.AnalysisEnabled = enabled == "true"
		return nil
	})
}

// leetgrinderReanalyse queues an attempt's analysis again from scratch.
func (s *Server) leetgrinderReanalyse(w http.ResponseWriter, r *http.Request) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	problem, ok := leetgrinder.FindProblem(chi.URLParam(r, "slug"))
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if !ok || err != nil {
		http.NotFound(w, r)
		return
	}
	err = s.db.RequeueLeetgrinderAnalysis(r.Context(), problem.Slug, id.String(), s.clock())
	switch {
	case errors.Is(err, database.ErrNotFound):
		http.Error(w, "This attempt has no code and stated complexity to analyse.", http.StatusNotFound)
	case err != nil:
		http.Error(w, "The analysis could not be queued. Please retry.", http.StatusServiceUnavailable)
	default:
		http.Redirect(w, r, leetgrinder.ProblemURL(problem.Slug)+"#"+leetgrinder.AttemptAnchor(id.String()), http.StatusSeeOther)
	}
}
