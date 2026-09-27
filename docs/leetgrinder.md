# Leetgrinder

Open `/leetgrinder` after signing in. The curriculum contains 84 numbered sessions across 12 weeks, with 252 core problems and 48 optional problems. Sessions still unlock at your pace, but you can pin them to a calendar to see whether you are on track.

Allow two hours per session: 90 minutes for attempts and debugging, and 30 minutes for reading and review. Optional problems are for sessions where core work finishes early. Finishing a day advances to the earliest unfinished session without marking its problems solved.

## Schedule

Open `/leetgrinder/settings` to set the start date, time zone, and daily time. The schedule is 84 consecutive days, one session per day. Change the start date and the end follows (start + 83 days), or change the end date and the start follows. Changing both to an inconsistent pair is rejected. Clear both dates to remove the schedule. Settings saves use revisions, so two tabs cannot silently overwrite each other.

The overview shows a schedule card with the start and end dates, today's scheduled session, and your status. Status compares finished sessions with the sessions expected by the end of today in your time zone: `expected = clamp(today - start + 1, 0, 84)`. It reads "N sessions ahead", "On track", "N sessions behind", or "Starts in N days" before the start date. Dates never move on their own, and the "Continue learning" link still opens the earliest unfinished session.

Daily time runs from 2 to 4 hours in half-hour steps. Two hours covers the session plus one review. Every extra 25 minutes adds a review slot, so 3 hours gives two extra slots.

## Spaced repetition

Every attempted curriculum problem becomes a review card. Card state is never stored: each request replays the problem's attempts through FSRS ([go-fsrs](https://github.com/open-spaced-repetition/go-fsrs) v4, default weights and 90% desired retention), so corrections change due dates immediately. Unfinished rates Again. Struggled, or solved with help, rates Hard. An unassisted solve over 25 minutes rates Good, and 25 minutes or less rates Easy. Only the last attempt on each local day counts, so a same-day re-solve cannot inflate the interval. FSRS's minute-scale learning steps are turned off, since reviews happen at most once a day.

Each date gets a review plan the first time the overview, a session page, or the review queue opens that day. Slots are one base slot, or none when today's scheduled session has required reading, plus the extra slots from daily time. Candidates are cards due by the end of the day that are not assigned in today's session. Picks go to the lowest estimated recall first, then problems from earlier curriculum weeks, then the most overdue. The plan is stored in `leetgrinder_review_plan` and frozen for that date. Raising daily time later adds picks; lowering it never removes them.

"Today's review" appears on the overview, on today's scheduled session page, and on the page of the session you are working through. Each pick shows the problem, its curriculum week, and a reason such as "Struggled 9 days ago · recall estimate 62%". A review is done once any attempt on that problem is logged that local day. Attempts recorded from a review card are marked as reviews in history. `/leetgrinder/reviews` lists every card with its due date and current recall estimate, plus how many due cards today's plan left out.

## Attempts

Record each attempt as solved, struggled, or unfinished, with minutes spent and whether you used a hint or reviewed a solution. A solve means the solution passed on LeetCode. The overview counts distinct solved problems and independent solves separately. History retains repeated attempts. Use “Correct this attempt” to fix an entry; concurrent corrections cannot silently replace one another.

Use “Export attempt history” to download Leetgrinder attempts and completed sessions as JSON. This is separate from the existing workspace archive. There is no Leetgrinder import UI yet; PostgreSQL backups remain the full restore mechanism.

## Development

The feature uses Go templ, not Astro or a client-side editor. Original lessons are authored in `internal/leetgrinder/lessons*.templ`; assignments and reading references live beside them. The private source files in `data/algomonster` are not required to build or serve the feature.

The templ generator and runtime are pinned together in `go.mod`. Run:

```sh
make generate
make build
make test
make test-postgres
```

`make dev` generates templates before starting Go. Docker builds generate them too. Generated `_templ.go` files remain ignored, so run `make generate` before invoking `go test ./...` directly in a fresh checkout. Air watches `.templ` files and excludes generated files from triggering rebuild loops.

Leetgrinder pages require the existing login. Writes use same-origin HTML forms with server validation. Failed attempt saves display the submitted draft for retry. Attempt IDs prevent duplicate creates; revisions protect corrections. Tracking belongs to the app's existing single configured account.

ntfy notifications and the browser extension are still to come. Migration 011 already creates their tables: `leetgrinder_api_tokens`, `leetgrinder_notification_log`, the ntfy columns on `leetgrinder_settings`, and `source` on attempts (`web` or `extension`).

The ntfy access token will be stored encrypted with AES-256-GCM. Set `LEETGRINDER_SECRET_KEY` to 32 random bytes encoded as base64 (for example `openssl rand -base64 32`). Development derives a key from `JWT_SECRET` when it is unset. In production, `serve` refuses to start without the key once an encrypted token is stored. Changing the key makes a stored token unreadable, so re-enter the token afterwards.

See [sources and verification](leetgrinder-sources.md) for the reading bibliography and the command that checks all 300 assignments against LeetCode.

Migration 010 renames the existing tracking tables, indexes, and constraints to `leetgrinder` names while preserving saved attempts and completed days. Migration 009 keeps its historical SQL unchanged so previously applied checksums remain valid.

The shared local database already reserves migration 008 for note creation timestamps. Leetgrinder uses migrations 009 through 011. Migration 011 adds the settings row, review plans, notification log, API tokens, and the attempt `source` and `is_review` columns; existing attempts become web attempts that are not reviews. Run `make go-dev` to apply them; existing notes and learning history are retained.
