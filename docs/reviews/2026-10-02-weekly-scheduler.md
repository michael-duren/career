# Weekly scheduler review

Reviewed 2026-10-02. Scope includes the Astro page, all three scheduler React components, client time helpers, CSS, Go domain, database persistence and reconciliation, API handlers, Google client and worker, import/export, and scheduler tests. No feature code was changed.

The page is a small wrapper. Interaction behavior lives in `src/components/WeeklyScheduler.tsx`; reservation and accounting rules live in `internal/scheduler/domain.go`.

## Evidence and limits

- `node --test tests/scheduler.test.ts`: all 8 tests passed after installing the locked dependencies with `npm ci --ignore-scripts --no-audit --no-fund`.
- `go test ./internal/scheduler`: passed, including existing recurrence, accounting, DST, cancellation, and Google-client coverage.
- `go test -v ./internal/database -run Scheduler`: scheduler database tests skipped because `TEST_DATABASE_URL` is unset. The package's successful exit does not verify database behavior.
- `go test ./internal/server`: compilation fails in unrelated `internal/server/leetgrinder.go` references to missing `leetgrinder.Retired`, `Unavailable`, `Overview`, `Problems`, and `Reviews` symbols. Full API and worker verification is blocked by that baseline failure.
- Ran the actual Astro page in headless Chromium against controlled scheduler API responses. Used real `page.mouse` input. Confirmed the missing preview and save/navigation race. This verifies frontend behavior, not a live Go/database round trip.
- Four temporary Go probes called the real domain methods. Confirmed overnight date attribution, acceptance of forged unplanned identities, invalid skipped dates, and actual time on canceled sessions. The temporary test was removed from the source tree.
- Ran a client time-conversion probe demonstrating that losing original instants in a DST fold turns a valid one-hour interval into an invalid equal-time interval.
- No live Google account was connected. Google worker concerns below are code-level findings requiring controlled integration tests.

Temporary reproduction files remain at `/tmp/weekly-scheduler-browser-review.mjs`, `/tmp/weekly-scheduler-review-probe_test.go`, and `/tmp/weekly-scheduler-drag-review.png`. The browser probe needs the local Astro server on port 4539. Copy the Go probe into `internal/scheduler` only for its test run, then remove it.

## Issues to fix

Priority P1 means a primary interaction, accounting, or stored-data correctness problem. P2 means a narrower functional problem or a significant verification gap. Each item identifies the strength of its evidence.

### S01 · P1 · No destination preview while dragging or resizing

**Evidence:** browser reproduced, source confirmed. `WeeklyScheduler.tsx:130-145,184-205,224`; `weekly-scheduler.css:1`.

`pointerDragMove` calculates `hoverDate` and `hoverMinute`, but only renders a fixed label beside the pointer. That label contains the assignment title, with no proposed time, duration, destination outline, or validity indicator. Resize changes the tiny handle text rather than the block's geometry. A user cannot judge a placement before releasing.

**Fix:** render a translucent block in the target column with the exact snapped start, end, and duration that will be submitted. Preview resizing too. Distinguish overlaps, unavailable hours, ineligible goals, future actuals, and DST gaps. Exclude the moving session from its own overlap check. Keep the server authoritative when availability or revisions change.

**Acceptance:** inspect the preview before pointer release. Its interval must equal the submitted interval. Canceling or leaving the grid clears it and sends no mutation.

### S02 · P1 · Save responses can replace a newly selected week

**Evidence:** browser reproduced. `WeeklyScheduler.tsx:29-62,117-122,208,217`.

Reads have `loadSequence` protection; mutations call `setWeek(result)` unconditionally. With a delayed save followed by Next week, the toolbar showed Jan 14–20 while the grid showed Jan 7–13. A stale read can also arrive after a mutation and overwrite the newer result. `saving` is a UI flag, not an operation lock, and drag sources remain enabled during a save.

