package server

import (
	"errors"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/michael-duren/career-strategy/internal/database"
)

var staticPages = map[string]string{
	"/running": "running/index.html",
	"/": "index.html", "/login": "login/index.html", "/books": "books/index.html",
	"/companies": "companies/index.html", "/logs": "logs/index.html", "/progress": "progress/index.html",
	"/goals/graph": "goals/graph/index.html", "/agents": "agents/index.html", "/timeline": "timeline/index.html", "/journal": "journal/index.html",
	"/personal-journal": "personal-journal/index.html", "/notes": "notes/index.html",
	"/documents": "documents/index.html", "/manage/books": "manage/books/index.html",
	"/manage/companies": "manage/companies/index.html",
}

var referenceAliases = map[string]string{
	"/2026/algorithms": "/2026/career-study-plan", "/2026/algorithms/leetcode": "/2026/career-study-plan",
	"/2026/algorithms/mit-6006": "/2026/career-study-plan", "/2026/algorithms/mit-6042j": "/2026/career-study-plan",
	"/2026/os-oss/podman-contributions": "/2026/career-study-plan",
}

func (s *Server) staticFile(w http.ResponseWriter, r *http.Request, name string, private bool) {
	clean := filepath.Clean(name)
	root := filepath.Clean(s.config.StaticDir)
	full := filepath.Join(root, clean)
	if clean == "." || strings.HasPrefix(clean, "..") || (full != root && !strings.HasPrefix(full, root+string(os.PathSeparator))) {
		http.NotFound(w, r)
		return
	}
	f, err := os.Open(full)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	if private {
		w.Header().Set("Cache-Control", "private, no-store")
	} else if strings.HasPrefix(r.URL.Path, "/_astro/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
	if kind := mime.TypeByExtension(filepath.Ext(clean)); kind != "" {
		w.Header().Set("Content-Type", kind)
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}

func (s *Server) page(w http.ResponseWriter, r *http.Request) {
	requestPath := r.URL.Path
	if requestPath != "/" {
		requestPath = strings.TrimSuffix(requestPath, "/")
	}
	name, ok := staticPages[requestPath]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if requestPath == "/manage/companies" {
		if id := r.URL.Query().Get("id"); id != "" {
			if !database.ValidID("company", id) {
				http.NotFound(w, r)
				return
			}
			http.Redirect(w, r, "/companies/"+strings.Join(strings.Split(id, "/"), "/"), http.StatusFound)
			return
		}
	}
	s.staticFile(w, r, name, r.URL.Path != "/login")
}

func (s *Server) dynamicPage(kind, prefix, shell string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, prefix)
		decoded, err := url.PathUnescape(id)
		if err != nil || decoded == "" || !database.ValidID(kind, decoded) {
			http.NotFound(w, r)
			return
		}
		if _, err = s.db.Detail(r.Context(), kind, decoded); err != nil {
			if errors.Is(err, database.ErrNotFound) || errors.Is(err, database.ErrInvalid) {
				http.NotFound(w, r)
				return
			}
			failure(w, err)
			return
		}
		s.staticFile(w, r, shell, true)
	}
}

func (s *Server) asset(w http.ResponseWriter, r *http.Request) {
	clean := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if clean == "." || (!strings.HasPrefix(clean, "_astro/") && !strings.HasPrefix(clean, "icons/") && !strings.Contains(path.Base(clean), ".")) {
		http.NotFound(w, r)
		return
	}
	s.staticFile(w, r, clean, false)
}

func (s *Server) registerStatic(r chi.Router) {
	for route := range staticPages {
		r.Get(route, s.page)
		if route != "/" {
			r.Get(route+"/", s.page)
		}
	}
	for from, to := range referenceAliases {
		target := to
		r.Get(from, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target, http.StatusFound) })
	}
	for _, id := range []string{"2026/career-study-plan", "2026/languages/index", "2026/random/vintage-computers", "2026/system-design/index", "2026/system-design/alex-xu-vol1", "2026/system-design/ddia-read", "2026/system-design/hello-interview", "2026/os-oss/index", "2026/os-oss/build-runtime", "2026/os-oss/conference-talk", "2026/os-oss/container-internals", "2026/os-oss/ostep"} {
		route := "/" + strings.TrimSuffix(id, "/index")
		docID := id
		r.Get(route, func(w http.ResponseWriter, request *http.Request) {
			if _, err := s.db.Detail(request.Context(), "document", docID); err != nil {
				if errors.Is(err, database.ErrNotFound) {
					http.NotFound(w, request)
				} else {
					failure(w, err)
				}
				return
			}
			s.staticFile(w, request, "static-shells/document/index.html", true)
		})
	}
	r.Get("/notes/*", s.dynamicPage("note", "/notes/", "static-shells/note/index.html"))
	r.Get("/documents/*", s.dynamicPage("document", "/documents/", "static-shells/document/index.html"))
	r.Get("/companies/*", s.dynamicPage("company", "/companies/", "static-shells/company/index.html"))
	r.Get("/*", s.asset)
}
