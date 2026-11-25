# Home

**Current code:** src/pages/index.astro → HomeHub.tsx
**Reference:** desktop `reference/desktop/ARC-Home.html` · mobile `reference/mobile/M-Home.html`

## Purpose

Today at a glance: today's goal sessions as a checklist, missed sessions, active goals, and the Leetgrinder streak and review pick. Sections come from enabled modules (Schedule → sessions, Timeline → goals, Leetgrinder → streak card).

## Data

`GET /api/scheduler/week` (today's goal sessions, earlier unchecked), `POST /api/scheduler/mutate` (check / uncheck records the planned time as done, unchanged behavior), `GET /api/goals` (active goals and step progress), `GET /api/leetgrinder/today` (goal met, streak, review pick, due count).

## Desktop

- Header: mono label "Week 48 · America/Chicago", large date `24.11.26` (Inter Tight 200, 128px), weekday, handwritten accent note "1 of 3 done — next in 25 min".
- Measurement line for the day window ("5:00 AM — 15.5 hours — 8:30 PM").
- Left (680px): GOAL SESSIONS rows: checkbox, start time, duration drawn to scale (60px per hour, measurement line with label), title + sub-label, status. Next session circled in accent. Missed sessions row with accent icon. ACTIVE GOALS: name, hatched progress on a measurement line, `3 / 6` or LATE.
- Right: LEETGRINDER: big streak number, tick scale 0–record, New / Review / Due counts; review card (Box) with Open link.
- Faint full-height guide lines are decorative, optional.

## Mobile

Single column: date (70px), day-window line, accent note, sessions (checkbox + time + duration line + title), missed, goals, streak row, review card.

## Acceptance criteria

- [ ] Matches the reference in all four themes at 1280×800 and 390×844 (tokens only, no raw colors).
- [ ] Uses the shared primitives from `design-system.md`; times are 12-hour.
- [ ] Hidden (and 404) when its module is off.
- [ ] Checking a session records it exactly as today; the done count and accent note update.
- [ ] Next session = first unchecked session after now; its ring and note move as time passes.
- [ ] Sections disappear when their module is off.
- [ ] Empty states: "Nothing scheduled today. Plan in the scheduler." with a link.
