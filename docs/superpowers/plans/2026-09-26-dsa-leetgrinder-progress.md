# Leetgrinder implementation progress

Plan: 2026-09-26-dsa-leetgrinder.md

- Approved design captured; user authorized continued implementation without further approval gates.
- Working branch: feat/leetgrinder. Source data remains untracked and is not embedded or served.
- Baseline Go tests passed; PostgreSQL integration tests require the local disposable-schema test helpers.
- Storage task complete: migrations 009/010, retry-safe attempts, revision-checked corrections, independent session completion. Full database suite passed against PostgreSQL.
- UI and routes implemented; awaiting full curriculum generation to run integrated tests. Frontend build passed.
- Generator pinned to templ v0.3.1001 and wired into Make, CI, release verification, Docker, and Air.
- Independent review identified URL-encoded Unicode notes exceeding a 16 KiB body cap. Regression test added; fix and integrated verification pending curriculum compilation.
- Curriculum worker verified all 300 assigned IDs as free in LeetCode's live public metadata; original daily lessons and reading source checks in progress.

- All tasks complete. Curriculum: 84 original lessons, 252 core and 48 optional unique free problems. Live catalog verification and all 21 distinct reading documents passed.
- Corrected the reviewed URL-encoded form limit to 32 KiB after reproducing rejection of a valid 2,000-rune note. Regression passes.
- Reviewed curriculum examples and source sections; replaced generic window, prefix, monotonic-stack, and trie references with topic-specific readings. Optional challenges follow prerequisites. Mixed-practice guidance is collapsed until the learner chooses to review it.
- Verification: Go tests with race detector and disposable PostgreSQL schemas passed; go vet passed; frontend build and all 42 Node tests passed. Real Chromium workflow passed login, overview, lesson rendering, create, correction, day advancement, export, and mobile overflow checks.
- Build verified from a temporary copy with generated templates and private source data absent.
- Integration decision: leave changes on feat/leetgrinder for review. No push or deployment was requested.

- Local startup fix: shared PostgreSQL already held migration 008_note_created_at from another checkout. Its checksum matched that file exactly. Retained it and assigned Leetgrinder 009/010; added a version-8 upgrade regression and version-specific checksum diagnostics. Applied successfully to the local database; all four existing notes retained the same content fingerprint.
