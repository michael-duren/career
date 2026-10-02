package server

import (
	"bytes"
	"errors"
	"net/http"
	"slices"
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
		// The 84-day curriculum's pages are retired.
		retired := func(w http.ResponseWriter, r *http.Request) {
			renderLeetgrinder(w, r, http.StatusGone, leetgrinder.Retired())
		}
		r.HandleFunc("/leetgrinder/day/*", retired)
		r.HandleFunc("/leetgrinder/about", retired)
		r.Get("/leetgrinder/log", s.leetgrinderLog)
		r.Get("/leetgrinder/problem/{slug}", s.leetgrinderProblem)
		r.Post("/leetgrinder/problem/{slug}/attempts", s.leetgrinderAttempt)
		r.Get("/leetgrinder/problems", s.leetgrinderProblems)
		r.Get("/leetgrinder/todos", s.leetgrinderTodos)
		r.Get("/leetgrinder/todos/add", s.leetgrinderTodoAddProblem)
		r.Get("/leetgrinder/todos/sets/new", s.leetgrinderTodoAddSet)
		r.Post("/leetgrinder/todos/sets", s.leetgrinderCreateTodoSet)
		r.Get("/leetgrinder/todos/sets/{id}", s.leetgrinderTodoSet)
		r.Post("/leetgrinder/todos/items", s.leetgrinderAddTodoItem)
		r.Post("/leetgrinder/todos/sets/{id}/delete", s.leetgrinderDeleteTodoSet)
		r.Post("/leetgrinder/todos/items/{id}/delete", s.leetgrinderDeleteTodoItem)
		r.Get("/leetgrinder/reviews", s.leetgrinderReviews)
		r.Get("/leetgrinder/stats", s.leetgrinderStats)
		r.Get("/leetgrinder/settings", s.leetgrinderSettings)
		r.Post("/leetgrinder/settings/general", s.leetgrinderSaveGeneral)
		s.registerLeetgrinderNotify(r)
		s.registerLeetgrinderAnalysis(r)
		r.Get("/leetgrinder/export", s.leetgrinderExport)
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
	s.renderOverview(w, r, 200, "", "")
}

// renderOverview renders the dashboard, keeping a rejected "Log an attempt"
// entry and its message.
func (s *Server) renderOverview(w http.ResponseWriter, r *http.Request, status int, ref, message string) {
	today, err := s.db.LeetgrinderToday(r.Context(), s.clock())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your saved progress is unavailable. Please retry."))
		return
	}
	nextTodos, err := s.db.LeetgrinderNextTodoItems(r.Context(), 5)
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your todos are unavailable. Please retry."))
		return
	}
	page := leetgrinder.OverviewPage{Today: today, IDs: reviewIDs(map[string]string{}, today), NextTodos: nextTodos, LogRef: ref, LogError: message}
	renderLeetgrinder(w, r, status, leetgrinder.Overview(page))
}

// leetgrinderLog opens the problem page for a LeetCode URL or slug.
func (s *Server) leetgrinderLog(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("problem")
	slug, ok := leetgrinder.NormalizeProblemRef(ref)
	if !ok {
		if utf8.RuneCountInString(ref) > 300 {
			ref = string([]rune(ref)[:300])
		}
		s.renderOverview(w, r, 400, ref, "Enter a LeetCode problem link, such as https://leetcode.com/problems/two-sum/, or a slug such as two-sum.")
		return
	}
	http.Redirect(w, r, leetgrinder.ProblemURL(slug), http.StatusSeeOther)
}

func (s *Server) leetgrinderProblems(w http.ResponseWriter, r *http.Request) {
	state, err := s.db.LeetgrinderState(r.Context())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your saved progress is unavailable. Please retry."))
		return
	}
	settings, err := s.db.LeetgrinderSettings(r.Context())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your settings are unavailable. Please retry."))
		return
	}
	now, loc := s.clock(), settings.Location()
	rows := leetgrinder.ProblemRows(state, leetgrinder.BuildCards(state, loc), now, loc)
	filter := leetgrinder.ParseProblemFilter(r.URL.Query())
	renderLeetgrinder(w, r, 200, leetgrinder.Problems(leetgrinder.ProblemsPage{Rows: leetgrinder.FilterProblems(rows, filter), Total: len(rows), Topics: leetgrinder.ProblemTopics(rows), Filter: filter, Location: loc}))
}
func (s *Server) leetgrinderReviews(w http.ResponseWriter, r *http.Request) {
	today, err := s.db.LeetgrinderToday(r.Context(), s.clock())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your review queue is unavailable. Please retry."))
		return
	}
	renderLeetgrinder(w, r, 200, leetgrinder.Reviews(leetgrinder.ReviewsPage{Today: today, IDs: reviewIDs(map[string]string{}, today)}))
}

