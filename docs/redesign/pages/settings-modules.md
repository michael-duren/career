# Settings → Modules and Theme

**Current code:** new: src/pages/settings/*.astro
**Reference:** desktop `reference/desktop/ARC-Settings.html` · mobile `reference/mobile/M-Modules.html` (from `reference/mobile/M-More.html`)

## Purpose

Turn modules on/off, order the phone tab bar, choose a theme.

## Data

New `GET/PATCH /api/settings` (see `modules.md`).

## Desktop

- Left settings nav: General, Modules (active), Theme, Integrations, Reminders, Account.
- Modules list: switch, name, status (accent "needs Timeline"; "paused · 7 recordings kept"; dim "Schedule and Agent context use this"), description, "Appears in" tags (struck through when off).
- Right: PHONE TAB BAR preview (Home + 3 + More) that updates live, draggable order list, accent hint.
- Theme page: the four theme previews from setup step 4.

## Mobile

Back link to More, switches with descriptions and status, tab bar preview.

## Acceptance criteria

- [ ] Matches the reference in all four themes at 1280×800 and 390×844 (tokens only, no raw colors).
- [ ] Uses the shared primitives from `design-system.md`; times are 12-hour.
- [ ] Hidden (and 404) when its module is off.
- [ ] Changes save immediately and update the shell without reload.
- [ ] Dependencies enforced client- and server-side.
- [ ] Counts in "paused · N kept" come from real data.
