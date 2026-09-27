# Leetgrinder sources and verification

Verified on 2026-09-26. The curriculum has 84 sessions with 252 core problems and 48 optional problems. All 300 assignments are distinct, and all are free according to LeetCode's public catalog at verification time. Access rules and external pages can change.

## Authorship and local source material

The local `data/algomonster/CURRICULUM.md` was used to identify subject coverage and prerequisite relationships. The new lessons, worked examples, review prompts, and daily assignments were authored independently. No source lesson text, source code, problem statements, or solutions are embedded in the application. The local source directory is not required at build time and is excluded by the Docker context allowlist.

All instructional prose is in `internal/leetgrinder/lessons.templ`. Day metadata, reading directions, and public problem titles are in `internal/leetgrinder/curriculum.go`. The runtime uses these compiled files and never fetches or renders the private source material.

## Reading references

Required reading normally has a ten-minute allowance. Later references to the same introductory material are optional five-minute refreshers. Daily directions identify sections to read and questions to answer. Reading and solution review share the 30-minute budget; the linked textbooks are references, not assignments to read whole chapters.

| Topic | Original educational source |
| --- | --- |
| Hash tables | Pat Morin, [Hashing with chaining](https://opendatastructures.org/ods-python/5_1_ChainedHashTable_Hashin.html) |
| Arrays and introductory binary search | Robert Sedgewick and Kevin Wayne, [Programming model](https://algs4.cs.princeton.edu/11model/) |
| Merge sort | Sedgewick and Wayne, [Mergesort](https://algs4.cs.princeton.edu/22mergesort/) |
| Boundaries, predicates, and answer-space search | [CP-Algorithms: binary search](https://cp-algorithms.com/num_methods/binary_search.html) |
| Two pointers and sliding windows | [USACO Guide: two pointers](https://usaco.guide/silver/two-pointers) |
| Prefix sums | [USACO Guide: prefix sums](https://usaco.guide/silver/prefix-sums) |
| Linked lists | Morin, [Singly-linked lists](https://opendatastructures.org/ods-python/3_1_SLList_Singly_Linked_Li.html) |
| Stacks and queues | Sedgewick and Wayne, [Bags, queues, and stacks](https://algs4.cs.princeton.edu/13stacks/) |
| Recursion | Jeff Erickson, [Recursion](https://jeffe.cs.illinois.edu/teaching/algorithms/book/01-recursion.pdf) |
| Trees | Morin, [A basic binary tree](https://opendatastructures.org/ods-python/6_1_BinaryTree_Basic_Binary.html) |
| Backtracking | Erickson, [Backtracking](https://jeffe.cs.illinois.edu/teaching/algorithms/book/02-backtracking.pdf) |
| Graph traversal | Sedgewick and Wayne, [Undirected graphs](https://algs4.cs.princeton.edu/41graph/) |
| Directed dependencies | Sedgewick and Wayne, [Directed graphs](https://algs4.cs.princeton.edu/42digraph/) |
| Disjoint sets | Sedgewick and Wayne, [Union-find](https://algs4.cs.princeton.edu/15uf/) |
| Weighted paths | Sedgewick and Wayne, [Shortest paths](https://algs4.cs.princeton.edu/44sp/) |
| Heaps | Morin, [Binary heap](https://opendatastructures.org/ods-python/10_1_BinaryHeap_Implicit_Bi.html) |
| Tries | Carnegie Mellon, Thomas Cortina's course with notes by Frank Pfenning, [Tries, sections 2–3](https://www.cs.cmu.edu/~wlovas/15122-r11/lectures/24-tries.pdf) |
| Dynamic programming | Erickson, [Dynamic programming](https://jeffe.cs.illinois.edu/teaching/algorithms/book/03-dynprog.pdf), with separate directions for text segmentation, sequence states, edit distance, subset sum, and interval dependencies |
| Greedy proofs and scheduling | Erickson, [Greedy algorithms](https://jeffe.cs.illinois.edu/teaching/algorithms/book/04-greedy.pdf) |
| Monotonic stacks | [USACO Guide: stacks](https://usaco.guide/gold/stacks) |
| Bit operations | [CP-Algorithms: bit manipulation](https://cp-algorithms.com/algebra/bit-manipulation.html) |

Sources were opened and their relevant sections inspected. A link returning HTTP 200 is checked separately from its educational relevance. The final mixed-practice lessons are collapsed by default so learners can choose an approach before seeing the discussion.

## Problem catalog verification

The [LeetCode public problem catalog](https://leetcode.com/api/problems/all/) supplied canonical public problem IDs, slugs, titles, difficulty, and the `paid_only` field. Each assignment was checked against those fields. Practice links use `https://leetcode.com/problems/<canonical-slug>/`; the app does not reproduce the corresponding statements or solutions.

[leetgrinder-problems.json](leetgrinder-problems.json) records the verified assignment metadata. This is an audit snapshot, not a second runtime curriculum. Re-run the live check with:

```sh
make generate
go run ./cmd/verify-leetgrinder
```

The verifier reads the actual compiled curriculum, checks all 300 problems against the live catalog, and requests each distinct reading document. It exits nonzero for metadata mismatches, paid problems, duplicates, count errors, inaccessible documents, and network failures. It does not submit solutions or access a LeetCode account.

`go test ./internal/leetgrinder` separately checks day ordering, assignment totals, lookup behavior, reading budgets, unique lesson rendering, draft escaping, and mixed-practice guidance. PostgreSQL and HTTP workflow tests cover the saved learning history.

## Optimal complexity table

Each curriculum problem in `internal/leetgrinder/curriculum.go` carries `OptimalTime`, `OptimalSpace`, and, where needed, `OptimalNote`. The values were hand-curated on 2026-09-27 from each problem's LeetCode constraints and the standard editorial and textbook solutions, then checked by independent reviewers. `TestEveryProblemHasOptimalComplexity` checks that all 300 problems have both values, already in the form `NormalizeComplexity` (in `internal/leetgrinder/complexity.go`, shared with stated attempt complexities) produces.

### Conventions

- **Notation.** Values use the stored complexity notation: `O(1)`, `O(log n)`, `O(√n)`, `O(n)`, `O(n log n)`, `O(n²)`, `O(n³)`, `O(2ⁿ)`, `O(n!)`, or a precise custom form such as `O(m·n)`, `O(V + E)`, or `O(k log n)`. Each value is at most 40 characters, starts with `O(` and ends with `)`. Exponents other than `n` use `^`, for example `O(m·n·3^L)`.
- **Variables.** `n` is the main input size. Grids are `m × n`, graphs have `V` vertices and `E` edges, and trees have `n` nodes and height `h`. Any other variable (target, amount, capacity, word length, value range) is defined in the problem's note. When the problem statement uses `n` for something other than the input size (the cooldown in 621, the position in 19, the target in 216), the table uses a different letter and the note says so.
- **Best known, not just typical.** The value is the best bound among the standard editorial and textbook approaches. When the usual interview solution is slower, the note names it, for example Fibonacci by matrix exponentiation (`O(log n)`) against the iterative DP (`O(n)`).
- **Space is auxiliary.** Space excludes the returned value (a list of subsets, the answer array) but counts working structures such as recursion stacks, hash maps, heaps, and memo tables. The design allowed counting the output when it dominates; the table never does, because the time bound already reflects the output size and a stated space complexity is normally auxiliary. Where a DP or scan can reuse the answer array or the mutable input array, the table records that (`O(1)`), and the note gives the rolling-row cost.
- **Sorting in place** counts as `O(1)` auxiliary space, since heapsort achieves that. Library sorts may use `O(log n)` or `O(n)`.
- **Constraints.** Fixed alphabets (26 letters, ASCII) and fixed boards (a 9 × 9 Sudoku) are constants, so their counters are `O(1)`. Numeric value ranges are not treated as constants: comparison sorts stay `O(n log n)` even when a counting sort over a bounded range would also pass. Notes mention the counting sort only where it is a common alternative (912, 561, 1710).
- **Design problems** record the time per operation and the space of the stored structure. The note gives constructor or build costs.
- **Expected bounds.** Hash-map operations count as `O(1)`, so hash-based bounds are expected rather than worst case, as is usual. Quickselect bounds are average case.
- **Trees.** Recursive traversals use `O(h)` stack space. Morris threading can reduce some traversals to `O(1)` by temporarily rewiring the tree; the table does not use it and notes it only for Binary Tree Inorder Traversal. Level-order problems record `O(n)`: DFS by depth needs `O(h)` and BFS needs `O(w)` for the widest level, and neither dominates the other.

### Disputed and ambiguous entries

| Problem | Recorded | Reasoning |
| --- | --- | --- |
| 912 Sort an Array | `O(n log n)`, `O(1)` | The statement requires `O(n log n)` and the smallest possible space, so heapsort. Merge sort is `O(n)` space; counting sort over the value range is `O(n + r)`. |
| 119 Pascal's Triangle II | `O(n)`, `O(1)` | The binomial recurrence builds the row in `O(n)`; the row DP most learners write is `O(n²)`. |
| 509, 70, 1137 (Fibonacci-style) | `O(log n)`, `O(1)` | Matrix exponentiation or fast doubling. The iterative DP is `O(n)`. |
| 279 Perfect Squares | `O(√n)`, `O(1)` | Lagrange's four-square and Legendre's three-square theorems. The DP is `O(n·√n)`. |
| 343 Integer Break | `O(log n)`, `O(1)` | Split into 3s with fast power. With floating-point `pow` it is often quoted as `O(1)`. |
| 877 Stone Game | `O(1)`, `O(1)` | The first player always wins under the constraints (even pile count, odd total). The interval DP is `O(n²)`. |
| 1863 Sum of All Subset XOR Totals | `O(n)`, `O(1)` | Every set bit of the OR contributes to half the subsets. Enumeration is `O(2ⁿ)`. |
| 62 Unique Paths | `O(min(m, n))`, `O(1)` | Binomial coefficient. The grid DP is `O(m·n)`. |
| 647 Palindromic Substrings | `O(n)`, `O(n)` | Manacher's algorithm is in the editorial. Expanding around centres is `O(n²)` time, `O(1)` space. |
| 718 Maximum Length of Repeated Subarray | `O((m + n)·log(min(m, n)))`, `O(min(m, n))` | Binary search on length with rolling hashes, an editorial approach, gives an expected bound. The DP is `O(m·n)`; suffix automata reach `O(m + n)` but are outside the standard approaches. |
| 673 Number of LIS | `O(n log n)`, `O(n)` | Segment or Fenwick tree over values. The DP is `O(n²)`. |
| 1335 Minimum Difficulty of a Job Schedule | `O(n·d)`, `O(n)` | Monotonic stack per day. The plain DP is `O(n²·d)`. |
| 1155 Number of Dice Rolls | `O(n·t)`, `O(t)` | Sliding-window sums remove the face loop from `O(n·k·t)`. |
| 692 Top K Frequent Words | `O(n)`, `O(n)` | Bucket sort with a trie per bucket. The follow-up targets `O(n log k)` with a heap. |
| 215, 973 (kth element, k closest) | `O(n)`, `O(1)` | Average-case quickselect. A heap gives `O(n log k)`; median-of-medians makes the bound worst case. |
| 378 Kth Smallest in a Sorted Matrix | `O(n log r)`, `O(1)` | Binary search on value with a staircase count, `r` = max − min. A heap gives `O(k log n)`; Frederickson–Johnson reaches `O(n)` but is outside the standard approaches. |
| 406 Queue Reconstruction by Height | `O(n²)`, `O(1)` | Editorial greedy with list insertion. A Fenwick tree gives `O(n log n)`. |
| 1029 Two City Scheduling | `O(n log n)`, `O(1)` | Editorial sort. Quickselect on the cost difference gives `O(n)` average. |
| 572 Subtree of Another Tree | `O(m + n)`, `O(m + n)` | Serialisation with KMP, or subtree hashing. Direct comparison is `O(m·n)`, the common interview answer. |
| 445 Add Two Numbers II | `O(m + n)`, `O(1)` | Reverses the inputs. The follow-up forbids modifying them, which needs `O(m + n)` stacks. |
| 739 Daily Temperatures | `O(n)`, `O(1)` | Right-to-left scan jumping through the answer array. The monotonic stack uses `O(n)`. |
| 542 01 Matrix | `O(m·n)`, `O(1)` | Two DP passes over the output. Multi-source BFS uses `O(m·n)`. |
| 63, 64, 120, 931, 1277 (grid DP) | `O(1)` space | DP in place over the mutable input. A rolling row keeps the input intact at `O(n)`. |
| 430 Flatten a Multilevel Doubly Linked List | `O(n)`, `O(1)` | Splicing each child list in place walks each node at most twice. The editorial DFS uses `O(n)`. |
| 23 Merge k Sorted Lists | `O(n log k)`, `O(1)` | Iterative pairwise merging, `n` = total nodes. A heap uses `O(k)`. |
| 394 Decode String | `O(n + L)`, `O(n)` | `L` = decoded length. Writing repeats into one buffer costs `O(n + L)`; a stack of partial strings recopies nested text. |
| 846 Hand of Straights | `O(n)`, `O(n)` | The editorial's reverse-decrement approach walks back to each run start. A sorted map gives `O(n log n)`. |
| 221 Maximal Square | `O(m·n)`, `O(min(m, n))` | The input holds characters, so unlike the integer grid DPs it is not reused as DP storage. |
| 139 Word Break | `O((n + m)·k)`, `O(n + m·k)` | Trie walk from each start, `k` = longest word. Substring hashing gives `O(n·k²)`. |
| 36 Valid Sudoku, 37 Sudoku Solver | `O(1)`; `O(9^m)`, `O(m)` | The board is fixed. The solver records the search in terms of `m` empty cells to stay informative, though it is formally constant. |
| 3 Longest Substring Without Repeating Characters | `O(n)`, `O(1)` | The character set is bounded. Editorials often write `O(min(n, k))` for a charset of size `k`. |
| 1268 Search Suggestions System | `O(n log n + m)`, `O(1)` | String comparisons are counted as `O(1)`, as the editorial does; with word length `L` multiply by `L`. |
| 779 K-th Symbol in Grammar | `O(log k)`, `O(1)` | Parity of the set bits of `k − 1`. Recursion on the row is `O(n)`, never better because `k ≤ 2ⁿ⁻¹`. |