// statsWeeks is how many weeks the goal calendar shows.
const statsWeeks = 12

func (s *Server) leetgrinderStats(w http.ResponseWriter, r *http.Request) {
	today, err := s.db.LeetgrinderToday(r.Context(), s.clock())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your stats are unavailable. Please retry."))
		return
	}
	renderLeetgrinder(w, r, 200, leetgrinder.Stats(leetgrinder.NewStatsPage(today, leetgrinder.ParseStatsFilter(r.URL.Query()), statsWeeks)))
}

// leetgrinderRouteProblem validates the route's slug and returns its catalog
// row, or a bare problem for a slug not in the catalog. Only saving an
// attempt adds a row, so a GET never queues a LeetCode fetch.
func (s *Server) leetgrinderRouteProblem(w http.ResponseWriter, r *http.Request) (leetgrinder.Problem, bool) {
	slug := chi.URLParam(r, "slug")
	if !leetgrinder.ValidSlug(slug) {
		http.NotFound(w, r)
		return leetgrinder.Problem{}, false
	}
	problem, err := s.db.LeetgrinderProblem(r.Context(), slug)
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("This problem is unavailable. Please retry."))
		return leetgrinder.Problem{}, false
	}
	return problem, true
}

func (s *Server) leetgrinderProblem(w http.ResponseWriter, r *http.Request) {
	problem, ok := s.leetgrinderRouteProblem(w, r)
	if !ok {
		return
	}
	state, err := s.db.LeetgrinderState(r.Context())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your attempt history is unavailable. Please retry."))
		return
	}
	renderLeetgrinder(w, r, 200, leetgrinder.ProblemHistory(problem, state, leetgrinder.NewForm(uuid.NewString()), s.historyAnalysis(r), s.problemReview(r, state, problem.Slug)))
}

// problemReview is slug's review state for its page. It reads today without
// freezing today's goal or plan; a settings failure only hides it.
func (s *Server) problemReview(r *http.Request, state leetgrinder.State, slug string) leetgrinder.ProblemReview {
	settings, err := s.db.LeetgrinderSettings(r.Context())
	if err != nil {
		return leetgrinder.ProblemReview{}
	}
	return leetgrinder.NewProblemReview(leetgrinder.NewToday(settings, state, s.clock()), slug)
}

