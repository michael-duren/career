# Leetgrinder lesson upgrade implementation plan

> **For agentic workers:** Use `superpowers:subagent-driven-development` or `superpowers:executing-plans` to implement this plan task by task. Read the linked design and your day assignment before editing. Checkboxes track deliverables, not permission to skip review.

**Goal:** Make all 84 existing lessons self-contained, with four-language runnable examples and code-synchronized SVG demonstrations for every distinct technique.

**Architecture:** Keep Go templ and the existing `Lesson(key string)` entry point. Add a shared structured lesson renderer and plain JavaScript player; distribute authoring into independent day directories. Validate authored traces against real C++, Python, Java, and Go programs.

**Tech stack:** Repository Go/templ toolchain, HTML/CSS, browser ES modules, Node test runner, Python verification scripts, and Playwright for browser verification.

**Spec:** [Lesson upgrade design](../specs/2026-09-29-leetgrinder-lesson-upgrade-design.md).

**Assignments:** [All 84 lesson assignments](2026-09-29-leetgrinder-lesson-assignments.md).

**Status:** Implementation is underway on `feat/leetgrinder-lessons`. The shared renderer, player, verifier, browser checks, and pilot days 1, 32, and 66 are in place. Day 2 is complete, making 4 of 84 days authored. The remaining day content and final migration audit are open.

## Global constraints

- Provide C++, Python, Java, and Go with a language switcher.
- Teach intuition, baseline and improved approaches, invariants, complete examples, complexity, and mistakes.
- Animate every distinct algorithm taught, with a normal case and a revealing edge case.
- Start resets to the first frame without playing. Stop pauses in place. Play continues. Include Previous, Next, and speed controls.
- Remove the duplicate Reading path panel. Keep one References footer per lesson with sources presented as bonus material.
- Keep days 78–84 behind the existing “Review after your attempts” disclosure.
- Preserve session numbers, titles, problem assignments, attempts, completion, and review scheduling.
- Treat `/home/mduren/Documents/data/algomonster/articles` as research only. Write original site prose, code, examples, and SVGs. Do not copy article content or assets into the repository or the built site, and do not require the directory at build or runtime.

Do not edit generated `*_templ.go` files manually. Run `make generate`. Do not hand-edit `readings.go`; edit source manifests and run their integration script if citation changes are necessary. Source corpus path: `/home/mduren/Documents/data/algomonster`.

## Review focus

1. A visually convincing trace can disagree with the real algorithm. Task 2 compares executable events, state, and results; Tasks 4–5 require semantic scene review.
2. Switching languages or cases during playback can highlight stale lines or leave multiple timers. Task 3 tests the actual controller and DOM under those actions.
3. Mixed-practice animations can reveal techniques before an attempt or keep running inside collapsed guidance. Tasks 3 and 6 check disclosure containment and pause behavior.
4. Reading metadata also controls review slots and notifications. Tasks 1 and 6 verify that removing the panel leaves those results unchanged.
5. Tree/DP diagrams can become illegible on small screens or unusable without mouse, color, or JavaScript. Tasks 3–6 test mobile layout, keyboard access, transcripts, and reduced motion.

## Ownership and sequencing

One integration owner implements Tasks 1–3 and owns shared files, `package.json`, bibliography manifests, and routing. A reviewer can reject each task independently. Task 4 pilots the authoring contract. Only after its checks pass can Task 5 fan out.

Each content worker owns one day's `lessons/day-NN.json`, `examples/day-NN/`, and `docs/leetgrinder-authoring/day-NN.md`. Workers do not edit the renderer, player, bibliography manifests, dispatcher, or another day's files. Request shared changes through the integration owner with a failing example and expected behavior. Use isolated worktrees for concurrent implementation; do not run generation or commit against a shared worktree.

Days are the review/commit unit. Assignment batches below are scheduling conveniences, not permission to submit seven unreviewed lessons at once. A capable reviewer must check teaching correctness; lower-capability workers receive exact files, source mappings, and the established pilot examples.

