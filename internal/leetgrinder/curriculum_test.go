package leetgrinder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestEveryLessonHasDistinctSourcesAndCitedReadings(t *testing.T) {
	documents := map[string]int{}
	for _, week := range Curriculum() {
		for _, day := range week.Days {
			t.Run(day.Lesson, func(t *testing.T) {
				if len(day.Readings) < 2 {
					t.Fatal("lesson needs an introduction and a deeper reading")
				}
				intro := day.Readings[0]
				if intro.Kind != "introduction" || intro.Minutes > 10 {
					t.Error("first reading must be an introduction of at most ten minutes")
				}
				var b bytes.Buffer
				if err := Lesson(day.Lesson).Render(context.Background(), &b); err != nil {
					t.Fatal(err)
				}
				body := b.String()
				start := strings.Index(body, ">References</h3>")
				if start < strings.Index(body, "Before you finish") || start < 0 {
					t.Fatal("references must follow the lesson content")
				}
				papers := 0
				for _, reading := range day.Readings {
					document, err := url.Parse(reading.URL)
					if err != nil {
						t.Fatal(err)
					}
					document.Fragment = ""
					if document.Scheme != "https" || document.Host == "" {
						t.Errorf("reference must use an absolute HTTPS URL: %s", reading.URL)
					}
					if earlier, exists := documents[document.String()]; exists {
						t.Errorf("reading document repeats day %d: %s", earlier, reading.URL)
					}
					documents[document.String()] = day.Number
					if reading.Kind == "paper" {
						papers++
						if !reading.Optional {
							t.Error("papers must be optional deeper reading")
						}
					}
					if reading.Supports == "" || !strings.Contains(body[start:], html.EscapeString(reading.Supports)) {
						t.Errorf("reference %q must explain which lesson concepts it supports", reading.Title)
					}
					if !strings.Contains(body[start:], `href="`+html.EscapeString(reading.URL)+`"`) {
						t.Errorf("references omit %q", reading.Title)
					}
				}
				if papers == 0 {
					t.Error("lesson needs a research paper or technical report")
				}
			})
		}
	}
}

func TestCurriculumAssignments(t *testing.T) {
	weeks := Curriculum()
	if len(weeks) != 12 {
		t.Fatalf("weeks = %d, want 12", len(weeks))
	}
	seen := map[string]bool{}
	ids := map[int]bool{}
	dayNumber := 0
	for wi, week := range weeks {
		if week.Number != wi+1 || len(week.Days) != 7 {
			t.Fatalf("invalid week %#v", week)
		}
		optional := 0
		for _, day := range week.Days {
			dayNumber++
			if day.Number != dayNumber || len(day.Core) != 3 {
				t.Fatalf("invalid day %d", day.Number)
			}
			if day.Title == "" || day.Lesson == "" || len(day.Readings) == 0 {
				t.Fatalf("day %d lacks instruction", day.Number)
			}
			minutes := 0
			for _, r := range day.Readings {
				if !strings.HasPrefix(r.URL, "https://") || r.Title == "" || r.Guidance == "" || r.Minutes <= 0 {
					t.Fatalf("day %d invalid reading: %#v", day.Number, r)
				}
				minutes += r.Minutes
			}
			if minutes > 30 {
				t.Fatalf("day %d reading exceeds budget: %d", day.Number, minutes)
			}
			optional += len(day.Optional)
			for _, p := range append(append([]Problem{}, day.Core...), day.Optional...) {
				if p.ID <= 0 || p.Slug == "" || p.Title == "" || seen[p.Slug] || ids[p.ID] {
					t.Fatalf("invalid or duplicate problem %#v", p)
				}
				if p.Difficulty != "Easy" && p.Difficulty != "Medium" && p.Difficulty != "Hard" {
					t.Fatalf("invalid difficulty %#v", p)
				}
				seen[p.Slug], ids[p.ID] = true, true
				got, ok := FindProblem(p.Slug)
				if !ok || got != p {
					t.Fatalf("lookup failed: %#v", p)
				}
			}
			got, ok := FindDay(day.Number)
			if !ok || got.Title != day.Title {
				t.Fatalf("day lookup failed: %d", day.Number)
			}
		}
		if optional != 4 {
			t.Fatalf("week %d optional = %d, want 4", week.Number, optional)
		}
	}
	if dayNumber != 84 || len(seen) != 300 {
		t.Fatalf("days=%d problems=%d", dayNumber, len(seen))
	}
	for _, n := range []int{-1, 0, 85} {
		if _, ok := FindDay(n); ok {
			t.Errorf("FindDay(%d) found invalid day", n)
		}
	}
	if _, ok := FindProblem("missing"); ok {
		t.Error("found unknown problem")
	}
}

