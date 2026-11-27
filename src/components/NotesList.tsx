import type { MouseEvent } from 'react';
import type { Note } from '../lib/workspace';
import { noteCardDate, noteStats, topicCounts } from '../lib/note-card';
import { cn } from '../lib/utils';
import { Box, Button, IconButton, Label, Progress, SearchField, SectionHeader } from './ui';

export const noteHref = (note: Pick<Note, 'id'>) => `/notes/${note.id.split('/').map(encodeURIComponent).join('/')}`;

type Props = {
  /** null while the list is loading. */
  notes: Note[] | null;
  /** The notes to show: already filtered by topic and search, in display order. */
  shown: Note[];
  topic: string | null;
  onTopic: (topic: string | null) => void;
  query: string;
  onQuery: (query: string) => void;
  onOpen: (note: Note) => void;
  onCreate: () => void;
  busy: boolean;
  /** The list failed to load: say so and offer a retry instead of "Loading…". */
  failed: boolean;
  onReload: () => void;
};

const plural = (count: number, word: string) => `${count} ${word}${count === 1 ? '' : 's'}`;

/** Words, then the todo progress (hatch on a measurement line + "done/total") when the note has todos. */
function NoteFooter({ note }: { note: Note }) {
  const stats = noteStats(note);
  return <span className="mt-auto flex items-center justify-between gap-3">
    <span className="font-mono text-[10px] text-ink-60">{plural(stats.words, 'word')}</span>
    {stats.todos > 0 && <span className="flex items-center gap-2">
      <Progress value={stats.todosDone} max={stats.todos} label={`${note.title} todos done`} className="w-[70px]" />
      <span className="font-mono text-[10px]">{stats.todosDone}/{stats.todos}</span>
    </span>}
  </span>;
}

/**
 * The Notes list (docs/redesign/pages/notes.md): topics and a card grid on desktop,
 * topic chips and rows on phones. Cards are links to the note's page, but a plain click
 * opens the note in place so the editor keeps its draft guard and list position.
 */
