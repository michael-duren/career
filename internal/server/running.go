package server

import (
	"bytes"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/database"
	"go.opentelemetry.io/otel"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"
)

func (s *Server) uploadRunningClip(w http.ResponseWriter, r *http.Request) {
	if !s.mutation(w, r, "multipart/form-data") {
		return
	}
	ctx, span := otel.Tracer("career/running").Start(r.Context(), "running.upload")
	defer span.End()
	r.Body = http.MaxBytesReader(w, r.Body, (15<<20)+(64<<10))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			respond(w, 413, map[string]string{"error": "Clip exceeds 15 MiB"})
		} else {
			failure(w, database.ErrInvalid)
		}
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, h, err := r.FormFile("audio")
	if err != nil {
		failure(w, database.ErrInvalid)
		return
	}
	defer f.Close()
	audio, err := io.ReadAll(io.LimitReader(f, (15<<20)+1))
	if err != nil {
		failure(w, err)
		return
	}
	if len(audio) > 15<<20 {
		respond(w, 413, map[string]string{"error": "Clip exceeds 15 MiB"})
		return
	}
	recorded, err := time.Parse(time.RFC3339Nano, r.FormValue("recordedAt"))
	if err != nil {
		failure(w, database.ErrInvalid)
		return
	}
	duration, err := strconv.Atoi(r.FormValue("durationMs"))
	if err != nil {
		failure(w, database.ErrInvalid)
		return
	}
	contentType, _, err := mime.ParseMediaType(h.Header.Get("Content-Type"))
	if err != nil {
		failure(w, database.ErrInvalid)
		return
	}
	clip, err := s.db.AddRunningClip(ctx, database.RunningClip{ID: r.FormValue("clientId"), NoteID: r.FormValue("noteId"), RecordedAt: recorded, DurationMs: duration, MIME: contentType, Audio: audio})
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, clip)
}
func runningIDs(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	note, id := chi.URLParam(r, "noteId"), chi.URLParam(r, "clipId")
	_, err := uuid.Parse(id)
	if !database.ValidID("run", note) || (id != "" && err != nil) {
		failure(w, database.ErrInvalid)
		return "", "", false
	}
	return note, id, true
}
func (s *Server) runningAudio(w http.ResponseWriter, r *http.Request) {
	note, id, ok := runningIDs(w, r)
	if !ok {
		return
	}
	c, err := s.db.RunningAudio(r.Context(), note, id)
	if err != nil {
		failure(w, err)
		return
	}
	w.Header().Set("Content-Type", c.MIME)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, id, time.Time{}, bytes.NewReader(c.Audio))
}
func (s *Server) runningStatus(w http.ResponseWriter, r *http.Request) {
	note, _, ok := runningIDs(w, r)
	if !ok {
		return
	}
	clips, err := s.db.RunningClips(r.Context(), note)
	if err != nil {
		failure(w, err)
		return
	}
	// The window lets the client find unsent takes the server would group into this thought.
	respond(w, 200, map[string]any{"clips": clips, "groupWindowMs": database.RunningGroupWindow.Milliseconds()})
}
func (s *Server) retranscribe(w http.ResponseWriter, r *http.Request) {
	if !s.mutation(w, r) {
		return
	}
	note, id, ok := runningIDs(w, r)
	if !ok {
		return
	}
	if err := s.db.Retranscribe(r.Context(), note, id); err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, map[string]string{"status": "pending"})
}
