package server

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/michael-duren/career-strategy/internal/database"
	otelchimetric "github.com/riandyrn/otelchi/metric"
	"io"
	"net/http"
	"strconv"
)

func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func decode(w http.ResponseWriter, r *http.Request, limit int64, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	err := d.Decode(v)
	if err == nil {
		var extra any
		err = d.Decode(&extra)
		if err == io.EOF {
			return true
		}
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		respond(w, 413, map[string]string{"error": "Request too large"})
	} else {
		respond(w, 400, map[string]string{"error": "Invalid JSON request"})
	}
	return false
}
func failure(w http.ResponseWriter, err error) {
	code := 503
	switch {
	case errors.Is(err, database.ErrConflict):
		code = 409
	case errors.Is(err, database.ErrNotFound):
		code = 404
	case errors.Is(err, database.ErrInvalid):
		code = 400
	case errors.Is(err, database.ErrCore):
		code = 400
	}
	message := "Storage operation failed; please retry. Your draft is kept."
	if code != 503 {
		message = err.Error()
	}
	respond(w, code, map[string]string{"error": message})
}
func (s *Server) RegisterRoutes() http.Handler {
	r := chi.NewRouter()
	metricConfig := otelchimetric.NewBaseConfig("career-strategy")
	r.Use(
		otelchimetric.NewServerRequestDuration(metricConfig),
		otelchimetric.NewServerActiveRequests(metricConfig),
		otelchimetric.NewServerRequestBodySize(metricConfig),
		otelchimetric.NewServerResponseBodySize(metricConfig),
	)
	r.Use(middleware.Recoverer)
	r.Use(s.pageAccess)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { respond(w, 200, map[string]string{"status": "up"}) })
	r.Get("/readyz", s.healthHandler)
	r.Get("/health", s.healthHandler)
	r.Post("/api/auth/login", s.login)
	r.Group(func(r chi.Router) {
		r.Use(s.private)
		r.Get("/api/auth/verify", func(w http.ResponseWriter, r *http.Request) {
			respond(w, 200, map[string]any{"authenticated": true, "username": s.config.Username})
		})
		r.Post("/api/auth/logout", func(w http.ResponseWriter, r *http.Request) {
			if !s.mutation(w, r) {
				return
			}
			s.cookie(w, "", -1)
			respond(w, 200, map[string]bool{"authenticated": false})
		})
		r.Get("/api/entries/{kind}", s.readEntries)
		r.Post("/api/entries/{kind}", s.saveEntry)
		r.Delete("/api/entries/{kind}", s.deleteEntry)
		r.Post("/api/entries/{kind}/toggle", s.toggle)
		r.Get("/api/goals", s.readGoals)
		r.Post("/api/goals", s.saveGoal)
		r.Delete("/api/goals", s.deleteGoal)
		r.Get("/api/agent-context", s.agentContext)
		r.Get("/api/search", s.search)
		r.Get("/api/summaries/progress", func(w http.ResponseWriter, r *http.Request) {
			entries, err := s.db.ProgressSummary(r.Context())
			if err != nil {
				failure(w, err)
				return
			}
			respond(w, 200, map[string]any{"entries": entries})
		})
		r.Get("/api/export", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if err := s.db.Export(r.Context(), w); err != nil {
				panic(http.ErrAbortHandler)
			}
		})
	})
	s.registerStatic(r)
	return r
}
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	h := s.db.Health()
	status := 200
	if h["status"] != "up" {
		status = 503
	}
	respond(w, status, h)
}
func (s *Server) readEntries(w http.ResponseWriter, r *http.Request) {
	kind := chi.URLParam(r, "kind")
	q := r.URL.Query()
	if id := q.Get("id"); id != "" {
		if !database.ValidID(kind, id) {
			respond(w, 400, map[string]string{"error": "Invalid kind or ID"})
			return
		}
		v, err := s.db.Detail(r.Context(), kind, id)
		if err != nil {
			failure(w, err)
			return
		}
		respond(w, 200, v)
		return
	}
	limit, offset := 50, 0
	var err error
	if q.Has("limit") {
		limit, err = strconv.Atoi(q.Get("limit"))
		if err != nil || limit < 1 || limit > 100 {
			respond(w, 400, map[string]string{"error": "limit must be 1–100"})
			return
		}
	}
	if q.Has("offset") {
		offset, err = strconv.Atoi(q.Get("offset"))
		if err != nil || offset < 0 || offset > 1000000 {
			respond(w, 400, map[string]string{"error": "invalid offset"})
			return
		}
	}
	filter := database.Filter{Limit: limit, Offset: offset, Topic: q.Get("topic"), Status: q.Get("status"), Category: q.Get("category"), Priority: q.Get("priority"), From: q.Get("from"), To: q.Get("to")}
	var v database.Page
	if q.Get("view") == "detail" && (kind == "company" || kind == "book" || kind == "week") {
		v, err = s.db.ListDetails(r.Context(), kind, filter)
	} else if q.Has("view") {
		respond(w, 400, map[string]string{"error": "unsupported view"})
		return
	} else {
		v, err = s.db.List(r.Context(), kind, filter)
	}
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, v)
}
func (s *Server) saveEntry(w http.ResponseWriter, r *http.Request) {
	if !s.mutation(w, r) {
		return
	}
	var input struct {
		Entry    database.Entity `json:"entry"`
		Revision json.RawMessage `json:"revision"`
	}
	if !decode(w, r, 150000, &input) {
		return
	}
	rev, ok := parseRevision(input.Revision)
	if !ok {
		respond(w, 400, map[string]string{"error": "revision must be string or null"})
		return
	}
	kind := chi.URLParam(r, "kind")
	var validationErr error
	input.Entry, validationErr = database.PrepareSave(kind, input.Entry)
	if validationErr != nil {
		err := validationErr
		respond(w, 400, map[string]string{"error": err.Error()})
		return
	}
	v, err := s.db.Save(r.Context(), kind, input.Entry, rev)
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, v)
}
func parseRevision(raw json.RawMessage) (*string, bool) {
	if string(raw) == "null" {
		return nil, true
	}
	var v string
	if len(raw) == 0 || json.Unmarshal(raw, &v) != nil {
		return nil, false
	}
	return &v, true
}
func (s *Server) deleteEntry(w http.ResponseWriter, r *http.Request) {
	if !s.mutation(w, r) {
		return
	}
	var input struct {
		ID       string          `json:"id"`
		Revision json.RawMessage `json:"revision"`
	}
	if !decode(w, r, 2000, &input) {
		return
	}
	rev, ok := parseRevision(input.Revision)
	if !ok || !database.ValidID(chi.URLParam(r, "kind"), input.ID) {
		respond(w, 400, map[string]string{"error": "ID and revision required"})
		return
	}
	if err := s.db.Delete(r.Context(), chi.URLParam(r, "kind"), input.ID, rev); err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, map[string]bool{"deleted": true})
}
func (s *Server) toggle(w http.ResponseWriter, r *http.Request) {
	if !s.mutation(w, r) {
		return
	}
	var input struct {
		ID       string `json:"id"`
		Revision string `json:"revision"`
		Index    *int   `json:"index"`
		Checked  *bool  `json:"checked"`
	}
	if !decode(w, r, 2000, &input) {
		return
	}
	if input.ID == "" || input.Revision == "" || input.Index == nil || input.Checked == nil {
		respond(w, 400, map[string]string{"error": "id, revision, index and checked required"})
		return
	}
	v, err := s.db.Toggle(r.Context(), chi.URLParam(r, "kind"), input.ID, input.Revision, *input.Index, *input.Checked)
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, v)
}
