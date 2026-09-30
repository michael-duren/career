# Day 17 authoring record

## Scope

Assigned problems: Container With Most Water, 3Sum, 3Sum Closest (all medium); optional 4Sum. None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: fixing one value and counting triples below a target with two pointers.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| triples-below | normal | `4 2 / -2 0 1 3` | `2` |
| triples-below | edge-none | `4 -10 / -2 0 1 3` | `0` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 17` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
