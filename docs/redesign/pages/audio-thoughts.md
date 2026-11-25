# Audio thoughts

**Current code:** src/pages/audio-thoughts.astro → RunningRecorder.tsx, WorkspaceEditor
**Reference:** desktop `reference/desktop/ARC-Audio.html` · mobile `reference/mobile/M-Audio.html`

## Purpose

Record voice memos (works offline, queued), transcribed on the homelab.

## Data

`/api/running/*` (clips, status, retranscribe, audio), offline queue in `running-queue.ts`.

## Desktop

- Header: "7 recordings · transcribed on your homelab", title, search transcripts.
- Left: drawn target (circle, dashed inner circle, crosshair) with a round record button (accent dot), "TAP TO RECORD", waveform line while recording, Waiting to upload / Transcriber stats, accent note "works offline — uploads when you're back".
- Right: RECORDINGS list: date + 12-hour time, two-line transcript, play button; "+ N OLDER".

## Mobile

Smaller drawn target and record button, queue line, search, recordings list.

## Acceptance criteria

- [ ] Matches the reference in all four themes at 1280×800 and 390×844 (tokens only, no raw colors).
- [ ] Uses the shared primitives from `design-system.md`; times are 12-hour.
- [ ] Hidden (and 404) when its module is off.
- [ ] Recording, offline queue and retranscribe behave exactly as today.
- [ ] Waveform reflects live input level while recording.
