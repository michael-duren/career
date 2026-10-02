package leetgrinder

import (
	"cmp"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

// ProblemRow is one attempted problem with the learner's progress on it.
type ProblemRow struct {
	Problem Problem
	// Best is the best result over every attempt, and Latest the result of
	// the newest one (see bestOf and latestOf).
	Best, Latest string
	Attempts     int
	// Last is the newest attempt.
	Last Attempt
	// ComplexityWrong reports that the analysis of Last judged a stated
	// complexity wrong.
	ComplexityWrong bool
	Card            Card
	// Due reports that the card is due by the end of today.
	Due bool
	// Recall is the estimated recall now.
	Recall float64
}

// Status filter keys, in the order the filter lists them.
var problemStatuses = []struct{ Key, Label string }{
	{"due", "Due for review"},
	{"flagged", "Flagged"},
	{"solved", "Ever solved"},
	{"solved-last", "Solved last time"},
	{"helped", "Needed help last time"},
	{"struggled", "Struggled last time"},
	{"unfinished", "Unfinished last time"},
}

var problemSorts = []struct{ Key, Label string }{
	{"recent", "Last attempt"},
	{"due", "Next due"},
	{"number", "LeetCode number"},
	{"title", "Title"},
	{"difficulty", "Difficulty"},
	{"recall", "Recall"},
}

var difficultyRank = map[string]int{"Easy": 1, "Medium": 2, "Hard": 3}

// ProblemFilter holds the search form; zero values match everything.
type ProblemFilter struct {
	Query      string
	Difficulty string
	Topic      string
	Status     string
	Sort       string
}

// ParseProblemFilter reads the search form, ignoring values it does not know.
func ParseProblemFilter(q url.Values) ProblemFilter {
	f := ProblemFilter{Query: strings.TrimSpace(q.Get("q")), Sort: "recent"}
	if r := []rune(f.Query); len(r) > 100 {
		f.Query = string(r[:100])
	}
	if _, ok := difficultyRank[q.Get("difficulty")]; ok {
		f.Difficulty = q.Get("difficulty")
	}
	if topic := q.Get("topic"); topic == untaggedTopic || (len(topic) <= MaxTopicLength && topicPattern.MatchString(topic)) {
		f.Topic = topic
	}
	for _, s := range problemStatuses {
		if q.Get("status") == s.Key {
			f.Status = s.Key
		}
	}
	for _, s := range problemSorts {
		if q.Get("sort") == s.Key {
			f.Sort = s.Key
		}
	}
	return f
}

// untaggedTopic is the topic filter value for problems without tags.
const untaggedTopic = "untagged"

// Active reports whether any filter narrows the list.
func (f ProblemFilter) Active() bool {
	return f.Query != "" || f.Difficulty != "" || f.Topic != "" || f.Status != ""
}

func (f ProblemFilter) match(r ProblemRow) bool {
	if !matchProblem(r.Problem, f.Query, f.Difficulty, f.Topic) {
		return false
	}
	switch f.Status {
	case "due":
		return r.Due
	case "flagged":
		return r.Card.Flagged()
	case "solved":
		return isSolve(r.Best)
	case "solved-last":
		return isSolve(r.Latest)
	case "helped":
		return r.Latest == ResultWithHelp
	case "struggled":
		return r.Latest == ResultStruggled
	case "unfinished":
		return r.Latest == ResultUnfinished
	}
	return true
}

// matchProblem applies the search text, difficulty and topic filters shared by
// the problem and todo lists; empty values match everything.
func matchProblem(p Problem, query, difficulty, topic string) bool {
	if query != "" {
		// A bare number is a LeetCode number; anything else searches the text.
		if n, err := strconv.Atoi(query); err == nil {
			if p.Number != n {
				return false
			}
		} else {
			text := p.Title + " " + p.Slug
			for _, t := range p.Topics {
				text += " " + t + " " + TopicLabel(t) + " " + p.TopicLabel(t)
			}
			if !strings.Contains(strings.ToLower(text), strings.ToLower(query)) {
				return false
			}
		}
	}
	if difficulty != "" && p.Difficulty != difficulty {
		return false
	}
	return topic != untaggedTopic && (topic == "" || slices.Contains(p.Topics, topic)) || topic == untaggedTopic && len(p.Topics) == 0
}

// ProblemRows lists every attempted problem, newest attempt first.
func ProblemRows(state State, cards []Card, now time.Time, loc *time.Location) []ProblemRow {
	byCard := make(map[string]Card, len(cards))
	for _, c := range cards {
		byCard[c.Problem.Slug] = c
	}
	// Attempts are newest first, so each slug's list is too, and slugs
	// are in order of their newest attempt.
	bySlug := map[string][]Attempt{}
	var slugs []string
	for _, a := range state.Attempts {
		if _, ok := bySlug[a.ProblemSlug]; !ok {
			slugs = append(slugs, a.ProblemSlug)
		}
		bySlug[a.ProblemSlug] = append(bySlug[a.ProblemSlug], a)
	}
	end := EndOfDate(Date(now, loc), loc)
	rows := make([]ProblemRow, 0, len(slugs))
	for _, slug := range slugs {
		attempts := bySlug[slug]
		a := attempts[0]
		row := ProblemRow{Problem: state.Problem(slug), Best: bestOf(attempts), Latest: latestOf(attempts), Attempts: len(attempts), Last: a, ComplexityWrong: state.StatedWrong(a.ID)}
		if c, ok := byCard[slug]; ok {
			row.Card, row.Due, row.Recall = c, c.Due.Before(end), c.Retrievability(now)
		}
		rows = append(rows, row)
	}
	return rows
}

// FilterProblems applies the filter and sort to rows without modifying them.
func FilterProblems(rows []ProblemRow, f ProblemFilter) []ProblemRow {
	var result []ProblemRow
	for _, r := range rows {
		if f.match(r) {
			result = append(result, r)
		}
	}
	slices.SortStableFunc(result, func(a, b ProblemRow) int {
		switch f.Sort {
		case "due":
			return a.Card.Due.Compare(b.Card.Due)
		case "number":
			// Unknown numbers sort last.
			return cmp.Compare(numberKey(a.Problem.Number), numberKey(b.Problem.Number))
		case "title":
			return cmp.Compare(strings.ToLower(a.Problem.DisplayTitle()), strings.ToLower(b.Problem.DisplayTitle()))
		case "difficulty":
			return cmp.Compare(difficultyRank[a.Problem.Difficulty], difficultyRank[b.Problem.Difficulty])
		case "recall":
			return cmp.Compare(a.Recall, b.Recall)
		}
		return b.Last.CreatedAt.Compare(a.Last.CreatedAt)
	})
	return result
}

func numberKey(n int) int {
	if n == 0 {
		return MaxNumber + 1
	}
	return n
}

// ProblemTopics lists the topic tags on rows, by label, with "untagged"
// last when any row has no tags.
func ProblemTopics(rows []ProblemRow) []FilterOption {
	problems := make([]Problem, len(rows))
	for i, r := range rows {
		problems[i] = r.Problem
	}
	return topicOptions(problems)
}

func topicOptions(problems []Problem) []FilterOption {
	seen := map[string]bool{}
	var out []FilterOption
	untagged := false
	for _, p := range problems {
		if len(p.Topics) == 0 {
			untagged = true
		}
		for _, t := range p.Topics {
			if !seen[t] {
				seen[t] = true
				out = append(out, FilterOption{t, p.TopicLabel(t)})
			}
		}
	}
	slices.SortFunc(out, func(a, b FilterOption) int { return cmp.Compare(a.Label, b.Label) })
	if untagged {
		out = append(out, FilterOption{untaggedTopic, "Untagged"})
	}
	return out
}

// ProblemsPage is the searchable list of every attempted problem.
type ProblemsPage struct {
	Rows     []ProblemRow
	Total    int
	Topics   []FilterOption
	Filter   ProblemFilter
	Location *time.Location
}

// FilterOption is one choice in a filter select.
type FilterOption struct{ Key, Label string }

func ProblemStatusOptions() []FilterOption {
	var o []FilterOption
	for _, s := range problemStatuses {
		o = append(o, FilterOption(s))
	}
	return o
}

func ProblemSortOptions() []FilterOption {
	var o []FilterOption
	for _, s := range problemSorts {
		o = append(o, FilterOption(s))
	}
	return o
}

// StatusClass maps a progress label to its pill class.
func StatusClass(label string) string {
	switch label {
	case ResultIndependent:
		return "status-pill status-solved"
	case ResultWithHelp:
		return "status-pill status-solved-help"
	case ResultStruggled:
		return "status-pill status-struggled"
	case ResultUnfinished:
		return "status-pill status-unfinished"
	}
	return "status-pill status-not-attempted"
}