**Fix:** coordinate reads and writes using the viewed week, request generation, and a synchronous mutation guard. Apply results only to their owning view; reload the current view when another week's save completes. Preserve failed proposals independently of the currently selected week. Disable competing mutations while one is pending.

**Acceptance:** delayed save plus navigation, delayed read plus save, two rapid drops, and save failure after navigation never mix toolbar/grid weeks or lose drafts.

### S03 · P1 · Drag and resize misattribute overnight actual work

**Evidence:** domain probe plus source trace. `WeeklyScheduler.tsx:175-181,199-203`; `domain.go:697-700,862-869`.

The client sets actual `date` to `startFields.date` or `start.date`. The server assigns post-midnight work inside yesterday's overnight interval to yesterday. With a 09:00–02:00 day, an Oct 2 01:00 start belongs to Oct 1. Moving or resizing actuals into that interval sends Oct 2 and is rejected. Explicit start-date edits can also leave `draft.date` inconsistent.

**Fix:** share the scheduling-date rule between actual placement, resize, and editor submission. Prefer an authoritative server-derived actual date; if retaining explicit client dates, derive them using the same interval precedence and overnight containment rules. Preserve a separate calendar date for time inputs.

**Acceptance:** post-midnight moves and resizes save to the correct scheduling day, including Sunday night into Monday, date overrides, and starts outside configured hours.

### S04 · P1 · Drag and resize lose DST-fold instants

**Evidence:** client conversion probe plus source trace. `WeeklyScheduler.tsx:169-181,192-203`; `scheduler.ts:77-84`.

`placeItem` computes a real end instant, converts both endpoints to wall times, then omits `originalStart` and `originalEnd`. `draftMutation` resolves ambiguous times to their earlier occurrence. A valid Chicago Nov 1 01:30 CDT–01:30 CST one-hour interval becomes equal instants and throws "End must be after start." Resize preserves only the unchanged endpoint, so a changed endpoint in the second repeated hour can also resolve incorrectly.

**Fix:** carry both computed absolute instants through move and resize proposals. Resolve changed manual wall-time fields explicitly; preserve computed instants when the gesture already determined them.

**Acceptance:** moves and both resize edges across spring gaps and autumn folds retain elapsed duration and the intended offsets.

### S05 · P1 · Unplanned actual requests can persist invalid identities

**Evidence:** domain probe reproduced. `domain.go:648-677,711-712`.

The new actual branch copies the whole client session and replaces only ID, plan, and state. It trusts `ruleId`, `occurrenceDate`, `exception`, and session `date`. A request with `ruleId: "missing-rule"` and an invalid occurrence date succeeds, then `Document.Validate()` rejects the stored record. The planned-session branch already clears client-supplied recurring identity. Exports containing these invalid actuals cannot pass import validation.

**Fix:** construct an unplanned actual session from allowed fields. Derive its date from the validated actual, clear recurring identity and attention/conflict metadata, and keep assignment validation at the boundary.

**Acceptance:** client-supplied recurring metadata is ignored or rejected; every accepted unplanned actual survives document validation and export/import round trip.

### S06 · P2 · Skipped actuals bypass date validation

**Evidence:** domain probe reproduced. `domain.go:275-290,694-706`.

`validateActual` returns immediately for `skipped`. A nonempty invalid date is accepted by `Apply`, while import validation rejects it. A valid but unrelated date can move the rendered skipped plan into the wrong column because the grid chooses `actual.date` first.

**Fix:** validate identity/date for every actual status. Treat a skipped planned session as belonging to its original scheduling date and plan; skipping must not accept arbitrary relocation fields.

**Acceptance:** malformed dates cannot be stored, a skipped session remains on its planned day, and valid skipped records round trip.

### S07 · P2 · Actual time attached to canceled sessions can be counted but hidden

**Evidence:** domain probe reproduced plus render trace. `domain.go:652-678,711-712,785-802`; `WeeklyScheduler.tsx:221`.

