# Design system

The whole app is drawn like a precise architectural pen drawing. Everything is ink on
paper: no gradients, no shadows, no rounded cards, no color fills except crosshatching
and the single accent.

## Themes and tokens

All color comes from these tokens. Themes only swap values. Set the theme with
`data-theme="dark|vellum|cobalt|concrete"` on `<html>`; default `dark`.

| Token | Use | Dark | Vellum | Cobalt | Concrete |
| --- | --- | --- | --- | --- | --- |
| `--paper` | page background | `#0C0C0B` | `#F3F1EA` | `#F7F7F4` | `#2C2D2F` |
| `--ink` | text, lines, solid fills | `#ECEAE3` | `#1A1A17` | `#0047AB` | `#EFEDE6` |
| `--ink-rgb` | for alpha variants | `236,234,227` | `26,26,23` | `0,71,171` | `239,237,230` |
| `--accent` | "red pencil": next, late, warnings, delete, now line, handwritten notes | `#E0533D` | `#C2361E` | `#D6402A` | `#F2C230` |
| `--ink-hover` | link hover | `#FFFFFF` | `#000000` | `#002F73` | `#FFFFFF` |

Alpha steps used throughout (as `rgba(var(--ink-rgb), a)`):

| Name | Alpha | Use |
| --- | --- | --- |
| `--ink-60` | 0.6 in Dark and Concrete; per theme, see below | labels, secondary text |
| `--ink-50` | 0.5 | secondary button borders |
| `--ink-30` | 0.3 | section rules, field borders |
| `--ink-14` | 0.14 | row dividers |
| `--ink-07` | 0.07 | faint grid and guide lines |
| `--ink-05` | 0.05 | selected row tint |

At 0.6, labels fall below 4.5:1 on Vellum (4.39:1) and Cobalt (3.17:1). So `--ink-60` is
`rgba(var(--ink-rgb), var(--ink-60-alpha))`, and each theme sets its own `--ink-60-alpha`:

| Theme | `--ink-60-alpha` | Label contrast on `--paper` |
| --- | --- | --- |
| Dark | 0.6 | 6.2:1 |
| Vellum | 0.64 | 5.0:1 |
| Cobalt | 0.78 | 4.8:1 |
| Concrete | 0.6 | 5.3:1 |

Use `text-ink-60` for labels, not `text-ink/60`, because the fixed 60% skips the per-theme alpha.
`--ink-50` and lower are for borders, rules and tints only, never for text.

Known gaps for the design owner:

- `--accent` on Cobalt paper is 4.2:1, under 4.5:1 for normal text (it passes only as WCAG large
  text: 24px, or 18.66px bold). Small accent text on Cobalt fails until the accent is adjusted.
  Accent text should always come with a non-color cue (LATE, strike-through, the ring).
- `--ink-50` on Cobalt is 2.5:1, under the 3:1 non-text minimum for borders that identify
  controls.

The token values live in `public/ui/tokens.css` (plain CSS, so Go templ pages can link
`/ui/tokens.css`); `src/styles/global.css` imports it. Map these into Tailwind v4 with
`@theme inline` in `src/styles/global.css` so classes like
`bg-paper text-ink border-ink/30 text-ink-60` work. `inline` makes utilities reference the
variables directly, so an element with its own `data-theme` (a theme preview) re-resolves them.

## Typography

| Role | Font | Notes |
| --- | --- | --- |
| Display and headings | **Inter Tight** 200 | Page titles 52px desktop / 40px mobile, line-height 0.9. Home date 128px / 70px. |
| Body | **Inter Tight** 300 / 400 / 500 | Rows 15–18px, descriptions 12–13px. |
| Labels | **IBM Plex Mono** 500, 10px, uppercase, letter-spacing 0.16em, `--ink-60` | Section labels, column headers, metadata. |
| Section headings | Inter Tight 500, 12px (11px mobile), letter-spacing 0.24em, uppercase | e.g. GOAL SESSIONS |
| Numbers, times, dates | **IBM Plex Mono** 400 / 500 | Counts, times, steps (`3 / 6`). |
| Handwritten notes | **Architects Daughter**, 13–16px, `--accent` | Short margin notes only ("next in 25 min", "pace slipping since Oct"). Never for data. |

