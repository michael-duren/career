package server

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

// leetgrinderAPIBodyLimit fits 2,000 notes characters even when every one is
// escaped as a JSON surrogate pair.
const leetgrinderAPIBodyLimit = 64 << 10

// registerLeetgrinderExtension adds token management (cookie session) and the
// extension API (bearer token). /api/ bypasses pageAccess, so the API
// handlers authenticate every request themselves and never accept cookies.
func (s *Server) registerLeetgrinderExtension(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(s.private)
		r.Post("/leetgrinder/settings/tokens", s.leetgrinderCreateToken)
		r.Post("/leetgrinder/settings/tokens/{id}/revoke", s.leetgrinderRevokeToken)
	})
	r.Group(func(r chi.Router) {
		r.Use(s.leetgrinderBearer)
		r.Post("/api/leetgrinder/attempts", s.leetgrinderAPIAttempt)
		r.Get("/api/leetgrinder/problem/{slug}", s.leetgrinderAPIProblem)
	})
}

// withAPITokens loads the token list into the settings page, keeping any
// message or draft already in page.APITokens. A failure only affects the
// token section, not the rest of the page.
func (s *Server) withAPITokens(r *http.Request, page leetgrinder.SettingsPage) leetgrinder.SettingsPage {
	tokens, err := s.db.LeetgrinderTokens(r.Context())
	page.APITokens.Tokens, page.APITokens.Unavailable = tokens, err != nil
	return page
}

func (s *Server) renderTokenSettings(w http.ResponseWriter, r *http.Request, status int, section leetgrinder.APITokensSection) {
	settings, err := s.db.LeetgrinderSettings(r.Context())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your settings are unavailable. Please retry."))
		return
	}
	page := leetgrinder.SettingsPage{Settings: settings, Now: s.clock(), Schedule: leetgrinder.NewScheduleForm(settings), APITokens: section}
	renderLeetgrinder(w, r, status, leetgrinder.SettingsView(s.withAPITokens(r, s.withNotify(r.Context(), page))))
}

func (s *Server) leetgrinderCreateToken(w http.ResponseWriter, r *http.Request) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	name := r.PostForm.Get("name")
	if _, err := leetgrinder.ValidateAPITokenName(name); err != nil {
		s.renderTokenSettings(w, r, 400, leetgrinder.APITokensSection{Name: name, Error: "Name the token in 1 to 64 printable characters."})
		return
	}
	token, plain, err := s.db.CreateLeetgrinderToken(r.Context(), name)
	switch {
	case err == nil:
		// Rendered directly rather than redirected: this response is the only
		// place the plaintext ever appears.
		s.renderTokenSettings(w, r, 200, leetgrinder.APITokensSection{Created: plain, CreatedName: token.Name})
	case errors.Is(err, database.ErrTokenLimit):
		s.renderTokenSettings(w, r, 409, leetgrinder.APITokensSection{Name: name, Error: "You have 20 active tokens. Revoke one before creating another."})
	case errors.Is(err, database.ErrInvalid):
		s.renderTokenSettings(w, r, 400, leetgrinder.APITokensSection{Name: name, Error: "Name the token in 1 to 64 printable characters."})
	default:
		s.renderTokenSettings(w, r, 503, leetgrinder.APITokensSection{Name: name, Error: "The token could not be created. Please retry."})
	}
}

func (s *Server) leetgrinderRevokeToken(w http.ResponseWriter, r *http.Request) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	err := s.db.RevokeLeetgrinderToken(r.Context(), chi.URLParam(r, "id"))
	switch {
	case err == nil:
		http.Redirect(w, r, "/leetgrinder/settings#api-tokens", http.StatusSeeOther)
	case errors.Is(err, database.ErrInvalid), errors.Is(err, database.ErrNotFound):
		s.renderTokenSettings(w, r, 404, leetgrinder.APITokensSection{Error: "That token no longer exists."})
	default:
		s.renderTokenSettings(w, r, 503, leetgrinder.APITokensSection{Error: "The token could not be revoked. Please retry."})
	}
}

