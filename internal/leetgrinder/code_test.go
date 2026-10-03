package leetgrinder

import (
	"regexp"
	"strings"
	"testing"
)

func TestHighlightCode(t *testing.T) {
	html := string(HighlightCode("def f(x):\n    return x < 1  # <b>\n", "python3"))
	for _, want := range []string{`<span class="k">def</span>`, `&lt;`, `&lt;b&gt;`} {
		if !strings.Contains(html, want) {
			t.Errorf("python highlight missing %q in %s", want, html)
		}
	}
	if strings.Contains(html, "<b>") || strings.Contains(html, "<pre") {
		t.Fatalf("unescaped or wrapped: %s", html)
	}
	// Every LeetCode language the app labels gets a lexer, or plain text.
	// Every language the app labels is escaped; only span tags are added.
	spans := regexp.MustCompile(`</?span[^>]*>`)
	for lang := range languageLabels {
		out := string(HighlightCode("x <y>", lang))
		if text := spans.ReplaceAllString(out, ""); text != "x &lt;y&gt;" {
			t.Errorf("%s: %s", lang, out)
		}
	}
	if got := string(HighlightCode("a<b", "brainfuck-ish")); got != "a&lt;b" {
		t.Fatalf("unknown language: %s", got)
	}
}

func TestDiffLines(t *testing.T) {
	lines, ok := DiffLines("a\nb\nc\n", "a\nc\nd\n")
	if !ok {
		t.Fatal("too large")
	}
	var got []string
	for _, l := range lines {
		got = append(got, string(l.Op)+l.Text)
	}
	if strings.Join(got, "|") != " a|-b| c|+d" {
		t.Fatalf("diff %q", got)
	}
	if r, a := DiffCounts(lines); r != 1 || a != 1 {
		t.Fatalf("counts %d %d", r, a)
	}
	if lines, _ := DiffLines("x\r\ny", "x\ny"); len(lines) != 2 || lines[0].Op != ' ' || lines[1].Op != ' ' {
		t.Fatalf("CRLF: %v", lines)
	}
	if lines, _ := DiffLines("", "a"); len(lines) != 1 || lines[0].Op != '+' {
		t.Fatalf("from empty: %v", lines)
	}
	big := strings.Repeat("line\n", 2100)
	if _, ok := DiffLines(big, big); ok {
		t.Fatal("over the cell limit was diffed")
	}
}

func TestLatestCodeID(t *testing.T) {
	attempts := []Attempt{{ID: "new"}, {ID: "mid", Code: "x"}, {ID: "old", Code: "y"}}
	if LatestCodeID(attempts) != "mid" || LatestCodeID(nil) != "" || len(CodeAttempts(attempts)) != 2 {
		t.Fatal("latest code")
	}
}
