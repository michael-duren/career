# Day 15 authoring record

## Scope

Assigned problems: Valid Palindrome, Reverse String, Two Sum II - Input Array Is Sorted (medium). None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: mirror comparison from both ends, and counting pairs with sum ≤ T by moving two pointers inward.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| mirror-check | normal | `5 0 / 1 2 3 2 1` | `true` |
| mirror-check | edge-mismatch | `4 0 / 1 2 2 3` | `false` |
| mirror-check | edge-single | `1 0 / 7` | `true` |
| pairs-two-pointers | normal | `5 6 / 1 2 4 5 7` | `4` |
| pairs-two-pointers | edge-none | `5 2 / 1 2 4 5 7` | `0` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 15` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
