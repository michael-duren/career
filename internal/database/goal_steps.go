package database

import (
	"context"
	"fmt"
	"slices"
)

// StepMove relocates one goal step (a "mini goal") within its goal or to
// another goal. Both goals' revisions must match so a stale graph never
// overwrites newer edits.
type StepMove struct {
	StepID       string `json:"stepId"`
	From         string `json:"from"`
	FromRevision string `json:"fromRevision"`
	To           string `json:"to"`
	ToRevision   string `json:"toRevision"`
	// Index is the step's position in the destination goal after the move.
	Index int `json:"index"`
}

// MoveGoalStep applies a StepMove in one transaction, so a step moved between
// goals is never lost or duplicated. It returns the saved goals, the source
// first; a move within one goal returns a single result.
func (s *Store) MoveGoalStep(ctx context.Context, m StepMove) ([]Result, error) {
	if !ValidID("goal", m.From) || !ValidID("goal", m.To) || !uuidRE.MatchString(m.StepID) || m.Index < 0 {
		return nil, fmt.Errorf("%w: goal IDs, step ID and a non-negative index required", ErrInvalid)
	}
	// One goal has one revision; differing values are a malformed request.
	if m.From == m.To && m.FromRevision != m.ToRevision {
		return nil, fmt.Errorf("%w: a move within one goal needs matching revisions", ErrInvalid)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err = lockGoals(ctx, tx); err != nil {
		return nil, err
	}
	from, err := readOne(ctx, tx, "goal", m.From)
	if err != nil {
		return nil, err
	}
	if from.Revision != m.FromRevision {
		return nil, ErrConflict
	}
	steps, _ := from.Entry["steps"].([]any)
	at := slices.IndexFunc(steps, func(v any) bool { o, _ := v.(map[string]any); return o["id"] == m.StepID })
	if at < 0 {
		return nil, fmt.Errorf("%w: step not found in source goal", ErrNotFound)
	}
	step := steps[at]
	from.Entry["steps"] = slices.Delete(slices.Clone(steps), at, at+1)
	to := from
	if m.From != m.To {
		if to, err = readOne(ctx, tx, "goal", m.To); err != nil {
			return nil, err
		}
		if to.Revision != m.ToRevision {
			return nil, ErrConflict
		}
	}
	dest, _ := to.Entry["steps"].([]any)
	if slices.ContainsFunc(dest, func(v any) bool { o, _ := v.(map[string]any); return o["id"] == m.StepID }) {
		return nil, fmt.Errorf("%w: destination goal already has this step", ErrInvalid)
	}
	to.Entry["steps"] = slices.Insert(slices.Clone(dest), min(m.Index, len(dest)), step)
	var saved []Result
	goals := []Result{from}
	if m.From != m.To {
		goals = append(goals, to)
	}
	for _, g := range goals {
		e, err := PrepareSave("goal", g.Entry)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
		}
		r, err := saveTx(ctx, tx, "goal", e, &g.Revision, false)
		if err != nil {
			return nil, dbError(err)
		}
		saved = append(saved, r)
	}
	if err = bump(ctx, tx, "goal"); err != nil {
		return nil, err
	}
	return saved, tx.Commit()
}
