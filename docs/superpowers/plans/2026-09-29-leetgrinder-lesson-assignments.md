# Lesson assignments for the Leetgrinder upgrade

Companion to [the implementation plan](2026-09-29-leetgrinder-lesson-upgrade.md). Read its lesson contract and example schema before taking a day.

The AlgoMonster articles listed here are research pointers only. Do not transfer their text, code, image files, or diagram designs into the site. Create original lesson content and SVGs; the built site must have no dependency on the local article directory.

All source paths below are relative to `/home/mduren/Documents/data/algomonster/`. These are verified local article paths, not proof that every required subtopic is covered. Read the actual article before attributing a claim. Where a source covers only a related idea, use the existing lesson and its bottom references to fill the gap and record the limitation in the day authoring note. Do not fabricate a source match.

The existing walkthrough below is a seed to retain or improve, not a complete specification for every animation. Each semicolon-separated technique requires its own runnable, four-language example and two trace cases. Before coding, the worker writes the exact inputs, expected outputs, invariants, and visible steps in `docs/leetgrinder-authoring/day-NN.md`; the reviewer checks them against this assignment and the existing lesson.

Mixed-practice days additionally derive their technique inventory from that day’s actual core assignments in `curriculum.go`. This avoids revealing or inventing a pattern based only on the broad round title. They reuse existing examples only when input, invariant, and explanation suit the assigned problems; all remain inside the review disclosure.

## Day 01: Hash lookup and honest baselines

- Own: `internal/leetgrinder/lessons/day-01.json`, `internal/leetgrinder/examples/day-01/`, `docs/leetgrinder-authoring/day-01.md`.

- Required demonstrations: Exhaustive pair search; complement lookup before insertion; frequency lookup.

- Source articles:
  - `articles/01-getting-started/02-04-hashmap_intro.md`

- Existing walkthrough seed: For values [8, 3, 6] and target 9, record 8, then 3. At 6, the required value is 3, which already has an earlier index. A set detects membership; a frequency map distinguishes one occurrence from two.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 02: Counting as state

- Own: `internal/leetgrinder/lessons/day-02.json`, `internal/leetgrinder/examples/day-02/`, `docs/leetgrinder-authoring/day-02.md`.

- Required demonstrations: Character counters; cumulative brick-edge counts; Boyer–Moore cancellation with verification.

- Source articles:
  - `articles/01-getting-started/02-04-hashmap_intro.md`

- Existing walkthrough seed: The letters in "cacao" have counts c=2, a=2, o=1. Removing the letters in "cocoa" would require two o characters, so the count becomes negative and exposes the shortage.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 03: Canonical forms and bijections

- Own: `internal/leetgrinder/lessons/day-03.json`, `internal/leetgrinder/examples/day-03/`, `docs/leetgrinder-authoring/day-03.md`.

- Required demonstrations: Sorted and count-vector anagram keys; forward/reverse bijection maps.

- Source articles:
  - `articles/01-getting-started/02-04-hashmap_intro.md`

- Existing walkthrough seed: "stone" and "tones" share the sorted key "enost". For pattern "aba" and words "red blue red", both directions agree. "red red red" violates the reverse map because two pattern symbols would share one word.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 04: Sets, multisets, and repeated states

- Own: `internal/leetgrinder/lessons/day-04.json`, `internal/leetgrinder/examples/day-04/`, `docs/leetgrinder-authoring/day-04.md`.

- Required demonstrations: Multiset consumption; repeated-state detection; row/column/box sets.

- Source articles:
  - `articles/01-getting-started/02-04-hashmap_intro.md`
  - `articles/03-two-pointers/07-02-linked_list_cycle.md`

- Existing walkthrough seed: Intersect [4,4,7] with [4,4,4,9]. Set intersection contains one 4; multiset intersection contains two. For the transition 1→4→2→1, the second visit to 1 proves repetition without predicting how many steps remain.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 05: Sorting and loop invariants

- Own: `internal/leetgrinder/lessons/day-05.json`, `internal/leetgrinder/examples/day-05/`, `docs/leetgrinder-authoring/day-05.md`.

- Required demonstrations: Insertion-sort invariant; merge operation and recursion; quicksort partition invariant.

- Source articles:
  - `articles/01-getting-started/03-01-sorting_intro.md`
  - `articles/01-getting-started/03-02-advanced_sorting.md`

- Existing walkthrough seed: Merge [2,9] and [1,5,8] by taking 1,2,5,8,9. When filling an array from the back, taking the larger remaining value protects unread input near the front.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 06: Array transformations and carries

- Own: `internal/leetgrinder/lessons/day-06.json`, `internal/leetgrinder/examples/day-06/`, `docs/leetgrinder-authoring/day-06.md`.

- Required demonstrations: In-place array movement; digit carry propagation; index read/write dependencies.

- Source articles:
  - `articles/01-getting-started/04-01-simulation_intro.md`
  - `articles/03-two-pointers/03-04-move_zeros.md`

- Existing walkthrough seed: For [3,1,4], running totals are [3,4,8]. Adding one to [2,9,9] produces a carry through both trailing nines and stops at the leading 2, giving [3,0,0]. An all-nine input needs an extra position.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 07: Matrix coordinates and mutation

- Own: `internal/leetgrinder/lessons/day-07.json`, `internal/leetgrinder/examples/day-07/`, `docs/leetgrinder-authoring/day-07.md`.

- Required demonstrations: Coordinate transforms; matrix rotation; marker-based row/column mutation.

- Source articles:
  - `articles/07-graph/03-01-matrix_as_graph.md`
  - `articles/01-getting-started/04-01-simulation_intro.md`

- Existing walkthrough seed: In a 3 by 3 clockwise rotation, coordinate (0,1) moves to (1,2). For a two-row rectangle, a spiral visits the top row, then the right edge, then the bottom row in reverse, with no remaining left edge.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 08: Binary search intervals

- Own: `internal/leetgrinder/lessons/day-08.json`, `internal/leetgrinder/examples/day-08/`, `docs/leetgrinder-authoring/day-08.md`.

- Required demonstrations: Closed and half-open search intervals; found and absent targets.

- Source articles:
  - `articles/02-binary-search/01-01-binary_search_intro.md`
  - `articles/02-binary-search/01-02-binary_search_boundary.md`

- Existing walkthrough seed: Search [2,6,11,15] for 11. The initial midpoint at index 2 matches. For insertion of 8, the boundary ends at index 2 even though no equal value exists.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 09: Monotone predicates on integers

- Own: `internal/leetgrinder/lessons/day-09.json`, `internal/leetgrinder/examples/day-09/`, `docs/leetgrinder-authoring/day-09.md`.

- Required demonstrations: First-true integer predicate; square root with overflow-safe comparison.

- Source articles:
  - `articles/02-binary-search/02-01-binary-search-monotonic.md`
  - `articles/02-binary-search/02-04-sqrt.md`

- Existing walkthrough seed: For the predicate x²≥30 on nonnegative integers, 5 is false and 6 is true. The first true candidate is 6; the integer floor square root is therefore 5 because 30 is not a square.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 10: Lower and upper boundaries

- Own: `internal/leetgrinder/lessons/day-10.json`, `internal/leetgrinder/examples/day-10/`, `docs/leetgrinder-authoring/day-10.md`.

- Required demonstrations: Lower bound; upper bound; duplicate ranges; staircase negative counting.

- Source articles:
  - `articles/02-binary-search/02-02-binary_search_first_element_not_smaller_than_target.md`
  - `articles/02-binary-search/02-03-binary_search_duplicates.md`

