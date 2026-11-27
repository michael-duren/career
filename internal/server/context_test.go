package server

import (
	"strings"
	"testing"
)

func TestMarkdownEntriesSkipsMissingBody(t *testing.T) {
	out := markdownEntries([]any{
		map[string]any{"title": "Give a talk", "status": "active"},
		map[string]any{"title": "Raft", "body": "Terms and elections."},
	})
	if strings.Contains(out, "<nil>") {
		t.Fatalf("a goal without a body printed <nil>:\n%s", out)
	}
	if strings.Count(out, "````markdown") != 1 || !strings.Contains(out, "````markdown\nTerms and elections.\n````") {
		t.Fatalf("want one Markdown block, for the entry with a body:\n%s", out)
	}
	if !strings.Contains(out, "## Give a talk\n\n```json\n") {
		t.Fatalf("goal metadata missing:\n%s", out)
	}
}
