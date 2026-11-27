# Timeline

**Current code:** src/pages/timeline.astro → GoalTimeline.tsx
**Reference:** desktop `reference/desktop/ARC-Timeline.html` · mobile `reference/mobile/M-Timeline.html`

## Purpose

All goals on a month axis with their dates, status and step progress; click a goal to see its steps. The axis fits the goals' date range by default (Fit), with 6M / 1Y / 2Y / 5Y / 10Y tabs and previous / Today / next.

## Data

`GET /api/goals`, `POST /api/goals/steps/move` (keep existing editing, reordering and moving steps; restyle only).

## Desktop

- Header: label (date range), title "Goal timeline", counts Active / Planned / Done / Late (late in accent).
- Month axis with faint month guides. Each goal row: name (left 236px), span drawn as a measurement line from start to end, hatched fill for step progress above it, tag after the span (`3/6`, done, late).
- Planned = dashed line, late = accent line, done = 45% opacity.
- Today = dash-dot accent vertical line with handwritten "today".
- Selected goal panel at the bottom: name, dates, numbered steps in two columns (done ones struck/dimmed).
- Click selects a goal; double-click (or Edit goal in the panel) opens the editor. Drag a span to move it, drag its ends to resize. Dropped goals draw dashed at 45% and are never late. An "All goals" list below reaches goals outside the visible range.

## Mobile

List of goals; each row shows name + status tag above a span on the same compressed axis (single-letter months). Today line runs through the list. Tapping a goal expands its steps inline.

## Acceptance criteria

- [ ] Matches the reference in all four themes at 1280×800 and 390×844 (tokens only, no raw colors).
- [ ] Uses the shared primitives from `design-system.md`; times are 12-hour.
- [ ] Hidden (and 404) when its module is off.
- [ ] Every existing timeline action (create, edit, reorder, move steps, dependencies) still works, restyled.
- [ ] Late = not done and end date before today.
- [ ] Axis covers the goals' date range (not hard-coded to Aug–Jul).
