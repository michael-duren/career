# Day 5 authoring record

## Scope

Assigned problems: Merge Sorted Array, Sort an Array (medium), Sort Colors (medium). None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only; the learner writes the solution.

Techniques are taught on separate examples: Loop invariants through insertion sort, merging two sorted lists into a new array, and last-element quicksort including its all-equal worst case.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| insertion-sort | normal | `4 / 5 2 4 1` | `[1,2,4,5]` |
| insertion-sort | edge-sorted | `4 / 1 2 3 4` | `[1,2,3,4]` |
| merge-front | normal | `2 / 2 9 / 3 / 1 5 8` | `[1,2,5,8,9]` |
| merge-front | edge-one-empty | `0 /  / 2 / 3 4` | `[3,4]` |
| quicksort | normal | `5 / 4 7 2 6 3` | `[2,3,4,6,7]` |
| quicksort | edge-all-equal | `3 / 2 2 2` | `[2,2,2]` |

## Writing style

Problem-first, plain language, concrete numbers. Step captions describe the values at that step. See day 1 for the reference style.

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 5` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
