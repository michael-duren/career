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
