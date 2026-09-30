# Day 16 authoring record

## Scope

Assigned problems: Remove Duplicates from Sorted Array, Remove Duplicates from Sorted Array II (medium), Move Zeroes. None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: read/write compaction that removes every copy of a value while keeping order.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| remove-value | normal | `4 5 / 5 2 5 7` | `[2,7]` |
| remove-value | edge-all-removed | `2 5 / 5 5` | `[]` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 16` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
