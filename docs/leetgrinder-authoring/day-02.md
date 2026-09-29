# Day 2 authoring record

## Scope and research

The core assignments are Ransom Note, Brick Wall, and Majority Element. The existing lesson also distinguishes presence, multiplicity, and first occurrence; limits fixed arrays to a bounded alphabet; and requires a verification pass for a majority candidate when existence is not guaranteed. Day 1 already introduces fixed letter counts, so this day applies the same state idea to available supplies, wall boundaries, and cancellation.

Research: `/home/mduren/Documents/data/algomonster/articles/01-getting-started/02-04-hashmap_intro.md`, specifically "Hash Tables", "Efficiency", and the frequency-counter demonstration. It covers key-to-count maps and expected lookup cost. It does not cover brick edges or Boyer-Moore. The existing public references in `readings.go` point to Python Counter, Misra and Gries, the Boyer-Moore description, and C++ counting; those are optional follow-up reading, not source material for the programs. All prose, code, inputs, and SVG scenes here are original. The article directory is absent from runtime assets.

## Inputs, exact results, and invariants

Strings are lowercase ASCII. Each Ransom Note program reads two newline-terminated tokens: note, then magazine. The output is lowercase `true` or `false`. Each Brick Wall program reads row count, then one line per row with brick count followed by positive widths. Rows share a total width. Its output is the decimal minimum crossed-brick count. Each Majority Element program reads count and then signed integers; its output is the majority value or `none`. This standalone contract permits no majority, unlike LeetCode 169's guarantee.

| Example | Normal stdin | Output | Revealing edge stdin | Output |
| --- | --- | --- | --- | --- |
| `ransom-baseline` | `cacao\ncaacao\n` | `true` | `cocoa\ncacao\n` | `false` |
| `ransom-counts` | `cacao\ncaacao\n` | `true` | `cocoa\ncacao\n` | `false` |
| `brick-baseline` | `3\n3 1 1 2\n3 2 1 1\n3 1 1 2\n` | `0` | `2\n1 4\n1 4\n` | `2` |
| `brick-edges` | `3\n3 1 1 2\n3 2 1 1\n3 1 1 2\n` | `0` | `2\n1 4\n1 4\n` | `2` |
| `majority-baseline` | `5\n2 1 2 3 2\n` | `2` | `4\n1 2 3 4\n` | `none` |
| `majority-vote` | `5\n2 1 2 3 2\n` | `2` | `4\n1 2 3 4\n` | `none` |

For Ransom Note, the baseline marks each magazine position at most once; before each note letter, marked positions are exactly the supplies consumed by earlier letters. It scans the magazine again for each note letter. The improved version first fills 26 buckets, then decrements the requested letter's bucket. After a prefix of the note, bucket `c` equals magazine copies of `c` minus copies consumed in that prefix. A negative bucket proves a shortage. With note length `m` and magazine length `n`, baseline time is O(mn), auxiliary space O(n); counts take O(m+n) time and O(1) auxiliary space. The `cocoa` edge requires two `o` letters, while `cacao` supplies one.

For Brick Wall, the baseline tests each integer interior coordinate from 1 to wall width minus 1. It scans each row's cumulative boundaries and counts rows without a boundary there. Coordinate zero and the wall end are forbidden because a cut along the outside is not a valid interior line. The improved version visits every internal cumulative edge once; after any prefix of rows, `edges[x]` is the number of those rows with an internal boundary at x. The minimum crossed count is `rowCount - max(edges[x])`; zero counted edges means every row is crossed. With `B` total bricks and integer wall width `W`, baseline time is O(WB), auxiliary space O(1); improved expected time and space are O(B) for a hash map. Width addition requires a sufficiently wide integer type in a production adaptation.

For Majority Element, the baseline counts occurrences of each tested value across the full array and returns the first count greater than `n/2`. Before a tested value, no earlier tested value was a majority. It takes O(n²) time and O(1) auxiliary space. Boyer-Moore keeps a candidate and a vote balance. After each prefix, all discarded elements can be partitioned into unequal pairs; the balance counts identical, unpaired candidate values. A true majority cannot be eliminated by pair cancellation. The second pass counts actual candidate occurrences and returns it only if the count exceeds `n/2`. The vote and verification passes take O(n) time and O(1) auxiliary space. The edge case has no majority, so cancellation alone cannot certify a result.

## Planned trace and drawing decisions

Every case begins with `start` before a mutation and ends with `done`. Ransom baseline records each attempted magazine position and each consumed match; the count version records each supply addition and each requested decrement. Its scene shows note characters, magazine characters, and relevant count buckets, including the negative `o` bucket. Brick baseline records each candidate coordinate and each row test. Its scene shows rows as width-scaled brick rectangles with an interior cut marker. Brick edges records each internal boundary and a final best coordinate; its scene shows each boundary count and omits the wall end from the count boxes. Majority baseline records each candidate's full-array tally. The vote version records each vote adjustment and each verification match, with a candidate/balance strip and crossed-out unequal pairs. State labels remain readable without color.

Frame variables will come from observed JSONL output, in stable order and identical across C++, Python, Java, and Go. Each frame maps to the physical line that performs its event's operation or emits its observed state. The source range includes every event line. A visual reviewer should inspect the wall-end exclusion, negative supply bucket, and vote cancellation/verification transition at narrow width.

## Verification and review

On 2026-09-29, `python3 scripts/leetgrinder/verify_examples.py --days 2 --report /tmp/leetgrinder-day02.json` exited 0. The JSON report marks all six examples and all 48 language/case runs passed. The programs produced 139 event frames across 12 cases; the verifier compared every ordered variable list and final result with `example.json`. Go sources are stored as `main.go.txt` and staged as `main.go` by the verifier. The trace line numbers point to physical `emit` calls in each program's algorithm range.

`GOCACHE=/tmp/career-leetgrinder-go-cache go test ./internal/leetgrinder` exited 0. This loaded the embedded lesson, checked every scene and line range, and ran the package tests. `python3 scripts/leetgrinder/audit_lessons.py --days 2` exited 0 and reported one complete day, six examples, 12 cases, and 48 language runs.

`LEETGRINDER_DAYS=2 node tests/leetgrinder-lessons.browser.test.mjs` exited 0 after sandbox escalation allowed its local Go preview server and Chromium to start. The browser swept all six linked examples and both cases at a 360-pixel viewport, stepped every frame, checked Python line highlights, checked each language's first-frame highlight, and found no page-level overflow or page errors. The initial sandboxed attempt failed with `spawnSync go EPERM`; the direct preview launch also failed to bind a localhost socket. Both are sandbox restrictions, not lesson failures.

Review decisions: the baseline traces are separate because their repeated scans have different steps from the improved methods. The Ransom Note edge case stops at the first negative `o` bucket; a failed request needs no further work. The wall's no-gap case has only start and done frames in the improved example, faithfully showing that no internal edge event occurs. The majority example verifies even though LeetCode 169 guarantees a majority, so the reusable algorithm handles inputs without that promise. Scenes use ASCII letters and small integer widths as stated; adaptation to Unicode or huge widths needs a different input contract and numeric type.

Semantic review correction: the majority baseline now emits `candidate`, `tally`, and `answer` instead of vote-specific state. Its start, test, and terminal explanations and SVG boxes describe only full-array counts. The normal terminal frame reports tally 3 and answer 2. After this correction, the day 2 example verifier, Go package tests, and day 2 audit all exited 0 again.
