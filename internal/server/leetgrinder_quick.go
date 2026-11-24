package server

import (
	"context"
	"errors"
	"math/rand/v2"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func (s *Server) registerLeetgrinderQuick(r chi.Router) {
	r.Get("/leetgrinder/problem/{slug}/quick", s.leetgrinderQuick)
	r.Post("/leetgrinder/problem/{slug}/quick", s.leetgrinderQuickPost)
	r.Get("/leetgrinder/reviews/backlog", s.leetgrinderBacklog)
}

// randFloat draws from [0, 1) for spot-check picks; tests override it.
func (s *Server) randFloat() float64 {
	if s.rand != nil {
		return s.rand()
	}
	return rand.Float64()
}

// errQuickBlocked carries the learner-facing reason a problem cannot be
// quick-reviewed now.
type errQuickBlocked string

func (e errQuickBlocked) Error() string { return string(e) }

// prepareQuickReview checks that a's problem can be quick-reviewed now and,
// for a recalled review, decides whether it becomes a spot check at the
// current rate. It plans today first, like any attempt. A retry of a saved
// attempt skips the checks, so saving it again returns the saved attempt
// (or a conflict when its values differ).
func (s *Server) prepareQuickReview(ctx context.Context, a leetgrinder.Attempt) (leetgrinder.Attempt, error) {
	today, err := s.db.LeetgrinderToday(ctx, s.clock())
	if err != nil {
		return a, err
	}
	if slices.ContainsFunc(today.State.Attempts, func(saved leetgrinder.Attempt) bool { return saved.ID == a.ID }) {
		return a, nil
	}
	card, ok := today.Card(a.ProblemSlug)
	if reason := leetgrinder.QuickBlocked(today, card, ok); reason != "" {
		return a, errQuickBlocked(reason)
	}
	if a.Outcome == leetgrinder.QuickRecalled {
		rate := leetgrinder.SpotChecks(today.State, today.Settings.SpotCheckRate).Rate
		a.SpotCheck = leetgrinder.PickSpotCheck(rate, s.randFloat())
	}
	return a, nil
}

// quickBatch is the batch query or form value, kept only when it parses.
func quickBatch(value string) string {
	if _, ok := leetgrinder.ParseBatch(value); ok {
		return value
	}
	return ""
}

func (s *Server) leetgrinderQuick(w http.ResponseWriter, r *http.Request) {
	problem, ok := s.leetgrinderRouteProblem(w, r)
	if !ok {
		return
	}
	today, err := s.db.LeetgrinderToday(r.Context(), s.clock())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your review queue is unavailable. Please retry."))
		return
	}
	form := leetgrinder.NewQuickForm(uuid.NewString(), quickBatch(r.URL.Query().Get("batch")))
	renderLeetgrinder(w, r, 200, leetgrinder.QuickReview(leetgrinder.NewQuickReviewPage(today, problem, form)))
}

