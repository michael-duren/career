# Leetgrinder: complexity, code capture, and analysis

Status: approved 2026-09-27.
Builds on: [2026-09-26-leetgrinder-next-design.md](2026-09-26-leetgrinder-next-design.md), [docs/leetgrinder.md](../../leetgrinder.md).

## Goals

1. Record the learner's stated time and space complexity with each attempt, from the extension and the web form.
2. Capture the code LeetCode judged (and its language) with extension-logged attempts.
3. Store each problem's optimal time and space complexity in the curriculum.
4. Analyse captured code with the Claude API: the actual complexity, whether the stated one matches, and whether the solution is optimal.

Non-goals: running user code, keeping every submission, analysing attempts without code, a browser code editor.

## Decisions (from Q&A)

| Topic | Decision |
|---|---|
| Input | Dropdown of common classes plus "Other…" free text, for both time and space. |
| Required | Both are required for `solved` and `struggled` attempts, and optional for `unfinished`. This applies to the web form, the API, and the extension. |
| Code | Store only the code from the submission that produced the logged result, plus its language. Capped at 64 KiB. |
| Analysis | A Claude API background job, plus an optimal-complexity table in the curriculum. |

## Complexity values

Canonical options, stored exactly as shown: `O(1)`, `O(log n)`, `O(√n)`, `O(n)`, `O(n log n)`, `O(n²)`, `O(n³)`, `O(2ⁿ)`, `O(n!)`.

"Other" accepts free text up to 40 characters, such as `O(m·n)`, `O(V + E)` or `O(k log n)`. The value is trimmed, and must start with `O(` and end with `)`. A light normalisation maps common spellings to the canonical form: `n^2` becomes `n²`, `nlogn` becomes `n log n`, and `logn` becomes `log n`.

A shared Go validator enforces these rules; the extension mirrors them in `lib.js`.

## Data model

Migration 013 (Phase 1):

```sql
ALTER TABLE leetgrinder_attempts
    ADD COLUMN time_complexity TEXT NOT NULL DEFAULT '' CHECK (char_length(time_complexity) <= 40),
    ADD COLUMN space_complexity TEXT NOT NULL DEFAULT '' CHECK (char_length(space_complexity) <= 40),
    ADD COLUMN code TEXT NOT NULL DEFAULT '' CHECK (octet_length(code) <= 65536),
    ADD COLUMN code_language TEXT NOT NULL DEFAULT '' CHECK (char_length(code_language) <= 32);
```

Existing attempts keep empty values. The "required" rule applies only to new attempts and to corrections of solved or struggled attempts.

Migration 014 (Phase 3):

