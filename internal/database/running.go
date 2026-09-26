package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"regexp"
	"strings"
	"time"
)

type RunningClip struct {
	NoteID     string    `json:"noteId"`
	ID         string    `json:"clipId"`
	Position   int       `json:"position"`
	RecordedAt time.Time `json:"recordedAt"`
	DurationMs int       `json:"durationMs"`
	MIME       string    `json:"mimeType"`
	Status     string    `json:"status"`
	Transcript string    `json:"transcript"`
	Error      string    `json:"error"`
	Model      string    `json:"model"`
	Attempts   int       `json:"attempts"`
	Audio      []byte    `json:"-"`
}

// ThoughtPlaceholder names a thought until its first transcript arrives.
const ThoughtPlaceholder = "New audio thought"

// autoTitle matches generated titles, including the timestamped ones used before
// audio thoughts, so transcripts only ever replace a title the user did not write.
var autoTitle = regexp.MustCompile(`^(New audio thought|Run \d{4}-\d{2}-\d{2} \d{2}:\d{2}|Audio thought \d{4}-\d{2}-\d{2})$`)

// ThoughtTitle is the first six words of a transcript, ending in "..." when cut short.
func ThoughtTitle(body string) string {
	words := strings.Fields(body)
	if len(words) == 0 {
		return ""
	}
	title := strings.Join(words[:min(6, len(words))], " ")
	cut := len(words) > 6
	if runes := []rune(title); len(runes) > 60 {
		title, cut = string(runes[:60]), true
	}
	if !cut {
		return title
	}
	return strings.TrimRight(title, " .,;:!?-") + "..."
}

