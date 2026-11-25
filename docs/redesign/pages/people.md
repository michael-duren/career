# People (Connections) and catch-up queue

**Current code:** src/pages/connections.astro → ConnectionsBoard.tsx
**Reference:** desktop `reference/desktop/ARC-Connections.html` (People tab), `reference/desktop/ARC-CatchUp.html` (Catch-up tab) · mobile `reference/mobile/M-Connections.html`

## Purpose

Network: all people, a catch-up queue for warming up contacts at target companies, and where the network works.

## Data

`GET /api/connections`, `POST /api/connections/import` (LinkedIn export), `/api/connections/photo`. **New:** a `stage` on connections (`to_contact | messaged | follow_up | referral_ask`, nullable = not in queue), `last_contacted_on` (exists from imports), and endpoints to add/move/remove queue entries and mark "sent" (sets last contact = today, moves to next stage) and "snooze" (hides until a date). Queue candidates = people at companies on the board.

## Desktop

- Header: label "819 people · 653 employers", title, segmented tabs PEOPLE / CATCH-UP QUEUE · n / EMPLOYERS, search, IMPORT LINKEDIN.
- People tab: table (avatar, name, role, company + ◆ if on the board, last contact) + detail panel (Box: avatar, name, role, last contact, count at company, notes, LOG CATCH-UP, LINKEDIN ↗).
- Catch-up tab: goal strip (linked goal name, its steps on a measurement line, "0 / 2 messages this week"); four stage columns with person cards (selected card has accent outline); empty column shows dashed "Nobody here yet"; detail panel with next action per stage, accent "last contact Sep 11 · 75 days ago", message draft, MARK SENT, SNOOZE 1 WK, move-to-stage buttons.
- Employers tab: people per employer on measurement lines (on-board ones solid, others hatched) + "Not on your board yet" list with ADD TO COMPANIES.

## Mobile

Segmented tabs (PEOPLE / CATCH-UP · n / EMPLOYERS), search, lists. Catch-up groups people under stage headers with a four-step goal bar and weekly counter.

## Acceptance criteria

- [ ] Matches the reference in all four themes at 1280×800 and 390×844 (tokens only, no raw colors).
- [ ] Uses the shared primitives from `design-system.md`; times are 12-hour.
- [ ] Hidden (and 404) when its module is off.
- [ ] Re-import keeps notes and queue stages (extends today's guarantee).
- [ ] Weekly message counter counts "sent" actions this week; the target is configurable (default 2).
- [ ] Catch-up tab hidden with a note when Companies is off.
- [ ] Migration adds the new columns with safe defaults; MCP connection tools expose stage.