The actual branch allows an existing canceled session and preserves its canceled state. Weekly summaries count its actual hour, while the frontend excludes canceled sessions. This creates work that affects totals but cannot be edited in the grid.

**Fix:** define this transition explicitly. Allow explicit actual work to reactivate the visible record while retaining canceled plan history, or reject the operation and require a separate unplanned actual. Do not allow an invisible actual contribution.

**Acceptance:** canceled-session actual submissions either return a clear error or produce visible, editable work with matching totals.

### S08 · P2 · Move placement ignores where the block was grabbed

**Evidence:** source confirmed, real mouse probe confirms pointer position becomes the new start. `WeeklyScheduler.tsx:9,127-143,160-181`.

Drag state records pointer coordinates but no offset inside the session. Grabbing a session 30 minutes below its start and dropping at 10:00 places its start at 10:00 instead of 09:30. The session jumps relative to the hand.

**Fix:** record the initial pointer-to-start offset for session moves and apply it before snapping. New goal placement can anchor its start to the pointer. Compute the final target from the release event too, rather than relying only on the last move.

**Acceptance:** dragging from the middle preserves the grab position and duration, and the preview matches the final save.

### S09 · P2 · Ordinary scrolling and long drags are unsupported

**Evidence:** source confirmed. `WeeklyScheduler.tsx:127-150`; `weekly-scheduler.css:1`; `playwright.config.ts:28-31`.

There is no drag auto-scroll, Escape cancellation, active pointer ID tracking, or lost-capture cleanup. Goal and subgoal buttons use `touch-action:none`, so a touch gesture beginning on those large controls cannot scroll normally. The browser-test viewport was deliberately made tall enough to fit the full grid, which avoids testing the ordinary scrolling case.

**Fix:** add edge auto-scroll for an active drag; constrain events to one pointer; clear state on Escape, cancellation, lost capture, and view change. Preserve ordinary touch scrolling with an explicit handle or deliberate touch activation.

**Acceptance:** a 900px-tall desktop viewport can reach an offscreen target, touch users can scroll the requirement list, and canceled gestures send no write.

### S10 · P2 · Time-based UI becomes stale without Google

**Evidence:** source confirmed. `WeeklyScheduler.tsx:29-49,63-92,153-158`; `WeeklySchedulerGoogle.tsx:16-20`.

Local schedules load on mount/navigation/manual error refresh. Periodic refresh exists only when Google is connected. The worker may finalize actuals and rotate the revision while an open local page continues showing planned time, old coverage, and old capacity. The rendering also reads `Date.now()` without a clock state, so start/end transitions do not cause a render.

**Fix:** refresh the visible scheduler on a modest timer and visibility/focus changes regardless of Google. Coordinate it with S02 and preserve drafts. Provide a normal refresh action, clear resolved errors, and update editability at session boundaries.

**Acceptance:** an open local-only page reflects an elapsed session and updated totals without navigation; refresh cannot overwrite a newer save or draft.

### S11 · P2 · Some real intervals cannot be represented accurately on the grid

**Evidence:** source confirmed; full browser coverage still needed. `WeeklyScheduler.tsx:151-154,221`; `domain.go:724-729`.

The axis includes fixed 05:00 and 20:30 limits even when all configured days are shorter. Actuals outside the axis are clamped to its top or extend beyond its bottom, where the grid clips them. Every local session has a 30px minimum height, so two adjacent 15-minute reservations overlap visually. Folded-hour intervals can have equal wall-time endpoints but positive elapsed duration. A DST gap at a configured day boundary removes that date from `week.days`, shifting the seven-column layout.

**Fix:** separate exact interval geometry from usable labels/actions. Keep outside-hours actuals discoverable, expose elapsed duration/offsets for folds, and preserve a placeholder for an invalid configured day. Size the axis from the chosen day ranges and the visible records.

