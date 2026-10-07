package leetgrinder

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const MaxCorrectnessAnswer = 2000

var ErrCorrectnessInvalid = errors.New("each correctness answer must be valid text of 2,000 characters or fewer")

type Correctness struct {
	Claim       string `json:"claim"`
	Invariant   string `json:"invariant"`
	Initially   string `json:"initially"`
	AfterStep   string `json:"afterStep"`
	Therefore   string `json:"therefore"`
	Termination string `json:"termination"`
}

type CorrectnessField struct{ Name, Label, Hint, Value string }

func (c Correctness) Fields() []CorrectnessField {
	return []CorrectnessField{
		{"claim", "Claim", "What exactly does my algorithm guarantee?", c.Claim},
		{"invariant", "Invariant / induction hypothesis / recurrence relation", "What remains true throughout the algorithm?", c.Invariant},
		{"initially", "1. Initially...", "Why does it hold initially?", c.Initially},
		{"afterStep", "2. After one step...", "Why does one step preserve it?", c.AfterStep},
		{"therefore", "3. Therefore...", "What follows from initialization and preservation?", c.Therefore},
		{"termination", "Termination", "When the algorithm ends, why does this imply the answer is correct?", c.Termination},
	}
}

func (c Correctness) Validate() error {
	for _, f := range c.Fields() {
		if !utf8.ValidString(f.Value) || strings.ContainsRune(f.Value, 0) || utf8.RuneCountInString(f.Value) > MaxCorrectnessAnswer {
			return ErrCorrectnessInvalid
		}
	}
	return nil
}
