# Weekly scheduler

Status: draft for user review, 2026-09-28. Product behavior and defaults were agreed in conversation. Details explicitly marked as proposed below complete the design for review.

## Purpose

Turn timeline goals into a realistic weekly schedule around work and other commitments. Users place goal work into available time, see unmet requirements, and record what they actually did. As goals change along the timeline, the scheduler updates requirements and identifies new work needing time.

Success means a user can build a recurring routine, adjust one week without changing the routine, and compare planned time with actual time. Every work session belongs to a goal, optionally through a subgoal. Fixed commitments and Google busy intervals reserve time without earning goal hours.

## Agreed behavior

| Area | Decision |
| --- | --- |
| Calendar | A dedicated Weekly Scheduler page with seven day columns and a goal list. |
| Scheduling list | Show goals relevant to the displayed week and their unfinished subgoals. Either can be dragged onto the calendar. |
| Requirement | Use the goal's existing expected hours per day. Changes update the weekly requirement. |
| Applicable days | Default to all seven days, with days individually selectable per goal. |
| Weekly flexibility | Time on any day can fulfill the weekly requirement. Extra time on one day covers another day's shortfall. |
| Partial weeks | Count only selected days within the goal's inclusive timeline dates. |
| Subgoals | Their time counts toward the parent goal, without counting the same session twice. |
| Recurrence | A recurring template with overrides for individual dates. |
| Fixed commitments | Named recurring blocks, such as work, commuting, and meals, with date-specific overrides. |
| Day boundaries | One default start and end time, weekday overrides, and date-specific overrides. No fixed 16-hour limit. |
| Past days | Preserve the original plan and allow users to edit actual time, skip sessions, or add unplanned work. |
| Automatic actuals | After a session ends, assume it happened as planned unless the user edits or skips it. |
| Overlaps | Reject conflicting drops and highlight the conflicting block. |
| Goal changes | Remove future blocks for ended goals. Retain blocks when required hours change and update shortfall or excess indicators. |
| New goals | Leave them unscheduled and show unmet-requirement warnings. |
| Google Calendar | Send scheduled blocks to Google and read existing Google busy times. Google edits do not update the app's schedule. |
| Defaults | Monday week start, 15-minute drag and resize increments, and 60-minute new blocks. |
| Completion | Recording time never automatically completes a goal or subgoal. |

## Existing app integration

The app serves a static Astro frontend with React components through a Go server backed by PostgreSQL.

- `src/lib/timeline.ts` defines `Goal`, including `dailyHours`, inclusive `startDate` and `endDate`, status, dependencies, and steps.
- `src/lib/mini-goals.ts` defines subgoals as existing goal steps. They have an ID, title, and completion flag, without independent dates or hour requirements.
- `src/components/GoalSchedule.tsx` edits timeline dates and daily estimates. It is a goal date editor, not a weekly time-slot calendar.
- `internal/database/model.go` validates goals. Missing `dailyHours` differs from zero.
- `internal/database/goal_steps.go` supports moving a subgoal between parent goals.
- `internal/server/routes.go` exposes authenticated goal mutations and uses revision checks for concurrent edits.

Reuse these goal and subgoal identities. Add selected weekdays to the goal model and its editor. Existing goals default to all days. Update the timeline's workload calculations and explanatory text so they respect selected weekdays and agree with scheduler requirements.

The scheduler uses the current workspace and authentication model. This feature does not introduce a separate multiuser account system.

## Calendar experience

The page has previous week, next week, and Today controls, a date range, settings, and Google connection status. The main area contains a goal list beside the weekly grid.

Each goal row shows its color, title, required hours, actual hours, remaining scheduled hours, and uncovered hours. Expand the goal to see unfinished subgoals. Fully covered goals remain available for additional scheduling. Completed subgoals remain visible on historical sessions.

Proposed subgoal indicator: show unfinished subgoals without sessions as **Unscheduled**. Subgoals do not have independent hour requirements, so that indicator does not add hours to the parent's shortfall.

Dragging a goal or subgoal creates a 60-minute session. Users can resize, move, edit, or remove it. The block editor provides equivalent controls for keyboard and touch use. If 60 minutes does not fit, reject the drop and offer duration editing rather than silently shortening the block.