## Task 1: Render structured lessons and remove the duplicate panel

**Files:** Create `internal/leetgrinder/lesson_content.go`, `lesson_content.templ`, `lesson_content_test.go`, and `lessons/day-01.json`; modify `lessons.templ`, `views.templ`, `style.css`, `views_test.go`, and `curriculum_test.go`. Update `docs/leetgrinder-sources.md` where it describes the removed panel.

**Consumes:** The spec's exact data types and current `lessonReferences(day int)` component.

**Produces:** `LookupLessonContent(key string) (LessonContent, bool)`, `LookupExampleContent(id string) (ExampleContent, bool)`, `ValidateLessonContent(lesson LessonContent) error`, and `templ lessonContent(content LessonContent)`. Example rendering is completed in Task 3; do not expose placeholder panels in a shipped lesson.

- [ ] Add `TestStructuredLessonRendersSectionsAndEscapesText`: render authored paragraphs with `<script>` and assert escaped text, not an executable element; render headings, lists, prerequisites, checks, and a labeled static diagram.
- [ ] Add `TestLessonContentValidation`: unknown prerequisite day, duplicate section ID, path traversal, unknown example reference, malformed scene, and unknown JSON fields fail with actionable location information. Prerequisites must be earlier existing days, without self-links.
- [ ] Add `TestDayPageHasOneReferencesSectionAndNoReadingPath`: check ordinary day 1 and mixed day 78. Assert exactly one References footer, no Reading path panel, and day 78's footer remains inside the closed guidance disclosure.
- [ ] Implement the spec's types and strict embedded JSON loading. Load only authored runtime data; the private corpus is never consulted. Validate once at initialization and fail with the exact file/field on invalid embedded content. Public lookup returns `false` for an unknown lesson/example rather than crashing on request input.
- [ ] Preserve `Lesson(key string)`. Route migrated days to `lessonContent`; retain the old switch as a temporary migration fallback for unmigrated days. The renderer calls the existing references component exactly once after lesson sections.
- [ ] Delete both `@readingPath(page.Day)` calls and the `readingPath` component. Update the References introduction to “Sources supporting this lesson. Optional background and further detail.” Preserve each source's `Supports` text and link.
- [ ] Leave `Reading`, required-minute values, `RequiredReading()`, `ReviewSlots`, and notification selection unchanged. Record this scope in the source documentation; do not claim required-reading scheduling was removed.
- [ ] Run `make generate` and `go test ./internal/leetgrinder/...`. Existing curriculum/source synchronization, review scheduling, and notification tests must pass alongside the new tests.
- [ ] Review the diff for accidental generated-file edits and out-of-scope changes; commit this deliverable.

## Task 2: Validate four-language programs and authored traces

**Files:** Create `scripts/leetgrinder/verify_examples.py`, `scripts/leetgrinder/test_verify_examples.py`, `internal/leetgrinder/example_content_test.go`, and the day 1 example directories specified by the assignment. Extend `lesson_content.go` for example validation.

**Consumes:** `ExampleContent`, `TraceCase`, `TraceFrame`, and `CodeVariant` from the spec.

**Produces:** CLI `python3 scripts/leetgrinder/verify_examples.py --days 1,32,66` or `--all`, returning zero only when all selected examples pass in all four languages. `--report PATH` writes a JSON report with per-day/example/case/language results and exact commands/toolchain versions. No flags silently skip missing languages.