// historyAnalysis reports whether analysis runs, for the history page. A
// settings failure is reported as an unknown status on the cards.
func (s *Server) historyAnalysis(r *http.Request) leetgrinder.AnalysisAvailability {
	a := leetgrinder.AnalysisAvailability{KeyConfigured: s.config.AnthropicAPIKey.Reveal() != "", ZeroLimit: s.config.AnalysisDailyLimit <= 0}
	settings, err := s.db.LeetgrinderSettings(r.Context())
	if err != nil {
		a.Unknown = true
		return a
	}
	a.Enabled = settings.AnalysisEnabled
	if !a.On() {
		return a
	}
	now := s.clock()
	// A failed read is an unknown status, so re-analyse is refused rather
	// than clearing a result while analysis may be on hold.
	pause, err := s.db.LeetgrinderAnalysisPause(r.Context())
	if err != nil {
		a.Unknown = true
		return a
	}
	a.Paused = now.Before(pause.Until)
	used, err := s.db.LeetgrinderAnalysisUsage(r.Context(), leetgrinder.Date(now, settings.Location()))
	if err != nil {
		a.Unknown = true
		return a
	}
	a.LimitReached = used >= s.config.AnalysisDailyLimit
	return a
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
	problem, ok := s.leetgrinderRouteProblem(w, r)
	if !ok {
		return
	}
	form := leetgrinder.AttemptForm{ID: r.PostForm.Get("id"), Revision: r.PostForm.Get("revision"), Outcome: r.PostForm.Get("outcome"), Minutes: r.PostForm.Get("minutes"), Assisted: r.PostForm.Get("assisted") == "true", Notes: r.PostForm.Get("notes"), Review: r.PostForm.Get("review") == "true",
		WantsReview: r.PostForm.Get("wantsReview") == "true", Approach: r.PostForm.Get("approach"),
		Time:  leetgrinder.ComplexityInput{Choice: r.PostForm.Get("timeComplexity"), Other: r.PostForm.Get("timeComplexityOther")},
		Space: leetgrinder.ComplexityInput{Choice: r.PostForm.Get("spaceComplexity"), Other: r.PostForm.Get("spaceComplexityOther")}}
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
		renderLeetgrinder(w, r, status, leetgrinder.ProblemHistory(problem, state, form, s.historyAnalysis(r), leetgrinder.ProblemReview{}))
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
	if v := r.PostForm.Get("wantsReview"); v != "" && v != "true" && v != "false" {
		reject(400, "Choose whether you want to review this problem.")
		return
	}
	if !leetgrinder.ValidApproach(form.Approach) {
		reject(400, "Choose whether you reached the optimal solution or took a simpler approach.")
		return
	}
	attempt := leetgrinder.Attempt{ID: form.ID, ProblemSlug: problem.Slug, Outcome: form.Outcome, Minutes: minutes, Assisted: form.Assisted, Notes: form.Notes, Source: "web", WantsReview: form.WantsReview, Approach: form.Approach, TimeComplexity: form.Time.Value(), SpaceComplexity: form.Space.Value()}
	switch err := attempt.NormalizeDetails(); {
	case errors.Is(err, leetgrinder.ErrComplexityRequired):
		reject(400, "Choose the time and space complexity of your solution. They are required for solved and struggled attempts.")
		return
	case err != nil:
		reject(400, "Write complexity in big-O notation, such as O(m·n): start with O( and end with ), in 40 characters or fewer.")
		return
	}
	if form.Revision == "" {
		// Freeze today's goal and picks first, as the extension API does.
		_, _ = s.db.LeetgrinderToday(r.Context(), s.clock())
	}
	_, err = s.db.SaveLeetgrinderAttempt(r.Context(), attempt, form.Revision)
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
	case form.Return == "overview":
		target = "/leetgrinder#" + anchor
	case form.Return == "reviews":
		target = "/leetgrinder/reviews#" + anchor
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
func (s *Server) leetgrinderSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.db.LeetgrinderSettings(r.Context())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your settings are unavailable. Please retry."))
		return
	}
	page := s.withNotify(r.Context(), leetgrinder.SettingsPage{Settings: settings, Now: s.clock(), General: leetgrinder.NewGeneralForm(settings), Saved: r.URL.Query().Get("saved") != ""})
	page.TodayGoal = s.todayGoal(r, settings)
	page.Notify.TestSent = r.URL.Query().Get("tested") != ""
	renderLeetgrinder(w, r, 200, leetgrinder.SettingsView(s.withAPITokens(r, page)))
}

// settingsError carries a message that is safe to show next to the form.
type settingsError string

func (e settingsError) Error() string { return string(e) }

// todayGoal is today's frozen goal, or nil when today has none yet or it
// cannot be read.
func (s *Server) todayGoal(r *http.Request, settings leetgrinder.Settings) *leetgrinder.DailyGoal {
	if settings.Revision == "" {
		return nil
	}
	goal, ok, err := s.db.LeetgrinderDailyGoal(r.Context(), leetgrinder.Date(s.clock(), settings.Location()))
	if err != nil || !ok {
		return nil
	}
	return &goal
}