Self-host the three fonts (no runtime Google Fonts dependency).

## Spacing and layout

- Desktop page padding: 40px top/bottom, 56px left/right. Content max-width follows the window; reference frames are 1280×800.
- Header row: 28px tall, then a `--ink-30` rule 12px below.
- Mobile: 22px side padding, 390px reference width, scroll area above a 78px tab bar.
- Rows: 40–64px tall with `--ink-14` bottom dividers. Touch targets at least 44px.
- Gaps: 10/14/18/22/26/40/56px. Grids use `gap`, never margins between siblings.

## Drawing primitives

Build these once (React components in `src/components/ui/`, plus a CSS layer for
Astro-only and Go templ pages) and use them everywhere.

| Primitive | Spec | Reference |
| --- | --- | --- |
| **Box** | 1px `--ink` border whose lines run **3px past each corner** (2px for small boxes like checkboxes). Implement with `::before` (top/bottom, extended left/right) and `::after` (left/right, extended top/bottom). No radius. | `.bx` in any reference file |
| **Measurement line** | 1px `--ink-60` horizontal line with 10px end ticks skewed −45° (architectural slashes). Used for spans, durations, progress baselines, day windows. | `.dim` |
| **Hatch fill** | Diagonal: `repeating-linear-gradient(-45deg, var(--ink) 0 0.8px, transparent 0.8px 4px)`. Crosshatch: same plus 45°. Dots: `radial-gradient(var(--ink) 0.9px, transparent 1.2px) 0 0 / 4px 4px`. Progress fills sit on a measurement line. | goals, bars |
| **Goal patterns** | With no color, goals are told apart by pattern: diagonal (first goal / LeetCode), crosshatch (second / System design), dots (third / Alloy). Shown as a 6–8px strip on the left of schedule blocks. Assign by goal order; fall back to outline. | Schedule |
| **Tick scale** | Row of 1px ticks (every 5th taller), filled ticks 2px solid `--ink`, rest `--ink-30`. Used for streak (0–record), checklist steps, chapters. | Home, Leetgrinder, Companies |
| **Switch** | 42×22 box, 14px knob. On: hatch-filled track, solid knob right. Off: empty track, outline knob left. Locked: dashed track. `role="switch"`. | Settings |
| **Checkbox** | 16–18px small Box. Checked: an X drawn corner to corner. The next item is circled with a 38px `--accent` ring. Hit area 44px. | Home |
| **Segmented tabs** | Bordered group, active segment filled `--ink` with `--paper` text, mono 11px labels. | Companies, People, Journal |
| **Button** | 44px tall, mono 11px uppercase, letter-spacing 0.16em. Primary: solid `--ink` / `--paper` text. Secondary: `--ink-50` border. Icon buttons 34–44px square. Delete uses `--accent`. | |
| **Search field** | 44px (34px in header), `--ink-50` border, thin magnifier icon, Inter Tight 300 placeholder. | |
| **Field** | Mono label over a 44px borderless input with a 1px `--ink` underline, Inter Tight 300 18px (16px below 1024px). Focus thickens the underline. | Login (setup step 5 will reuse it) |
| **Section header** | Uppercase heading left, mono label right, `--ink` 1px rule under table headers. | |
| **Now line** | 1px `--accent` horizontal line with handwritten "now 5:20 PM". Today markers use a dash-dot `--accent` vertical line. | Timeline, Schedule |
| **Logo spot** | 36–40px square, `--ink-50` border, company logo in single-color ink (fallback: two-letter monogram). | Companies |
| **Avatar** | 28–46px circle, 1px border, initials. | People |

Icons are thin line drawings (1–1.2px stroke, `currentColor`), never emoji or filled icons.

### Using the primitives

