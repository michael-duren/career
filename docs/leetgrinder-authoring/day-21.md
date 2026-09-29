# Day 21 authoring record

## Scope

Assigned problems: Product of Array Except Self, Continuous Subarray Sum, Subarray Sums Divisible by K (all medium). None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: running remainders on a clock with negative normalization; a repeated remainder marks a run summing to a multiple.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| clock-repeat | normal | `4 12 / 5 -3 10 4` | `0..2` |
| clock-repeat | edge-negative | `4 12 / -5 2 3 -1` | `0..2` |
| clock-repeat | edge-no-repeat | `2 12 / 1 1` | `none` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 21` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
