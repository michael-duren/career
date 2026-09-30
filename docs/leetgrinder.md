# Leetgrinder

> **Upgrading from the curriculum.** The 84-day curriculum is retired (see [`retired/leetgrinder/`](../retired/leetgrinder/README.md)); Leetgrinder now tracks any LeetCode problem. Before deploying the upgrade, download **Export attempt history** from the side navigation. The migration keeps every attempt, code and analysis, and adds a problem catalog (`leetgrinder_problems`) seeded with the curriculum's curated optimal complexities. Day pages and `/leetgrinder/about` answer 410 Gone.

Open `/leetgrinder` after signing in. The curriculum contains 84 numbered sessions across 12 weeks, with 252 core problems and 48 optional problems. Sessions still unlock at your pace, but you can pin them to a calendar to see whether you are on track.

A side navigation links the dashboard, problems, review queue, about page, and settings. The dashboard (`/leetgrinder`) shows the next unfinished session with its core problems and a Start or Resume button, progress stats (sessions, unique solves, independent-solve rate, practice streak), the schedule status, today's reviews, upcoming sessions with their scheduled dates, and recent attempts. The problems page (`/leetgrinder/problems`) lists all 300 problems in a table with the LeetCode number, title (linking to your attempt history), difficulty, optimal time and space complexity (shown once you have attempted the problem), topic and session, your status, attempt count, last attempt date, and a link to LeetCode. Search matches title, slug, topic, and session title, and a bare number matches that LeetCode number exactly; filters narrow by difficulty, week, status, and core or optional; sorting is by curriculum order, difficulty, number, title, or most recent attempt. Filters live in the URL, so a filtered view can be bookmarked. The about page (`/leetgrinder/about`) holds the introduction, the guide to the two-hour budget, and the full curriculum. Pages use the main site's zinc and blue palette.

Allow two hours per session: 90 minutes for attempts and debugging, and 30 minutes for reading and review. Optional problems are for sessions where core work finishes early. Finishing a day advances to the earliest unfinished session without marking its problems solved.

## Schedule

Open `/leetgrinder/settings` to set the start date, time zone, and daily time. The schedule is 84 consecutive days, one session per day. Change the start date and the end follows (start + 83 days), or change the end date and the start follows. Changing both to an inconsistent pair is rejected. Clear both dates to remove the schedule. Settings saves use revisions, so two tabs cannot silently overwrite each other.

The dashboard shows a schedule card with the start and end dates, today's scheduled session, and your status. Status compares finished sessions with the sessions expected by the end of today in your time zone: `expected = clamp(today - start + 1, 0, 84)`. It reads "N sessions ahead", "On track", "N sessions behind", or "Starts in N days" before the start date. Dates never move on their own, and the dashboard's Start or Resume button still opens the earliest unfinished session.

Daily time runs from 2 to 4 hours in half-hour steps. Two hours covers the session plus one review. Every extra 25 minutes adds a review slot, so 3 hours gives two extra slots.

## Spaced repetition

Every attempted curriculum problem becomes a review card. Card state is never stored: each request replays the problem's attempts through FSRS ([go-fsrs](https://github.com/open-spaced-repetition/go-fsrs) v4, default weights and 90% desired retention), so corrections change due dates immediately. Unfinished rates Again. Struggled, or solved with help, rates Hard. An unassisted solve over 25 minutes rates Good, and 25 minutes or less rates Easy. Only the last attempt on each local day counts, so a same-day re-solve cannot inflate the interval. FSRS's minute-scale learning steps are turned off, since reviews happen at most once a day.

Each date gets a review plan the first time the dashboard, a session page, or the review queue opens that day, or when the first enabled notification is evaluated. Slots are one base slot, or none when today's scheduled session has required reading, plus the extra slots from daily time. Candidates are cards due by the end of the day that are not assigned in today's session. Picks go to the lowest estimated recall first, then problems from earlier curriculum weeks, then the most overdue. The plan is stored in `leetgrinder_review_plan` and frozen for that date. Raising daily time later adds picks; lowering it never removes them.

"Today's review" appears on the dashboard, on today's scheduled session page, and on the page of the session you are working through. Each pick shows the problem, its curriculum week, and a reason such as "Struggled 9 days ago · recall estimate 62%". A review is done once any attempt on that problem is logged that local day. Attempts recorded from a review card are marked as reviews in history. `/leetgrinder/reviews` lists every card with its due date and current recall estimate, plus how many due cards today's plan left out.

## Notifications

