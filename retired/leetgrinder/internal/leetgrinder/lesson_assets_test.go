package leetgrinder

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestLessonPlayerModuleAvailableOnLayout(t *testing.T) {
	if !strings.Contains(LessonPlayerJS, "mountLessonExamples") {
		t.Fatal("player JavaScript missing")
	}
	var out bytes.Buffer
	if err := layout("Test", "dashboard").Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `src="/leetgrinder/lesson-player.js"`) {
		t.Fatal("layout does not load player")
	}
}
