package leetgrinder

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestCorrectnessValidationAndRendering(t *testing.T) {
	c := Correctness{Claim: "Returns <pair>\nwith target sum.", Invariant: "Seen contains earlier elements.", Initially: "Empty map.", AfterStep: "Insert one element.", Therefore: "Preserved.", Termination: "Every index was visited."}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{strings.Repeat("x", 2001), "nul\x00", string([]byte{0xff})} {
		if err := (Correctness{Claim: value}).Validate(); err == nil {
			t.Errorf("accepted invalid value: %q", value[:min(10, len(value))])
		}
	}
	if err := (Correctness{Claim: strings.Repeat("😀", 2000)}).Validate(); err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := correctnessFields(c).Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	for _, field := range c.Fields() {
		if !strings.Contains(b.String(), `name="`+field.Name+`"`) || !strings.Contains(b.String(), `maxlength="2000"`) {
			t.Errorf("missing control: %s", field.Name)
		}
	}
	if !strings.Contains(b.String(), "Returns &lt;pair&gt;\nwith target sum.") {
		t.Fatalf("form escaped or multiline value lost: %s", b.String())
	}
	b.Reset()
	if err := correctnessAnswers(c).Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "Returns &lt;pair&gt;\nwith target sum.") || !strings.Contains(b.String(), "Every index was visited.") {
		t.Fatalf("history: %s", b.String())
	}
	data, err := json.Marshal(Attempt{Correctness: c})
	if err != nil {
		t.Fatal(err)
	}
	var roundtrip Attempt
	if err := json.Unmarshal(data, &roundtrip); err != nil || roundtrip.Correctness != c {
		t.Fatalf("export round trip: %+v %v", roundtrip, err)
	}
}

func TestCorrectnessSections(t *testing.T) {
	tests := []struct {
		name, text string
		want       []FeedbackSection
	}{
		{"one per line", "Claim: Returns the pair.\n\nInvariant: Seen holds earlier values.\n\nWhy it holds:\n1. Initially: Empty.\n2. After one step: Insert.\n3. Therefore: Kept.\n\nTermination: Every index visited.", []FeedbackSection{
			{Label: "Claim", Body: "Returns the pair."}, {Label: "Invariant", Body: "Seen holds earlier values."}, {Label: "Why it holds"},
			{Label: "1. Initially", Body: "Empty.", Step: true}, {Label: "2. After one step", Body: "Insert.", Step: true}, {Label: "3. Therefore", Body: "Kept.", Step: true},
			{Label: "Termination", Body: "Every index visited."},
		}},
		{"bunched with long labels", "Overall sound. Claim: Right. Invariant / induction hypothesis / recurrence relation: the claim holds. Why it holds (1. Initially, 2. After one step, 3. Therefore) 1. Initially, empty. Termination: Done.", []FeedbackSection{
			{Body: "Overall sound."}, {Label: "Claim", Body: "Right."}, {Label: "Invariant / induction hypothesis / recurrence relation", Body: "the claim holds."},
			{Label: "Why it holds"}, {Label: "1. Initially", Body: "empty.", Step: true}, {Label: "Termination", Body: "Done."},
		}},
		{"unlabeled", "Termination is fine and the Claim stands.", nil},
		{"labels in prose", "Termination, however, needs n > 0.\nClaim, as stated, holds.\nWhy it holds is subtle.", nil},
		{"heading alone", "Some preamble.\nWhy it holds\nTermination: Done.", []FeedbackSection{{Body: "Some preamble."}, {Label: "Why it holds"}, {Label: "Termination", Body: "Done."}}},
	}
	for _, tt := range tests {
		got := CorrectnessSections(tt.text)
		if len(got) != len(tt.want) {
			t.Fatalf("%s: got %#v", tt.name, got)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("%s: section %d = %#v, want %#v", tt.name, i, got[i], tt.want[i])
			}
		}
	}
}

func TestCorrectnessFeedbackRendering(t *testing.T) {
	var b bytes.Buffer
	if err := correctnessFeedback("Claim: <Pair>.\nWhy it holds:\n1. Initially: Empty.").Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<h6>Claim</h6>", "&lt;Pair&gt;.", "<h6>Why it holds</h6>", `class="feedback-section feedback-step"`, "<h6>1. Initially</h6>"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("missing %q in %s", want, b.String())
		}
	}
	b.Reset()
	if err := correctnessFeedback("Looks right.").Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	if got := b.String(); got != `<p class="notes">Looks right.</p>` {
		t.Errorf("fallback = %s", got)
	}
}
