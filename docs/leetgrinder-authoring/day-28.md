# Day 28 authoring record

## Scope

Assigned problems: Fibonacci Number, Pow(x, n) (medium), K-th Symbol in Grammar (medium). None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: recursive contracts and base cases through digit sums and halving recursion for binary strings.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| binary-string | normal | `6` | `110` |
| binary-string | edge-zero | `0` | `0` |
| digit-sum | normal | `1234` | `10` |
| digit-sum | edge-single-digit | `7` | `7` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 28` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
