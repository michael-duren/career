package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPreviewRendersRealDayAndMixedDisclosure(t *testing.T) {
	h := previewHandler()
	for _, tc := range []struct{ path, want string }{{"/day/1", "Worked example: one-pass complement lookup"}, {"/day/84", "Review after your attempts"}} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), tc.want) {
			t.Errorf("%s: %d, missing %q", tc.path, w.Code, tc.want)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/day/85", nil))
	if w.Code != http.StatusNotFound {
		t.Fatal(w.Code)
	}
}
