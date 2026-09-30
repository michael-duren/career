# Day 18 authoring record

## Scope

Assigned problems: Maximum Average Subarray I, Maximum Number of Vowels in a Substring of Given Length (medium), Number of Sub-arrays of Size K and Average Greater than or Equal to Threshold (medium); optional Find All Anagrams in a String. None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: window letter counts with a repeats counter (count windows with no repeated letter). A plain window-sum example was removed because it matched Maximum Average Subarray I.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| distinct-windows | normal | `abcabb 3` | `3` |
| distinct-windows | edge-short | `ab 3` | `0` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 18` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