- [ ] Write verifier tests that execute tiny fixture programs. Cover a correct result, a wrong result, a wrong intermediate variable, an extra/missing event, nonzero exit, timeout, invalid JSON, and a missing compiler. Expected result is nonzero for each broken fixture and zero only for the correct fixture. Error text identifies case and language.
- [ ] Resolve each source file within its example directory, compile in a temporary directory, run each case with a five-second execution timeout, and compare ordered event/variable records and final result literally. Use separate 60-second compiler timeouts and argument arrays with `subprocess`, not shell interpolation.
- [ ] Compile C++ with `g++ -std=c++17 -O2`, Java with `javac --release 17`, and Go from a temporary copy of `main.go.txt` named `main.go` with `go build` using `GOWORK=off` and `GO111MODULE=off` for standalone standard-library examples; run Python with `python3`. Capture diagnostics in the report. Check Python/Java versions against the spec.
- [ ] Validate four unique languages, nonempty normal/edge cases, at least initial and terminal frames, line numbers within each actual file, algorithm display bounds, unique IDs, valid scene references, and safe file paths. Every frame supplies mappings for all four languages.
- [ ] Implement an independently reviewed day 1 complement-lookup example: input values `[8,3,6]`, target `9`, result `[1,2]`; duplicate-value case `[4,4]`, target `8`, result `[0,1]`; no-match case `[4]`, target `8`, result `[]`. Check lookup occurs before insertion. Each program computes results and emits actual observed state; no prerecorded trace output.
- [ ] In the successful main trace, explicitly show `i=2`, `need=3`, and the earlier index `1` before returning. The one-element trace must never pair index zero with itself. These are literal review assertions, not facts inferred solely from a matching final answer.
- [ ] Run `python3 -m unittest discover -s scripts/leetgrinder -p 'test_verify_examples.py'`, then `python3 scripts/leetgrinder/verify_examples.py --days 1 --report /tmp/leetgrinder-day-01.json` and `go test ./internal/leetgrinder/...`.
- [ ] Review code instrumentation and expected outputs independently, then commit. This step proves one example; the remaining day 1 demonstrations belong to Task 4.

## Task 3: Ship the shared SVG player and language switcher

**Files:** Create `internal/leetgrinder/lesson_player.js`, `lesson_player.templ`, `lesson_assets.go`, `tests/leetgrinder-player.test.mjs`, `tests/leetgrinder-lessons.browser.test.mjs`, and `cmd/leetgrinder-preview/main.go`; modify `lesson_content.templ`, `style.css`, `views.templ`, `internal/server/leetgrinder.go`, and `package.json`.

**Consumes:** Validated examples and frames from Tasks 1–2.

**Produces:** `templ lessonExample(example ExampleContent)`, `var LessonPlayerJS string` embedded from `lesson_player.js`, and authenticated GET `/leetgrinder/lesson-player.js` with JavaScript content type. Preview command serves real lesson components at `/day/{day}` on localhost only; it does not add an authentication bypass to the application.

The browser module exports:

```js
// Frames are the validated JSON from the Go renderer.
export function createPlayback(frameCount, state = {index: 0, playing: false, speed: 1})
// Returns a new {index, playing, speed}; no DOM or timer mutation.
export function advancePlayback(state, action, frameCount)
// action.type: start, stop, play, previous, next, tick, speed.
// speed actions have action.value in [0.5, 1, 2].
export function mountExample(element)
// Returns {destroy()}; owns listeners/timer for one example.
export function mountLessonExamples(root = document)
// Returns cleanup function; repeated calls must not double-mount panels.
```

