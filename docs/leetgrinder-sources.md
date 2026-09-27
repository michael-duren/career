# Leetgrinder sources and verification

Problem catalog verified on 2026-09-26. The curriculum has 84 sessions with 252 core problems and 48 optional problems. All 300 assignments are distinct, and all are free according to LeetCode's public catalog at verification time. Access rules and external pages can change.

## Authorship and local source material

The local `data/algomonster/CURRICULUM.md` was used to identify subject coverage and prerequisite relationships. The new lessons, worked examples, review prompts, and daily assignments were authored independently. No source lesson text, source code, problem statements, or solutions are embedded in the application. The local source directory is not required at build time and is excluded by the Docker context allowlist.

All instructional prose is in `internal/leetgrinder/lessons.templ`. Day metadata and public problem titles are in `internal/leetgrinder/curriculum.go`. Reading directions and source attribution are in `internal/leetgrinder/readings.go`. The runtime uses these compiled files and never fetches or renders the private source material.

## Reading references

Each lesson has its own introductory document, followed by optional deeper reading. The first reading assigns a focused excerpt of at most ten minutes. Paper assignments identify a section, algorithm, or example to inspect. Their time estimates cover that excerpt, not a complete paper. All assigned excerpts together fit within the existing 30-minute reading and review budget.

The source selection favors free textbooks, university course notes, author-hosted articles, and open research papers or technical reports. A paper is useful when it explains an algorithm's origin, proof, limitations, or application. It is not a prerequisite for the day's practice. A shared foundational source can support more than one lesson, but the introductory documents must be distinct and the reading directions must match the day's concepts.

Every lesson ends with a References section. Each linked citation states which concepts or analysis it supports. Worked examples and practice prompts remain original instructional material. The Reading path panel gives the reading order and excerpt instructions. For mixed practice, both the lesson and its readings stay inside the review disclosure so they do not reveal the intended technique before an attempt.

The existing required/optional status of each day's introduction is preserved because it also determines the base spaced-review allowance. Optional introductions are still listed first. Deeper research readings are always optional.

`internal/leetgrinder/readings.go` is the source of truth for the bibliography. Per-lesson Sol review records are in:

- [Lessons 1–28](leetgrinder-review-01-28.md)
- [Lessons 29–56](leetgrinder-review-29-56.md)
- [Lessons 57–84](leetgrinder-review-57-84.md)

Run `go test ./internal/leetgrinder` after `make generate` to check all 84 introductory documents for uniqueness, reading budgets, optional papers, and rendered bottom citations. These tests check structure and rendering. Source relevance requires the lesson-by-lesson review; an HTTP success code alone does not establish it.

## Problem catalog verification

The [LeetCode public problem catalog](https://leetcode.com/api/problems/all/) supplied canonical public problem IDs, slugs, titles, difficulty, and the `paid_only` field. Each assignment was checked against those fields. Practice links use `https://leetcode.com/problems/<canonical-slug>/`; the app does not reproduce the corresponding statements or solutions.

[leetgrinder-problems.json](leetgrinder-problems.json) records the verified assignment metadata. This is an audit snapshot, not a second runtime curriculum. Re-run the live check with:

```sh
make generate
go run ./cmd/verify-leetgrinder
```

The verifier reads the actual compiled curriculum, checks all 300 problems against the live catalog, and requests each distinct reading document. It exits nonzero for metadata mismatches, paid problems, duplicates, count errors, inaccessible documents, and network failures. It does not submit solutions or access a LeetCode account.

`go test ./internal/leetgrinder` separately checks day ordering, assignment totals, lookup behavior, reading budgets, unique lesson rendering, draft escaping, and mixed-practice guidance. PostgreSQL and HTTP workflow tests cover the saved learning history.
