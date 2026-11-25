# First-run setup

**Current code:** new: src/pages/setup.astro
**Reference:** desktop `reference/setup/ARC-Setup-1..5.html` · mobile `reference/setup/M-Setup-1..5.html`

## Purpose

Walk new users through focus → modules → connect → theme → first goal.

## Data

`PATCH /api/settings` (modules, theme, setup_completed_at), `POST /api/goals` for the first goal, existing Google/LinkedIn/ntfy/extension/MCP setup links.

## Desktop

- Top: wordmark, "Setup · step N of 5", SKIP SETUP. Step indicator: measurement line with five dots (done filled, current hatched) and labels.
- 1 Focus: five multi-select cards (Interview prep, Learning, Networking, Planning my time, Reflecting) each listing the modules it turns on.
- 2 Modules: two-column switches pre-set from focus, "suggested" tags, dependency locks.
- 3 Connect: only integrations for enabled modules (Google Calendar, LinkedIn export, ntfy reminders, browser extension, Claude MCP) with CTA buttons; all optional.
- 4 Theme: four previews; selecting applies the theme immediately.
- 5 First goal: goal name, start, end; summary of modules on; FINISH · OPEN HOME.
- Footer: ← BACK, CONTINUE → (primary).

## Mobile

Same five steps in one column; focus cards stacked; theme previews in a 2×2 grid.

## Acceptance criteria

- [ ] Matches the reference in all four themes at 1280×800 and 390×844 (tokens only, no raw colors).
- [ ] Uses the shared primitives from `design-system.md`; times are 12-hour.
- [ ] Hidden (and 404) when its module is off.
- [ ] New users (setup_completed_at null) are redirected here after login; existing users never see it.
- [ ] Skip sets defaults (all modules on, dark) and marks setup complete.
- [ ] Re-runnable from Settings → General.