export function NotesList({ notes, shown, topic, onTopic, query, onQuery, onOpen, onCreate, busy, failed, onReload }: Props) {
  const topics = topicCounts(notes ?? []);
  const total = notes?.length ?? 0;
  const summary = notes ? `${plural(total, 'note')} · ${plural(topics.length, 'topic')}` : failed ? 'Notes unavailable' : 'Loading notes…';
  const retry = !notes && failed && <Button disabled={busy} onClick={onReload}>Reload latest</Button>;
  const open = (note: Note) => (event: MouseEvent<HTMLAnchorElement>) => {
    // While saving or deleting, cards don't open at all (not even in a new tab).
    if (busy) { event.preventDefault(); return; }
    if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
    event.preventDefault();
    onOpen(note);
  };
  const choices: { name: string | null; label: string; count: number }[] = [{ name: null, label: 'All', count: total }, ...topics.map(t => ({ name: t.name, label: t.name, count: t.count }))];
  const empty = notes && shown.length === 0 && <p className="py-6 text-sm text-ink-60">{total ? 'No notes match. Try another topic or fewer words.' : 'No notes yet. Start one with + New note.'}</p>;

  return <>
    {/* Desktop and tablet */}
    <div className="max-md:hidden">
      <header className="flex items-end justify-between gap-6">
        <div className="flex flex-col gap-2"><Label>{summary}</Label><h1 className="m-0 font-extralight !text-[52px] !leading-[0.9]">Notes</h1></div>
        <div className="flex gap-2.5">
          {retry}
          <SearchField label="Search notes" value={query} onChange={event => onQuery(event.target.value)} className="w-[240px]" />
          <Button variant="primary" disabled={!notes || busy} onClick={onCreate}>+ New note</Button>
        </div>
      </header>
      <div className="mt-[30px] grid grid-cols-[170px_minmax(0,1fr)] gap-10 lg:grid-cols-[200px_minmax(0,1fr)] lg:gap-12">
        <nav aria-label="Topics" className="flex flex-col">
          <h2 className="mb-2.5 text-xs font-medium tracking-[0.24em]">TOPICS</h2>
          {notes && choices.map(choice => {
            const active = choice.name === topic;
            return <button key={choice.name ?? '\u0000all'} type="button" aria-pressed={active} onClick={() => onTopic(choice.name)}
              className={cn('flex min-h-[34px] items-center justify-between gap-3 border-0 border-l border-solid bg-transparent py-1 pr-0 text-left text-sm text-ink hover:bg-ink-05',
                active ? 'border-ink pl-2.5 font-normal' : 'border-transparent pl-2.5 font-light')}>
              <span className="min-w-0 truncate">{choice.label}</span><span className="font-mono text-[11px] text-ink-60">{choice.count}</span>
            </button>;
          })}
        </nav>
        <section aria-labelledby="notes-heading" className="flex min-w-0 flex-col">
          <SectionHeader id="notes-heading" title="RECENTLY UPDATED" meta={<span data-testid="notes-topic">{topic ?? 'All topics'}</span>} className="mb-3.5" />
          {empty}
          <ul className="m-0 grid list-none grid-cols-1 gap-x-[26px] gap-y-5 p-[3px] lg:grid-cols-2 xl:grid-cols-3">
            {shown.map(note => <li key={note.id} className="min-w-0">
              <Box className="h-full">
                <a href={noteHref(note)} onClick={open(note)} aria-disabled={busy || undefined}
                  className="flex h-full min-h-[150px] flex-col gap-1.5 px-4 py-3.5 text-ink no-underline hover:bg-ink-05 hover:text-ink">
                  <span className="flex justify-between gap-3"><Label className="truncate !text-[9px]">{note.topic}</Label><span className="shrink-0 font-mono text-[10px] text-ink-60">{noteCardDate(note.updatedAt)}</span></span>
                  <span className="line-clamp-2 text-[19px] font-light leading-[1.15]">{note.title}</span>
                  {note.description && <span className="line-clamp-2 text-xs font-light text-ink-60">{note.description}</span>}
                  <NoteFooter note={note} />
                </a>
              </Box>
            </li>)}
          </ul>
        </section>
      </div>
    </div>

    {/* Phone */}
    <div className="md:hidden">
      <header className="flex items-end justify-between gap-4">
        <div className="flex flex-col gap-1.5"><Label>{summary}</Label><h1 className="m-0 font-extralight !text-[40px] !leading-[0.9]">Notes</h1></div>
        <span className="flex gap-2">{retry}<IconButton label="New note" variant="primary" disabled={!notes || busy} onClick={onCreate} className="!text-lg">+</IconButton></span>
      </header>
      <SearchField label="Search notes" value={query} onChange={event => onQuery(event.target.value)} className="mt-4" />
      {notes && <div role="group" aria-label="Topics" className="-mx-[22px] mt-3 flex gap-2 overflow-x-auto px-[22px] pb-1">
        {choices.map(choice => {
          const active = choice.name === topic;
          return <button key={choice.name ?? '\u0000all'} type="button" aria-pressed={active} onClick={() => onTopic(choice.name)}
            className={cn('h-9 flex-none border border-solid px-3 font-mono text-[10px] font-medium uppercase tracking-[0.08em] whitespace-nowrap',
              active ? 'border-ink bg-ink text-paper' : 'border-ink-30 bg-transparent text-ink')}>
            {choice.label} {choice.count}
          </button>;
        })}
      </div>}
      <ul className="m-0 mt-2 flex list-none flex-col p-0">
        {shown.map(note => <li key={note.id}>
          <a href={noteHref(note)} onClick={open(note)} aria-disabled={busy || undefined}
            className="flex flex-col gap-1.5 border-0 border-b border-solid border-ink-14 py-3.5 text-ink no-underline hover:text-ink">
            <span className="flex justify-between gap-3"><Label className="truncate !text-[9px]">{note.topic}</Label><span className="shrink-0 font-mono text-[10px] text-ink-60">{noteCardDate(note.updatedAt)}</span></span>
            <span className="text-[17px] font-light leading-[1.2]">{note.title}</span>
            <NoteFooter note={note} />
          </a>
        </li>)}
      </ul>
      {empty}
    </div>
  </>;
}
