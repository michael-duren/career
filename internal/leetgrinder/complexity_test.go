package leetgrinder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// The extension runs the same vectors (extension/leetgrinder/test), so the
// Go and JavaScript validators cannot drift apart.
func TestNormalizeComplexityVectors(t *testing.T) {
	raw, err := os.ReadFile("../../extension/leetgrinder/test/complexity-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors []struct {
		In    string `json:"in"`
		Out   string `json:"out"`
		Error bool   `json:"error"`
	}
	if err = json.Unmarshal(raw, &vectors); err != nil || len(vectors) < 20 {
		t.Fatalf("vectors: %d %v", len(vectors), err)
	}
	for _, v := range vectors {
		got, err := NormalizeComplexity(v.In)
		if v.Error {
			if !errors.Is(err, ErrComplexityFormat) {
				t.Errorf("%q: got %q, want an error", v.In, got)
			}
			continue
		}
		if err != nil || got != v.Out {
			t.Errorf("%q: got %q %v, want %q", v.In, got, err, v.Out)
		}
	}
	for _, c := range Complexities {
		if got, err := NormalizeComplexity(c); err != nil || got != c || !IsCanonicalComplexity(c) {
			t.Errorf("canonical %q changed: %q %v", c, got, err)
		}
	}
	if _, err := NormalizeComplexity("O(\xff)"); err == nil {
		t.Error("invalid UTF-8 accepted")
	}
}

func TestAttemptNormalizeDetails(t *testing.T) {
	valid := Attempt{Outcome: "solved", TimeComplexity: " O(nlogn) ", SpaceComplexity: "O(n)", Code: "print(1)\n", CodeLanguage: "python3"}
	a := valid
	if err := a.NormalizeDetails(); err != nil || a.TimeComplexity != "O(n log n)" {
		t.Fatalf("valid attempt: %v %+v", err, a)
	}
	for name, test := range map[string]struct {
		change func(*Attempt)
		want   error
	}{
		"solved without time":        {func(a *Attempt) { a.TimeComplexity = "" }, ErrComplexityRequired},
		"struggled without space":    {func(a *Attempt) { a.Outcome, a.SpaceComplexity = "struggled", " " }, ErrComplexityRequired},
		"bad format":                 {func(a *Attempt) { a.SpaceComplexity = "linear" }, ErrComplexityFormat},
		"bad format when unfinished": {func(a *Attempt) { a.Outcome, a.TimeComplexity = "unfinished", "n" }, ErrComplexityFormat},
		"code too large":             {func(a *Attempt) { a.Code = strings.Repeat("x", MaxCodeBytes+1) }, ErrCodeTooLarge},
		"code without language":      {func(a *Attempt) { a.CodeLanguage = "" }, ErrCodeInvalid},
		"language without code":      {func(a *Attempt) { a.Code = "" }, ErrCodeInvalid},
		"bad language":               {func(a *Attempt) { a.CodeLanguage = "python 3" }, ErrCodeInvalid},
		"long language":              {func(a *Attempt) { a.CodeLanguage = strings.Repeat("a", 33) }, ErrCodeInvalid},
		"NUL in code":                {func(a *Attempt) { a.Code = "a\x00b" }, ErrCodeInvalid},
		"invalid UTF-8 code":         {func(a *Attempt) { a.Code = "\xff" }, ErrCodeInvalid},
		"unknown approach":           {func(a *Attempt) { a.Approach = "brute-force" }, ErrApproachInvalid},
	} {
		a := valid
		test.change(&a)
		if err := a.NormalizeDetails(); !errors.Is(err, test.want) {
			t.Errorf("%s: got %v, want %v", name, err, test.want)
		}
	}
	for _, a := range []Attempt{
		{Outcome: "unfinished"},
		{Outcome: "unfinished", Code: "x", CodeLanguage: "cpp"},
		{Outcome: "unfinished", Approach: ApproachSuboptimal, WantsReview: true},
		{Outcome: "solved", TimeComplexity: "O(1)", SpaceComplexity: "O(1)", Code: strings.Repeat("é", MaxCodeBytes/2), CodeLanguage: "c#"},
	} {
		if err := a.NormalizeDetails(); err != nil {
			t.Errorf("%+v: %v", a.Outcome, err)
		}
	}
}

