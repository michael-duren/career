package server

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func (s *Server) registerLeetgrinder(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(s.private)
		redirectLegacy := func(w http.ResponseWriter, r *http.Request) {
			target := "/leetgrinder" + strings.TrimPrefix(r.URL.Path, "/bootcamp")
			if r.URL.RawQuery != "" {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusPermanentRedirect)
		}
		r.HandleFunc("/bootcamp", redirectLegacy)
		r.HandleFunc("/bootcamp/*", redirectLegacy)
		r.Get("/leetgrinder", s.leetgrinderOverview)
		r.Get("/leetgrinder/", s.leetgrinderOverview)
		r.Get("/leetgrinder/style.css", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
			_, _ = w.Write([]byte(leetgrinder.Styles))
		})
		r.Get("/leetgrinder/day/{day}", s.leetgrinderDay)
		r.Post("/leetgrinder/day/{day}/complete", s.leetgrinderComplete)
		r.Get("/leetgrinder/problem/{slug}", s.leetgrinderProblem)
		r.Post("/leetgrinder/problem/{slug}/attempts", s.leetgrinderAttempt)
		r.Get("/leetgrinder/export", func(w http.ResponseWriter, r *http.Request) {
			state, err := s.db.LeetgrinderState(r.Context())
			if err != nil {
				failure(w, err)
				return
			}
			w.Header().Set("Content-Disposition", `attachment; filename="leetgrinder-history.json"`)
			respond(w, 200, state)
		})
	})
}

