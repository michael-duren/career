package analysis

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

// maxPerStep bounds the requests one tick sends, one after another.
const maxPerStep = 10

// Worker analyses queued attempts in the background, one request at a time.
type Worker struct {
	Store *database.Store
	// Key is ANTHROPIC_API_KEY. Without it the worker never touches the network.
	Key        string
	Model      string
	DailyLimit int
	// BaseURL, HTTP and Now are overridden in tests.
	BaseURL string
	HTTP    *http.Client
	Now     func() time.Time

	once   sync.Once
	client *Client
}

func (w *Worker) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}

func (w *Worker) api() *Client {
	w.once.Do(func() { w.client = NewClient(w.Key, w.Model, w.BaseURL, w.HTTP) })
	return w.client
}

// Run ticks every minute until ctx ends. It returns at once without a key.
func (w *Worker) Run(ctx context.Context) {
	if w.Key == "" {
		log.Print("leetgrinder analysis: ANTHROPIC_API_KEY is not set, so complexity analysis is off")
		return
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if err := w.Step(ctx); err != nil && ctx.Err() == nil {
			log.Printf("leetgrinder analysis: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Step analyses queued attempts, newest first, until the queue is empty, the
// daily limit is reached, or maxPerStep requests were sent. It does nothing
// without a key, while analysis is turned off in settings, or while another
// worker holds the analysis lock.
func (w *Worker) Step(ctx context.Context) error {
	if w.Key == "" {
		return nil
	}
	settings, err := w.Store.LeetgrinderSettings(ctx)
	if err != nil || !settings.AnalysisEnabled {
		return err
	}
	loc := settings.Location()
	_, err = w.Store.WithLeetgrinderAnalysisLock(ctx, func(ctx context.Context) error {
		for range maxPerStep {
			if ctx.Err() != nil {
				return nil
			}
			job, ok, err := w.Store.NextLeetgrinderAnalysis(ctx, w.now())
			if err != nil || !ok {
				return err
			}
			reserved, err := w.Store.ReserveLeetgrinderAnalysisRequest(ctx, leetgrinder.Date(w.now(), loc), w.DailyLimit)
			if err != nil || !reserved {
				return err
			}
			if err = w.analyse(ctx, job); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

// analyse sends one request and records its outcome. A request cut short by
// shutdown is not recorded, so it runs again after the restart.
func (w *Worker) analyse(ctx context.Context, job database.LeetgrinderAnalysisJob) error {
	a := job.Attempt
	problem, known := leetgrinder.FindProblem(a.ProblemSlug)
	if !known {
		problem = leetgrinder.Problem{Slug: a.ProblemSlug}
	}
	client := w.api()
	result, err := client.Analyze(ctx, Input{Problem: problem, Known: known, Language: a.CodeLanguage, Code: a.Code, StatedTime: a.TimeComplexity, StatedSpace: a.SpaceComplexity})
	if err != nil && ctx.Err() != nil {
		return nil
	}
	tries := job.Tries + 1
	status, detail := leetgrinder.AnalysisDone, ""
	if err != nil {
		message, retry := "the analysis failed", true
		var e *Error
		if errors.As(err, &e) {
			message, retry = e.Message, e.Retry
		}
		detail = Redact(message, w.Key)
		result = leetgrinder.AnalysisResult{}
		status = leetgrinder.AnalysisFailed
		if retry && tries < leetgrinder.AnalysisMaxTries {
			status = leetgrinder.AnalysisPending
		}
		log.Printf("leetgrinder analysis: attempt %s, try %d: %s", a.ID, tries, detail)
	}
	// The request already happened, so record it even if ctx was cancelled.
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return w.Store.FinishLeetgrinderAnalysis(finish, job, status, tries, result, client.Model(), detail, w.now())
}