```sql
CREATE TABLE leetgrinder_analyses (
    attempt_id UUID PRIMARY KEY REFERENCES leetgrinder_attempts(id) ON DELETE CASCADE,
    code_sha256 BYTEA NOT NULL,          -- re-analyse only when the code changes
    status TEXT NOT NULL CHECK (status IN ('pending', 'done', 'failed')),
    tries INTEGER NOT NULL DEFAULT 0,
    actual_time TEXT NOT NULL DEFAULT '',
    actual_space TEXT NOT NULL DEFAULT '',
    time_matches BOOLEAN,
    space_matches BOOLEAN,
    optimal BOOLEAN,
    explanation TEXT NOT NULL DEFAULT '' CHECK (char_length(explanation) <= 2000),
    model TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

The attempt JSON export gains the four new attempt fields. Analyses are derived data and are not exported.

## Phase 1: complexity and code capture (one PR)

**Go and web form:**
- `Attempt` gains `TimeComplexity`, `SpaceComplexity`, `Code` and `CodeLanguage`.
- `SaveLeetgrinderAttempt` validates them. The idempotent-retry comparison includes all four.
- The attempt form (day, review and history pages) gets two complexity controls, each a select plus a free-text input when "Other" is chosen. They work without JS: the select has an `other` value and a text field sits beside it. A small inline script only toggles visibility.
- Failed saves keep drafts, as they do today.
- Web-logged attempts have no code.

**Attempt history page:**
- Shows the stated complexities.
- Shows the captured code in an escaped `<pre><code>` block with its language, collapsed in a `<details>`.

**API:** `POST /api/leetgrinder/attempts` accepts `timeComplexity`, `spaceComplexity`, `code` and `codeLanguage`. The request body limit rises to fit 64 KiB of code plus overhead, about 96 KiB. Responses:
- 422 when complexity is missing on solved or struggled attempts, or has an invalid format;
- 413 when the code is over the cap.

**Extension:**
- **Capture:** `leetcode-detect.js` (page world) records the JSON body of `POST /problems/{slug}/submit/` (`lang`, `typed_code`) along with the `submission_id` from the response. When the matching `/submissions/detail/{id}/check/` reports a result, it passes `{slug, submissionId, status, lang, code}` to the content script.
- **Trust boundary:** the content script treats page-world messages as untrusted. It checks the origin, message shape, types and size before using them.
- **Confirm panel:**
  - Adds Time and Space selects with "Other…" and a text field, validated with the shared rules.
  - The panel can't submit a solved or struggled attempt without both values.
  - Shows "Code captured (Python3, 1.2 KB)" with a checkbox to leave the code out.
  - Code is sent only through the background worker to the configured app origin.
- **Unfinished nudge:** unchanged. Complexity is optional there, and code from the latest submission is attached if one exists.
- **Tests:** node tests for complexity validation and normalisation, message validation, and payload building.

**Docs:** `docs/leetgrinder.md` and the extension README.

## Phase 2: optimal complexity table (one PR, parallel with Phase 1)

- `Problem` gains `OptimalTime` and `OptimalSpace`, in the same notation, for all 300 problems in `curriculum.go`. The values are the standard best-known complexities for the intended solution.
- Record the convention per problem:
  - Space means auxiliary space, excluding the output unless the output dominates.
  - Where the accepted optimum depends on a stated constraint (fixed alphabet, bounded values), record the complexity under the problem's constraints.
- Add a `docs/leetgrinder-sources.md` section on how the values were chosen, and list any disputed ones.
- A test checks that all 300 problems have both values in valid notation.
- The problems page gets an "Optimal" column showing time and space. The problem history page shows them after your first attempt, so you aren't handed the answer beforehand.

## Phase 3: Claude analysis (one PR, after Phases 1 and 2 merge)

**Setup:**
- `ANTHROPIC_API_KEY` is read from the environment only, never the DB.
- `LEETGRINDER_ANALYSIS_MODEL` sets the model. The default is the current Sonnet model; the implementer confirms the model ID using the `claude-api` skill.
- A settings toggle turns analysis on or off, and defaults to on when a key is present.

**Worker:** `internal/leetgrinder/analysis`, started with the server like the notification worker.
- **Queue:** it picks attempts that have code and at least one stated complexity, and no `done` analysis for the current `code_sha256`.
- **Limits:** one request at a time. Failures are retried at most 3 times with backoff, then marked `failed` with the error. Any key material is redacted from stored errors.
- **Request:** uses the official Go SDK with structured output (a tool or JSON schema). It sends:
  - problem title, slug and link;
  - language and code;
  - stated time and space complexity;
  - the optimal time and space from Phase 2.
- **Response:** `{actualTime, actualSpace, timeMatches, spaceMatches, optimal, explanation}`, with the complexities in the canonical notation where possible.
- **Validation:** the response is checked, and explanations are capped at 2000 characters.
- **Cost:** capped with a max-tokens setting and a daily request cap (`LEETGRINDER_ANALYSIS_DAILY_LIMIT`, default 50).

**Display:**
- The attempt history page shows an analysis card for each attempt:
  - the actual time and space;
  - a ✓ or ✗ next to each stated value;
  - whether the solution is optimal, with the explanation;
  - a "Re-analyse" button, which is same-origin and resets the status to pending.
- The problems table and the dashboard's recent attempts get a small badge when a stated complexity was wrong.

**Tests:**
- An `httptest` fake of the Messages API, with no network in tests.
- Queue selection and code-hash invalidation.
- The retry cap and error redaction.
- Rendering.

**Docs:** configuration, cost notes, and the privacy note: code is sent to Anthropic only when analysis is enabled.

## Delivery

Each phase gets its own worktree, branch and PR. Separate reviewer subagents review each PR, and findings are fixed until a review round is clean. Then the PR is admin squash-merged to `main`. Commits, PR text and squash messages never carry AI attribution or co-author trailers.

1. Phases 1 and 2 run in parallel as separate agents. Phase 1 owns migration 013. Phase 2 needs no migration. Phase 1 touches `Attempt`, forms, the API and the extension. Phase 2 touches `Problem`, `curriculum.go`, the problems page and history display. Whichever merges second rebases.
2. Phase 3 starts after both merge. It owns migration 014.

## Open risks

- The LeetCode submit and check endpoints are undocumented. Capture code stays isolated in `leetcode-detect.js`, alongside the existing detection.
- LLM complexity judgements can be wrong for subtle amortised or input-dependent cases. The UI labels them "Claude's assessment" and keeps the stated values untouched.
- The optimal table is hand-curated: 300 values, reviewed by a subagent, but errors are possible. Disputed entries are listed in the sources doc.

## Phase 3 implementation notes

- `code_sha256` hashes the analysed inputs (language, stated time, stated space, code), not the code alone. Corrections keep the captured code but can change the stated values, which the match verdicts depend on.
- Migration 014 also adds `leetgrinder_analysis_usage` (requests per local date, for the daily cap), `leetgrinder_analysis_pause` (a shared 15-minute pause after errors that would hit every attempt), and `leetgrinder_settings.analysis_enabled` (the toggle, default true).
- "Retried at most 3 times" means four requests in all, as for notifications; the SDK's own retries are disabled so every request is counted.
