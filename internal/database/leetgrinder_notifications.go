package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

// LeetgrinderNotifyRetryDelay is how long a failed (or abandoned) send waits
// before a later tick may retry it.
const LeetgrinderNotifyRetryDelay = 5 * time.Minute

const leetgrinderNotificationColumns = "kind,local_date,sent_at,status,detail,attempts"

func scanLeetgrinderNotification(row interface{ Scan(...any) error }) (leetgrinder.NotificationLogEntry, error) {
	var e leetgrinder.NotificationLogEntry
	err := row.Scan(&e.Kind, &e.LocalDate, &e.SentAt, &e.Status, &e.Detail, &e.Attempts)
	e.LocalDate = leetgrinder.Date(e.LocalDate, time.UTC)
	return e, err
}

// LeetgrinderNotification returns the log row for kind on the local date, or
// ErrNotFound.
func (s *Store) LeetgrinderNotification(ctx context.Context, kind string, date time.Time) (leetgrinder.NotificationLogEntry, error) {
	e, err := scanLeetgrinderNotification(s.DB.QueryRowContext(ctx, "SELECT "+leetgrinderNotificationColumns+" FROM leetgrinder_notification_log WHERE kind=$1 AND local_date=$2", kind, date.Format(time.DateOnly)))
	if errors.Is(err, sql.ErrNoRows) {
		return e, ErrNotFound
	}
	return e, err
}

// ClaimLeetgrinderNotification reserves a send of kind for the local date. It
// succeeds for a new row, or for a failed or abandoned send that is older than
// the retry delay and still under the attempt cap. The returned attempt number
// counts this send. A false result means nothing should be sent.
func (s *Store) ClaimLeetgrinderNotification(ctx context.Context, kind string, date, now time.Time) (int, bool, error) {
	var attempt int
	err := s.DB.QueryRowContext(ctx, `INSERT INTO leetgrinder_notification_log(kind,local_date,sent_at,status,detail,attempts) VALUES($1,$2,$3,'sending','',1)
ON CONFLICT (kind,local_date) DO UPDATE SET status='sending', sent_at=EXCLUDED.sent_at, detail='', attempts=leetgrinder_notification_log.attempts+1
WHERE leetgrinder_notification_log.status IN ('failed','sending') AND leetgrinder_notification_log.attempts < $4 AND leetgrinder_notification_log.sent_at <= $5
RETURNING attempts`, kind, date.Format(time.DateOnly), now, leetgrinder.NotifyMaxAttempts, now.Add(-LeetgrinderNotifyRetryDelay)).Scan(&attempt)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	return attempt, err == nil, err
}

// FinishLeetgrinderNotification records the outcome of a claimed send.
func (s *Store) FinishLeetgrinderNotification(ctx context.Context, kind string, date time.Time, status, detail string, now time.Time) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE leetgrinder_notification_log SET status=$3, detail=$4, sent_at=$5 WHERE kind=$1 AND local_date=$2 AND status='sending'", kind, date.Format(time.DateOnly), status, detail, now)
	return err
}

// SkipLeetgrinderNotification records that kind had nothing to report on the
// local date, so it is not evaluated again that day. A pending retry of a
// failed or abandoned send is skipped too, since its condition no longer
// holds; a send still in progress is left alone.
func (s *Store) SkipLeetgrinderNotification(ctx context.Context, kind string, date time.Time, detail string, now time.Time) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO leetgrinder_notification_log(kind,local_date,sent_at,status,detail,attempts) VALUES($1,$2,$3,'skipped',$4,0)
ON CONFLICT (kind,local_date) DO UPDATE SET status='skipped', detail=EXCLUDED.detail, sent_at=EXCLUDED.sent_at
WHERE leetgrinder_notification_log.status IN ('failed','sending') AND leetgrinder_notification_log.sent_at <= $5`, kind, date.Format(time.DateOnly), now, detail, now.Add(-LeetgrinderNotifyRetryDelay))
	return err
}

// RecordLeetgrinderTestNotification keeps the latest test send for the date.
func (s *Store) RecordLeetgrinderTestNotification(ctx context.Context, date time.Time, status, detail string, now time.Time) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO leetgrinder_notification_log(kind,local_date,sent_at,status,detail,attempts) VALUES($1,$2,$3,$4,$5,1)
ON CONFLICT (kind,local_date) DO UPDATE SET sent_at=EXCLUDED.sent_at, status=EXCLUDED.status, detail=EXCLUDED.detail, attempts=leetgrinder_notification_log.attempts+1`,
		leetgrinder.NotifyTestKind, date.Format(time.DateOnly), now, status, detail)
	return err
}

// RecentLeetgrinderNotifications returns the newest log rows first.
func (s *Store) RecentLeetgrinderNotifications(ctx context.Context, limit int) ([]leetgrinder.NotificationLogEntry, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT "+leetgrinderNotificationColumns+" FROM leetgrinder_notification_log ORDER BY sent_at DESC, kind LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []leetgrinder.NotificationLogEntry
	for rows.Next() {
		e, err := scanLeetgrinderNotification(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
