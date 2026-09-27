package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/michael-duren/career-strategy/internal/leetgrinder"
)

func TestLeetgrinderFormComplexity(t *testing.T) {
	_, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	path := "/leetgrinder/problem/two-sum/attempts"
	// Solved without complexity: rejected, and the draft is kept.
	draft := url.Values{"id": {uuid.NewString()}, "outcome": {"solved"}, "minutes": {"20"}, "notes": {"keep me"}, "timeComplexity": {"O(n)"}, "spaceComplexity": {"other"}, "spaceComplexityOther": {""}}
	w := request("POST", path, draft)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "required for solved and struggled") || !strings.Contains(w.Body.String(), "keep me") || !strings.Contains(w.Body.String(), `<option value="O(n)" selected>`) {
		t.Fatalf("missing complexity: %d %s", w.Code, w.Body.String())
	}
	draft.Set("spaceComplexityOther", "linear-ish")
	w = request("POST", path, draft)
	if w.Code != 400 || !strings.Contains(w.Body.String(), "big-O notation") || !strings.Contains(w.Body.String(), `value="linear-ish"`) {
		t.Fatalf("bad notation: %d %s", w.Code, w.Body.String())
	}
	// "Other" text is normalised; without JavaScript it also counts with
	// nothing selected.
	draft.Set("spaceComplexityOther", " O(nlogn) ")
	if w = request("POST", path, draft); w.Code != 303 {
		t.Fatalf("save: %d %s", w.Code, w.Body.String())
	}
	noScript := url.Values{"id": {uuid.NewString()}, "outcome": {"struggled"}, "minutes": {"30"}, "timeComplexity": {""}, "timeComplexityOther": {"O(V + E)"}, "spaceComplexity": {"O(1)"}, "spaceComplexityOther": {"ignored"}}
	if w = request("POST", path, noScript); w.Code != 303 {
		t.Fatalf("no-script save: %d %s", w.Code, w.Body.String())
	}
	// Unfinished attempts may leave complexity out.
	if w = request("POST", path, url.Values{"id": {uuid.NewString()}, "outcome": {"unfinished"}, "minutes": {"25"}}); w.Code != 303 {
		t.Fatalf("unfinished: %d %s", w.Code, w.Body.String())
	}
	state, err := db.LeetgrinderState(ctx)
	if err != nil || len(state.Attempts) != 3 {
		t.Fatalf("state: %+v %v", state, err)
	}
	got := map[string]leetgrinder.Attempt{}
	for _, a := range state.Attempts {
		got[a.Outcome] = a
	}
	if a := got["solved"]; a.TimeComplexity != "O(n)" || a.SpaceComplexity != "O(n log n)" || a.Code != "" {
		t.Fatalf("solved: %+v", a)
	}
	if a := got["struggled"]; a.TimeComplexity != "O(V + E)" || a.SpaceComplexity != "O(1)" {
		t.Fatalf("struggled: %+v", a)
	}
	// Correcting a solved attempt needs both values too.
	solved := got["solved"]
	correction := url.Values{"id": {solved.ID}, "revision": {solved.Revision}, "outcome": {"solved"}, "minutes": {"21"}, "timeComplexity": {"O(n)"}}
	if w = request("POST", path, correction); w.Code != 400 || !strings.Contains(w.Body.String(), "Save correction") {
		t.Fatalf("correction without space: %d", w.Code)
	}
	correction.Set("spaceComplexity", "O(1)")
	if w = request("POST", path, correction); w.Code != 303 {
		t.Fatalf("correction: %d %s", w.Code, w.Body.String())
	}
	w = request("GET", "/leetgrinder/problem/two-sum", nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Time <strong>O(V + E)</strong>") || !strings.Contains(w.Body.String(), "Space <strong>O(1)</strong>") {
		t.Fatalf("history: %d", w.Code)
	}
}

