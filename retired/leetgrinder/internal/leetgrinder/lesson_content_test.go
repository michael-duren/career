package leetgrinder

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestStructuredLessonEscapesContent(t *testing.T) {
	lesson := LessonContent{Day: 1, Objectives: []string{"Explain <script>alert(1)</script>"}, Sections: []LessonSection{{ID: "intro", Heading: "Intuition", Paragraphs: []string{"The <value> is stored."}, Diagram: &Scene{Width: 200, Height: 80, Description: "Array values", Nodes: []SceneNode{{ID: "a", Shape: "rect", X: 10, Y: 10, Width: 60, Height: 40, Text: "8", Role: "active"}, {ID: "b", Shape: "rect", X: 80, Y: 10, Width: 60, Height: 40, Text: "3", Role: "result"}}, Edges: []SceneEdge{{ID: "arrow", From: "a", To: "b", Directed: true, Label: "need"}}}}}}
	var out bytes.Buffer
	if err := lessonContent(lesson).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "<script>") || strings.Contains(got, "<value>") {
		t.Fatalf("unescaped lesson: %s", got)
	}
	for _, want := range []string{"Explain &lt;script&gt;", "Intuition", "The &lt;value&gt;", "Array values", "<svg", "<line", "need"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestLessonValidationRejectsBrokenRelationships(t *testing.T) {
	base := LessonContent{Day: 1, Sections: []LessonSection{{ID: "intro", Heading: "Intro", Paragraphs: []string{"A real lesson."}}}}
	if err := ValidateLessonContent(base); err != nil {
		t.Fatal(err)
	}
	bad := base
	bad.Prerequisites = []int{1}
	if err := ValidateLessonContent(bad); err == nil {
		t.Fatal("self prerequisite accepted")
	}
	bad = base
	bad.Sections = append(append([]LessonSection{}, base.Sections...), base.Sections[0])
	if err := ValidateLessonContent(bad); err == nil {
		t.Fatal("duplicate section accepted")
	}
	bad = base
	bad.Sections = []LessonSection{{ID: "intro", Heading: "Diagram", Diagram: &Scene{Width: 20, Height: 20, Edges: []SceneEdge{{From: "missing", To: "other"}}}}}
	if err := ValidateLessonContent(bad); err == nil {
		t.Fatal("invalid scene accepted")
	}
}

func TestOneReferencesFooterNoReadingPath(t *testing.T) {
	for _, day := range []int{1, 78} {
		page := Curriculum()[0].Days[0]
		for _, week := range Curriculum() {
			for _, d := range week.Days {
				if d.Number == day {
					page = d
				}
			}
		}
		var out bytes.Buffer
		if err := Lesson(page.Lesson).Render(context.Background(), &out); err != nil {
			t.Fatal(err)
		}
		if got := strings.Count(out.String(), `class="lesson-references"`); got != 1 {
			t.Errorf("day %d references count %d", day, got)
		}
	}
}
