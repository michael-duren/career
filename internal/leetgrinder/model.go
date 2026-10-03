package leetgrinder

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Problem is one row of the problem catalog (leetgrinder_problems). Any
// LeetCode problem can be tracked; a problem whose metadata is not known yet
// has only Slug set.
type Problem struct {
	Slug string
	// Number is LeetCode's frontend id, or 0 until known.
	Number     int
	Title      string
	Difficulty string
	// Topics are LeetCode topic tag slugs, such as "hash-table".
	Topics []string
	// TopicNames maps a topic slug to the display name LeetCode sent for it.
	// A slug without a name is labelled by TopicLabel; use Problem.TopicLabel.
	TopicNames map[string]string
	// OptimalTime and OptimalSpace are the best-known bounds under the
	// problem's constraints, in the notation NormalizeComplexity produces. Space is auxiliary
	// space and excludes the returned output.
	OptimalTime  string
	OptimalSpace string
	// OptimalNote defines variables other than n and names alternatives.
	OptimalNote string
	// OptimalSource is "curated" for the reviewed table, "model" for
	// Claude's estimate, or "" when no optimum is known.
	OptimalSource string
	// MetadataSource is "seed", "extension", "leetcode", "mcp", or "" for a bare row.
	MetadataSource string
	// ImportMetadata keeps source-specific problem snapshots. Each snapshot is
	// stored once per problem, under the ID of the todo item that first
	// archived it; a later item with identical content adds nothing, and
	// snapshots stay when their item is deleted. Migration 024 still says the
	// keys are item IDs and that entries remain on removal; its SQL is applied
	// and checksummed, so this comment is the current description.
	ImportMetadata map[string]any
	// NotFound reports that LeetCode answered that no such problem exists.
	NotFound bool
	// FetchAttempts counts server-side metadata fetches; see MaxFetchAttempts.
	FetchAttempts int
	// InCatalog reports that the problem has a catalog row. Rows are added
	// when the first attempt is saved; only they are fetched.
	InCatalog bool
}

func (p Problem) URL() string { return "https://leetcode.com/problems/" + p.Slug + "/" }

// Known reports whether the problem's metadata has been filled in.
func (p Problem) Known() bool { return p.Title != "" }

// DisplayTitle is the title, or the slug until the title is known.
func (p Problem) DisplayTitle() string {
	if p.Title != "" {
		return p.Title
	}
	return p.Slug
}

// Fetching reports that metadata is missing and the server will still try
// to fetch it from LeetCode.
func (p Problem) Fetching() bool {
	return p.InCatalog && !p.Known() && !p.NotFound && p.FetchAttempts < MaxFetchAttempts
}

// HasOptimal reports whether an optimal time and space are known.
func (p Problem) HasOptimal() bool { return p.OptimalTime != "" && p.OptimalSpace != "" }

// OptimalEstimated reports that the optimum is Claude's estimate rather than
// the curated table.
func (p Problem) OptimalEstimated() bool { return p.OptimalSource == "model" }

// OptimalManual reports that the optimum was entered by the learner.
func (p Problem) OptimalManual() bool { return p.OptimalSource == "manual" }

// OptimalSourceLabel names where the optimum came from: "Your value" for a
// manual entry, "Claude's estimate" for a model estimate, "Curated" otherwise.
func (p Problem) OptimalSourceLabel() string {
	switch p.OptimalSource {
	case "manual":
		return "Your value"
	case "model":
		return "Claude's estimate"
	}
	return "Curated"
}

// OptimalRevision identifies the optimal values and their source, so a form
// opened before they changed (an edit, a re-estimate, a first estimate) is
// refused instead of silently replacing the newer values.
func (p Problem) OptimalRevision() string {
	sum := sha256.Sum256([]byte(p.OptimalSource + "\x00" + p.OptimalTime + "\x00" + p.OptimalSpace + "\x00" + p.OptimalNote))
	return hex.EncodeToString(sum[:])
}

// Attempt IDs make retries safe; revisions prevent silent overwrites during corrections.
type Attempt struct {
	ID          string    `json:"id"`
	ProblemSlug string    `json:"problemSlug"`
	Outcome     string    `json:"outcome"`
	Minutes     int       `json:"minutes"`
	Assisted    bool      `json:"assisted"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"createdAt"`
	Revision    string    `json:"revision"`
	// Source is "web" or "extension". Corrections keep Source and IsReview.
	Source   string `json:"source"`
	IsReview bool   `json:"isReview"`
	// Stated complexities, "" when not stated (see NormalizeComplexity).
	TimeComplexity  string `json:"timeComplexity"`
	SpaceComplexity string `json:"spaceComplexity"`
	// Code is the judged submission captured by the extension, with its
	// LeetCode language slug; both are "" for web-logged attempts.
	Code         string `json:"code"`
	CodeLanguage string `json:"codeLanguage"`
	// WantsReview and Approach are the learner's own assessment. Approach is
	// ApproachOptimal, ApproachSuboptimal, or "" when not stated. Either one,
	// like a struggle, flags the problem for review (see SelfFlagged).
	WantsReview bool   `json:"wantsReview"`
	Approach    string `json:"approach"`
	// MarkedAt is when the attempt first flagged the problem, or flagged it
	// again after a correction cleared it: the attempt's time, or the
	// correction that raised the flag.
	MarkedAt time.Time `json:"markedAt"`
}

