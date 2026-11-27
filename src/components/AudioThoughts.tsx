import { useEffect, useRef, useState, type ReactNode, type Ref } from 'react';
import type { RunningNote } from '../lib/workspace';
import { isAutoTitle, recordingStamp, thoughtExcerpt, thoughtTitle } from '../lib/audio-thoughts';
import { cn } from '../lib/utils';
import { Button, DeleteIcon, IconButton, Label, SearchField, SectionHeader } from './ui';

/** Rows shown before "+ N OLDER" (all matches show while searching). */
export const RECENT_RECORDINGS = 8;

const plural = (count: number, word: string) => `${count} ${word}${count === 1 ? '' : 's'}`;
const clipAudio = (noteId: string, clipId: string) => `/api/running/clips/${encodeURIComponent(noteId)}/${clipId}/audio`;

/** The row playing now, so starting another row stops it. */
let nowPlaying: (() => void) | null = null;

type PlayState = 'idle' | 'loading' | 'playing' | 'none' | 'error';

/** Plays a thought's takes in order; pressing it again stops. */
function PlayButton({ noteId, name }: { noteId: string; name: string }) {
  const [state, setState] = useState<PlayState>('idle');
  const stopRef = useRef<(() => void) | null>(null);
  useEffect(() => () => stopRef.current?.(), []);
  async function toggle() {
    if (stopRef.current) { stopRef.current(); return; }
    nowPlaying?.();
    setState('loading');
    try {
      const response = await fetch(`/api/running/${encodeURIComponent(noteId)}/status`, { cache: 'no-store' });
      // Thoughts typed in the editor have no takes.
      if (response.status === 404) { setState('none'); return; }
      if (!response.ok) throw new Error('status');
      const clips: { clipId: string }[] = (await response.json()).clips ?? [];
      if (!clips.length) { setState('none'); return; }
      const audio = new Audio();
      let index = 0, stopped = false;
      const stop = (next: PlayState = 'idle') => {
        if (stopped) return;
        stopped = true; audio.pause(); audio.removeAttribute('src');
        if (nowPlaying === stop) nowPlaying = null;
        if (stopRef.current === stop) stopRef.current = null;
        setState(next);
      };
      const playNext = () => {
        if (stopped) return;
        if (index >= clips.length) { stop(); return; }
        audio.src = clipAudio(noteId, clips[index++].clipId);
        audio.play().catch(() => stop('error'));
      };
      audio.onended = playNext;
      audio.onerror = () => stop('error');
      nowPlaying = stop; stopRef.current = stop;
      setState('playing'); playNext();
    } catch { setState('error'); }
  }
  const label = state === 'playing' ? `Stop ${name}` : state === 'none' ? `No audio for ${name}` : state === 'error' ? `Audio unavailable for ${name}. Try again` : `Play ${name}`;
  return <button type="button" aria-label={label} title={label} aria-pressed={state === 'playing'} disabled={state === 'none' || state === 'loading'} onClick={() => void toggle()}
    className="flex size-11 shrink-0 items-center justify-center border-0 bg-transparent p-0 text-ink disabled:cursor-not-allowed disabled:opacity-40">
    <span aria-hidden="true" className={cn('flex size-[26px] items-center justify-center rounded-full border border-solid', state === 'error' ? 'border-accent text-accent' : 'border-ink-50')}>
      {state === 'playing'
        ? <svg width="8" height="10" viewBox="0 0 8 10"><path d="M1.5 1 V9 M6.5 1 V9" fill="none" stroke="currentColor" strokeWidth="1.2" /></svg>
        : <svg width="8" height="10" viewBox="0 0 8 10" className={state === 'loading' ? 'animate-pulse' : ''}><path d="M1 1 L7 5 L1 9 Z" fill="none" stroke="currentColor" strokeWidth="1" /></svg>}
    </span>
  </button>;
}

function RecordingRow({ thought, busy, onOpen, onDelete }: { thought: RunningNote; busy: boolean; onOpen: () => void; onDelete: () => void }) {
  const { date, time } = recordingStamp(thought.startedAt);
  const excerpt = thoughtExcerpt(thought);
  // Transcribed thoughts are named after their first words; only a title the user chose is worth repeating.
  const titled = !isAutoTitle(thought.title) && thought.title !== thoughtTitle(excerpt);
  const name = titled ? thought.title : `thought from ${date} at ${time}`;
  return <li className="flex items-start gap-1 border-0 border-b border-solid border-ink-14 hover:bg-ink-05">
    <button type="button" data-thought-open={thought.id} disabled={busy} onClick={onOpen} aria-label={`Open ${name}`}
      className="flex min-w-0 flex-1 flex-col gap-[5px] border-0 bg-transparent px-0 py-3.5 text-left text-ink disabled:cursor-not-allowed md:grid md:grid-cols-[130px_minmax(0,1fr)] md:gap-5 md:py-4">
      <span className="flex gap-1 font-mono text-[11px] md:flex-col md:gap-0.5 md:text-xs">
        <span className="text-ink-60 md:text-ink">{date}</span>
        <span className="text-ink-60"><span className="md:hidden">· </span>{time}</span>
      </span>
      {excerpt || titled
        ? <span className="line-clamp-2 text-[15px] font-light leading-[1.45] md:leading-[1.5]">
          {titled && <span className="font-normal">{thought.title}{excerpt ? ' — ' : ''}</span>}{excerpt}
        </span>
        : <span className="inline-flex items-center gap-2 text-sm font-light text-ink-60"><span className="size-1.5 animate-pulse rounded-full bg-accent" />Waiting for transcription…</span>}
    </button>
    <span className="flex shrink-0 items-center pt-1.5 md:pt-2.5">
      <PlayButton noteId={thought.id} name={name} />
      <button type="button" disabled={busy} onClick={onDelete} aria-label={`Delete ${name}`} title="Delete thought"
        className="flex size-11 items-center justify-center border-0 bg-transparent p-0 text-ink-60 hover:text-accent disabled:cursor-not-allowed disabled:opacity-40">
        <DeleteIcon />
      </button>
    </span>
  </li>;
}

