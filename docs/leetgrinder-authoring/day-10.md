# Day 10 authoring record

## Scope

Assigned problems: Find First and Last Position of Element in Sorted Array (medium), Find Smallest Letter Greater Than Target, Count Negative Numbers in a Sorted Matrix. None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: lower bound for counting values below a cutoff, and a staircase walk counting cells ≤ x in a row- and column-sorted grid.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| count-below | normal | `5 3 / 1 3 3 3 8` | `1` |
| count-below | edge-absent | `5 4 / 1 3 3 3 8` | `4` |
| count-below | edge-all-below | `5 10 / 1 3 3 3 8` | `5` |
| staircase-count | normal | `3 3 5 / 1 4 7 / 2 5 8 / 3 6 9` | `5` |
| staircase-count | edge-none | `3 3 0 / 1 4 7 / 2 5 8 / 3 6 9` | `0` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 10` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
