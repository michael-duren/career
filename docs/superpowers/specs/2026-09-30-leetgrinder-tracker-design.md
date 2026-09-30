# Leetgrinder: from curriculum to tracker

Status: draft 2026-09-30.
Builds on: [2026-09-26-leetgrinder-next-design.md](2026-09-26-leetgrinder-next-design.md), [2026-09-27-leetgrinder-complexity-design.md](2026-09-27-leetgrinder-complexity-design.md), [docs/leetgrinder.md](../../leetgrinder.md).

## Goals

1. Retire the 84-day curriculum (lessons, readings, examples, day pages, schedule) into `retired/leetgrinder/`, out of the build but kept for later.
2. Track any LeetCode problem. The user solves whatever they choose; the app works out what the problem is and whether it is new or a review.
3. A daily goal of 2 new problems + 1 review (about two hours), with extra problems counted as a bonus and a streak for goal-met days.
4. Keep and extend spaced repetition: one frozen FSRS review pick per day, the rest of the due list as optional, and earlier reviews for problems whose complexity analysis found a mistake.
5. Configurable ntfy reminders built around the daily goal, reviews, and the streak.
6. Extension upgrades: accept any problem, toolbar badge, popup dashboard, on-page status banner.
7. Topic and difficulty stats from LeetCode tags.

Non-goals: any lesson or reading content, problem recommendations, email or calendar reminders, browser notifications from the extension, multi-user support, store-published extension, an import UI.

## Decisions (from Q&A)