- Existing walkthrough seed: In [1,3,3,3,8], lower_bound(3)=1 and upper_bound(3)=4, so there are three copies. A missing target such as 4 has equal boundaries at index 4.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 11: Search structured matrices and rotations

- Own: `internal/leetgrinder/lessons/day-11.json`, `internal/leetgrinder/examples/day-11/`, `docs/leetgrinder-authoring/day-11.md`.

- Required demonstrations: Flattened matrix search; staircase matrix search; rotated minimum.

- Source articles:
  - `articles/02-binary-search/03-01-min_in_rotated_sorted_array.md`
  - `articles/02-binary-search/01-01-binary_search_intro.md`

- Existing walkthrough seed: Rows [1,4] and [2,7] are sorted by row and column, but their flattened sequence is not sorted. For rotated [10,14,2,5,8], midpoint 2 is no larger than the final 8, so the minimum remains at or left of the midpoint.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 12: Rotated search and local peaks

- Own: `internal/leetgrinder/lessons/day-12.json`, `internal/leetgrinder/examples/day-12/`, `docs/leetgrinder-authoring/day-12.md`.

- Required demonstrations: Rotated target search; duplicate ambiguity; local peak elimination.

- Source articles:
  - `articles/02-binary-search/03-01-min_in_rotated_sorted_array.md`
  - `articles/02-binary-search/03-02-peak_of_mountain_array.md`

- Existing walkthrough seed: In [12,16,3,5,9], the right half beginning at 3 is sorted. A target of 16 forces the other side. In [2,6,4], the slope rises then falls, so 6 is a peak even without global sorting.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 13: Search the answer with feasibility

- Own: `internal/leetgrinder/lessons/day-13.json`, `internal/leetgrinder/examples/day-13/`, `docs/leetgrinder-authoring/day-13.md`.

- Required demonstrations: Capacity feasibility; minimum speed/divisor search; monotonicity counterexample.

- Source articles:
  - `articles/02-binary-search/04-01-newspapers_split.md`

- Existing walkthrough seed: For piles [5,8] and five hours, speed 3 takes two plus three hours and is feasible. Speed 2 takes three plus four hours and fails. Faster speeds cannot require more time, which supplies the monotonicity proof.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 14: Binary search review and pair counts

- Own: `internal/leetgrinder/lessons/day-14.json`, `internal/leetgrinder/examples/day-14/`, `docs/leetgrinder-authoring/day-14.md`.

- Required demonstrations: Boundary review; sorted pair counting without double counting.

- Source articles:
  - `articles/02-binary-search/05-01-binary-search-speedrun.md`
  - `articles/03-two-pointers/04-02-two_sum_sorted.md`

- Existing walkthrough seed: For sorted potions [2,5,9] and spell 4, a success threshold of 20 first holds at potion 5. Every potion after that also succeeds, so the answer is the suffix length two.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 15: Opposing pointers

- Own: `internal/leetgrinder/lessons/day-15.json`, `internal/leetgrinder/examples/day-15/`, `docs/leetgrinder-authoring/day-15.md`.

- Required demonstrations: Palindrome comparisons; sorted pair sum; container limiting-height elimination.

- Source articles:
  - `articles/03-two-pointers/04-01-opposite_direction_family_map.md`
  - `articles/03-two-pointers/04-03-valid_palindrome.md`
  - `articles/03-two-pointers/04-04-container_with_most_water.md`

- Existing walkthrough seed: For sorted [1,4,6,10] and target 10, endpoints sum to 11, so move the right pointer. Now 1+6=7, so move the left pointer; 4+6=10 succeeds.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 16: Read and write pointers

- Own: `internal/leetgrinder/lessons/day-16.json`, `internal/leetgrinder/examples/day-16/`, `docs/leetgrinder-authoring/day-16.md`.

- Required demonstrations: Stable compaction; unique-prefix invariant; move-zero preservation.

- Source articles:
  - `articles/03-two-pointers/03-01-same_direction_family_map.md`
  - `articles/03-two-pointers/03-02-remove_duplicates.md`
  - `articles/03-two-pointers/03-04-move_zeros.md`

- Existing walkthrough seed: Removing 5 from [5,2,5,7] writes 2 into index 0 and 7 into index 1, producing the valid prefix [2,7]. The rest of the backing array is irrelevant unless the task requires filling it.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 17: Eliminate pairs without skipping answers

- Own: `internal/leetgrinder/lessons/day-17.json`, `internal/leetgrinder/examples/day-17/`, `docs/leetgrinder-authoring/day-17.md`.

- Required demonstrations: Three-sum anchor and two pointers; duplicate skipping; closest-sum updates.

- Source articles:
  - `articles/03-two-pointers/04-02-two_sum_sorted.md`
  - `articles/03-two-pointers/09-03-two_pointers_synthesis.md`

- Existing walkthrough seed: With sorted [-3,-1,2,4,6], freezing -3 and targeting a total of 3 leaves a pair target of 6. The pair 2 and 4 works. For heights 4 and 9 at distance 5, the area is 20 and the height-4 side limits it.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 18: Fixed-size windows

- Own: `internal/leetgrinder/lessons/day-18.json`, `internal/leetgrinder/examples/day-18/`, `docs/leetgrinder-authoring/day-18.md`.

- Required demonstrations: Fixed-window sum; frequency-window anagrams; outgoing/incoming update order.

- Source articles:
  - `articles/03-two-pointers/05-01-sliding_window_family_map.md`
  - `articles/03-two-pointers/05-02-subarray_sum_fixed.md`
  - `articles/03-two-pointers/05-03-find_all_anagrams.md`

- Existing walkthrough seed: For [6,1,5,2] and width 2, sums are 7,6,7. Moving from the first window to the second subtracts 6 and adds 5. To compare an average with 3, compare the sum with 2×3 instead of dividing.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 19: Variable windows and validity

- Own: `internal/leetgrinder/lessons/day-19.json`, `internal/leetgrinder/examples/day-19/`, `docs/leetgrinder-authoring/day-19.md`.

- Required demonstrations: Longest valid window; shortest covering window; multiplicity and validity counters.

- Source articles:
  - `articles/03-two-pointers/05-04-subarray_sum_longest.md`
  - `articles/03-two-pointers/05-05-longest_substring_without_repeating_characters.md`
  - `articles/03-two-pointers/09-01-minimum_window_substring.md`

- Existing walkthrough seed: For positive [2,1,4,2] and target 6, the window [2,1,4] is valid. Removing 2 makes it invalid, but after adding the last 2 the window [4,2] reaches the target with length two.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 20: Prefix sums and frequency maps

- Own: `internal/leetgrinder/lessons/day-20.json`, `internal/leetgrinder/examples/day-20/`, `docs/leetgrinder-authoring/day-20.md`.

- Required demonstrations: Prefix differences; zero-prefix initialization; prefix-frequency counting with negative values.

- Source articles:
  - `articles/03-two-pointers/06-01-prefix_sum_intro.md`
  - `articles/03-two-pointers/06-02-subarray_sum.md`
  - `articles/03-two-pointers/06-03-range_sum_query_immutable.md`

- Existing walkthrough seed: For [3,-2,4], prefixes are [0,3,1,5]. The last two entries sum to 5-3=2. If the target is zero, a repeated prefix indicates a zero-sum range between its occurrences.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 21: Prefix products and modular state

- Own: `internal/leetgrinder/lessons/day-21.json`, `internal/leetgrinder/examples/day-21/`, `docs/leetgrinder-authoring/day-21.md`.