func TestLeetgrinderAPIComplexityAndCode(t *testing.T) {
	s, db, request := leetgrinderTestServer(t)
	ctx := context.Background()
	handler := s.RegisterRoutes()
	_, plain, err := db.CreateLeetgrinderToken(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	post := func(body map[string]any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", "/api/leetgrinder/attempts", strings.NewReader(string(b)))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer "+plain)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	base := func() map[string]any {
		return map[string]any{"id": uuid.NewString(), "problemSlug": "two-sum", "outcome": "solved", "minutes": 12, "assisted": false, "notes": "", "timeComplexity": "O(n)", "spaceComplexity": "O(n)", "code": "def f():\n    return '<b>'\n", "codeLanguage": "python3"}
	}
	for name, test := range map[string]struct {
		change func(map[string]any)
		want   int
	}{
		"missing time":     {func(m map[string]any) { delete(m, "timeComplexity") }, 422},
		"missing space":    {func(m map[string]any) { m["outcome"], m["spaceComplexity"] = "struggled", "" }, 422},
		"bad notation":     {func(m map[string]any) { m["timeComplexity"] = "n^2" }, 422},
		"too long":         {func(m map[string]any) { m["timeComplexity"] = "O(" + strings.Repeat("n", 39) + ")" }, 422},
		"code too large":   {func(m map[string]any) { m["code"] = strings.Repeat("x", leetgrinder.MaxCodeBytes+1) }, 413},
		"no language":      {func(m map[string]any) { m["codeLanguage"] = "" }, 400},
		"bad language":     {func(m map[string]any) { m["codeLanguage"] = "<script>" }, 400},
		"body over limit":  {func(m map[string]any) { m["code"] = strings.Repeat("\x01", leetgrinder.MaxCodeBytes) }, 413},
		"unfinished plain": {func(m map[string]any) { m["outcome"], m["timeComplexity"], m["spaceComplexity"] = "unfinished", "", "" }, 200},
	} {
		body := base()
		test.change(body)
		if w := post(body); w.Code != test.want {
			t.Errorf("%s: %d %s", name, w.Code, w.Body.String())
		}
	}
	body := base()
	body["code"] = strings.Repeat("é", leetgrinder.MaxCodeBytes/2)
	body["timeComplexity"] = "O(n^2)"
	for i := 0; i < 2; i++ {
		if w := post(body); w.Code != 200 {
			t.Fatalf("max code %d: %d %s", i, w.Code, w.Body.String())
		}
	}
	body["codeLanguage"] = "python"
	if w := post(body); w.Code != 409 {
		t.Fatalf("retry with other language: %d", w.Code)
	}
	r := httptest.NewRequest("GET", "/api/leetgrinder/problem/two-sum", nil)
	r.Header.Set("Authorization", "Bearer "+plain)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	var info leetgrinderAPIProblem
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &info) != nil || info.Latest == nil || info.Latest.Code != "" || info.Latest.TimeComplexity != "O(n²)" {
		t.Fatalf("latest attempt: %d %.300s", w.Code, w.Body.String())
	}
	// The export carries all four fields.
	w = request("GET", "/leetgrinder/export", nil)
	var exported leetgrinder.State
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &exported) != nil {
		t.Fatalf("export: %d", w.Code)
	}
	found := false
	for _, a := range exported.Attempts {
		if a.Code == body["code"] && a.CodeLanguage == "python3" && a.TimeComplexity == "O(n²)" && a.SpaceComplexity == "O(n)" {
			found = true
		}
	}
	for _, key := range []string{`"timeComplexity"`, `"spaceComplexity"`, `"code"`, `"codeLanguage"`} {
		if !strings.Contains(w.Body.String(), key) {
			t.Errorf("export missing %s", key)
		}
	}
	if !found {
		t.Fatal("export lost captured code")
	}
	w = request("GET", "/leetgrinder/problem/two-sum", nil)
	if !strings.Contains(w.Body.String(), "Submitted code (Python3, 64.0 KB)") {
		t.Fatalf("history does not show code: %d", w.Code)
	}
}
