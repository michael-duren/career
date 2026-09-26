# DSA leetgrinder implementation plan

> **For agentic workers:** Use superpowers:executing-plans; independent content and storage work may use superpowers:dispatching-parallel-agents.

**Goal:** Deliver the approved 12-week templ curriculum with manual tracking.
**Architecture:** A compiled Go curriculum and templ lessons feed server-rendered pages. Existing authenticated Go routes save attempts and session completion to PostgreSQL.
**Tech stack:** Go, templ, chi, PostgreSQL, ordinary HTML forms.
**Spec:** docs/superpowers/specs/2026-09-26-dsa-leetgrinder-design.md

## Global constraints

84 sessions; 252 core + 48 optional unique free LeetCode problems. Original lessons in templ. Numbered self-paced sessions. No further approval questions. Preserve untracked source data and exclude it from deliverables.

## Review focus

- Retried form submissions must not duplicate history.
- Corrected results must update distinct-solve totals without overwriting a concurrent edit.
- Failed saves must preserve form input and never advance progress.
- Finishing a day must not imply that any problem was solved.
- Reading and problem links must match the prerequisites and advertised titles.

## Task 1: curriculum

- [x] Create internal/leetgrinder curriculum metadata and templ lessons using the shared types in model.go.
- [x] Research primary reading sources and validate LeetCode slugs, IDs, difficulty, and free access.
- [x] Test 12 weeks, 84 ordered days, exactly 3 core/day, 4 optional/week, 300 distinct problems, every day has a rendered lesson.
- [x] Record sourcing and link verification in docs/leetgrinder-sources.md.

## Task 2: storage

- [x] Retain 008_note_created_at.sql; add 009_leetgrinder.sql and 010_leetgrinder_names.sql; increment schema version to 10.
- [x] Add LeetgrinderState, SaveLeetgrinderAttempt, SetLeetgrinderDay store methods and integration tests.
- [x] Prove repeated creates are idempotent, stale corrections conflict, attempts persist, and completed days are independent.

## Task 3: pages and routes

- [x] Add pinned templ tooling, generation target, Docker generation.
- [x] Add overview, day, and problem-history templ pages with normal POST forms.
- [x] Register authenticated routes, validate inputs and origins, preserve inputs on failures, redirect after successful saves.
- [x] Add navigation entry and leetgrinder tracking export.
- [x] Test auth, unknown identifiers, malformed input, preserved drafts, correction and continuation behavior.

## Task 4: verification

- [x] Generate templates, run Go tests with PostgreSQL and frontend build.
- [x] Exercise login, overview, day, save, correction, completion and export against running app.
- [x] Review full diff, fix material findings, and document tested behavior and remaining limitations.