func (s *Server) leetgrinderSaveGeneral(w http.ResponseWriter, r *http.Request) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	form := leetgrinder.GeneralForm{Timezone: strings.TrimSpace(r.PostForm.Get("timezone")), GoalNew: r.PostForm.Get("goalNew"), GoalReview: r.PostForm.Get("goalReview"), Revision: r.PostForm.Get("revision")}
	reject := func(status int, message string) {
		settings, err := s.db.LeetgrinderSettings(r.Context())
		if err != nil {
			message += " Your settings could not be reloaded; your draft is retained below."
		}
		page := leetgrinder.SettingsPage{Settings: settings, Now: s.clock(), General: form, Error: message, TodayGoal: s.todayGoal(r, settings)}
		renderLeetgrinder(w, r, status, leetgrinder.SettingsView(s.withAPITokens(r, s.withNotify(r.Context(), page))))
	}
	goalNew, errNew := strconv.Atoi(form.GoalNew)
	goalReview, errReview := strconv.Atoi(form.GoalReview)
	goal := leetgrinder.DailyGoal{New: goalNew, Review: goalReview}
	if errNew != nil || errReview != nil || goal.Validate() != nil {
		reject(400, "Choose daily targets from 0 to 10, with at least one above 0.")
		return
	}
	_, err := s.db.UpdateLeetgrinderSettings(r.Context(), form.Revision, func(settings *leetgrinder.Settings) error {
		if _, err := leetgrinder.LoadTimezone(form.Timezone); err != nil {
			return settingsError("Choose an IANA time zone such as America/Chicago.")
		}
		settings.Timezone, settings.Goal = form.Timezone, goal
		if err := settings.Validate(); err != nil {
			return settingsError("Check the settings: " + err.Error() + ".")
		}
		return nil
	})
	var invalid settingsError
	switch {
	case err == nil:
		http.Redirect(w, r, "/leetgrinder/settings?saved=general#general", http.StatusSeeOther)
	case errors.As(err, &invalid):
		reject(400, invalid.Error())
	case errors.Is(err, database.ErrConflict):
		reject(409, "Settings changed since you opened this page. Reload settings to see the current values, then reapply your draft shown below.")
	case errors.Is(err, database.ErrInvalid):
		reject(400, "Reload the settings page and try again.")
	default:
		reject(503, "Your settings could not be saved. Your draft is retained; please retry.")
	}
}

// leetgrinderExport is the attempt history download: every attempt, and the
// catalog rows of attempted problems.
func (s *Server) leetgrinderExport(w http.ResponseWriter, r *http.Request) {
	state, err := s.db.LeetgrinderState(r.Context())
	if err != nil {
		failure(w, err)
		return
	}
	type exportProblem struct {
		Slug          string   `json:"slug"`
		Number        int      `json:"number,omitempty"`
		Title         string   `json:"title"`
		Difficulty    string   `json:"difficulty"`
		Topics        []string `json:"topics"`
		OptimalTime   string   `json:"optimalTime"`
		OptimalSpace  string   `json:"optimalSpace"`
		OptimalNote   string   `json:"optimalNote"`
		OptimalSource string   `json:"optimalSource"`
	}
	problems := []exportProblem{}
	seen := map[string]bool{}
	for _, a := range state.Attempts {
		if seen[a.ProblemSlug] {
			continue
		}
		seen[a.ProblemSlug] = true
		p := state.Problem(a.ProblemSlug)
		topics := p.Topics
		if topics == nil {
			topics = []string{}
		}
		problems = append(problems, exportProblem{p.Slug, p.Number, p.Title, p.Difficulty, topics, p.OptimalTime, p.OptimalSpace, p.OptimalNote, p.OptimalSource})
	}
	slices.SortFunc(problems, func(a, b exportProblem) int { return strings.Compare(a.Slug, b.Slug) })
	w.Header().Set("Content-Disposition", `attachment; filename="leetgrinder-history.json"`)
	type exportGoal struct {
		Date   string `json:"date"`
		New    int    `json:"new"`
		Review int    `json:"review"`
	}
	goals := []exportGoal{}
	for date, g := range state.Goals {
		goals = append(goals, exportGoal{date.Format(time.DateOnly), g.New, g.Review})
	}
	slices.SortFunc(goals, func(a, b exportGoal) int { return strings.Compare(a.Date, b.Date) })
	respond(w, 200, map[string]any{"attempts": state.Attempts, "problems": problems, "dailyGoals": goals})
}
