# Day 3 authoring record

## Scope and sources

The existing lesson covers canonical forms, grouping, two-way symbolic mappings, and ambiguous count serialization. Core problems are Group Anagrams, Isomorphic Strings, and Word Pattern. Research used `articles/01-getting-started/02-04-hashmap_intro.md` only to check map lookup and collision concepts. The article is reference-only; no prose, code, or visual assets from it are included in runtime content.

## Exact contracts and decisions

All demo strings are lowercase ASCII. Grouping input begins with a word count and then one word per line. The result joins words within a group with commas and groups in first-seen order with bars. This stable format makes grouping independently checkable even though the problem accepts arbitrary group order. Isomorphic input is two whitespace-delimited strings. Word Pattern input is one pattern line and one whitespace-delimited word sequence line. Boolean output uses lowercase `true` or `false`.

| Technique | Normal input | Result | Revealing input | Result |
| --- | --- | --- | --- | --- |
| Sorted anagram key | `4\nstone\ntones\nnote\neat\n` | `stone,tones|note|eat` | `3\nab\nba\naa\n` | `ab,ba|aa` |
| Count-vector key | same | same | `2\nabbbbbbbbbbb\naaaaaaaaaaab\n` | `abbbbbbbbbbb|aaaaaaaaaaab` |
| Pairwise bijection baseline | `aba\nred blue red\n` | `true` | `abb\nred red red\n` | `false` |
| Isomorphic two maps | `paper title\n` | `true` | `ab aa\n` | `false` |
| Word-pattern two maps | `aba\nred blue red\n` | `true` | `abb\nred red red\n` | `false` |

For grouping, before each word, the map associates each canonical key with precisely the indices of previously visited words in that class. A sorted key is equal exactly when letter multisets match; sorting each length-k word costs O(k log k). A 26-integer tuple records the same multiset in O(k) time for lowercase ASCII. Both groupers store O(nk) output text and O(nk) grouping information; the fixed vector uses O(1) temporary counters per word. The count-vector edge uses `abbbbbbbbbbb` (a=1,b=11) and `aaaaaaaaaaab` (a=11,b=1). Their delimited keys are `a:1,b:11` and `a:11,b:1`, so the executable result has two groups. Neither uses a mere hash value as an identity: dictionary equality checks keys after hashing. Never concatenate counts without separators: counts `[1,11]` and `[11,1]` both become `111`.

For isomorphism and pattern matching, after each accepted pair the forward and reverse maps are inverses over every symbol seen so far. A pair is valid only when both existing assignments agree, or neither side has an assignment. One map alone lets two distinct source symbols map to the same destination. Strings of unequal length, or a different number of words than pattern symbols, fail before mapping. Isomorphism is O(n) expected time and O(1) map space over a fixed byte alphabet. Word Pattern is O(n) expected map operations but O(total word bytes) storage; splitting the input also costs O(total word bytes). These programs intentionally use ASCII bytes and whitespace-delimited words; Unicode graphemes, punctuation-aware tokenization, and normalization need another explicit contract.

The pairwise baseline records each compared `(i,j)` pair in both source and target boxes. Its scene and explanation show whether equality agrees on the two sides, including the rejecting pair. Each case emits start, state-change/check events, and a terminal done event as JSONL, then the computed result. Variable values and event order are compared with all four executable programs. Each scene labels the source symbol, canonical key, or map assignment and the corresponding group/decision. Line mappings point to the event's real source line. Edge cases remain separate selector entries.

## Verification and review

On 2026-09-29, `python3 scripts/leetgrinder/verify_examples.py --days 3 --report /tmp/leetgrinder-day03.json` exited 0: five examples, 12 cases, and all 48 C++/Python/Java/Go runs passed. The verifier compared each observed event and variable list in order and each computed final result. `python3 scripts/leetgrinder/audit_lessons.py --days 3` exited 0 with one complete day and no duplicate links. `GOCACHE=/tmp/career-leetgrinder-go-cache go test ./internal/leetgrinder/...` passed under localhost-enabled sandbox approval; the restricted initial run could not bind test listeners in analysis and notify. `LEETGRINDER_DAYS=3 node tests/leetgrinder-lessons.browser.test.mjs` passed in Chromium, including both case selectors, code tabs, line highlights, SVG frames, and narrow viewport checks.

Semantic review checked the count collision edge by observation: the first word emitted `a:1,b:11`, the second `a:11,b:1`, and the result kept them in separate groups. The pairwise baseline recorded `(i=2,j=0)` as equal/equal in its normal case and `(i=1,j=0)` as unequal/equal in its rejecting case. Its baseline panel is linked once, while sorted-key grouping appears only in its worked section. The AlgoMonster article remains research-only; the runtime lesson and example files do not contain it or depend on its directory.
