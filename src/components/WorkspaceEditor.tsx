import { useEffect, useRef, useState } from 'react';
import { RunningClips } from './RunningRecorder';
import MarkdownPreview from './MarkdownPreview';
import type { JournalWeek, WorkspaceEntry as Entry, EntryKind, NoteTodo } from '../lib/workspace';
import { getEntryId as entryId, getEntryTitle as entryTitle } from '../lib/workspace';
import EntryFields, { EntryChecklist } from './EntryFields';
import { NoteTodos, TagInput } from './NoteInputs';
import { appendDailyEntry, localDate, newWeek } from '../lib/workspace';
import type { Note, RunningNote } from '../lib/workspace';
import { THOUGHT_PLACEHOLDER, isAutoTitle, thoughtDate, thoughtExcerpt, thoughtTitle } from '../lib/audio-thoughts';
import { AudioLines, CalendarPlus, Clock, ListChecks, PencilLine } from 'lucide-react';
import { entryTone, tagChipClass } from '../lib/tag-colors';
import { noteDate, noteStats } from '../lib/note-card';

const field = 'w-full rounded-lg border border-zinc-700 bg-zinc-950 px-3 py-2 text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500';
const button = 'rounded-lg border border-zinc-700 px-3 py-2 text-sm hover:bg-zinc-800 disabled:opacity-50 disabled:cursor-not-allowed';
const primary = `${button} bg-blue-600 border-blue-500 text-white hover:bg-blue-500`;
const labels = { run: 'audio thought', personal: 'personal journal entry', note: 'note', week: 'week', book: 'book or course', company: 'company', document: 'page' };

type EntryResult = { entry: Entry; revision: string };
async function request(kind: EntryKind, method = 'GET', input?: unknown, id?: string): Promise<any> {
  if (method === 'GET' && !id) {
    const entries: EntryResult[] = [];
    let offset = 0;
    do {
      const response = await fetch(`/api/entries/${kind}?limit=100&offset=${offset}`, { cache: 'no-store' });
      if (response.status === 401) { location.assign(`/login?redirect=${encodeURIComponent(location.pathname + location.search)}`); throw new Error('Your session expired. Sign in again.'); }
      const page = await response.json();
      if (!response.ok) throw new Error(page.error || 'Request failed. Please retry.');
      entries.push(...page.entries); offset = page.nextOffset ?? -1;
    } while (offset >= 0);
    return { entries };
  }
  const query = id ? `?id=${encodeURIComponent(id)}` : '';
  const response = await fetch(`/api/entries/${kind}${method === 'GET' ? query : ''}`, method === 'GET' ? { cache: 'no-store' } : {
    method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(input),
  });
  if (response.status === 401) { location.assign(`/login?redirect=${encodeURIComponent(location.pathname + location.search)}`); throw new Error('Your session expired. Sign in again.'); }
  const result = await response.json();
  if (!response.ok) throw new Error(result.error || 'Request failed. Please retry.');
  return result;
}

