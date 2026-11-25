# Companies

**Current code:** src/pages/companies.astro → CompaniesView; manage at /manage/companies
**Reference:** desktop `reference/desktop/ARC-Companies.html` · mobile `reference/mobile/M-Companies.html`

## Purpose

Target companies as cards with logo, priority, featured mark, why, checklist progress, edit and delete.

## Data

`GET /api/entries/company` (and existing write paths used by edit/delete). Logos: derive from the company `url` domain via a logo/favicon service or a cached server-side fetch; render monochrome (CSS filter or SVG mask) so it matches the ink. Fallback: two-letter monogram.

## Desktop

- Header: "61 tracked · 7 featured", title, segmented priority filter (ALL n / HIGH / MEDIUM / LOW), search field, MANAGE, primary + ADD COMPANY.
- Legend row: priority dots (filled / half / empty), featured ◆, checklist ticks; "N shown".
- 4-column card grid. Card (Box): logo spot + name + category label, priority dot (+◆), two-line why, checklist tick scale + `2/5`, edit and delete icon buttons (delete in accent, with a confirm).

## Mobile

Title + `+` button, search, priority segmented control, rows: logo spot, name (+◆, dot), category + ticks, edit and delete buttons (40px).

## Acceptance criteria

- [ ] Matches the reference in all four themes at 1280×800 and 390×844 (tokens only, no raw colors).
- [ ] Uses the shared primitives from `design-system.md`; times are 12-hour.
- [ ] Hidden (and 404) when its module is off.
- [ ] Search filters by name, category, tags and why text.
- [ ] Edit opens the existing editor; delete confirms first (same backend behavior as today).
- [ ] Logos load lazily, are cached, and never break layout when missing.
