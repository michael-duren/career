# Weekly scheduler repair implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task by task. Steps use checkbox syntax for tracking.

**Goal:** Make scheduler placement visible and predictable, preserve valid time records, and prevent stale UI or synchronization work from losing progress.

**Architecture:** Keep React responsible for interaction previews and Go responsible for accepted reservations and accounting. Use one pure placement calculation for drag preview and submission. Coordinate client reads/writes, construct actual records from allowed fields, and make Google synchronization progress persist across bounded batches.

**Tech stack:** Astro, React, TypeScript, Zod, Go, PostgreSQL, Playwright.

**Spec:** `docs/superpowers/specs/2026-09-28-weekly-scheduler-design.md` plus the defect evidence and scope in `docs/reviews/2026-10-02-weekly-scheduler.md`.

## Global constraints

- Preserve original plans once a session starts and preserve historical attribution.
- Planning sessions must fit fully inside one configured scheduling day. Actuals may be outside configured hours.
- An overnight column belongs to the date on which its configured day begins.
- Local recurrence follows wall-clock time. Count elapsed duration; reject nonexistent local times and preserve computed absolute instants in repeated hours.
- Manual editor times retain minute precision. Drag and resize snap to 15 minutes.
- A date exception affects one occurrence. A future template edit preserves explicit exceptions.
- Server validation and revision checks remain authoritative. Preview validation must not promise remote availability is current.
- Keep failed proposals as drafts. Do not silently retry stale writes over another user's edits.
- Local saves survive export failures. Tokens remain encrypted and excluded from browser state and backups.
- The requested deliverable is this review and plan. Execution is a subsequent task.

## Review focus

- Delayed reads and writes across navigation must never combine one week's toolbar with another week's grid.
- Post-midnight actuals must use the scheduling date rather than the calendar date, especially across Sunday/Monday.
- Repeated-hour moves/resizes must preserve actual elapsed time and offsets.
- Every accepted actual request must produce a record accepted by backup/import validation.
- Large or slow Google batches must eventually reach all events without repeatedly spending the cycle on completed work.

## File responsibilities

- `src/pages/weekly-scheduler.astro`: page composition and deterministic hydration strategy if needed.
- `src/components/WeeklyScheduler.tsx`: viewed week, request coordination, editors, and shared scheduler state.
- Create `src/components/WeeklySchedulerGrid.tsx`: columns, interval rendering, pointer gestures, and previews.
- Create `src/lib/scheduler-placement.ts`: pure placement proposals and geometry. Keep date/instant primitives in `src/lib/scheduler.ts`.
- `src/components/WeeklySchedulerGoogle.tsx`: connection status and selection recovery.
- `src/styles/weekly-scheduler.css`: exact interval geometry, previews, and touch/scroll behavior.
- `internal/scheduler/domain.go`: actual construction, skipped/canceled transitions, and date ownership.
- `internal/database/scheduler.go`: recurrence validation against known dated reservations.
- `internal/server/scheduler.go`, `scheduler_google.go`, `scheduler_worker.go`: API orchestration, availability freshness, and bounded export work.
- `internal/database/scheduler_google.go` and a new migration following the latest migration present at execution: durable per-event synchronization progress.
- `tests/scheduler.test.ts`, `tests/e2e/scheduler.spec.ts`, `internal/scheduler/domain_test.go`, database scheduler tests, and new `internal/server/scheduler_test.go` / `scheduler_worker_test.go`: regression coverage at the owning boundary.

## Task 1: Establish the verification baseline

Issues: S12 and the review's environment limitations.

- [x] Record the current unrelated Go server compilation failure. Restore a buildable baseline through the existing LeetGrinder work before API/worker execution; keep that repair separate from scheduler changes.
- [x] Run `npm ci --ignore-scripts --no-audit --no-fund`, `node --test tests/scheduler.test.ts`, and `go test ./internal/scheduler`. Expected: the existing 8 client tests and domain tests pass.
- [x] Set `TEST_DATABASE_URL` to a disposable PostgreSQL database and run `go test -v ./internal/database -run 'Scheduler|ImportDoesNotFabricate'`. Expected: tests run without skips and pass.
- [x] Start a disposable authenticated app as required by `playwright.config.ts`. Replace synthetic input in the scheduler Playwright helper with real `page.mouse.move/down/up`; do not change expected user behavior to accommodate synthetic events.
- [x] Retire `tests/scheduler.browser.mjs` or rewrite it to use current pointer interaction and dialog selectors. Run `npx playwright test tests/e2e/scheduler.spec.ts`. Expected: current behavior passes except new regressions deliberately introduced by later tasks.

