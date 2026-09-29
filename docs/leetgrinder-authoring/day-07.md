# Day 7 authoring record

## Scope

Assigned problems: Spiral Matrix (medium), Rotate Image (medium), Set Matrix Zeroes (medium); optional Rotate Array. None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only; the learner writes the solution.

Techniques are taught on separate examples: Coordinates and bounds (border walk with thin-grid guards), cell movement rules (in-place transpose), and separating decisions from changes (mark original 1s, then light the cells below).

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| border | normal | `3 4 / 1 2 3 4 / 5 6 7 8 / 9 10 11 12` | `[1,2,3,4,8,12,11,10,9,5]` |
| border | edge-one-row | `1 3 / 1 2 3` | `[1,2,3]` |
| border | edge-one-column | `3 1 / 1 / 2 / 3` | `[1,2,3]` |
| light-below | normal | `3 3 / 1 0 0 / 0 0 1 / 0 0 0` | `[[1,0,0],[1,0,1],[0,0,1]]` |
| light-below | edge-cascade-bait | `3 1 / 1 / 0 / 0` | `[[1],[1],[0]]` |
| transpose | normal | `3 3 / 1 2 3 / 4 5 6 / 7 8 9` | `[[1,4,7],[2,5,8],[3,6,9]]` |
| transpose | edge-2x2 | `2 2 / 1 2 / 3 4` | `[[1,3],[2,4]]` |

## Writing style

Problem-first, plain language, concrete numbers. Step captions describe the values at that step. See day 1 for the reference style.

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 7` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