type Props = {
  /** null while the list is loading. */
  thoughts: RunningNote[] | null;
  /** Thoughts matching the search, newest first. */
  shown: RunningNote[];
  query: string;
  onQuery: (query: string) => void;
  onOpen: (thought: RunningNote) => void;
  onDelete: (thought: RunningNote) => void;
  onCreate: () => void;
  busy: boolean;
  /** The list failed to load: say so and offer a retry instead of "Loading…". */
  failed: boolean;
  onReload: () => void;
  /** A thought is open: only the recorder slot stays (it hides itself unless recording needs attention). */
  reading: boolean;
  /** Focus target for the empty state after the last visible thought is deleted. */
  emptyRef: Ref<HTMLDivElement>;
  /** The recorder. It stays mounted while a thought is open so a take in progress keeps recording. */
  recorder: ReactNode;
};

/**
 * The Audio thoughts page (docs/redesign/pages/audio-thoughts.md): header with transcript search,
 * the drawn record target on the left and the RECORDINGS list on the right (stacked below 1024px).
 */
export function AudioThoughts({ thoughts, shown, query, onQuery, onOpen, onDelete, onCreate, busy, failed, onReload, reading, emptyRef, recorder }: Props) {
  const [expanded, setExpanded] = useState(false);
  const total = thoughts?.length ?? 0;
  const searching = query.trim() !== '';
  const visible = expanded || searching ? shown : shown.slice(0, RECENT_RECORDINGS);
  const older = shown.length - visible.length;
  const summary = thoughts ? plural(total, 'recording') : failed ? 'Recordings unavailable' : 'Loading recordings…';
  const retry = !thoughts && failed && <Button disabled={busy} onClick={onReload}>Reload latest</Button>;
  const search = (className: string) => <SearchField label="Search transcripts" value={query} onChange={event => onQuery(event.target.value)} className={className} />;

  return <>
    <header hidden={reading} className="flex items-end justify-between gap-4 md:gap-6">
      <div className="flex flex-col gap-1.5 md:gap-2">
        <Label>{summary}<span className="max-md:hidden">{thoughts ? ' · transcribed on your homelab' : ''}</span></Label>
        <h1 className="m-0 font-extralight !text-[40px] !leading-[0.9] md:!text-[52px]">Audio thoughts</h1>
      </div>
      <div className="flex gap-2.5">
        {retry}
        {search('w-[260px] max-md:hidden')}
        <Button className="max-md:hidden" disabled={!thoughts || busy} onClick={onCreate}>+ Write</Button>
        <IconButton label="Write a thought" disabled={!thoughts || busy} onClick={onCreate} className="!text-lg md:hidden">+</IconButton>
      </div>
    </header>
    <div className={reading ? '' : 'mt-6 grid gap-8 md:mt-[30px] lg:grid-cols-[340px_minmax(0,1fr)] lg:gap-12 xl:grid-cols-[400px_minmax(0,1fr)] xl:gap-16'}>
      <div className="min-w-0">{recorder}</div>
      <section hidden={reading} aria-labelledby="recordings-heading" className="flex min-w-0 flex-col">
        {search('md:hidden')}
        <SectionHeader id="recordings-heading" title="RECORDINGS" meta="Newest first" className="mb-2 max-md:mt-5" />
        {thoughts && shown.length === 0 && <div ref={emptyRef} tabIndex={-1} className="py-6 text-sm font-light text-ink-60">
          {total ? 'No recordings match. Try fewer words, or clear the search.' : 'No audio thoughts yet. Tap record: each take shows up here and is named after its first words once it is transcribed.'}
        </div>}
        <ul className="m-0 list-none p-0">
          {visible.map(thought => <RecordingRow key={thought.id} thought={thought} busy={busy} onOpen={() => onOpen(thought)} onDelete={() => onDelete(thought)} />)}
        </ul>
        {older > 0 && <button type="button" onClick={() => setExpanded(true)}
          className="mt-3.5 min-h-11 self-start border-0 bg-transparent p-0 font-mono text-[11px] tracking-[0.14em] text-ink-60 hover:text-ink">+ {older} OLDER</button>}
        <a className="mt-4 self-start text-xs font-light text-ink-60 underline hover:text-ink" href="/api/agent-context?format=markdown&journal=running">Share audio thoughts context explicitly</a>
      </section>
    </div>
  </>;
}