func TestEveryLessonRendersInstruction(t *testing.T) {
	seen := map[string]bool{}
	for _, week := range Curriculum() {
		for _, day := range week.Days {
			var b bytes.Buffer
			if err := Lesson(day.Lesson).Render(context.Background(), &b); err != nil {
				t.Fatal(err)
			}
			html := b.String()
			for _, marker := range []string{"Worked example", "Complexity and cautions", "Before you finish"} {
				if !strings.Contains(html, marker) {
					t.Errorf("day %d lacks %s", day.Number, marker)
				}
			}
			if len(strings.Fields(html)) < 100 {
				t.Errorf("day %d lesson is too thin (%d words)", day.Number, len(strings.Fields(html)))
			}
			if seen[html] {
				t.Errorf("day %d repeats another lesson", day.Number)
			}
			seen[html] = true
			if strings.Contains(html, "TODO") || strings.Contains(html, "placeholder") {
				t.Errorf("unfinished lesson %d", day.Number)
			}
		}
	}
}

// TestReadingsMatchSourceManifests guards against readings.go drifting from
// the sources_*.json manifests it is generated from: a hand edit to one, or
// forgetting to rerun the generator after editing the other, fails this test
// even though every other test only exercises the compiled lessonReadings.
func TestReadingsMatchSourceManifests(t *testing.T) {
	type manifestDay struct {
		Day      int       `json:"day"`
		Readings []Reading `json:"readings"`
	}
	var manifest []manifestDay
	paths, err := filepath.Glob("sources_*.json")
	if err != nil || len(paths) == 0 {
		t.Fatalf("no source manifests found: %v", err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var rows []manifestDay
		if err := json.Unmarshal(data, &rows); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		manifest = append(manifest, rows...)
	}
	sort.Slice(manifest, func(i, j int) bool { return manifest[i].Day < manifest[j].Day })

	validKinds := map[string]bool{"introduction": true, "paper": true, "reference": true}
	for day := 1; day <= 84; day++ {
		if manifest[day-1].Day != day {
			t.Fatalf("manifests are missing or duplicate day %d", day)
		}
		for _, reading := range manifest[day-1].Readings {
			if !validKinds[reading.Kind] {
				t.Errorf("day %d: reading %q has unknown kind %q", day, reading.Title, reading.Kind)
			}
		}
		if got, want := lessonReadings(day), manifest[day-1].Readings; !reflect.DeepEqual(got, want) {
			t.Errorf("day %d: readings.go does not match its source manifest; rerun scripts/leetgrinder/integrate_readings.py\n got:  %+v\n want: %+v", day, got, want)
		}
	}
}

func TestKnownCurriculumEndpoints(t *testing.T) {
	day, ok := FindDay(1)
	if !ok || day.Core[0].Slug != "two-sum" {
		t.Fatal("day one must start with Two Sum")
	}
	day, ok = FindDay(84)
	if !ok || !strings.Contains(day.Title, "Mock") {
		t.Fatalf("final session is not a mock: %s", fmt.Sprint(day))
	}
}
