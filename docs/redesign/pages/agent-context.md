# Agent context

**Current code:** src/pages/agents.astro → AgentContextView (helpers in src/lib/agent-view.ts)
**Reference:** desktop `reference/desktop/ARC-Agents.html` · mobile `reference/mobile/M-Agents.html`

## Purpose

Goals rendered as Markdown or JSON to hand to an AI assistant.

## Data

`GET /api/agent-context`.

## Desktop

- Header: label, title, segmented MARKDOWN / JSON, primary COPY (shows "COPIED ✓").
- Left: Goals / Active / Steps / Characters stats; Endpoint box `GET /api/agent-context`; Include options (steps, finished goals, this week's schedule) if the API supports them (otherwise omit).
- Right: Box with a mono preview of the generated output and the file name label.

## Mobile

Segmented control + COPY, preview Box, endpoint note.

## Acceptance criteria

- [ ] Matches the reference in all four themes at 1280×800 and 390×844 (tokens only, no raw colors).
- [ ] Uses the shared primitives from `design-system.md`; times are 12-hour.
- [ ] Hidden (and 404) when its module is off.
- [ ] Copy copies the full output, not the visible preview.
