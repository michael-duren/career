# Epic and issues

Instructions for the agent creating these in GitHub (`michael-duren/career`):

1. Create the labels below if missing.
2. Create the **epic** first, then every issue in order. Each local id (`R01`…) becomes a real
   issue number; replace `Depends on: R0x` with `Depends on: #<number>` as you go.
3. Add each issue as a **sub-issue** of the epic (GitHub sub-issues; if unavailable, a task list
   in the epic body linking every issue).
4. Use the title and body exactly as written. Every body links back to the spec files in
   `docs/redesign/`.

Paths below are relative to `docs/redesign/`.

## Labels

| Label | Color | Meaning |
| --- | --- | --- |
| `epic` | `#3E4B9E` | The tracking issue |
| `redesign` | `#E0533D` | Part of the Architect UI redesign |
| `area:frontend` | `#1D76DB` | Astro / React / CSS |
| `area:backend` | `#5319E7` | Go API, database, templ |
| `phase:1-foundation` | `#C5DEF5` | |
| `phase:2-shell` | `#BFD4F2` | |
| `phase:3-pages` | `#D4C5F9` | |
| `phase:4-finish` | `#F9D0C4` | |

---

## EPIC — Redesign: Architect UI (themes, mobile, modules, setup)

**Labels:** `epic`, `redesign`

```md
Replace the current zinc/blue UI with the "Architect" pen-and-ink design, add four themes,
a real mobile layout, switchable modules, and a first-run setup.

Spec: docs/redesign/README.md (start here) · design-system.md · modules.md · shell.md · pages/
Reference screens: docs/redesign/reference/

Phases
1. Foundation — tokens/themes, fonts, primitives, 12-hour formatting
2. Modules + shell — registry, settings API, desktop header, mobile tab bar, global search
3. Pages — one issue each, desktop + mobile (parallel once phases 1–2 merge)
4. Finish — settings UI, setup flow, login, visual tests, cleanup

Rules for every issue
- No API/behavior changes unless the issue says so.
- Tokens only (no raw colors); shared primitives only; 12-hour times.
- Check against the reference at 1280×800 and 390×844 in all four themes.
- npm test, make test and npm run test:e2e stay green.
```

---

## Phase 1 — Foundation

### R01 — Design tokens and four themes

**Labels:** `redesign`, `area:frontend`, `phase:1-foundation` · **Depends on:** —

```md
Add the theme tokens from docs/redesign/design-system.md (Themes and tokens) to
src/styles/global.css as CSS variables, mapped into Tailwind v4 with @theme.

- data-theme="dark|vellum|cobalt|concrete" on <html>, default dark; no flash on load
  (set server-side or inline before paint).
- Self-host Inter Tight (200–500), IBM Plex Mono (400–500), Architects Daughter.
- Body background/text use --paper/--ink.

Acceptance
- [ ] Switching data-theme in devtools re-themes every token-based element.
- [ ] Labels at --ink-60 pass 4.5:1 contrast in all four themes.
- [ ] No Google Fonts request at runtime.
```

### R02 — Drawing primitives

**Labels:** `redesign`, `area:frontend`, `phase:1-foundation` · **Depends on:** R01

```md
Build the primitives in docs/redesign/design-system.md (Drawing primitives) as React
components in src/components/ui/ plus a CSS layer usable from Astro and Go templ pages:
Box (3px corner overshoot), MeasurementLine, HatchFill (diagonal/crosshatch/dots),
GoalPattern, TickScale, Switch, Checkbox (X + accent ring), SegmentedTabs, Button
(primary/secondary/icon/danger), SearchField, SectionHeader, Label, NowLine, LogoSpot,
Avatar, line icons.

Add a dev-only /dev/ui page showing every primitive in every state for visual review.

Acceptance
- [ ] Values match the reference files (.bx, .dim, .lbl, .btn in any reference/*.html).
- [ ] Keyboard and screen-reader behavior per design-system.md (Accessibility).
```

### R03 — 12-hour time and date formatting

**Labels:** `redesign`, `area:frontend`, `area:backend`, `phase:1-foundation` · **Depends on:** —

```md
One formatter for the frontend (src/lib) and one for Go templ pages, per
design-system.md (Formatting): 7:15 AM, 5:45 PM, ranges "9 AM – 4 PM", dates "Nov 24".
Replace existing 24-hour displays as pages are migrated (pages issues use these).

Acceptance
- [ ] Unit tests for both formatters, including midnight/noon and minutes == 0.
```

## Phase 2 — Modules and shell

### R04 — Module registry and settings API

**Labels:** `redesign`, `area:backend`, `area:frontend`, `phase:2-shell` · **Depends on:** —

```md
Implement docs/redesign/modules.md.

- Registry (TS) with ids, labels, routes, home section, search source, reminders, MCP tools,
  API prefixes, depends-on; mirror ids in Go.
- New settings storage + migration: modules (id→bool), module_order, theme,
  setup_completed_at. Existing users: all modules on, setup marked complete.
- GET/PATCH /api/settings with validation and server-side dependency enforcement.
- Server guard: disabled module pages and API routes return 404; MCP tools of disabled
  modules are hidden/refused; Leetgrinder extension gets a clear error; reminders skip.
- Search results filtered by enabled modules.

Acceptance
- [ ] Turning a module off hides it everywhere and keeps its data; on restores it.
- [ ] Tests for dependency rules, 404 guard, MCP filtering.
```