- Required demonstrations: Prefix/suffix products with zeros; modular prefix equivalence and negative normalization.

- Source articles:
  - `articles/03-two-pointers/06-04-product_of_array_except_self.md`
  - `articles/03-two-pointers/06-01-prefix_sum_intro.md`

- Existing walkthrough seed: For [2,0,5], left products before each index are [1,2,0] and right products after each index are [0,5,1]. Their pairwise products are [0,10,0]. Modulo 4, prefixes 3 and 11 share a remainder, so their difference is divisible by 4.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 22: Linked nodes and local rewiring

- Own: `internal/leetgrinder/lessons/day-22.json`, `internal/leetgrinder/examples/day-22/`, `docs/leetgrinder-authoring/day-22.md`.

- Required demonstrations: Iterative reversal; saved successor; pairwise/local rewiring.

- Source articles:
  - `articles/01-getting-started/02-01-basic_dsa.md`
  - `articles/03-two-pointers/03-05-remove_nth_from_end.md`

- Existing walkthrough seed: For a→b→c, save b, set a.next to nil, then advance. On the next step save c and set b.next to a. Losing b before saving it would make the suffix unreachable.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 23: Fast and slow references

- Own: `internal/leetgrinder/lessons/day-23.json`, `internal/leetgrinder/examples/day-23/`, `docs/leetgrinder-authoring/day-23.md`.

- Required demonstrations: Middle selection for even/odd length; cycle detection; entry recovery.

- Source articles:
  - `articles/03-two-pointers/07-01-fast_slow_family_map.md`
  - `articles/03-two-pointers/03-03-middle_of_linked_list.md`
  - `articles/03-two-pointers/07-02-linked_list_cycle.md`

- Existing walkthrough seed: A five-node list puts the slow pointer at the third node when the fast pointer reaches the end. Two lists containing separate nodes with value 7 do not intersect; sharing the exact tail node does.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 24: Gaps, cycles, and half-list comparisons

- Own: `internal/leetgrinder/lessons/day-24.json`, `internal/leetgrinder/examples/day-24/`, `docs/leetgrinder-authoring/day-24.md`.

- Required demonstrations: Fixed node gap; cycle entry; reverse-half palindrome comparison and restoration.

- Source articles:
  - `articles/03-two-pointers/03-05-remove_nth_from_end.md`
  - `articles/03-two-pointers/07-02-linked_list_cycle.md`
  - `articles/03-two-pointers/04-03-valid_palindrome.md`

- Existing walkthrough seed: To remove the second node from the end of a→b→c→d, keep a two-node gap until the leading pointer reaches the end. The trailing position identifies c; a predecessor reference lets the link bypass it.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 25: Sentinels and stable list partitions

- Own: `internal/leetgrinder/lessons/day-25.json`, `internal/leetgrinder/examples/day-25/`, `docs/leetgrinder-authoring/day-25.md`.

- Required demonstrations: Dummy-head deletion; stable partition into two chains; stable sorted merge.

- Source articles:
  - `articles/01-getting-started/02-01-basic_dsa.md`
  - `articles/03-two-pointers/03-05-remove_nth_from_end.md`

- Existing walkthrough seed: For a→b→c→d, swapping the first pair produces b→a→c→d. The predecessor for the next pair is a, not b. Separating odd and even positions yields a→c and b→d before joining them.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 26: Stacks as unfinished work

- Own: `internal/leetgrinder/lessons/day-26.json`, `internal/leetgrinder/examples/day-26/`, `docs/leetgrinder-authoring/day-26.md`.

- Required demonstrations: Bracket matching; token/operator work stack; nested-expression evaluation.

- Source articles:
  - `articles/01-getting-started/02-02-stack_intro.md`
  - `articles/11-miscellaneous/02-02-basic_calculator.md`

- Existing walkthrough seed: For postfix "7 2 - 3 *", pop 2 then 7, compute 7-2=5, then multiply 5 by 3. Reversing the operand order would silently break subtraction and division.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 27: Implement one interface with another

- Own: `internal/leetgrinder/lessons/day-27.json`, `internal/leetgrinder/examples/day-27/`, `docs/leetgrinder-authoring/day-27.md`.

- Required demonstrations: Queue via two stacks and amortized transfers; stack via queues.

- Source articles:
  - `articles/01-getting-started/02-02-stack_intro.md`
  - `articles/01-getting-started/02-03-queue_intro.md`

- Existing walkthrough seed: Enqueue a,b,c into an input stack. Transferring all three gives an output stack whose top is a. After dequeuing a, enqueue d into the input stack; b and c still leave before d.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 28: Recursion and smaller subproblems

- Own: `internal/leetgrinder/lessons/day-28.json`, `internal/leetgrinder/examples/day-28/`, `docs/leetgrinder-authoring/day-28.md`.

- Required demonstrations: Recursive contract and base case; call stack unwinding; split/combine.

- Source articles:
  - `articles/04-depth-first-search/01-01-recursion_intro.md`
  - `articles/11-miscellaneous/04-01-divide_and_conquer_intro.md`

- Existing walkthrough seed: To compute 3⁵, compute 3²=9 from a smaller exponent, square to 81, and multiply by 3 to get 243. For an even exponent, no extra factor is needed. The exponent reaches zero, where the answer is one.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 29: Tree recursion contracts

- Own: `internal/leetgrinder/lessons/day-29.json`, `internal/leetgrinder/examples/day-29/`, `docs/leetgrinder-authoring/day-29.md`.

- Required demonstrations: Height return contract; balance sentinel propagation; empty subtree.

- Source articles:
  - `articles/04-depth-first-search/02-01-dfs_on_trees_intro.md`
  - `articles/04-depth-first-search/02-02-tree_max_depth.md`
  - `articles/04-depth-first-search/02-04-balanced_binary_tree.md`

- Existing walkthrough seed: A root with only a left child has minimum depth two, not one. Taking the minimum of zero for the missing right child and one for the left child would give the wrong answer.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 30: Compare tree structure

- Own: `internal/leetgrinder/lessons/day-30.json`, `internal/leetgrinder/examples/day-30/`, `docs/leetgrinder-authoring/day-30.md`.

- Required demonstrations: Paired-tree comparison; mirror comparison; subtree match versus traversal.

- Source articles:
  - `articles/04-depth-first-search/02-05-subtree_of_another_tree.md`
  - `articles/04-depth-first-search/02-06-invert_binary_tree.md`

- Existing walkthrough seed: A root 4 with left child 2 differs from a root 4 with right child 2, even though both contain the same values. A mirror comparison pairs the first tree's left child with the second tree's right child.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 31: Paths and branch-local state

- Own: `internal/leetgrinder/lessons/day-31.json`, `internal/leetgrinder/examples/day-31/`, `docs/leetgrinder-authoring/day-31.md`.

- Required demonstrations: Root-to-leaf sum; path push/pop; branch-local state versus shared result.

- Source articles:
  - `articles/05-backtracking/01-01-dfs_with_states.md`
  - `articles/04-depth-first-search/02-03-visible_tree_node.md`

- Existing walkthrough seed: For root 6 with a left path 2→5, the root-to-leaf sum is 13. Reaching node 2 with remaining target 5 is not success because node 2 is not a leaf.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 32: Tree breadth-first search

- Own: `internal/leetgrinder/lessons/day-32.json`, `internal/leetgrinder/examples/day-32/`, `docs/leetgrinder-authoring/day-32.md`.

- Required demonstrations: Level boundary queue; rightmost node; first-leaf minimum depth.