| Topic | Decision |
|---|---|
| Catalog | Any LeetCode problem. Metadata cached in a new `leetgrinder_problems` table. The curated optimal-complexity table for the 300 curriculum problems becomes seed data. |
| Picking new problems | Free choice. No queue, no suggestions. An attempt on a never-attempted problem is new; the server decides. |
| Retire | `retired/leetgrinder/` directory, excluded from build, tests, and templ generation. |
| Review pick | FSRS picks 1 review per day (lowest recall), frozen for the date. Other due cards listed as optional. Any due problem satisfies the review goal. |
| Extra flagging | Assisted or struggled attempts keep flowing through FSRS ratings. New: a wrong stated complexity or a non-optimal solution (from Claude's analysis) makes the problem due the next day. No manual flag. |
| Daily goal | 2 new + 1 review by default; both targets configurable. Extras shown as bonus. Streak counts goal-met days. |
| Daily hours | Removed; replaced by the two goal targets. |
| Reminders | ntfy only: morning plan, goal incomplete, streak at risk, review backlog. Each individually enabled, timed, prioritised, and (where it applies) thresholded in settings. |
| Extension | Accept any problem; badge with remaining goal; popup dashboard; on-page banner. |
| Stats | Weakness view per topic plus difficulty mix. |
| Old data | Keep all attempts, code, and analyses (they seed review cards). Drop curriculum state (`completed_days`, `start_date`, `daily_hours`). Export JSON before migrating. |
| Delivery | Three PRs, in order (see Delivery). |

## 1. Retirement

Move to `retired/leetgrinder/`, preserving relative layout under subdirectories `internal/`, `docs/`, `scripts/`, `cmd/`, `tests/`:

- `internal/leetgrinder/`: `curriculum.go`, `curriculum_test.go`, `readings.go`, `lesson_content.go` (+ test, templ), `lesson_assets.go` (+ test), `lesson_player.js`, `lesson_player.templ`, `lessons.templ`, `optimal.templ`, `optimal_test.go`, `example_content_test.go`, `lessons/`, `examples/`, `sources_*.json`, and the day and about page parts of `views.templ`.
- `docs/`: `leetgrinder-authoring/`, `leetgrinder-review-*.md`, `leetgrinder-sources.md`, `leetgrinder-content-state.md`, `leetgrinder-problems.json`, and the curriculum-era specs and plans under `docs/superpowers/` (kept where they are, as history, not moved).
- `scripts/leetgrinder/`, `cmd/verify-leetgrinder/`, `cmd/leetgrinder-preview/`.
- `tests/leetgrinder-lessons.browser.test.mjs`, `tests/leetgrinder-player.test.mjs`, and the two `test:leetgrinder:*` npm scripts.

Keeping it out of the build:

- `retired/go.mod` declares a separate stub module, so `go build ./...`, `go vet ./...` and `go test ./...` in the main module skip it.
- Retired `.templ` files are renamed `*.templ.retired` so `go tool templ generate` (which walks from the repo root) ignores them.
- `retired/leetgrinder/README.md` says what the curriculum was, the last commit where it ran (`git rev-parse` at move time), and what would be needed to revive it (restore the package, migration for `completed_days`, schedule settings).

Before moving `curriculum.go`, extract every problem's `slug, number, title, difficulty, optimalTime, optimalSpace, optimalNote` and its curriculum topic into `internal/leetgrinder/catalog_seed.json`, embedded and used by the migration seeder (section 2). The number is taken from `docs/leetgrinder-problems.json`.

Routes removed: `/leetgrinder/day/{n}`, `/leetgrinder/day/{n}/complete`, `/leetgrinder/about`, `/leetgrinder/settings/schedule`, and lesson asset routes. `/leetgrinder/day/*` and `/leetgrinder/about` return 410 Gone with a link to the dashboard.

## 2. Problem catalog

```sql
CREATE TABLE leetgrinder_problems (
    slug TEXT PRIMARY KEY CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    number INTEGER,                       -- LeetCode frontend id; NULL until known
    title TEXT NOT NULL DEFAULT '',
    difficulty TEXT NOT NULL DEFAULT '' CHECK (difficulty IN ('', 'Easy', 'Medium', 'Hard')),
    topics TEXT[] NOT NULL DEFAULT '{}',  -- LeetCode topic tag slugs, e.g. {array,hash-table}
    optimal_time TEXT NOT NULL DEFAULT '',
    optimal_space TEXT NOT NULL DEFAULT '',
    optimal_note TEXT NOT NULL DEFAULT '',
    optimal_source TEXT NOT NULL DEFAULT '' CHECK (optimal_source IN ('', 'curated', 'model')),
    metadata_source TEXT NOT NULL DEFAULT '' CHECK (metadata_source IN ('', 'seed', 'extension', 'leetcode')),
    fetched_at TIMESTAMPTZ,
    fetch_attempts SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

`Problem` in Go gains `Number`, `Topics`, and `OptimalSource`; `FindProblem` becomes a DB lookup (the state loader reads the whole table, which stays small for one user).

How a problem gets in:

1. **Seed.** Migration 015 inserts every slug from `catalog_seed.json` with `optimal_source = 'curated'` and `metadata_source = 'seed'`, plus a bare row for any attempted slug not in the seed.
2. **Extension.** The extension runs on leetcode.com, so it reads metadata same-origin from `https://leetcode.com/graphql` (`question(titleSlug) { questionFrontendId title difficulty topicTags { slug name } }`) and sends it with the attempt or a new `PUT /api/leetgrinder/problem/{slug}` call. The server validates it (lengths, difficulty enum, tag slug format, at most 20 tags) and upserts; it never overwrites curated optimal values.
3. **Server fetch.** For slugs with no title (web-logged, or extension metadata failed), a background job calls the same GraphQL query once per minute at most, one slug at a time, with a 10 s timeout and backoff after 1, 10, and 60 minutes, then gives up (`fetch_attempts = 4`). A failed fetch never blocks saving an attempt; pages fall back to the slug as the title.
4. **Web form.** A "Log an attempt" form on the dashboard takes a LeetCode URL or slug, normalises it to a slug, and opens the problem page. Saving the first attempt creates the row (a GET never does, so a cross-site request cannot queue fetches).

Unknown slugs are allowed everywhere; the 422 "not in the curriculum" response goes away. A slug that LeetCode reports as nonexistent is kept, marked in the UI as "Not found on LeetCode".

**Optimal complexity for non-curated problems.** The analysis prompt gains an instruction: when no reference optimal is supplied, also return `optimalTime`, `optimalSpace`, and a one-line `optimalNote`. The server normalises them with `NormalizeComplexity` and stores them on the problem with `optimal_source = 'model'`, only if the problem has none yet. The UI labels model values "Claude's estimate". Curated values are never replaced.

## 3. New versus review, daily goal, streak

All days are local dates in the settings time zone.

- An attempt is **new** if its problem has no attempt on an earlier local day. Several attempts on a new problem the same day count once.
- An attempt is a **review** if the problem has an attempt on an earlier local day.
- A review counts toward the **review goal** when the problem was due by the end of that day (FSRS due date, or the flag in section 4) or was that day's review pick. A re-solve of a problem that was not due counts as **practice**: shown, but not toward the goal.
- Unfinished attempts count. The goal is about time spent, not only solves.

`is_review` stays on attempts but is now set by the server at save time from the rules above (the client value is ignored, and the field stays accepted for old extension versions). History and export keep it.

Goal settings:

```sql
ALTER TABLE leetgrinder_settings
    ADD COLUMN goal_new SMALLINT NOT NULL DEFAULT 2 CHECK (goal_new BETWEEN 0 AND 10),
    ADD COLUMN goal_review SMALLINT NOT NULL DEFAULT 1 CHECK (goal_review BETWEEN 0 AND 10),
    DROP COLUMN start_date,
    DROP COLUMN daily_hours;
-- 0 + 0 is rejected in Go.

CREATE TABLE leetgrinder_daily_goal (
    local_date DATE PRIMARY KEY,
    goal_new SMALLINT NOT NULL,
    goal_review SMALLINT NOT NULL
);
```

The day's targets are frozen into `leetgrinder_daily_goal` on first access that day (dashboard, API, or reminder worker), the same way review plans are. Changing settings affects today only if today has not been frozen yet, otherwise from tomorrow; the settings page says which. Past days without a row (before this feature) are evaluated with the defaults, 2 + 1.

A day's **goal is met** when `new >= goal_new` and `goal-counting reviews >= goal_review`. Work beyond that is **bonus**: extra new problems, extra due reviews, and practice.

**Streak.** Consecutive goal-met days ending today, or ending yesterday if today is not met yet (so the streak is still alive during the day). The dashboard shows the current and longest streak. A second, looser "active days" count (any attempt) is shown beside it.

## 4. Reviews

FSRS stays as is: cards replayed from attempts on every request, go-fsrs v4, default weights, 90% retention, same outcome-to-rating mapping, last attempt per local day. Cards now exist for every attempted problem, not just curriculum ones.

**Daily pick.** `leetgrinder_review_plan` is kept. Each date gets `goal_review` picks the first time it is needed, frozen for the day. Candidates are cards due by the end of the day that were not first attempted today. Order: flagged first (below), then lowest estimated recall, then most overdue. The curriculum-week tiebreak is removed. Raising `goal_review` later adds picks; lowering never removes them.

**Due list.** All other cards due by the end of today, sorted the same way, shown as optional on the dashboard and review queue.

**Complexity flag.** Derived, not stored: when a problem's latest attempt has a completed analysis with a stated value judged wrong (`timeMatch = false` or `spaceMatch = false`) or `optimal = false`, its effective due date is `min(fsrs due, local date of the analysis completion + 1 day)`. The next attempt on the problem clears it, since the flag only looks at the latest attempt. The reason line reads, for example, "Time complexity judged wrong 2 days ago" or "Not optimal: O(n²) vs O(n)". Re-analysing that clears the verdict clears the flag.

**Review backlog** = due cards (including flagged) not in today's plan and not attempted today.

## 5. Reminders

The notification worker, log, dedupe, retry, token encryption, and ntfy settings stay as they are. The schedule gates (before start, after end, no schedule) are removed; reminders run whenever a topic is set. Kinds:

| Key | Label | Default | Sends when | Threshold |
|---|---|---|---|---|
| `morning_plan` | Morning plan | on, 08:00, default | Always: today's targets, the review pick(s), due count, current streak. | none |
| `goal_incomplete` | Goal incomplete | on, 18:00, default | Goal not met. Lists what is left: "1 new, review: Two Sum". | none |
| `streak_at_risk` | Streak at risk | on, 21:00, high | Goal not met and the streak is at least the threshold (0 means always when the goal is not met). | Streak days, default 1 |
| `review_backlog` | Review backlog | off, 18:00, default | Backlog is at least the threshold. | Due reviews, default 10 |

Each has enable, time, priority, and threshold (where listed) in settings, as today. Migration 015 rewrites the `notifications` JSONB: `missing_work` becomes `goal_incomplete` (keeping its values), `late_escalation` becomes `streak_at_risk`, `behind_schedule` is dropped, `morning_plan` and `review_backlog` keep their values. Old log rows keep their old kind names; the settings log shows them with their old labels.

Click links go to the dashboard, or to the review queue for `review_backlog`.

## 6. Extension

**Any problem.** The content script activates on every `leetcode.com/problems/{slug}/` page. The timer, Accepted detection, 25-minute unfinished offer, complexity requirement, and code capture are unchanged.

**API changes** (all Bearer-token, same auth as today):

- `GET /api/leetgrinder/problem/{slug}` returns `known`, metadata when known, and `status`: one of `new` (never attempted), `due` (with `recall`, `dueDate`, `flagReason?`, `todaysPick`), `notDue` (with `nextDue`, `lastAttemptedAt`), plus `latestAttempt` without code and `attemptedToday`. `inCurriculum`, `session`, `week`, `todaysReview`, and `reviewDone` are removed.
- `PUT /api/leetgrinder/problem/{slug}` with `{number, title, difficulty, topics:[{slug,name}]}` upserts metadata (section 2). 204 on success.
- `POST /api/leetgrinder/attempts` accepts an optional `problem` object with the same metadata and upserts it in the same transaction. `isReview` is ignored. Response gains `kind`: `new`, `review`, or `practice`.
- `GET /api/leetgrinder/today` returns `{date, goal:{new, review}, done:{new, review, bonus}, met, remaining, streak, picks:[{slug,title,difficulty,recall,reason,done}], due:[...same, max 20], dueCount}`. Plans today's goal and reviews on first access.

**Badge.** The background worker refreshes `/today` on startup, after every logged attempt, and every 15 minutes via `chrome.alarms` (adds the `alarms` permission). Badge text is `remaining` (new + review left); a green check when the goal is met; empty with a grey `?` title when signed out or unreachable.

**Popup** (`popup.html`, `popup.js`, `popup.css`): today's progress (new x/2, review x/1, bonus n), streak, the review pick(s) with links to LeetCode, the due list (top 10), and a link to the dashboard. Same styling as the options page. No writes.

**On-page banner.** A small pill injected near the problem title (fallback: fixed top-right), shadow-DOM isolated like the confirm panel: "New", "Review due · recall 62%", "Today's review", "Flagged: time complexity judged wrong", or "Reviewed 3 d ago · next due Oct 14", plus a one-line last-attempt summary ("Struggled · 32 min · O(n log n)/O(n)"). Clicking opens the problem's history page in the app. Hidden when not configured.

**Confirm panel** shows the attempt kind it will be ("New", "Review", "Practice") from the status call.

Extension version bumps to 2.0.0. The README's manual test checklist is updated.

## 7. Web pages

Side nav: Dashboard, Problems, Reviews, Stats, Settings.

- **Dashboard `/leetgrinder`:** today's goal card (new x/2, review x/1, bonus, met state), streak (current, longest, active days), the review pick(s) with reason and "Log attempt" link, the optional due list (top 10, link to reviews), the "Log an attempt" URL/slug form, and recent attempts (with the "Check complexity" badge, as now).
- **Problems `/leetgrinder/problems`:** every problem with at least one attempt. Columns: number, title, difficulty, topics, optimal time/space (with "Claude's estimate" marker), status, attempts, last attempt, next due, LeetCode link. Filters: search (title, slug, topic; bare number matches exactly), difficulty, topic, status (`due`, `flagged`, `solved`, `struggled`, `unfinished`). Sort: last attempt (default), next due, number, title, difficulty, recall. Week, session, and core/optional filters are removed.
- **Problem `/leetgrinder/problem/{slug}`:** unchanged history, analysis cards, and attempt/correction forms; header shows metadata, topics, status, recall, next due, and flag reason. Works for any valid slug; an unknown slug shows the form and "Fetching details from LeetCode…" until metadata arrives.
- **Reviews `/leetgrinder/reviews`:** every card with due date, recall, flag reason, and today's pick marked; backlog count.
- **Stats `/leetgrinder/stats`:** per topic: problems attempted, solved, struggle rate (struggled + unfinished + assisted over attempts), average current recall, and count due; sortable, weakest (lowest average recall) first by default. Difficulty mix (attempted and solved per Easy/Medium/Hard). A 12-week calendar grid of goal-met days (plain HTML table, no charting library). Topics with fewer than 3 problems are grouped under "Other" unless filtered. Problems without metadata count as "Untagged".
- **Settings:** time zone, goal targets, reminders, ntfy, analysis, extension tokens. The schedule section is removed.

Export JSON drops `completedDays`, adds `problems` (catalog rows for attempted slugs) and `dailyGoals`.

## 8. Migration 015

One migration, run after taking an export (the PR description and `docs/leetgrinder.md` say to use "Export attempt history" first):

1. Create `leetgrinder_problems` and `leetgrinder_daily_goal`.
2. Seed problems from the embedded seed (done in Go right after migrations, since SQL migrations cannot read embedded JSON; idempotent `INSERT ... ON CONFLICT DO NOTHING`), plus bare rows for attempted slugs.
3. Add `goal_new`, `goal_review`; drop `start_date`, `daily_hours`.
4. Rewrite `notifications` JSONB keys (section 5).
5. `DROP TABLE leetgrinder_completed_days`.
6. Recompute `is_review` for existing attempts with the new rules (a set-based `UPDATE` using the settings time zone).

Old review-plan rows stay; they are history and do not affect future dates.

## 9. Testing

- Go unit: new/review/practice classification across day boundaries and time zones; goal met and bonus; streak (alive during today, broken by a missed day, pre-feature days); frozen daily goal on settings change; flag derivation and clearing; plan ordering with flags; each reminder condition; metadata validation and upsert precedence (curated never overwritten); slug/URL normalisation.
- Go server: API shapes for `/problem`, `/today`, `PUT /problem`, attempts with `problem` metadata and `kind`; 410 for retired routes; unknown-slug problem page; stats page.
- Postgres: migration 015 on a database with 011–014 data (attempts preserved, `is_review` recomputed, notification keys rewritten, completed days dropped).
- GraphQL fetcher against an `httptest` server: success, 404-like null question, timeout, backoff limit.
- Extension (`node --test`): banner status text, badge text, popup rendering from fixture `/today` responses; complexity vectors unchanged.
- `go build ./...` and `go test ./...` pass with `retired/` present, and `templ generate` produces nothing from it.

## Delivery

1. **Retire + catalog.** Section 1, section 2, the catalog parts of migration 015, any-slug problem pages, API accepts any slug, extension accepts any problem (no badge/popup/banner yet). Dashboard temporarily shows recent attempts and reviews without goals.
2. **Goal, reviews, reminders.** Sections 3, 4, 5, the rest of migration 015, dashboard and settings rework, `/today` API.
3. **Extension UI + stats.** Section 6 badge, popup, banner, kind in confirm panel; section 7 stats page; `docs/leetgrinder.md` rewrite.

## Open points

- LeetCode's GraphQL endpoint may reject server-side requests (Cloudflare). If it does in practice, server fetch is dropped and metadata relies on the extension; web-logged problems show the slug until visited with the extension.
- Whether practice re-solves of not-due problems should update FSRS. Current rule: yes, every attempt feeds FSRS as today; only goal accounting treats them differently.
