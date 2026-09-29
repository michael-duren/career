# Day 27 authoring record

## Scope

Assigned problems: Implement Queue using Stacks, Implement Stack using Queues, Design Circular Queue (medium). None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: undo and redo with two stacks, including amortized cost in prose.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| undo-redo | normal | `a b undo undo redo c` | `ac` |
| undo-redo | edge-nothing-to-undo | `undo a redo` | `a` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 27` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
