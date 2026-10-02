package server

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func (s *Server) registerLeetgrinderOptimal(r chi.Router) {
	r.Get("/leetgrinder/problem/{slug}/optimal", s.leetgrinderOptimalEdit)
	r.Post("/leetgrinder/problem/{slug}/optimal", s.leetgrinderSaveOptimal)
	r.Post("/leetgrinder/problem/{slug}/optimal/reestimate", s.leetgrinderReestimateOptimal)
}

func (s *Server) leetgrinderOptimalEdit(w http.ResponseWriter, r *http.Request) {
	problem, ok := s.leetgrinderRouteProblem(w, r)
	if !ok {
		return
	}
	renderLeetgrinder(w, r, 200, leetgrinder.OptimalEdit(problem, leetgrinder.NewOptimalForm(problem)))
}

// leetgrinderSaveOptimal stores the learner's optimal complexity as 'manual'.
// The form carries the revision of the values it was opened on, so an edit
// made after a model estimate, another edit or a re-estimate is refused.
func (s *Server) leetgrinderSaveOptimal(w http.ResponseWriter, r *http.Request) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	problem, ok := s.leetgrinderRouteProblem(w, r)
	if !ok {
		return
	}
	form := leetgrinder.OptimalForm{
		Time:     leetgrinder.ComplexityInput{Choice: r.PostForm.Get("timeComplexity"), Other: r.PostForm.Get("timeComplexityOther")},
		Space:    leetgrinder.ComplexityInput{Choice: r.PostForm.Get("spaceComplexity"), Other: r.PostForm.Get("spaceComplexityOther")},
		Note:     strings.TrimSpace(r.PostForm.Get("note")),
		Revision: r.PostForm.Get("revision"),
	}
	reject := func(status int, message string) {
		form.Error = message
		renderLeetgrinder(w, r, status, leetgrinder.OptimalEdit(problem, form))
	}
	optimalTime, errTime := leetgrinder.NormalizeComplexity(form.Time.Value())
	optimalSpace, errSpace := leetgrinder.NormalizeComplexity(form.Space.Value())
	switch {
	case errTime != nil || errSpace != nil:
		reject(400, "Write complexity in big-O notation, such as O(m·n): start with O( and end with ), in 40 characters or fewer.")
		return
	case optimalTime == "" || optimalSpace == "":
		reject(400, "Choose both the optimal time and the optimal space complexity.")
		return
	case utf8.RuneCountInString(form.Note) > leetgrinder.MaxOptimalNote:
		reject(400, "Keep the note to 300 characters or fewer.")
		return
	case strings.ContainsFunc(form.Note, func(r rune) bool { return r < ' ' || r == 0x7f }):
		reject(400, "The note cannot contain control characters.")
		return
	}
	err := s.db.SaveLeetgrinderManualOptimal(r.Context(), problem.Slug, optimalTime, optimalSpace, form.Note, form.Revision)
	switch {
	case err == nil:
		http.Redirect(w, r, leetgrinder.ProblemURL(problem.Slug)+"#optimal", http.StatusSeeOther)
	case errors.Is(err, database.ErrConflict):
		reject(409, "The optimal complexity changed since you opened this page. Reload the current value, then reapply your draft shown below.")
	case errors.Is(err, database.ErrInvalid):
		reject(400, "Check the optimal complexity and try again.")
	default:
		reject(503, "The optimal complexity could not be saved. Your draft is retained; please retry.")
	}
}

// leetgrinderReestimateOptimal clears Claude's estimate (and only a model
// estimate), so the next analysis of the problem estimates the optimum again.
// It does not requeue anything: attempts keep their verdicts until they are
// re-analysed, and analysis may be off, paused or at its limit.
func (s *Server) leetgrinderReestimateOptimal(w http.ResponseWriter, r *http.Request) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	problem, ok := s.leetgrinderRouteProblem(w, r)
	if !ok {
		return
	}
	err := s.db.ClearLeetgrinderModelOptimal(r.Context(), problem.Slug, r.PostForm.Get("revision"))
	switch {
	case err == nil:
		http.Redirect(w, r, leetgrinder.ProblemURL(problem.Slug)+"#optimal", http.StatusSeeOther)
	case errors.Is(err, database.ErrConflict):
		renderLeetgrinder(w, r, http.StatusConflict, leetgrinder.OptimalError(problem, "The optimal complexity changed since you opened this page, so nothing was cleared. Reload the problem and try again."))
	case errors.Is(err, database.ErrInvalid):
		renderLeetgrinder(w, r, http.StatusConflict, leetgrinder.OptimalError(problem, "Only Claude's estimate can be re-estimated; a curated or manual value is kept."))
	default:
		renderLeetgrinder(w, r, http.StatusServiceUnavailable, leetgrinder.OptimalError(problem, "The estimate could not be cleared. Please retry."))
	}
}
