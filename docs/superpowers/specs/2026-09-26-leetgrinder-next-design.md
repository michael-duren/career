# Leetgrinder: schedule, spaced repetition, notifications, extension

Status: approved 2026-09-26.
Builds on: [2026-09-26-dsa-leetgrinder-design.md](2026-09-26-dsa-leetgrinder-design.md), [docs/leetgrinder.md](../../leetgrinder.md).

## Goals

1. A modifiable start/end date so the learner sees whether they are on track.
2. Spaced repetition (FSRS) that schedules review problems into days.
3. Adjustable daily time budget; extra time becomes extra review slots.
4. ntfy notifications for missing work and other configurable events.
5. A Chrome + Firefox extension that reports LeetCode submissions (time, outcome) to the app.

Non-goals: score visualizations, in-browser editor, store-published extension, multi-user support, replanning the curriculum around rest days.

## Decisions (from Q&A)

| Topic | Decision |
|---|---|
| Calendar | 84 consecutive days, one session per day. Editing start recomputes end (start + 83); editing end recomputes start. |
| On/off track | `expected = clamp(today - start + 1, 0, 84)`; `delta = completedSessions - expected`. Show "N ahead / on track / N behind". |
| SR algorithm | FSRS via `github.com/open-spaced-repetition/go-fsrs` (latest major), default parameters. |
| Review slots | Base 1/day at 2h. Sessions with any required (non-optional) reading get 0 base slots. Extra slots = `floor((hours - 2) * 60 / 25)`. |
| Daily hours | Setting, 2.0–4.0 in 0.5 steps, default 2.0. Does not move dates. |
| ntfy config | All in the settings UI (server URL, topic, access token). Token encrypted at rest. |
| Notifications | Every type individually configurable. Defaults: 5pm missing-work reminder **on**, behind-schedule alert **on**, others off. |
| Extension | MV3, Chrome + Firefox, loaded unpacked. Personal API token. Confirm popup prefilled on Accepted. |
| Delivery | Worktree per feature, one PR each, foundation first. |

## Data model (migration 011, owned by foundation PR)

All new tables land in one migration so later parallel PRs never race on migration numbers.

```sql
-- Singleton settings row (id = 1).
CREATE TABLE leetgrinder_settings (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    start_date DATE,                         -- NULL = schedule not set; UI prompts
    timezone TEXT NOT NULL DEFAULT 'America/Chicago', -- IANA name, validated in Go
    daily_hours NUMERIC(2,1) NOT NULL DEFAULT 2.0 CHECK (daily_hours BETWEEN 2.0 AND 4.0),
    ntfy_url TEXT NOT NULL DEFAULT 'https://ntfy.sh',
    ntfy_topic TEXT NOT NULL DEFAULT '',
    ntfy_token_ciphertext BYTEA,             -- AES-GCM, nonce-prefixed
    notifications JSONB NOT NULL DEFAULT '{}',  -- per-type {enabled, time, threshold, priority}
    revision UUID NOT NULL
);

-- Extension/API tokens. Only sha256(token) is stored.
CREATE TABLE leetgrinder_api_tokens (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    token_hash BYTEA NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
);

-- Stable daily review picks, so notifications and the day page agree.
CREATE TABLE leetgrinder_review_plan (
    plan_date DATE NOT NULL,
    problem_slug TEXT NOT NULL,
    slot SMALLINT NOT NULL,
    PRIMARY KEY (plan_date, slot),
    UNIQUE (plan_date, problem_slug)
);

-- Deduplicates sends: at most one of each kind per local date.
CREATE TABLE leetgrinder_notification_log (
    kind TEXT NOT NULL,
    local_date DATE NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    status TEXT NOT NULL,                    -- sent | failed
    detail TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (kind, local_date)
);

ALTER TABLE leetgrinder_attempts
    ADD COLUMN source TEXT NOT NULL DEFAULT 'web' CHECK (source IN ('web', 'extension')),
    ADD COLUMN is_review BOOLEAN NOT NULL DEFAULT false;
```