- React: `import { Box, Switch, ... } from '../components/ui'` (`src/components/ui/index.ts`).
  Pure helpers (goal pattern by order, tick layout, monogram, tab keys, `buttonClassName`, `resolveUrl`, `cssUrl`) are in `src/lib/ui.ts`.
- CSS: `public/ui/primitives.css`, prefixed `ui-`. Main classes (most have modifiers, see the file): `.ui-box`, `.ui-box--sm`, `.ui-dim`, `.ui-dim--plan`,
  `.ui-dim--late`, `.ui-hatch--diagonal|crosshatch|dots`, `.ui-goal-strip`, `.ui-goal-swatch`, `.ui-progress`,
  `.ui-ticks`, `.ui-label`, `.ui-section-header`, `.ui-btn` + `--primary|--danger|--icon|--sm`, `.ui-switch`,
  `.ui-check`, `.ui-tabs`, `.ui-search`, `.ui-field`, `.ui-now`, `.ui-today`, `.ui-note`, `.ui-logo`, `.ui-avatar`, `.ui-icon`,
  `.ui-sr-only`. Astro bundles it into `@layer components` (Tailwind utilities still override it); Go templ
  pages should link `/ui/tokens.css` and `/ui/primitives.css` (public, no login needed) and copy the markup from
  the React components. Some primitives are compound: the checkbox is `.ui-check.ui-box.ui-box--sm` around
  the input, the X svg and the ring; the switch is `.ui-switch > .ui-switch__track > .ui-switch__knob`.
- Stateful markup is styled from ARIA and native state, so a templ page gets the same look from the same
  attributes: `.ui-switch[aria-checked]`, `.ui-tabs__tab[aria-selected]`, `.ui-check__input:checked`, and
  `:disabled` for a locked switch, a disabled checkbox and disabled buttons (set the `disabled` attribute).
- Fonts are not in those two files: Astro pages get them from the bundle; a templ page links `/ui/fonts.css`
  (Latin copies of the same @fontsource files in `public/ui/fonts/`, checked by `tests/fonts.test.ts`) and sets
  `--font-sans`/`--font-mono`/`--font-hand` itself, as `internal/leetgrinder/style.css` does.
- A review gallery of every primitive and its main states: `/dev/ui`, served only when `APP_ENV` is not `production` (login still required).

## Status language

| State | Treatment |
| --- | --- |
| Done | 45% opacity, line-through, X in checkbox |
| Next / current | Larger text, `--accent` ring or handwritten note |
| Late / past end date | `--accent` text "LATE", measurement line in `--accent` |
| Planned | Dashed measurement line |
| Skipped / canceled | `--accent` outline (skipped) or dashed `--ink-30` outline (canceled), line-through |
| Off module | Line-through tags, "paused · N items kept" |

## Formatting

- Times: 12-hour, no leading zero, space before AM/PM: `7:15 AM`, `5:45 PM`, ranges `9 AM – 4 PM` (drop `:00` in tight places).
- Dates: `Nov 24`, `Tue 24 Nov`, or `24.11.26` only for the large Home date.
- Counts: `3 / 6`, `16 / 10.5 h`.
- Shared formatters: `src/lib/format.ts` for the frontend (`formatTime`, `formatTimeRange`, `formatDate`, `formatWeekdayDate`, `formatNumericDate`; pass `{ compact: true }` to drop `:00`) and `internal/timefmt` for Go templ pages (`Time`, `CompactTime`, `Range`, `CompactRange`, `Date`, `WeekdayDate`, `NumericDate`). Pages switch to these as they are migrated; use them for on-screen labels instead of ad hoc `toLocaleTimeString` or `Format("15:04")` calls; values that are parsed back or serialized (form inputs, API fields, `clockLabel`) keep their 24-hour formats.

## Accessibility

- Real `<button>`, `<a>`, `<input>` + `<label>`; `role="switch"`/`tablist`/`tab` where shown.
- Labels at `--ink-60` must still meet 4.5:1 on every theme; check Vellum and Cobalt especially.
- State is never color-only: late also says LATE, done is also struck through.
