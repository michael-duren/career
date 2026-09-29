package leetgrinder

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

type LessonContent struct {
	Day           int             `json:"day"`
	Objectives    []string        `json:"objectives"`
	Prerequisites []int           `json:"prerequisites"`
	Sections      []LessonSection `json:"sections"`
}
type LessonSection struct {
	ID         string           `json:"id"`
	Heading    string           `json:"heading"`
	Paragraphs []string         `json:"paragraphs"`
	Bullets    []string         `json:"bullets"`
	Diagram    *Scene           `json:"diagram"`
	ExampleIDs []string         `json:"exampleIds"`
	Checks     []KnowledgeCheck `json:"checks"`
	Optional   bool             `json:"optional"`
}
type KnowledgeCheck struct {
	Prompt string `json:"prompt"`
	Answer string `json:"answer"`
}
type CodeVariant struct {
	Language       string `json:"language"`
	File           string `json:"file"`
	AlgorithmStart int    `json:"algorithmStart"`
	AlgorithmEnd   int    `json:"algorithmEnd"`
}
type ExampleContent struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Invariant string        `json:"invariant"`
	Variants  []CodeVariant `json:"variants"`
	Cases     []TraceCase   `json:"cases"`
}
type TraceCase struct {
	ID             string       `json:"id"`
	Label          string       `json:"label"`
	Input          string       `json:"input"`
	ExpectedOutput string       `json:"expectedOutput"`
	Frames         []TraceFrame `json:"frames"`
}
type TraceFrame struct {
	Event       string           `json:"event"`
	Lines       map[string][]int `json:"lines"`
	Explanation string           `json:"explanation"`
	Variables   []Variable       `json:"variables"`
	Scene       Scene            `json:"scene"`
}
type Variable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type Scene struct {
	Width       int          `json:"width"`
	Height      int          `json:"height"`
	Description string       `json:"description"`
	Nodes       []SceneNode  `json:"nodes"`
	Edges       []SceneEdge  `json:"edges"`
	Labels      []SceneLabel `json:"labels"`
}
type SceneNode struct {
	ID     string `json:"id"`
	Shape  string `json:"shape"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Text   string `json:"text"`
	Role   string `json:"role"`
}
type SceneEdge struct {
	ID       string `json:"id"`
	From     string `json:"from"`
	To       string `json:"to"`
	Label    string `json:"label"`
	Directed bool   `json:"directed"`
	Role     string `json:"role"`
}
type SceneLabel struct {
	ID   string `json:"id"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	Text string `json:"text"`
}

//go:embed lessons examples
var lessonFiles embed.FS
var lessons = map[string]LessonContent{}
var examples = map[string]ExampleContent{}

