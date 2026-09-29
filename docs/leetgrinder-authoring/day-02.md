# Day 2 authoring record

## Scope

Assigned problems: Ransom Note, Brick Wall (medium), Majority Element. None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only; the learner writes the solution.

Techniques are taught on separate examples: Counters as a running summary: spending stock counts as orders arrive, turning segment lengths into stop positions, and the pair-cancellation argument for a majority (static diagram only).

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| shared-stops | normal | `2 3 1 / 1 1 3 1` | `[2,5]` |
| shared-stops | edge-none-shared | `3 3 / 2 4` | `[]` |
| stock-orders | normal | `aabbc / abcab` | `filled` |
| stock-orders | edge-short | `aab / abaa` | `short at 3` |

## Writing style

Problem-first, plain language, concrete numbers. Step captions describe the values at that step. See day 1 for the reference style.

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 2` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
