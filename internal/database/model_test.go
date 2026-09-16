package database

import (
	"encoding/json"
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
