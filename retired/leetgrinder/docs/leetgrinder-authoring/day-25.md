# Day 25 authoring record

## Scope

Assigned problems: Remove Linked List Elements, Odd Even Linked List (medium), Swap Nodes in Pairs (medium); optional Add Two Numbers II, Reorder List, Design Linked List. None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: dummy-head deletion of nodes above a limit.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| drop-large | normal | `5 5 / 1 6 2 7 3` | `1->2->3` |
| drop-large | edge-head | `3 5 / 9 9 1` | `1` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 25` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
