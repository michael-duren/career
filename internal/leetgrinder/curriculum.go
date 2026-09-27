package leetgrinder

// Curriculum returns a fresh copy of the complete, ordered study plan.
func Curriculum() []Week {
	return []Week{
		{Number: 1, Title: "Fundamentals", Summary: "Recover array, hashing, sorting, and simulation skills. State loop invariants and account for time and space.", Days: []Day{
			{Number: 1, Title: "Hash lookup and honest baselines", Lesson: "day-01", Readings: lessonReadings(1), Core: []Problem{
				{ID: 1, Slug: "two-sum", Title: "Two Sum", Difficulty: "Easy"},
				{ID: 217, Slug: "contains-duplicate", Title: "Contains Duplicate", Difficulty: "Easy"},
				{ID: 242, Slug: "valid-anagram", Title: "Valid Anagram", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 2, Title: "Counting as state", Lesson: "day-02", Readings: lessonReadings(2), Core: []Problem{
				{ID: 383, Slug: "ransom-note", Title: "Ransom Note", Difficulty: "Easy"},
				{ID: 387, Slug: "first-unique-character-in-a-string", Title: "First Unique Character in a String", Difficulty: "Easy"},
				{ID: 169, Slug: "majority-element", Title: "Majority Element", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 3, Title: "Canonical forms and bijections", Lesson: "day-03", Readings: lessonReadings(3), Core: []Problem{
				{ID: 49, Slug: "group-anagrams", Title: "Group Anagrams", Difficulty: "Medium"},
				{ID: 205, Slug: "isomorphic-strings", Title: "Isomorphic Strings", Difficulty: "Easy"},
				{ID: 290, Slug: "word-pattern", Title: "Word Pattern", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 4, Title: "Sets, multisets, and repeated states", Lesson: "day-04", Readings: lessonReadings(4), Core: []Problem{
				{ID: 350, Slug: "intersection-of-two-arrays-ii", Title: "Intersection of Two Arrays II", Difficulty: "Easy"},
				{ID: 349, Slug: "intersection-of-two-arrays", Title: "Intersection of Two Arrays", Difficulty: "Easy"},
				{ID: 202, Slug: "happy-number", Title: "Happy Number", Difficulty: "Easy"},
			}, Optional: []Problem{
				{ID: 36, Slug: "valid-sudoku", Title: "Valid Sudoku", Difficulty: "Medium"},
			}},
			{Number: 5, Title: "Sorting and loop invariants", Lesson: "day-05", Readings: lessonReadings(5), Core: []Problem{
				{ID: 88, Slug: "merge-sorted-array", Title: "Merge Sorted Array", Difficulty: "Easy"},
				{ID: 912, Slug: "sort-an-array", Title: "Sort an Array", Difficulty: "Medium"},
				{ID: 75, Slug: "sort-colors", Title: "Sort Colors", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 6, Title: "Array transformations and carries", Lesson: "day-06", Readings: lessonReadings(6), Core: []Problem{
				{ID: 1929, Slug: "concatenation-of-array", Title: "Concatenation of Array", Difficulty: "Easy"},
				{ID: 1480, Slug: "running-sum-of-1d-array", Title: "Running Sum of 1d Array", Difficulty: "Easy"},
				{ID: 66, Slug: "plus-one", Title: "Plus One", Difficulty: "Easy"},
			}, Optional: []Problem{
				{ID: 118, Slug: "pascals-triangle", Title: "Pascal's Triangle", Difficulty: "Easy"},
				{ID: 119, Slug: "pascals-triangle-ii", Title: "Pascal's Triangle II", Difficulty: "Easy"},
			}},
			{Number: 7, Title: "Matrix coordinates and mutation", Lesson: "day-07", Readings: lessonReadings(7), Core: []Problem{
				{ID: 54, Slug: "spiral-matrix", Title: "Spiral Matrix", Difficulty: "Medium"},
				{ID: 48, Slug: "rotate-image", Title: "Rotate Image", Difficulty: "Medium"},
				{ID: 73, Slug: "set-matrix-zeroes", Title: "Set Matrix Zeroes", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 189, Slug: "rotate-array", Title: "Rotate Array", Difficulty: "Medium"},
			}},
		}},
		{Number: 2, Title: "Binary search", Summary: "Search explicit order, boundaries, and monotone answer spaces. Prove what each discarded range cannot contain.", Days: []Day{
			{Number: 8, Title: "Binary search intervals", Lesson: "day-08", Readings: lessonReadings(8), Core: []Problem{
				{ID: 704, Slug: "binary-search", Title: "Binary Search", Difficulty: "Easy"},
				{ID: 35, Slug: "search-insert-position", Title: "Search Insert Position", Difficulty: "Easy"},
				{ID: 374, Slug: "guess-number-higher-or-lower", Title: "Guess Number Higher or Lower", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 9, Title: "Monotone predicates on integers", Lesson: "day-09", Readings: lessonReadings(9), Core: []Problem{
				{ID: 278, Slug: "first-bad-version", Title: "First Bad Version", Difficulty: "Easy"},
				{ID: 367, Slug: "valid-perfect-square", Title: "Valid Perfect Square", Difficulty: "Easy"},
				{ID: 69, Slug: "sqrtx", Title: "Sqrt(x)", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 10, Title: "Lower and upper boundaries", Lesson: "day-10", Readings: lessonReadings(10), Core: []Problem{
				{ID: 34, Slug: "find-first-and-last-position-of-element-in-sorted-array", Title: "Find First and Last Position of Element in Sorted Array", Difficulty: "Medium"},
				{ID: 744, Slug: "find-smallest-letter-greater-than-target", Title: "Find Smallest Letter Greater Than Target", Difficulty: "Easy"},
				{ID: 1351, Slug: "count-negative-numbers-in-a-sorted-matrix", Title: "Count Negative Numbers in a Sorted Matrix", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 11, Title: "Search structured matrices and rotations", Lesson: "day-11", Readings: lessonReadings(11), Core: []Problem{
				{ID: 74, Slug: "search-a-2d-matrix", Title: "Search a 2D Matrix", Difficulty: "Medium"},
				{ID: 240, Slug: "search-a-2d-matrix-ii", Title: "Search a 2D Matrix II", Difficulty: "Medium"},
				{ID: 153, Slug: "find-minimum-in-rotated-sorted-array", Title: "Find Minimum in Rotated Sorted Array", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 12, Title: "Rotated search and local peaks", Lesson: "day-12", Readings: lessonReadings(12), Core: []Problem{
				{ID: 33, Slug: "search-in-rotated-sorted-array", Title: "Search in Rotated Sorted Array", Difficulty: "Medium"},
				{ID: 81, Slug: "search-in-rotated-sorted-array-ii", Title: "Search in Rotated Sorted Array II", Difficulty: "Medium"},
				{ID: 162, Slug: "find-peak-element", Title: "Find Peak Element", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 154, Slug: "find-minimum-in-rotated-sorted-array-ii", Title: "Find Minimum in Rotated Sorted Array II", Difficulty: "Hard"},
			}},
			{Number: 13, Title: "Search the answer with feasibility", Lesson: "day-13", Readings: lessonReadings(13), Core: []Problem{
				{ID: 875, Slug: "koko-eating-bananas", Title: "Koko Eating Bananas", Difficulty: "Medium"},
				{ID: 1011, Slug: "capacity-to-ship-packages-within-d-days", Title: "Capacity To Ship Packages Within D Days", Difficulty: "Medium"},
				{ID: 1283, Slug: "find-the-smallest-divisor-given-a-threshold", Title: "Find the Smallest Divisor Given a Threshold", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 410, Slug: "split-array-largest-sum", Title: "Split Array Largest Sum", Difficulty: "Hard"},
			}},
			{Number: 14, Title: "Binary search review and pair counts", Lesson: "day-14", Readings: lessonReadings(14), Core: []Problem{
				{ID: 1482, Slug: "minimum-number-of-days-to-make-m-bouquets", Title: "Minimum Number of Days to Make m Bouquets", Difficulty: "Medium"},
				{ID: 1855, Slug: "maximum-distance-between-a-pair-of-values", Title: "Maximum Distance Between a Pair of Values", Difficulty: "Medium"},
				{ID: 2300, Slug: "successful-pairs-of-spells-and-potions", Title: "Successful Pairs of Spells and Potions", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 1760, Slug: "minimum-limit-of-balls-in-a-bag", Title: "Minimum Limit of Balls in a Bag", Difficulty: "Medium"},
				{ID: 2064, Slug: "minimized-maximum-of-products-distributed-to-any-store", Title: "Minimized Maximum of Products Distributed to Any Store", Difficulty: "Medium"},
			}},
		}},
		{Number: 3, Title: "Two pointers, windows, and prefix sums", Summary: "Use order and incremental summaries to avoid repeating work. Check the assumptions behind each moving boundary.", Days: []Day{
			{Number: 15, Title: "Opposing pointers", Lesson: "day-15", Readings: lessonReadings(15), Core: []Problem{
				{ID: 125, Slug: "valid-palindrome", Title: "Valid Palindrome", Difficulty: "Easy"},
				{ID: 344, Slug: "reverse-string", Title: "Reverse String", Difficulty: "Easy"},
				{ID: 167, Slug: "two-sum-ii-input-array-is-sorted", Title: "Two Sum II - Input Array Is Sorted", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 16, Title: "Read and write pointers", Lesson: "day-16", Readings: lessonReadings(16), Core: []Problem{
				{ID: 26, Slug: "remove-duplicates-from-sorted-array", Title: "Remove Duplicates from Sorted Array", Difficulty: "Easy"},
				{ID: 27, Slug: "remove-element", Title: "Remove Element", Difficulty: "Easy"},
				{ID: 283, Slug: "move-zeroes", Title: "Move Zeroes", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 17, Title: "Eliminate pairs without skipping answers", Lesson: "day-17", Readings: lessonReadings(17), Core: []Problem{
				{ID: 11, Slug: "container-with-most-water", Title: "Container With Most Water", Difficulty: "Medium"},
				{ID: 15, Slug: "3sum", Title: "3Sum", Difficulty: "Medium"},
				{ID: 16, Slug: "3sum-closest", Title: "3Sum Closest", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 18, Slug: "4sum", Title: "4Sum", Difficulty: "Medium"},
			}},
			{Number: 18, Title: "Fixed-size windows", Lesson: "day-18", Readings: lessonReadings(18), Core: []Problem{
				{ID: 643, Slug: "maximum-average-subarray-i", Title: "Maximum Average Subarray I", Difficulty: "Easy"},
				{ID: 1456, Slug: "maximum-number-of-vowels-in-a-substring-of-given-length", Title: "Maximum Number of Vowels in a Substring of Given Length", Difficulty: "Medium"},
				{ID: 1343, Slug: "number-of-sub-arrays-of-size-k-and-average-greater-than-or-equal-to-threshold", Title: "Number of Sub-arrays of Size K and Average Greater than or Equal to Threshold", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 438, Slug: "find-all-anagrams-in-a-string", Title: "Find All Anagrams in a String", Difficulty: "Medium"},
			}},
			{Number: 19, Title: "Variable windows and validity", Lesson: "day-19", Readings: lessonReadings(19), Core: []Problem{
				{ID: 3, Slug: "longest-substring-without-repeating-characters", Title: "Longest Substring Without Repeating Characters", Difficulty: "Medium"},
				{ID: 209, Slug: "minimum-size-subarray-sum", Title: "Minimum Size Subarray Sum", Difficulty: "Medium"},
				{ID: 1004, Slug: "max-consecutive-ones-iii", Title: "Max Consecutive Ones III", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 424, Slug: "longest-repeating-character-replacement", Title: "Longest Repeating Character Replacement", Difficulty: "Medium"},
				{ID: 904, Slug: "fruit-into-baskets", Title: "Fruit Into Baskets", Difficulty: "Medium"},
			}},
			{Number: 20, Title: "Prefix sums and frequency maps", Lesson: "day-20", Readings: lessonReadings(20), Core: []Problem{
				{ID: 303, Slug: "range-sum-query-immutable", Title: "Range Sum Query - Immutable", Difficulty: "Easy"},
				{ID: 724, Slug: "find-pivot-index", Title: "Find Pivot Index", Difficulty: "Easy"},
				{ID: 560, Slug: "subarray-sum-equals-k", Title: "Subarray Sum Equals K", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 21, Title: "Prefix products and modular state", Lesson: "day-21", Readings: lessonReadings(21), Core: []Problem{
				{ID: 238, Slug: "product-of-array-except-self", Title: "Product of Array Except Self", Difficulty: "Medium"},
				{ID: 523, Slug: "continuous-subarray-sum", Title: "Continuous Subarray Sum", Difficulty: "Medium"},
				{ID: 974, Slug: "subarray-sums-divisible-by-k", Title: "Subarray Sums Divisible by K", Difficulty: "Medium"},
			}, Optional: []Problem{}},
		}},
		{Number: 4, Title: "Linked lists, stacks, and recursion", Summary: "Track node identity, pending work, and recursive contracts. Draw mutations before changing references.", Days: []Day{
			{Number: 22, Title: "Linked nodes and local rewiring", Lesson: "day-22", Readings: lessonReadings(22), Core: []Problem{
				{ID: 206, Slug: "reverse-linked-list", Title: "Reverse Linked List", Difficulty: "Easy"},
				{ID: 21, Slug: "merge-two-sorted-lists", Title: "Merge Two Sorted Lists", Difficulty: "Easy"},
				{ID: 83, Slug: "remove-duplicates-from-sorted-list", Title: "Remove Duplicates from Sorted List", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 23, Title: "Fast and slow references", Lesson: "day-23", Readings: lessonReadings(23), Core: []Problem{
				{ID: 876, Slug: "middle-of-the-linked-list", Title: "Middle of the Linked List", Difficulty: "Easy"},
				{ID: 141, Slug: "linked-list-cycle", Title: "Linked List Cycle", Difficulty: "Easy"},
				{ID: 160, Slug: "intersection-of-two-linked-lists", Title: "Intersection of Two Linked Lists", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 24, Title: "Gaps, cycles, and half-list comparisons", Lesson: "day-24", Readings: lessonReadings(24), Core: []Problem{
				{ID: 19, Slug: "remove-nth-node-from-end-of-list", Title: "Remove Nth Node From End of List", Difficulty: "Medium"},
				{ID: 142, Slug: "linked-list-cycle-ii", Title: "Linked List Cycle II", Difficulty: "Medium"},
				{ID: 234, Slug: "palindrome-linked-list", Title: "Palindrome Linked List", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 25, Title: "Sentinels and stable list partitions", Lesson: "day-25", Readings: lessonReadings(25), Core: []Problem{
				{ID: 203, Slug: "remove-linked-list-elements", Title: "Remove Linked List Elements", Difficulty: "Easy"},
				{ID: 328, Slug: "odd-even-linked-list", Title: "Odd Even Linked List", Difficulty: "Medium"},
				{ID: 24, Slug: "swap-nodes-in-pairs", Title: "Swap Nodes in Pairs", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 445, Slug: "add-two-numbers-ii", Title: "Add Two Numbers II", Difficulty: "Medium"},
				{ID: 143, Slug: "reorder-list", Title: "Reorder List", Difficulty: "Medium"},
				{ID: 707, Slug: "design-linked-list", Title: "Design Linked List", Difficulty: "Medium"},
			}},
			{Number: 26, Title: "Stacks as unfinished work", Lesson: "day-26", Readings: lessonReadings(26), Core: []Problem{
				{ID: 20, Slug: "valid-parentheses", Title: "Valid Parentheses", Difficulty: "Easy"},
				{ID: 155, Slug: "min-stack", Title: "Min Stack", Difficulty: "Medium"},
				{ID: 150, Slug: "evaluate-reverse-polish-notation", Title: "Evaluate Reverse Polish Notation", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 394, Slug: "decode-string", Title: "Decode String", Difficulty: "Medium"},
			}},
			{Number: 27, Title: "Implement one interface with another", Lesson: "day-27", Readings: lessonReadings(27), Core: []Problem{
				{ID: 232, Slug: "implement-queue-using-stacks", Title: "Implement Queue using Stacks", Difficulty: "Easy"},
				{ID: 225, Slug: "implement-stack-using-queues", Title: "Implement Stack using Queues", Difficulty: "Easy"},
				{ID: 682, Slug: "baseball-game", Title: "Baseball Game", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 28, Title: "Recursion and smaller subproblems", Lesson: "day-28", Readings: lessonReadings(28), Core: []Problem{
				{ID: 509, Slug: "fibonacci-number", Title: "Fibonacci Number", Difficulty: "Easy"},
				{ID: 50, Slug: "powx-n", Title: "Pow(x, n)", Difficulty: "Medium"},
				{ID: 779, Slug: "k-th-symbol-in-grammar", Title: "K-th Symbol in Grammar", Difficulty: "Medium"},
			}, Optional: []Problem{}},
		}},
		{Number: 5, Title: "Trees", Summary: "Build traversals from subtree contracts, then use ordering, levels, and path state to answer structural questions.", Days: []Day{
			{Number: 29, Title: "Tree recursion contracts", Lesson: "day-29", Readings: lessonReadings(29), Core: []Problem{
				{ID: 104, Slug: "maximum-depth-of-binary-tree", Title: "Maximum Depth of Binary Tree", Difficulty: "Easy"},
				{ID: 111, Slug: "minimum-depth-of-binary-tree", Title: "Minimum Depth of Binary Tree", Difficulty: "Easy"},
				{ID: 226, Slug: "invert-binary-tree", Title: "Invert Binary Tree", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 30, Title: "Compare tree structure", Lesson: "day-30", Readings: lessonReadings(30), Core: []Problem{
				{ID: 100, Slug: "same-tree", Title: "Same Tree", Difficulty: "Easy"},
				{ID: 101, Slug: "symmetric-tree", Title: "Symmetric Tree", Difficulty: "Easy"},
				{ID: 572, Slug: "subtree-of-another-tree", Title: "Subtree of Another Tree", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 31, Title: "Paths and branch-local state", Lesson: "day-31", Readings: lessonReadings(31), Core: []Problem{
				{ID: 112, Slug: "path-sum", Title: "Path Sum", Difficulty: "Easy"},
				{ID: 113, Slug: "path-sum-ii", Title: "Path Sum II", Difficulty: "Medium"},
				{ID: 257, Slug: "binary-tree-paths", Title: "Binary Tree Paths", Difficulty: "Easy"},
			}, Optional: []Problem{
				{ID: 129, Slug: "sum-root-to-leaf-numbers", Title: "Sum Root to Leaf Numbers", Difficulty: "Medium"},
			}},
			{Number: 32, Title: "Tree breadth-first search", Lesson: "day-32", Readings: lessonReadings(32), Core: []Problem{
				{ID: 102, Slug: "binary-tree-level-order-traversal", Title: "Binary Tree Level Order Traversal", Difficulty: "Medium"},
				{ID: 107, Slug: "binary-tree-level-order-traversal-ii", Title: "Binary Tree Level Order Traversal II", Difficulty: "Medium"},
				{ID: 199, Slug: "binary-tree-right-side-view", Title: "Binary Tree Right Side View", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 33, Title: "Binary search tree bounds", Lesson: "day-33", Readings: lessonReadings(33), Core: []Problem{
				{ID: 98, Slug: "validate-binary-search-tree", Title: "Validate Binary Search Tree", Difficulty: "Medium"},
				{ID: 700, Slug: "search-in-a-binary-search-tree", Title: "Search in a Binary Search Tree", Difficulty: "Easy"},
				{ID: 701, Slug: "insert-into-a-binary-search-tree", Title: "Insert into a Binary Search Tree", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 34, Title: "Ancestors and ordered traversal", Lesson: "day-34", Readings: lessonReadings(34), Core: []Problem{
				{ID: 235, Slug: "lowest-common-ancestor-of-a-binary-search-tree", Title: "Lowest Common Ancestor of a Binary Search Tree", Difficulty: "Medium"},
				{ID: 236, Slug: "lowest-common-ancestor-of-a-binary-tree", Title: "Lowest Common Ancestor of a Binary Tree", Difficulty: "Medium"},
				{ID: 230, Slug: "kth-smallest-element-in-a-bst", Title: "Kth Smallest Element in a BST", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 35, Title: "Return local facts, update global answers", Lesson: "day-35", Readings: lessonReadings(35), Core: []Problem{
				{ID: 543, Slug: "diameter-of-binary-tree", Title: "Diameter of Binary Tree", Difficulty: "Easy"},
				{ID: 110, Slug: "balanced-binary-tree", Title: "Balanced Binary Tree", Difficulty: "Easy"},
				{ID: 637, Slug: "average-of-levels-in-binary-tree", Title: "Average of Levels in Binary Tree", Difficulty: "Easy"},
			}, Optional: []Problem{
				{ID: 105, Slug: "construct-binary-tree-from-preorder-and-inorder-traversal", Title: "Construct Binary Tree from Preorder and Inorder Traversal", Difficulty: "Medium"},
				{ID: 106, Slug: "construct-binary-tree-from-inorder-and-postorder-traversal", Title: "Construct Binary Tree from Inorder and Postorder Traversal", Difficulty: "Medium"},
				{ID: 297, Slug: "serialize-and-deserialize-binary-tree", Title: "Serialize and Deserialize Binary Tree", Difficulty: "Hard"},
			}},
		}},
		{Number: 6, Title: "Backtracking", Summary: "Enumerate decisions with explicit state and restoration. Prune only when a branch cannot produce a valid answer.", Days: []Day{
			{Number: 36, Title: "Decision trees and reversible choices", Lesson: "day-36", Readings: lessonReadings(36), Core: []Problem{
				{ID: 784, Slug: "letter-case-permutation", Title: "Letter Case Permutation", Difficulty: "Medium"},
				{ID: 1863, Slug: "sum-of-all-subset-xor-totals", Title: "Sum of All Subset XOR Totals", Difficulty: "Easy"},
				{ID: 401, Slug: "binary-watch", Title: "Binary Watch", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 37, Title: "Subsets, combinations, and permutations", Lesson: "day-37", Readings: lessonReadings(37), Core: []Problem{
				{ID: 78, Slug: "subsets", Title: "Subsets", Difficulty: "Medium"},
				{ID: 77, Slug: "combinations", Title: "Combinations", Difficulty: "Medium"},
				{ID: 46, Slug: "permutations", Title: "Permutations", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 38, Title: "Duplicate-aware enumeration", Lesson: "day-38", Readings: lessonReadings(38), Core: []Problem{
				{ID: 90, Slug: "subsets-ii", Title: "Subsets II", Difficulty: "Medium"},
				{ID: 47, Slug: "permutations-ii", Title: "Permutations II", Difficulty: "Medium"},
				{ID: 491, Slug: "non-decreasing-subsequences", Title: "Non-decreasing Subsequences", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 39, Title: "Combination sums and pruning", Lesson: "day-39", Readings: lessonReadings(39), Core: []Problem{
				{ID: 39, Slug: "combination-sum", Title: "Combination Sum", Difficulty: "Medium"},
				{ID: 40, Slug: "combination-sum-ii", Title: "Combination Sum II", Difficulty: "Medium"},
				{ID: 216, Slug: "combination-sum-iii", Title: "Combination Sum III", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 40, Title: "Partition strings into valid pieces", Lesson: "day-40", Readings: lessonReadings(40), Core: []Problem{
				{ID: 17, Slug: "letter-combinations-of-a-phone-number", Title: "Letter Combinations of a Phone Number", Difficulty: "Medium"},
				{ID: 93, Slug: "restore-ip-addresses", Title: "Restore IP Addresses", Difficulty: "Medium"},
				{ID: 131, Slug: "palindrome-partitioning", Title: "Palindrome Partitioning", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 41, Title: "Backtracking with coupled constraints", Lesson: "day-41", Readings: lessonReadings(41), Core: []Problem{
				{ID: 79, Slug: "word-search", Title: "Word Search", Difficulty: "Medium"},
				{ID: 306, Slug: "additive-number", Title: "Additive Number", Difficulty: "Medium"},
				{ID: 1219, Slug: "path-with-maximum-gold", Title: "Path with Maximum Gold", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 42, Title: "Symmetry and constraint-first search", Lesson: "day-42", Readings: lessonReadings(42), Core: []Problem{
				{ID: 526, Slug: "beautiful-arrangement", Title: "Beautiful Arrangement", Difficulty: "Medium"},
				{ID: 473, Slug: "matchsticks-to-square", Title: "Matchsticks to Square", Difficulty: "Medium"},
				{ID: 1239, Slug: "maximum-length-of-a-concatenated-string-with-unique-characters", Title: "Maximum Length of a Concatenated String with Unique Characters", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 51, Slug: "n-queens", Title: "N-Queens", Difficulty: "Hard"},
				{ID: 22, Slug: "generate-parentheses", Title: "Generate Parentheses", Difficulty: "Medium"},
				{ID: 37, Slug: "sudoku-solver", Title: "Sudoku Solver", Difficulty: "Hard"},
				{ID: 698, Slug: "partition-to-k-equal-sum-subsets", Title: "Partition to K Equal Sum Subsets", Difficulty: "Medium"},
			}},
		}},
		{Number: 7, Title: "Graphs", Summary: "Model connectivity, distances, dependencies, and equivalence. Match traversal state to the question being asked.", Days: []Day{
			{Number: 43, Title: "Graphs hidden in grids", Lesson: "day-43", Readings: lessonReadings(43), Core: []Problem{
				{ID: 733, Slug: "flood-fill", Title: "Flood Fill", Difficulty: "Easy"},
				{ID: 200, Slug: "number-of-islands", Title: "Number of Islands", Difficulty: "Medium"},
				{ID: 695, Slug: "max-area-of-island", Title: "Max Area of Island", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 44, Title: "Components and boundary reachability", Lesson: "day-44", Readings: lessonReadings(44), Core: []Problem{
				{ID: 463, Slug: "island-perimeter", Title: "Island Perimeter", Difficulty: "Easy"},
				{ID: 547, Slug: "number-of-provinces", Title: "Number of Provinces", Difficulty: "Medium"},
				{ID: 130, Slug: "surrounded-regions", Title: "Surrounded Regions", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 45, Title: "Breadth-first distances and multiple sources", Lesson: "day-45", Readings: lessonReadings(45), Core: []Problem{
				{ID: 994, Slug: "rotting-oranges", Title: "Rotting Oranges", Difficulty: "Medium"},
				{ID: 542, Slug: "01-matrix", Title: "01 Matrix", Difficulty: "Medium"},
				{ID: 1091, Slug: "shortest-path-in-binary-matrix", Title: "Shortest Path in Binary Matrix", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 127, Slug: "word-ladder", Title: "Word Ladder", Difficulty: "Hard"},
			}},
			{Number: 46, Title: "Explicit and implicit adjacency", Lesson: "day-46", Readings: lessonReadings(46), Core: []Problem{
				{ID: 797, Slug: "all-paths-from-source-to-target", Title: "All Paths From Source to Target", Difficulty: "Medium"},
				{ID: 841, Slug: "keys-and-rooms", Title: "Keys and Rooms", Difficulty: "Medium"},
				{ID: 1971, Slug: "find-if-path-exists-in-graph", Title: "Find if Path Exists in Graph", Difficulty: "Easy"},
			}, Optional: []Problem{
				{ID: 399, Slug: "evaluate-division", Title: "Evaluate Division", Difficulty: "Medium"},
			}},
			{Number: 47, Title: "Directed dependencies and topological order", Lesson: "day-47", Readings: lessonReadings(47), Core: []Problem{
				{ID: 207, Slug: "course-schedule", Title: "Course Schedule", Difficulty: "Medium"},
				{ID: 210, Slug: "course-schedule-ii", Title: "Course Schedule II", Difficulty: "Medium"},
				{ID: 802, Slug: "find-eventual-safe-states", Title: "Find Eventual Safe States", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 1557, Slug: "minimum-number-of-vertices-to-reach-all-nodes", Title: "Minimum Number of Vertices to Reach All Nodes", Difficulty: "Medium"},
			}},
			{Number: 48, Title: "Disjoint sets and equivalence", Lesson: "day-48", Readings: lessonReadings(48), Core: []Problem{
				{ID: 684, Slug: "redundant-connection", Title: "Redundant Connection", Difficulty: "Medium"},
				{ID: 721, Slug: "accounts-merge", Title: "Accounts Merge", Difficulty: "Medium"},
				{ID: 990, Slug: "satisfiability-of-equality-equations", Title: "Satisfiability of Equality Equations", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 1319, Slug: "number-of-operations-to-make-network-connected", Title: "Number of Operations to Make Network Connected", Difficulty: "Medium"},
			}},
			{Number: 49, Title: "Weighted paths and state dimensions", Lesson: "day-49", Readings: lessonReadings(49), Core: []Problem{
				{ID: 743, Slug: "network-delay-time", Title: "Network Delay Time", Difficulty: "Medium"},
				{ID: 787, Slug: "cheapest-flights-within-k-stops", Title: "Cheapest Flights Within K Stops", Difficulty: "Medium"},
				{ID: 1631, Slug: "path-with-minimum-effort", Title: "Path With Minimum Effort", Difficulty: "Medium"},
			}, Optional: []Problem{}},
		}},
		{Number: 8, Title: "Heaps and tries", Summary: "Maintain ranked frontiers and shared prefixes. Compare specialized structures with sorting and simpler scans.", Days: []Day{
			{Number: 50, Title: "Heaps and top-k streams", Lesson: "day-50", Readings: lessonReadings(50), Core: []Problem{
				{ID: 703, Slug: "kth-largest-element-in-a-stream", Title: "Kth Largest Element in a Stream", Difficulty: "Easy"},
				{ID: 1046, Slug: "last-stone-weight", Title: "Last Stone Weight", Difficulty: "Easy"},
				{ID: 215, Slug: "kth-largest-element-in-an-array", Title: "Kth Largest Element in an Array", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 51, Title: "Ranking by derived keys", Lesson: "day-51", Readings: lessonReadings(51), Core: []Problem{
				{ID: 347, Slug: "top-k-frequent-elements", Title: "Top K Frequent Elements", Difficulty: "Medium"},
				{ID: 692, Slug: "top-k-frequent-words", Title: "Top K Frequent Words", Difficulty: "Medium"},
				{ID: 973, Slug: "k-closest-points-to-origin", Title: "K Closest Points to Origin", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 52, Title: "Frequency ordering and greedy removal", Lesson: "day-52", Readings: lessonReadings(52), Core: []Problem{
				{ID: 451, Slug: "sort-characters-by-frequency", Title: "Sort Characters By Frequency", Difficulty: "Medium"},
				{ID: 1338, Slug: "reduce-array-size-to-the-half", Title: "Reduce Array Size to The Half", Difficulty: "Medium"},
				{ID: 1636, Slug: "sort-array-by-increasing-frequency", Title: "Sort Array by Increasing Frequency", Difficulty: "Easy"},
			}, Optional: []Problem{
				{ID: 1962, Slug: "remove-stones-to-minimize-the-total", Title: "Remove Stones to Minimize the Total", Difficulty: "Medium"},
			}},
			{Number: 53, Title: "Merge sorted streams with a heap", Lesson: "day-53", Readings: lessonReadings(53), Core: []Problem{
				{ID: 378, Slug: "kth-smallest-element-in-a-sorted-matrix", Title: "Kth Smallest Element in a Sorted Matrix", Difficulty: "Medium"},
				{ID: 373, Slug: "find-k-pairs-with-smallest-sums", Title: "Find K Pairs with Smallest Sums", Difficulty: "Medium"},
				{ID: 786, Slug: "k-th-smallest-prime-fraction", Title: "K-th Smallest Prime Fraction", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 54, Title: "Tries and prefix state", Lesson: "day-54", Readings: lessonReadings(54), Core: []Problem{
				{ID: 208, Slug: "implement-trie-prefix-tree", Title: "Implement Trie (Prefix Tree)", Difficulty: "Medium"},
				{ID: 211, Slug: "design-add-and-search-words-data-structure", Title: "Design Add and Search Words Data Structure", Difficulty: "Medium"},
				{ID: 677, Slug: "map-sum-pairs", Title: "Map Sum Pairs", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 55, Title: "Prefix choices and lexicographic order", Lesson: "day-55", Readings: lessonReadings(55), Core: []Problem{
				{ID: 648, Slug: "replace-words", Title: "Replace Words", Difficulty: "Medium"},
				{ID: 720, Slug: "longest-word-in-dictionary", Title: "Longest Word in Dictionary", Difficulty: "Medium"},
				{ID: 1268, Slug: "search-suggestions-system", Title: "Search Suggestions System", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 212, Slug: "word-search-ii", Title: "Word Search II", Difficulty: "Hard"},
			}},
			{Number: 56, Title: "Scheduling with available choices", Lesson: "day-56", Readings: lessonReadings(56), Core: []Problem{
				{ID: 1834, Slug: "single-threaded-cpu", Title: "Single-Threaded CPU", Difficulty: "Medium"},
				{ID: 767, Slug: "reorganize-string", Title: "Reorganize String", Difficulty: "Medium"},
				{ID: 621, Slug: "task-scheduler", Title: "Task Scheduler", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 295, Slug: "find-median-from-data-stream", Title: "Find Median from Data Stream", Difficulty: "Hard"},
				{ID: 1405, Slug: "longest-happy-string", Title: "Longest Happy String", Difficulty: "Medium"},
			}},
		}},
		{Number: 9, Title: "Basic dynamic programming", Summary: "Define subproblems and transitions before compressing storage. Learn choice-or-skip, sequence, and knapsack states.", Days: []Day{
			{Number: 57, Title: "Dynamic programming as cached contracts", Lesson: "day-57", Readings: lessonReadings(57), Core: []Problem{
				{ID: 70, Slug: "climbing-stairs", Title: "Climbing Stairs", Difficulty: "Easy"},
				{ID: 1137, Slug: "n-th-tribonacci-number", Title: "N-th Tribonacci Number", Difficulty: "Easy"},
				{ID: 746, Slug: "min-cost-climbing-stairs", Title: "Min Cost Climbing Stairs", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 58, Title: "Choose or skip with adjacency constraints", Lesson: "day-58", Readings: lessonReadings(58), Core: []Problem{
				{ID: 198, Slug: "house-robber", Title: "House Robber", Difficulty: "Medium"},
				{ID: 213, Slug: "house-robber-ii", Title: "House Robber II", Difficulty: "Medium"},
				{ID: 740, Slug: "delete-and-earn", Title: "Delete and Earn", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 59, Title: "Unbounded choices and decoding", Lesson: "day-59", Readings: lessonReadings(59), Core: []Problem{
				{ID: 322, Slug: "coin-change", Title: "Coin Change", Difficulty: "Medium"},
				{ID: 279, Slug: "perfect-squares", Title: "Perfect Squares", Difficulty: "Medium"},
				{ID: 91, Slug: "decode-ways", Title: "Decode Ways", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 377, Slug: "combination-sum-iv", Title: "Combination Sum IV", Difficulty: "Medium"},
			}},
			{Number: 60, Title: "State machines and suffix feasibility", Lesson: "day-60", Readings: lessonReadings(60), Core: []Problem{
				{ID: 139, Slug: "word-break", Title: "Word Break", Difficulty: "Medium"},
				{ID: 309, Slug: "best-time-to-buy-and-sell-stock-with-cooldown", Title: "Best Time to Buy and Sell Stock with Cooldown", Difficulty: "Medium"},
				{ID: 714, Slug: "best-time-to-buy-and-sell-stock-with-transaction-fee", Title: "Best Time to Buy and Sell Stock with Transaction Fee", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 61, Title: "Sequences with predecessor choices", Lesson: "day-61", Readings: lessonReadings(61), Core: []Problem{
				{ID: 300, Slug: "longest-increasing-subsequence", Title: "Longest Increasing Subsequence", Difficulty: "Medium"},
				{ID: 646, Slug: "maximum-length-of-pair-chain", Title: "Maximum Length of Pair Chain", Difficulty: "Medium"},
				{ID: 673, Slug: "number-of-longest-increasing-subsequence", Title: "Number of Longest Increasing Subsequence", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 62, Title: "Subset sums and loop direction", Lesson: "day-62", Readings: lessonReadings(62), Core: []Problem{
				{ID: 416, Slug: "partition-equal-subset-sum", Title: "Partition Equal Subset Sum", Difficulty: "Medium"},
				{ID: 494, Slug: "target-sum", Title: "Target Sum", Difficulty: "Medium"},
				{ID: 1049, Slug: "last-stone-weight-ii", Title: "Last Stone Weight II", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 518, Slug: "coin-change-ii", Title: "Coin Change II", Difficulty: "Medium"},
			}},
			{Number: 63, Title: "Local ending states and global optima", Lesson: "day-63", Readings: lessonReadings(63), Core: []Problem{
				{ID: 152, Slug: "maximum-product-subarray", Title: "Maximum Product Subarray", Difficulty: "Medium"},
				{ID: 53, Slug: "maximum-subarray", Title: "Maximum Subarray", Difficulty: "Medium"},
				{ID: 121, Slug: "best-time-to-buy-and-sell-stock", Title: "Best Time to Buy and Sell Stock", Difficulty: "Easy"},
			}, Optional: []Problem{
				{ID: 343, Slug: "integer-break", Title: "Integer Break", Difficulty: "Medium"},
				{ID: 413, Slug: "arithmetic-slices", Title: "Arithmetic Slices", Difficulty: "Medium"},
			}},
		}},
		{Number: 10, Title: "Multidimensional dynamic programming", Summary: "Handle grid, sequence-pair, resource, and interval dependencies. Derive evaluation order from the recurrence.", Days: []Day{
			{Number: 64, Title: "Grid DP and blocked states", Lesson: "day-64", Readings: lessonReadings(64), Core: []Problem{
				{ID: 62, Slug: "unique-paths", Title: "Unique Paths", Difficulty: "Medium"},
				{ID: 63, Slug: "unique-paths-ii", Title: "Unique Paths II", Difficulty: "Medium"},
				{ID: 64, Slug: "minimum-path-sum", Title: "Minimum Path Sum", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 65, Title: "Local geometry in grid states", Lesson: "day-65", Readings: lessonReadings(65), Core: []Problem{
				{ID: 120, Slug: "triangle", Title: "Triangle", Difficulty: "Medium"},
				{ID: 931, Slug: "minimum-falling-path-sum", Title: "Minimum Falling Path Sum", Difficulty: "Medium"},
				{ID: 221, Slug: "maximal-square", Title: "Maximal Square", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 1277, Slug: "count-square-submatrices-with-all-ones", Title: "Count Square Submatrices with All Ones", Difficulty: "Medium"},
			}},
			{Number: 66, Title: "Two-sequence alignment", Lesson: "day-66", Readings: lessonReadings(66), Core: []Problem{
				{ID: 1143, Slug: "longest-common-subsequence", Title: "Longest Common Subsequence", Difficulty: "Medium"},
				{ID: 583, Slug: "delete-operation-for-two-strings", Title: "Delete Operation for Two Strings", Difficulty: "Medium"},
				{ID: 1035, Slug: "uncrossed-lines", Title: "Uncrossed Lines", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 67, Title: "Edit choices and palindrome intervals", Lesson: "day-67", Readings: lessonReadings(67), Core: []Problem{
				{ID: 72, Slug: "edit-distance", Title: "Edit Distance", Difficulty: "Medium"},
				{ID: 516, Slug: "longest-palindromic-subsequence", Title: "Longest Palindromic Subsequence", Difficulty: "Medium"},
				{ID: 647, Slug: "palindromic-substrings", Title: "Palindromic Substrings", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 68, Title: "Counting matches and interleaving", Lesson: "day-68", Readings: lessonReadings(68), Core: []Problem{
				{ID: 115, Slug: "distinct-subsequences", Title: "Distinct Subsequences", Difficulty: "Hard"},
				{ID: 718, Slug: "maximum-length-of-repeated-subarray", Title: "Maximum Length of Repeated Subarray", Difficulty: "Medium"},
				{ID: 97, Slug: "interleaving-string", Title: "Interleaving String", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 69, Title: "More than one resource", Lesson: "day-69", Readings: lessonReadings(69), Core: []Problem{
				{ID: 474, Slug: "ones-and-zeroes", Title: "Ones and Zeroes", Difficulty: "Medium"},
				{ID: 879, Slug: "profitable-schemes", Title: "Profitable Schemes", Difficulty: "Hard"},
				{ID: 1155, Slug: "number-of-dice-rolls-with-target-sum", Title: "Number of Dice Rolls With Target Sum", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 741, Slug: "cherry-pickup", Title: "Cherry Pickup", Difficulty: "Hard"},
				{ID: 1463, Slug: "cherry-pickup-ii", Title: "Cherry Pickup II", Difficulty: "Hard"},
			}},
			{Number: 70, Title: "Interval games and last actions", Lesson: "day-70", Readings: lessonReadings(70), Core: []Problem{
				{ID: 877, Slug: "stone-game", Title: "Stone Game", Difficulty: "Medium"},
				{ID: 486, Slug: "predict-the-winner", Title: "Predict the Winner", Difficulty: "Medium"},
				{ID: 312, Slug: "burst-balloons", Title: "Burst Balloons", Difficulty: "Hard"},
			}, Optional: []Problem{
				{ID: 1335, Slug: "minimum-difficulty-of-a-job-schedule", Title: "Minimum Difficulty of a Job Schedule", Difficulty: "Hard"},
			}},
		}},
		{Number: 11, Title: "Greedy, intervals, and monotonic stacks", Summary: "Prove safe commitments, process interval order, and account for stack work over an entire scan.", Days: []Day{
			{Number: 71, Title: "Greedy choices need exchange arguments", Lesson: "day-71", Readings: lessonReadings(71), Core: []Problem{
				{ID: 455, Slug: "assign-cookies", Title: "Assign Cookies", Difficulty: "Easy"},
				{ID: 860, Slug: "lemonade-change", Title: "Lemonade Change", Difficulty: "Easy"},
				{ID: 1710, Slug: "maximum-units-on-a-truck", Title: "Maximum Units on a Truck", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 72, Title: "Pairing and assignment by sorted order", Lesson: "day-72", Readings: lessonReadings(72), Core: []Problem{
				{ID: 561, Slug: "array-partition", Title: "Array Partition", Difficulty: "Easy"},
				{ID: 881, Slug: "boats-to-save-people", Title: "Boats to Save People", Difficulty: "Medium"},
				{ID: 1029, Slug: "two-city-scheduling", Title: "Two City Scheduling", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 846, Slug: "hand-of-straights", Title: "Hand of Straights", Difficulty: "Medium"},
				{ID: 406, Slug: "queue-reconstruction-by-height", Title: "Queue Reconstruction by Height", Difficulty: "Medium"},
			}},
			{Number: 73, Title: "Intervals and commitments", Lesson: "day-73", Readings: lessonReadings(73), Core: []Problem{
				{ID: 435, Slug: "non-overlapping-intervals", Title: "Non-overlapping Intervals", Difficulty: "Medium"},
				{ID: 452, Slug: "minimum-number-of-arrows-to-burst-balloons", Title: "Minimum Number of Arrows to Burst Balloons", Difficulty: "Medium"},
				{ID: 763, Slug: "partition-labels", Title: "Partition Labels", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 74, Title: "Merge and intersect intervals", Lesson: "day-74", Readings: lessonReadings(74), Core: []Problem{
				{ID: 56, Slug: "merge-intervals", Title: "Merge Intervals", Difficulty: "Medium"},
				{ID: 57, Slug: "insert-interval", Title: "Insert Interval", Difficulty: "Medium"},
				{ID: 986, Slug: "interval-list-intersections", Title: "Interval List Intersections", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 75, Title: "Greedy repair and reachable ranges", Lesson: "day-75", Readings: lessonReadings(75), Core: []Problem{
				{ID: 738, Slug: "monotone-increasing-digits", Title: "Monotone Increasing Digits", Difficulty: "Medium"},
				{ID: 134, Slug: "gas-station", Title: "Gas Station", Difficulty: "Medium"},
				{ID: 45, Slug: "jump-game-ii", Title: "Jump Game II", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 76, Title: "Monotonic stacks and unresolved indices", Lesson: "day-76", Readings: lessonReadings(76), Core: []Problem{
				{ID: 739, Slug: "daily-temperatures", Title: "Daily Temperatures", Difficulty: "Medium"},
				{ID: 496, Slug: "next-greater-element-i", Title: "Next Greater Element I", Difficulty: "Easy"},
				{ID: 503, Slug: "next-greater-element-ii", Title: "Next Greater Element II", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 84, Slug: "largest-rectangle-in-histogram", Title: "Largest Rectangle in Histogram", Difficulty: "Hard"},
			}},
			{Number: 77, Title: "Compressed stack summaries", Lesson: "day-77", Readings: lessonReadings(77), Core: []Problem{
				{ID: 901, Slug: "online-stock-span", Title: "Online Stock Span", Difficulty: "Medium"},
				{ID: 853, Slug: "car-fleet", Title: "Car Fleet", Difficulty: "Medium"},
				{ID: 402, Slug: "remove-k-digits", Title: "Remove K Digits", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 42, Slug: "trapping-rain-water", Title: "Trapping Rain Water", Difficulty: "Hard"},
			}},
		}},
		{Number: 12, Title: "Mixed practice", Summary: "Choose the model without a topic hint from the problem. Revisit assumptions, compare alternatives, and complete a timed mock.", Days: []Day{
			{Number: 78, Title: "Mixed practice: round 1", Lesson: "day-78", Readings: lessonReadings(78), Core: []Problem{
				{ID: 128, Slug: "longest-consecutive-sequence", Title: "Longest Consecutive Sequence", Difficulty: "Medium"},
				{ID: 287, Slug: "find-the-duplicate-number", Title: "Find the Duplicate Number", Difficulty: "Medium"},
				{ID: 338, Slug: "counting-bits", Title: "Counting Bits", Difficulty: "Easy"},
			}, Optional: []Problem{}},
			{Number: 79, Title: "Mixed practice: round 2", Lesson: "day-79", Readings: lessonReadings(79), Core: []Problem{
				{ID: 658, Slug: "find-k-closest-elements", Title: "Find K Closest Elements", Difficulty: "Medium"},
				{ID: 1248, Slug: "count-number-of-nice-subarrays", Title: "Count Number of Nice Subarrays", Difficulty: "Medium"},
				{ID: 930, Slug: "binary-subarrays-with-sum", Title: "Binary Subarrays With Sum", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 76, Slug: "minimum-window-substring", Title: "Minimum Window Substring", Difficulty: "Hard"},
			}},
			{Number: 80, Title: "Mixed practice: round 3", Lesson: "day-80", Readings: lessonReadings(80), Core: []Problem{
				{ID: 61, Slug: "rotate-list", Title: "Rotate List", Difficulty: "Medium"},
				{ID: 94, Slug: "binary-tree-inorder-traversal", Title: "Binary Tree Inorder Traversal", Difficulty: "Easy"},
				{ID: 430, Slug: "flatten-a-multilevel-doubly-linked-list", Title: "Flatten a Multilevel Doubly Linked List", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 23, Slug: "merge-k-sorted-lists", Title: "Merge k Sorted Lists", Difficulty: "Hard"},
			}},
			{Number: 81, Title: "Mixed practice: round 4", Lesson: "day-81", Readings: lessonReadings(81), Core: []Problem{
				{ID: 417, Slug: "pacific-atlantic-water-flow", Title: "Pacific Atlantic Water Flow", Difficulty: "Medium"},
				{ID: 133, Slug: "clone-graph", Title: "Clone Graph", Difficulty: "Medium"},
				{ID: 310, Slug: "minimum-height-trees", Title: "Minimum Height Trees", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 82, Title: "Mixed practice: round 5", Lesson: "day-82", Readings: lessonReadings(82), Core: []Problem{
				{ID: 1552, Slug: "magnetic-force-between-two-balls", Title: "Magnetic Force Between Two Balls", Difficulty: "Medium"},
				{ID: 1642, Slug: "furthest-building-you-can-reach", Title: "Furthest Building You Can Reach", Difficulty: "Medium"},
				{ID: 1980, Slug: "find-unique-binary-string", Title: "Find Unique Binary String", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 83, Title: "Mixed practice: round 6", Lesson: "day-83", Readings: lessonReadings(83), Core: []Problem{
				{ID: 1984, Slug: "minimum-difference-between-highest-and-lowest-of-k-scores", Title: "Minimum Difference Between Highest and Lowest of K Scores", Difficulty: "Easy"},
				{ID: 1140, Slug: "stone-game-ii", Title: "Stone Game II", Difficulty: "Medium"},
				{ID: 1024, Slug: "video-stitching", Title: "Video Stitching", Difficulty: "Medium"},
			}, Optional: []Problem{}},
			{Number: 84, Title: "Mock interview and invariant review", Lesson: "day-84", Readings: lessonReadings(84), Core: []Problem{
				{ID: 146, Slug: "lru-cache", Title: "LRU Cache", Difficulty: "Medium"},
				{ID: 886, Slug: "possible-bipartition", Title: "Possible Bipartition", Difficulty: "Medium"},
				{ID: 1209, Slug: "remove-all-adjacent-duplicates-in-string-ii", Title: "Remove All Adjacent Duplicates in String II", Difficulty: "Medium"},
			}, Optional: []Problem{
				{ID: 124, Slug: "binary-tree-maximum-path-sum", Title: "Binary Tree Maximum Path Sum", Difficulty: "Hard"},
				{ID: 1851, Slug: "minimum-interval-to-include-each-query", Title: "Minimum Interval to Include Each Query", Difficulty: "Hard"},
			}},
		}},
	}
}

func FindDay(number int) (Day, bool) {
	if number < 1 || number > 84 {
		return Day{}, false
	}
	weeks := Curriculum()
	return weeks[(number-1)/7].Days[(number-1)%7], true
}

func FindProblem(slug string) (Problem, bool) {
	for _, week := range Curriculum() {
		for _, day := range week.Days {
			for _, problem := range day.Core {
				if problem.Slug == slug {
					return problem, true
				}
			}
			for _, problem := range day.Optional {
				if problem.Slug == slug {
					return problem, true
				}
			}
		}
	}
	return Problem{}, false
}
