# Day 8 authoring record

## Scope

Assigned problems: Binary Search, Search Insert Position, Single Element in a Sorted Array (medium). None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: exact search with both ends included and with hi excluded, on found, absent, and empty inputs.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| closed-interval | normal | `4 11 / 2 6 11 15` | `2` |
| closed-interval | edge-absent | `4 8 / 2 6 11 15` | `-1` |
| closed-interval | edge-empty | `0 5` | `-1` |
| half-open | normal | `4 11 / 2 6 11 15` | `2` |
| half-open | edge-absent | `4 8 / 2 6 11 15` | `-1` |
| half-open | edge-empty | `0 5` | `-1` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 8` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
