# App shell

Replaces `src/components/Sidebar.astro` and the current layout chrome. Built from the
module registry (see `modules.md`).

## Desktop header (≥ 768px)

Reference: top of any `reference/desktop/*.html`.

One 28px row, then a `--ink-30` rule:

1. Wordmark `CAREER` (Inter Tight 500, 13px, letter-spacing 0.32em).
2. Menu links for Home + every enabled module, short labels: HOME, TIMELINE, SCHEDULE, LEET,
   COMPANIES, PEOPLE, BOOKS, JOURNAL, NOTES, AUDIO, AGENTS. 11.5px, letter-spacing 0.06em,
   `--ink-60` (60% ink, raised per theme for contrast); the current page is full opacity with a 1px underline and `aria-current="page"`.
3. Global search field (210px, right-aligned), placeholder "Search everything", `/` shortcut hint.
4. Settings button (34px square, line-drawn gear) → `/settings/modules`.
5. Current time in mono, 12-hour (`5:20 PM`).

If the links don't fit (narrow desktop), collapse the overflow into a `MORE +` menu.

Implementation: `src/components/AppHeader.astro` (Astro pages) and `internal/shell/header.templ`
(Go-rendered pages) render the same markup, with every module in registry order and the current
one marked. `public/ui/shell.css` styles it; `public/ui/shell.js` hides turned-off modules, applies
the user's order, fills `MORE +` and runs the clock; `public/ui/site-search.js` runs the search field.
`saveSettings()` in `src/lib/modules.ts` fires a `career:settings` window event and posts on the
`career-settings` BroadcastChannel, so the menu updates without a reload in every open tab.
Until Settings → Account exists, Sign out is the last item in `MORE +`, so `MORE +` always shows.

## Global search

- Desktop: typing in the header field opens a results panel under it. Mobile: the search icon
  opens a full-screen search page (`reference/mobile/M-Search.html`).
- Results are grouped by module (Goals, Notes, Journal, Schedule, Companies, People, Books…)
  with a count per group, title + one-line context per result. Only enabled modules appear.
- Uses the existing `GET /api/search`; extend it to return module/group and context if needed.
- Recent searches as chips when the query is empty.

## Mobile (< 768px)

Reference: any `reference/mobile/*.html`.

- **Top bar** (in the scroll area): `CAREER`, a 44px search icon button, time. Then a `--ink-30` rule.
- **Tab bar** fixed at the bottom, 78px tall (includes safe-area padding): five equal tabs with
  22px line icons and mono 9px labels. Slots: Home, the first 3 enabled modules in the user's
  order, More. Active tab: full opacity + 1px line on top, `aria-current="page"`.
- **More page** (`M-More.html`): every enabled module not in the tab bar with its count, then a
  Settings section (Modules), a Theme picker, and Sign out.
- No fake status bar; respect `viewport-fit=cover` and safe areas.

## Page header pattern

Every page starts with: mono label (counts, e.g. "61 tracked · 7 featured"), Inter Tight 200
title, and actions on the right (search field, segmented tabs, secondary buttons, one primary
button). On mobile, actions collapse to a `+` square button and the search field moves under
the title.