The grid uses a shared time axis covering the displayed days' configured ranges. Shade unavailable hours for each day. Fixed commitments and external busy intervals are visually distinct from goal work. Goal colors must have accompanying text labels.

At narrow widths, show one selected day with week navigation and the same goal list and editor. A desktop user sees seven columns.

Proposed recurrence controls: editing a generated session offers **This date** and **This and future occurrences**. The latter updates the recurring rule from that date. Creating a session defaults to one occurrence, with an option to repeat weekly. Either goal or subgoal sessions can repeat until their parent goal ends or the subgoal completes.

## Requirements and weekly accounting

For goal `g` and displayed week `w`:

```text
applicableDays = selected weekdays in w within g's inclusive start/end dates
requiredHours = g.dailyHours × count(applicableDays)
coveredHours = actualHours + remainingScheduledHours
uncoveredHours = max(0, requiredHours - coveredHours)
excessHours = max(0, coveredHours - requiredHours)
```

Selected weekdays determine the amount owed, not the only days work is allowed. Sessions can be placed on any day inside the goal's active date range.

A session contributes once to its parent goal. A subgoal breakdown is a view of that same time, not an additional contribution. Fixed commitments and Google busy intervals contribute no goal time. Extra hours do not carry into another week.

Examples:

| Goal configuration | Weekly requirement | Example outcome |
| --- | --- | --- |
| 1 hour/day, all days | 7 hours | 3 hours Monday and 4 hours Saturday cover the week. |
| 1 hour/day, Monday/Wednesday/Friday | 3 hours | 3 hours Tuesday cover the week. |
| Starts Thursday, 1 hour/day, all days | 4 hours | Thursday through Sunday contribute to the requirement. |
| Ends Wednesday, 2 hours/day, weekdays | 6 hours | Monday through Wednesday contribute. |
| 7 required, 2 actual, 4 still scheduled | 7 hours | Show 1 hour uncovered. |

Proposed calculation details:

- An unset estimate shows **Hours not set** and a warning. It never silently means zero. A zero estimate creates no required hours.
- Selecting no weekdays is allowed and creates a zero weekly requirement while retaining existing sessions.
- Calculate durations in seconds and requirements without snapping them to calendar increments. Round only display values. Manual editing supports minute precision.
- An unfinished session contributes its full planned duration as scheduled time. Once its planned end passes, its actual record replaces that contribution. Never add planned and actual durations for the same session.
- An explicit actual record overrides the automatic assumption. A skipped session contributes zero and reopens the shortfall.
- Updating daily hours or selected weekdays recalculates the entire current week and future weeks. Closed weeks retain their saved requirement, while actual-time corrections can still change their totals.
- Work added to historical dates uses the goal assignment saved for that session. Recording work outside a goal's historical timeline is allowed as an actual-time correction, with an explanatory indicator. It does not expand the goal's requirement.

## Day boundaries, dates, and time zones

The user's day is a configurable interval, such as 05:00–20:30 or 09:00–00:00. Settings resolve in this order: date override, weekday override, default.

Proposed time rules:

- Initialize the scheduler's IANA time zone from the browser and make it editable. Store it independently of browser changes.
- Initialize the default day to 05:00–20:30, matching the requested example, and allow changes during setup or later.
- Allow day intervals longer or shorter than 16 hours, up to 24 hours. Represent next-day endings explicitly. Equal start and end times require an explicit full-day choice.
- An overnight column belongs to the date on which its configured day begins. Its post-midnight sessions count toward that scheduling date and week.
- Day intervals for adjacent dates cannot overlap. Sessions must fit fully inside one configured day when planning.
- Local recurrence follows wall-clock time. Dated sessions store absolute instants and their scheduling date. Preserve past instants when time-zone settings change.
- For daylight-saving gaps, flag an occurrence that has no valid local time instead of silently moving it. For repeated local times, use the earlier occurrence and show its UTC offset in the editor. Count elapsed duration.
- Changing future day boundaries or time zone regenerates future recurrence. Existing dated sessions that no longer fit require attention rather than disappearing.

## Recurring templates and dated sessions