Encryption key: new env `LEETGRINDER_SECRET_KEY` (32 bytes, base64). In dev, fall back to a key derived from `JWT_SECRET` with HKDF. In production, config load fails if it is missing while an ntfy token is stored. The UI shows "token set / not set" and never echoes the token.

Settings writes use the existing revision-check pattern.

## Schedule (package `internal/leetgrinder`)

Pure functions, no DB:

- `Schedule{Start time.Time, Loc *time.Location}` with `End()`, `ExpectedSessions(now)`, `SessionForDate(date)`, `Delta(completed, now)`.
- Before start: expected 0, status "starts in N days". After end: expected 84.
- Overview gains a schedule card: start, end, today's scheduled session number, status. It keeps the existing "continue with earliest unfinished session" link.

## Spaced repetition

**Card state is derived, not stored.** On each request, replay every attempt for a slug through FSRS in `created_at` order. This means corrections and deletes flow through automatically. 300 problems × a few attempts is trivial cost.

Rating map (per attempt):

| Attempt | FSRS rating |
|---|---|
| unfinished | Again |
| struggled, or solved with `assisted = true` | Hard |
| solved, unassisted, minutes > 25 | Good |
| solved, unassisted, minutes ≤ 25 | Easy |

Multiple attempts on the same local day: only the last one counts for scheduling. This avoids same-day re-solves inflating stability.

**Daily pick** (`PlanReviews(date)`), run lazily on first access for a date (day page, overview, or notification worker) and persisted to `leetgrinder_review_plan`:

1. Slots = base (0 or 1 per the reading rule for *today's scheduled session*) + extra from daily hours.
2. Candidates = cards with `due <= end of date`, excluding problems assigned as core/optional in today's session.
3. Sort by lowest retrievability. Tiebreak: prefer a problem from an earlier curriculum week (interleaving), then most overdue.
4. Take the top N. The plan is frozen for that date. Changing daily hours mid-day adds slots but never removes picks already made.

A review is done when an attempt for that slug is logged on that local date. Attempts logged from a review card set `is_review = true`.

**UI:**
- Day page and overview show a "Today's review" section: problem link, lesson-week label, plain-text reason, and the attempt form.
- Reason example: "Struggled 9 days ago · recall estimate 62%".
- A review queue page `/leetgrinder/reviews` lists every card with due date and retrievability, plus the backlog count.

## Notifications (ntfy)

**Worker:** `internal/leetgrinder/notify` runs in the server process like `running.Worker`.
- Ticks every minute. Loads settings, computes local now in the configured timezone.
- For each enabled kind whose trigger time has passed today and has no log row: evaluate the condition, send, then insert the log row. Insert uses `ON CONFLICT DO NOTHING` so restarts don't double-send.
- ntfy client: `POST {url}/{topic}` with `Title`, `Priority`, `Tags`, `Click` (deep link to the day page), and `Authorization: Bearer <token>` if a token is set. 10s timeout.
- On failure, log `failed` with detail, and retry on later ticks up to 3 times that day.

| Kind | Default | Trigger | Condition |
|---|---|---|---|
| `missing_work` | on, 17:00 | time | Today's scheduled session not completed, or any planned review for today has no attempt. Body lists what's missing. |
| `behind_schedule` | on, threshold 3 | evaluated at 17:00 | `delta <= -threshold`. Once per day. |
| `morning_plan` | off, 08:00 | time | Always: session title, required reading, review problems. |
| `late_escalation` | off, 21:00, priority high | time | Same condition as `missing_work`. |
| `review_backlog` | off, threshold 10 | evaluated at 17:00 | Due-and-unplanned review count ≥ threshold. |

Nothing sends before `start_date`, after end + 1 day, or while the schedule is unset.

**Settings page** `/leetgrinder/settings` covers:
- dates
- timezone
- daily hours
- ntfy URL, topic and token
- per-kind toggles, times and thresholds
- a "Send test notification" button
- the last 14 notification log rows

## Extension API

- `POST /api/leetgrinder/attempts` with `Authorization: Bearer lg_<random>`.
  - Body: `{id, problemSlug, outcome, minutes, assisted, notes, isReview?}`.
  - Reuses the existing idempotent-ID attempt creation. Sets `source = 'extension'`.
  - 422 for a slug not in the curriculum, 401 for a bad or revoked token.
- `GET /api/leetgrinder/problem/{slug}`: whether the slug is in the curriculum, its session/week, whether it's today's review, and the latest attempt. The popup uses this for context.
- Token management lives on the settings page: create (shown once), list, revoke. Constant-time hash lookup. `last_used_at` updated at most once per minute.
- `/api/` already bypasses the cookie redirect, so these handlers authenticate themselves. No CORS: the extension calls from its background service worker using host permissions for the configured app origin.

## Browser extension (`extension/leetgrinder/`)

- MV3, plain JS, no build step. The same `manifest.json` works in Chrome and Firefox, with a `browser_specific_settings.gecko` id. Small `browser`/`chrome` namespace shim.
- Options page: app origin + API token (stored in `storage.local`), and a "Test connection" button that calls the GET endpoint.
- Content script on `https://leetcode.com/problems/*`:
  - Starts a timer when the problem page first opens. Timer state persists per slug in `storage.session`, so reloads don't reset it.
  - A page-world injected script watches LeetCode's submission check responses (`/submissions/detail/*/check/`) for `status_msg == "Accepted"`. If that proves brittle, it falls back to a DOM observer on the result panel.
  - Detects Solutions/Editorial tab visits to prefill `assisted`.
- On Accepted, an in-page confirm panel opens, prefilled with:
  - outcome: solved if ≤ 25 min, else struggled
  - minutes from the timer
  - assisted
  - notes
  - Submit sends through the background worker. Dismiss sends nothing.
- At 25 minutes without Accepted, a nudge offers "Log as unfinished".
- Slugs outside the curriculum are ignored (no panel).
- README covers loading unpacked in both browsers.

## Testing

- Schedule: boundary dates, timezone edges, editing start or end.
- FSRS replay: rating map, same-day collapse, and that corrections change due dates.
- Review planning: reading-day rule, extra slots, frozen plans, excluding today's assignments, interleaving tiebreak.
- Notifications: fake clock + `httptest` ntfy server. Covers each kind, dedupe across restarts, retry cap, no sends outside the schedule, test button.
- Crypto: round-trip, the token never rendered, and a missing prod key failing.
- API: bearer auth, revoked token, idempotent retry, unknown slug, `source` recorded.
- Postgres tests for migration 011 (`make test-postgres`).
- Extension: manual checklist in its README. Optionally unit-test pure helpers (outcome inference) with `node --test`.

## Delivery plan (subagents)

Each phase runs in its own worktree/branch and opens its own PR (no AI attribution or co-author trailers in commits/PRs). Each implementing agent has a separate reviewer agent review its PR, fixes findings, and repeats until a review finds no issues, then merges to `main`. Phase 2 branches from `main` after the foundation merges; whichever phase-2 PR merges second rebases onto `main` first.

1. **Foundation** (`feat/leetgrinder-schedule-srs`, base `main`), one agent:
   - migration 011
   - settings store + crypto
   - schedule
   - FSRS replay + review planning
   - settings page (dates, timezone, hours only)
   - overview schedule card, today's reviews, `/leetgrinder/reviews`
   - `is_review` on web attempts
2. **Parallel, branched from `main` after foundation merges**:
   - **Notifications** (`feat/leetgrinder-ntfy`): ntfy client, worker, settings UI section, test button, log view.
   - **Extension** (`feat/leetgrinder-extension`): token store/UI, API endpoints, `extension/leetgrinder/`.
3. **Docs**: update `docs/leetgrinder.md` in each PR for its own feature.

## Open risks

- LeetCode's submission endpoint and DOM are undocumented and can change. Keep the detection code isolated in one file.
- FSRS default parameters are tuned for flashcards. Intervals may feel long for hard problems. A desired-retention setting (default 0.9) is a cheap later knob.
- Only one learner and one settings row exist. Multi-user support would need a rework, which is acceptable per the non-goals.
