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

// analysisPanel builds the settings section. It reports only whether a key
// is configured, never the key.
func (s *Server) analysisPanel(ctx context.Context, settings leetgrinder.Settings) leetgrinder.AnalysisPanel {
	p := leetgrinder.AnalysisPanel{
		KeyConfigured: s.config.AnthropicAPIKey.Reveal() != "",
		Enabled:       settings.AnalysisEnabled,
		Model:         s.config.AnalysisModel,
		Limit:         s.config.AnalysisDailyLimit,
		Revision:      settings.Revision,
		Location:      settings.Location(),
	}
	if pause, err := s.db.LeetgrinderAnalysisPause(ctx); err == nil {
		p.Pause, p.PausedNow = pause, s.clock().Before(pause.Until)
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
	enabled, revision := r.PostForm.Get("enabled"), r.PostForm.Get("revision")
	// reject re-renders settings with the submitted toggle and revision.
	reject := func(status int, message string) {
		settings, err := s.db.LeetgrinderSettings(r.Context())
		if err != nil {
			message += " Your settings could not be reloaded; your draft is retained below."
		}
		page := s.withNotify(r.Context(), leetgrinder.SettingsPage{Settings: settings, Now: s.clock(), Schedule: leetgrinder.NewScheduleForm(settings), Error: message})
		page.Analysis.Enabled, page.Analysis.Revision = enabled == "true", revision
		renderLeetgrinder(w, r, status, leetgrinder.SettingsView(s.withAPITokens(r, page)))
	}
	if enabled != "" && enabled != "true" {
		reject(400, "Choose whether to analyse captured code.")
		return
	}
	_, err := s.db.UpdateLeetgrinderSettings(r.Context(), revision, func(settings *leetgrinder.Settings) error {
		settings.AnalysisEnabled = enabled == "true"
		return nil
	})
	switch {
	case err == nil:
		http.Redirect(w, r, "/leetgrinder/settings?saved=analysis#analysis", http.StatusSeeOther)
	case errors.Is(err, database.ErrConflict):
		reject(409, "Settings changed since you opened this page. Reload settings to see the current values, then reapply your analysis choice shown below.")
	case errors.Is(err, database.ErrInvalid):
		reject(400, "Reload the settings page and try again.")
	default:
		reject(503, "The analysis setting could not be saved. Please retry.")
	}
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
		renderLeetgrinder(w, r, http.StatusNotFound, leetgrinder.ReanalyseError(problem, "This attempt has no captured code and stated complexity to analyse."))
	case err != nil:
		renderLeetgrinder(w, r, http.StatusServiceUnavailable, leetgrinder.ReanalyseError(problem, "The analysis could not be queued. Please retry."))
	default:
		http.Redirect(w, r, leetgrinder.ProblemURL(problem.Slug)+"#"+leetgrinder.AttemptAnchor(id.String()), http.StatusSeeOther)
	}
}
