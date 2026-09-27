package server

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
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
		r.Get("/leetgrinder/reviews", s.leetgrinderReviews)
		r.Get("/leetgrinder/settings", s.leetgrinderSettings)
		r.Post("/leetgrinder/settings/schedule", s.leetgrinderSaveSchedule)
		s.registerLeetgrinderNotify(r)
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
func (s *Server) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}
func reviewIDs(ids map[string]string, today leetgrinder.Today) map[string]string {
	for _, item := range today.Reviews {
		ids[leetgrinder.ReviewKey(item.Problem.Slug)] = uuid.NewString()
	}
	return ids
}
func (s *Server) leetgrinderOverview(w http.ResponseWriter, r *http.Request) {
	today, err := s.db.LeetgrinderToday(r.Context(), s.clock())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your saved progress is unavailable. Please retry."))
		return
	}
	renderLeetgrinder(w, r, 200, leetgrinder.Overview(leetgrinder.OverviewPage{Today: today, IDs: reviewIDs(map[string]string{}, today)}))
}
func (s *Server) leetgrinderReviews(w http.ResponseWriter, r *http.Request) {
	today, err := s.db.LeetgrinderToday(r.Context(), s.clock())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your review queue is unavailable. Please retry."))
		return
	}
	cards := leetgrinder.BuildCards(today.State.Attempts, today.Settings.Location())
	renderLeetgrinder(w, r, 200, leetgrinder.Reviews(leetgrinder.ReviewsPage{Today: today, Cards: cards, IDs: reviewIDs(map[string]string{}, today)}))
}
func leetgrinderRouteDay(r *http.Request) (leetgrinder.Day, bool) {
	n, err := strconv.Atoi(chi.URLParam(r, "day"))
	if err != nil {
		return leetgrinder.Day{}, false
	}
	return leetgrinder.FindDay(n)
}
func dayPage(day leetgrinder.Day, state leetgrinder.State, today *leetgrinder.Today, message string) leetgrinder.DayPage {
	ids := map[string]string{}
	for _, p := range day.Core {
		ids[p.Slug] = uuid.NewString()
	}
	for _, p := range day.Optional {
		ids[p.Slug] = uuid.NewString()
	}
	if today != nil {
		reviewIDs(ids, *today)
	}
	return leetgrinder.DayPage{Day: day, State: state, IDs: ids, Error: message, Today: today}
}
func (s *Server) leetgrinderDay(w http.ResponseWriter, r *http.Request) {
	day, ok := leetgrinderRouteDay(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	today, err := s.db.LeetgrinderToday(r.Context(), s.clock())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your saved progress is unavailable. Please retry."))
		return
	}
	renderLeetgrinder(w, r, 200, leetgrinder.Session(dayPage(day, today.State, &today, "")))
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
	form := leetgrinder.AttemptForm{ID: r.PostForm.Get("id"), Revision: r.PostForm.Get("revision"), Outcome: r.PostForm.Get("outcome"), Minutes: r.PostForm.Get("minutes"), Assisted: r.PostForm.Get("assisted") == "true", Notes: r.PostForm.Get("notes"), ReturnDay: returnDay, Review: r.PostForm.Get("review") == "true"}
	switch ret := r.PostForm.Get("return"); ret {
	case "overview", "reviews":
		form.Return = ret
	}
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
	_, err = s.db.SaveLeetgrinderAttempt(r.Context(), leetgrinder.Attempt{ID: form.ID, ProblemSlug: problem.Slug, Outcome: form.Outcome, Minutes: minutes, Assisted: form.Assisted, Notes: form.Notes, Source: "web", IsReview: form.Review}, form.Revision)
	if err != nil {
		switch {
		case errors.Is(err, database.ErrConflict) && form.Revision == "":
			// The ID already belongs to a different saved attempt, so retrying with it can never succeed.
			form.ID = uuid.NewString()
			reject(409, "This form was already used to save a different attempt. Your draft now has a new identifier; please save again.")
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
	anchor := problem.Slug
	if form.Review {
		anchor = leetgrinder.ReviewKey(problem.Slug)
	}
	target := leetgrinder.ProblemURL(problem.Slug)
	switch {
	case form.ReturnDay > 0:
		target = leetgrinder.DayURL(form.ReturnDay) + "#" + anchor
	case form.Return == "overview":
		target = "/leetgrinder#" + anchor
	case form.Return == "reviews":
		target = "/leetgrinder/reviews#" + anchor
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
		renderLeetgrinder(w, r, 503, leetgrinder.Session(dayPage(day, state, nil, "The session status could not be saved. Please retry; your attempts have not changed.")))
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

func (s *Server) leetgrinderSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.db.LeetgrinderSettings(r.Context())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your settings are unavailable. Please retry."))
		return
	}
	page := s.withNotify(r.Context(), leetgrinder.SettingsPage{Settings: settings, Now: s.clock(), Schedule: leetgrinder.NewScheduleForm(settings), Saved: r.URL.Query().Get("saved") != ""})
	page.Notify.TestSent = r.URL.Query().Get("tested") != ""
	renderLeetgrinder(w, r, 200, leetgrinder.SettingsView(s.withAPITokens(r, page)))
}

// settingsError carries a message that is safe to show next to the form.
type settingsError string

func (e settingsError) Error() string { return string(e) }

func (s *Server) leetgrinderSaveSchedule(w http.ResponseWriter, r *http.Request) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	form := leetgrinder.ScheduleForm{Start: strings.TrimSpace(r.PostForm.Get("start")), End: strings.TrimSpace(r.PostForm.Get("end")), Timezone: strings.TrimSpace(r.PostForm.Get("timezone")), Hours: r.PostForm.Get("hours"), Revision: r.PostForm.Get("revision")}
	reject := func(status int, message string) {
		settings, err := s.db.LeetgrinderSettings(r.Context())
		if err != nil {
			message += " Your settings could not be reloaded; your draft is retained below."
		}
		renderLeetgrinder(w, r, status, leetgrinder.SettingsView(s.withAPITokens(r, s.withNotify(r.Context(), leetgrinder.SettingsPage{Settings: settings, Now: s.clock(), Schedule: form, Error: message}))))
	}
	hours, err := strconv.ParseFloat(form.Hours, 64)
	if err != nil || !leetgrinder.ValidDailyHours(hours) {
		reject(400, "Choose between 2 and 4 hours in half-hour steps.")
		return
	}
	var start, end *time.Time
	for _, field := range []struct {
		value  string
		target **time.Time
		name   string
	}{{form.Start, &start, "start"}, {form.End, &end, "end"}} {
		if field.value == "" {
			continue
		}
		t, err := time.Parse(time.DateOnly, field.value)
		if err != nil {
			reject(400, "Enter the "+field.name+" date as YYYY-MM-DD.")
			return
		}
		*field.target = &t
	}
	_, err = s.db.UpdateLeetgrinderSettings(r.Context(), form.Revision, func(settings *leetgrinder.Settings) error {
		loc, err := leetgrinder.LoadTimezone(form.Timezone)
		if err != nil {
			return settingsError("Choose an IANA time zone such as America/Chicago.")
		}
		current := leetgrinder.NewScheduleForm(*settings)
		startChanged, endChanged := form.Start != current.Start, form.End != current.End
		switch {
		case start == nil && end == nil:
			settings.StartDate = nil
		case end != nil && (start == nil || (endChanged && !startChanged)):
			first := leetgrinder.ScheduleEndingOn(*end, loc).Start
			settings.StartDate = &first
		default:
			if end != nil && startChanged && endChanged && !leetgrinder.NewSchedule(*start, loc).End().Equal(*end) {
				return settingsError("Change the start date or the end date, not both. The schedule is always 84 consecutive days.")
			}
			first := leetgrinder.NewSchedule(*start, loc).Start
			settings.StartDate = &first
		}
		settings.Timezone, settings.DailyHours = form.Timezone, hours
		if err := settings.Validate(); err != nil {
			return settingsError("Check the schedule: " + err.Error() + ".")
		}
		return nil
	})
	var invalid settingsError
	switch {
	case err == nil:
		http.Redirect(w, r, "/leetgrinder/settings?saved=schedule#schedule", http.StatusSeeOther)
	case errors.As(err, &invalid):
		reject(400, invalid.Error())
	case errors.Is(err, database.ErrConflict):
		reject(409, "Settings changed since you opened this page. Reload settings to see the current values, then reapply your draft shown below.")
	case errors.Is(err, database.ErrInvalid):
		reject(400, "Reload the settings page and try again.")
	default:
		reject(503, "The schedule could not be saved. Your draft is retained; please retry.")
	}
}
