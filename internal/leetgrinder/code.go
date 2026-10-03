package leetgrinder

import (
	"bytes"
	"html/template"
	"net/url"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
)

// chromaLexers maps LeetCode language slugs to chroma lexer names where
// they differ.
var chromaLexers = map[string]string{
	"python3": "python", "golang": "go", "cpp": "c++", "csharp": "c#", "text": "plaintext",
}

var codeFormatter = html.New(html.WithClasses(true), html.PreventSurroundingPre(true))

// HighlightCode renders code as HTML spans with chroma's token classes (see
// style.css), for inside a <code> element. Chroma escapes the code; an
// unknown language or a lexer failure falls back to escaped plain text.
func HighlightCode(code, lang string) template.HTML {
	name := lang
	if n, ok := chromaLexers[lang]; ok {
		name = n
	}
	lexer := lexers.Get(name)
	if lexer == nil || lang == "" {
		return template.HTML(template.HTMLEscapeString(code))
	}
	tokens, err := chroma.Coalesce(lexer).Tokenise(nil, code)
	if err != nil {
		return template.HTML(template.HTMLEscapeString(code))
	}
	var out bytes.Buffer
	if err = codeFormatter.Format(&out, chromaStyle, tokens); err != nil {
		return template.HTML(template.HTMLEscapeString(code))
	}
	return template.HTML(out.String())
}

// chromaStyle only satisfies the formatter: with classes on, colours come
// from style.css.
var chromaStyle = chroma.MustNewStyle("leetgrinder", chroma.StyleEntries{})

// LatestCodeID is the ID of the newest attempt with code, captured or
// pasted, or "".
// Attempts are newest first.
func LatestCodeID(attempts []Attempt) string {
	for _, a := range attempts {
		if a.Code != "" {
			return a.ID
		}
	}
	return ""
}

// CodeAttempts are the attempts with code, newest first, for comparing.
func CodeAttempts(attempts []Attempt) []Attempt {
	var out []Attempt
	for _, a := range attempts {
		if a.Code != "" {
			out = append(out, a)
		}
	}
	return out
}

// DiffLine is one line of a line diff from a to b: Op is ' ' (in both),
// '-' (only in a) or '+' (only in b).
type DiffLine struct {
	Op   byte
	Text string
}

// MaxDiffCells bounds the diff table (lines × lines), about 2,000 lines of
// each, so a compare stays fast.
const MaxDiffCells = 4_000_000

// DiffLines is a line diff from a to b by longest common subsequence, or
// false when the inputs are too long to compare.
func DiffLines(a, b string) ([]DiffLine, bool) {
	x, y := splitLines(a), splitLines(b)
	if len(x)*len(y) > MaxDiffCells {
		return nil, false
	}
	// lcs[i][j] is the LCS length of x[i:] and y[j:].
	lcs := make([][]int32, len(x)+1)
	for i := range lcs {
		lcs[i] = make([]int32, len(y)+1)
	}
	for i := len(x) - 1; i >= 0; i-- {
		for j := len(y) - 1; j >= 0; j-- {
			if x[i] == y[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var out []DiffLine
	i, j := 0, 0
	for i < len(x) && j < len(y) {
		switch {
		case x[i] == y[j]:
			out = append(out, DiffLine{' ', x[i]})
			i, j = i+1, j+1
		case lcs[i+1][j] >= lcs[i][j+1]:
			out = append(out, DiffLine{'-', x[i]})
			i++
		default:
			out = append(out, DiffLine{'+', y[j]})
			j++
		}
	}
	for ; i < len(x); i++ {
		out = append(out, DiffLine{'-', x[i]})
	}
	for ; j < len(y); j++ {
		out = append(out, DiffLine{'+', y[j]})
	}
	return out, true
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// DiffCounts are the lines removed and added.
func DiffCounts(lines []DiffLine) (removed, added int) {
	for _, l := range lines {
		switch l.Op {
		case '-':
			removed++
		case '+':
			added++
		}
	}
	return removed, added
}

// DiffClass is the CSS class for a diff line.
func DiffClass(op byte) string {
	switch op {
	case '-':
		return "diff-del"
	case '+':
		return "diff-add"
	}
	return "diff-same"
}

// CompareCodePage compares the code of two attempts on one problem.
type CompareCodePage struct {
	Problem  Problem
	From, To Attempt
	// Choices are the attempts with code, newest first.
	Choices []Attempt
	Lines   []DiffLine
	// TooLarge reports code too long to diff.
	TooLarge bool
	Review   ProblemReview
}

// CompareURL is the compare page for two attempts' code.
func CompareURL(slug, from, to string) string {
	return ProblemURL(slug) + "/compare?" + url.Values{"from": {from}, "to": {to}}.Encode()
}

// CodeChoiceLabel names an attempt in the compare selects, by its local
// time and outcome.
func CodeChoiceLabel(a Attempt, review ProblemReview) string {
	return review.AttemptTime(a).Format("02 Jan 2006 15:04") + " · " + OutcomeLabel(a.Outcome)
}

// DiffLabel reads a diff line's change for screen readers.
func DiffLabel(op byte) string {
	switch op {
	case '-':
		return "Removed: "
	case '+':
		return "Added: "
	}
	return ""
}
