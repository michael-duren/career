package leetgrinder

import "time"

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
	// MetadataSource is "seed", "extension", "leetcode", or "" for a bare row.
	MetadataSource string
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

// ProblemStatus summarises every attempt on slug in one label.
func (s State) ProblemStatus(slug string) string {
	attempts := s.ProblemAttempts(slug)
	assisted := false
	for _, a := range attempts {
		if a.Outcome == "solved" {
			if !a.Assisted {
				return "Solved independently"
			}
			assisted = true
		}
	}
	if assisted {
		return "Solved with help"
	}
	if len(attempts) > 0 {
		if attempts[0].Outcome == "struggled" {
			return "Struggled"
		}
		return "Unfinished"
	}
	return "Not attempted"
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
