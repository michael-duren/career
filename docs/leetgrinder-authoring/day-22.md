# Day 22 authoring record

## Scope

Assigned problems: Reverse Linked List, Merge Two Sorted Lists, Reverse Linked List II (medium). None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: building a reversed copy by adding to the front, and inserting a copy after each node without losing the rest. In-place reversal is left to the learner.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| insert-copies | normal | `2 0 / 1 2` | `1->1->2->2` |
| insert-copies | edge-empty | `0 0` | `empty` |
| reversed-copy | normal | `3 0 / 1 2 3` | `3->2->1` |
| reversed-copy | edge-single | `1 0 / 7` | `7` |
| reversed-copy | edge-empty | `0 0` | `empty` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 22` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
