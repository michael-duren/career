# Day 12 authoring record

## Scope

Assigned problems: Search in Rotated Sorted Array (medium), Search in Rotated Sorted Array II (medium), Find Peak Element (medium); optional Find Minimum in Rotated Sorted Array II. None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: slope-based elimination on a single-peak mountain array.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| mountain-peak | normal | `5 0 / 1 3 7 4 2` | `2` |
| mountain-peak | edge-early-peak | `5 0 / 2 9 5 3 1` | `1` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 12` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
