# Career

A private, single-user workspace for planning a career move: goals and timelines, study
progress, reading, target companies, network connections, journals, and LeetCode practice.
One Go service serves the Astro frontend and the API, backed by PostgreSQL, and runs on a
homelab k3s cluster.

To run it locally:

```sh
make dev
```

Then open http://localhost:8080 and log in with `admin` / `password123`. See the
[developer guide](docs/developer.md) for setup, configuration, testing and releases.

## Features

### Home, progress and logs

The home page (`/`) shows the home page document, a quick work-journal entry box, a compact
progress summary, and links to the main sections. **Progress** (`/progress`)
is a dashboard of weekly study hours, metrics and notes. **Logs** (`/logs`) is a searchable,
faceted explorer over the weekly study journal.

### Goals: timeline, graph and agent context

**Timeline** (`/timeline`) is the source of truth for goals, their dates and their steps (mini
goals), which can be reordered or moved between goals. **Goal graph** (`/goals/graph`) shows
prerequisites between goals and their critical paths. **Agent context** (`/agents`) renders the
saved goals as Markdown or JSON (`/api/agent-context`) to hand to an AI assistant.

### Weekly scheduler

`/weekly-scheduler` plans goal work across the week on a drag-and-drop grid, with recurring
rules, and records what actually happened. It can optionally read busy time from Google
Calendar and export accepted plans to a dedicated calendar.
See [Weekly scheduler](docs/weekly-scheduler.md).

### Leetgrinder

`/leetgrinder` tracks LeetCode practice: a daily goal of new problems and reviews, spaced
repetition, todo sets, stats, ntfy reminders, and optional Claude analysis of each attempt's
complexity, correctness and code. A browser extension logs LeetCode and NeetCode submissions.
See [Leetgrinder](docs/leetgrinder.md) and the [extension](extension/leetgrinder/README.md).

### Bookshelf

**Bookshelf** (`/books`) tracks books, video courses and papers, with progress driven by a
Markdown chapter checklist and a dated reading log. Items are managed at `/manage/books`.

### Companies

**Companies** (`/companies`) holds target companies tiered by fit, each with notes, a checklist
and status. Companies can be edited or deleted from their cards, and managed at
`/manage/companies`.

### Connections

**Connections** (`/connections`) lists people in your network, linked to target companies, with
a catch-up queue. Import contacts and messages from a LinkedIn data export; re-importing updates
names, roles and companies but keeps your notes.

### Journals, notes and pages

- **Work journal** (`/journal`): dated work reflections and the weekly work journal.
- **Personal journal** (`/personal-journal`): life and personal background.
- **Notes** (`/notes`): ideas, questions and references organized by topic.
- **Pages** (`/documents`): the study plan and other reference pages, edited in Markdown.

Site search covers all of them.

### Audio thoughts

**Audio thoughts** (`/audio-thoughts`) records voice memos in the browser, queues them offline,
and transcribes them with a self-hosted whisper.cpp server. Audio never leaves the homelab.
See [running notes deployment](deploy/running/README.md).

### MCP server

The service exposes the workspace to AI clients such as Claude over MCP at `/api/mcp`, with
built-in single-owner OAuth. Tools read and edit entries, goals, the weekly schedule,
connections and Leetgrinder data. See [MCP](docs/mcp/README.md).

### Offline and mobile

The site is an installable PWA with an offline fallback page.
