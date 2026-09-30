# Day 20 authoring record

## Scope

Assigned problems: Range Sum Query - Immutable, Find Pivot Index, Subarray Sum Equals K (medium). None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: prefix sums with a leading zero answering range totals, including negative values.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| range-totals | normal | `3 / 3 -2 4 / 2 / 0 2 / 1 2` | `[5,2]` |
| range-totals | edge-single | `3 / 3 -2 4 / 1 / 1 1` | `[-2]` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 20` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