func init() {
	for _, pattern := range []string{"lessons/day-*.json", "examples/day-*/*/example.json"} {
		paths, err := fs.Glob(lessonFiles, pattern)
		if err != nil {
			panic(err)
		}
		for _, path := range paths {
			file, err := lessonFiles.Open(path)
			if err != nil {
				panic(err)
			}
			decoder := json.NewDecoder(file)
			decoder.DisallowUnknownFields()
			if strings.HasPrefix(path, "lessons/") {
				var lesson LessonContent
				err = decoder.Decode(&lesson)
				if err == nil {
					err = ValidateLessonContent(lesson)
				}
				if err == nil {
					lessons[fmt.Sprintf("day-%02d", lesson.Day)] = lesson
				}
			} else {
				var example ExampleContent
				err = decoder.Decode(&example)
				if err == nil {
					err = ValidateExampleContent(example)
				}
				if err == nil {
					err = validateEmbeddedExample(path, example)
				}
				if err == nil {
					examples[example.ID] = example
				}
			}
			file.Close()
			if err != nil {
				panic(fmt.Errorf("%s: %w", path, err))
			}
		}
	}
	for key, lesson := range lessons {
		for _, section := range lesson.Sections {
			for _, id := range section.ExampleIDs {
				if _, ok := examples[id]; !ok {
					panic(fmt.Errorf("lesson %s references missing example %s", key, id))
				}
			}
		}
	}
}
func LookupLessonContent(key string) (LessonContent, bool)  { v, ok := lessons[key]; return v, ok }
func LookupExampleContent(id string) (ExampleContent, bool) { v, ok := examples[id]; return v, ok }
func ExampleSource(id, language string) (string, error) {
	example, ok := examples[id]
	if !ok {
		return "", fmt.Errorf("unknown example %q", id)
	}
	for _, v := range example.Variants {
		if v.Language == language {
			if filepath.Base(v.File) != v.File || v.File == "." || v.File == ".." {
				return "", fmt.Errorf("invalid file %q", v.File)
			}
			day := strings.SplitN(id, "-", 3)
			if len(day) < 3 {
				return "", fmt.Errorf("invalid example ID %q", id)
			}
			raw, err := lessonFiles.ReadFile("examples/" + day[0] + "-" + day[1] + "/" + day[2] + "/" + v.File)
			return string(raw), err
		}
	}
	return "", fmt.Errorf("example %s missing language %s", id, language)
}
func ValidateLessonContent(lesson LessonContent) error {
	if lesson.Day < 1 || lesson.Day > 84 {
		return fmt.Errorf("invalid day %d", lesson.Day)
	}
	if len(lesson.Sections) == 0 {
		return fmt.Errorf("day %d has no sections", lesson.Day)
	}
	for _, n := range lesson.Prerequisites {
		if n < 1 || n >= lesson.Day {
			return fmt.Errorf("day %d invalid prerequisite %d", lesson.Day, n)
		}
	}
	seen := map[string]bool{}
	for _, s := range lesson.Sections {
		if s.ID == "" || seen[s.ID] || s.Heading == "" {
			return fmt.Errorf("day %d invalid section %q", lesson.Day, s.ID)
		}
		seen[s.ID] = true
		if s.Diagram != nil {
			if err := validateScene(*s.Diagram); err != nil {
				return fmt.Errorf("day %d section %s: %w", lesson.Day, s.ID, err)
			}
		}
	}
	return nil
}
func validateScene(s Scene) error {
	if s.Width <= 0 || s.Height <= 0 || s.Description == "" {
		return fmt.Errorf("invalid scene dimensions or description")
	}
	ids := map[string]bool{}
	for _, n := range s.Nodes {
		if n.ID == "" || ids[n.ID] || n.X < 0 || n.Y < 0 || n.Width <= 0 || n.Height <= 0 || n.X+n.Width > s.Width || n.Y+n.Height > s.Height || (n.Shape != "rect" && n.Shape != "circle") || (n.Shape == "circle" && n.Width != n.Height) {
			return fmt.Errorf("invalid node %q", n.ID)
		}
		ids[n.ID] = true
	}
	used := map[string]bool{}
	for id := range ids {
		used[id] = true
	}
	for _, e := range s.Edges {
		if e.ID == "" || used[e.ID] || !ids[e.From] || !ids[e.To] {
			return fmt.Errorf("invalid edge %q", e.ID)
		}
		used[e.ID] = true
	}
	for _, l := range s.Labels {
		if l.ID == "" || used[l.ID] || l.X < 0 || l.Y < 0 || l.X > s.Width || l.Y > s.Height {
			return fmt.Errorf("invalid label %q", l.ID)
		}
		used[l.ID] = true
	}
	return nil
}
func LessonTitle(day int) string {
	for _, week := range Curriculum() {
		for _, d := range week.Days {
			if d.Number == day {
				return d.Title
			}
		}
	}
	return "Lesson"
}

