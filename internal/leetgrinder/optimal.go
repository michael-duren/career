package leetgrinder

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrOptimalRequired = errors.New("optimal time and space complexity are both required")
	ErrOptimalNote     = errors.New("the optimal note must be 300 characters or fewer, without control characters")
)

// NormalizeOptimal trims and validates a manually entered optimum: time and
// space pass NormalizeComplexity and are both present, and the note is
// trimmed, at most MaxOptimalNote characters, with no control characters.
func NormalizeOptimal(optimalTime, optimalSpace, note string) (string, string, string, error) {
	t, err := NormalizeComplexity(optimalTime)
	if err != nil {
		return "", "", "", err
	}
	sp, err := NormalizeComplexity(optimalSpace)
	if err != nil {
		return "", "", "", err
	}
	if t == "" || sp == "" {
		return "", "", "", ErrOptimalRequired
	}
	note = strings.TrimSpace(note)
	if !utf8.ValidString(note) || utf8.RuneCountInString(note) > MaxOptimalNote || strings.ContainsFunc(note, func(r rune) bool { return r < ' ' || r == 0x7f }) {
		return "", "", "", ErrOptimalNote
	}
	return t, sp, note, nil
}

// OptimalForm is the "Edit optimal complexity" form. Revision is the
// Problem.OptimalRevision the form was rendered from.
type OptimalForm struct {
	Time, Space ComplexityInput
	Note        string
	Revision    string
	Error       string
	// Conflict is set when the values changed since the form was opened; Revision
	// is then the current one, so saving again is a deliberate overwrite.
	Conflict bool
}

// NewOptimalForm prefills the form with the problem's current optimum.
func NewOptimalForm(p Problem) OptimalForm {
	return OptimalForm{Time: NewComplexityInput(p.OptimalTime), Space: NewComplexityInput(p.OptimalSpace), Note: p.OptimalNote, Revision: p.OptimalRevision()}
}

// OptimalEditURL is the page that edits slug's optimal complexity; it also
// receives the form post.
func OptimalEditURL(slug string) string { return ProblemURL(slug) + "/optimal" }

// OptimalReestimateURL receives the "Re-estimate" post.
func OptimalReestimateURL(slug string) string { return OptimalEditURL(slug) + "/reestimate" }
