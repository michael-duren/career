# Day 4 authoring record

## Scope

Assigned problems: Intersection of Two Arrays II, Happy Number, Valid Sudoku (medium); optional Intersection of Two Arrays. None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only; the learner writes the solution.

Techniques are taught on separate examples: Set versus counts (sock pairs), loop detection on the toy rule next(x) = (x·x + 1) % 10 with a visited set and with slow/fast pointers, and one set per row and column in a letter grid.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| cycle-floyd | normal | `3` | `7` |
| cycle-floyd | edge-starts-in-loop | `0` | `0` |
| cycle-visited | normal | `3` | `0` |
| cycle-visited | edge-starts-in-loop | `0` | `0` |
| distinct-grid | normal | `3 / abc / bca / cab` | `true` |
| distinct-grid | edge-column-clash | `3 / ab. / .ca / a..` | `false` |
| sock-pairs | normal | `6 / red blue red red blue green` | `2` |
| sock-pairs | edge-no-pairs | `3 / red blue green` | `0` |

## Writing style

Problem-first, plain language, concrete numbers. Step captions describe the values at that step. See day 1 for the reference style.

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 4` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