- [ ] Write Node tests for the exact state transitions in the spec: Start/Stop distinction, clamping, final-frame behavior, speed change, and repeated Play. Assert literal states after actions.
- [ ] Add Playwright as a development-only dependency using the repository's package manager/lockfile. Add `test:leetgrinder:player` and `test:leetgrinder:browser` scripts for the two test files. Use `node --test` with Playwright's browser API; no second application runtime is needed.
- [ ] Create the localhost preview command using the actual `Lesson` component, CSS, and player assets. Default address is `127.0.0.1:8097`. Browser tests start/stop this command as a child process. Include ordinary and mixed-practice wrappers from the actual view components or extract a shared guidance wrapper used by both preview and production, so the spoiler check exercises production markup.
- [ ] Render selectable numbered HTML source lines, source-copy controls, four accessible language tabs, case selector, SVG, variable table, explanation, transcript, and controls. Use the source programs directly, with display ranges from `CodeVariant`.
- [ ] Serve the browser module beside the existing CSS route inside the private group. Load as a module once per page. Keep existing form scripts working.
- [ ] Implement SVG rendering using the spec's fixed primitives. Prefix SVG marker/title/description IDs with example and case IDs. Multiple examples must not reference each other's arrowheads or labels. Use `textContent`, not HTML interpolation, for state text.
- [ ] Implement one timer per mounted example and cleanup on destruction/navigation. Pause when document visibility changes or a containing disclosure closes. Catch localStorage failures. Never autoplay on mount, case/language switch, or disclosure opening.
- [ ] Browser-test the day 1 example: click Next to lookup, assert the shown variable values and highlighted lines, switch to each language and assert same case/frame, Play then Stop and assert no further step after two normal intervals, Start and assert frame zero. Use controlled browser timers when available to avoid flaky sleeps.
- [ ] Browser-test two players for independence, repeated mounting, rapid Play presses, speed changes, final-frame behavior, case resets, and hostile text rendering. Confirm a string containing `</script>` remains text and cannot inject an element.
- [ ] Browser-test keyboard tabs/buttons, 360-pixel viewport, reduced motion, and JavaScript disabled. Without JavaScript, all four source variants and full trace transcript remain readable and playback controls are hidden. No page-level horizontal overflow; code/large scenes scroll inside their regions.
- [ ] Run `make generate`, `go test ./internal/leetgrinder/... ./internal/server`, `npm run test:leetgrinder:player`, and `npm run test:leetgrinder:browser`. Document database/environment requirements if server tests need services; an unavailable service is not a pass.
- [ ] Review a working player in the browser and commit. Screenshots support the review but do not replace interaction tests.

## Task 4: Establish three complete pilot lessons

**Files:** `lessons/day-01.json`, `lessons/day-32.json`, `lessons/day-66.json`, their `examples/day-NN/` directories, and `docs/leetgrinder-authoring/day-NN.md`. Use the exact source articles and required techniques in the assignment appendix.

**Consumes:** Shared renderer/player and executable verifier.

**Produces:** Three examples of complete authoring quality and conventions that every later worker follows.

- [ ] For each pilot, inventory the current lesson's concepts and compare with its appendix row. Write exact example cases, expected results, invariants, frame events, and source coverage in its authoring note before writing programs.
- [ ] Finish every lesson-contract section. Include a static concept diagram and an animated panel for every distinct technique, with four language variants and meaningful edge cases. Use small, hand-checkable inputs.
- [ ] Day 1 must explain hashing assumptions, distinguish membership/frequency/index maps, contrast exhaustive search, and show why insertion order prevents reusing an element.
- [ ] Day 32 must visibly preserve BFS level boundaries, show the queue order, distinguish right-side view from minimum-depth stopping, and handle an empty tree and an asymmetric tree.
- [ ] Day 66 must show sequence labels and table coordinates, distinguish subsequence from substring, trace reconstruction tie choices, and explain how a shortest common supersequence uses the alignment. Include repeated characters and empty input.
- [ ] Run the example verifier for `--days 1,32,66` and all shared tests. Browser-check every pilot example and case in all languages, including first, middle, and final frames.
- [ ] A capable reviewer compares the prose, programs, and diagrams against the sources and literal expected results. Record corrections and final acceptance in each authoring note. Fix renderer/schema problems now; do not require later workers to invent local workarounds.
- [ ] Commit each accepted day separately. Only then open bulk authoring.

## Task 5: Complete the remaining 81 lessons

**Files:** Each worker owns only its assigned day files from the appendix.

**Consumes:** Accepted pilots, the full design, the assignment row, and shared tools.

**Produces:** One accepted lesson per day, with code/trace proof and a review note.

Dispatch these batches after dependencies are available. Within a batch, each day is a separate task and commit. Days 1, 32, and 66 are already pilots.