Do not interpret mocked frontend responses as proof of live API behavior. Controlled responses remain useful for race tests.

## Task 2: Protect stored actual records

Issues: S05, S06, S07.

**Files:** `internal/scheduler/domain.go`, `internal/scheduler/domain_test.go`, `internal/database/scheduler_test.go`, `internal/server/scheduler_test.go`.

**Interfaces:** Retain `Document.Apply(Mutation, time.Time, []Busy) error` and `Document.Validate() error`. Successful actual mutations must always create structurally valid sessions.

- [ ] Add `TestUnplannedActualIgnoresRecurringIdentity`. Submit missing `ruleId`, invalid `occurrenceDate`, invalid session date, and stale attention/conflict metadata. Assert accepted work has empty recurring identity, no exception, no attention/conflicts, and the date derived from actual start. Assert `Validate()` succeeds.
- [ ] Add `TestSkippedActualUsesOriginalSchedulingDate`. Skip a planned record using invalid or unrelated supplied dates. Assert canonical plan date is retained, no hours are counted, and validation succeeds. Reject malformed dates for statuses that permit relocation.
- [ ] Add `TestCanceledPlanCannotHideActualWork`. Choose the transition of rejecting actual writes on canceled records with a clear error requiring an unplanned log. Assert no actual is stored and totals stay zero.
- [ ] Construct unplanned actual sessions from validated assignment and actual data. Clear recurring metadata. Validate shared identity/date fields before status-specific checks; canonicalize skipped plan fields.
- [ ] Add a database export/import regression for accepted explicit and skipped actual records. Run `go test ./internal/scheduler ./internal/database ./internal/server -run 'Actual|Scheduler'` with the disposable database. Expected: new tests pass, import accepts every accepted actual, and historical-plan tests still pass.
- [ ] Review the diff and commit this independently testable backend repair.

## Task 3: Unify placement and preserve time semantics

Issues: S03, S04, S08; supports S01.

**Files:** new `src/lib/scheduler-placement.ts`, `src/lib/scheduler.ts`, `src/components/WeeklyScheduler.tsx`, `internal/server/scheduler.go`, `internal/scheduler/domain.go`, client/domain/API tests.

**Interfaces:** Add `proposePlacement(input: PlacementInput, week: SchedulerWeek, now: Date): PlacementResult`. Define `PlacementInput` as assignment placement, session move, or session resize variants. Move inputs include target scheduling date and snapped start minute after subtracting the grab offset. Resize inputs include session, edge, and snapped elapsed delta minutes. Define `PlacementResult` as either a valid `{ kind: 'valid', draft: SessionDraft, start: string, end: string, conflictIds: string[] }` or `{ kind: 'invalid', message: string }`. An overlapping proposal remains valid geometry with nonempty conflicts; invalid geometry/time has no submittable draft.

- [ ] Add literal expected-value tests for 09:00–10:00 moved using a 30-minute grab offset to a 10:00 pointer target. Expected interval: 09:30–10:30.
- [ ] Add move/resize tests for Chicago Nov 1, 2026 repeated-hour work. Expected 01:30 CDT–01:30 CST remains `06:30Z–07:30Z` with 60 elapsed minutes; changed endpoints keep their computed offset. Add spring-gap rejection.
- [ ] Add overnight attribution tests for 09:00–02:00 days, Sunday/Monday, dated overrides, and work outside configured hours. Use server `ActualDate` as the authority. Normalize explicit actual dates at the server boundary and derive new unplanned session dates from that result. Return canonical records; update the client draft from the response on success.
- [ ] Implement the shared proposal function. Preserve both computed instants in `originalStart` and `originalEnd` for move/resize drafts. Resolve changed manual fields through `localInstant`. Validate day bounds, eligibility, local accepted-plan or actual overlaps, and future actual ends for previews.
- [ ] Use `proposePlacement` for release and resize saves. Record grab offset and active pointer ID at start; compute release position from the actual release event.
- [ ] Run `node --test tests/scheduler.test.ts` and the relevant Go actual-date tests. Expected: exact dates, instants, and durations pass while manual minute precision and historical offsets remain intact.
- [ ] Commit the time and placement repair.