function draftKey(kind: string) {
  return `career-workspace:v1:${localStorage.getItem('auth_username') || 'user'}:${kind}`;
}
function Preview({ body }: { body: string }) {
  return <MarkdownPreview body={body} />;
}
function Transcribing({ label }: { label: string }) {
  return <span className="inline-flex items-center gap-2 text-sm text-zinc-500"><span className="size-1.5 animate-pulse rounded-full bg-amber-400" />{label}</span>;
}
function ThoughtCard({ thought, selected, disabled, onOpen }: { thought: RunningNote; selected: boolean; disabled: boolean; onOpen: () => void }) {
  const excerpt = thoughtExcerpt(thought), untitled = isAutoTitle(thought.title);
  return <button type="button" disabled={disabled} onClick={onOpen} className={`group flex w-full flex-col gap-2 rounded-xl border p-4 text-left transition-colors ${selected ? 'border-sky-500 bg-sky-950/30' : 'border-zinc-800 bg-zinc-900/60 hover:border-zinc-600 hover:bg-zinc-900'}`}>
    <span className={`block font-medium ${untitled ? 'text-zinc-400 italic' : 'text-zinc-100'}`}>{untitled ? 'Untitled thought' : thought.title}</span>
    {excerpt ? <span className="line-clamp-2 text-sm leading-relaxed text-zinc-400">{excerpt}</span> : <Transcribing label="Waiting for transcription…" />}
    <span className="mt-auto flex flex-wrap items-center gap-2 pt-1 text-xs text-zinc-500">{thoughtDate(thought.startedAt)}{thought.tags.map(tag => <span key={tag} className={`${tagChipClass(tag)} px-2 py-0.5`}>{tag}</span>)}</span>
  </button>;
}
function NoteCard({ note, selected, disabled, onOpen }: { note: Note; selected: boolean; disabled: boolean; onOpen: () => void }) {
  const tone = entryTone(note.tags), stats = noteStats(note);
  const created = noteDate(note.createdAt), updated = noteDate(note.updatedAt);
  return <button type="button" disabled={disabled} onClick={onOpen} data-tone={tone.name} className={`flex w-full flex-col gap-2 rounded-lg border border-l-4 p-3 text-left transition-colors sm:p-4 ${selected ? 'border-blue-500 bg-blue-950/30' : `border-zinc-800 ${tone.card}`}`}>
    <span className="flex items-start justify-between gap-3">
      <span className="min-w-0 font-medium text-zinc-100">{note.title}</span>
      <span className="shrink-0 text-xs text-zinc-500">{note.topic}</span>
    </span>
    {note.description && <span className="line-clamp-2 text-sm leading-relaxed text-zinc-400">{note.description}</span>}
    {note.tags.length > 0 && <span className="flex flex-wrap gap-1.5">{note.tags.map(tag => <span key={tag} className={`${tagChipClass(tag)} px-2 py-0.5 text-xs`}>{tag}</span>)}</span>}
    <span className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-zinc-500">
      {created && <span className="inline-flex items-center gap-1" title={note.createdAt}><CalendarPlus className="size-3.5" aria-hidden />Created {created}</span>}
      {updated && updated !== created && <span className="inline-flex items-center gap-1" title={note.updatedAt}><PencilLine className="size-3.5" aria-hidden />Edited {updated}</span>}
      <span className="inline-flex items-center gap-1"><Clock className="size-3.5" aria-hidden />{stats.words ? `${stats.words.toLocaleString()} words · ${stats.minutes} min read` : 'Empty'}</span>
      {stats.todos > 0 && <span className="inline-flex items-center gap-1"><ListChecks className="size-3.5" aria-hidden />{stats.todosDone}/{stats.todos} todos</span>}
    </span>
  </button>;
}
export function WorkspaceEditor({ kind, initialId }: { kind: EntryKind; initialId?: string }) {
  const listFirst = kind === 'note' || kind === 'week' || kind === 'personal' || kind === 'run';
  const openedInitialEntry = useRef(false);
  const selectionRequest = useRef(0);
  const todoQueue = useRef<{ running: boolean; pending: NoteTodo[] | null; waiters: ((ok: boolean) => void)[] }>({ running: false, pending: null, waiters: [] });
  const backButton = useRef<HTMLButtonElement>(null);
  const listPosition = useRef<{ scroll: number; focus: HTMLElement | null } | null>(null);
  const recovered = useRef(false);
  const [entries, setEntries] = useState<Entry[] | null>(null);
  const [entry, setEntry] = useState<Entry | null>(null);
  const [baseRevision, setBaseRevision] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const [selectedTopics, setSelectedTopics] = useState<string[] | null>(null);
  const initializedTopics = useRef(false);
  const [preview, setPreview] = useState(false);
  const [editing, setEditing] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');

  async function cancel() {
    const saved = entry && entries ? entries.find(item => entryId(item) === entryId(entry)) : null;
    setBusy(true); setError('');
    try {
      // List responses are summaries; restore the complete saved body and revision.
      const detail = saved ? await request(kind, 'GET', undefined, entryId(saved)) as EntryResult : null;
      setEntry(detail?.entry ?? null);
      setBaseRevision(detail?.revision ?? null);
      setEditing(false); setDirty(false); setPreview(false); setStatus('Cancelled.');
      try { localStorage.removeItem(draftKey(kind)); } catch { /* Optional storage. */ }
    } catch (e) { setError((e as Error).message); }
    finally { setBusy(false); }
  }
  async function load() {
    setError('');
    try {
      const next = await request(kind); const list = next.entries.map((item: EntryResult) => item.entry) as Entry[]; setEntries(list);
      const pathID = kind === 'note' && window.location.pathname.startsWith('/notes/') ? decodeURIComponent(window.location.pathname.slice('/notes/'.length)) : null;
      const requested = initialId || pathID || new URLSearchParams(window.location.search).get('id');
      if (kind === 'note' && requested && !initializedTopics.current) {
        const linkedNote = list.find(note => entryId(note) === requested);
        if (linkedNote && 'topic' in linkedNote) setSelectedTopics([linkedNote.topic]);
        initializedTopics.current = true;
      }
      if (requested && !openedInitialEntry.current && !recovered.current) {
        openedInitialEntry.current = true;
        const selection = ++selectionRequest.current;
        const found = await request(kind, 'GET', undefined, requested).catch(() => null) as EntryResult | null;
        if (selection !== selectionRequest.current || recovered.current) return;
        if (found) { setEntry(found.entry); setBaseRevision(found.revision); }
        else setError('Entry not found. Choose another entry or create a new one.');
      }
    } catch (e) { setError((e as Error).message); }
  }
  useEffect(() => {
    void load();
    const refresh = () => { void load(); };
    window.addEventListener('workspace-saved', refresh);
    return () => window.removeEventListener('workspace-saved', refresh);
  }, []);
  useEffect(() => {
    try {
      const saved = localStorage.getItem(draftKey(kind));
      if (saved) {
        const draft = JSON.parse(saved);
        if (draft.entry && typeof draft.entry.body === 'string' && Array.isArray(draft.entry.tags)) {
          recovered.current = true;
          setEntry(draft.entry); setBaseRevision(draft.revision);
          setEditing(true); setDirty(true); setStatus('Recovered your unsaved draft.');
        }
      }
    } catch { setStatus('Draft recovery is unavailable in this browser.'); }
  }, [kind]);
  useEffect(() => {
    if (!dirty || !entry) return;
    const onLeave = (event: BeforeUnloadEvent) => { event.preventDefault(); };
    window.addEventListener('beforeunload', onLeave);
    try { localStorage.setItem(draftKey(kind), JSON.stringify({ entry, revision: baseRevision })); }
    catch { setStatus('Local draft could not be saved. Keep this tab open until you save.'); }
    return () => { window.removeEventListener('beforeunload', onLeave); };
  }, [entry, dirty, baseRevision, kind]);

  function choose(next: Entry, edit = false) {
    if (dirty && !window.confirm('Discard the unsaved draft and open this entry?')) return;
    if (listFirst && !entry) listPosition.current = { scroll: window.scrollY, focus: document.activeElement instanceof HTMLElement ? document.activeElement : null };
    const selection = ++selectionRequest.current;
    try { localStorage.removeItem(draftKey(kind)); } catch { /* Storage is optional. */ }
    if (edit && !entries?.some(item => entryId(item) === entryId(next))) {
      setEntry(next); setBaseRevision(null);
    } else {
      void request(kind, 'GET', undefined, entryId(next)).then((detail: EntryResult) => { if (selection !== selectionRequest.current) return; setEntry(detail.entry); setBaseRevision(detail.revision); }).catch(e => setError(e instanceof Error ? e.message : 'Entry could not be loaded.'));
    }
    setEditing(edit); setDirty(false); setPreview(false); setStatus(''); setError('');
  }
  function backToList() {
    if (dirty && !window.confirm('Discard the unsaved draft and return to the list?')) return;
    selectionRequest.current++;
    openedInitialEntry.current = true;
    setEntry(null); setBaseRevision(null); setEditing(false); setDirty(false);
    setPreview(false); setError(''); setStatus('');
    try { localStorage.removeItem(draftKey(kind)); } catch { /* Optional storage. */ }
    const url = new URL(window.location.href);
    url.searchParams.delete('id');
    if (kind === 'note' && url.pathname.startsWith('/notes/')) url.pathname = '/notes';
    window.history.replaceState(window.history.state, '', url);
  }
  function change(patch: Partial<Entry>) {
    setEntry(previous => previous ? { ...previous, ...patch } as Entry : previous);
    setDirty(true); setStatus('Unsaved changes');
  }
  function create() {
    if (!entries) return;
    const id = crypto.randomUUID();
    if (kind === 'run') choose({ id, title: THOUGHT_PLACEHOLDER, runDate: localDate(), startedAt: new Date().toISOString(), tags: [], body: '' }, true);
    else if (kind === 'personal') choose({ id, title: '', date: localDate(), description: '', tags: [], body: '' }, true);
    else if (kind === 'note') choose({ id, title: '', topic: selectedTopics?.length === 1 ? selectedTopics[0] : 'General', description: '', tags: [], body: '' }, true);
    else if (kind === 'book') choose({ slug: id, title: '', authors: [], category: 'Computer Science', type: 'book', status: 'backlog', featured: false, priority: 'medium', tags: [], body: '## Chapters\n\n- [ ] First chapter\n\n## Log\n' }, true);
    else if (kind === 'company') choose({ slug: id, title: '', category: 'Observability / Infra', type: 'company', url: '', status: 'not_started', featured: false, priority: 'medium', tags: [], body: '## Why\n\n## Steps\n\n- [ ] Identify one person at the company to connect with\n- [ ] Reach out and start building a relationship\n- [ ] Research team & open roles\n- [ ] Tailor resume/cover letter\n- [ ] Apply\n\n## Log\n' }, true);
    else if (kind === 'document') choose({ id, title: '', description: '', tags: [], body: '' }, true);
    else {
      const week = newWeek(localDate());
      choose(entries.find(w => 'dates' in w && w.dates === week.dates) ?? week, true);
    }
  }

  async function save() {
    if (!entry) return;
    setBusy(true); setError('');
    try {
      // A thought left untitled is named after its opening words, like transcribed ones.
      const input = 'runDate' in entry && (!entry.title.trim() || isAutoTitle(entry.title)) ? { ...entry, title: thoughtTitle(entry.body) || THOUGHT_PLACEHOLDER } : entry;
      const next = await request(kind, 'POST', { entry: input, revision: baseRevision }) as EntryResult;
      const savedEntry = next.entry as Entry;
      setEntries((current: Entry[] | null) => [...(current ?? []).filter(n => entryId(n) !== entryId(entry)), savedEntry]); setBaseRevision(next.revision); setDirty(false); setEditing(false);
      setEntry(savedEntry);
      try { localStorage.removeItem(draftKey(kind)); } catch { /* Optional draft storage. */ }
      setStatus('Saved.');
      window.dispatchEvent(new Event('workspace-saved'));
    } catch (e) { setError((e as Error).message); }
    finally { setBusy(false); }
  }
  /**
   * Todo changes while reading save at once without disabling the todo controls,
   * so keyboard focus survives. Changes made mid-save are queued and sent with
   * the revision the previous save returned.
   */
  async function saveTodos(todos: NoteTodo[]): Promise<boolean> {
    if (!entry || !('topic' in entry)) return false;
    setEntry(current => current ? { ...current, todos } as Entry : current);
    const queue = todoQueue.current;
    queue.pending = todos;
    // Queued callers learn the outcome of the save that carries their change.
    if (queue.running) return new Promise(resolve => queue.waiters.push(resolve));
    queue.running = true; setBusy(true); setError('');
    let saved: Entry = entry;
    let revision = baseRevision;
    try {
      while (queue.pending) {
        const next = queue.pending; queue.pending = null;
        const result = await request(kind, 'POST', { entry: { ...saved, todos: next }, revision }) as EntryResult;
        saved = result.entry; revision = result.revision;
        setBaseRevision(revision);
      }
      setEntry(saved);
      setEntries(current => [...(current ?? []).filter(item => entryId(item) !== entryId(saved)), saved]);
      setStatus('Saved.');
      window.dispatchEvent(new Event('workspace-saved'));
      queue.waiters.splice(0).forEach(resolve => resolve(true));
      return true;
    } catch (e) {
      queue.pending = null;
      setError((e as Error).message);
      // Reload the saved note so a conflict does not leave a stale revision behind.
      const detail = await request(kind, 'GET', undefined, entryId(entry)).catch(() => null) as EntryResult | null;
      if (detail) { setEntry(detail.entry); setBaseRevision(detail.revision); }
      queue.waiters.splice(0).forEach(resolve => resolve(false));
      return false;
    } finally {
      // Changes made during the failure reload were reverted with it.
      queue.pending = null; queue.running = false; setBusy(false);
    }
  }
  async function remove() {
    if (!entry || !window.confirm(`Delete ${entryTitle(entry)}? This cannot be undone.`)) return;
    setBusy(true); setError('');
    try {
      await request(kind, 'DELETE', { id: entryId(entry), revision: baseRevision });
      setEntries(current => (current ?? []).filter(item => entryId(item) !== entryId(entry)));
      if (listFirst) backToList();
      else { setEntry(null); setDirty(false); }
      setStatus('Deleted.');
      window.dispatchEvent(new Event('workspace-saved'));
      try { localStorage.removeItem(draftKey(kind)); } catch { /* Optional draft storage. */ }
    } catch (e) { setError((e as Error).message); }
    finally { setBusy(false); }
  }
  const loadedEntries = entries ?? [];
  const topicCounts = loadedEntries.reduce((counts, note) => 'topic' in note ? counts.set(note.topic, (counts.get(note.topic) ?? 0) + 1) : counts, new Map<string, number>());
  const topics = [...topicCounts.keys()].sort((a, b) => a.localeCompare(b));
  const tagOptions = [...new Set(loadedEntries.flatMap(item => item.tags))].sort((a, b) => a.localeCompare(b));
  const filtered = [...loadedEntries].filter(e => kind !== 'note' || selectedTopics === null || ('topic' in e && selectedTopics.includes(e.topic))).filter(e => `${entryTitle(e)} ${e.tags.join(' ')} ${'topic' in e ? e.topic : ''}`.toLowerCase().includes(query.toLowerCase()))
    .sort((a, b) => kind === 'run' ? ('startedAt' in b ? b.startedAt : '').localeCompare('startedAt' in a ? a.startedAt : '') : kind === 'personal' ? (('date' in b ? b.date : '') || '').localeCompare(('date' in a ? a.date : '') || '') || (b.updatedAt ?? '').localeCompare(a.updatedAt ?? '') : kind === 'week' ? (b as JournalWeek).dates.localeCompare((a as JournalWeek).dates) : (b.updatedAt ?? '').localeCompare(a.updatedAt ?? '') || entryTitle(a).localeCompare(entryTitle(b)));

  const reading = listFirst && entry !== null;
  useEffect(() => {
    if (reading) {
      backButton.current?.focus();
      backButton.current?.scrollIntoView({ block: 'start' });
    } else if (listPosition.current) {
      listPosition.current.focus?.focus({ preventScroll: true });
      window.scrollTo(0, listPosition.current.scroll);
      listPosition.current = null;
    }
  }, [reading]);

  return <section className="space-y-5" data-thought-reading={kind === 'run' && reading ? '' : undefined}>
    {reading && <div className="flex flex-wrap gap-2">
      <button ref={backButton} type="button" className={button} disabled={busy} onClick={backToList}>← Back to {kind === 'note' ? 'notes' : kind === 'run' ? 'audio thoughts' : 'journal'}</button>
      <button type="button" className={button} disabled={busy} onClick={() => void load()}>Reload latest</button>
    </div>}
    <div hidden={reading}>
      <div className="flex flex-wrap gap-2">
      <button type="button" className={primary} disabled={!entries || busy} onClick={create}>{kind === 'week' ? '+ This week' : `+ New ${labels[kind]}`} </button>
      <a className={button} href="/api/export" download={`career-workspace-${localDate()}.json`}>Export all data</a>
      <button type="button" className={button} disabled={busy} onClick={() => void load()}>Reload latest</button>
      </div>
    </div>
    {error && <div role="alert" className="rounded-lg border border-red-800 bg-red-950/30 p-4 text-red-200">{error} {error.includes('session') && <a className="underline" href="/login">Sign in</a>}
      {error.includes('changed') && <p className="mt-2 text-sm">Copy your draft before reopening an entry. Reloading refreshes the list without replacing your draft.</p>}
    </div>}
    <p role="status" className="text-sm text-zinc-400">{status || (!entries ? (error ? 'Content is unavailable. Use Reload latest to retry.' : 'Loading your content…') : `${loadedEntries.length} entries`)}</p>
    <div hidden={reading} className="space-y-5">
    <label className="block text-sm text-zinc-400">Search {kind === 'week' ? 'journal' : `${labels[kind]} entries`}<input className={`${field} mt-1`} type="search" value={query} onChange={e => setQuery(e.target.value)} placeholder="Search titles, tags, and content" /></label>
    {kind === 'note' && entries && <fieldset className="space-y-2">
      <legend className="mb-2 text-sm text-zinc-400">Topics</legend>
      <div className="flex flex-wrap gap-2">
        <button type="button" className="rounded-full border border-zinc-600 px-4 py-2 text-sm hover:bg-zinc-800 disabled:opacity-50 focus-visible:outline-2 focus-visible:outline-blue-400" disabled={selectedTopics === null} onClick={() => setSelectedTopics(null)}>Select all</button>
        <button type="button" className="rounded-full border border-zinc-600 px-4 py-2 text-sm hover:bg-zinc-800 disabled:opacity-50 focus-visible:outline-2 focus-visible:outline-blue-400" disabled={selectedTopics?.length === 0} onClick={() => setSelectedTopics([])}>Clear all</button>
        <span aria-hidden="true" className="w-px self-stretch bg-zinc-700" />
        {topics.map(name => {
          const selected = selectedTopics === null || selectedTopics.includes(name);
          return <button type="button" key={name} aria-pressed={selected} className={`rounded-full border px-4 py-2 text-sm focus-visible:outline-2 focus-visible:outline-blue-400 ${selected ? 'border-blue-500 bg-blue-600/25 text-blue-200 hover:bg-blue-600/40' : 'border-zinc-700 text-zinc-400 hover:bg-zinc-800'}`} onClick={() => setSelectedTopics(current => {
            const active = current ?? topics;
            return active.includes(name) ? active.filter(topic => topic !== name) : [...active, name];
          })}>{name} <span className="text-xs opacity-70">{topicCounts.get(name)}</span></button>;
        })}
      </div>
      <p className="text-xs text-zinc-400">{filtered.length} of {loadedEntries.length} notes shown · {selectedTopics?.length === 0 ? 'No topics selected. Pick topics or use Select all.' : 'Toggle topics to include or exclude them.'}</p>
    </fieldset>}
    </div>
    <div className={listFirst ? "space-y-5" : "grid gap-5 md:grid-cols-[220px_minmax(0,1fr)]"}>
      <nav hidden={reading} aria-label={`${labels[kind]} entries`} className={kind === 'run' ? 'grid gap-3 sm:grid-cols-2' : listFirst ? "space-y-2" : "max-h-72 overflow-y-auto space-y-2 md:max-h-[700px]"}>
        {kind === 'run' && entries && filtered.length === 0 && <div className="flex flex-col items-center gap-2 rounded-xl border border-dashed border-zinc-700 px-6 py-10 text-center sm:col-span-2">
          <AudioLines className="size-8 text-zinc-600" aria-hidden />
          <p className="font-medium text-zinc-300">{loadedEntries.length ? 'No thoughts match your search' : 'No audio thoughts yet'}</p>
          <p className="max-w-sm text-sm text-zinc-500">{loadedEntries.length ? 'Try fewer words, or clear the search to see everything.' : 'Record a take above. It shows up here and is named after its first few words once it is transcribed.'}</p>
        </div>}
        {kind !== 'run' && entries && filtered.length === 0 && <p className="text-sm text-zinc-400">No matching entries.</p>}
        {kind === 'run' && filtered.map(item => <ThoughtCard key={entryId(item)} thought={item as RunningNote} selected={!!entry && entryId(entry) === entryId(item)} disabled={busy} onOpen={() => choose(item)} />)}
        {kind === 'note' && filtered.map(item => <NoteCard key={entryId(item)} note={item as Note} selected={!!entry && entryId(entry) === entryId(item)} disabled={busy} onOpen={() => choose(item)} />)}
        {kind !== 'run' && kind !== 'note' && filtered.map(item => <button type="button" disabled={busy} key={entryId(item)} onClick={() => choose(item)} className={`w-full rounded-lg border p-3 text-left ${entry && entryId(entry) === entryId(item) ? 'border-blue-500 bg-blue-950/30' : 'border-zinc-800 hover:bg-zinc-900'}`}>
          <span className="block text-sm font-medium">{entryTitle(item)}</span>
          <span className="mt-1 block text-xs text-zinc-400">{'topic' in item ? item.topic : 'week' in item ? (item.hours ? `${Object.values(item.hours as Record<string, number>).reduce((a, b) => a + b, 0)}h logged` : 'Work journal') : 'status' in item ? item.status.replaceAll('_', ' ') : 'description' in item ? item.description : 'runDate' in item ? item.runDate : ''}</span>
        </button>)}
      </nav>
      {entry && <div className="order-first min-w-0 rounded-xl md:order-last border border-zinc-800 bg-zinc-900/40 p-4 sm:p-5">
        {editing ? <form onSubmit={e => { e.preventDefault(); void save(); }} className="space-y-4">
          <fieldset disabled={busy} className="space-y-4 disabled:opacity-60">
            <EntryFields entry={entry} change={change} topics={topics} />
            {(kind === 'book' || kind === 'company') && <EntryChecklist body={entry.body} onChange={body => change({ body })} />}
            <TagInput value={entry.tags} options={tagOptions} onChange={tags => change({ tags })} />
            {'topic' in entry && <NoteTodos todos={entry.todos ?? []} onChange={todos => change({ todos })} />}
            <div className="flex gap-2"><button type="button" className={button} aria-pressed={!preview} onClick={() => setPreview(false)}>Write</button><button type="button" className={button} aria-pressed={preview} onClick={() => setPreview(true)}>Preview</button></div>
            {preview ? <div className="min-h-72 rounded-lg bg-zinc-950 p-4"><Preview body={entry.body} /></div> : <label className="block text-sm">Markdown<textarea className={`${field} mt-1 min-h-80 font-mono text-sm`} maxLength={100000} value={entry.body} onChange={e => change({ body: e.target.value })} placeholder={kind === 'run' ? 'Transcripts land here as each take is processed. Add your own notes around them.' : 'Write your thoughts. Markdown, lists, links, and code blocks are welcome.'} /></label>}
            <button className={primary} type="submit">{busy ? 'Saving…' : 'Save'}</button> <button className={button} type="button" onClick={cancel}>Cancel</button>
          </fieldset>
        </form> : <div className="space-y-4">
          <div className="flex flex-wrap items-start justify-between gap-3"><h2 className={`text-xl font-semibold ${'runDate' in entry && isAutoTitle(entry.title) ? 'text-zinc-400 italic' : ''}`}>{'runDate' in entry && isAutoTitle(entry.title) ? 'Untitled thought' : entryTitle(entry)}</h2><div className="flex gap-2"><button type="button" className={button} disabled={busy} onClick={() => setEditing(true)}>Edit</button>{!(kind === 'document' && ['index', '2026/career-study-plan'].includes(entryId(entry))) && <button type="button" className={`${button} text-red-300`} disabled={busy} onClick={() => void remove()}>Delete</button>}</div></div>
          {kind === 'document' && <a className="inline-block text-sm text-blue-400 hover:underline" href={entryId(entry) === 'index' ? '/' : `/documents/${entryId(entry).split('/').map(encodeURIComponent).join('/')}`}>Open page →</a>}
          <div className="flex flex-wrap gap-2">{entry.tags.map(tag => <span key={tag} className={`${tagChipClass(tag)} px-2 py-1 text-xs`}>{tag}</span>)}</div>
          {'topic' in entry && <NoteTodos todos={entry.todos ?? []} onChange={saveTodos} />}
          {'status' in entry && <p className="text-sm text-zinc-400">{entry.status.replaceAll('_', ' ')} · {entry.priority} priority</p>}
          {'date' in entry && <p className="text-sm text-zinc-400">{entry.date || 'Undated background'}</p>}
          {'startedAt' in entry && <p className="text-sm text-zinc-500">{thoughtDate(entry.startedAt)}</p>}
          {'runDate' in entry && !entry.body.trim()
            ? <div className="rounded-xl border border-dashed border-zinc-700 px-6 py-8 text-center"><Transcribing label="The transcript appears here once your homelab finishes the take." /></div>
            : <Preview body={entry.body} />}
        </div>}
        {kind === 'run' && entry && baseRevision && <RunningClips noteId={entryId(entry)} />}
      </div>}
    </div>
  </section>;
}

