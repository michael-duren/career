# Day 19 authoring record

## Scope

Assigned problems: Longest Substring Without Repeating Characters, Minimum Size Subarray Sum, Max Consecutive Ones III (all medium); optional Longest Repeating Character Replacement, Fruit Into Baskets. None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: grow-and-shrink windows for the longest run of positive values within a budget.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| longest-budget | normal | `6 4 / 1 2 1 3 1 1` | `3` |
| longest-budget | edge-nothing-fits | `2 4 / 5 6` | `0` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 19` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
