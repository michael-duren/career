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
	if len(latest.FindAllString(body, -1)) != 1 || strings.Count(body, `<details class="attempt-code" open`) != 1 || strings.Count(body, `<details class="attempt-code">`) != 1 {
		t.Fatal("latest code not the only open, labelled block")
	}
	// The copy button stays hidden until the script finds a clipboard.
	if !strings.Contains(body, `data-copy="code-`+newer+`" hidden`) || !strings.Contains(body, "if (!navigator.clipboard) return;") {
		t.Fatal("copy button not hidden by default")
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
	for _, want := range []string{"−1", "+1", `<span class="diff-del"><span class="diff-op" aria-hidden="true">-</span><span class="visually-hidden">Removed: </span>    return 1</span>`, `<span class="diff-add"><span class="diff-op" aria-hidden="true">+</span><span class="visually-hidden">Added: </span>    return 2  # &lt;b&gt;</span>`} {
		if !strings.Contains(body, want) {
			t.Errorf("compare missing %q", want)
		}
	}
	// The plain attempt has no code, so it cannot be compared.
	state, err := db.LeetgrinderProblemState(ctx, "two-sum")
	if err != nil {
		t.Fatal(err)
	}
	plain := state.ProblemAttempts("two-sum")[0].ID
	for _, bad := range []string{leetgrinder.CompareURL("two-sum", older, uuid.NewString()), leetgrinder.CompareURL("valid-anagram", older, newer), "/leetgrinder/problem/two-sum/compare", leetgrinder.CompareURL("two-sum", plain, newer)} {
		if w := request("GET", bad, nil); w.Code != 404 {
			t.Errorf("%s: %d", bad, w.Code)
		}
	}
	// Code too long to diff gets a message, not a 500.
	long1, long2 := save(strings.Repeat("a\n", 2100)), save(strings.Repeat("b\n", 2100))
	if w := request("GET", leetgrinder.CompareURL("two-sum", long1, long2), nil); w.Code != 200 || !strings.Contains(w.Body.String(), "This code is too long to compare line by line.") {
		t.Fatalf("too long: %d", w.Code)
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
	if status, body := post(base("print(1)", "cobol")); status != 400 || !strings.Contains(body, "Choose the language of your code.") {
		t.Fatalf("unknown language: %d", status)
	}
	if status, body := post(base("a\x00b", "text")); status != 400 || !strings.Contains(body, "without null characters") {
		t.Fatalf("null character: %d", status)
	}
	// A rejected draft keeps code that starts with a blank line.
	if _, body := post(base("\nprint(3)", "")); !strings.Contains(body, "placeholder=\"Paste the code you wrote\">\n\nprint(3)</textarea>") {
		t.Fatal("leading newline dropped from the draft")
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
	// A correction form has no code field, and a correction keeps the saved
	// code even if code is posted with it.
	body := request("GET", "/leetgrinder/problem/two-sum", nil).Body.String()
	if strings.Count(body, `name="code"`) != 1 {
		t.Fatalf("code fields: %d", strings.Count(body, `name="code"`))
	}
	fix := url.Values{"id": {withCode[0].ID}, "revision": {withCode[0].Revision}, "outcome": {"unfinished"}, "minutes": {"7"}, "code": {"a\x00b"}, "codeLanguage": {"cobol"}}
	if status, _ := post(fix); status != 303 {
		t.Fatalf("correction: %d", status)
	}
	if state, err = db.LeetgrinderProblemState(context.Background(), "two-sum"); err != nil {
		t.Fatal(err)
	}
	for _, a := range state.ProblemAttempts("two-sum") {
		if a.ID == withCode[0].ID && (a.Minutes != 7 || a.Code != "print(1)\nprint(2)" || a.CodeLanguage != "python3") {
			t.Fatalf("correction changed code: %+v", a.Code)
		}
	}
}