// leetgrinderQuickPost handles both steps of a quick review: step=reveal
// checks the answer and shows the last solution without saving anything;
// step=grade saves the quick review.
func (s *Server) leetgrinderQuickPost(w http.ResponseWriter, r *http.Request) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	problem, ok := s.leetgrinderRouteProblem(w, r)
	if !ok {
		return
	}
	form := leetgrinder.QuickForm{
		ID: r.PostForm.Get("id"), Approach: r.PostForm.Get("approach"), Grade: r.PostForm.Get("grade"), Missed: r.PostForm.Get("missed"),
		Minutes: r.PostForm.Get("minutes"), Batch: quickBatch(r.PostForm.Get("batch")),
		Time:  leetgrinder.ComplexityInput{Choice: r.PostForm.Get("timeComplexity"), Other: r.PostForm.Get("timeComplexityOther")},
		Space: leetgrinder.ComplexityInput{Choice: r.PostForm.Get("spaceComplexity"), Other: r.PostForm.Get("spaceComplexityOther")},
	}
	grading := r.PostForm.Get("step") == "grade"
	// render shows the form again: the answer step, or after Reveal the
	// grade step with the last solution.
	render := func(status int, message string, revealed bool) {
		form.Error = message
		today, err := s.db.LeetgrinderToday(r.Context(), s.clock())
		if err != nil {
			renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your review queue is unavailable. Please retry."))
			return
		}
		page := leetgrinder.NewQuickReviewPage(today, problem, form)
		if revealed && page.Blocked == "" {
			state, err := s.db.LeetgrinderProblemState(r.Context(), problem.Slug)
			if err != nil {
				renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your attempt history is unavailable. Please retry."))
				return
			}
			timeC, _ := leetgrinder.NormalizeComplexity(form.Time.Value())
			spaceC, _ := leetgrinder.NormalizeComplexity(form.Space.Value())
			page.Reveal(state.ProblemAttempts(problem.Slug), timeC, spaceC)
		}
		renderLeetgrinder(w, r, status, leetgrinder.QuickReview(page))
	}
	if _, err := uuid.Parse(form.ID); err != nil {
		form.ID = uuid.NewString()
		render(400, "Invalid review identifier. Your answer now has a new identifier; please submit again.", false)
		return
	}
	if utf8.RuneCountInString(form.Approach) > 2000 || !utf8.ValidString(form.Approach) || strings.ContainsRune(form.Approach, 0) {
		render(400, "Keep the approach to 2,000 characters or fewer.", false)
		return
	}
	timeC, errT := leetgrinder.NormalizeComplexity(form.Time.Value())
	spaceC, errS := leetgrinder.NormalizeComplexity(form.Space.Value())
	switch {
	case errT != nil || errS != nil:
		render(400, "Write complexity in big-O notation, such as O(m·n): start with O( and end with ), in 40 characters or fewer.", false)
		return
	case timeC == "" || spaceC == "":
		render(400, "Choose the time and space complexity before revealing your solution.", false)
		return
	}
	if !grading {
		form.Missed = ""
		render(200, "", true)
		return
	}
	minutes, err := strconv.Atoi(form.Minutes)
	if err != nil || minutes < 1 || minutes > 240 {
		render(400, "Enter a duration between 1 and 240 minutes.", true)
		return
	}
	if !leetgrinder.ValidOutcome(leetgrinder.ModeQuick, form.Grade) {
		render(400, "Choose recalled, partial, or forgot.", true)
		return
	}
	missed := ""
	if form.Grade == leetgrinder.QuickPartial {
		if form.Missed == "" || !leetgrinder.ValidMissed(form.Missed) {
			render(400, "Choose what you missed.", true)
			return
		}
		missed = form.Missed
	}
	attempt := leetgrinder.Attempt{ID: form.ID, ProblemSlug: problem.Slug, Mode: leetgrinder.ModeQuick, Outcome: form.Grade, Missed: missed, Minutes: minutes, Notes: strings.TrimSpace(form.Approach), Source: "web", TimeComplexity: timeC, SpaceComplexity: spaceC}
	if err := attempt.NormalizeDetails(); err != nil {
		render(400, "Check your answer and try again.", true)
		return
	}
	attempt, err = s.prepareQuickReview(r.Context(), attempt)
	var blocked errQuickBlocked
	switch {
	case errors.As(err, &blocked):
		render(409, "", false)
		return
	case err != nil:
		render(503, "The quick review could not be saved. Your answer is retained; please retry.", true)
		return
	}
	if _, err = s.db.SaveLeetgrinderAttempt(r.Context(), attempt, ""); err != nil {
		switch {
		case errors.Is(err, database.ErrConflict):
			form.ID = uuid.NewString()
			render(409, "This form was already used to save a different review. Your answer now has a new identifier; please save again.", true)
		case errors.Is(err, database.ErrInvalid):
			render(400, "Check your answer and try again.", true)
		default:
			render(503, "The quick review could not be saved. Your answer is retained; please retry.", true)
		}
		return
	}
	target := "/leetgrinder/reviews#" + leetgrinder.ReviewKey(problem.Slug)
	if form.Batch != "" {
		target = leetgrinder.BacklogURL(form.Batch)
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

// leetgrinderBacklog shows a Clear backlog batch. Without a batch it starts
// one with today's counts and redirects to it, so the counts at the start
// travel in the URL.
func (s *Server) leetgrinderBacklog(w http.ResponseWriter, r *http.Request) {
	today, err := s.db.LeetgrinderToday(r.Context(), s.clock())
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your review queue is unavailable. Please retry."))
		return
	}
	batch, ok := leetgrinder.ParseBatch(r.URL.Query().Get("batch"))
	if !ok {
		http.Redirect(w, r, leetgrinder.BacklogURL(leetgrinder.StartBatch(today).String()), http.StatusSeeOther)
		return
	}
	renderLeetgrinder(w, r, 200, leetgrinder.Backlog(leetgrinder.NewBacklogPage(today, batch)))
}