**Acceptance:** adjacent short sessions remain selectable; outside-hours actuals can be found and edited; DST days retain their date column and explain unavailable local times.

### S12 · P2 · Drag tests bypass important browser behavior

**Evidence:** test source confirmed and real mouse drag succeeded in the review. `tests/e2e/scheduler.spec.ts:44-55,90-96`; `tests/scheduler.browser.mjs:50-74`.

The Playwright suite synchronously dispatches synthetic PointerEvents and skips real pointer capture, hit testing, scrolling, click generation, and intermediate rendering. It asserts saved state but never checks a preview before release. The older browser script dispatches native DragEvents even though this component uses pointer handlers, and still queries the retired `.scheduler-editor` selector.

**Fix:** use real mouse input for user journeys and assert during held-pointer gestures. Retain helper tests for calculations. Replace or retire the obsolete standalone script. Add mobile/touch, ordinary viewport, cancellation, resize, and delayed-response cases.

**Acceptance:** the interaction suite fails when the destination preview is absent, and it exercises the actual browser input path.

## Integration risks requiring targeted tests

### R01 · P1 · Full export retries can starve later Google events

`internal/server/scheduler_worker.go:24-26,112-147` upserts every future accepted session on each batch, with a 55-second cycle deadline, random map iteration, and no durable per-event completion cursor. A sufficiently large schedule or repeated slow event can exhaust every batch before all jobs complete. Event mappings preserve identity but do not record which plan version was successfully synchronized.

Use per-event durable work or stored successful plan fingerprints, bounded batches, and retry state so completed writes do not consume each retry. Test more work than fits one cycle and failure partway through a batch. This is a code-level liveness risk, not a measured live-Google outage.

### R02 · P2 · Google health can remain stale in an open page

`WeeklySchedulerGoogle.tsx:11-20` reads status on mount and after user operations or a five-minute connected refresh. Worker-discovered export/reconnect failures can remain invisible for that interval. Once `reconnectRequired` is true, `connected` is still true and periodic refresh attempts continue. Calendar selection failure retains its old revision and does not refetch current selection/status.

Poll visible connection health with scheduler refresh; stop background availability attempts that require reconnection. On selection conflicts, reload the current revision while retaining proposed selections for review. Test worker-reported revocation and concurrent selections.

### R03 · P2 · Each manual availability refresh fetches Google twice

`WeeklySchedulerGoogle.tsx:18,21` posts `google/refresh`, then calls `refreshWeek()`. The week GET calls `schedulerAvailability`, which always delegates to a fresh FreeBusy check in `scheduler_google.go:390-391`. Persisted freshness/range fields are not used to avoid this second fetch.

Return refreshed week data from one operation or use a range-aware fresh cache for reads while retaining fresh checks for planning writes. Test request counts, range coverage, cache expiry, and failure behavior.

### R04 · P2 · Far-future recurrence conflicts exceed the checked horizon

`internal/database/scheduler.go:144-148,217-234` generates and checks through requested week plus 62 days. An unbounded rule can be accepted even if it collides with an already persisted one-off later than that horizon. Later viewing/generation exposes the conflict. Include known reservations and dated day overrides outside the horizon in recurrence validation, or explicitly define and display the checked horizon. Test a one-off conflict several months after rule creation. Server conflicts already correctly reject overlaps inside the generated window.

## Improvements after correctness fixes

- Split the large JSX lines into grid, block, editor, and requirement-list components. Extract shared placement calculation so preview, save, and resize use one proposal. Keep API and persistence ownership intact.
- Group sessions and busy intervals by day before rendering. Every pointer move currently updates state in the full scheduler, which repeatedly filters sessions and converts busy times for seven columns. Measure with a large week before adding memoization.
- Make warnings identify and link to affected goals, dates, and sessions. The API currently returns plain strings and the UI renders noninteractive list items, although the design calls for actionable warnings.
- Make quick-add start/duration use the selected configured day and remaining room. Sidebar quick-add defaults to 09:00 even if that hour is unavailable; an assignment drop can suggest up to four hours without considering room at the target.
- Use deterministic initial SSR markup for the week heading. It depends on server/browser date, timezone, and locale. The frozen browser-clock probe produced a hydration mismatch; actual deployment timezone/locale mismatch remains to be measured.
- Improve network error parsing so HTML/empty responses and auth expiry preserve the draft with a useful message instead of a JSON parse error.

