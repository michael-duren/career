# Day 4 authoring record

## Scope and source review

The old lesson teaches set membership, multiset multiplicity, and visited-state cycle detection. The actual assignments add Intersection of Two Arrays II, Happy Number, Valid Sudoku, and optional distinct intersection. Happy Number's curriculum note specifically requires Floyd's slow and fast pointer algorithm. I read the mapped hashmap introduction for collision and equality context and the linked-list cycle article for the pointer invariant. Both are research only. All lesson prose, programs, cases, and diagrams here are original.

## Exact case contracts

Intersection input is `n`, then `n` integers, then `m`, then `m` integers, all whitespace separated. Output is matched values in second-array traversal order, comma separated, or `empty`. The stable order makes the examples checkable; the course problem permits any order. Happy Number input is one positive integer and output is lowercase `true` or `false`. Sudoku input is nine rows of nine characters, using `.` for an empty cell; output is lowercase `true` or `false`. The board validator checks existing clues only and does not solve the puzzle.

| Algorithm | Normal stdin | Output | Revealing stdin | Output |
| --- | --- | --- | --- | --- |
| Distinct set intersection | `3\n4 4 7\n4\n4 4 4 9\n` | `4` | `2\n1 2\n2\n3 4\n` | `empty` |
| Multiset scan and consume baseline | `3\n4 4 7\n4\n4 4 4 9\n` | `4,4` | `2\n4 4\n3\n4 4 4\n` | `4,4` |
| Multiset count and decrement | same as baseline | `4,4` | same as baseline edge | `4,4` |
| Happy Number visited set | `19\n` | `true` | `2\n` | `false` |
| Happy Number Floyd pointers | `19\n` | `true` | `2\n` | `false` |
| Sudoku scan baseline | nine rows with single `5` in row 1 | `true` | `5........` and `.5.......` as first two rows | `false` |
| Sudoku row/column/box sets | same Sudoku normal | `true` | same Sudoku edge | `false` |

The remaining Sudoku rows are `.........`. The duplicate 5 clues in the edge input occupy different rows and columns but the same top-left 3x3 box, so only the box constraint rejects them.

## Invariants and costs

For distinct intersection, the set holds values from the first array and a second set holds values already emitted. Membership answers existence; duplicates must not be emitted twice. Expected time O(n+m), auxiliary space O(n+u) for the source and emitted-identity sets, where u is distinct result size. Hash equality resolves collisions; a hash value by itself is not proof of equal keys.

The multiset baseline records which first-array positions have been consumed. Before processing each second-array item, an unused first-array position represents one available copy. Scanning for a match costs O(nm) time and O(n) auxiliary flags. The improved map starts with exact first-array frequencies; after each second-array item, `remaining[x]` is unused supply. A positive count permits one output and then decreases. Expected O(n+m) time and O(d) auxiliary map entries for d distinct first-array values; result storage is separate O(k). The edge's third 4 cannot match after two copies have been consumed.

Happy Number repeatedly replaces n with the sum of the squares of its base-ten digits. The sequence is deterministic and all positive inputs eventually reach 1 or a repeated state. The set baseline stops when it sees 1 or tries to revisit a value. After each insertion, the set contains exactly the previously visited sequence values. It uses O(s) space for s visited values. Floyd starts `slow=n`, `fast=n`, then advances one and two transitions respectively. If either reaches 1 the answer is true; if the pointers meet elsewhere, the sequence cycles and can never reach 1. Both use O(s) transitions in the finite sequence; Floyd has O(1) auxiliary space. Each digit-square transition uses O(log n) digit work for its current argument, and after one step the values are bounded by a small function of input digit count. For fixed machine-size integers both approaches have bounded state, but the set-versus-constant-space distinction still matters in the general technique.

The Sudoku baseline checks each nonempty clue against every earlier clue for a shared row, column, or 3x3 box. After checking the prefix, no two earlier clues conflict. This costs O(c²) comparisons for c filled cells; a fixed 9x9 board bounds it by a constant. The improved approach records digits in separate row, column, and box sets. The box index is `(r/3)*3 + c/3`; before inserting a clue, each set contains exactly prior clues in its region. A repeated digit in any region rejects the board. Time O(81) and auxiliary O(81) for the fixed board, conventionally O(1) under the fixed-size problem contract. Do not treat `.` as a digit.

## Review questions

- Why does one set lose information for Intersection II? It cannot distinguish one available 4 from three available 4s.
- Why is a repeated Happy Number value enough to stop? The transition is deterministic; the same value has the same entire future.
- Why can two 5s in different rows and columns still be invalid? Their 3x3 box may coincide.

## Verification

Pending implementation and independent review.