### R05 — Desktop header and app shell

**Labels:** `redesign`, `area:frontend`, `phase:2-shell` · **Depends on:** R01, R02, R04

```md
Replace Sidebar.astro with the desktop header in docs/redesign/shell.md: wordmark, menu
links from enabled modules (short labels, current page underlined), search field,
settings button, 12-hour clock. Overflow collapses to MORE +.

Reference: header of any docs/redesign/reference/desktop/*.html

Acceptance
- [ ] Menu updates when modules change, without reload.
- [ ] aria-current on the active link; works on Go-rendered Leetgrinder pages too.
```

### R06 — Mobile top bar, tab bar and More page

**Labels:** `redesign`, `area:frontend`, `phase:2-shell` · **Depends on:** R05

```md
Below 768px: top bar (wordmark, search button, time), fixed bottom tab bar
(Home + first 3 enabled modules in module_order + More), and the More page with remaining
modules, Settings → Modules, theme picker, sign out. See docs/redesign/shell.md.

Reference: docs/redesign/reference/mobile/M-Home.html, M-More.html

Acceptance
- [ ] Safe-area aware; 44px touch targets; no layout shift between pages.
- [ ] Turning a tab's module off fills the slot with the next enabled module.
```

### R07 — Global search UI

**Labels:** `redesign`, `area:frontend`, `phase:2-shell` · **Depends on:** R05, R06

```md
Header search panel (desktop) and full-screen search (mobile) with results grouped by
module, counts, one-line context, recent searches, "/" shortcut. Extend GET /api/search to
return group and context if needed. See docs/redesign/shell.md (Global search).

Reference: docs/redesign/reference/mobile/M-Search.html

Acceptance
- [ ] Only enabled modules appear; keyboard navigation through results.
```

## Phase 3 — Pages (desktop + mobile in each issue)

Each page issue body:

```md
Restyle <Page> per docs/redesign/pages/<file>.md (desktop and mobile).
Reference screens are listed at the top of that file.
Keep all existing behavior; only the data changes listed under "Data" are in scope.
Acceptance criteria: see the spec file (all boxes must be checked in the PR).
```

| Id | Title | Spec | Labels (besides `redesign`, `phase:3-pages`) | Depends on |
| --- | --- | --- | --- | --- |
| R08 | Home page | `pages/home.md` | `area:frontend` | R02, R03, R05, R06 |
| R09 | Timeline page | `pages/timeline.md` | `area:frontend` | R02, R03, R05, R06 |
| R10 | Weekly scheduler page | `pages/schedule.md` | `area:frontend` | R02, R03, R05, R06 |
| R11 | Leetgrinder pages (Go templ) | `pages/leetgrinder.md` | `area:backend` | R02, R03, R05 |
| R12 | Companies page with logos, edit/delete and search | `pages/companies.md` | `area:frontend`, `area:backend` | R02, R05, R06 |
| R13 | People page (People + Employers tabs) | `pages/people.md` | `area:frontend` | R02, R05, R06 |
| R14 | Catch-up queue: connection stages (data + API) | `pages/people.md` (Data) | `area:backend` | R04 |
| R15 | Catch-up queue tab UI | `pages/people.md` | `area:frontend` | R13, R14 |
| R16 | Bookshelf page | `pages/bookshelf.md` | `area:frontend` | R02, R05, R06 |
| R17 | Journal page (Work, Personal, Logs) | `pages/journal.md` | `area:frontend` | R02, R03, R05, R06 |
| R18 | Notes page | `pages/notes.md` | `area:frontend` | R02, R05, R06 |
| R19 | Audio thoughts page | `pages/audio-thoughts.md` | `area:frontend` | R02, R03, R05, R06 |
| R20 | Agent context page | `pages/agent-context.md` | `area:frontend` | R02, R05, R06 |

## Phase 4 — Finish

### R21 — Settings: Modules and Theme pages

**Labels:** `redesign`, `area:frontend`, `phase:4-finish` · **Depends on:** R04, R05, R06

```md
Build docs/redesign/pages/settings-modules.md: settings nav, Modules list with switches,
status and "appears in" tags, live phone tab bar preview with drag order, Theme page.
```

### R22 — First-run setup flow

**Labels:** `redesign`, `area:frontend`, `area:backend`, `phase:4-finish` · **Depends on:** R04, R21

```md
Build docs/redesign/pages/setup.md (five steps, desktop + mobile), redirect new users after
login, Skip with defaults, re-run from Settings → General.
```

### R23 — Login page restyle

**Labels:** `redesign`, `area:frontend`, `phase:4-finish` · **Depends on:** R01, R02

```md
No screen was designed. Apply the design system to src/pages/login.astro: wordmark,
Inter Tight 200 title, Box around the form, 44px fields with --ink underline, primary button,
accent error text. Follow the setup screens' look (docs/redesign/reference/setup/).
```

### R24 — Visual regression tests

**Labels:** `redesign`, `area:frontend`, `phase:4-finish` · **Depends on:** R08–R23

```md
Playwright screenshot tests for every page at 1280×800 and 390×844 in all four themes,
with seeded data, plus a test that a disabled module's routes 404.
```

### R25 — Remove old styles and components

**Labels:** `redesign`, `area:frontend`, `phase:4-finish` · **Depends on:** R24

```md
Delete Sidebar.astro, unused zinc/blue utility styles and components replaced by the
primitives; make sure no raw color values remain outside the token file.
```
