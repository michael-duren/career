package leetgrinder

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Complexities are the canonical choices offered for stated time and space
// complexity, stored exactly as shown. Any other value is free text that
// passes NormalizeComplexity.
var Complexities = []string{"O(1)", "O(log n)", "O(√n)", "O(n)", "O(n log n)", "O(n²)", "O(n³)", "O(2ⁿ)", "O(n!)"}

const (
	MaxComplexityLen   = 40
	MaxCodeBytes       = 64 << 10
	MaxCodeLanguageLen = 32
)

var (
	ErrComplexityRequired = errors.New("time and space complexity are required for solved and struggled attempts")
	ErrComplexityFormat   = errors.New("complexity must start with O( and end with ), in 40 characters or fewer")
	ErrCodeTooLarge       = errors.New("code is over 64 KiB")
	ErrCodeInvalid        = errors.New("code and its language are invalid")
	ErrApproachInvalid    = errors.New("approach must be optimal, suboptimal, or empty")
)

var codeLanguage = regexp.MustCompile(`^[A-Za-z0-9_+#.-]{1,32}$`)

// complexitySpellings are applied in order, one after another; the extension
// applies the same list the same way (lib.js), so both agree on every input.
var complexitySpellings = [][2]string{{"nlogn", "n log n"}, {"logn", "log n"}, {"n^2", "n²"}, {"n^3", "n³"}, {"2^n", "2ⁿ"}}

func asciiSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	}
	return false
}

// NormalizeComplexity trims and canonicalises a stated complexity. An empty
// input stays empty (not stated). Anything else must read O(...) in at most
// MaxComplexityLen characters after normalisation.
func NormalizeComplexity(s string) (string, error) {
	if !utf8.ValidString(s) {
		return "", ErrComplexityFormat
	}
	s = strings.Join(strings.FieldsFunc(s, asciiSpace), " ")
	if s == "" {
		return "", nil
	}
	if strings.HasPrefix(s, "o(") {
		s = "O(" + s[2:]
	}
	for _, spelling := range complexitySpellings {
		s = strings.ReplaceAll(s, spelling[0], spelling[1])
	}
	if utf8.RuneCountInString(s) > MaxComplexityLen || len(s) < 4 || !strings.HasPrefix(s, "O(") || !strings.HasSuffix(s, ")") || strings.IndexFunc(s, unicode.IsControl) >= 0 {
		return "", ErrComplexityFormat
	}
	return s, nil
}

// IsCanonicalComplexity reports whether s is one of Complexities.
func IsCanonicalComplexity(s string) bool {
	for _, c := range Complexities {
		if s == c {
			return true
		}
	}
	return false
}

// NeedsComplexity reports whether an outcome requires stated complexity.
func NeedsComplexity(outcome string) bool { return outcome == "solved" || outcome == "struggled" }

// NormalizeDetails normalises the attempt's stated complexities in place and
// checks them and the captured code. Complexity is required for solved and
// struggled attempts. Code and its language are both set or both empty.
func (a *Attempt) NormalizeDetails() error {
	var err error
	if a.TimeComplexity, err = NormalizeComplexity(a.TimeComplexity); err != nil {
		return err
	}
	if a.SpaceComplexity, err = NormalizeComplexity(a.SpaceComplexity); err != nil {
		return err
	}
	if NeedsComplexity(a.Outcome) && (a.TimeComplexity == "" || a.SpaceComplexity == "") {
		return ErrComplexityRequired
	}
	if len(a.Code) > MaxCodeBytes {
		return ErrCodeTooLarge
	}
	if !utf8.ValidString(a.Code) || strings.ContainsRune(a.Code, 0) {
		return ErrCodeInvalid
	}
	if a.Code == "" && a.CodeLanguage != "" || a.Code != "" && !codeLanguage.MatchString(a.CodeLanguage) {
		return ErrCodeInvalid
	}
	if !ValidApproach(a.Approach) {
		return ErrApproachInvalid
	}
	return nil
}

var languageLabels = map[string]string{
	"c": "C", "cpp": "C++", "csharp": "C#", "dart": "Dart", "elixir": "Elixir", "erlang": "Erlang", "golang": "Go", "java": "Java",
	"javascript": "JavaScript", "kotlin": "Kotlin", "php": "PHP", "python": "Python", "python3": "Python3", "racket": "Racket",
	"ruby": "Ruby", "rust": "Rust", "scala": "Scala", "swift": "Swift", "typescript": "TypeScript",
	// Code pasted into the extension's log panel may be plain text.
	"text": "Plain text",
}

// CodeLanguages are the choices for code pasted on the web form, in the
// order the extension's PASTE_LANGUAGES lists them.
var CodeLanguages = []string{"python3", "java", "cpp", "c", "csharp", "javascript", "typescript", "golang", "rust", "kotlin", "swift", "ruby", "scala", "php", "dart", "elixir", "erlang", "racket", "text"}

// LanguageLabel names a LeetCode language slug for display.
func LanguageLabel(lang string) string {
	if label, ok := languageLabels[lang]; ok {
		return label
	}
	return lang
}
