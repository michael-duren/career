package database

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestSaveNormalization(t *testing.T) {
	var w Entity
	json.Unmarshal(fixture(t), &w)
	e := w["books"].([]any)[0].(map[string]any)
	e["title"] = "  Title  "
	e["authors"] = []any{" A ", ""}
	e["url"] = ""
	e["started"] = ""
	normalized, err := PrepareSave("book", e)
	if err != nil {
		t.Fatal(err)
	}
	if normalized["title"] != "Title" || len(normalized["authors"].([]any)) != 1 {
		t.Fatal("normalization failed")
	}
	if _, ok := normalized["url"]; ok {
		t.Fatal("empty optional URL retained")
	}
	if e["title"] != "  Title  " {
		t.Fatal("normalization mutated input")
	}
	e["authors"] = []any{strings.Repeat(" ", 201)}
	if _, err = PrepareSave("book", e); err == nil {
		t.Fatal("overlong raw author accepted")
	}
}

func TestGoalSelectedWeekdaysDefaultAndValidation(t *testing.T) {
	goal := dependencyGoal("11111111-1111-4111-8111-111111111111")
	prepared, err := PrepareSave("goal", goal)
	if err != nil {
		t.Fatal(err)
	}
	want := []any{float64(1), float64(2), float64(3), float64(4), float64(5), float64(6), float64(7)}
	if got := prepared["selectedWeekdays"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("missing weekdays defaulted to %#v, want %#v", got, want)
	}

	for _, weekdays := range []any{
		[]any{float64(0)},
		[]any{float64(8)},
		[]any{float64(1.5)},
		[]any{float64(1), float64(1)},
		[]any{"1"},
	} {
		candidate := dependencyGoal("11111111-1111-4111-8111-111111111111")
		candidate["selectedWeekdays"] = weekdays
		if _, err := PrepareSave("goal", candidate); err == nil {
			t.Fatalf("accepted invalid selectedWeekdays %#v", weekdays)
		}
	}

	goal["selectedWeekdays"] = []any{}
	if _, err := PrepareSave("goal", goal); err != nil {
		t.Fatalf("rejected an empty weekday selection: %v", err)
	}
}