- Source articles:
  - `articles/06-breadth-first-search/01-01-bfs_intro.md`
  - `articles/06-breadth-first-search/02-01-binary_tree_level_order_traversal.md`
  - `articles/06-breadth-first-search/02-03-binary_tree_right_side_view.md`

- Existing walkthrough seed: A root with children 3 and 8 and grandchildren 1 and 5 has levels [root], [3,8], [1,5]. The rightmost visible value of a level is the last node processed when children are enqueued left before right.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 33: Binary search tree bounds

- Own: `internal/leetgrinder/lessons/day-33.json`, `internal/leetgrinder/examples/day-33/`, `docs/leetgrinder-authoring/day-33.md`.

- Required demonstrations: Ancestor bounds; BST search/insertion; invalid deep descendant.

- Source articles:
  - `articles/04-depth-first-search/03-01-bst_intro.md`
  - `articles/04-depth-first-search/03-02-valid_bst.md`
  - `articles/04-depth-first-search/03-03-insert_into_bst.md`

- Existing walkthrough seed: A root 10 with left child 5 and that child's right child 12 violates the root's upper bound, even though 12 is greater than its parent 5. A local parent-child check misses this error.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 34: Ancestors and ordered traversal

- Own: `internal/leetgrinder/lessons/day-34.json`, `internal/leetgrinder/examples/day-34/`, `docs/leetgrinder-authoring/day-34.md`.

- Required demonstrations: BST LCA; general-tree LCA; iterative inorder and kth visitation.

- Source articles:
  - `articles/04-depth-first-search/03-04-lowest_common_ancestor_on_bst.md`
  - `articles/04-depth-first-search/04-03-lowest_common_ancestor.md`

- Existing walkthrough seed: If one target is in the left subtree and the other is in the right, the current node is their lowest common ancestor. If the current node is itself a target, it can also be the ancestor of the other.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 35: Return local facts, update global answers

- Own: `internal/leetgrinder/lessons/day-35.json`, `internal/leetgrinder/examples/day-35/`, `docs/leetgrinder-authoring/day-35.md`.

- Required demonstrations: Subtree height versus diameter; downward gain versus complete path; negative values.

- Source articles:
  - `articles/04-depth-first-search/02-04-balanced_binary_tree.md`
  - `articles/09-dynamic-prog/11-01-tree_dp_intro.md`

- Existing walkthrough seed: If a node's left and right subtree heights are 3 and 2, a path through it spans five edges. The parent can extend only one side, so this node returns height four, not the full diameter.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 36: Decision trees and reversible choices

- Own: `internal/leetgrinder/lessons/day-36.json`, `internal/leetgrinder/examples/day-36/`, `docs/leetgrinder-authoring/day-36.md`.

- Required demonstrations: Choose/explore/undo; letter Cartesian product; branch ownership.

- Source articles:
  - `articles/05-backtracking/01-02-backtracking.md`
  - `articles/05-backtracking/01-03-letter_combinations_of_phone_number.md`

- Existing walkthrough seed: For two letters "aB", choose each letter's lower or upper case independently, giving four leaves. For a subset mask 101 over [2,5,9], the selected values are 2 and 9; its XOR is 11.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 37: Subsets, combinations, and permutations

- Own: `internal/leetgrinder/lessons/day-37.json`, `internal/leetgrinder/examples/day-37/`, `docs/leetgrinder-authoring/day-37.md`.

- Required demonstrations: Include/exclude subsets; start-index combinations; used-index permutations.

- Source articles:
  - `articles/05-backtracking/03-03-permutations.md`
  - `articles/05-backtracking/06-01-subsets_backtracking.md`

- Existing walkthrough seed: For [2,5,8], combination size two has [2,5], [2,8], [5,8]. Permutations include both [2,5,8] and [5,2,8], because order is part of the answer.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 38: Duplicate-aware enumeration

- Own: `internal/leetgrinder/lessons/day-38.json`, `internal/leetgrinder/examples/day-38/`, `docs/leetgrinder-authoring/day-38.md`.

- Required demonstrations: Same-depth duplicate skipping; duplicate-aware permutations; sibling versus child distinction.

- Source articles:
  - `articles/05-backtracking/05-01-deduplication.md`
  - `articles/05-backtracking/06-01-subsets_backtracking.md`

- Existing walkthrough seed: From [1,1,3], unique subsets include [1,1] but contain [1,3] only once. At the root, choosing either copy of 1 starts an equivalent branch; deeper in the branch, choosing the second 1 is legitimate.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 39: Combination sums and pruning

- Own: `internal/leetgrinder/lessons/day-39.json`, `internal/leetgrinder/examples/day-39/`, `docs/leetgrinder-authoring/day-39.md`.

- Required demonstrations: Reusable candidates; single-use candidates; sorted pruning and remaining sum.

- Source articles:
  - `articles/05-backtracking/05-02-combination_sum.md`
  - `articles/05-backtracking/02-01-backtracking_pruning.md`

- Existing walkthrough seed: Using [2,5] to make 9 with unlimited reuse permits [2,2,5]. If each candidate can be used once, that branch is illegal. With exactly two slots remaining, a branch that already uses too much sum cannot recover using positive values.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 40: Partition strings into valid pieces

- Own: `internal/leetgrinder/lessons/day-40.json`, `internal/leetgrinder/examples/day-40/`, `docs/leetgrinder-authoring/day-40.md`.

- Required demonstrations: Partition boundaries; palindrome validity; prefix segmentation and failed suffixes.

- Source articles:
  - `articles/05-backtracking/02-02-palindrome_partitioning.md`
  - `articles/05-backtracking/04-03-word_break.md`

- Existing walkthrough seed: Splitting "1221" into palindromes permits ["1","22","1"] and ["1221"]. A numeric segment "07" may be invalid because of a leading-zero rule even though its numeric value fits the range.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 41: Backtracking with coupled constraints

- Own: `internal/leetgrinder/lessons/day-41.json`, `internal/leetgrinder/examples/day-41/`, `docs/leetgrinder-authoring/day-41.md`.

- Required demonstrations: Balanced parentheses constraints; grid visited/unvisited; coupled state restoration.

- Source articles:
  - `articles/05-backtracking/03-01-backtracking-additional-states.md`
  - `articles/05-backtracking/03-02-generate_parentheses.md`
  - `articles/10-adv-data-structures/02-05-word_search_ii.md`

- Existing walkthrough seed: On a row [A,B,A], the word ABA can use all three positions once. The word ABAB cannot reuse the middle B. For initial numbers 3 and 4, an additive continuation must begin with 7, then 11.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 42: Symmetry and constraint-first search

- Own: `internal/leetgrinder/lessons/day-42.json`, `internal/leetgrinder/examples/day-42/`, `docs/leetgrinder-authoring/day-42.md`.

- Required demonstrations: N-queens columns/diagonals; Sudoku candidate restrictions; symmetry pruning limits.

- Source articles:
  - `articles/05-backtracking/02-01-backtracking_pruning.md`
  - `articles/05-backtracking/03-01-backtracking-additional-states.md`

- Existing walkthrough seed: To distribute sticks [6,6,3,3,3,3] among four equal sides of length 6, placing a 6 in one empty side is equivalent to placing it in any other empty side. Try one representative before exploring distinct loads.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 43: Graphs hidden in grids

- Own: `internal/leetgrinder/lessons/day-43.json`, `internal/leetgrinder/examples/day-43/`, `docs/leetgrinder-authoring/day-43.md`.

- Required demonstrations: Grid neighbor generation; flood fill; island component traversal.

