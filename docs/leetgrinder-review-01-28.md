# Leetgrinder source review: days 1–28

Reviewed on 2026-09-27 against the lesson text in `internal/leetgrinder/lessons.templ`. I opened each source or its publisher/author record, checked the title and topic, and inspected the assigned excerpt. The first reading for every day has a distinct URL and takes at most ten minutes. Its existing `Optional` value is preserved because that field affects review scheduling. Every day also has a distinct research paper, author manuscript, or technical report. Course notes are labeled `reference`, not `paper`.

The `supports` text is intentionally narrow. It says what a source substantiates without implying that the lesson's examples or code were copied from it.

| Day | Source relevance checked | Limitations recorded |
| ---: | --- | --- |
| 1 | Open Data Structures covers chained-table operations and expected cost; Pagh–Rodler gives a research-grade dictionary with key-associated values and constant-time lookup. | Cuckoo hashing uses a different collision scheme. It is an extension beyond the lesson's chained table and does not supply the complement example. |
| 2 | Python's `Counter` documentation covers frequencies and multiplicity; Misra–Gries states the cancellation invariant for repeated elements. | Misra–Gries finds candidates under frequency bounds; explicit verification remains necessary when a majority is not guaranteed. |
| 3 | The Chicago page demonstrates canonical sorted-letter keys; the parameterized-matching paper defines a one-to-one symbol correspondence. | The paper studies indexed pattern matching, while the lesson uses the bijection idea in a small map check. |
| 4 | Python's set documentation and James Madison's multiset page distinguish membership from multiplicity; Nivasch's author page explains repeated-state cycle detection and links the paper. | Nivasch's stack detector is enrichment; the lesson's stored-state simulation is simpler and deterministic. |
| 5 | Princeton covers merge sort and merge invariants; Hoare's paper gives the original in-place partition procedure. | Hoare describes two-way quicksort partitioning. The lesson develops its three-way region invariant independently. |
| 6 | Goucher defines prefix computation; Kogge–Stone expresses carry propagation as a prefix recurrence. | The paper targets parallel hardware. The lesson uses the same dependency shape sequentially and adds its own mutation-safety examples. |
| 7 | Stanford supplies row/column indexing, rectangular shape, and nested traversal; Gustavson–Walker covers in-place transposition; UIC gives coordinate-transform equations. | These sources do not prescribe spiral traversal or zero marking. UIC's continuous transform needs an added translation when used to reason about finite array indices. |
| 8 | CP-Algorithms presents half-open binary-search invariants; Bentley derives correctness from a loop invariant. | Neither source supplies the lesson's exact trace cases; those are original checks of the stated invariant. |
| 9 | USACO introduces monotone integer predicates; Zimmermann's INRIA report specifies exact integer square root and remainder. | Zimmermann uses a faster divide-and-conquer algorithm rather than binary search. Overflow-safe division in the lesson's boundary predicate remains a lesson-specific engineering point. |
| 10 | Python `bisect` and the C++ standards paper define lower, upper, and equal-range boundaries; UPC derives staircase matrix elimination. | The successor wraparound rule and the lesson's concrete traces are separate applications of those boundaries. |
| 11 | Washington contrasts row-major indexing with row/column-sorted search; Dijkstra proves saddleback elimination; Princeton gives the distinct-key rotated-minimum algorithm. | The two matrix orders require different searches. The lesson still supplies its own examples and cautions about retaining a possible midpoint. |
| 12 | MIT's peak materials prove 1D peak finding by following a rising side; its Recitation 11 develops rotated-array search; Zhang formalizes the broader local-search problem on graphs. | Zhang is research context rather than the source of the array implementation. The lesson adds duplicate-heavy ambiguity and its possible linear worst case. |
| 13 | Topcoder explains binary search over a monotone answer predicate; Agarwal–Sharir–Toledo gives the parametric-search framework. | The paper's geometric machinery is outside scope. The lesson's ceiling division and candidate bounds need their own integer-safety reasoning. |
| 14 | Toronto's illustrated trace refreshes the binary-search invariant; de Berg–Thite studies ranks in sorted pair-combination matrices; Princeton supplies the feasibility checklist. | The paper concerns selection and I/O efficiency, not bouquet construction. It supports the pair-combination search setting; the lesson's three predicates remain separate derivations. |
| 15 | USACO proves opposite-end elimination on sorted sums; Berenbrink et al. define palindrome recognition and the longest-palindromic-substring problem in a streaming setting. | The paper's randomized streaming machinery is far deeper than normalization plus two pointers; it is research context only. |
| 16 | Open Data Structures establishes the valid array-prefix model; Katajainen–Pasanen defines stable in-place partitioning. | The paper's packed-word algorithm is intentionally excluded. The lesson uses a simpler read/write compaction when overwriting processed cells is safe. |
| 17 | Princeton introduces 3SUM and its complexity; Grønlund–Pettie gives modern 3SUM research context; Dartmouth derives sorted-pair elimination and its 3SUM extension. | The paper's subquadratic methods are not taught. The container-height proof and duplicate-skipping rules remain lesson derivations. |
| 18 | GSL describes fixed moving aggregates; Datar et al. formalizes sliding-window expiration in streams. | The paper develops approximate streaming summaries. The lesson keeps the exact fixed-width window in memory. |
| 19 | OI Wiki explains same-direction pointers and linear total movement; Braverman–Ostrovsky defines the sliding-window stream model. | Smooth histograms are approximate and do not justify shrinking an exact sum window with negative values; the lesson explicitly limits that monotonic argument to suitable inputs. |
| 20 | USACO derives subarray sums from prefix differences; Blelloch defines scan and its applications. | Blelloch does not supply the target-frequency-map trick. The lesson derives that lookup and the empty-prefix seed from the prefix identity. |
| 21 | CMU's scan page and Chatterjee–Blelloch–Zagha cover associative prefix operations and exclusive scan; the Go specification fixes signed-remainder behavior. | The scan paper supports prefix products, not the frequency-map choice. The lesson separately explains normalized modulo keys and when to store earliest indices versus counts. |
| 22 | Open Data Structures shows singly-linked pointer updates; Reynolds's open manuscript models disjoint list segments and includes an in-place reversal example. | Separation logic is proof machinery, not an implementation tutorial. The lesson explains reversal and stable merge in prose and emphasizes saving `next`. |
| 23 | Cornell establishes reference identity and traversal; Sedgewick–Szymanski–Yao states and analyzes the two-speed cycle detector. | The paper directly supports cycle detection but not head switching for intersection. The intersection proof and middle convention are lesson-specific consequences of path lengths. |
| 24 | Princeton visualizes the reversal invariant; Myreen proves reversal with separation logic; the CSES handbook explains cycle-entry reset. | None of these sources combines n-th-from-end, cycle entry, and palindrome restoration. The lesson composes those techniques and gives the sentinel/gap details. |
| 25 | Cornell shows sentinels and pointer rewiring; Salowe–Steiger formalizes stable unmerging. | Stable unmerging is broader than the lesson's tail-built partitions. Pair swapping and odd/even positional lists are derived from the same local rewiring discipline in the lesson. |
| 26 | Princeton covers stack APIs, delimiter matching, and expression evaluation; Dijkstra's report documents the operator-stack method. | The report predates the lesson's `MinStack` design. The minimum-aggregation invariant is explained and tested by the lesson itself. |
| 27 | Open Data Structures shows a two-sided representation with reversal and amortized rebalancing; Hood–Melville develops two-list queues. | `DualArrayDeque` is a deque, not the exact queue implementation. Hood–Melville's real-time schedule is enrichment beyond the lesson's amortized two-stack queue. |
| 28 | Erickson covers recursive contracts and exponentiation by squaring; Michie's two-page note introduces memoized Fibonacci. | Erickson's chapter is broader than the assigned excerpt. The lesson states the cost of naive Fibonacci and asks the learner to discuss memoization; Michie supplies the worked memoization example. |

## Mechanical checks

The manifest was checked for all of the following:

- exactly one record for each day 1 through 28;
- a unique document URL for every reading across the catalog;
- a first reading of ten minutes or less;
- a distinct `paper` URL for every day;
- no day above 30 total minutes;
- the original first-reading `Optional` values from the curriculum;
- nonempty `guidance`, `kind`, and `supports` fields on every reading.

The final replacements use direct university, author, standards-body, archive, or publisher pages that expose the assigned material without a sign-in prompt. I checked document identity as well as HTTP status; this caught endpoints that returned a denial payload or an unrelated document despite a successful status code.
