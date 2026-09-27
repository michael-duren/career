package leetgrinder

import "time"

type Problem struct {
	ID         int
	Slug       string
	Title      string
	Difficulty string
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
}
type State struct {
	Attempts      []Attempt `json:"attempts"`
	CompletedDays []int     `json:"completedDays"`
}