- Source articles:
  - `articles/07-graph/03-01-matrix_as_graph.md`
  - `articles/07-graph/03-02-flood_fill.md`
  - `articles/07-graph/03-03-number_of_islands.md`

- Existing walkthrough seed: In [[1,0],[0,1]], four-direction adjacency gives two components; allowing diagonals gives one. Flood fill changes only the component whose cells match the starting color, not every cell with that color.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 44: Components and boundary reachability

- Own: `internal/leetgrinder/lessons/day-44.json`, `internal/leetgrinder/examples/day-44/`, `docs/leetgrinder-authoring/day-44.md`.

- Required demonstrations: Boundary flood fill; surrounded regions; reverse water reachability intersection.

- Source articles:
  - `articles/07-graph/03-03-number_of_islands.md`
  - `articles/07-graph/03-06-pacific_atlantic_water_flow.md`

- Existing walkthrough seed: A land cell contributes four candidate edges. Each adjacent land neighbor removes one exposed edge from that cell's contribution. A surrounded open region is one that no boundary-started search can reach.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 45: Breadth-first distances and multiple sources

- Own: `internal/leetgrinder/lessons/day-45.json`, `internal/leetgrinder/examples/day-45/`, `docs/leetgrinder-authoring/day-45.md`.

- Required demonstrations: BFS distance layers; multi-source initialization; visited-on-enqueue.

- Source articles:
  - `articles/07-graph/02-01-shortest_path_unweight.md`
  - `articles/07-graph/03-05-walls_and_gates.md`

- Existing walkthrough seed: On a line of five cells with sources at positions 0 and 4, the distances are [0,1,2,1,0]. A queue seeded with both endpoints discovers the middle once at distance two.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 46: Explicit and implicit adjacency

- Own: `internal/leetgrinder/lessons/day-46.json`, `internal/leetgrinder/examples/day-46/`, `docs/leetgrinder-authoring/day-46.md`.

- Required demonstrations: Adjacency lists; clone identity map; implicit neighbor generation.

- Source articles:
  - `articles/07-graph/01-01-graph_intro.md`
  - `articles/07-graph/02-02-clone_graph.md`
  - `articles/07-graph/04-01-word_ladder.md`
  - `articles/07-graph/04-02-open_the_lock.md`

- Existing walkthrough seed: If room 0 has keys [1,2] and room 1 has key [2], room 2 should be processed once for reachability. When listing paths 0→2 and 0→1→2, both routes are distinct outputs and must be retained.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 47: Directed dependencies and topological order

- Own: `internal/leetgrinder/lessons/day-47.json`, `internal/leetgrinder/examples/day-47/`, `docs/leetgrinder-authoring/day-47.md`.

- Required demonstrations: Indegree queue; DFS cycle states; invalid prefix and ordering constraints.

- Source articles:
  - `articles/07-graph/05-01-topo_intro.md`
  - `articles/07-graph/05-06-course_schedule.md`
  - `articles/07-graph/05-05-alien_dictionary.md`

- Existing walkthrough seed: For dependencies A→C and B→C, either A or B may appear first, but C waits for both. Adding C→A creates a cycle involving A and C, so no complete ordering exists.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 48: Disjoint sets and equivalence

- Own: `internal/leetgrinder/lessons/day-48.json`, `internal/leetgrinder/examples/day-48/`, `docs/leetgrinder-authoring/day-48.md`.

- Required demonstrations: Find/union; path compression; size/rank merge; account grouping.

- Source articles:
  - `articles/10-adv-data-structures/01-01-dsu_intro.md`
  - `articles/10-adv-data-structures/01-02-dsu_optimizations.md`
  - `articles/10-adv-data-structures/01-05-dsu_account_merge.md`

- Existing walkthrough seed: After unions (a,b) and (c,d), there are two components. Union (b,c) joins all four. An extra edge (a,d) connects vertices already sharing a representative, so it closes a cycle in an undirected graph.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 49: Weighted paths and state dimensions

- Own: `internal/leetgrinder/lessons/day-49.json`, `internal/leetgrinder/examples/day-49/`, `docs/leetgrinder-authoring/day-49.md`.

- Required demonstrations: Weighted relaxation; stale heap entry; expanded states with bounded resource.

- Source articles:
  - `articles/07-graph/06-01-dijkstra_intro.md`
  - `articles/07-graph/04-02-open_the_lock.md`

- Existing walkthrough seed: Edges A→B cost 8, A→C cost 2, and C→B cost 3. The two-edge route costs 5 and beats the direct edge. For a minimax route, combine a path with an edge using max instead of addition.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 50: Heaps and top-k streams

- Own: `internal/leetgrinder/lessons/day-50.json`, `internal/leetgrinder/examples/day-50/`, `docs/leetgrinder-authoring/day-50.md`.

- Required demonstrations: Bounded top-k heap; heap push/pop; balanced two-heap median.

- Source articles:
  - `articles/08-priority-queue-heap/01-01-heap_intro.md`
  - `articles/08-priority-queue-heap/02-03-kth_largest_element_in_an_array.md`
  - `articles/08-priority-queue-heap/04-01-median_of_data_stream.md`

- Existing walkthrough seed: For k=2 and stream [7,2,9,4], retain [2,7], replace 2 with 9, and reject 4. The final retained values are 7 and 9, so the second largest is 7.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 51: Ranking by derived keys

- Own: `internal/leetgrinder/lessons/day-51.json`, `internal/leetgrinder/examples/day-51/`, `docs/leetgrinder-authoring/day-51.md`.

- Required demonstrations: Derived distance keys; deterministic ties; safe comparator arithmetic.

- Source articles:
  - `articles/08-priority-queue-heap/02-01-k_closest_points.md`
  - `articles/01-getting-started/03-04-custom_comparator_sort.md`

- Existing walkthrough seed: Counts apple=4, pear=2, plum=2 rank apple first. If alphabetical order breaks equal counts, pear precedes plum. A heap that discards the least desirable item must reverse both frequency and tie directions consistently.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 52: Frequency ordering and greedy removal

- Own: `internal/leetgrinder/lessons/day-52.json`, `internal/leetgrinder/examples/day-52/`, `docs/leetgrinder-authoring/day-52.md`.

- Required demonstrations: Frequency ranking; hold-back last symbol; greedy frequency removal.

- Source articles:
  - `articles/08-priority-queue-heap/03-01-reorganize_string.md`
  - `articles/08-priority-queue-heap/01-01-heap_intro.md`

- Existing walkthrough seed: For group sizes [5,3,2], removing at least half of ten items needs only the size-five group. Sorting values by frequency may use a different tie rule, so separate the count key from the value tie breaker.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 53: Merge sorted streams with a heap

- Own: `internal/leetgrinder/lessons/day-53.json`, `internal/leetgrinder/examples/day-53/`, `docs/leetgrinder-authoring/day-53.md`.

- Required demonstrations: One frontier per sorted stream; heap replacement; stable source identity.

- Source articles:
  - `articles/08-priority-queue-heap/02-02-merge_k_sorted_lists.md`
  - `articles/08-priority-queue-heap/02-04-kth_smallest_element_in_a_sorted_matrix.md`

- Existing walkthrough seed: Merge streams [1,8], [3,6], and [2,9]. Start with 1,3,2; after taking 1, insert 8 from its stream. The heap then exposes 2. The source index tells you which successor to insert.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 54: Tries and prefix state

- Own: `internal/leetgrinder/lessons/day-54.json`, `internal/leetgrinder/examples/day-54/`, `docs/leetgrinder-authoring/day-54.md`.

