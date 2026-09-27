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
	} else {
		p.PauseError = true
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
	// Requeueing clears the current result, so it is refused whenever the
	// worker would not pick the attempt up soon (as the hidden button implies).
	if avail := s.historyAnalysis(r); !avail.On() || avail.Waiting() {
		if avail.Unknown {
			renderLeetgrinder(w, r, http.StatusServiceUnavailable, leetgrinder.ReanalyseError(problem, id.String(), "The analysis status could not be loaded, so nothing was changed. Please retry."))
			return
		}
		reason := "Analysis is not running right now"
		switch {
		case !avail.KeyConfigured:
			reason = "Analysis needs an Anthropic API key on the server"
		case avail.ZeroLimit:
			reason = "The server's daily analysis limit is 0"
		case !avail.Enabled:
			reason = "Analysis is turned off"
		case avail.Paused:
			reason = "Analysis is paused after an error that affects every attempt"
		case avail.LimitReached:
			reason = "Today's analysis limit is used up"
		}
		renderLeetgrinder(w, r, http.StatusConflict, leetgrinder.ReanalyseError(problem, id.String(), reason+", so nothing was changed."))
		return
	}
	err = s.db.RequeueLeetgrinderAnalysis(r.Context(), problem.Slug, id.String(), s.clock())
	switch {
	case errors.Is(err, database.ErrNotFound):
		renderLeetgrinder(w, r, http.StatusNotFound, leetgrinder.ReanalyseError(problem, id.String(), "This attempt was not found, or it needs captured code and a stated time or space complexity before it can be analysed."))
	case err != nil:
		renderLeetgrinder(w, r, http.StatusServiceUnavailable, leetgrinder.ReanalyseError(problem, id.String(), "The analysis could not be queued. Please retry."))
	default:
		http.Redirect(w, r, leetgrinder.ProblemURL(problem.Slug)+"#"+leetgrinder.AttemptAnchor(id.String()), http.StatusSeeOther)
	}
}
