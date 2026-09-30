# Day 32 authoring record

## Scope and source use

This lesson teaches FIFO tree traversal, saved level boundaries, left-to-right order, right-side view, first-leaf minimum depth, bottom-up level order, empty trees, and queue storage. The previous lesson's short walkthrough supplied the root/children/grandchildren scenario and cautions about width and slow front removal. The lesson now gives complete code and state traces for each distinct BFS technique.

Local AlgoMonster research was limited to these files and sections:

- `/home/mduren/Documents/data/algomonster/articles/06-breadth-first-search/01-01-bfs_intro.md`: the “DFS vs BFS” and “Template” sections establish FIFO traversal and child enqueue order.
- `/home/mduren/Documents/data/algomonster/articles/06-breadth-first-search/02-01-binary_tree_level_order_traversal.md`: the “Explanation” and “How to get a node's level” sections identify the saved queue-length boundary.
- `/home/mduren/Documents/data/algomonster/articles/06-breadth-first-search/02-03-binary_tree_right_side_view.md`: the explanation uses a right-first alternative. This lesson instead enqueues left first and takes the final node of a fixed level, matching the existing walkthrough.

No article prose, code, image, or diagram was copied into the authored files. Existing public References remain in `readings.go`; that shared file was not edited.

## Input and examples

Each program reads `n`, then `n` rows of `value leftIndex rightIndex`. Index zero is the root, and `-1` means missing child. This fixed input makes the four implementations and trace states directly comparable.

| Demonstration | Branching tree | Single-child chain | Empty tree |
| --- | --- | --- | --- |
| Baseline depth rescan | `[[7],[3,8],[1,5]]` | `[[7],[3],[5]]` | `[]` |
| Level boundary | `[[7],[3,8],[1,5]]` | `[[7],[3],[5]]` | `[]` |
| Rightmost node | `[7,8,5]` | `[7,3,5]` | `[]` |
| First-leaf depth | `2` | `3` | `0` |

The branching tree has rows `7 1 2`, `3 3 4`, `8 -1 -1`, `1 -1 -1`, and `5 -1 -1`. The chain has rows `7 1 -1`, `3 -1 2`, and `5 -1 -1`. The chain reveals why one missing child does not make a leaf. The empty case checks initialization and terminal output.

## Trace and drawing decisions

Every trace starts at `init` and ends at `done`. The baseline adds `height`, `pass_start`, and `pass_end` checkpoints. Its variables are `depth`, `visits`, `level`, and `result`; `visits` counts collection passes after the separate height calculation. Level grouping and right-side view record `level_start`, each `visit`, and `level_end`. Minimum depth records `level_start`, each non-leaf `visit`, and the first `leaf`; it stops before deeper queued nodes. Each frame's ordered variables are `depth`, `current`, `queue`, `level`, and `result`. Queue entries are node indices in front-to-back order, while circle text is node value. Queue boxes show both index and value. Tree edges represent child links, and the bottom label repeats the current level and result. An empty tree has no circles or edges and an empty queue.

The frame line maps point at the actual emit call in each complete source file. Algorithm display ranges start at result initialization and end at the terminal emit. The Go files are named `main.go.txt` per integration direction; the verifier stages them as `main.go` before building. Each case's `expectedOutput` is the exact final result string emitted by all four programs.

## Coverage and review

The required three BFS demonstrations are separate examples. A fourth runnable example shows the baseline repeated-depth scan in all four languages; its trace records each complete depth pass and the cumulative visits spent revisiting upper nodes. Bottom-up level order is explained as reversing the outer list of completed levels; it does not need a different queue invariant. A reviewer should check the animated queue against the visible tree, especially the moment node 8 ends the minimum-depth trace while nodes 3 and 4 remain queued.

Execution evidence: `python3 scripts/leetgrinder/verify_examples.py --days 32` exited 0 after checking all 4 examples, 3 cases each, and C++, Python, Java, and Go variants (48 runs). The verifier compares every event and variable record with the authored frames and checks final result strings.

## Review corrections

The depth-rescan trace now marks nodes reached again as active and current-level nodes as result. The right-side trace marks the chosen node for each completed level. Source mappings for visit, level boundary, and commit frames now highlight the queue operations and result updates rather than only the trace-emission calls. The 48 language and case executions passed again after the frame changes. The mobile browser sweep rendered all four examples at 360 pixels without page-level overflow.
