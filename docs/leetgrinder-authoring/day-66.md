# Day 66 authoring record

## Scope and source research

The lesson covers the existing two-sequence alignment walkthrough and all four assigned demonstrations. The local AlgoMonster files below were used for concepts only. The lesson prose, code, cases, and diagram data were written for this course.

- `/home/mduren/Documents/data/algomonster/articles/09-dynamic-prog/05-01-dp_two_sequence_intro.md`: "The state: one index into each sequence", "Every transition reads the same three neighbours", "Fill order", and "Space" helped check prefix-state and rolling-row coverage.
- `/home/mduren/Documents/data/algomonster/articles/09-dynamic-prog/05-02-longest_common_subsequence.md`: "Why Dynamic Programming?", "Bottom-Up: Filling the Table", and "Recovering the subsequence itself" helped check repeated work, recurrence, and backtracking coverage.
- `/home/mduren/Documents/data/algomonster/articles/09-dynamic-prog/05-06-shortest_common_supersequence.md`: "The Trick: Decide Who Writes the Last Character", "Bottom-Up: Filling the Table", and "Complexity" helped check the merge rule and output-space explanation.

The three assigned files cover LCS and shortest common supersequence directly. The lesson also needed a distinct common-substring reset demonstration. That demonstration was derived from the contiguous-match state definition and checked against the existing day 66 warning about confusing subsequences and substrings; no article text or implementation was reused. The existing lesson also mentions uncrossed lines and deletion cost, so both relations appear in the recognition section and checks.

The published bibliography is owned by the integration work. This record does not alter source manifests or generated readings.

## Examples and trace plan

| Example | Normal input | Result | Edge input | Result | State transitions |
| --- | --- | --- | --- | --- | --- |
| `baseline` | `ab`, `ba` | `1` | empty, `cat` | `0` | Initial state, each recursive call in evaluation order, final length. A mismatch visits both branches. |
| `lcs-table` | `cab`, `acb` | `2` | `a`, `b` | `0` | Initial state, completed table row for each prefix of `a`, final bottom-right value. |
| `reconstruct` | `cab`, `acb` | `cb` | `a`, `b` | empty | Initial state, each completed row, each backward move with collected symbols, reversed result. Upper predecessor wins ties. |
| `substring` | `abxc`, `zabq` | `ab` | `abc`, `ace` | `a` | Initial state, each completed suffix-length row and best end position, final slice. Mismatches write zero. |
| `supersequence` | `ab`, `ba` | `bab` | empty, `cat` | `cat` | Initial state, each LCS row, each backward merge move, reversed result. Upper predecessor wins ties. |

Each `example.json` contains the exact newline-terminated stdin, expected result, ordered event records, variables, source-line mappings, and a labeled SVG scene. Scenes show the two input rows; row events show every value of the completed DP row. Backtrack events label the current prefix boundaries and the backward collection. The normal cases expose a match, a mismatch, or a tie; the edge cases expose an empty prefix, no shared symbol, or a substring reset.

## Execution evidence

On 2026-09-29, `python3 scripts/leetgrinder/verify_examples.py --days 66 --root internal/leetgrinder/examples` exited 0. The verifier compiled or ran all four variants of all five examples and compared every normal and edge event, ordered variable list, and final result against `example.json`. Go sources are named `main.go.txt` and staged by the verifier before compilation to keep the enclosing Go package testable.

The first run exposed a C++ recursive-call ordering mismatch. Its `max` arguments were evaluated in a different order from Python. The baseline now stores the first branch result before calling the second branch, making emitted call order stable. The successful verifier run followed that correction.

`GOCACHE=/tmp/career-leetgrinder-go-cache go test ./internal/leetgrinder -run '^$'` exited 0 after the source-line range correction. This loads and validates every embedded day 66 scene and line map against the current Go schema. The four focused lesson and example content tests also exited 0.

## Review decisions

- Use lowercase ASCII inputs in this pilot so byte indexing in C++, Java, and Go matches Python character indexing. A Unicode lesson would need a separate code path and explicit symbol semantics.
- Use one event per completed row rather than one per cell. The row scene still exposes all cell values, while a learner can step through a normal case without dozens of frames.
- Keep full tables for reconstruction and supersequence. The length-only optimization is explained but the trace and backtrack need the stored predecessor values.
- Use an upper-predecessor tie rule for deterministic reconstruction and supersequence outputs. Other tie rules can yield different equally valid strings.
- A content reviewer should inspect that the row and backtrack scenes remain readable at the mobile viewport and that the source-line highlight follows the shown event.

## Review corrections

The reviewed baseline case now revisits recursive states, and the table example includes repeated characters. Reconstruction frames state and draw the upper and left predecessor values at a tie. The substring trace marks the best cell. Go reconstruction uses byte buffers, preserving the stated output-construction bound for the ASCII cases in this lesson. The verifier passed all 40 language and case runs after these changes. The mobile browser sweep rendered all five examples at 360 pixels without page-level overflow.