func (s *Store) AddRunningClip(ctx context.Context, c RunningClip) (RunningClip, error) {
	if !uuidRE.MatchString(c.ID) || (c.NoteID != "" && !ValidID("run", c.NoteID)) || c.RecordedAt.IsZero() || c.DurationMs < 0 || c.DurationMs > 1800000 || len(c.Audio) == 0 || len(c.Audio) > 15<<20 || !strings.HasPrefix(c.MIME, "audio/") || len(c.MIME) > 100 {
		return c, ErrInvalid
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return c, err
	}
	defer tx.Rollback()
	// Serialize uploads to make idempotency, grouping and segment positions atomic.
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(724193602)"); err != nil {
		return c, err
	}
	var existing RunningClip
	err = tx.QueryRowContext(ctx, "SELECT note_id,id,transcript_status FROM running_note_clips WHERE id=$1", c.ID).Scan(&existing.NoteID, &existing.ID, &existing.Status)
	if err == nil {
		return existing, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return c, err
	}
	if c.NoteID == "" {
		// Group by recording time, not upload time, so an offline backlog remains one run.
		err = tx.QueryRowContext(ctx, `SELECT note_id FROM running_note_clips WHERE recorded_at BETWEEN $1::timestamptz-interval '90 minutes' AND $1::timestamptz+interval '90 minutes' ORDER BY abs(extract(epoch FROM recorded_at-$1::timestamptz)) LIMIT 1`, c.RecordedAt).Scan(&c.NoteID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return c, err
		}
		if c.NoteID == "" {
			c.NoteID = uuid.NewString()
			_, err = saveTx(ctx, tx, "run", Entity{"id": c.NoteID, "title": ThoughtPlaceholder, "runDate": c.RecordedAt.Format("2006-01-02"), "startedAt": c.RecordedAt.Format(time.RFC3339Nano), "tags": []any{}, "body": ""}, nil, false)
			if err != nil {
				return c, err
			}
		}
	}
	var note string
	if err = tx.QueryRowContext(ctx, "SELECT id FROM running_notes WHERE id=$1 FOR UPDATE", c.NoteID).Scan(&note); errors.Is(err, sql.ErrNoRows) {
		return c, ErrNotFound
	} else if err != nil {
		return c, err
	}
	c.Status = "pending"
	err = tx.QueryRowContext(ctx, `INSERT INTO running_note_clips(note_id,id,position,recorded_at,duration_ms,mime_type,audio,transcript_status) SELECT $1,$2,COALESCE(max(position)+1,0),$3,$4,$5,$6,'pending' FROM running_note_clips WHERE note_id=$1 RETURNING position`, c.NoteID, c.ID, c.RecordedAt, c.DurationMs, c.MIME, c.Audio).Scan(&c.Position)
	if err != nil {
		return c, err
	}
	if err = bump(ctx, tx, "run"); err != nil {
		return c, err
	}
	return c, tx.Commit()
}
func (s *Store) RunningClips(ctx context.Context, note string) ([]RunningClip, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT note_id,id,position,recorded_at,duration_ms,mime_type,transcript_status,COALESCE(transcript,''),COALESCE(transcript_error,''),COALESCE(model,''),attempts FROM running_note_clips WHERE note_id=$1 ORDER BY position`, note)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RunningClip{}
	for rows.Next() {
		var c RunningClip
		if err = rows.Scan(&c.NoteID, &c.ID, &c.Position, &c.RecordedAt, &c.DurationMs, &c.MIME, &c.Status, &c.Transcript, &c.Error, &c.Model, &c.Attempts); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (s *Store) RunningAudio(ctx context.Context, note, id string) (RunningClip, error) {
	var c RunningClip
	err := s.DB.QueryRowContext(ctx, "SELECT audio,mime_type FROM running_note_clips WHERE note_id=$1 AND id=$2", note, id).Scan(&c.Audio, &c.MIME)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return c, err
}
func (s *Store) Retranscribe(ctx context.Context, note, id string) error {
	r, err := s.DB.ExecContext(ctx, `UPDATE running_note_clips SET transcript_status='pending',attempts=0,claimed_at=NULL,next_attempt_at=now(),transcript_error=NULL WHERE note_id=$1 AND id=$2 AND transcript_status<>'transcribing'`, note, id)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return ErrConflict
	}
	return nil
}
func (s *Store) ResetRunningClaims(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE running_note_clips SET transcript_status=CASE WHEN attempts>=3 THEN 'failed' ELSE 'pending' END,claimed_at=NULL,next_attempt_at=now(),transcript_error='Transcription interrupted; retry available' WHERE transcript_status='transcribing' AND claimed_at<now()-interval '10 minutes'`)
	return err
}
func (s *Store) ClaimRunningClip(ctx context.Context) (RunningClip, error) {
	var c RunningClip
	err := s.DB.QueryRowContext(ctx, `UPDATE running_note_clips SET transcript_status='transcribing',attempts=attempts+1,claimed_at=now() WHERE id=(SELECT id FROM running_note_clips WHERE transcript_status='pending' AND next_attempt_at<=now() AND attempts<3 ORDER BY next_attempt_at,recorded_at FOR UPDATE SKIP LOCKED LIMIT 1) RETURNING note_id,id,audio,mime_type,recorded_at,attempts`).Scan(&c.NoteID, &c.ID, &c.Audio, &c.MIME, &c.RecordedAt, &c.Attempts)
	return c, err
}
func (s *Store) FinishRunningClip(ctx context.Context, c RunningClip, transcript, model string) error {
	if size(transcript) > 50000 {
		return fmt.Errorf("transcript too long")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Lock the note first, matching saves and deletes. Append against the latest body;
	// retries/retranscriptions never overwrite user text or append a second copy.
	var body, title string
	if err = tx.QueryRowContext(ctx, "SELECT body,title FROM running_notes WHERE id=$1 FOR UPDATE", c.NoteID).Scan(&body, &title); err != nil {
		return err
	}
	var merged bool
	if err = tx.QueryRowContext(ctx, "SELECT body_merged FROM running_note_clips WHERE id=$1 AND transcript_status='transcribing' AND attempts=$2 FOR UPDATE", c.ID, c.Attempts).Scan(&merged); err != nil {
		return err
	}
	if !merged {
		if body = strings.TrimRight(body, " \t\r\n"); body != "" {
			body += "\n\n"
		}
		body += strings.TrimSpace(transcript)
		// Name the thought after its opening words until the user picks a title.
		if autoTitle.MatchString(title) {
			if derived := ThoughtTitle(body); derived != "" {
				title = derived
			}
		}
		if size(body) > 100000 {
			return fmt.Errorf("run body full; clip transcript cannot be appended")
		}
		if _, err = tx.ExecContext(ctx, "UPDATE running_notes SET body=$2,title=$3,revision=$4,updated_at=now(),position=nextval('entity_position') WHERE id=$1", c.NoteID, body, title, uuid.NewString()); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE running_note_clips SET transcript=$2,model=$3,transcript_status='done',transcript_error=NULL,body_merged=true,claimed_at=NULL WHERE id=$1`, c.ID, transcript, model); err != nil {
		return err
	}
	if err = bump(ctx, tx, "run"); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) RetryRunningClip(ctx context.Context, c RunningClip, message string, unavailable bool) error {
	status := "pending"
	attempts := c.Attempts
	if unavailable {
		attempts--
	} else if attempts >= 3 {
		status = "failed"
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE running_note_clips SET transcript_status=$2,transcript_error=$3,attempts=$4,claimed_at=NULL,next_attempt_at=now()+$5*interval '1 second' WHERE id=$1 AND transcript_status='transcribing' AND attempts=$6`, c.ID, status, message, attempts, 30*(c.Attempts+1), c.Attempts)
	return err
}
