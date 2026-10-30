package leetgrinder

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

const MaxCorrectnessAnswer = 2000

var ErrCorrectnessInvalid = errors.New("each correctness answer must be valid text of 2,000 characters or fewer")

type Correctness struct {
	Claim       string `json:"claim"`
	Invariant   string `json:"invariant"`
	Initially   string `json:"initially"`
	AfterStep   string `json:"afterStep"`
	Therefore   string `json:"therefore"`
	Termination string `json:"termination"`
}

type CorrectnessField struct{ Name, Label, Hint, Value string }

func (c Correctness) Fields() []CorrectnessField {
	return []CorrectnessField{
		{"claim", "Claim", "What exactly does my algorithm guarantee?", c.Claim},
		{"invariant", "Invariant / induction hypothesis / recurrence relation", "What remains true throughout the algorithm?", c.Invariant},
		{"initially", "1. Initially...", "Why does it hold initially?", c.Initially},
		{"afterStep", "2. After one step...", "Why does one step preserve it?", c.AfterStep},
		{"therefore", "3. Therefore...", "What follows from initialization and preservation?", c.Therefore},
		{"termination", "Termination", "When the algorithm ends, why does this imply the answer is correct?", c.Termination},
	}
}

func (c Correctness) Validate() error {
	for _, f := range c.Fields() {
		if !utf8.ValidString(f.Value) || strings.ContainsRune(f.Value, 0) || utf8.RuneCountInString(f.Value) > MaxCorrectnessAnswer {
			return ErrCorrectnessInvalid
		}
	}
	return nil
}

// FeedbackSection is one labeled part of Claude's correctness feedback. Step
// marks the numbered parts of "Why it holds".
type FeedbackSection struct {
	Label, Body string
	Step        bool
}

// feedbackLabel matches the section labels the analysis prompt asks for. A
// match counts when it ends in a colon, stands alone at the start of a line,
// or is "Why it holds" or a numbered step, so prose such as "Termination is
// fine" is not split.
var feedbackLabel = regexp.MustCompile(`(?m)(^|\s)(Claim|Invariant(?: / induction hypothesis / recurrence relation)?|Why it holds(?: \([^)\n]*\))?|[123]\. (?:Initially|After one step|Therefore)|Termination)(\s*:|,)?`)

// CorrectnessSections splits correctness feedback into its labeled sections.
// It returns nil when the text has no recognisable labels.
func CorrectnessSections(text string) []FeedbackSection {
	var sections []FeedbackSection
	body := 0
	add := func(end int) {
		if len(sections) == 0 {
			if s := strings.TrimSpace(text[:end]); s != "" {
				sections = append(sections, FeedbackSection{Body: s})
			}
			return
		}
		sections[len(sections)-1].Body = strings.TrimSpace(text[body:end])
	}
	for _, m := range feedbackLabel.FindAllStringSubmatchIndex(text, -1) {
		before, label, colon := text[m[2]:m[3]], text[m[4]:m[5]], m[6] >= 0 && strings.Contains(text[m[6]:m[7]], ":")
		step := label[0] >= '1' && label[0] <= '3'
		ownLine := (before == "" || before == "\n") && (m[6] >= 0 || m[1] == len(text) || text[m[1]] == '\n')
		if !colon && !step && !ownLine && !strings.HasPrefix(label, "Why it holds") {
			continue
		}
		add(m[2])
		if i := strings.Index(label, " ("); i >= 0 {
			label = label[:i]
		}
		sections = append(sections, FeedbackSection{Label: label, Step: step})
		body = m[1]
	}
	if len(sections) == 0 {
		return nil
	}
	add(len(text))
	return sections
}