- Required demonstrations: Insert/search/prefix; terminal versus prefix; wildcard branching.

- Source articles:
  - `articles/10-adv-data-structures/02-01-trie_intro.md`
  - `articles/10-adv-data-structures/02-03-prefix_count.md`
  - `articles/10-adv-data-structures/02-04-design_add_and_search_words_data_structure.md`

- Existing walkthrough seed: Insert "car" and "cart". The node after r is terminal and still has a child t. Searching "ca" fails as an exact word but succeeds as a prefix. A wildcard at the last position branches across matching children.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 55: Prefix choices and lexicographic order

- Own: `internal/leetgrinder/lessons/day-55.json`, `internal/leetgrinder/examples/day-55/`, `docs/leetgrinder-authoring/day-55.md`.

- Required demonstrations: Lexicographic trie traversal; prefix pruning; board search and state restoration.

- Source articles:
  - `articles/10-adv-data-structures/02-02-autocomplete.md`
  - `articles/10-adv-data-structures/02-05-word_search_ii.md`

- Existing walkthrough seed: With roots "be" and "bell", replacing "belly" stops at "be" because the shortest matching root wins. For suggestions after prefix "ca", explore child letters in sorted order and stop after the requested number of results.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 56: Scheduling with available choices

- Own: `internal/leetgrinder/lessons/day-56.json`, `internal/leetgrinder/examples/day-56/`, `docs/leetgrinder-authoring/day-56.md`.

- Required demonstrations: Available-job heap; cooldown/release events; idle time jumps.

- Source articles:
  - `articles/07-graph/05-02-task_scheduling.md`
  - `articles/08-priority-queue-heap/01-01-heap_intro.md`

- Existing walkthrough seed: Jobs released at times 0 and 4 with durations 2 and 1 leave an idle gap from 2 to 4. A scheduler must jump to 4 instead of repeatedly checking an empty heap. Equal durations may require an index tie breaker.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 57: Dynamic programming as cached contracts

- Own: `internal/leetgrinder/lessons/day-57.json`, `internal/leetgrinder/examples/day-57/`, `docs/leetgrinder-authoring/day-57.md`.

- Required demonstrations: Recursion tree to memo DAG; bottom-up order; rolling-state compression.

- Source articles:
  - `articles/09-dynamic-prog/01-01-dynamic_programming_intro.md`
  - `articles/05-backtracking/04-02-memoization_intro.md`
  - `articles/09-dynamic-prog/02-01-climbing_stairs.md`

- Existing walkthrough seed: If ways(n)=ways(n-1)+ways(n-2) with ways(0)=1 and ways(1)=1, the values through n=4 are 1,1,2,3,5. The empty construction counts once because it is the neutral starting choice.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 58: Choose or skip with adjacency constraints

- Own: `internal/leetgrinder/lessons/day-58.json`, `internal/leetgrinder/examples/day-58/`, `docs/leetgrinder-authoring/day-58.md`.

- Required demonstrations: Take/skip linear DP; circular split; tree take/skip contracts.

- Source articles:
  - `articles/09-dynamic-prog/03-02-house_robber.md`
  - `articles/09-dynamic-prog/11-02-house_robber_iii.md`

- Existing walkthrough seed: For rewards [4,7,3,8], the best prefix values are 4,7,7,15. Taking the last reward combines 8 with the best prefix ending before its neighbor, which is 7.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 59: Unbounded choices and decoding

- Own: `internal/leetgrinder/lessons/day-59.json`, `internal/leetgrinder/examples/day-59/`, `docs/leetgrinder-authoring/day-59.md`.

- Required demonstrations: Min coins; combination counts; decoding with zero handling.

- Source articles:
  - `articles/09-dynamic-prog/07-07-coin_change.md`
  - `articles/09-dynamic-prog/07-06-coin_change_ii.md`
  - `articles/05-backtracking/04-04-decode_ways.md`

- Existing walkthrough seed: With coins [3,5], amount 8 takes two coins through 3+5. Amount 4 is unreachable. For digits "101", the valid split is 10,1; the zero cannot stand alone.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 60: State machines and suffix feasibility

- Own: `internal/leetgrinder/lessons/day-60.json`, `internal/leetgrinder/examples/day-60/`, `docs/leetgrinder-authoring/day-60.md`.

- Required demonstrations: Suffix word feasibility; buy/hold/sell states; cooldown state transitions.

- Source articles:
  - `articles/05-backtracking/04-03-word_break.md`
  - `articles/09-dynamic-prog/03-01-dp_constant_transition_intro.md`

- Existing walkthrough seed: With a one-day cooldown, selling on day 2 cannot transition directly into buying on day 3. Model sold-today separately from resting. For text "rainbow", a valid split after "rain" depends on whether the remaining suffix can also be segmented.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 61: Sequences with predecessor choices

- Own: `internal/leetgrinder/lessons/day-61.json`, `internal/leetgrinder/examples/day-61/`, `docs/leetgrinder-authoring/day-61.md`.

- Required demonstrations: Predecessor DP and reconstruction; LIS tails; divisibility/string-chain eligibility.

- Source articles:
  - `articles/09-dynamic-prog/06-02-longest_increasing_subsequence.md`
  - `articles/09-dynamic-prog/06-04-largest_divisible_subset.md`
  - `articles/09-dynamic-prog/10-03-longest_string_chain.md`

- Existing walkthrough seed: For [2,5,3,7], the best lengths ending at each position are [1,2,2,3]. The final 7 extends either 2,5 or 2,3, so there are two longest increasing subsequences.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 62: Subset sums and loop direction

- Own: `internal/leetgrinder/lessons/day-62.json`, `internal/leetgrinder/examples/day-62/`, `docs/leetgrinder-authoring/day-62.md`.

- Required demonstrations: 0/1 subset transition; descending capacity; target-sign transformation with parity checks.

- Source articles:
  - `articles/09-dynamic-prog/07-03-partition_equal_subset_sum.md`
  - `articles/09-dynamic-prog/07-04-target_sum.md`
  - `articles/09-dynamic-prog/08-01-dp_knapsack_01.md`

- Existing walkthrough seed: With one item of weight 3 and capacity 6, an upward scan would mark 3 and then incorrectly use that new state to mark 6. A downward scan leaves only totals 0 and 3 reachable.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 63: Local ending states and global optima

- Own: `internal/leetgrinder/lessons/day-63.json`, `internal/leetgrinder/examples/day-63/`, `docs/leetgrinder-authoring/day-63.md`.

- Required demonstrations: Maximum ending sum; min/max ending product; global optimum updates.

- Source articles:
  - `articles/09-dynamic-prog/03-01-dp_constant_transition_intro.md`
  - `articles/09-dynamic-prog/06-01-dp_non_constant_transition_intro.md`

- Existing walkthrough seed: For products [2,-3,-4], after -3 the smallest ending product is -6. Multiplying that by -4 creates 24, the new largest ending product. Keeping only the previous maximum would miss it.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 64: Grid DP and blocked states

- Own: `internal/leetgrinder/lessons/day-64.json`, `internal/leetgrinder/examples/day-64/`, `docs/leetgrinder-authoring/day-64.md`.

- Required demonstrations: Path counts; obstacles; min-cost grid; rolling-row dependencies.

- Source articles:
  - `articles/09-dynamic-prog/04-01-dp_grid_intro.md`
  - `articles/09-dynamic-prog/04-03-unique_paths_ii.md`
  - `articles/09-dynamic-prog/04-04-minimal_path_sum.md`