// Struggle labels an attempt that was not an unassisted solve, the same
// struggles the stats page counts: "Struggled", "Unfinished" or "Solved with
// help". It is "" for an unassisted solve.
func Struggle(a Attempt) string {
	switch {
	case a.Outcome == "struggled":
		return "Struggled"
	case a.Outcome == "unfinished":
		return "Unfinished"
	case a.Assisted:
		return "Solved with help"
	}
	return ""
}

// SelfFlagged reports whether the attempt itself flags its problem for
// review: a struggle, a review request, or a simpler approach.
func (a Attempt) SelfFlagged() bool {
	return Struggle(a) != "" || a.WantsReview || a.Approach == ApproachSuboptimal
}

const (
	ApproachOptimal = "optimal"
	// ApproachSuboptimal means a simpler solution was taken for time.
	ApproachSuboptimal = "suboptimal"
)

// ValidApproach reports whether s is a stored approach value.
func ValidApproach(s string) bool {
	return s == "" || s == ApproachOptimal || s == ApproachSuboptimal
}

// ApproachLabel describes a stated approach, or "" when not stated.
func ApproachLabel(s string) string {
	switch s {
	case ApproachOptimal:
		return "Reached the optimal solution"
	case ApproachSuboptimal:
		return "Took a simpler approach for time"
	}
	return ""
}

type State struct {
	// Attempts are ordered newest first.
	Attempts []Attempt
	// Problems is the catalog, keyed by slug.
	Problems map[string]Problem
	// Analyses maps attempt IDs to their LLM analysis.
	Analyses map[string]Analysis
	// Plans are the frozen review picks, and Goals the frozen daily goals,
	// by local date (see Date).
	Plans map[time.Time][]string
	Goals map[time.Time]DailyGoal
	// NewPlans are the frozen new-problem picks drawn from todos, by local
	// date, in slot order.
	NewPlans map[time.Time][]NewPick
}

// Problem returns the catalog row for slug, or a bare problem when the slug
// is not in the catalog yet.
func (s State) Problem(slug string) Problem {
	if p, ok := s.Problems[slug]; ok {
		return p
	}
	return Problem{Slug: slug}
}

// ProblemAttempts returns slug's attempts, newest first.
func (s State) ProblemAttempts(slug string) []Attempt {
	result := []Attempt{}
	for _, a := range s.Attempts {
		if a.ProblemSlug == slug {
			result = append(result, a)
		}
	}
	return result
}

// ProblemBest is the best result on slug over every attempt.
func (s State) ProblemBest(slug string) string {
	return bestOf(s.ProblemAttempts(slug))
}

// ProblemLatest is the result of the newest attempt on slug.
func (s State) ProblemLatest(slug string) string {
	return latestOf(s.ProblemAttempts(slug))
}

// Results of a problem. A best result is one of the first three; a latest
// result is any of the first four, or "Not attempted".
const (
	ResultIndependent = "Solved independently"
	ResultWithHelp    = "Solved with help"
	ResultStruggled   = "Struggled"
	ResultUnfinished  = "Unfinished"
	ResultNotSolved   = "Not solved"
	ResultNone        = "Not attempted"
)

// bestOf is the best result over one problem's attempts: solved
// independently, else solved with help, else not solved, whatever came later.
func bestOf(attempts []Attempt) string {
	if len(attempts) == 0 {
		return ResultNone
	}
	best := ResultNotSolved
	for _, a := range attempts {
		if a.Outcome == "solved" {
			if !a.Assisted {
				return ResultIndependent
			}
			best = ResultWithHelp
		}
	}
	return best
}

// latestOf is the result of the newest attempt, the first of attempts.
func latestOf(attempts []Attempt) string {
	if len(attempts) == 0 {
		return ResultNone
	}
	a := attempts[0]
	switch {
	case a.Outcome == "solved" && a.Assisted:
		return ResultWithHelp
	case a.Outcome == "solved":
		return ResultIndependent
	case a.Outcome == "struggled":
		return ResultStruggled
	}
	return ResultUnfinished
}

// ShowBest reports whether the best result adds to the latest: it is a solve
// and differs from it.
func ShowBest(best, latest string) bool {
	return isSolve(best) && best != latest
}

// isSolve reports whether a best or latest result is a solve.
func isSolve(result string) bool {
	return result == ResultIndependent || result == ResultWithHelp
}

// Progress counts distinct problems solved, and solved without help.
type Progress struct {
	Attempted   int
	Solved      int
	Independent int
}

func Summarize(state State) Progress {
	attempted, solved, independent := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, a := range state.Attempts {
		attempted[a.ProblemSlug] = true
		if a.Outcome == "solved" {
			solved[a.ProblemSlug] = true
			if !a.Assisted {
				independent[a.ProblemSlug] = true
			}
		}
	}
	return Progress{Attempted: len(attempted), Solved: len(solved), Independent: len(independent)}
}

// TopicLabel is the display name of one of the problem's topic slugs: the
// name LeetCode sent when stored, otherwise the slug title-cased.
func (p Problem) TopicLabel(slug string) string {
	if name := p.TopicNames[slug]; name != "" {
		return name
	}
	return TopicLabel(slug)
}
