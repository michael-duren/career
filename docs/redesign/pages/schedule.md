# Weekly scheduler

**Current code:** src/pages/weekly-scheduler.astro → WeeklyScheduler*.tsx
**Reference:** desktop `reference/desktop/ARC-Scheduler.html` · mobile `reference/mobile/M-Schedule.html`

## Purpose

Plan goal work across the week, see recurring routines and Google busy time, accept plans.

## Data

`GET /api/scheduler/week`, `POST /api/scheduler/mutate`, `GET /api/scheduler/goal-hours`, `/api/scheduler/google/*`. Keep drag-and-drop and rules behavior.

## Desktop

- Header: "Week 48" label, "23 — 29 November", prev/next, RULES, GOOGLE SYNC, primary ACCEPT PLAN.
- Week grid (Mon–Sun columns, hours 5 AM–8:30 PM, 12-hour labels). Today's column header filled ink.
- Blocks: goal sessions = Box with a pattern strip (diagonal/crosshatch/dots per goal); routines = faint outline; Google busy = dotted outline; done = 40% + struck; skipped = accent outline; canceled = dashed; conflict = accent outline.
- Now line in accent across today's column with handwritten "now 5:20 PM".
- Right column: GOAL HOURS table (goal with pattern swatch, Req, Plan, Δ with shortfalls in accent), capacity remaining, WARNINGS list in accent.

## Mobile

Day strip (7 buttons, today underlined in accent, selected filled) above a single-day timeline of blocks; weekly goal-hours list and warnings below.

## Notes

- Plans are accepted the moment they are saved (there is no pending proposal to accept), so the primary header button is ADD SESSION: a draft for the first goal in the first free slot of the selected day.
- RULES opens the day-window settings (default day, weekday and date overrides). Weekly repeating sessions are still made from the session editor ("Repeat weekly").
- GOOGLE SYNC opens the Google Calendar panel; its connection state also shows in the line under the title.
- The grid stays at 1px per minute (5 AM to 8:30 PM by default), so it is taller than the 800px reference frame; drag and resize snap to 15 minutes.
- Goal patterns follow the order of the week's goals, which the API sorts by title, so a goal keeps its pattern while the set of goals stays the same. Goals after the third, and historical goals, get an outline.
- Plan is recorded plus still-scheduled hours; Δ is Plan − Req. The row also shows hours done and still scheduled; subgoals, a missing estimate and dependencies sit under the row.
- Canceled sessions are not drawn (unchanged behavior), so the dashed "canceled" style is not used.
- Below 1024px goal hours and warnings move under the grid; below 768px the day strip picks the one day shown.

## Acceptance criteria

- [ ] Matches the reference in all four themes at 1280×800 and 390×844 (tokens only, no raw colors).
- [ ] Uses the shared primitives from `design-system.md`; times are 12-hour.
- [ ] Hidden (and 404) when its module is off.
- [ ] All current scheduler interactions (drag, resize, rules, accept, Google) work.
- [ ] Goal patterns are stable per goal and shown in the legend/hours table.
- [ ] Times are 12-hour everywhere, including block labels.