- Existing walkthrough seed: In a 2 by 3 empty grid, path counts by row are [1,1,1] and [1,2,3]. Blocking the middle cell of the second row changes that row to [1,0,1].

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 65: Local geometry in grid states

- Own: `internal/leetgrinder/lessons/day-65.json`, `internal/leetgrinder/examples/day-65/`, `docs/leetgrinder-authoring/day-65.md`.

- Required demonstrations: Square geometry; triangle predecessor choices; reverse minimum-health DP.

- Source articles:
  - `articles/09-dynamic-prog/04-05-maximal_square.md`
  - `articles/09-dynamic-prog/04-06-triangle.md`
  - `articles/09-dynamic-prog/04-07-dungeon_game.md`

- Existing walkthrough seed: If the upper, left, and upper-left square sizes are 3,1,2, a current one supports a square of side 2, using one plus their minimum. Using their maximum would extend through unsupported cells.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 66: Two-sequence alignment

- Own: `internal/leetgrinder/lessons/day-66.json`, `internal/leetgrinder/examples/day-66/`, `docs/leetgrinder-authoring/day-66.md`.

- Required demonstrations: LCS table; reconstruction; common substring reset; supersequence merge.

- Source articles:
  - `articles/09-dynamic-prog/05-01-dp_two_sequence_intro.md`
  - `articles/09-dynamic-prog/05-02-longest_common_subsequence.md`
  - `articles/09-dynamic-prog/05-06-shortest_common_supersequence.md`

- Existing walkthrough seed: For "cab" and "acb", a common subsequence of length two is "ab" or "cb". Matching equal b characters extends the best answer for the two shorter prefixes; unmatched final symbols require choosing a skip.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 67: Edit choices and palindrome intervals

- Own: `internal/leetgrinder/lessons/day-67.json`, `internal/leetgrinder/examples/day-67/`, `docs/leetgrinder-authoring/day-67.md`.

- Required demonstrations: Edit operations; palindrome interval recurrence; increasing interval length.

- Source articles:
  - `articles/09-dynamic-prog/05-03-edit_distance.md`
  - `articles/09-dynamic-prog/09-04-longest_palindromic_subsequence.md`
  - `articles/09-dynamic-prog/09-02-palindromic_substrings.md`

- Existing walkthrough seed: Changing "cat" to "cut" uses one replacement. For interval "abca", equal outer a characters let a palindrome subsequence problem consult "bc"; a palindrome substring check still requires the entire interior to be palindromic.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 68: Counting matches and interleaving

- Own: `internal/leetgrinder/lessons/day-68.json`, `internal/leetgrinder/examples/day-68/`, `docs/leetgrinder-authoring/day-68.md`.

- Required demonstrations: Distinct-match counting; reverse update order; interleaving two-prefix feasibility.

- Source articles:
  - `articles/09-dynamic-prog/05-05-distinct_subsequences.md`
  - `articles/09-dynamic-prog/05-01-dp_two_sequence_intro.md`

- Existing walkthrough seed: For source "aab" and target "ab", there are two index-distinct subsequences. For repeated contiguous matches, a mismatch makes the current matching suffix length zero. Interleaving "ab" and "xy" can produce "axby" while preserving each source's order.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 69: More than one resource

- Own: `internal/leetgrinder/lessons/day-69.json`, `internal/leetgrinder/examples/day-69/`, `docs/leetgrinder-authoring/day-69.md`.

- Required demonstrations: Two-capacity 0/1 knapsack; bounded transaction/resource dimension; state size.

- Source articles:
  - `articles/09-dynamic-prog/08-02-knapsack_intro.md`
  - `articles/09-dynamic-prog/07-01-dp_knapsack_intro.md`

- Existing walkthrough seed: If a plan needs at least 5 profit, states with profit 5 and 8 can share a capped profit value 5 when only threshold attainment matters. They cannot be merged if the exact final profit is part of the objective.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 70: Interval games and last actions

- Own: `internal/leetgrinder/lessons/day-70.json`, `internal/leetgrinder/examples/day-70/`, `docs/leetgrinder-authoring/day-70.md`.

- Required demonstrations: Score-difference game DP; choose last burst; open interval boundaries.

- Source articles:
  - `articles/09-dynamic-prog/09-01-dp_interval_intro.md`
  - `articles/09-dynamic-prog/09-03-coin_game.md`

- Existing walkthrough seed: For a game [4,9,2], taking 4 leaves an opponent who can take 9, so immediate reward alone is misleading. A difference recurrence compares left_value minus the remaining interval's advantage with the analogous right choice.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 71: Greedy choices need exchange arguments

- Own: `internal/leetgrinder/lessons/day-71.json`, `internal/leetgrinder/examples/day-71/`, `docs/leetgrinder-authoring/day-71.md`.

- Required demonstrations: Smallest adequate assignment; highest-unit-value exchange; denomination-specific change.

- Source articles:
  - `articles/11-miscellaneous/07-01-greedy_intro.md`
  - `articles/12-company-oas/01-05-fill_the_truck.md`

- Existing walkthrough seed: Demands [2,4] and resources [3,5] can both be satisfied by pairing 2 with 3 and 4 with 5. Giving 5 to demand 2 first leaves no adequate resource for 4.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 72: Pairing and assignment by sorted order

- Own: `internal/leetgrinder/lessons/day-72.json`, `internal/leetgrinder/examples/day-72/`, `docs/leetgrinder-authoring/day-72.md`.

- Required demonstrations: Two-person boat pairing; adjacent sorted pair minima; two-city cost differences.

- Source articles:
  - `articles/11-miscellaneous/07-01-greedy_intro.md`
  - `articles/03-two-pointers/04-01-opposite_direction_family_map.md`

- Existing walkthrough seed: For travel costs (A=20,B=70) and (A=60,B=30), assigning the first person to A and the second to B costs 50. The differences -50 and 30 measure how much assigning A changes the total relative to assigning everyone B.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 73: Intervals and commitments

- Own: `internal/leetgrinder/lessons/day-73.json`, `internal/leetgrinder/examples/day-73/`, `docs/leetgrinder-authoring/day-73.md`.

- Required demonstrations: Earliest finish scheduling; shared arrow boundary; last-occurrence partitions.

- Source articles:
  - `articles/11-miscellaneous/01-06-non_overlapping_intervals.md`
  - `articles/11-miscellaneous/01-07-minimum_number_of_arrows_to_burst_balloons.md`
  - `articles/11-miscellaneous/01-08-partition_labels.md`

- Existing walkthrough seed: Intervals [1,4], [2,3], and [3,5] favor selecting [2,3] first when touching endpoints are compatible. Choosing the longest interval first can remove two opportunities. For closed balloon intervals, touching endpoints may still share one arrow.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 74: Merge and intersect intervals

- Own: `internal/leetgrinder/lessons/day-74.json`, `internal/leetgrinder/examples/day-74/`, `docs/leetgrinder-authoring/day-74.md`.

- Required demonstrations: Merge union; sorted insertion; two-list intersection.

- Source articles:
  - `articles/11-miscellaneous/01-02-merge_intervals.md`
  - `articles/11-miscellaneous/01-03-insert_interval.md`
  - `articles/11-miscellaneous/01-01-interval_pattern_intro.md`

- Existing walkthrough seed: Intersect [1,6] with [4,9] to obtain [4,6]. Since the first interval ends at 6, advance that list while retaining [4,9] for possible overlap with later intervals.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 75: Greedy repair and reachable ranges

