package leetgrinder

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestSessionRendersLessonAndPracticeForms(t *testing.T) {
	day, ok := FindDay(1)
	if !ok {
		t.Fatal("missing first day")
	}
	ids := map[string]string{}
	for _, p := range day.Core {
		ids[p.Slug] = uuid.NewString()
	}
	for _, p := range day.Optional {
		ids[p.Slug] = uuid.NewString()
	}
	var body bytes.Buffer
	if err := Session(DayPage{Day: day, IDs: ids}).Render(context.Background(), &body); err != nil {
		t.Fatal(err)
	}
	html := body.String()
	for _, want := range []string{"Before you start", "/leetgrinder/day/1/complete", `name="minutes"`, `name="assisted"`, `name="notes"`, `name="id"`, `name="revision"`} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, p := range day.Core {
		if !strings.Contains(html, p.URL()) || !strings.Contains(html, ids[p.Slug]) {
			t.Errorf("missing assignment/form for %s", p.Slug)
		}
	}
	var lesson bytes.Buffer
	if err := Lesson(day.Lesson).Render(context.Background(), &lesson); err != nil {
		t.Fatal(err)
	}
	if lesson.Len() < 200 || !strings.Contains(html, lesson.String()) {
		t.Fatal("lesson not rendered inside session")
	}
	for _, unexpected := range []string{"@Lesson", "@attemptFields", "if form.", "if reading."} {
		if strings.Contains(html, unexpected) {
			t.Errorf("template syntax leaked: %s", unexpected)
		}
	}
}
func TestAttemptDraftEscapesAndRetainsInput(t *testing.T) {
	problem := Problem{ID: 1, Slug: "two-sum", Title: "Two Sum"}
	form := AttemptForm{ID: uuid.NewString(), Revision: uuid.NewString(), Outcome: "struggled", Minutes: "26", Assisted: true, Notes: "<script>alert(1)</script>", Error: "Please retry"}
	var out bytes.Buffer
	if err := ProblemHistory(problem, State{}, form, AnalysisAvailability{}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, want := range []string{`value="26"`, `value="struggled" selected`, `checked`, `&lt;script&gt;`, `Please retry`, form.ID, form.Revision, "Save correction"} {
		if !strings.Contains(html, want) {
			t.Errorf("draft missing %q", want)
		}
	}
	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Fatal("unescaped draft")
	}
}

func TestMockKeepsPatternGuidanceBehindReview(t *testing.T) {
	day, _ := FindDay(84)
	var out bytes.Buffer
	if err := Session(DayPage{Day: day, IDs: map[string]string{}}).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `<details class="review-guidance"><summary>Review after your attempts</summary>`) {
		t.Fatal("mock exposes pattern guidance before attempting")
	}
	body := out.String()
	start := strings.Index(body, `<details class="review-guidance">`)
	end := start + strings.Index(body[start:], "</details>")
	if strings.Contains(body, "Reading path") {
		t.Fatal("duplicate reading panel remains")
	}
	if references := strings.Index(body, `<footer class="lesson-references"`); references < start || references > end {
		t.Fatal("mixed-practice references must stay inside the review disclosure")
	}
}
