# Notes

**Current code:** src/pages/notes/index.astro and the note page (src/pages/static-shells/note.astro) → WorkspaceEditor, which draws the list with NotesList (helpers in src/lib/note-card.ts)
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

## Notes on the build

- Topics are single-select (All or one topic), as drawn; the older multi-select with Select all / Clear all is gone.
- Cards are links to `/notes/<id>`; a plain click opens the note in place (same reader and editor, with its draft guard).
- The reader shows topic, edited date and word count over the title, then summary, tags, todos and the body. A leading `# Title` heading in the body that repeats the title is not drawn twice.
- The editor's shared controls (fields, buttons, tags, todos, Markdown preview via `prose-ink`) use the tokens, so other pages that open the workspace editor (journal, audio thoughts, manage pages) get the restyled controls too.

## Acceptance criteria

- [ ] Matches the reference in all four themes at 1280×800 and 390×844 (tokens only, no raw colors).
- [ ] Uses the shared primitives from `design-system.md`; times are 12-hour.
- [ ] Hidden (and 404) when its module is off.
- [ ] Editing a note still uses the existing editor, restyled.
