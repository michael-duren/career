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

## Notes on the build

- One row per thought (a thought groups takes recorded within 90 minutes). The row's play button fetches the thought's takes and plays them in order; pressing it again stops, and starting another row stops the first. Thoughts typed in the editor have no audio, so their play button reports that and disables itself.
- A row shows a title the user chose before the transcript; generated titles (the first words) aren't repeated. Rows without a transcript yet show "Waiting for transcription…".
- Search covers title, tags and the transcript excerpt the list API returns (the first 240 characters).
- Eight rows show before "+ N OLDER"; searching shows every match.
- Kept from the old page, restyled: Import memo, Sync now (retry uploads), the delete button on each row (with the unsent-takes check), "+ Write" for a typed thought, the "Share audio thoughts context explicitly" link, the screen-on hint, and the reader's takes panel (status, audio player, original transcript, Retranscribe).
- Dropped: the studio-style recorder (timeline lane with ruler, segmented level meters, LEDs, codec / bitrate / input dB readout, R / I indicators), the fixed bottom dock on phones (the tab bar lives there now), and the list's Export all data and Reload latest buttons (Reload latest shows when the list fails to load and stays in the reader).
- The waveform is the last 73 level samples (one per 100 ms) drawn as a line that swings with the input; it stays flat and dim when idle.

## Acceptance criteria

- [x] Matches the reference in all four themes at 1280×800 and 390×844 (tokens only, no raw colors).
- [x] Uses the shared primitives from `design-system.md`; times are 12-hour.
- [x] Hidden (and 404) when its module is off.
- [x] Recording, offline queue and retranscribe behave exactly as today.
- [x] Waveform reflects live input level while recording.