- Own: `internal/leetgrinder/lessons/day-75.json`, `internal/leetgrinder/examples/day-75/`, `docs/leetgrinder-authoring/day-75.md`.

- Required demonstrations: Farthest reachable prefix; jump layers; gas candidate reset; monotone digit repair.

- Source articles:
  - `articles/11-miscellaneous/07-02-gas_station.md`
  - `articles/11-miscellaneous/07-01-greedy_intro.md`
  - `articles/12-company-oas/02-17-jump_game.md`

- Existing walkthrough seed: If a candidate fuel route becomes negative after station j, any later start inside the same failed segment also lacks the discarded positive prefix, so those starts cannot work. Start again after j and continue tracking the total balance.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 76: Monotonic stacks and unresolved indices

- Own: `internal/leetgrinder/lessons/day-76.json`, `internal/leetgrinder/examples/day-76/`, `docs/leetgrinder-authoring/day-76.md`.

- Required demonstrations: Next-greater unresolved indices; equal values; circular traversal.

- Source articles:
  - `articles/11-miscellaneous/03-01-mono_stack_intro.md`
  - `articles/11-miscellaneous/03-03-daily_temperatures.md`
  - `articles/11-miscellaneous/03-04-next_greater_element_ii.md`

- Existing walkthrough seed: For temperatures [60,55,63], index 0 waits, then index 1 waits. Reading 63 resolves index 1 with distance 1 and index 0 with distance 2. Both pops are charged to their earlier pushes.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 77: Compressed stack summaries

- Own: `internal/leetgrinder/lessons/day-77.json`, `internal/leetgrinder/examples/day-77/`, `docs/leetgrinder-authoring/day-77.md`.

- Required demonstrations: Min-stack summaries; monotonic deque expiration; histogram popped spans.

- Source articles:
  - `articles/11-miscellaneous/02-01-min_stack.md`
  - `articles/11-miscellaneous/03-02-sliding_window_maximum.md`
  - `articles/11-miscellaneous/03-05-largest_rectangle_in_histogram.md`

- Existing walkthrough seed: For prices [40,30,35], spans are [1,1,2]. The last price absorbs the previous block for 30 but stops at 40. For digits "4312" with one deletion, removing 4 gives "312", better than deleting a later smaller digit.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 78: Mixed practice: round 1

- Own: `internal/leetgrinder/lessons/day-78.json`, `internal/leetgrinder/examples/day-78/`, `docs/leetgrinder-authoring/day-78.md`.

- Required demonstrations: Compare plausible patterns; reveal invariants and counterexamples after attempt.

- Source articles:
  - `articles/03-two-pointers/08-01-two_pointers_decision_rule.md`
  - `articles/01-getting-started/01-08-keyword_to_algo.md`

- Existing walkthrough seed: For values [8,4,6,5], start the run at 4 and count 4,5,6; 5 and 6 are not separate starts. A bitwise AND keeps only positions set in both numbers. For bit counts, n & (n - 1) removes the lowest set bit: 1100 AND 1011 becomes 1000. Thus count(12)=count(8)+1, with count(0)=0.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 79: Mixed practice: round 2

- Own: `internal/leetgrinder/lessons/day-79.json`, `internal/leetgrinder/examples/day-79/`, `docs/leetgrinder-authoring/day-79.md`.

- Required demonstrations: Compare plausible patterns; reveal invariants and counterexamples after attempt.

- Source articles:
  - `articles/02-binary-search/05-01-binary-search-speedrun.md`
  - `articles/03-two-pointers/09-03-two_pointers_synthesis.md`

- Existing walkthrough seed: For sorted [1,3,6,8,10], compare the two candidate windows [3,6,8] and [6,8,10] around 7. The endpoint distances are 4 for 3 and 3 for 10, so choose [6,8,10]. For binary values, equal prefix differences count sums exactly.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 80: Mixed practice: round 3

- Own: `internal/leetgrinder/lessons/day-80.json`, `internal/leetgrinder/examples/day-80/`, `docs/leetgrinder-authoring/day-80.md`.

- Required demonstrations: Compare plausible patterns; reveal invariants and counterexamples after attempt.

- Source articles:
  - `articles/05-backtracking/07-01-backtracking-speedrun.md`
  - `articles/07-graph/08-01-graph-speedrun.md`

- Existing walkthrough seed: A list a→b→c→d rotated right once becomes d→a→b→c. Closing the ring makes the order easy to find, but forgetting the final cut leaves an infinite traversal. An inorder stack descends left before visiting a node.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 81: Mixed practice: round 4

- Own: `internal/leetgrinder/lessons/day-81.json`, `internal/leetgrinder/examples/day-81/`, `docs/leetgrinder-authoring/day-81.md`.

- Required demonstrations: Compare plausible patterns; reveal invariants and counterexamples after attempt.

- Source articles:
  - `articles/09-dynamic-prog/13-01-dp-list.md`
  - `articles/08-priority-queue-heap/01-01-heap_intro.md`

- Existing walkthrough seed: If water can descend from height 7 to 4, reverse reachability moves from 4 to 7. A cycle A→B→A in a clone must return to the already allocated clone of A, not allocate a second copy.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 82: Mixed practice: round 5

- Own: `internal/leetgrinder/lessons/day-82.json`, `internal/leetgrinder/examples/day-82/`, `docs/leetgrinder-authoring/day-82.md`.

- Required demonstrations: Compare plausible patterns; reveal invariants and counterexamples after attempt.

- Source articles:
  - `articles/07-graph/08-02-graph-speedrun2.md`
  - `articles/11-miscellaneous/01-01-interval_pattern_intro.md`

- Existing walkthrough seed: For climbs [2,7,3] and one ladder, spending bricks on 2 and 3 while assigning the ladder to 7 uses five bricks. A min-heap of ladder-covered climbs can eject the smallest climb when ladder capacity is exceeded.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 83: Mixed practice: round 6

- Own: `internal/leetgrinder/lessons/day-83.json`, `internal/leetgrinder/examples/day-83/`, `docs/leetgrinder-authoring/day-83.md`.

- Required demonstrations: Compare plausible patterns; reveal invariants and counterexamples after attempt.

- Source articles:
  - `articles/09-dynamic-prog/13-01-dp-list.md`
  - `articles/11-miscellaneous/03-01-mono_stack_intro.md`

- Existing walkthrough seed: For sorted scores [2,4,9,11] and k=2, adjacent ranges are 2,5,2. A noncontiguous choice cannot improve the range. In a game where the next move limit depends on the previous take, two identical suffixes with different limits are different states.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.

## Day 84: Mock interview and invariant review

- Own: `internal/leetgrinder/lessons/day-84.json`, `internal/leetgrinder/examples/day-84/`, `docs/leetgrinder-authoring/day-84.md`.

- Required demonstrations: Mock interview explanation; baseline refinement; invariant and adversarial-input review.

- Source articles:
  - `articles/01-getting-started/01-03-interview-process.md`
  - `articles/01-getting-started/01-04-getting_started.md`
  - `articles/01-getting-started/01-08-keyword_to_algo.md`

- Existing walkthrough seed: For cache capacity two, access A, then B, then A. Inserting C should evict B because the second access moved A to the most recent position. For a run-removal threshold of three, "abbba" becomes "aa" after removing the b run.

- Acceptance: each demonstration has C++, Python, Java, and Go sources, a normal trace and a revealing edge case, executable expected-output checks, a labeled SVG, line mappings, and an explanation of why each state change is valid. Cover every concept already taught by the lesson, including any concept omitted from this minimum inventory.
