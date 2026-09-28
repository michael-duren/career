# Weekly scheduler implementation plan

> **For agentic workers:** Use superpowers:subagent-driven-development. Track implementation and verification below.

**Goal:** Implement the recurring weekly scheduler, historical actuals, timeline weekday policies, and optional Google Calendar synchronization.

**Architecture:** Go owns schedule generation, reservation validation, revisions, and accounting. PostgreSQL persists the workspace schedule and history under a workspace lock. React renders the authenticated API inside the existing Astro layout. Google writes use durable pending work after local commits.

**Tech Stack:** Go, PostgreSQL, React, Astro, TypeScript.

**Spec:** docs/superpowers/specs/2026-09-28-weekly-scheduler-design.md

## Global constraints

- Monday week start, 15-minute drag and resize increments, 60-minute new blocks.
- Existing goals default to all seven weekdays. Empty selection means zero requirements.
- Preserve started plans and historical attribution. Never automatically complete goals.
- Scheduler timezone is an editable IANA name. Initial day is 05:00–20:30.
- Enforce reservations on the server, with revisions and serialized writes.
- Keep credentials out of browser storage and backups.

## Review focus

- DST gaps, repeated times, and overnight attribution must preserve real elapsed time.
- Generic/MCP writes, subgoal moves, and deletion must preserve historical attribution.
- Concurrent reservations and stale revisions must preserve drafts without double booking.
- Repeated background runs and Google retries must not duplicate sessions or events.
- Missing estimates, zero hours, and empty weekdays must remain distinct.

## Task 1: Weekday policy

Files: goal model and persistence, timeline schemas/calculations, GoalSchedule editor, migration 015, associated tests.
Interface: `selectedWeekdays?: number[]`, ISO weekday numbers 1 (Monday) through 7 (Sunday); omitted means all seven. Persist as JSONB, expose in all goal read/write/import paths.

- [x] Add failing Go validation and TypeScript requirement/workload tests for default, empty, partial weeks, and invalid/duplicate weekdays.
- [x] Implement policy persistence, goal editor checkboxes, and selected-day timeline calculations and labels.
- [x] Run targeted tests, then existing suites.

## Task 2: Scheduler service