## Existing strengths

The server checks reservations, goal eligibility, actual overlap, and revisions. Database writes and generic goal/subgoal changes share advisory locking and reconciliation. Recurring occurrence identities are stable; date exceptions, historical plans, closed-week requirements, and actual replacements have domain tests. OAuth credentials are encrypted and excluded from export; callback state is bound to the authenticated session. Google mappings and outbox generations address duplicate identities and concurrent local writes. Preserve these properties through the repair work.

Implementation order and concrete checks are in [the repair plan](../superpowers/plans/2026-10-02-weekly-scheduler-repair.md).

## Repair regression map and provider limits

The repair PRs preserve the existing domain and database gates and add the following regressions. Names below refer to checked-in tests; browser cases use real pointer input unless a controlled response race is the behavior under test.

| Issue | Regression evidence |
| --- | --- |
| S01 | Browser `held assignment shows its snapped destination before the matching interval is saved` and `held move and both resize edges preview the interval that the API stores`. |
| S02 | Browser `request coordination keeps Jan 14 visible when a delayed Jan 7 save completes`, `request coordination retains failed proposal ownership after navigation and guards rapid submit`, and `failed drag after ordinary navigation opens its original week draft and saves with original settings`. |
| S03 | Client `post-midnight actual proposal keeps the owning column date through submission`; domain `TestActualDateOvernightPrecedenceAcrossWeekAndOverrides`; server `TestSchedulerActualRouteDerivesOvernightDates`. |
| S04 | Client `move and resize retain both computed fold instants and elapsed duration` and `unchanged editor times retain the original instants across a repeated hour`; browser `fold actual shows elapsed minutes and both UTC offsets`. |
| S05 | Domain `TestUnplannedActualIgnoresRecurringIdentity`; server `TestSchedulerActualRouteStoresCanonicalUnplannedSession`. |
| S06 | Domain `TestSkippedActualUsesOriginalSchedulingDate`; server malformed actual date rejection in `TestSchedulerActualRouteStoresCanonicalUnplannedSession`. |
| S07 | Domain `TestCanceledPlanCannotHideActualWork`, `TestCancelPreservesActualOnNonrecurringPlan`, `TestFutureSessionEditCannotEraseRecordedActual`, and `TestFutureRuleEditPreservesRecordedActual`; database `TestSchedulerAcceptedActualsSurviveExportImport`. |
| S08 | Client `moving a block subtracts the 30-minute grab offset before snapping`; browser held move and resize case. |
| S09 | Browser `viewport edge scrolling reaches a target below a 1280 by 900 screen`, `mobile touch scrolls the requirement list and drags only from its handle`, and `tablet touch can start a visible drag handle`. |
| S10 | Browser `visible local refresh updates worker actuals, clock editability and reconnect health without availability retries`, `local refresh keeps drafts, clears resolved errors and pauses while the page is hidden`, and `an open plan draft locks its original plan at the start boundary and offers actual recording`; database `TestSchedulerIdleTickDoesNotRotateRevisionOrDirtyOutbox`. |
| S11 | Browser `geometry keeps adjacent short blocks selectable and outside-hours actuals discoverable`, `invalid dated column remains visible and warning opens its affected editable session`, and `fold actual shows elapsed minutes and both UTC offsets`; client `display geometry includes early and late actuals and elapsed fold duration`. |
| S12 | The checked-in scheduler browser suite covers pointer capture, both held resize edges, cancellation, actual hit testing, viewport scrolling, touch, and delayed responses. `tests/verify-scheduler-e2e.test.sh` checks disposable database/port cleanup safety. |
| R01 | Server `TestSchedulerGoogleBoundedRetryAdvances`, `TestSchedulerGooglePersistentFirstFailureCannotStarveLaterWork`, `TestSchedulerGooglePeriodicCursorRestoresEntireRemoteSet`, `TestSchedulerGooglePeriodicFailureQueuesRetryAndAdvancesCursor`, `TestSchedulerGoogleChangedPlanDuringBatchKeepsNewGeneration`, `TestSchedulerGoogleCancellationAndReconnectPreserveIdentities`, `TestSchedulerGoogleRemoteTombstoneReplacesIdentityOnce`, `TestSchedulerGoogleWorkerAndDisconnectShareAuthorityLock`, and `TestSchedulerGoogleCanceledSessionDeletesOnlyMappedEvent`; database `TestSchedulerGoogleProgressCannotCompleteNewDesiredWork` and `TestSchedulerGoogleReconciliationCursorScopedToAuthority`. |
| R02 | Browser visible health refresh case, `Google selection conflict retains choices and reloads current revision for explicit retry`, and `obsolete Google health errors cannot replace a newer successful status`; server `TestSchedulerGoogleRevocationPersistsAfterConcurrentHealthWrite`. |
| R03 | Server `TestSchedulerGoogleReadCacheAndFreshPlanning`, `TestSchedulerGoogleManualRefreshOneQueryAndLocalSaveSurvivesExportFailure`, `TestSchedulerGoogleCacheExpiryRangeSelectionAndFailure`, `TestSchedulerGoogleDisconnectDuringAvailabilityFetch`, and `TestSchedulerGoogleOverlappingRefreshCoalescesAcrossServers`. |
| R04 | Database `TestRuleChecksKnownReservationBeyondGenerationWindow`, `TestRuleChecksDatedBoundaryBeyondGenerationWindow`, `TestRuleChecksOvernightReservationOnSchedulingDate`, `TestRuleChecksMatchingOccurrenceInsideDistantBusySpan`, `TestRuleChecksKnownFutureRuleWhenProbeOrdersItAfterNewRule`, and `TestRuleChecksPreservedFutureDateException`. |

