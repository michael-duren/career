# Day 1 authoring record: remainder frequency

## Scope and sources

This record covers `day-01-remainder-frequency`, the fixed array of 60 counters for song durations. The existing day 1 lesson and curriculum establish the gap: the lesson describes this technique, but its current examples only animate pair search and a value-to-index map. The new example gives the count-array technique its own runnable trace. The lesson owner will link it from the lesson.

The following local AlgoMonster files were consulted as research only:

- `/home/mduren/Documents/data/algomonster/articles/12-company-oas/01-10-pairs_of_songs.md`, “Explanation” and “Implementation”: the modulo-60 pairing rule and the need to count earlier songs.
- `/home/mduren/Documents/data/algomonster/articles/01-getting-started/02-04-hashmap_intro.md`, “Hash Function” and “Hash Tables”: background on key-to-value lookup. The example uses a fixed array because the keys are exactly the integers 0 through 59.

No article prose, code, image, diagram, or input was copied into the site. The source paths here document research and are not runtime dependencies. The existing public References footer remains the bibliography for the lesson.

## Input and behavior

Each program reads the song count followed by that many nonnegative integer durations. The first line of the normal input is `5`; its durations are `30 20 150 100 40`. The result is `3`, from index pairs `(0, 2)`, `(1, 3)`, and `(1, 4)`. The edge input is `3` followed by `60 60 60`. Its result is `3`, since each new zero-remainder song pairs with every earlier zero-remainder song. Each program emits one JSONL event per frame and a final `{"result":"3"}` record.

The invariant is that `counts[r]` contains only earlier songs with remainder `r`. At each song, the algorithm reads bucket `(60 - remainder) % 60`, adds its count to the answer, and then increments the current remainder bucket. The second modulo is essential when the remainder is zero: the complement is bucket zero. Querying before insertion also prevents a song from pairing with itself. The scan takes O(n) time and O(1) auxiliary space for 60 counters, excluding input and trace output.

## Trace and drawing decisions

Both cases start with `start`, then emit `lookup`, `count`, and `store` for each song, and finish with `done`. The ordered variables come from the Python program's observed JSONL output. The verifier checks those records against all four compiled or interpreted programs. Source line maps point to the actual event emit calls, and each variant's displayed algorithm range runs from `start` through `done`.

The scene shows songs in input order above frequency buckets. For the normal input it displays six selected buckets `0, 10, 20, 30, 40, 50`; every omitted bucket remains zero. The edge input displays bucket zero. Each bucket's label gives its remainder and current count. An arrow points from the active song to the bucket being read or updated. A separate label shows the running pair total. The `count` frame still shows the old frequency array, making the lookup-before-store order visible. The zero-remainder edge frames show totals increasing by zero, one, then two, for a final total of three.

## Verification and review

On 2026-09-29, `python3 scripts/leetgrinder/verify_examples.py --days 1` exited 0. The verifier ran all day 1 examples in C++, Python, Java, and Go, including both cases of this example, and compared each JSONL event, ordered variable list, and result against the authored frames. Go source is stored as `main.go.txt`; the verifier stages it as `main.go` to compile it.

A visual reviewer should check that the edge-case bucket zero remains readable at a narrow viewport and that the arrow and count label communicate the selected bucket without relying on color alone.
