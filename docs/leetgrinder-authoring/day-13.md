# Day 13 authoring record

## Scope

Assigned problems: Koko Eating Bananas, Capacity To Ship Packages Within D Days, Find the Smallest Divisor Given a Threshold (all medium); optional Split Array Largest Sum. None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: search on the answer with a feasibility check (least time for buses to finish P trips) and a monotonicity counterexample in prose.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| minimum-time | normal | `3 5 / 1 2 3` | `3` |
| minimum-time | edge-one-bus | `1 1 / 2` | `2` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 13` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
