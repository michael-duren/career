package leetgrinder

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Analysis statuses. An attempt whose inputs changed since its analysis (see
// Analysis.Current) is queued again whatever its stored status.
const (
	AnalysisPending = "pending"
	AnalysisDone    = "done"
	AnalysisFailed  = "failed"
)

const (
	// AnalysisMaxTries is the first request plus three retries.
	AnalysisMaxTries = 4
	// MaxAnalysisExplanation caps the stored explanation, in characters.
	MaxAnalysisExplanation = 2000
	// MaxAnalysisError caps the stored error summary, in characters.
	MaxAnalysisError = 300
)

// Analysis is the LLM's assessment of an attempt's captured code. It is
// derived data: it is never exported, and it is re-created when the attempt's
// code, language or stated complexities change.
type Analysis struct {
	AttemptID   string
	Status      string
	Tries       int
	ActualTime  string
	ActualSpace string
	// TimeMatches and SpaceMatches are nil when that complexity was not
	// stated. Optimal is nil until the analysis is done.
	TimeMatches, SpaceMatches, Optimal *bool
	Explanation                        string
	Model                              string
	// Error is a redacted summary of the latest failure.
	Error     string
	UpdatedAt time.Time
	// Current reports whether the analysis was made for the attempt's
	// current code, language and stated complexities.
	Current bool
}

// AnalysisResult is a validated model response.
type AnalysisResult struct {
	ActualTime, ActualSpace   string
	TimeMatches, SpaceMatches *bool
	Optimal                   bool
	Explanation               string
}

// Analysable reports whether an attempt can be analysed: it has code and at
// least one stated complexity.
func (a Attempt) Analysable() bool {
	return a.Code != "" && (a.TimeComplexity != "" || a.SpaceComplexity != "")
}

// Done reports whether a is a finished analysis of the attempt's current inputs.
func (a Analysis) Done() bool { return a.Current && a.Status == AnalysisDone }

// StatedWrong reports whether a finished, current analysis judged a stated
// complexity wrong.
func (a Analysis) StatedWrong() bool {
	return a.Done() && (isFalse(a.TimeMatches) || isFalse(a.SpaceMatches))
}

func isFalse(b *bool) bool { return b != nil && !*b }

// AnalysisFor returns the analysis stored for an attempt, if any.
func (s State) AnalysisFor(attemptID string) (Analysis, bool) {
	a, ok := s.Analyses[attemptID]
	return a, ok
}

// StatedWrong reports whether the attempt's current analysis judged a stated
// complexity wrong.
func (s State) StatedWrong(attemptID string) bool {
	a, ok := s.Analyses[attemptID]
	return ok && a.StatedWrong()
}

// CleanAnalysisText keeps printable text and newlines, turns other control
// characters into spaces, trims the result, and caps it at max characters.
func CleanAnalysisText(s string, max int) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "�")
	}
	s = strings.Map(func(r rune) rune {
		switch {
		case r == '\n':
			return r
		case unicode.IsControl(r) || r == ' ' || r == ' ':
			return ' '
		}
		return r
	}, s)
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:max-1])) + "…"
}

// AnalysisQueue counts analysable attempts by state. Queued includes
// attempts waiting for a retry.
type AnalysisQueue struct {
	Queued, Failed, Done int
}

// AnalysisPause holds analysis requests back after an error that would hit
// every attempt. Reason is a redacted error summary.
type AnalysisPause struct {
	Until  time.Time
	Reason string
}

// AnalysisAvailability is what the history page needs to explain queued
// attempts.
type AnalysisAvailability struct {
	KeyConfigured, Enabled bool
	// Paused and LimitReached explain why queued attempts are waiting.
	Paused, LimitReached bool
	// Unknown reports that the settings could not be loaded.
	Unknown bool
}

// On reports whether the worker sends requests.
func (a AnalysisAvailability) On() bool { return a.KeyConfigured && a.Enabled && !a.Unknown }

// Waiting reports whether requests are on hold for now.
func (a AnalysisAvailability) Waiting() bool { return a.Paused || a.LimitReached }

// AnalysisPanel is the complexity analysis section of the settings page.
// It never carries the API key, only whether one is configured.
type AnalysisPanel struct {
	KeyConfigured bool
	Enabled       bool
	Model         string
	Used, Limit   int
	UsageError    bool
	Queue         AnalysisQueue
	QueueError    bool
	// Pause is set while requests are held back; PausedNow reports whether
	// it is still in effect.
	Pause     AnalysisPause
	PausedNow bool
	Location  *time.Location
	Revision  string
}

// Active reports whether the worker sends requests.
func (p AnalysisPanel) Active() bool { return p.KeyConfigured && p.Enabled }

// LimitReached reports whether today's requests used the daily limit.
func (p AnalysisPanel) LimitReached() bool { return !p.UsageError && p.Used >= p.Limit }

// AnalysisReanalyseURL is the form action that queues an attempt again.
func AnalysisReanalyseURL(slug, attemptID string) string {
	return ProblemURL(slug) + "/attempts/" + attemptID + "/analysis"
}

// AttemptAnchor is the history page fragment for an attempt.
func AttemptAnchor(attemptID string) string { return "attempt-" + attemptID }
