# Leetgrinder lesson upgrade design

Status: planning artifact based on the user's September 29 answers. Implementation has not started.

## Outcome and agreed scope

All 84 existing sessions become self-contained instructional lessons. A learner can understand the concepts, read complete code, follow its execution visually, and attempt the existing assignments without first reading external material.

- Provide C++, Python, Java, and Go with a language switcher.
- Teach intuition, baseline and improved approaches, invariants, complete examples, complexity, and mistakes.
- Animate every distinct algorithm taught, with a normal case and a revealing edge case.
- Start resets to the first frame without playing. Stop pauses in place. Play continues. Include Previous, Next, and speed controls.
- Remove the duplicate Reading path panel. Keep one References footer per lesson with sources presented as bonus material.
- Keep days 78–84 behind the existing “Review after your attempts” disclosure.
- Preserve session numbers, titles, problem assignments, attempts, completion, and review scheduling.

The last constraint follows the user's choice to remove only the duplicate panel. Existing required-reading metadata also affects dashboard text, notifications, and base review slots. Do not silently change those behaviors in this project. References become supplementary within the lesson; legacy scheduling terminology remains a known limitation for a separate decision.

The AlgoMonster articles under `/home/mduren/Documents/data/algomonster/articles` are research material only. No article text, code, image, diagram, extracted dataset, or article file may be copied, embedded, served, or committed as site content. Agents must write original explanations, implementations, and SVGs after studying the concepts. The local directory is unavailable to the built site and must never be a build or runtime dependency. The existing public References footer may retain its separately curated links and attribution.

## Evidence from the repository

`internal/leetgrinder/lessons.templ` dispatches all 84 lessons and defines short prose walkthroughs. It has no code blocks or SVG diagrams. `views.templ` calls `readingPath(day)` twice in alternative branches and renders the lesson inside a disclosure for mixed practice. The Go server serves embedded course CSS at `/leetgrinder/style.css` inside its authenticated route group.

`Reading` and `Day.Readings` are in `model.go`. Bibliography manifests `sources_01_28.json`, `sources_29_56.json`, and `sources_57_84.json` generate `readings.go` through `scripts/leetgrinder/integrate_readings.py`. `dashboard.go`, `srs.go`, and `notify/messages.go` consume reading status. Leave their scheduling contracts intact.

The local AlgoMonster inventory lists 72 lessons and 212 problems, with 283 full articles and one preview. Inspected articles include instructional prose, language-specific starter/solution material, and image references. The local collection has no SVG/PNG/JPG assets. Local source paths have been checked for all 84 assignment rows; coverage still needs semantic review per article.

## Approach

Use a shared Go templ lesson renderer, structured authored lesson data, and one plain JavaScript SVG player. Keep the existing `Lesson(key string) templ.Component` entry point and existing bibliography footer. Embed authored JSON and example files in the Go binary. Each day gets its own files so parallel workers do not edit the same large template.

Considered alternatives were inline bespoke templates/scripts for every lesson, and a React application embedded into the Go pages. Bespoke scripts would make controls and accessibility vary across hundreds of examples. A React application would introduce a second rendering stack into this feature. The shared Go renderer and browser module fit the current server and allow independent content work.

These are authored, deterministic execution traces. The browser does not execute submitted C++, Python, Java, or Go. Authors supply runnable programs, and a local verifier executes those programs against their trace expectations before publication.

## Lesson contract

Each lesson contains, in order:

1. Learning objectives and prerequisite links to earlier course days.
2. Intuition and a static concept diagram with visible labels and a text explanation.
3. Definitions, input assumptions, and recognition cues, including when the technique does not apply.
4. A correct baseline with its cost and a specific explanation of repeated work.
5. Each distinct technique: invariant, improved approach, complete four-language program, interactive trace, and explanation of correctness.
6. A second trace case per technique showing a meaningful boundary or failure mode.
7. “Complexity and cautions,” with variables defined, auxiliary/output space separated, and reasoning tied to operations in the code.
8. “Before you finish,” with prediction, explanation, and transfer questions. Answers are in disclosures.
9. One References footer, retaining source-specific attribution.

Keep “Worked example” as a visible heading for continuity with current tests. Do not use a word-count quota as a substitute for teaching completeness. Deeper proofs and variant discussions can use disclosures, but core explanation must remain visible. Concept diagrams and algorithm traces must be specific to the day's content, not decorative repeated art.

