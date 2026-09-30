# Day 14 authoring record

## Scope

Assigned problems: Minimum Number of Days to Make m Bouquets, Maximum Distance Between a Pair of Values, Successful Pairs of Spells and Potions (all medium); optional Minimum Limit of Balls in a Bag, Minimized Maximum of Products. None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: a template review and counting pairs with sum ≤ T by one binary search per element.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| pairs-at-most | normal | `5 6 / 1 2 4 5 7` | `4` |
| pairs-at-most | edge-none | `5 2 / 1 2 4 5 7` | `0` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 14` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
