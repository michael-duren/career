# Journal

**Current code:** src/pages/journal/index.astro → JournalHub.tsx, LogExplorer.tsx; /journal/new → QuickJournal
**Reference:** desktop `reference/desktop/ARC-Journal.html` · mobile `reference/mobile/M-Journal.html`

## Purpose

Weekly work journal, private personal journal, and the searchable Logs explorer as tabs.

## Data

`GET /api/entries/work_journal`, `GET /api/entries/personal_journal`, existing logs/search behavior.

## Desktop

- Header: "15 work weeks · 2 personal", title, search, segmented WORK / PERSONAL / LOGS, + NEW ENTRY.
- Work: week list (W26 + dates, selected has left rule) beside the entry: date label, "Week 26", EDIT, measurement line with hours ("No hours logged this week"), sections What I did / Blockers / Notes as ruled lists; accent note "last entry Oct 16 — catch up?" when stale.
- Logs: filters (year, section) + large search with highlighted matches.
- Personal: lock drawing, count, "Private · open to read".

## Mobile

Tabs, search, accent note, week rows that expand inline to show the entry.

## Acceptance criteria

- [ ] Matches the reference in all four themes at 1280×800 and 390×844 (tokens only, no raw colors).
- [ ] Uses the shared primitives from `design-system.md`; times are 12-hour.
- [ ] Hidden (and 404) when its module is off.
- [ ] `/logs` and `/personal-journal` redirects keep working.
- [ ] Stale note only when the latest entry is older than 7 days.
