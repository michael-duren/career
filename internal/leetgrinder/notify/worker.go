package notify

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

// Worker checks every minute whether a notification is due and sends it.
// The notification log deduplicates sends across ticks and restarts.
type Worker struct {
	Store *database.Store
	// SecretKey decrypts the stored ntfy token (config.LeetgrinderSecretKey).
	SecretKey []byte
	// Origin is the app's public origin for Click deep links.
	Origin string
	// HTTP and Now are overridden in tests.
	HTTP *http.Client
	Now  func() time.Time
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
			log.Printf("leetgrinder notifications: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Step sends every enabled notification whose time has passed today and
// that has not been sent, skipped, or retried to the cap yet.
func (w *Worker) Step(ctx context.Context) error {
	settings, err := w.Store.LeetgrinderSettings(ctx)
	if err != nil {
		return err
	}
	if settings.NtfyTopic == "" {
		return nil
	}
	now := w.now()
	loc := settings.Location()
	date := leetgrinder.Date(now, loc)
	var today *leetgrinder.Today
	var errs []error
	for _, kind := range leetgrinder.NotificationKinds {
		pref := settings.NotificationPref(kind.Key)
		at, ok := pref.TriggerAt(date, loc)
		if !pref.Enabled || !ok || now.Before(at) {
			continue
		}
		entry, err := w.Store.LeetgrinderNotification(ctx, kind.Key, date)
		switch {
		case errors.Is(err, database.ErrNotFound):
		case err != nil:
			errs = append(errs, err)
			continue
		case !retryable(entry, now):
			continue
		}
		if today == nil {
			t, err := w.Store.LeetgrinderToday(ctx, now)
			if err != nil {
				return errors.Join(append(errs, err)...)
			}
			today = &t
		}
		msg, due := Compose(kind.Key, pref, *today, w.Origin)
		if !due {
			if err = w.Store.SkipLeetgrinderNotification(ctx, kind.Key, date, "Nothing to report", now); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		if _, claimed, err := w.Store.ClaimLeetgrinderNotification(ctx, kind.Key, date, now); err != nil || !claimed {
			if err != nil {
				errs = append(errs, err)
			}
			continue
		}
		status, detail := leetgrinder.NotifySent, Truncate(msg.Body, 240)
		if err = Send(ctx, w.HTTP, settings, w.SecretKey, msg); err != nil {
			status, detail = leetgrinder.NotifyFailed, err.Error()
		}
		// The send already happened, so record it even if ctx was cancelled.
		finish, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		err = w.Store.FinishLeetgrinderNotification(finish, kind.Key, date, status, detail, w.now())
		cancel()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// retryable reports whether an existing row may be claimed again.
func retryable(e leetgrinder.NotificationLogEntry, now time.Time) bool {
	return (e.Status == leetgrinder.NotifyFailed || e.Status == leetgrinder.NotifySending) &&
		e.Attempts < leetgrinder.NotifyMaxAttempts && !e.SentAt.After(now.Add(-database.LeetgrinderNotifyRetryDelay))
}
