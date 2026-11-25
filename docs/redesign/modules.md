# Modules

Every feature is a module the user can turn off. This must be built into the shell from
the start: menus, Home, search and the phone tab bar are generated from the enabled
modules, never hard-coded.

References: `reference/desktop/ARC-Settings.html`, `reference/mobile/M-Modules.html`,
`reference/mobile/M-More.html`, `reference/setup/`.

## The registry

One registry shared by frontend and backend (define in TypeScript, mirror the ids in Go).

| id | Label (menu / tab) | Route | Home section | Search | Reminders | Depends on |
| --- | --- | --- | --- | --- | --- | --- |
| `timeline` | Timeline / TIMELINE | `/timeline` | Active goals | goals | | |
| `schedule` | Schedule / SCHEDULE | `/weekly-scheduler` | Today's sessions, missed | sessions | | `timeline` |
| `leetgrinder` | Leetgrinder / LEET | `/leetgrinder` | Streak + review card | problems | ntfy | |
| `companies` | Companies / COMPANIES | `/companies`, `/manage/companies` | | companies | | |
| `people` | People / PEOPLE | `/connections` | | people | | (catch-up queue needs `companies`) |
| `bookshelf` | Bookshelf / BOOKS | `/books`, `/manage/books` | | books | | |
| `journal` | Journal / JOURNAL | `/journal` | | entries | | |
| `notes` | Notes / NOTES | `/notes` | | notes | | |
| `audio` | Audio thoughts / AUDIO | `/audio-thoughts` | | transcripts | | |
| `agents` | Agent context / AGENTS | `/agents` | | | | `timeline` |

Home, Search, Settings and Setup are always on.

Each entry also lists its icon (line drawing), MCP tools it owns, and the API prefixes it owns
(for the server guard).

## Behavior

- **Off = paused.** Hide the module everywhere: header, More list, tab bar, Home section,
  search results, reminders. Keep all data. Turning it on restores everything.
- **Dependencies.** If a dependency is off, the dependent module is effectively off and its
  switch is locked (dashed) with "needs Timeline". Turning off Companies keeps People on but
  hides the catch-up queue tab with a note.
- **Server guard.** Pages and API routes of a disabled module return 404. MCP tools of a
  disabled module are not listed and return an error if called. The browser extension
  gets a clear "Leetgrinder is turned off" response.
- **Phone tab bar.** Home + the first 3 enabled modules in the user's order + More. The
  order is editable in Settings → Modules (drag handle). If a module is off, the next one
  takes its slot.

## Storage and API

- New user setting `modules`: map of id → enabled, plus `module_order` (array of ids).
  Default for existing users: all on (no visible change). Default for new users: chosen in setup.
- New setting `theme` (`dark|vellum|cobalt|concrete`), default `dark`.
- New setting `setup_completed_at` (null for new users → redirect to `/setup` after login).
- Suggested endpoints: `GET /api/settings`, `PATCH /api/settings` (modules, module_order, theme,
  setup_completed_at). Validate ids against the registry and enforce dependencies server-side.

## Setup suggestions

The setup flow's focus picks pre-select modules:

| Focus | Turns on |
| --- | --- |
| Interview prep | leetgrinder, timeline, schedule, companies |
| Learning | bookshelf, notes, timeline |
| Networking | people, companies |
| Planning my time | timeline, schedule, agents |
| Reflecting | journal, audio |