export function QuickJournal({ returnTo }: { returnTo?: string } = {}) {
  const [date, setDate] = useState('');
  const [body, setBody] = useState('');
  const [preview, setPreview] = useState(false);
  const [busy, setBusy] = useState(false);
  const [ready, setReady] = useState(false);
  const [status, setStatus] = useState('');
  const [error, setError] = useState('');
  useEffect(() => {
    setDate(localDate());
    try {
      const raw = localStorage.getItem(draftKey('quick'));
      if (raw) { const draft = JSON.parse(raw); if (typeof draft.body === 'string' && typeof draft.date === 'string') { setBody(draft.body); setDate(draft.date); setStatus('Recovered your unsaved draft.'); } }
    } catch { /* Draft recovery is optional. */ }
    setReady(true);
  }, []);
  useEffect(() => {
    if (!ready) return;
    try {
      if (body) localStorage.setItem(draftKey('quick'), JSON.stringify({ date, body }));
      else localStorage.removeItem(draftKey('quick'));
    } catch { setStatus('Local draft recovery unavailable. Keep this tab open until you save.'); }
  }, [body, date, ready]);
  async function save() {
    setBusy(true); setError(''); setStatus('');
    let leaving = false;
    try {
      const candidate = newWeek(date);
      const page = await request('week');
      const summary = page.entries.find((item: EntryResult) => 'dates' in item.entry && item.entry.dates === candidate.dates) as EntryResult | undefined;
      const detail = summary ? await request('week', 'GET', undefined, entryId(summary.entry)) as EntryResult : null;
      const week = detail?.entry as JournalWeek | undefined ?? candidate;
      await request('week', 'POST', { revision: detail?.revision ?? null, entry: { ...week, body: appendDailyEntry(week.body, date, body) } });
      setBody(''); setPreview(false); setStatus('Saved to your journal.');
      window.dispatchEvent(new Event('workspace-saved'));
      if (returnTo) {
        try { localStorage.removeItem(draftKey('quick')); } catch { /* Draft recovery is optional. */ }
        // Keep the form disabled while the browser navigates away.
        leaving = true;
        window.location.assign(returnTo);
      }
    } catch (e) { setError((e as Error).message); }
    finally { if (!leaving) setBusy(false); }
  }
  return <section className="rounded-xl border border-blue-900/60 bg-zinc-900 p-5 space-y-4">
    <div className="flex flex-wrap justify-between gap-2"><div>{!returnTo && <h2 className="text-xl font-semibold">A moment to reflect</h2>}<p className={`${returnTo ? '' : 'mt-1 '}text-sm text-zinc-400`}>What did you learn, finish, or get stuck on?</p></div>{returnTo ? <a className="text-sm text-blue-400 hover:underline" href={returnTo}>← Back to journal</a> : <a className="text-sm text-blue-400 hover:underline" href="/journal">Open journal →</a>}</div>
    <form onSubmit={e => { e.preventDefault(); void save(); }} className="space-y-3"><fieldset disabled={busy || !ready} className="space-y-3">
      <div className="flex flex-wrap gap-3 items-end"><label className="text-sm">Entry date<input type="date" required min="2026-04-20" max="2199-12-31" className={`${field} mt-1`} value={date} onChange={e => setDate(e.target.value)} /></label><button type="button" className={button} aria-pressed={preview} onClick={() => setPreview(!preview)}>{preview ? 'Write' : 'Markdown preview'}</button></div>
      {preview ? <div className="min-h-32 rounded-lg bg-zinc-950 p-4"><Preview body={body} /></div> : <textarea aria-label="Journal entry in Markdown" className={`${field} min-h-32`} required maxLength={50000} placeholder="Today I learned…" value={body} onChange={e => { setBody(e.target.value); setStatus(''); }} />}
      <button className={primary} type="submit" disabled={!body.trim()}>{busy ? 'Saving…' : 'Save entry'}</button> <button className={button} type="button" onClick={() => { if (returnTo && body.trim() && !window.confirm('Discard this unsaved entry?')) return; setBody(''); setDate(localDate()); setPreview(false); setError(''); setStatus('Cancelled.'); if (returnTo) { try { localStorage.removeItem(draftKey('quick')); } catch { /* Draft recovery is optional. */ } window.location.assign(returnTo); } }}>Cancel</button>
    </fieldset></form>
    {error && <p role="alert" className="text-sm text-red-300">{error} {error.includes('session') && <a href="/login" className="underline">Sign in</a>}</p>}
    <p role="status" className="text-sm text-zinc-400">{status}</p>
  </section>;
}