The controlled Google tests run the production HTTP client and persistence against a local provider and an explicit disposable PostgreSQL database. The 47-event regression exceeds the 20-event batch limit, fails after five successful writes, and verifies convergence without repeating those writes. Separate periodic passes restore edited and deleted events across the full set; tombstones that reject a reused ID get a persisted replacement. Revocation stays visible despite a concurrent health write. Manual refresh plus week read makes one FreeBusy request, overlapping refreshes coalesce across independent server instances, and each planning write checks fresh availability.

No live Google credentials were available. OAuth consent behavior, production quota/latency limits, and a live connected-account end-to-end check remain unverified. This is a controlled liveness regression, not a measured Google production outage. Large sets require multiple bounded periodic reconciliation cycles. Local saves survive outbound failures; failed fresh availability checks keep planning proposals as drafts.

Verification before independent review: all 197 Go top-level cases passed with zero skips against a disposable `career_scheduler_task8` database on Docker `migration-postgres-1`, localhost5433. Google concurrency tests passed under the race detector. All 148 npm tests, including 18 scheduler client tests, passed; TypeScript passed. The authenticated disposable-app Playwright scheduler suite passed all 35 cases, and runner cleanup/port safety checks passed. No live Google credentials were used.

Independent scoped review of PR89 at `94b2d2d1a7418c52a11ce91484e7fee4a542cb5c` reproduced a persistent earliest-event failure starving healthy later events despite those tests. The additional persistent-failure regression failed before the fix. Durable attempted-event rotation and continuation after recoverable errors now cover that boundary; the follow-up exact-head checks and whole-feature verdict are pending.