// leetgrinderBearer authenticates extension API requests by personal token.
func (s *Server) leetgrinderBearer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") {
			unauthorized(w)
			return
		}
		_, err := s.db.AuthenticateLeetgrinderToken(r.Context(), strings.TrimSpace(token), s.clock())
		switch {
		case err == nil:
			next.ServeHTTP(w, r)
		case errors.Is(err, database.ErrUnauthorized):
			unauthorized(w)
		default:
			respond(w, 503, map[string]string{"error": "Authentication is temporarily unavailable; please retry."})
		}
	})
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="leetgrinder"`)
	respond(w, 401, map[string]string{"error": "Missing, invalid, or revoked API token."})
}

type leetgrinderAPIAttemptInput struct {
	ID          string `json:"id"`
	ProblemSlug string `json:"problemSlug"`
	Outcome     string `json:"outcome"`
	Minutes     int    `json:"minutes"`
	Assisted    bool   `json:"assisted"`
	Notes       string `json:"notes"`
	IsReview    *bool  `json:"isReview"`
}

func (s *Server) leetgrinderAPIAttempt(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		respond(w, 415, map[string]string{"error": "application/json required"})
		return
	}
	var input leetgrinderAPIAttemptInput
	if !decode(w, r, leetgrinderAPIBodyLimit, &input) {
		return
	}
	bad := func(message string) { respond(w, 400, map[string]string{"error": message}) }
	if _, err := uuid.Parse(input.ID); err != nil {
		bad("id must be a UUID.")
		return
	}
	switch input.Outcome {
	case "solved", "struggled", "unfinished":
	default:
		bad("outcome must be solved, struggled, or unfinished.")
		return
	}
	if input.Minutes < 1 || input.Minutes > 240 {
		bad("minutes must be between 1 and 240.")
		return
	}
	if !utf8.ValidString(input.Notes) || strings.ContainsRune(input.Notes, 0) || utf8.RuneCountInString(input.Notes) > 2000 {
		bad("notes must be 2,000 characters or fewer.")
		return
	}
	problem, ok := leetgrinder.FindProblem(input.ProblemSlug)
	if !ok {
		respond(w, 422, map[string]string{"error": "This problem is not in the Leetgrinder curriculum."})
		return
	}
	attempt := leetgrinder.Attempt{ID: input.ID, ProblemSlug: problem.Slug, Outcome: input.Outcome, Minutes: input.Minutes, Assisted: input.Assisted, Notes: input.Notes, Source: "extension", IsReview: input.IsReview != nil && *input.IsReview}
	saved, err := s.db.SaveLeetgrinderAttempt(r.Context(), attempt, "")
	switch {
	case err == nil:
		respond(w, 200, map[string]any{"attempt": saved})
	case errors.Is(err, database.ErrConflict):
		respond(w, 409, map[string]string{"error": "This attempt id was already used for a different attempt. Correct it in the app instead."})
	case errors.Is(err, database.ErrInvalid):
		bad("Check the attempt fields and try again.")
	default:
		respond(w, 503, map[string]string{"error": "The attempt could not be saved; please retry."})
	}
}

type leetgrinderAPIProblem struct {
	Slug         string               `json:"slug"`
	InCurriculum bool                 `json:"inCurriculum"`
	Title        string               `json:"title,omitempty"`
	Difficulty   string               `json:"difficulty,omitempty"`
	Session      int                  `json:"session,omitempty"`
	Week         int                  `json:"week,omitempty"`
	TodaysReview bool                 `json:"todaysReview"`
	ReviewDone   bool                 `json:"reviewDone"`
	Latest       *leetgrinder.Attempt `json:"latestAttempt"`
	HistoryURL   string               `json:"historyUrl,omitempty"`
}

func (s *Server) leetgrinderAPIProblem(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	problem, ok := leetgrinder.FindProblem(slug)
	if !ok {
		respond(w, 200, leetgrinderAPIProblem{Slug: slug})
		return
	}
	today, err := s.db.LeetgrinderToday(r.Context(), s.clock())
	if err != nil {
		respond(w, 503, map[string]string{"error": "Your progress is unavailable; please retry."})
		return
	}
	session := leetgrinder.ProblemDay(problem.Slug)
	out := leetgrinderAPIProblem{Slug: problem.Slug, InCurriculum: true, Title: problem.Title, Difficulty: problem.Difficulty, Session: session, Week: leetgrinder.WeekNumber(session), HistoryURL: leetgrinder.ProblemURL(problem.Slug)}
	for _, item := range today.Reviews {
		if item.Problem.Slug == problem.Slug {
			out.TodaysReview, out.ReviewDone = true, item.Done
		}
	}
	// Attempts are loaded newest first.
	for _, a := range today.State.Attempts {
		if a.ProblemSlug == problem.Slug {
			latest := a
			out.Latest = &latest
			break
		}
	}
	respond(w, 200, out)
}
