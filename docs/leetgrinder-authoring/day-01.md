# Day 1 authoring record

## Scope

Day 1 teaches Two Sum (easy) end to end: exhaustive pair search, then one-pass complement lookup with a value-to-index map and the look-before-store rule. It then teaches counting pairs with a value-to-count map and counting letters with a 26-slot array (finding repeated letters).

Valid Anagram and Pairs of Songs With Total Durations Divisible by 60 are left to the learner. Each has a "Your turn" section that restates the problem and gives a process hint only. The former remainder-bucket and anagram-count examples were removed because they were complete solutions. Two Sum remains the worked problem by agreement.

The hashmap introduction in the research corpus was consulted as research only. All prose, programs, cases, and diagrams are original.

## Examples

| Example | Case | stdin | Output |
| --- | --- | --- | --- |
| baseline | normal / edge / duplicate | `3 9\n8 3 6\n`, `1 8\n4\n`, `2 8\n4 4\n` | `[1,2]`, `[]`, `[0,1]` |
| complement-lookup | normal / duplicate / single | same inputs | `[1,2]`, `[0,1]`, `[]` |
| pair-counts | normal | `5 6\n1 5 3 3 3\n` | `4` |
| pair-counts | edge-all-equal | `4 4\n2 2 2 2\n` | `6` |
| repeated-letters | normal / edge-no-repeats | `banana`, `abc` | `an`, `none` |

Pair counting adds the count of earlier complements before recording the current value, so each pair is counted once by its later position. The all-equal edge shows the running total growing by 0, 1, 2, 3.

## Writing style

Problem-first, plain language: state what the problem gives and asks, work a tiny example, show the obvious approach, then the improvement. Terms are defined when introduced. Step captions describe the concrete values at that step.

## Verification

`python3 scripts/leetgrinder/verify_examples.py --days 1` passes for all four examples in C++, Python, Java, and Go. Go tests and the day 1 browser sweep pass.
