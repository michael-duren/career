# Day 3 authoring record

## Scope

Assigned problems: Group Anagrams (medium), Isomorphic Strings, Word Pattern. None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only; the learner writes the solution.

Techniques are taught on separate examples: Grouping by a normalized key (license plates ignoring case and dashes) and one-to-one matching with one map per direction (students and lockers).

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| lockers | normal | `3 / ann 3 bob 5 ann 3` | `true` |
| lockers | edge-shared-locker | `2 / ann 3 bob 3` | `false` |
| lockers | edge-two-lockers | `2 / ann 3 ann 4` | `false` |
| plate-groups | normal | `5 / AB-12 cd3 ab12 C-D3 xy9` | `ab12:AB-12,ab12|cd3:cd3,C-D3|xy9:xy9` |
| plate-groups | edge-one-group | `3 / a-b AB A-B` | `ab:a-b,AB,A-B` |

## Writing style

Problem-first, plain language, concrete numbers. Step captions describe the values at that step. See day 1 for the reference style.

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 3` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