func ValidateExampleContent(e ExampleContent) error {
	if !strings.HasPrefix(e.ID, "day-") || e.Title == "" || e.Invariant == "" {
		return fmt.Errorf("example %q missing identity, title or invariant", e.ID)
	}
	langs := map[string]CodeVariant{}
	for _, v := range e.Variants {
		if v.Language != "cpp" && v.Language != "python" && v.Language != "java" && v.Language != "go" {
			return fmt.Errorf("%s invalid language %q", e.ID, v.Language)
		}
		if _, ok := langs[v.Language]; ok {
			return fmt.Errorf("%s duplicate language %s", e.ID, v.Language)
		}
		if filepath.Base(v.File) != v.File || v.File == "." || v.File == ".." || v.AlgorithmStart < 1 || v.AlgorithmEnd < v.AlgorithmStart {
			return fmt.Errorf("%s invalid source range/file for %s", e.ID, v.Language)
		}
		langs[v.Language] = v
	}
	if len(langs) != 4 {
		return fmt.Errorf("%s needs C++, Python, Java and Go", e.ID)
	}
	if len(e.Cases) == 0 {
		return fmt.Errorf("%s has no cases", e.ID)
	}
	cases := map[string]bool{}
	for _, c := range e.Cases {
		if c.ID == "" || cases[c.ID] || len(c.Frames) < 2 {
			return fmt.Errorf("%s invalid case %s", e.ID, c.ID)
		}
		cases[c.ID] = true
		for i, f := range c.Frames {
			if f.Event == "" || f.Explanation == "" {
				return fmt.Errorf("%s/%s frame %d missing event or explanation", e.ID, c.ID, i)
			}
			if err := validateScene(f.Scene); err != nil {
				return fmt.Errorf("%s/%s frame %d: %w", e.ID, c.ID, i, err)
			}
			for language, v := range langs {
				if len(f.Lines[language]) == 0 {
					return fmt.Errorf("%s/%s frame %d missing %s lines", e.ID, c.ID, i, language)
				}
				for _, line := range f.Lines[language] {
					if line < v.AlgorithmStart || line > v.AlgorithmEnd {
						return fmt.Errorf("%s/%s frame %d %s line %d outside algorithm", e.ID, c.ID, i, language, line)
					}
				}
			}
		}
	}
	return nil
}

type ExampleDisplay struct {
	ExampleContent
	Sources map[string]string `json:"sources"`
}

func displayExample(e ExampleContent) ExampleDisplay {
	d := ExampleDisplay{ExampleContent: e, Sources: map[string]string{}}
	for _, v := range e.Variants {
		s, err := ExampleSource(e.ID, v.Language)
		if err == nil {
			d.Sources[v.Language] = s
		}
	}
	return d
}
func exampleJSON(e ExampleContent) string {
	raw, err := json.Marshal(displayExample(e))
	if err != nil {
		return "{}"
	}
	return string(raw)
}
func sceneCenterX(s Scene, id string) int {
	for _, n := range s.Nodes {
		if n.ID == id {
			return n.X + n.Width/2
		}
	}
	return 0
}
func sceneCenterY(s Scene, id string) int {
	for _, n := range s.Nodes {
		if n.ID == id {
			return n.Y + n.Height/2
		}
	}
	return 0
}

func validateEmbeddedExample(path string, e ExampleContent) error {
	parts := strings.Split(path, "/")
	if len(parts) != 4 || parts[0] != "examples" || parts[3] != "example.json" || parts[1]+"-"+parts[2] != e.ID {
		return fmt.Errorf("example ID %q does not match %s", e.ID, path)
	}
	for _, v := range e.Variants {
		sourcePath := filepath.ToSlash(filepath.Join(filepath.Dir(path), v.File))
		raw, err := lessonFiles.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("%s %s source %s: %w", e.ID, v.Language, sourcePath, err)
		}
		lineCount := len(strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n"))
		if v.AlgorithmEnd > lineCount {
			return fmt.Errorf("%s %s algorithm line %d past source end %d", e.ID, v.Language, v.AlgorithmEnd, lineCount)
		}
	}
	return nil
}