| Batch | Days | Review emphasis |
|---|---|---|
| A | 2–7 | Counts, mutations, coordinates, sorting invariants |
| B | 8–14 | Interval conventions, duplicates, feasibility, overflow |
| C | 15–21 | Pointer elimination, window validity, prefixes |
| D | 22–28 | Node identity, rewiring, amortization, call stacks |
| E | 29–31, 33–35 | Tree contracts, branch ownership, global results |
| F | 36–42 | Decision trees, reversible state, duplicate pruning |
| G | 43–49 | Visited timing, graph identity, weights, resource states |
| H | 50–56 | Heap contents, comparator ties, tries, event time |
| I | 57–63 | State meaning, recurrence, update order, reconstruction |
| J | 64–65, 67–70 | Table geometry, intervals, resource dimensions |
| K | 71–77 | Exchange proofs, endpoint rules, amortized stacks |
| L | 78–84 | Mixed selection, transfer, delayed technique disclosure |

Repeat this checklist for each day:

- [ ] Read the actual lesson, assigned problems, mapped local articles, and existing bottom citations. Record uncovered concepts; use the existing cited materials for gaps. Do not attribute a topic to an article based only on its filename.
- [ ] Use the mapped articles only to research the concepts. Write original explanations, programs, inputs, and visuals. The reviewer checks the draft against the source articles for copied passages, solutions, and diagrams before acceptance.
- [ ] Author `docs/leetgrinder-authoring/day-NN.md`: concepts and sources; exact normal/edge stdin and output for each technique; expected state transitions; baseline versus improved costs; failure cases; questions and answers. A reviewer checks these decisions before the worker writes all four programs.
- [ ] Write original instructional content and diagrams following the lesson contract. Every algorithm's state invariant is explicit. Explain why a step is valid, not merely what the arrow moved to.
- [ ] Implement the full runnable programs in four languages, using the same semantic algorithm and events. Discuss language differences that affect correctness: C++/Java integer overflow, Go integer/byte/rune semantics, Python unbounded integers, library heap order, collection equality, and recursion depth when relevant.
- [ ] Author frame scenes from observed execution, with exact code-line mappings and meaningful explanatory text. Edge cases need their own case selector entry and trace, not only prose.
- [ ] Run `python3 scripts/leetgrinder/verify_examples.py --days N --report /tmp/leetgrinder-day-NN.json`, `make generate`, and `go test ./internal/leetgrinder/...`. Record command outcomes in the authoring note.
- [ ] Run the browser sweep for that day with `LEETGRINDER_DAYS=N npm run test:leetgrinder:browser`. This selection must exercise every example/case and all four language tabs for the day. Inspect its saved full-page and representative frame screenshots at desktop and mobile sizes.
- [ ] Reviewer checks every required technique against the appendix and current lesson. Check final answers by independent reasoning and inspect at least one nontrivial intermediate state per case. Reject superficial filler, unreadable diagrams, hardcoded event output, or copied mismatched language code.
- [ ] Record acceptance and commit only the day's owned files. The integration owner regenerates shared outputs and resolves shared-file changes.

Mixed-practice workers run after their prerequisite topic lessons are accepted. They inspect the actual core problems for their day and explain contrasting approaches after the attempt. Reusing an earlier example is allowed only with a day-specific comparison and suitable inputs; a generic link back to a topic does not satisfy the lesson contract.

### Worker dispatch template

Give the worker this entire block with `NN` replaced, plus the linked artifacts:

> Implement day NN from `docs/superpowers/plans/2026-09-29-leetgrinder-lesson-assignments.md`. Read the design, Task 5 checklist, and pilot day 1/32/66 files. Own only `internal/leetgrinder/lessons/day-NN.json`, `internal/leetgrinder/examples/day-NN/`, and `docs/leetgrinder-authoring/day-NN.md`. First record your exact cases, outputs, invariants, and source coverage for review. Then implement all required techniques in C++, Python, Java, and Go with normal and edge-case traces. Run the verifier and browser checks for your day. Return changed files, verified example/case counts, command outcomes, and unresolved gaps. Do not declare success with a missing language, a placeholder diagram, or an unverified trace. Report shared-component problems with a concrete failing case; do not fork the player.

> The AlgoMonster articles are research material only. Do not copy article text, code, diagrams, images, or files into the repository or site. Author every explanation, program, example, and SVG independently after reading them.

