package server

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderCorrectnessWebForm(t *testing.T) {
	_, db, request := leetgrinderTestServer(t)
	path := "/leetgrinder/problem/two-sum/attempts"
	draft := url.Values{"id": {uuid.NewString()}, "outcome": {"solved"}, "minutes": {"20"}, "claim": {"Returns <pair>\nwith target sum."}, "invariant": {"The map contains previous values."}, "initially": {"The map is empty."}, "afterStep": {"Insert this value."}, "therefore": {"Previous values remain stored."}, "termination": {"All elements were checked."}, "timeComplexity": {"O(n)"}}
	w := request("POST", path, draft)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "Returns &lt;pair&gt;") || !strings.Contains(w.Body.String(), "All elements were checked.") {
		t.Fatalf("draft: %d %s", w.Code, w.Body.String())
	}
	draft.Set("spaceComplexity", "O(n)")
	w = request("POST", path, draft)
	if w.Code != 303 {
		t.Fatalf("save: %d %s", w.Code, w.Body.String())
	}
	state, err := db.LeetgrinderState(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	a := state.Attempts[0]
	for _, f := range a.Correctness.Fields() {
		if f.Value != draft.Get(f.Name) {
			t.Errorf("%s = %q", f.Name, f.Value)
		}
	}
	w = request("GET", "/leetgrinder/problem/two-sum", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Returns &lt;pair&gt;") || strings.Contains(w.Body.String(), "Returns <pair>") {
		t.Fatalf("history: %d %s", w.Code, w.Body.String())
	}
	draft.Set("revision", a.Revision)
	draft.Set("claim", "New claim")
	w = request("POST", path, draft)
	if w.Code != 303 {
		t.Fatalf("correction: %d %s", w.Code, w.Body.String())
	}
	state, err = db.LeetgrinderState(context.Background())
	if err != nil || state.Attempts[0].Claim != "New claim" {
		t.Fatalf("correction not saved: %+v %v", state.Attempts, err)
	}
	draft.Set("claim", strings.Repeat("x", 2001))
	if w = request("POST", path, draft); w.Code != 400 {
		t.Fatalf("oversized answer: %d", w.Code)
	}
}

func TestLeetgrinderCorrectnessAPIInput(t *testing.T) {
	body := `{"id":"` + uuid.NewString() + `","problemSlug":"two-sum","outcome":"unfinished","minutes":10,"claim":"A pair","invariant":"Seen values","initially":"Empty","afterStep":"Insert","therefore":"Preserved","termination":"Done"}`
	var input leetgrinderAPIAttemptInput
	if err := json.Unmarshal([]byte(body), &input); err != nil {
		t.Fatal(err)
	}
	a, status, msg := newLeetgrinderAttempt(input, "extension")
	if status != 0 || a.Correctness != (leetgrinder.Correctness{Claim: "A pair", Invariant: "Seen values", Initially: "Empty", AfterStep: "Insert", Therefore: "Preserved", Termination: "Done"}) {
		t.Fatalf("input: %+v %d %s", a, status, msg)
	}
	input.Termination = strings.Repeat("x", 2001)
	if _, status, _ := newLeetgrinderAttempt(input, "extension"); status != 400 {
		t.Fatalf("oversized answer: %d", status)
	}
}
