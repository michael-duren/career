# Day 9 authoring record

## Scope

Assigned problems: First Bad Version, Sqrt(x), Sum of Mutated Array Closest to Target (medium). None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: first-true search on the staircase question k(k+1)/2 ≥ n, including a 64-bit overflow case.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| first-true | normal | `11` | `5` |
| first-true | edge-exact | `10` | `4` |
| first-true | edge-large | `2147483647` | `65536` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 9` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