Every semicolon-separated technique in the assignment appendix needs a demonstration. The appendix is a minimum inventory; retain concepts from the current lesson that its row does not enumerate. Closely related cases can share one algorithm example with separate case selectors. Distinct state transitions or algorithms need separate examples. Baseline code is also available in all four languages; it may share the example panel as an approach selector, and needs its own trace when its execution differs.

## Data and file contract

New files live under `internal/leetgrinder/`:

- `lesson_content.go`: types, embedded content loader, validation, and lookup.
- `lesson_content.templ`: generic lesson/section/diagram/example rendering.
- `lesson_player.templ`: example panel and initial non-JavaScript rendering.
- `lesson_player.js`: browser controller and SVG updates, exported pure playback functions.
- `lesson_assets.go`: embedded JavaScript export.
- `lessons/day-NN.json`: prose and ordered references to example directories.
- `examples/day-NN/<slug>/example.json`: example metadata, cases, and frames.
- `examples/day-NN/<slug>/main.cpp`, `main.py`, `Main.java`, `main.go.txt`: complete runnable, instrumented programs shown in the code panel.

The Go source is stored as `main.go.txt` so the repository-wide Go test command does not treat a directory containing C++ and Go examples as a Go package. The verifier stages that file as `main.go` in a temporary build directory. Physical line mappings refer to the stored source and remain unchanged by staging.

The standard headers/imports and trace-emission helper are collapsed in the code display; the algorithm body is expanded. Full program source is available to copy. Do not maintain a separate display-only algorithm that can drift from the executed program.

Use these Go interfaces, with matching JSON names in lower camel case:

```go
type LessonContent struct {
    Day int
    Objectives []string
    Prerequisites []int
    Sections []LessonSection
}
type LessonSection struct {
    ID string
    Heading string
    Paragraphs []string
    Bullets []string
    Diagram *Scene
    ExampleIDs []string
    Checks []KnowledgeCheck
    Optional bool
}
type KnowledgeCheck struct { Prompt string; Answer string }
type CodeVariant struct {
    Language string // cpp, python, java, go
    File string     // basename within the example directory
    AlgorithmStart int // inclusive physical line number
    AlgorithmEnd int   // inclusive physical line number
}
type ExampleContent struct {
    ID string // globally unique: day-NN-slug
    Title string
    Invariant string
    Variants []CodeVariant
    Cases []TraceCase
}
type TraceCase struct {
    ID string
    Label string
    Input string // exact stdin, newline terminated
    ExpectedOutput string // exact result string emitted by program
    Frames []TraceFrame
}
type TraceFrame struct {
    Event string // stable semantic checkpoint, not a source line number
    Lines map[string][]int // each of the four languages; physical source lines
    Explanation string
    Variables []Variable
    Scene Scene
}
type Variable struct { Name string; Value string }
type Scene struct {
    Width int
    Height int
    Description string
    Nodes []SceneNode
    Edges []SceneEdge
    Labels []SceneLabel
}
type SceneNode struct {
    ID string
    Shape string // rect or circle
    X int; Y int; Width int; Height int
    Text string
    Role string // neutral, active, visited, discarded, result
}
type SceneEdge struct {
    ID string
    From string; To string // node IDs
    Label string
    Directed bool
    Role string
}
type SceneLabel struct { ID string; X int; Y int; Text string }
func LookupLessonContent(key string) (LessonContent, bool)
func LookupExampleContent(id string) (ExampleContent, bool)
func ValidateLessonContent(lesson LessonContent) error
```

Rect coordinates are top-left; circle coordinates use the same bounding box and equal width/height. Edge endpoints connect node centers, clipped to their shapes. Rows of boxes depict arrays, matrices, tables, queues, and stacks; circles and edges depict lists, trees, graphs, tries, and calls. Add labels for indices, pointers, intervals, and active ranges. All primitives have stable IDs within a case. Do not put executable JavaScript, arbitrary SVG markup, or raw HTML in JSON.

Render text through templ escaping and DOM `textContent`. JSON payloads must use safe serialization, including escaping `<` so a string containing `</script>` cannot end its container. File paths must stay inside the example directory. Unknown IDs, invalid source lines, missing languages, missing frames, duplicate scene IDs, nonexistent edge targets, and out-of-bounds geometry fail validation with the day, example, and case in the error.

## Animation behavior

