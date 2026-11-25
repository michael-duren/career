# Notes

**Current code:** src/pages/notes/index.astro → WorkspaceEditor (+ note-card)
**Reference:** desktop `reference/desktop/ARC-Notes.html` · mobile `reference/mobile/M-Notes.html`

## Purpose

Ideas, questions and references by topic.

## Data

`GET /api/entries/note` (topic, description, word count, todo counts), `POST /api/entries/note`.

## Desktop

- Header: "21 notes · 10 topics", title, search, + NEW NOTE.
- Left: TOPICS with counts (All active).
- Grid of note cards (Box): topic label, date, title (2 lines), description (2 lines), word count, todo progress (hatched bar on a measurement line + `18/33`) when the note has todos.

## Mobile

Title + `+`, search, scrolling topic chips with counts, note rows.

## Acceptance criteria

- [ ] Matches the reference in all four themes at 1280×800 and 390×844 (tokens only, no raw colors).
- [ ] Uses the shared primitives from `design-system.md`; times are 12-hour.
- [ ] Hidden (and 404) when its module is off.
- [ ] Editing a note still uses the existing editor, restyled.