A recurring rule contains a weekday, local start, duration, effective dates, and either a goal assignment or a named fixed commitment. Goal assignments optionally identify a subgoal.

The dated schedule consists of generated occurrences, exceptions to those occurrences, and one-off sessions. Each generated occurrence has a stable identity based on its rule and scheduling date. Reopening a week or retrying generation must not duplicate sessions.

Proposed lifecycle rules:

- Templates apply from their effective date and never invent sessions in earlier weeks.
- A date exception can move, resize, replace, or cancel an occurrence. It does not change the recurring rule.
- Editing a rule preserves explicit date exceptions, then revalidates them against goal eligibility, day boundaries, and conflicts.
- A background process persists upcoming occurrences and finalizes elapsed sessions, even when the page is closed. On restart, it catches up from persisted rules and effective dates.
- Changes to a future plan can update it. Once a session starts, preserve its plan as the baseline and direct changes to the actual record.
- When a recurring occurrence collides with another reservation, retain it visibly as needing attention. It is not an accepted reservation, contributes no scheduled coverage, and cannot automatically earn actual time.
- A template change that would create conflicts reports the affected dates. It never moves neighboring sessions automatically.

## Planned and actual time

Every goal session retains a planned record when one exists. Its actual record is one of: assumed as planned, explicitly recorded, or skipped. Unplanned actual work has no original planned record.

After a valid session's planned end, automatically record its planned interval as assumed actual time. Show that status so the user can distinguish an assumption from a manual entry. Actual-time edits preserve the original plan and its goal assignment.

Users can change actual start and end times, mark a session skipped, restore the as-planned value, or add unplanned goal or subgoal work. Manual entries replace assumptions instead of adding a second contribution.

Proposed historical conflict handling: actual goal sessions cannot overlap each other. Historical fixed commitments, unavailable hours, or Google busy times do not prevent recording what actually happened. Such a difference can be displayed without rewriting the original plan. Moving an actual interval into another scheduling day moves its contribution to that day's week.

## Timeline and subgoal changes

Goal dates define eligibility for a displayed week. A goal with status `planned` can appear in a future week when its dates overlap that week. Current status alone must not hide future goals.

For automatic changes, **future sessions** means sessions that have not started. Preserve the plan for an in-progress session. An ended goal remains available when viewing or correcting its historical work.

Proposed status handling: `done` and `dropped` stop future generation immediately. Save the date of this transition for partial-week accounting. Keep historical sessions and already elapsed requirements. Reopening the goal resumes eligibility from the reopening date within its timeline dates, without rewriting history.

| Change | Result |
| --- | --- |
| Goal reaches its end date | Stop recurrence after that date and cancel future sessions outside its range, including overrides. |
| Another goal becomes relevant | Add it and its unfinished subgoals to the scheduling list. Show the new requirement as uncovered until scheduled. |
| Daily hours or selected days change | Recalculate current and future requirements. Keep valid sessions and report shortfall or excess. |
| Start or end date changes | Recalculate eligibility and partial-week requirements. Remove future sessions outside the new range. Do not shift sessions automatically. |
| Subgoal completes | Stop future sessions for that subgoal. Keep historical records and the parent goal's independent requirement. |
| Subgoal moves to another goal | Assign its future rules and sessions to the new parent, then validate the new dates. Past attribution stays with the original parent. |
| Goal or subgoal is deleted | Cancel future affected work and keep historical IDs and title snapshots. Do not cascade-delete time history. |
| Goal title or color changes | Update future blocks. Historical snapshots preserve the original labels. |

Proposed dependency handling: show the existing dependency warning on affected goals. Dependencies do not automatically reschedule work or remove hour requirements. This feature does not introduce a second dependency engine.

## Conflicts and warnings

Planning rejects overlapping goal sessions, fixed commitments, and known external busy intervals. Adjacent blocks are valid. Validate create, move, resize, and repeat operations on the server as well as in the browser.

An external event can arrive after local scheduling. Preserve the local session, mark the conflict, and exclude its planned duration from coverage until resolved. Do not automatically move or delete either event. A conflict discovered after a session has ended does not erase its historical actual time.

Warnings identify:

- A goal's uncovered weekly hours, including newly active goals.
- Missing daily estimates.
- Conflicting or out-of-bounds sessions that need placement.
- More uncovered hours than the remaining unreserved time in the week.
- Unavailable or stale Google availability data and failed outbound sync.

Show excess hours as an informational amount rather than an error. Warnings link to the affected goal or date. Capacity counts each blocked interval once, even if external calendars overlap. Exclude elapsed time when assessing remaining capacity in the current week.

## Google Calendar connection

Sync is optional. Local scheduling and actual-time recording work without a Google connection.

Proposed integration contract:

- Connect one Google account for the current workspace using server-side OAuth. Users select calendars that provide busy times.
- Create a dedicated **Career Weekly Scheduler** destination calendar. Exclude it from imported busy calendars so exported sessions cannot conflict with themselves.
- Export accepted goal sessions and fixed commitments as dated events. Include the goal and subgoal title where applicable and a link to the scheduler.
- Export the current week and the next eight weeks. Extend that window daily. Opening another future week also schedules it for export. Initial connection does not backfill historical weeks.
- Update or delete only events created by this integration. A removed future local session removes its exported event. Never modify unrelated Google events.
- Export the plan. Actual-time corrections stay in the career app and do not rewrite past Google events.
- Changes made directly to exported Google events do not update local sessions. Reconciliation restores the app's plan, including recreating deleted exported events. Explain this in connection settings.
- Fetch busy intervals when opening a week, on manual refresh, and every five minutes while the scheduler is visible. Show when availability was last refreshed.
- Use a fresh availability check before accepting new or moved reservations when connected. If it fails, keep the proposed edit as a draft and show a retry option. Users can explicitly disconnect to continue without external conflict checking.
- Queue outbound writes after the local transaction commits. Retries use stable event mappings and cannot create duplicates. Local saves remain valid if Google is temporarily unavailable.
- Reconciliation detects conflicts introduced after the availability check. No cross-system transaction can eliminate that race.
- Disconnecting stops synchronization and removes stored credentials. Leave exported events in Google, explain that outcome, and retain event mappings for reconnection to the same account.

