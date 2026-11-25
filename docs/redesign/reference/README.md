# Reference screens

Exported from the Claude Design canvas. Each file is one screen, self-contained HTML/CSS.
Use them for **exact values and structure**: colors, font sizes, spacing, borders, copy, and
what goes where. They are mockups, not production code.

## How to read them

- They were written for the canvas runtime: `<x-dc>`, `<helmet>`, `<sc-for>` (loops),
  `<sc-if>` (conditionals) and `{{holes}}` filled by the `class Component extends DCLogic`
  script at the bottom. `support.js` is the canvas runtime and is not included, so the files
  won't render on their own. Read the markup and the script's `renderVals()` for the data and
  the computed styles.
- Inline `style` attributes hold the real values. The `<helmet><style>` block defines the shared
  primitives (`.bx` box with corner overshoot, `.dim` measurement line, `.lbl` label, `.btn`).
- Data in them is a real snapshot of the owner's workspace from Nov 24–25, 2026. Wire pages to
  the live API instead; never hard-code it.
- Colors are written as raw values from the **Dark** theme (`#0C0C0B` paper, `#ECEAE3` ink,
  `rgba(236,234,227,a)`, `#E0533D` accent). Replace them with tokens from `design-system.md`.

## Index

| Screen | Desktop | Mobile |
| --- | --- | --- |
| Home | `desktop/ARC-Home.html` | `mobile/M-Home.html` |
| Timeline | `desktop/ARC-Timeline.html` | `mobile/M-Timeline.html` |
| Weekly scheduler | `desktop/ARC-Scheduler.html` | `mobile/M-Schedule.html` |
| Leetgrinder | `desktop/ARC-Leetgrinder.html` | `mobile/M-Leetgrinder.html` |
| Companies | `desktop/ARC-Companies.html` | `mobile/M-Companies.html` |
| People (people tab) | `desktop/ARC-Connections.html` | `mobile/M-Connections.html` |
| People (catch-up queue) | `desktop/ARC-CatchUp.html` | `mobile/M-Connections.html` |
| Bookshelf | `desktop/ARC-Bookshelf.html` | `mobile/M-Bookshelf.html` |
| Journal | `desktop/ARC-Journal.html` | `mobile/M-Journal.html` |
| Notes | `desktop/ARC-Notes.html` | `mobile/M-Notes.html` |
| Audio thoughts | `desktop/ARC-Audio.html` | `mobile/M-Audio.html` |
| Agent context | `desktop/ARC-Agents.html` | `mobile/M-Agents.html` |
| Settings → Modules | `desktop/ARC-Settings.html` | `mobile/M-Modules.html` |
| More menu | — | `mobile/M-More.html` |
| Global search | header of every desktop file | `mobile/M-Search.html` |
| First-run setup (5 steps) | `setup/ARC-Setup-1..5.html` | `setup/M-Setup-1..5.html` |
| Theme examples | `themes/VEL-Home.html` (Vellum), `themes/BLU-Home.html` (Cobalt), `themes/CON-Home.html` (Concrete), `themes/MVEL-Home.html` (Vellum mobile) | |

The canvas itself (owner access only): https://claude.ai/artifact/98dLhhpG9k5tb1s2e5k5TA