## Task 6: Finish migration and audit all 84 lessons

**Files:** Modify `lessons.templ`, `curriculum_test.go`, `views_test.go`, browser tests, `docs/leetgrinder.md`, `docs/leetgrinder-sources.md`, and `docs/leetgrinder-content-state.md`. Create `scripts/leetgrinder/audit_lessons.py` and `docs/leetgrinder-lesson-upgrade-review.md`.

**Consumes:** All accepted day content, authoring notes, and verification reports.

**Produces:** Complete migration without legacy lesson bodies, one audit report with coverage and test evidence, and updated course documentation.

- [ ] Add a coverage audit comparing actual curriculum days against authored lesson days, technique/example inventory, four-language variants, normal/edge cases, static concept diagrams, knowledge checks, references, and authoring review outcomes. Emit a row for every day with example/case/language totals. Exit nonzero for missing days or incomplete requirements; counts alone do not substitute for reviewer acceptance.
- [ ] Strengthen `TestEveryLessonRendersInstruction` to check the lesson contract rather than the current 100-word threshold. Keep curriculum assignment/source tests. Retain the existing headings or update assertions to the agreed headings, not weaker generic HTML checks.
- [ ] Delete all superseded `lessonDayNN` bodies and the fallback switch once all 84 JSON lessons render. Keep `Lesson(key string)` and `lessonReferences(day int)` as the public integration points. Remove dead CSS for the deleted reading panel where no other component uses it.
- [ ] Make the browser suite enumerate all lesson pages, examples, cases, languages, and frames with `LEETGRINDER_DAYS` as an optional filter. Assert no console errors; correct line/variable/description updates; valid SVG labels; reachable controls; one references footer; no duplicate Reading path. Save failure screenshots with day/example/case/frame IDs.
- [ ] For days 78–84 assert that all instructional diagrams, code panels, transcripts, and references are descendants of the initially closed disclosure. Opening reveals paused examples; closing pauses them. Avoid claiming this is a security boundary; it is spoiler-conscious presentation.
- [ ] Re-run existing scheduling, review-slot, notification, and attempt/completion tests. Compare curriculum problem IDs and day assignments with the pre-change baseline; no differences are expected.
- [ ] Run `make generate`, `go test ./internal/leetgrinder/... ./internal/server`, the verifier unit suite, `python3 scripts/leetgrinder/verify_examples.py --all --report /tmp/leetgrinder-examples-all.json`, `python3 scripts/leetgrinder/audit_lessons.py`, and both npm player/browser scripts without a day filter.
- [ ] Update documentation to describe in-lesson learning, animation controls, language choice, references, and source authorship. Explicitly retain the note that legacy required-reading metadata still governs review scheduling; do not imply the scheduling model changed.
- [ ] Write `docs/leetgrinder-lesson-upgrade-review.md` with 84 day rows, test commands/results, toolchain versions, reviewed diagrams, unresolved issues, and final coverage totals. Completion requires zero unresolved missing techniques/languages/cases and zero failing gates.
- [ ] Review the full diff, verify no source corpus or temporary binary/report artifacts were added, and commit. Present the completed work for user review; deployment is outside this plan.
- [ ] Inspect the production build output and embedded assets for copied article files, text, code, and images. Search tracked files for references that would load the local article directory at build or runtime. Confirm the app builds and serves lessons without that directory mounted. Record the result in the review document.

## Plan self-review

Confirmed user choices map to Tasks 1–6: four languages and switcher in Tasks 2–3; deeper teaching and every-technique coverage in Tasks 4–5; all playback controls in Task 3; duplicate-panel removal in Task 1; mixed-practice containment in Tasks 1, 3, and 6. The shared schema is defined once in the design. The appendix has 84 ordered assignments and its local source paths exist.

Planning decisions that can be adjusted before execution: Python as the first-visit language, instrumented program traces, and structured JSON content. No external reference links were rechecked in this planning pass, and the mapped articles have not all received full semantic review. Task 5 requires that review before using their claims.
