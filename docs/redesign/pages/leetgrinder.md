# Leetgrinder

**Current code:** Go templ: internal/leetgrinder/*.templ (views, todos, quick, analysis, …), served at /leetgrinder
**Reference:** desktop `reference/desktop/ARC-Leetgrinder.html` · mobile `reference/mobile/M-Leetgrinder.html`

## Purpose

Daily practice: goal and streak, today's review pick, trends, weakest topics.

## Data

Server-rendered templ views and `/api/leetgrinder/*`. Restyle the templates with the shared CSS layer (tokens + primitives); do not rewrite them in React.

## Desktop

- Sub-tabs (TODAY, REVIEWS, SETS, LOG) at the top of the page content.
- Left: big streak "02", "DAY STREAK · Record 29 · goal met ✓", tick scale 0–record, TYPE/DONE/GOAL table (New, Review, Quick/bonus, Still due), review card (Box) with recall, topics, accent tip, OPEN and LOG ATTEMPT.
- Middle: SOLVED PER WEEK bars (hatched; current week dashed) with 10/20 dashed guides and a handwritten trend note; Independent / Complexity ok / Avg medium stats; DIFFICULTY bars on measurement lines.
- Right: WEAKEST TOPICS table (topic, qty, recall, struggle; below 80% recall or above 55% struggle in accent); BUILD TODO SET FROM TOP 6.

## Mobile

Sub-tabs, big streak + scale, counts row, review card, weekly bars, stats, weakest topics list, build button.

## Acceptance criteria

- [ ] Matches the reference in all four themes at 1280×800 and 390×844 (tokens only, no raw colors).
- [ ] Uses the shared primitives from `design-system.md`; times are 12-hour.
- [ ] Hidden (and 404) when its module is off.
- [ ] Every existing Leetgrinder view (reviews, todo sets, quick review, analysis, notify, API tokens) uses the new tokens and primitives.
- [ ] The trend note is generated from data (e.g. compares last 4 weeks to the 4 before) or omitted.
