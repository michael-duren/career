# Redesign: the "Architect" UI

This folder is the handoff for a full visual redesign of Career. It replaces the current
zinc/blue Tailwind look with a precise, hand-drawn **architectural drawing** style:
hairline ink lines, measurement lines with slashed ends, crosshatched fills, a thin
condensed sans for headings, monospace for numbers, and a sparing "red pencil" accent.

The design was made on a Claude Design canvas. Every screen from that canvas is exported
to [`reference/`](reference/README.md) as a self-contained HTML file, so exact values
(colors, sizes, spacing, copy) can be read directly instead of guessed.

Paths in these docs are relative to `docs/redesign/`.

## Read in this order

1. [`design-system.md`](design-system.md) — tokens, themes, type, the drawing primitives.
2. [`modules.md`](modules.md) — every feature becomes a module that can be turned off.
3. [`shell.md`](shell.md) — desktop header, global search, mobile top bar and tab bar.
4. [`pages/`](pages/) — one spec per page, desktop and mobile.
5. [`issues.md`](issues.md) — the epic and every issue, ready to create in GitHub.

## Decisions already made

| Decision | Choice |
| --- | --- |
| Visual direction | "Architect": precise pen-and-ink drawing style, no drawing-set jargon (no sheet numbers, title blocks or north arrows) |
| Themes | **Dark** (default), **Vellum** (light), **Cobalt** (cobalt ink on white), **Concrete** (warm mid-gray, safety-yellow accent). Same layout, only tokens change. |
| Mobile | In scope. Phone layout below 768px with a bottom tab bar. |
| Time format | 12-hour with AM/PM everywhere (`5:20 PM`, `7:15 AM – 8:45 AM`). |
| Navigation | Desktop: one top header with every enabled module, global search, settings. Mobile: Home + 3 module tabs + More. |
| Modules | Every feature can be turned off in Settings → Modules. Off = hidden everywhere, data kept. |
| First run | New users go through a five-step setup (focus → modules → connect → theme → first goal). |
| Logos | Company logos come from the company's website domain and render in single-color ink to match the theme. |
| Catch-up queue | Connections get a `stage` (to contact → messaged → follow up → referral ask). This is new data. |

## Out of scope

- No change to existing API behavior, data, or business rules unless a page spec says so
  (the only new data: module settings, theme setting, setup-completed flag, connection stage).
- No new features beyond what the screens show.
- The Go-rendered Leetgrinder pages are restyled, not rewritten.

## Order of work

1. **Foundation** — tokens and themes, fonts, drawing primitives, time formatting.
2. **Modules and shell** — module registry (frontend + backend), desktop header, mobile tab bar, global search.
3. **Pages** — one issue per page, desktop and mobile together. Pages can run in parallel once 1–2 are merged.
4. **Settings, setup, cleanup** — Modules and Theme settings, first-run setup, visual tests, removing old styles.

## How to check work

- `make dev` (or `npm run dev`) and open http://localhost:8080, log in `admin` / `password123`.
- Compare each page against its reference file in `reference/` at 1280×800 (desktop) and 390×844 (mobile), in all four themes.
- `npm test`, `make test`, and `npm run test:e2e` must stay green.
