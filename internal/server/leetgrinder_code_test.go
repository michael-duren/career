package server

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderCodeDisplayAndCompare(t *testing.T) {
	_, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	save := func(code string) string {
		t.Helper()
		a := leetgrinder.Attempt{ID: uuid.NewString(), ProblemSlug: "two-sum", Outcome: "solved", Minutes: 10, TimeComplexity: "O(n)", SpaceComplexity: "O(n)", Code: code, CodeLanguage: "python3", Source: "extension"}
		if _, err := db.SaveLeetgrinderAttempt(ctx, a, ""); err != nil {
			t.Fatal(err)
		}
		return a.ID
	}
	older := save("def f(nums):\n    return 1\n")
	newer := save("def f(nums):\n    return 2  # <b>\n")
	// The newest attempt has no code, so the newer one is the latest code.
	if w := request("POST", "/leetgrinder/problem/two-sum/attempts", url.Values{"id": {uuid.NewString()}, "outcome": {"unfinished"}, "minutes": {"5"}}); w.Code != 303 {
		t.Fatalf("plain attempt: %d", w.Code)
	}
	body := request("GET", "/leetgrinder/problem/two-sum", nil).Body.String()
	latest := regexp.MustCompile(`(?s)<details class="attempt-code" open>\s*<summary><span class="badge">Latest code</span>`)
	if len(latest.FindAllString(body, -1)) != 1 {
		t.Fatal("latest code not open and labelled once")
	}
	for _, want := range []string{
		`<span class="k">def</span>`, "&lt;b&gt;", `data-copy="code-` + newer + `"`,
		`href="` + strings.ReplaceAll(leetgrinder.CompareURL("two-sum", older, newer), "&", "&amp;") + `"`,
		`action="/leetgrinder/problem/two-sum/compare"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("problem page missing %q", want)
		}
	}
	if strings.Contains(body, "# <b>") {
		t.Fatal("code not escaped")
	}

	w := request("GET", leetgrinder.CompareURL("two-sum", older, newer), nil)
	body = w.Body.String()
	if w.Code != 200 {
		t.Fatalf("compare: %d", w.Code)
	}
	for _, want := range []string{"−1", "+1", `class="diff-del"`, `class="diff-add"`, "return 1", "return 2  # &lt;b&gt;", "Removed: "} {
		if !strings.Contains(body, want) {
			t.Errorf("compare missing %q", want)
		}
	}
	for _, bad := range []string{leetgrinder.CompareURL("two-sum", older, uuid.NewString()), leetgrinder.CompareURL("valid-anagram", older, newer), "/leetgrinder/problem/two-sum/compare"} {
		if w := request("GET", bad, nil); w.Code != 404 {
			t.Errorf("%s: %d", bad, w.Code)
		}
	}
}

func TestLeetgrinderWebFormCode(t *testing.T) {
	_, db, request := leetgrinderTestServer(t)
	post := func(v url.Values) (int, string) {
		w := request("POST", "/leetgrinder/problem/two-sum/attempts", v)
		return w.Code, w.Body.String()
	}
	base := func(code, lang string) url.Values {
		return url.Values{"id": {uuid.NewString()}, "outcome": {"unfinished"}, "minutes": {"5"}, "code": {code}, "codeLanguage": {lang}}
	}
	if status, body := post(base("print(1)", "")); status != 400 || !strings.Contains(body, "Choose the language of your code.") || !strings.Contains(body, "print(1)") {
		t.Fatalf("no language: %d", status)
	}
	if status, body := post(base(strings.Repeat("x", leetgrinder.MaxCodeBytes+1), "text")); status != 400 || !strings.Contains(body, "Keep the code to 64 KiB or less.") {
		t.Fatalf("too large: %d", status)
	}
	// 64 KiB of characters URL encoding triples still fits the form limit.
	if status, _ := post(base(strings.Repeat("{", leetgrinder.MaxCodeBytes), "text")); status != 303 {
		t.Fatalf("64 KiB of code: %d", status)
	}
	if status, _ := post(base("print(1)\r\nprint(2)", "python3")); status != 303 {
		t.Fatalf("save: %d", status)
	}
	// Blank code is ignored, whatever language is chosen.
	if status, _ := post(base("   ", "java")); status != 303 {
		t.Fatalf("blank: %d", status)
	}
	state, err := db.LeetgrinderProblemState(context.Background(), "two-sum")
	if err != nil {
		t.Fatal(err)
	}
	attempts := state.ProblemAttempts("two-sum")
	var withCode []leetgrinder.Attempt
	for _, a := range attempts {
		if a.Code != "" {
			withCode = append(withCode, a)
		}
	}
	// Newest first: the blank paste, the print paste, the 64 KiB paste.
	if len(attempts) != 3 || len(withCode) != 2 || withCode[0].Code != "print(1)\nprint(2)" || withCode[0].CodeLanguage != "python3" || len(withCode[1].Code) != leetgrinder.MaxCodeBytes {
		t.Fatalf("saved %d attempts, %d with code", len(attempts), len(withCode))
	}
	// A correction form has no code field and keeps the saved code.
	body := request("GET", "/leetgrinder/problem/two-sum", nil).Body.String()
	if strings.Count(body, `name="code"`) != 1 {
		t.Fatalf("code fields: %d", strings.Count(body, `name="code"`))
	}
}
