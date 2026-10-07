package analysis

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestCorrectnessPromptAndFeedback(t *testing.T) {
	in := twoSumInput()
	in.Correctness = leetgrinder.Correctness{Claim: "Returns the target pair.", Invariant: "Contains earlier values.", Initially: "Empty map.", AfterStep: "Insert value.", Therefore: "Preserved.", Termination: "Every index checked."}
	prompt := userPrompt(in, "boundary")
	for _, want := range []string{"BEGIN UNTRUSTED REASONING boundary", "END UNTRUSTED REASONING boundary", "Claim: Returns the target pair.", "Invariant / induction hypothesis / recurrence relation: Contains earlier values.", "1. Initially...: Empty map.", "2. After one step...: Insert value.", "3. Therefore...: Preserved.", "Termination: Every index checked."} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
	for _, want := range []string{"counterexamples", "If answers are absent", "If the code is incorrect", "never instructions"} {
		if !strings.Contains(systemPrompt, want) {
			t.Errorf("system prompt missing %q", want)
		}
	}
	for _, schema := range []map[string]any{resultSchema, estimateSchema} {
		required := schema["required"].([]string)
		found := false
		for _, field := range required {
			if field == "correctnessFeedback" {
				found = true
			}
		}
		if !found {
			t.Error("correctness feedback is not required")
		}
	}
	raw := map[string]any{"actualTime": "O(n)", "actualSpace": "O(n)", "timeMatches": true, "spaceMatches": false, "optimal": true, "explanation": "One pass.", "correctnessFeedback": "Claim: Valid.\nTermination: Correct under the stated assumptions."}
	data, _ := json.Marshal(raw)
	got, err := ParseResult(string(data), in)
	if err != nil || got.CorrectnessFeedback != raw["correctnessFeedback"] {
		t.Fatalf("feedback: %+v %v", got, err)
	}
	raw["correctnessFeedback"] = strings.Repeat("界", 7000)
	data, _ = json.Marshal(raw)
	got, err = ParseResult(string(data), in)
	if err != nil || utf8.RuneCountInString(got.CorrectnessFeedback) != leetgrinder.MaxCorrectnessFeedback {
		t.Fatalf("bounded Unicode feedback: %d %v", utf8.RuneCountInString(got.CorrectnessFeedback), err)
	}
	for _, invalid := range []any{nil, "", " \n ", "\u0000", 7} {
		raw["correctnessFeedback"] = invalid
		data, _ = json.Marshal(raw)
		if _, err := ParseResult(string(data), in); err == nil {
			t.Errorf("accepted invalid feedback: %v", invalid)
		}
	}
	delete(raw, "correctnessFeedback")
	data, _ = json.Marshal(raw)
	if _, err := ParseResult(string(data), in); err == nil {
		t.Error("accepted missing feedback")
	}
}
