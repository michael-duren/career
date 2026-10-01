package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func (s *Server) todoPage(w http.ResponseWriter, r *http.Request, status int, page leetgrinder.TodosPage) {
	var err error
	page.Sets, err = s.db.LeetgrinderTodoSets(r.Context())
	if err == nil {
		page.Standalone, err = s.db.LeetgrinderTodoItems(r.Context(), "")
	}
	if err != nil {
		renderLeetgrinder(w, r, 503, leetgrinder.Unavailable("Your todos are unavailable. Please retry."))
		return
	}
	renderLeetgrinder(w, r, status, leetgrinder.Todos(page))
}

func (s *Server) leetgrinderTodos(w http.ResponseWriter, r *http.Request) {
	s.todoPage(w, r, 200, leetgrinder.TodosPage{})
}

func (s *Server) leetgrinderCreateTodoSet(w http.ResponseWriter, r *http.Request) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	title, raw := r.PostForm.Get("title"), r.PostForm.Get("problems")
	slugs, ok := leetgrinder.ParseTodoRefs(raw)
	if !ok {
		s.todoPage(w, r, 400, leetgrinder.TodosPage{Title: title, Problems: raw, Error: "Enter at most 200 valid LeetCode links or slugs, one per line."})
		return
	}
	if _, err := s.db.CreateLeetgrinderTodoSet(r.Context(), title, slugs); err != nil {
		if errors.Is(err, database.ErrInvalid) {
			s.todoPage(w, r, 400, leetgrinder.TodosPage{Title: title, Problems: raw, Error: "Enter a set name of at most 120 characters and valid problems."})
			return
		}
		s.todoPage(w, r, 503, leetgrinder.TodosPage{Title: title, Problems: raw, Error: "The set could not be saved. Please retry."})
		return
	}
	http.Redirect(w, r, "/leetgrinder/todos", http.StatusSeeOther)
}

func (s *Server) leetgrinderAddTodoItem(w http.ResponseWriter, r *http.Request) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	ref, setID := r.PostForm.Get("problem"), r.PostForm.Get("setID")
	slug, ok := leetgrinder.NormalizeProblemRef(ref)
	if !ok {
		s.todoPage(w, r, 400, leetgrinder.TodosPage{Problem: ref, SetID: setID, Error: "Enter a valid LeetCode link or slug."})
		return
	}
	if _, err := s.db.AddLeetgrinderTodoItem(r.Context(), setID, slug); err != nil {
		status, message := 503, "The problem could not be saved. Please retry."
		if errors.Is(err, database.ErrInvalid) || errors.Is(err, database.ErrNotFound) {
			status, message = 400, "Choose an existing set."
		}
		s.todoPage(w, r, status, leetgrinder.TodosPage{Problem: ref, SetID: setID, Error: message})
		return
	}
	http.Redirect(w, r, "/leetgrinder/todos", http.StatusSeeOther)
}

func (s *Server) leetgrinderDeleteTodoSet(w http.ResponseWriter, r *http.Request) {
	s.deleteTodo(w, r, true)
}

func (s *Server) leetgrinderDeleteTodoItem(w http.ResponseWriter, r *http.Request) {
	s.deleteTodo(w, r, false)
}

func (s *Server) deleteTodo(w http.ResponseWriter, r *http.Request, set bool) {
	if !s.leetgrinderForm(w, r) {
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	var err error
	if set {
		err = s.db.DeleteLeetgrinderTodoSet(r.Context(), id)
	} else {
		err = s.db.DeleteLeetgrinderTodoItem(r.Context(), id)
	}
	if errors.Is(err, database.ErrInvalid) || errors.Is(err, database.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.todoPage(w, r, 503, leetgrinder.TodosPage{Error: "The todo could not be removed. Please retry."})
		return
	}
	http.Redirect(w, r, "/leetgrinder/todos", http.StatusSeeOther)
}
