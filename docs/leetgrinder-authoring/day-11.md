# Day 11 authoring record

## Scope

Assigned problems: Search a 2D Matrix (medium), Search a 2D Matrix II (medium), Find Minimum in Rotated Sorted Array (medium). None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: binary search over computed values (perfect squares) and flat-index/grid conversion (static diagram).

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| computed-array | normal | `10 49` | `7` |
| computed-array | edge-absent | `10 50` | `-1` |
| computed-array | edge-zero | `10 0` | `0` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 11` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