## Task 4: Render placement previews and support ordinary gestures

Issues: S01, S09, S12.

**Files:** new `src/components/WeeklySchedulerGrid.tsx`, `src/components/WeeklyScheduler.tsx`, `src/styles/weekly-scheduler.css`, `tests/e2e/scheduler.spec.ts`.

**Interfaces:** Grid consumes the authoritative week and Task 3 proposals. It emits a `SessionDraft` on accepted release; the parent owns asynchronous saves. Preview state is independent of accepted sessions.

- [ ] Add real mouse tests that hold the pointer over a day before release and assert the preview's date, time labels, top, and duration. Drop afterward and assert the API interval equals the preview. Cover assignment creation and existing-session move.
- [ ] Add held-pointer resize tests for both edges. Assert preview geometry changes before release and the final elapsed duration matches.
- [ ] Add overlap, unavailable-range, out-of-timeline, future-actual, and DST-gap cases. Assert visibly explained invalid placement and no mutation when invalid. For server-discovered conflicts after release, assert retained editable draft.
- [ ] Extract the grid and render `.scheduler-drop-preview` with `pointer-events:none`, clear destination labels, valid/conflicting appearance, and exact interval geometry. Keep a minimum-size action/label affordance separate from interval length.
- [ ] Implement one-pointer ownership, lost-capture/Escape/pointercancel cleanup, and auto-scroll near viewport edges. Clear gestures on week change. Use a deliberate touch drag handle or activation so requirement-list scrolling remains available.
- [ ] Test at desktop 1280×900 and mobile 390×844. Assert offscreen targets are reachable, ordinary touch scrolling works, cancellation writes nothing, and successful gestures do not flash the editor.
- [ ] Run `npx playwright test tests/e2e/scheduler.spec.ts`. Expected: preview assertions pass during the gesture and real input saves the expected intervals. Commit.

## Task 5: Coordinate requests and refresh local state

Issues: S02, S10, R02.

**Files:** `WeeklyScheduler.tsx`, `WeeklySchedulerGoogle.tsx`, client and browser tests.

**Interfaces:** Keep `schedulerRequest` as the transport. Add one request generation for viewed-week results and a synchronous in-flight mutation ref. Drafts retain their owning week and revision independently of the currently rendered week.

- [ ] Add controlled response-order tests: delay a Jan 7 save, navigate to Jan 14, then finish the save. Assert toolbar and first grid date both remain Jan 14. Repeat with a delayed read finishing after a save; assert newer saved state remains.
- [ ] Test two rapid saves, stale-revision responses, and a failed save after navigation. Assert one mutation at a time, no silent conflict retry, and a recoverable draft with its original week.
- [ ] Guard result application by request generation and viewed week. Invalidate obsolete reads when a mutation begins/completes. Reload the current week when a save for another view completes. Keep draft creation and discard actions consistent while saving.
- [ ] Add a visible-page refresh every 60 seconds plus focus/visibility refresh, preserving drafts and using the same coordination. Add a normal Refresh schedule action. Clear errors after a successful refresh. Update start/end editability from a clock state.
- [ ] Refresh Google status alongside visible schedule refresh. Stop availability polling when reconnection is required. Refetch selection/status revisions after conflicts while retaining user selections.
- [ ] Use a browser clock and controlled worker-updated responses to assert local-only assumed actuals appear after a session ends. Assert reconnect health appears and no repeated availability calls occur until reconnection.
- [ ] Run focused browser regressions and client tests. Expected: no stale results or lost drafts. Commit.

## Task 6: Correct grid geometry and day discovery

Issues: S11 and quick-add/warning improvements.

**Files:** `WeeklySchedulerGrid.tsx`, `WeeklyScheduler.tsx`, `scheduler.ts`, `weekly-scheduler.css`, `internal/scheduler/model.go`, `domain.go`, client/domain/browser tests.