Google provides a FreeBusy endpoint for availability and an authorization scope for managing app-created secondary calendars. Use calendar-list read access for the selection UI, availability access for inputs, and app-created calendar access for output. Confirm the exact consent scope combination during implementation against the [Google Calendar authorization reference](https://developers.google.com/workspace/calendar/api/auth) and [FreeBusy reference](https://developers.google.com/workspace/calendar/api/v3/reference/freebusy/query).

Keep tokens encrypted on the server, bind OAuth callbacks to the signed-in session, and never expose refresh tokens in browser storage or exports. When authorization expires or is revoked, show a reconnect state instead of retrying indefinitely.

## Proposed technical design

Use a dedicated scheduler domain in the existing Go service and PostgreSQL database. A React scheduler component on an Astro page consumes the API. The server owns requirement calculations, occurrence generation, conflict validation, and actual-time accounting. The client previews drag operations and renders authoritative results after saving.

| Record | Responsibility |
| --- | --- |
| Scheduler settings | Time zone, default day interval, weekday intervals, and dated overrides. |
| Goal scheduling policy | Selected weekdays and effective history needed to preserve past requirements. The existing goal remains the source of daily hours. |
| Recurring rule | Effective date range, weekday, local interval, and goal/subgoal or fixed commitment assignment. |
| Dated session | Stable occurrence identity, original plan, assignment snapshot, revision, and accepted/canceled/needs-attention state. |
| Date exception | One occurrence's edited placement or cancellation, separate from the recurring rule. |
| Actual record | Assumed, explicit, or skipped time for a session, or unplanned work with a goal assignment. |
| Weekly requirement snapshot | Closed-week requirement and the policy values used to calculate it. |
| Google connection | Encrypted credentials, selected input calendars, output calendar, and health. |
| Google event mapping and outbox | Dated-session-to-event identity and durable pending remote changes. |
| Busy cache | External reserved intervals with calendar identity, query range, and freshness. |

These are logical records, not a fixed table count. Choose storage details in the implementation plan. Historical records retain identity snapshots without requiring a live goal or subgoal row. Extend backup/import support for scheduler data, excluding credentials and transient sync jobs.

The authenticated API must support week reads with summaries and warnings, settings changes, recurring-rule changes, dated-session changes, actual-time recording, and Google connection and refresh operations. Mutations carry revisions. Return a conflict with current state when an edit is stale, preserving the user's draft. Serialize reservation checks and writes per workspace so simultaneous edits cannot double-book time.

Goal mutations and subgoal moves trigger scheduler reconciliation through a shared domain operation. This applies to every write path, including existing generic entry and MCP writes, rather than only changes made through the new page. Imports invalidate affected future schedule projections. Keep external Google calls outside database transactions and use the durable outbox for follow-up writes.

Proposed module ownership:

- `src/pages/weekly-scheduler.astro`: page and existing application layout.
- `src/components/WeeklyScheduler.tsx`: weekly navigation, grid, and interactions, with focused child components for the list and editors.
- `internal/scheduler/`: requirement calculation, recurrence, eligibility, and conflict rules.
- `internal/database/`: persistence, revisions, history, and synchronization jobs.
- `internal/server/`: authenticated scheduler endpoints and Google OAuth callback handling.

## Alternatives considered

A dedicated page with recurring rules and date exceptions fits the agreed behavior and preserves the distinction between a routine and a particular week.

Embedding time slots directly into the existing timeline would combine long-term goal dates with hour-level planning in one interaction. A link between the two pages gives each view a clearer purpose.

Copying independent schedules each week would simplify recurrence storage but would not reliably carry template edits or goal endings into future weeks. Google Calendar as the primary data store would make actual-time accounting and local scheduling depend on an optional connection.

## Scope boundaries

The first release includes the weekly scheduler, recurrence, fixed commitments, warnings, actual-time corrections, timeline integration, and optional Google export with busy-time input.

It excludes automatic placement or rearrangement of work, two-way Google editing, automatic goal completion, independent subgoal hour budgets, multiple Google accounts, and a new task system separate from existing subgoals.

## Acceptance criteria

1. A user can configure 05:00–20:30 and another configuration of 09:00–00:00 without a fixed 16-hour assumption.
2. Weekday and date overrides affect the correct columns, and overnight intervals cannot overlap the following day.
3. A seven-hour requirement can be covered by three hours on Monday and four on Saturday.
4. Selecting three weekdays at one hour per day creates a three-hour requirement, even when the work is scheduled Tuesday.
5. A Thursday start creates a four-hour requirement at one hour per day with all days selected.
6. General goal sessions and subgoal sessions contribute once to the same parent total.
7. Changing daily hours updates current and future requirements without moving existing valid sessions or rewriting closed-week requirements.
8. A date override affects one occurrence, while an effective template edit updates subsequent occurrences without changing historical plans.
9. Ending a goal removes future sessions and Google exports. A newly relevant goal appears with an uncovered-hours warning.
10. A subgoal move updates future attribution while past records retain the original parent. Deleting a goal preserves time history.
11. A completed session automatically counts as actual time while the page is closed. Editing its duration replaces that assumption and retains the plan.
12. Skipping a previously assumed session contributes zero. Adding unplanned historical work updates the correct week.
13. Conflicting drag, resize, repeat, and concurrent API operations cannot create accepted overlapping reservations.
14. New Google busy intervals flag existing conflicts. The output calendar never conflicts with its own exported events.
15. A synchronization retry creates no duplicate events. Deleting a local future session deletes only its mapped Google event.
16. An expired Google connection shows a reconnect state. Unsaved edits remain available, and local data is preserved.
17. DST gaps, repeated times, overnight week boundaries, and server restarts do not duplicate or silently lose time.
18. Keyboard and narrow-screen users can create, move, resize, and edit sessions without drag-and-drop.
19. A missing estimate produces an explicit warning. A zero estimate and no selected days produce zero required hours.
20. Goal changes through timeline, generic entry, MCP, and import paths cannot leave the scheduler permanently out of date.

Verification will combine domain tests for accounting and recurrence, database/API tests for history and concurrent reservations, and browser tests of the actual calendar interactions. Google integration tests use controlled API responses for retry and outage cases, followed by a connected-account end-to-end check during implementation when credentials are available.
