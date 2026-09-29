package leetgrinder

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestExampleValidationRejectsMissingLanguageAndBrokenLines(t *testing.T) {
	v := ExampleContent{ID: "day-01-probe", Title: "Probe", Invariant: "i is next", Variants: []CodeVariant{{Language: "python", File: "main.py", AlgorithmStart: 1, AlgorithmEnd: 3}}, Cases: []TraceCase{{ID: "normal", Label: "Normal", Frames: []TraceFrame{{Event: "start", Lines: map[string][]int{"python": {1}}, Explanation: "Start", Scene: Scene{Width: 100, Height: 50, Description: "One box", Nodes: []SceneNode{{ID: "a", Shape: "rect", X: 0, Y: 0, Width: 20, Height: 20, Text: "A"}}}}, {Event: "end", Lines: map[string][]int{"python": {2}}, Explanation: "End", Scene: Scene{Width: 100, Height: 50, Description: "Done"}}}}}}
	if err := ValidateExampleContent(v); err == nil {
		t.Fatal("missing three languages accepted")
	}
	v.Variants = []CodeVariant{{Language: "python", File: "main.py", AlgorithmStart: 1, AlgorithmEnd: 3}, {Language: "cpp", File: "main.cpp", AlgorithmStart: 1, AlgorithmEnd: 3}, {Language: "java", File: "Main.java", AlgorithmStart: 1, AlgorithmEnd: 3}, {Language: "go", File: "main.go", AlgorithmStart: 1, AlgorithmEnd: 3}}
	for i := range v.Cases[0].Frames {
		v.Cases[0].Frames[i].Lines = map[string][]int{"python": {1}, "cpp": {1}, "java": {1}, "go": {1}}
	}
	if err := ValidateExampleContent(v); err != nil {
		t.Fatal(err)
	}
	v.Cases[0].Frames[1].Lines["go"] = []int{4}
	if err := ValidateExampleContent(v); err == nil {
		t.Fatal("line outside algorithm accepted")
	}
}

func TestExampleRendererEscapesSourceAndKeepsTraceText(t *testing.T) {
	example := ExampleContent{ID: "day-01-probe", Title: "Probe", Cases: []TraceCase{{ID: "normal", Label: "Normal", Frames: []TraceFrame{{Event: "start", Explanation: "</script><img src=x onerror=alert(1)>", Scene: Scene{Width: 100, Height: 50, Description: "Test"}}}}}}
	var out bytes.Buffer
	if err := lessonExample(example).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if strings.Contains(got, "<img src=x") || strings.Contains(got, "</script><img") {
		t.Fatal("unsafe example text")
	}
	for _, want := range []string{"Probe", "<template", "trace-payload", "<svg"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestEmbeddedExampleRejectsMissingSourceAndOutOfRangePhysicalLine(t *testing.T) {
	example, ok := LookupExampleContent("day-01-complement-lookup")
	if !ok {
		t.Fatal("pilot example missing")
	}
	if err := validateEmbeddedExample("examples/day-01/complement-lookup/example.json", example); err != nil {
		t.Fatal(err)
	}
	broken := example
	broken.Variants = append([]CodeVariant{}, example.Variants...)
	broken.Variants[0].File = "absent.cpp"
	if err := validateEmbeddedExample("examples/day-01/complement-lookup/example.json", broken); err == nil {
		t.Fatal("missing source accepted")
	}
	broken = example
	broken.Variants = append([]CodeVariant{}, example.Variants...)
	broken.Variants[0].AlgorithmEnd = 100000
	if err := validateEmbeddedExample("examples/day-01/complement-lookup/example.json", broken); err == nil {
		t.Fatal("algorithm range past file accepted")
	}
}

func TestSceneRejectsDuplicateAndOffCanvasLabels(t *testing.T) {
	scene := Scene{Width: 100, Height: 100, Nodes: []SceneNode{{ID: "n", Shape: "rect", Width: 20, Height: 20, Text: "A"}}, Labels: []SceneLabel{{ID: "n", X: 50, Y: 50, Text: "duplicate"}}}
	if err := validateScene(scene); err == nil {
		t.Fatal("duplicate scene ID accepted")
	}
	scene.Labels[0] = SceneLabel{ID: "label", X: 101, Y: 50, Text: "outside"}
	if err := validateScene(scene); err == nil {
		t.Fatal("off-canvas label accepted")
	}
}