- Initial frame is zero, paused, and describes the input before the first mutation.
- Start sets frame zero and pauses; Stop pauses without resetting; Play resumes and stops at the final frame. Play on the final frame does nothing until Start or Previous.
- Previous/Next pause and move one frame, clamped to the endpoints. Endpoint controls are disabled.
- Speeds are 0.5×, 1×, and 2×; one semantic step per second at 1×. A speed change replaces the pending timer; repeated Play never creates a second timer.
- The case selector resets to the new case's frame zero and pauses.
- Switching language preserves case and frame, pauses playback, and updates highlighted lines. First visit defaults to Python. Remember the chosen language in localStorage under `leetgrinder.lessonLanguage`; invalid values or blocked storage fall back safely. This default is a planning choice, not an explicit user preference.
- SVG, explanation, variables, and code highlight update atomically from one frame. Language-specific lines map to the same semantic event.
- Independent examples have independent playback state. Closing an ancestor disclosure, hiding the document, or leaving the page pauses active playback. Opening a disclosure never starts playback.
- No autoplay. Reduced-motion preference disables optional tweening; discrete stepping and explicitly requested playback still work.

Use real HTML buttons beside the SVG. Implement the language picker as labeled native select or accessible tabs; use tabs for the finished UI, with arrow-key navigation and correct tab/panel association. Code is selectable HTML, not text drawn into the SVG. SVG has a title and description. Each current frame has a readable explanation and variable table outside the graphic; a collapsed full transcript is available without JavaScript. Avoid live-announcing every timer tick; announce manual steps politely. Color is supplemented with labels, outlines, and a legend.

At 360 CSS pixels, page-level horizontal overflow is absent. Code may scroll horizontally inside its panel. Large scenes can scroll within a labeled diagram region while controls stay reachable. With JavaScript disabled, show all four labeled code variants and the trace transcript with its static first frame; hide inert playback controls. With JavaScript, enhance these into the switcher/player.

## Executable evidence contract

Programs accept a case's exact stdin. Their stdout consists of JSON lines:

```json
{"event":"lookup","variables":[{"name":"i","value":"2"},{"name":"need","value":"3"}]}
{"result":"[1,2]"}
```

One event record corresponds to each authored frame, including initial and terminal states. Its ordered variables must match that frame literally. The final result record matches `ExpectedOutput`. The same events and normalized state must come from all four programs. Include heap contents in a canonical form and stable ordering for maps/sets so verification is deterministic. Language-specific implementation details may use different lines but cannot silently change the algorithm or invariant.

The verifier executes actual code with timeouts, consumes exact outputs, and rejects missing/extra records. A reviewed SVG depicts those variables faithfully; machine checks verify declared relationships for arrays, pointer positions, queue/stack contents, and table values when represented. A reviewer must still inspect diagram meaning. Trace instrumentation observes state; it must not print a prerecorded answer or scripted event list independently of the algorithm.

Use C++17, Python 3.11+, Java 17+, and the repository's Go toolchain. No third-party libraries in educational examples. Implement tiny JSON emission helpers as needed; keep them collapsed and outside the highlighted algorithm region. Each normal/edge case must run in every language. Missing toolchains are an explicit failed/incomplete gate, not a skipped success.

## Authoring and source use

Read the existing lesson, its assignments, the mapped local source articles, and applicable bottom references before writing. Use AlgoMonster only to identify concepts, gaps, and useful teaching approaches. Write original prose, code, examples, and SVGs. Do not reproduce article passages, solution code, diagrams, or linked images in the site. The source directory is available to workers for research and absent from the built application.

Each `docs/leetgrinder-authoring/day-NN.md` records concepts, exact source paths and useful sections, coverage gaps, example inputs/results, intended frame transitions, four-language execution evidence, and reviewer decisions. Keep external citations accurate. The source manifests remain the bibliography authority; only the integration owner edits them and regenerates `readings.go` if citations change.

## Delivery and review

Build the shared renderer and player, then complete a three-lesson pilot: day 1 for arrays/maps, day 32 for trees/queues, and day 66 for DP tables. A capable reviewer checks those lessons before bulk authoring. The pilot must establish executable tracing, readable mobile scenes, and code synchronization.

After the pilot, workers take separate day directories. Each day has a content review and a running-browser review. A final audit checks all 84 lesson contracts and all examples; representative browser checks alone do not prove all authored diagrams work.

Completion means every lesson is migrated, every code variant executes correctly, all traces agree with real code, no duplicate Reading path exists, and mixed-practice guidance stays collapsed initially. Existing lesson bodies can then be deleted while retaining the public dispatcher and bibliography footer. The plan does not authorize deployment or alter stored learner data.

Before accepting any lesson, its reviewer compares the draft with the mapped articles for copied wording, code, and visual composition. The final audit checks that no files from the private article directory appear in the repository, build context, embedded assets, served routes, or generated site output.