Leetgrinder can send reminders to [ntfy](https://ntfy.sh). In `/leetgrinder/settings`, set the ntfy server URL (default `https://ntfy.sh`), a topic, and optionally an access token, then subscribe to the same topic in the ntfy app. Anyone who knows a public ntfy.sh topic can read it, so use a long random topic name or a protected topic with a token. "Send test notification" posts a test message with the saved settings and shows the result. An empty topic turns every notification off.

The token field is write-only. The page shows only "Token set" or "No token set"; leave the field blank to keep the saved token, or tick "Clear the saved token". Tokens are encrypted with AES-256-GCM before they are stored (see `LEETGRINDER_SECRET_KEY` below) and never appear in pages, logs, or notification errors. Changing the server to a different host requires re-entering or clearing the token, so a saved token is only ever sent to the server it was entered for.

Each reminder can be turned on or off and has its own time, priority, and, where it applies, threshold:

| Reminder | Default | Sends when |
|---|---|---|
| Morning plan | off, 08:00 | Always: today's session, required reading, and planned reviews. |
| Missing work | on, 17:00 | Today's scheduled session is unfinished, or a planned review has no attempt today. Lists what is missing. |
| Behind schedule | on, 17:00, threshold 3 | You are at least the threshold number of sessions behind. |
| Review backlog | off, 17:00, threshold 10 | At least the threshold number of due reviews did not fit in today's plan. |
| Late escalation | off, 21:00, high priority | Same condition as missing work. |

A worker inside the server checks once a minute, using the schedule's time zone. Each reminder is evaluated once its time has passed and sends at most once per local date; if its condition is not met at that point it is logged as "Nothing to send" and not checked again that day. Enabling a reminder after its time has passed evaluates it on the next tick. Nothing is sent before the start date, more than one day after the end date, or while no schedule is set. Notifications carry a title, priority, tags, and a click link to the day page (or the dashboard or review queue), built from `PUBLIC_ORIGIN`.

Every send is recorded in `leetgrinder_notification_log`, keyed by reminder and local date. The worker claims a row with `INSERT ... ON CONFLICT` before sending, so ticks and restarts never double-send. A failed send is logged with the error and retried on a later tick at least five minutes apart, up to three retries (four attempts) that day. A send interrupted by a crash is retried the same way. Settings show the latest 14 log rows, including test sends.

## Attempts

Record each attempt as solved, struggled, or unfinished, with minutes spent, whether you used a hint or reviewed a solution, and the time and space complexity of your solution. A solve means the solution passed on LeetCode.

Complexity is a select of common classes, stored exactly as shown (`O(1)`, `O(log n)`, `O(√n)`, `O(n)`, `O(n log n)`, `O(n²)`, `O(n³)`, `O(2ⁿ)`, `O(n!)`), or "Other…" with free text such as `O(m·n)`, `O(V + E)` or `O(k log n)`. Other values are trimmed, must start with `O(` and end with `)`, and fit in 40 characters. Runs of spaces collapse to one, and `n^2`, `n^3`, `2^n`, `nlogn` and `logn` become `n²`, `n³`, `2ⁿ`, `n log n` and `log n`. Both values are required for solved and struggled attempts, including corrections of them, and optional for unfinished ones. The form works without JavaScript (the text field is always shown, and counts when "Other…" or nothing is selected); a small script hides it until "Other…" is chosen. One Go validator (`internal/leetgrinder/complexity.go`) enforces these rules for the form, the API, and storage; the extension mirrors it in `lib.js`, and both run the vectors in `extension/leetgrinder/test/complexity-vectors.json`.

Attempts logged by the extension can carry the code LeetCode judged and its language, capped at 64 KiB. The attempt history shows the stated complexities, and the code, escaped, in a collapsed "Submitted code" block with its language and size. Corrections change the stated complexities but keep the captured code. Web-logged attempts have no code. The dashboard counts distinct solved problems and independent solves separately. History retains repeated attempts. Use “Correct this attempt” to fix an entry; concurrent corrections cannot silently replace one another. A problem's attempt history page shows its optimal time and space complexity once you have logged at least one attempt; before that it says the values appear after your first attempt. See [sources and verification](leetgrinder-sources.md#optimal-complexity-table) for how the values were chosen and which are disputed.

Use “Export attempt history” to download Leetgrinder attempts and completed sessions as JSON. Each attempt includes `timeComplexity`, `spaceComplexity`, `code` and `codeLanguage`. This is separate from the existing workspace archive. There is no Leetgrinder import UI yet; PostgreSQL backups remain the full restore mechanism.

## Complexity analysis

When the server has an Anthropic API key, a background worker asks Claude to assess each attempt that has captured code and at least one stated complexity. The attempt history shows a "Claude's assessment" card under the code: the actual time and space complexity, a ✓ or ✗ against each stated value, whether the solution is optimal, and a short explanation. The problems table (for the latest attempt) and the dashboard's recent attempts show a "Check complexity" badge when a stated value was judged wrong. The assessment can be wrong for subtle amortised or input-dependent cases; your stated values are never changed.

The request carries the problem's title, number, slug and link, the language, the code, your stated time and space, and the curriculum's optimal time, space and note. The optimal table is a reference: when its note says the usual interview solution is slower than the best-known bound (Fibonacci by matrix exponentiation, for example), a solution that reaches that accepted optimum still counts as optimal. The code is fenced by a random boundary, and the model is told to treat it as untrusted data, so instructions hidden in comments or strings are ignored. The response must match a JSON schema (structured output); the server then accepts only those six fields, requires both complexities to pass the same normaliser as stated values, drops a match verdict for an unstated value, and caps the explanation at 2,000 characters. Everything is rendered escaped.

Analyses live in `leetgrinder_analyses`, one row per attempt, keyed to a SHA-256 of the analysed inputs (language, stated time, stated space, and code). Correcting a stated complexity changes the hash and queues the attempt again. The worker ticks once a minute, sends one request at a time (a PostgreSQL advisory lock keeps it to one across replicas), and handles the newest attempts first. A failed request ends the tick, so an outage costs at most one request a minute, and is retried after 1, 4 and 9 minutes, up to four requests in all. Refusal and truncation errors fail at once. Errors that would hit every attempt pause all workers for 15 minutes; the pause is stored in `leetgrinder_analysis_pause`, so it holds across restarts and replicas, and settings show it with its reason. Configuration errors (401, 402, 403 or 404: a bad key, billing, permissions or model) also leave the attempt pending without using a try, so fixing the environment needs no re-analyse. A 400 (for example a model that does not support the request, or no credit) pauses too but counts as a try, so a single bad attempt cannot block the queue. Stored errors keep only the HTTP status and error type, with the key redacted, never a request or response body. "Re-analyse" on the card resets the analysis to pending (a same-origin form post). Analyses are derived data and are not exported.

Configuration (environment only):

| Variable | Default | Meaning |
|---|---|---|
| `ANTHROPIC_API_KEY` | empty | Enables analysis. Never stored in the database, logged, or shown; settings show only whether it is configured. |
| `LEETGRINDER_ANALYSIS_MODEL` | `claude-sonnet-5` | Model ID. |
| `LEETGRINDER_ANALYSIS_DAILY_LIMIT` | `50` | Requests per local day in the schedule's time zone, counting retries. `0` sends none. |

The **Complexity analysis** section on `/leetgrinder/settings` shows whether a key is configured, the model, today's requests against the limit, and queue counts, with a toggle that defaults to on (stored in `leetgrinder_settings.analysis_enabled`). Settings are read before every request, so turning it off stops the next one; queued attempts wait. While analysis is off or no key is configured, attempt cards say so and hide "Re-analyse". Without a key the worker never makes a network call.

Cost: each request uses adaptive thinking at medium effort with output capped at 16,000 tokens (thinking plus the answer), and each request times out after five minutes. At Claude Sonnet 5's list price ($2 per million input tokens, $10 per million output tokens), a typical attempt costs about one to two cents; the worst case, 64 KiB of code and a full output, is about 20 cents. The default daily limit therefore bounds spending at roughly $10 a day, and far less in practice. Check current pricing before raising the limit or choosing a larger model.

Privacy: your captured code, with the problem and your stated complexities, is sent to Anthropic only while a key is configured and the toggle is on. Anthropic's API data-retention terms apply to those requests. Web-logged attempts have no code and are never sent.

## Browser extension

`extension/leetgrinder/` is a Manifest V3 extension for Chrome and Firefox that logs LeetCode submissions. When a submission on a curriculum problem is Accepted, it opens a confirm panel on the LeetCode page prefilled with the outcome (solved at 25 minutes or less, otherwise struggled), the minutes since the problem was first opened, and whether you opened the Solutions or Editorial tab. The panel asks for time and space complexity, and it cannot send a solved or struggled attempt without both. It captures the code and language of the judged submission and shows "Code captured (Python3, 1.2 KB)" with a checkbox to leave the code out. Nothing is sent until you confirm. After 25 minutes without an Accepted submission it offers to log the problem as unfinished; complexity is optional there, and the latest submission's code is attached if there is one. Problems outside the curriculum are ignored. See the [extension README](../extension/leetgrinder/README.md) for loading it unpacked and the manual test checklist.

The extension authenticates with a personal API token. Create one under **Browser extension tokens** on `/leetgrinder/settings`. The token (`lg_` plus 32 random bytes in base64url) is shown once, in the response to the create request; only its SHA-256 hash is stored in `leetgrinder_api_tokens`. The list shows each token's name, creation time, last use (recorded at most once a minute), and revocation. Revoked tokens are rejected immediately. At most 20 tokens can be active.

API, authenticated only with `Authorization: Bearer <token>` (session cookies are ignored; `/api/` bypasses the login redirect, so these handlers check the token themselves):

- `POST /api/leetgrinder/attempts` with JSON `{id, problemSlug, outcome, minutes, assisted, notes, isReview?, timeComplexity?, spaceComplexity?, code?, codeLanguage?}`. It uses the same idempotent attempt creation as the web form and records `source = 'extension'`. Retrying with the same `id` and values (compared after complexity normalisation, and including the code and language) returns the saved attempt; the same `id` with different values is `409`. Other responses: `401` bad or revoked token; `422` slug not in the curriculum, or complexity missing on a solved or struggled attempt, or not in `O(...)` form; `400` invalid fields or JSON, including code without a language (a LeetCode language slug of up to 32 letters, digits, or `_+#.-`) or a language without code; `413` code over 64 KiB or body over 96 KiB; `415` non-JSON body. An omitted `isReview` means false; omitted complexity and code fields mean empty.
- `GET /api/leetgrinder/problem/{slug}` returns `inCurriculum`, and for curriculum problems the title, difficulty, session, week, `todaysReview`, `reviewDone`, and `latestAttempt` (without its code). It plans today's reviews on first access, like the day page.

## Development

The feature uses Go templ, not Astro or a client-side editor. Original lessons are authored in `internal/leetgrinder/lessons*.templ`; assignments and reading references live beside them. The private source files in `data/algomonster` are not required to build or serve the feature.

The templ generator and runtime are pinned together in `go.mod`. Run:

```sh
make generate
make build
make test
make test-postgres
cd extension/leetgrinder && node --test
```

`make dev` generates templates before starting Go. Docker builds generate them too. Generated `_templ.go` files remain ignored, so run `make generate` before invoking `go test ./...` directly in a fresh checkout. Air watches `.templ` files and excludes generated files from triggering rebuild loops.

Leetgrinder pages require the existing login. Writes use same-origin HTML forms with server validation. Failed attempt saves display the submitted draft for retry. Attempt IDs prevent duplicate creates; revisions protect corrections. Tracking belongs to the app's existing single configured account.

The ntfy access token is stored encrypted with AES-256-GCM. Set `LEETGRINDER_SECRET_KEY` to 32 random bytes encoded as base64 (for example `openssl rand -base64 32`). Development derives a key from `JWT_SECRET` when it is unset. In production, `serve` refuses to start without the key once an encrypted token is stored. Changing the key makes a stored token unreadable, so re-enter the token afterwards.

See [sources and verification](leetgrinder-sources.md) for the reading bibliography and the command that checks all 300 assignments against LeetCode.

Migration 010 renames the existing tracking tables, indexes, and constraints to `leetgrinder` names while preserving saved attempts and completed days. Migration 009 keeps its historical SQL unchanged so previously applied checksums remain valid.

The shared local database already reserves migration 008 for note creation timestamps. Leetgrinder uses migrations 009 through 014. Migration 011 adds the settings row, review plans, notification log, API tokens, and the attempt `source` and `is_review` columns; existing attempts become web attempts that are not reviews. Run `make go-dev` to apply them; existing notes and learning history are retained. Migration 012 adds an `attempts` count to the notification log for bounded retries. Migration 013 adds `time_complexity`, `space_complexity`, `code` and `code_language` to attempts, with length checks (40 characters, 40 characters, 65,536 bytes, 32 characters); existing attempts keep empty values. Migration 014 adds `leetgrinder_analyses` (deleted with its attempt), the daily request counter `leetgrinder_analysis_usage`, the shared pause `leetgrinder_analysis_pause`, and `leetgrinder_settings.analysis_enabled` (default true).
