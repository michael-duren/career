package leetgrinder

import "time"

type Problem struct {
	ID         int
	Slug       string
	Title      string
	Difficulty string
	// OptimalTime and OptimalSpace are the best-known bounds under the
	// problem's constraints, in the notation NormalizeComplexity produces. Space is auxiliary
	// space and excludes the returned output.
	OptimalTime  string
	OptimalSpace string
	// OptimalNote defines variables other than n and names alternatives.
	OptimalNote string
}

func (p Problem) URL() string { return "https://leetcode.com/problems/" + p.Slug + "/" }

type Reading struct {
	Title    string
	URL      string
	Guidance string
	Minutes  int
	Optional bool
}
type Day struct {
	Number   int
	Title    string
	Lesson   string
	Readings []Reading
	Core     []Problem
	Optional []Problem
}
type Week struct {
	Number  int
	Title   string
	Summary string
	Days    []Day
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
}
type State struct {
	Attempts      []Attempt `json:"attempts"`
	CompletedDays []int     `json:"completedDays"`
}