- [ ] Add tests for two adjacent 15-minute sessions, actuals at 03:00 and after the configured end, a repeated-hour actual, and a configured 02:30 boundary on Chicago March 8, 2026. Assert both short sessions are selectable; actuals remain discoverable; elapsed duration is visible; every scheduling date has a column.
- [ ] Derive the display axis from configured ranges and visible records. Render invalid-boundary days as dated placeholders with a reason; keep server accepted-day validation intact. Include explicit day validity in the API/client schema rather than interpreting missing array entries as date columns.
- [ ] Separate interval height from text/actions and explain repeated-hour offsets. Keep post-midnight labels and scheduling date clear.
- [ ] Make quick-add defaults start within the selected day and choose a duration that fits remaining room. If no valid future slot is available, open an explanatory draft rather than manufacture an invalid placement.
- [ ] Add structured warning targets to the API alongside message text and link warnings to the owning date/goal/session. Assert a conflict warning opens the affected editable record.
- [ ] Run domain, client, and focused browser checks. Expected: no clipped/unreachable actuals or misleading overlapping short-block geometry. Commit.

## Task 7: Check recurrence against known future exceptions and reservations

Issue: R04.

**Files:** `internal/database/scheduler.go`, `internal/scheduler/domain.go`, `internal/database/scheduler_test.go`.

- [ ] Add `TestRuleChecksKnownReservationBeyondGenerationWindow`: create a one-off six months out, then create an otherwise valid unbounded weekly rule that overlaps it. Assert rejection names that future date; adjacent placement succeeds.
- [ ] Add a distant date-override case and a preserved date exception. Assert future rule validation respects both without rewriting history or expanding every intervening date.
- [ ] Extend rule validation to occurrence dates implied by known future reservations and dated boundaries outside the normal 62-day generation window. Keep routine background generation bounded.
- [ ] Run the scheduler database suite with a disposable database. Expected: conflicts in known future dates are reported, and existing exception/revision tests pass. Commit.

## Task 8: Make Google work advance across retries

Issues: R01, R03; verify R02 recovery.

**Files:** `internal/server/scheduler_google.go`, `scheduler_worker.go`, new server tests, `internal/database/scheduler_google.go`, a new migration, Google database/client tests, `docs/weekly-scheduler.md`.

**Interfaces:** Add server injection of a `scheduler.GoogleClient` and clock for controlled provider responses. Persist per-event desired generation/fingerprint and successful synchronization progress. Retain stable mapped event IDs and the account/destination locking contract.

- [ ] Add a worker test with more pending events than fit one bounded batch, a failure after several successful writes, and a retry. Assert all events eventually synchronize, completed unchanged writes are skipped on ordinary retries, and event identities remain stable.
- [ ] Add cancellation during export, disconnect/reconnect, changed local plan during a batch, and deleted remote event cases. Assert no unrelated event deletion, no lost newer generation, and periodic reconciliation still restores remote edits/deletions even when fingerprints are unchanged.
- [ ] Implement deterministic bounded batches and durable progress. Ordinary outbox retries operate on changed/pending events; periodic reconciliation advances through the full accepted export set with its own cursor. Keep external requests outside local reservation transactions.
- [ ] Add request-count tests for manual/periodic availability refresh. Assert one FreeBusy query per refresh operation, fresh range-aware week reads, and a new fresh check before accepting a planning mutation. Test expiry, larger requested ranges, disconnect during fetch, and provider failure.
- [ ] Return refreshed week data from refresh or reuse a fresh range-aware read cache. Keep provider failures as recoverable drafts and preserve local saves on export failures.
- [ ] Run Go domain/database/server suites and scheduler Playwright tests. Expected: controlled provider retries converge and local interaction behavior remains valid. Update integration documentation and commit.

## Final review and handoff

- [ ] Confirm each S01–S12 issue and R01–R04 risk has a passing regression or a documented measured result.
- [ ] Run `node --test tests/scheduler.test.ts`, `go test ./internal/scheduler ./internal/database ./internal/server`, and `npx playwright test tests/e2e/scheduler.spec.ts` against the disposable app/database. Report skips and baseline blockers explicitly.
- [ ] Exercise create, move, both resize edges, cancel gesture, recurring date/future edits, actual correction, skip/restore, overnight work, stale save, and narrow-screen editor equivalents in the running app.
- [ ] Inspect the diff for historical-plan changes, duplicate accounting, credentials in responses/exports, and unrelated edits. Record provider integration limits if a live Google account remains unavailable.

Optional follow-up after the repairs: measure pointer-frame cost with many sessions/busy intervals, group intervals by day, make SSR initial headings deterministic across timezone/locale differences, and improve parsing of non-JSON network/auth errors. These do not need to delay the correctness work.
