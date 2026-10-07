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
