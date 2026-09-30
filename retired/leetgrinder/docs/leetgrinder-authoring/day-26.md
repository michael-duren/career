# Day 26 authoring record

## Scope

Assigned problems: Valid Parentheses, Min Stack (medium), Evaluate Reverse Polish Notation (medium); optional Decode String. None of them is solved in the lesson. Each has a "Your turn" section that restates the problem and gives a process hint only. At most one problem per day is close to a worked example; the medium problems are left as real challenges.

Techniques are taught on separate examples: tag matching and folder-path simplification with a stack.

Research articles listed in the assignment appendix were consulted as research only. All prose, programs, cases, and diagrams are original. Teaching examples were checked against all 300 curriculum problems so none of them solves an assigned problem on any day.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| folder-path | normal | `a b .. c .` | `/a/c` |
| folder-path | edge-above-root | `.. ..` | `/` |
| tag-matching | normal | `<a> <b> </b> </a>` | `true` |
| tag-matching | edge-crossed | `<a> <b> </a> </b>` | `false` |
| tag-matching | edge-unclosed | `<a>` | `false` |

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 26` passes for every example in C++, Python, Java, and Go. `go test ./internal/leetgrinder/...`, the lesson audit, and the browser sweep for this day pass.
