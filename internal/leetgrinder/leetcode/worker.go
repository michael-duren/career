package leetcode

import (
	"context"
	"log"
	"time"

	"github.com/michael-duren/career-strategy/internal/database"
)

// Worker fetches metadata for at most one problem a minute. A failed fetch
// never blocks saving attempts; pages show the slug until metadata arrives.
type Worker struct {
	Store  *database.Store
	Client *Client
	// Now is overridden in tests.
	Now func() time.Time
}

func (w *Worker) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}

// Run ticks every minute until ctx ends.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if err := w.Step(ctx); err != nil && ctx.Err() == nil {
			log.Printf("leetgrinder metadata: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Step fetches the next problem that needs metadata, if any.
func (w *Worker) Step(ctx context.Context) error {
	slug, ok, err := w.Store.NextLeetgrinderFetch(ctx, w.now())
	if err != nil || !ok {
		return err
	}
	meta, fetchErr := w.Client.Fetch(ctx, slug)
	if fetchErr != nil && ctx.Err() != nil {
		return nil
	}
	if fetchErr != nil {
		log.Printf("leetgrinder metadata: %s: %v", slug, fetchErr)
	}
	finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return w.Store.FinishLeetgrinderFetch(finish, slug, meta, fetchErr, w.now())
}
