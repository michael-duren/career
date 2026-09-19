// Package running transcribes exclusively through the configured local whisper server.
package running

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/michael-duren/career-strategy/internal/database"
	"go.opentelemetry.io/otel"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"time"
)

type Worker struct {
	Store      *database.Store
	URL, Model string
	Client     *http.Client
}

func (w *Worker) Run(ctx context.Context) {
	timer := time.NewTicker(5 * time.Second)
	defer timer.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		_ = w.Store.ResetRunningClaims(ctx)
		_ = w.Step(ctx)
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
	}
}
func (w *Worker) Step(ctx context.Context) error {
	c, err := w.Store.ClaimRunningClip(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	ctx, span := otel.Tracer("career/running").Start(ctx, "running.transcribe")
	defer span.End()
	meter := otel.Meter("career/running")
	duration, _ := meter.Float64Histogram("running_transcribe_duration_seconds")
	failures, _ := meter.Int64Counter("running_transcribe_failures_total")
	start := time.Now()
	defer func() { duration.Record(ctx, time.Since(start).Seconds()) }()
	text, unavailable, err := w.transcribe(ctx, c)
	if err == nil {
		err = w.Store.FinishRunningClip(ctx, c, text, w.Model)
	}
	if err != nil {
		span.RecordError(err)
		failures.Add(ctx, 1)
		return w.Store.RetryRunningClip(ctx, c, "Local transcription failed; retry pending or use Retranscribe", unavailable)
	}
	return nil
}
func (w *Worker) transcribe(ctx context.Context, c database.RunningClip) (string, bool, error) {
	var b bytes.Buffer
	m := multipart.NewWriter(&b)
	ext := ".webm"
	switch strings.Split(c.MIME, ";")[0] {
	case "audio/mp4", "audio/x-m4a":
		ext = ".m4a"
	case "audio/wav", "audio/x-wav":
		ext = ".wav"
	case "audio/mpeg":
		ext = ".mp3"
	case "audio/ogg":
		ext = ".ogg"
	case "audio/flac":
		ext = ".flac"
	}
	f, _ := m.CreateFormFile("file", path.Base(c.ID)+ext)
	_, _ = f.Write(c.Audio)
	_ = m.WriteField("response_format", "json")
	_ = m.Close()
	ctx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", strings.TrimRight(w.URL, "/")+"/inference", &b)
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Content-Type", m.FormDataContentType())
	client := w.Client
	if client == nil {
		client = &http.Client{Timeout: 4 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	response, err := client.Do(req)
	if err != nil {
		return "", true, err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return "", response.StatusCode >= 500, fmt.Errorf("whisper status %d", response.StatusCode)
	}
	var out struct {
		Text string `json:"text"`
	}
	err = json.NewDecoder(io.LimitReader(response.Body, 250000)).Decode(&out)
	return strings.TrimSpace(out.Text), false, err
}