func renderLeetgrinder(w http.ResponseWriter, r *http.Request, status int, component templ.Component) {
	var body bytes.Buffer
	if err := component.Render(r.Context(), &body); err != nil {
		http.Error(w, "Unable to render this page.", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(status)
	_, _ = w.Write(body.Bytes())
}
func (s *Server) leetgrinderOverview(w http.ResponseWriter, r *http.Request) {
	state, err := s.db.LeetgrinderState(r.Context())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your saved progress is unavailable. Please retry."))
		return
	}
	renderLeetgrinder(w, r, 200, leetgrinder.Overview(state))
}
func leetgrinderRouteDay(r *http.Request) (leetgrinder.Day, bool) {
	n, err := strconv.Atoi(chi.URLParam(r, "day"))
	if err != nil {
		return leetgrinder.Day{}, false
	}
	return leetgrinder.FindDay(n)
}
func dayPage(day leetgrinder.Day, state leetgrinder.State, message string) leetgrinder.DayPage {
	ids := map[string]string{}
	for _, p := range day.Core {
		ids[p.Slug] = uuid.NewString()
	}
	for _, p := range day.Optional {
		ids[p.Slug] = uuid.NewString()
	}
	return leetgrinder.DayPage{Day: day, State: state, IDs: ids, Error: message}
}
func (s *Server) leetgrinderDay(w http.ResponseWriter, r *http.Request) {
	day, ok := leetgrinderRouteDay(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	state, err := s.db.LeetgrinderState(r.Context())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your saved progress is unavailable. Please retry."))
		return
	}
	renderLeetgrinder(w, r, 200, leetgrinder.Session(dayPage(day, state, "")))
}
func (s *Server) leetgrinderProblem(w http.ResponseWriter, r *http.Request) {
	problem, ok := leetgrinder.FindProblem(chi.URLParam(r, "slug"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	state, err := s.db.LeetgrinderState(r.Context())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your attempt history is unavailable. Please retry."))
		return
	}
	renderLeetgrinder(w, r, 200, leetgrinder.ProblemHistory(problem, state, leetgrinder.NewForm(uuid.NewString(), 0)))
}
func (s *Server) leetgrinderForm(w http.ResponseWriter, r *http.Request) bool {
	if !s.mutation(w, r, "application/x-www-form-urlencoded") {
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid or oversized form. Notes must be 2,000 characters or fewer.", 400)
		return false
	}
	return true
}
func (s *Server) leetgrinderAttempt(w http.ResponseWriter, r *http.Request) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	problem, ok := leetgrinder.FindProblem(chi.URLParam(r, "slug"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	returnDay, _ := strconv.Atoi(r.PostForm.Get("returnDay"))
	if returnDay < 0 || returnDay > 84 {
		returnDay = 0
	}
	form := leetgrinder.AttemptForm{ID: r.PostForm.Get("id"), Revision: r.PostForm.Get("revision"), Outcome: r.PostForm.Get("outcome"), Minutes: r.PostForm.Get("minutes"), Assisted: r.PostForm.Get("assisted") == "true", Notes: r.PostForm.Get("notes"), ReturnDay: returnDay}
	reject := func(status int, message string) {
		form.Error = message
		state, err := s.db.LeetgrinderState(r.Context())
		if err != nil {
			form.Error += " Your history could not be loaded; your draft is retained below."
		}
		renderLeetgrinder(w, r, status, leetgrinder.ProblemHistory(problem, state, form))
	}
	if _, err := uuid.Parse(form.ID); err != nil {
		form.ID = uuid.NewString()
		reject(400, "Invalid attempt identifier. Your draft now has a new identifier; please save again.")
		return
	}
	if form.Revision != "" {
		if _, err := uuid.Parse(form.Revision); err != nil {
			reject(400, "Invalid revision. Reload the current attempt before applying your correction.")
			return
		}
	}
	minutes, err := strconv.Atoi(form.Minutes)
	if err != nil || minutes < 1 || minutes > 240 {
		reject(400, "Enter a duration between 1 and 240 minutes.")
		return
	}
	if form.Outcome != "solved" && form.Outcome != "struggled" && form.Outcome != "unfinished" {
		reject(400, "Choose solved, struggled, or unfinished.")
		return
	}
	if utf8.RuneCountInString(form.Notes) > 2000 {
		reject(400, "Keep notes to 2,000 characters or fewer.")
		return
	}
	if v := r.PostForm.Get("assisted"); v != "" && v != "true" && v != "false" {
		reject(400, "Choose whether you used hints or a solution.")
		return
	}
	_, err = s.db.SaveLeetgrinderAttempt(r.Context(), leetgrinder.Attempt{ID: form.ID, ProblemSlug: problem.Slug, Outcome: form.Outcome, Minutes: minutes, Assisted: form.Assisted, Notes: form.Notes}, form.Revision)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrConflict):
			reject(409, "This attempt changed since you opened it. Compare your draft with the current history before correcting it again.")
		case errors.Is(err, database.ErrNotFound):
			reject(404, "This attempt no longer exists. Your draft is retained below.")
		case errors.Is(err, database.ErrInvalid):
			reject(400, "Check the attempt fields and try again.")
		default:
			reject(503, "The attempt could not be saved. Your draft is retained; please retry.")
		}
		return
	}
	target := leetgrinder.ProblemURL(problem.Slug)
	if form.ReturnDay > 0 {
		target = leetgrinder.DayURL(form.ReturnDay) + "#" + problem.Slug
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
func (s *Server) leetgrinderComplete(w http.ResponseWriter, r *http.Request) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	day, ok := leetgrinderRouteDay(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	value := r.PostForm.Get("completed")
	if value != "true" && value != "false" {
		http.Error(w, "Choose finish or reopen.", 400)
		return
	}
	if err := s.db.SetLeetgrinderDay(r.Context(), day.Number, value == "true"); err != nil {
		state, _ := s.db.LeetgrinderState(r.Context())
		renderLeetgrinder(w, r, 503, leetgrinder.Session(dayPage(day, state, "The session status could not be saved. Please retry; your attempts have not changed.")))
		return
	}
	if value == "false" {
		http.Redirect(w, r, leetgrinder.DayURL(day.Number), 303)
		return
	}
	state, err := s.db.LeetgrinderState(r.Context())
	if err != nil {
		http.Redirect(w, r, "/leetgrinder", 303)
		return
	}
	target := "/leetgrinder"
	if next := leetgrinder.Summarize(state).NextDay; next > 0 {
		target = fmt.Sprintf("/leetgrinder/day/%d", next)
	}
	http.Redirect(w, r, target, 303)
}
