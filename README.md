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

### Home

The home page (`/`) shows today at a glance. Its core is today's todo list: the goal sessions planned for today in the weekly
scheduler. Nothing is completed automatically; checking an item records its planned time as
done, and unchecking it clears that again. Sessions from earlier in the week that were never
checked off are listed underneath so they can still be checked. The next session (the first one
neither done nor skipped that has not ended) is circled, and a note says how many are done and when the
next starts (or "now until …" while it runs). Home also shows active goals with their step progress (Timeline) and the
Leetgrinder streak, today's counts and review pick. Each section is hidden while its module is
off in Settings → Modules.

### Goals: timeline and agent context

**Timeline** (`/timeline`) is the source of truth for goals, their dates and their steps (mini
goals; the API can also reorder steps or move them between goals). It draws every goal on a month axis (fit to the goals'
dates by default, or 6 months to 10 years) with its status and step progress; select a goal to see its steps, drag
a span on desktop to move or resize it, and open the goal editor for everything else. **Agent context** (`/agents`) renders the
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

**People** (`/connections`) has three tabs: **People** (everyone, with a detail panel for notes,
logging a catch-up and the LinkedIn profile), **Catch-up queue** (needs the Companies module) and
**Employers** (people per employer, with the ones not on your companies board yet and a button to add
them). Import contacts and messages from a LinkedIn data export; re-importing updates names, roles
and companies but keeps your notes and catch-up stages. `?tab=queue|employers` opens a tab.

The catch-up queue moves people at your board companies through four stages (to contact, messaged,
follow up, referral ask). **Mark sent** logs a message, sets the last contact to today and moves the
person on (after the referral ask they leave the queue); **Snooze 1 wk** hides them for a week. The strip on top counts this week's messages
against a weekly target (default 2, change it with **Target**) and shows the steps of one linked
timeline goal (**Link a goal** sets `catchUpQueue` in that goal's metadata; it needs the Timeline
module, and only planned or active goals can be linked). Message drafts stay in
your browser only. People due by their cadence, or flagged in the edit form, are listed under the
board.

### Journal and notes

- **Journal** (`/journal`): the weekly work journal, the personal journal and the searchable,
  faceted **Logs** explorer over the work journal, as tabs. `/logs` and `/personal-journal`
  redirect to their tabs.
- **Notes** (`/notes`): ideas, questions and references organized by topic.

Reference pages such as the study plan are read-only views at `/documents/<id>`. Site search
covers all of them.

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
