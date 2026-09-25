package server

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/linkedin"
)

const maxPhotoBytes = 2 << 20

// readConnections returns every matching connection in one response; the
// connections page filters and queues client-side.
func (s *Server) readConnections(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	company := q.Get("company")
	if company != "" && !database.ValidID("company", company) {
		failure(w, database.ErrInvalid)
		return
	}
	connections := []database.Entity{}
	revisions := map[string]string{}
	for offset := 0; ; {
		page, err := s.db.List(r.Context(), "connection", database.Filter{Limit: 100, Offset: offset, Company: company, Linked: q.Get("linked") == "1"})
		if err != nil {
			failure(w, err)
			return
		}
		for _, item := range page.Entries {
			connections = append(connections, item.Entry)
			revisions[item.Entry["id"].(string)] = item.Revision
		}
		if page.NextOffset == nil {
			break
		}
		offset = *page.NextOffset
	}
	respond(w, 200, map[string]any{"connections": connections, "revisions": revisions})
}

func multipartForm(w http.ResponseWriter, r *http.Request, limit int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			respond(w, 413, map[string]string{"error": "Upload too large"})
		} else {
			failure(w, database.ErrInvalid)
		}
		return false
	}
	return true
}

func (s *Server) importConnections(w http.ResponseWriter, r *http.Request) {
	if !s.mutation(w, r, "multipart/form-data") || !multipartForm(w, r, 64<<20) {
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, _, err := r.FormFile("connections")
	if err != nil {
		respond(w, 400, map[string]string{"error": "Choose the Connections.csv file from your LinkedIn export."})
		return
	}
	defer f.Close()
	rows, err := linkedin.ParseConnections(f)
	if err != nil {
		respond(w, 400, map[string]string{"error": "Connections.csv could not be read: " + err.Error()})
		return
	}
	messages := map[string]string{}
	if m, _, err := r.FormFile("messages"); err == nil {
		defer m.Close()
		if messages, err = linkedin.ParseLastMessages(m); err != nil {
			respond(w, 400, map[string]string{"error": "messages.csv could not be read: " + err.Error()})
			return
		}
	} else if !errors.Is(err, http.ErrMissingFile) {
		failure(w, database.ErrInvalid)
		return
	}
	summary, err := s.db.ImportConnections(r.Context(), rows, messages)
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, summary)
}

var photoTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/webp": true, "image/gif": true}

func (s *Server) uploadConnectionPhoto(w http.ResponseWriter, r *http.Request) {
	if !s.mutation(w, r, "multipart/form-data") || !multipartForm(w, r, maxPhotoBytes+(64<<10)) {
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, _, err := r.FormFile("photo")
	if err != nil {
		failure(w, database.ErrInvalid)
		return
	}
	defer f.Close()
	data, err := readPhoto(f)
	if err != nil {
		respond(w, 413, map[string]string{"error": "Photo exceeds 2 MiB"})
		return
	}
	// Trust the bytes, not the client's declared type.
	kind := http.DetectContentType(data)
	if !photoTypes[kind] {
		respond(w, 400, map[string]string{"error": "Use a JPEG, PNG, WebP or GIF image"})
		return
	}
	rev, err := s.db.SetConnectionPhoto(r.Context(), r.FormValue("id"), kind, data)
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, map[string]string{"photo": rev})
}

func readPhoto(f multipart.File) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(f, maxPhotoBytes+1))
	if err == nil && (len(data) == 0 || len(data) > maxPhotoBytes) {
		err = errors.New("photo size")
	}
	return data, err
}

func (s *Server) connectionPhoto(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	p, err := s.db.ConnectionPhoto(r.Context(), id)
	if err != nil {
		failure(w, err)
		return
	}
	w.Header().Set("Content-Type", p.MIME)
	// Clients request ?v=<revision>, so a new upload changes the URL.
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'")
	http.ServeContent(w, r, id, time.Time{}, bytes.NewReader(p.Data))
}

func (s *Server) deleteConnectionPhoto(w http.ResponseWriter, r *http.Request) {
	if !s.mutation(w, r) {
		return
	}
	var input struct {
		ID string `json:"id"`
	}
	if !decode(w, r, 2000, &input) {
		return
	}
	if err := s.db.DeleteConnectionPhoto(r.Context(), input.ID); err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, map[string]bool{"deleted": true})
}
