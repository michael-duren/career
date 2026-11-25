# Bookshelf

**Current code:** src/pages/books.astro → BooksView; manage at /manage/books
**Reference:** desktop `reference/desktop/ARC-Bookshelf.html` · mobile `reference/mobile/M-Bookshelf.html`

## Purpose

Books, courses and papers with chapter progress and an in-progress card.

## Data

`GET /api/entries/book` (type book/course/paper, chapter checklist, reading log).

## Desktop

- Header: "32 books, courses & papers", title, search (title or author), MANAGE, + ADD ITEM.
- Left: SHELVES (categories, active one with left rule), TYPE legend with line icons (book, paper, course).
- Middle: CATALOG table (type icon, title, author, priority label, chapter tick scale).
- Right: IN PROGRESS card (line-drawn book, title, author, chapters as 10 hatched/empty cells, last updated, LOG READING, OPEN ↗) + accent note ("6 chapters to go").

## Mobile

Title + `+`, search, in-progress card, horizontally scrolling shelf chips, list rows.

## Acceptance criteria

- [ ] Matches the reference in all four themes at 1280×800 and 390×844 (tokens only, no raw colors).
- [ ] Uses the shared primitives from `design-system.md`; times are 12-hour.
- [ ] Hidden (and 404) when its module is off.
- [ ] In progress = started (completed chapters > 0) and not finished, most recently updated first.
- [ ] Type icons are line drawings shared with search results.