Files: internal/scheduler/*, internal/database/scheduler*.go, migration 016, internal/server/scheduler.go, shared goal mutation hooks and transfer support.
Interface: authenticated `/api/scheduler/week?week=YYYY-MM-DD`, settings and mutation endpoints; JSON types published in internal/scheduler/model.go and mirrored in src/lib/scheduler.ts. All mutation bodies carry revisions. Domain operations accept an explicit clock. Publish exact request/response types before frontend implementation.

- [x] Write failing tests for the spec's accounting examples, recurrence identity, exceptions, day validation, DST, actual replacement, and history.
- [x] Implement pure domain operations, transactional persistence, goal reconciliation, background catch-up, backup support, and authenticated routes.
- [x] Test concurrent writes, stale revisions, imports, and history against PostgreSQL.

## Task 3: Scheduler interface

Files: src/lib/scheduler.ts, src/components/WeeklyScheduler*.tsx, src/pages/weekly-scheduler.astro, sidebar and scheduler styles/tests.
Consumes Task 2 API. Google connection UI consumes Task 4 API.

- [x] Add tests for dates, drag snapping, editor requests, and response parsing.
- [x] Build week navigation, summaries, seven-column calendar, mobile day selector, drag/move/resize, accessible editor, recurrence, fixed commitments, actual editing, and settings.
- [x] Preserve failed drafts and highlight conflicts. Display attention warnings and Google status.
- [x] Verify desktop and narrow-screen creation and editing in a running browser. (Verified via a running dev server and direct API/HTML checks — no browser automation tool was available in this session, so drag/resize interaction was verified by code review rather than live pointer input.)

## Task 4: Google Calendar synchronization

Files: internal/scheduler/google*, internal/database/scheduler_google.go, migration 017, internal/server/scheduler_google.go, configuration, server worker wiring, operational documentation.
Interface: `/api/scheduler/google/status`, `/connect`, `/callback`, `/calendars`, `/refresh`, `/disconnect`. Credentials and event mappings are server-owned. Fresh busy checks precede planning mutations; outbound writes follow commit.

- [x] Confirm official Google OAuth scopes and API behavior.
- [x] Add failing tests for encrypted credentials, busy calendar filtering, refresh failures, reconnect, stable event IDs, and retry behavior.
- [x] Implement session-bound OAuth, selected calendars, encrypted token persistence, busy reads, durable export reconciliation, and background refresh/export.
- [x] Verify with controlled HTTP responses. Test a real account only if credentials are available. (No Google credentials were available in this session; verified via the existing controlled-response test suite only.)

## Task 5: Integration, review, and PR

- [x] Run Go and frontend suites, typecheck, build, database integration, and browser checks. All green: `go build/vet/test ./...` (including PostgreSQL integration via `TEST_DATABASE_URL`), `npm test`, `npm run tc`, `npm run build`. Also ran the Go server against the local dev DB and exercised `/api/scheduler/week` and `/api/scheduler/mutate` live (fetch, create, conflict-detect, cancel) via curl, and confirmed the WeeklyScheduler React island hydrates in the rendered page.
- [x] Dispatch independent subagent reviews for spec compliance and implementation correctness; resolve material findings and recheck.

  Two subagent reviews ran against the full diff: one for spec/plan compliance, one hunting silent failures and inappropriate fallbacks. Findings resolved:
  - **Critical — fabricated history:** `Document.Generate` could create a *new* session for a date already in the past whenever a goal or subgoal's eligibility changed retroactively (extended end date, reopened subgoal), and `Finalize` would then record it as `assumed` actual time — inventing hours in closed weeks. Fixed by threading a `sinceDate` watermark (the document's last-reconciled date) through `Generate`; a rule occurrence with no existing session is now skipped once its date is before that watermark. Regression test: `TestGenerateDoesNotReviveSessionsBeforeLastReconcile`.
  - **Nondeterministic revision churn:** conflict IDs (map iteration) and Google busy-interval IDs (index into a nondeterministically-ordered slice) varied run to run, and `schedulerReconcileTx` rotated the document revision unconditionally on every goal save and every one-minute tick — causing spurious 409s on open scheduler tabs and needlessly marking the Google export outbox dirty every minute. Fixed by sorting conflict IDs, iterating `FreeBusy` calendars in stable input order, and only rotating/saving the document when its serialized content actually changed (`schedulerReconcileTx`, `schedulerUpdate` already did this correctly).
  - **Unbounded generation cost:** `schedulerRange` always scanned back to the oldest rule's `EffectiveFrom`, so generation cost (and time holding the goals advisory lock) grew without bound as rule history accumulated. Now clamped to the last-reconciled watermark when that's more recent.
  - **Historical-attribution bugs:** canceling a DST-gap/plan-less generated occurrence produced a session with no plan, no actual, and state `canceled`, which `Document.Validate` rejected — breaking workspace import/restore. Canceling a manually-logged unplanned actual left it counted in `ActualHours` forever. Fixed: `Validate` now accepts a plan-less `canceled` session; `cancel` now deletes a plan-less, non-recurring actual outright (nothing to preserve) and rejects canceling any session that has recorded actual time tied to a recurring rule (must use skip instead). The scheduler UI now offers "Remove this session" for unplanned actuals with no plan, and disables "Restore as planned" until the planned session has actually ended (previously it could be used mid-session to record time that hadn't happened yet). Regression tests: `TestCancelingPlanlessOccurrenceRoundTripsThroughImport`, `TestCancelRemovesUnplannedActualButPreservesRecurringHistory`, `TestAssumedActualRejectedBeforePlannedEnd`.
  - **Invisible Google connection failures:** a credential decrypt/unmarshal failure in `googleAccess` never updated the stored connection health, so `/api/scheduler/google/status` could report a healthy connection indefinitely while every sync tick silently failed. `recordGoogleError` also discarded the error from its own `UpdateSchedulerGoogleHealth` write. Both now record/log correctly. Swallowed `json.Marshal`/`json.Unmarshal` errors around the credential-sealing and optimistic-concurrency revert paths (`scheduler_google.go`, `database/scheduler.go`) are now checked.
  - **No panic isolation:** the scheduler background loop (`RunScheduler`) had no panic recovery, so a single panic (crypto, JSON decoding, timezone data) would take down the whole API process, not just scheduling. Now recovers and logs per tick.
  - **Generic storage failures were unlogged:** `failure()` in routes.go returned a generic message on any non-sentinel error without logging it server-side. Now logs the real error before responding.

  A third review pass (against the pushed PR) found the first fix above was incomplete, plus two low-severity issues. Resolved:
  - **Same-day retroactive eligibility still fabricated assumed time:** the `sinceDate` watermark had only day granularity, and the background worker ticks every minute, so `sinceDate` was normally *today* — an occurrence dated today whose start had already passed (a Monday 09:00 rule, goal made eligible at 15:00 the same Monday) was not covered by the date comparison. Replaced the date-based guard with an instant-based one, threaded through `Generate` as `sinceInstant`: a new occurrence is now skipped whenever its computed `Plan.Start` is at or before that instant, not just when its date is earlier. Multi-day downtime catch-up still works, since a missed occurrence's start still falls after the last reconcile's instant. Regression test: `TestGenerateDoesNotReviveSameDayEligibilityChange` (plus the existing `TestGenerateDoesNotReviveSessionsBeforeLastReconcile`, updated to the new signature).
  - **Health-write race failed planning mutations:** `schedulerPlanningAvailability` returned an error (surfaced as a 503 "availability check failed") whenever persisting the refreshed Google health snapshot lost its optimistic-concurrency race against the background worker — even though the fresh busy data was already in hand and the draft would have been fine to keep. Now logs and returns the busy data instead of failing the mutation.
  - **Client-supplied fields trusted on session/rule creation:** a new session mutation kept whatever `ruleId`, `occurrenceDate`, and `exception` the client sent instead of clearing them (only an *edit* of an existing session reset these from the stored value), and a new/edited rule's assignment was never validated. Both could produce a document that later failed `Validate` on workspace import. Fixed: a genuinely new session always clears those three fields, and the `rule` action now runs the same `d.assignment` check used everywhere else. Regression tests: `TestNewSessionIgnoresClientSuppliedRecurringIdentity`, `TestRuleActionRejectsInvalidAssignment`.

  A fourth review pass caught a regression the third pass introduced: the initial fix stored the `sinceInstant` watermark as a new `LastReconciledAt` field on the scheduler `Document` itself, set on every `Reconcile`. Since that field changes on every one-minute tick, `schedulerReconcileTx`/`schedulerUpdate`'s "only save when content changed" check always saw a change — bringing back both the spurious-409 and outbox-dirtying bugs the second pass had just fixed, confirmed with a throwaway test that ticked between a load and a mutation. Fixed by moving the watermark out of the JSONB document entirely into its own `scheduler_state.reconciled_at` column (migration 018), updated independently of `document` via a dedicated watermark-only write when nothing else changed, and rescoped the `scheduler_google_dirty` trigger to `AFTER INSERT OR UPDATE OF document` so a watermark-only write can no longer dirty the Google outbox either. Regression test: `TestSchedulerIdleTickDoesNotRotateRevisionOrDirtyOutbox` (verified to fail against the prior commit and pass against this one).

  Known limitations not fixed in this pass (tracked for follow-up, not blocking):
  - There is still no way to end or delete a **fixed commitment's** recurring rule (goal-linked rules stop naturally when the goal ends; a bare "Work every Monday" commitment cannot be stopped short of editing it into something else). Editing a rule with "This and future occurrences" only changes its shape, not its lifetime.
  - The Google export worker still re-sends every accepted future session on each periodic (5-minute) reconciliation rather than diffing against what was last sent; the fix above removes the *spurious* every-minute resends but the periodic full resync is unchanged from the original design.
  - `schedulerGoogleCalendars` and `schedulerGoogleRefresh` don't take the `lockGoogleConnection` advisory lock that `connect`/`select`/`disconnect` do, so they can race the background worker's own health writes; the resulting `ErrConflict` is now logged rather than silently dropped, but the race itself isn't closed.
- [x] Update this plan with actual verification and any limitations.
- [ ] Commit, push the feature branch, and create the PR with `gh pr create --body-file`.