func TestComplexityInput(t *testing.T) {
	for _, test := range []struct {
		value string
		want  ComplexityInput
	}{
		{"", ComplexityInput{}},
		{"O(n²)", ComplexityInput{Choice: "O(n²)"}},
		{"O(V + E)", ComplexityInput{Choice: "other", Other: "O(V + E)"}},
	} {
		got := NewComplexityInput(test.value)
		if got != test.want || got.Value() != test.value {
			t.Errorf("%q: %+v %q", test.value, got, got.Value())
		}
	}
	for _, test := range []struct {
		in   ComplexityInput
		want string
	}{
		{ComplexityInput{Choice: "O(n)", Other: "O(1)"}, "O(n)"},
		{ComplexityInput{Choice: "other", Other: "O(k)"}, "O(k)"},
		// Without JavaScript the text field is submitted with nothing selected.
		{ComplexityInput{Other: "O(k)"}, "O(k)"},
	} {
		if got := test.in.Value(); got != test.want {
			t.Errorf("%+v: %q", test.in, got)
		}
	}
}

func TestHistoryShowsComplexityAndEscapedCode(t *testing.T) {
	problem := Problem{Number: 1, Slug: "two-sum", Title: "Two Sum"}
	state := State{Attempts: []Attempt{{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "solved", Minutes: 12, CreatedAt: time.Now(), Revision: uuid.NewString(), TimeComplexity: "O(n)", SpaceComplexity: "O(V + E)", Code: "if a < b && c > d:\n    return '</code><script>alert(1)</script>'", CodeLanguage: "python3"}}}
	var out bytes.Buffer
	if err := ProblemHistory(problem, state, NewForm(uuid.NewString()), AnalysisAvailability{}, ProblemReview{}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{"Time <strong>O(n)</strong>", "Space <strong>O(V + E)</strong>", "Submitted code (Python3,", `<details class="attempt-code" open>`, `<pre class="chroma"><code`, "&lt;/code&gt;&lt;script&gt;", `name="timeComplexityOther"`, `<option value="other" selected>Other…</option>`, `value="O(V + E)"`} {
		if !strings.Contains(html, want) {
			t.Errorf("history missing %q", want)
		}
	}
	if strings.Contains(html, "<script>alert(1)") {
		t.Fatal("code rendered unescaped")
	}
	// Highlighting splits the code into spans; its text stays escaped.
	if text := regexp.MustCompile(`</?span[^>]*>`).ReplaceAllString(html, ""); !strings.Contains(text, "if a &lt; b &amp;&amp; c &gt; d:") {
		t.Fatal("highlighted code text changed")
	}
}

func TestAttemptFormKeepsComplexityDraft(t *testing.T) {
	problem := Problem{Number: 1, Slug: "two-sum", Title: "Two Sum"}
	form := AttemptForm{ID: uuid.NewString(), Outcome: "solved", Minutes: "20", Time: ComplexityInput{Choice: "O(n log n)"}, Space: ComplexityInput{Choice: "other", Other: "O(<k>)"}, Error: "Please retry"}
	var out bytes.Buffer
	if err := ProblemHistory(problem, State{}, form, AnalysisAvailability{}, ProblemReview{}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{`<option value="O(n log n)" selected>`, `name="spaceComplexity"`, `<option value="other" selected>`, `value="O(&lt;k&gt;)"`, "Please retry"} {
		if !strings.Contains(html, want) {
			t.Errorf("draft missing %q", want)
		}
	}
}

// TestPasteLanguagesHaveLabels keeps the extension's paste language list,
// in lib.js, in step with the app: each is accepted and has a label.
func TestPasteLanguagesHaveLabels(t *testing.T) {
	src, err := os.ReadFile("../../extension/leetgrinder/lib.js")
	if err != nil {
		t.Fatal(err)
	}
	list := regexp.MustCompile(`const PASTE_LANGUAGES = \[([^\]]*)\]`).FindSubmatch(src)
	if list == nil {
		t.Fatal("PASTE_LANGUAGES not found in lib.js")
	}
	langs := regexp.MustCompile(`"([^"]+)"`).FindAllSubmatch(list[1], -1)
	if len(langs) == 0 {
		t.Fatal("PASTE_LANGUAGES is empty")
	}
	var js []string
	for _, m := range langs {
		js = append(js, string(m[1]))
	}
	if strings.Join(js, ",") != strings.Join(CodeLanguages, ",") {
		t.Errorf("web form languages %v differ from the extension's %v", CodeLanguages, js)
	}
	for _, m := range langs {
		lang := string(m[1])
		if LanguageLabel(lang) == lang {
			t.Errorf("paste language %q has no label in languageLabels", lang)
		}
		a := Attempt{Outcome: "unfinished", Code: "x", CodeLanguage: lang}
		if err := a.NormalizeDetails(); err != nil {
			t.Errorf("paste language %q rejected: %v", lang, err)
		}
	}
}
