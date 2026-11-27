import { useEffect, useRef, useState } from 'react';
import MarkdownPreview from './MarkdownPreview';
import { Button, Label } from './ui';
import { weekRange } from '../lib/journal';
import type { JournalWeek, PersonalJournalEntry } from '../lib/workspace';

export type JournalKind = 'week' | 'personal';
export type JournalItem<T> = { entry: T; revision: string | null };
export type JournalEntry = JournalWeek | PersonalJournalEntry;

/** Same storage key the shared workspace editor used, so drafts saved before this page changed still recover. */
export function draftKey(kind: JournalKind) {
  let user = 'user';
  try { user = localStorage.getItem('auth_username') || 'user'; } catch { /* Storage is optional. */ }
  return `career-workspace:v1:${user}:${kind}`;
}
/** A draft left by an unsaved edit, if any. */
export function readDraft<T extends JournalEntry>(kind: JournalKind): JournalItem<T> | null {
  try {
    const raw = localStorage.getItem(draftKey(kind));
    if (!raw) return null;
    const draft = JSON.parse(raw);
    if (draft?.entry && typeof draft.entry.body === 'string' && Array.isArray(draft.entry.tags)) return { entry: draft.entry, revision: draft.revision ?? null };
  } catch { /* Unreadable drafts are ignored. */ }
  return null;
}
export function clearDraft(kind: JournalKind) {
  try { localStorage.removeItem(draftKey(kind)); } catch { /* Storage is optional. */ }
}

export function signInAgain(): never {
  location.assign(`/login?redirect=${encodeURIComponent(location.pathname + location.search)}`);
  throw new Error('Your session expired. Sign in again.');
}
/** An API error that keeps its HTTP status (409 means the entry changed since it was read). */
export class RequestError extends Error {
  constructor(message: string, readonly status: number) { super(message); }
}
export async function journalRequest<T>(kind: JournalKind, method: 'POST' | 'DELETE', input: unknown): Promise<T> {
  const response = await fetch(`/api/entries/${kind}`, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(input) });
  if (response.status === 401) signInAgain();
  const result = await response.json().catch(() => ({}));
  if (!response.ok) throw new RequestError(result.error || 'Request failed. Please retry.', response.status);
  return result as T;
}
/** One saved entry with its current revision, or null when it no longer exists. */
export async function fetchEntry<T extends JournalEntry>(kind: JournalKind, id: string): Promise<JournalItem<T> | null> {
  const response = await fetch(`/api/entries/${kind}?id=${encodeURIComponent(id)}`, { cache: 'no-store' });
  if (response.status === 401) signInAgain();
  if (response.status === 404) return null;
  const result = await response.json().catch(() => ({}));
  if (!response.ok) throw new RequestError(result.error || 'This entry could not load.', response.status);
  return result as JournalItem<T>;
}

export const entryId = (entry: JournalEntry) => 'slug' in entry ? entry.slug : entry.id;

/** Writing for a journal entry: title, date and summary (personal), tags, then Markdown with a preview. */
export function JournalEditor({ kind, item, isNew, recovered, onSaved, onCancel, onDeleted, onDirtyChange }: {
  kind: JournalKind;
  item: JournalItem<JournalEntry>;
  isNew: boolean;
  recovered?: boolean;
  onSaved: (saved: JournalItem<JournalEntry>) => void;
  onCancel: () => void;
  onDeleted: (id: string) => void;
  onDirtyChange?: (dirty: boolean) => void;
}) {
  const [entry, setEntry] = useState(item.entry);
  const [tags, setTags] = useState(item.entry.tags.join(', '));
  const [dirty, setDirty] = useState(!!recovered);
  const [preview, setPreview] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [status, setStatus] = useState(recovered ? 'Recovered your unsaved draft.' : '');
  const [revision, setRevision] = useState(item.revision);
  const [conflict, setConflict] = useState(false);
  const first = useRef<HTMLInputElement & HTMLTextAreaElement>(null);

  useEffect(() => { first.current?.focus(); }, []);
  useEffect(() => { onDirtyChange?.(dirty); }, [dirty]);
  useEffect(() => () => onDirtyChange?.(false), []);
  useEffect(() => {
    if (!dirty) return;
    const onLeave = (event: BeforeUnloadEvent) => { event.preventDefault(); };
    window.addEventListener('beforeunload', onLeave);
    try { localStorage.setItem(draftKey(kind), JSON.stringify({ entry: withTags(entry, tags), revision })); }
    catch { setStatus('Local draft could not be saved. Keep this tab open until you save.'); }
    return () => window.removeEventListener('beforeunload', onLeave);
  }, [entry, tags, dirty, kind, revision]);

  function change(patch: Partial<PersonalJournalEntry> & Partial<JournalWeek>) {
    setEntry(current => ({ ...current, ...patch }) as JournalEntry);
    setDirty(true); setStatus('Unsaved changes');
  }
  async function save() {
    setBusy(true); setError('');
    try {
      const saved = await journalRequest<{ entry: JournalEntry; revision: string }>(kind, 'POST', { entry: withTags(entry, tags), revision });
      clearDraft(kind);
      setDirty(false);
      window.dispatchEvent(new Event('workspace-saved'));
      onSaved(saved);
    } catch (e) { fail(e, true); }
  }
  /** Only a save of an existing entry can recover from a conflict by saving over the newer revision. */
  function fail(e: unknown, saving = false) {
    setError((e as Error).message);
    setConflict(saving && !isNew && e instanceof RequestError && e.status === 409);
    setBusy(false);
  }
  /** After a conflict: adopt the latest saved revision but keep what was typed, so Save replaces it. */
  async function takeLatest() {
    setBusy(true);
    try {
      const latest = isNew ? null : await fetchEntry(kind, entryId(entry));
      setRevision(latest?.revision ?? null);
      setConflict(false); setError('');
      setStatus(latest ? 'Loaded the latest saved revision. Save replaces it with your text.' : 'The saved entry is gone. Save creates it again with your text.');
    } catch (e) { setError((e as Error).message); }
    finally { setBusy(false); }
  }
  async function remove() {
    if (!window.confirm(`Delete ${titleOf(entry)}? This cannot be undone.`)) return;
    setBusy(true); setError('');
    try {
      await journalRequest(kind, 'DELETE', { id: entryId(entry), revision });
      clearDraft(kind);
      setDirty(false);
      window.dispatchEvent(new Event('workspace-saved'));
      onDeleted(entryId(entry));
    } catch (e) { fail(e); }
  }
  function cancel() {
    clearDraft(kind);
    setDirty(false);
    onCancel();
  }

  const heading = 'slug' in entry ? `Week ${entry.week}` : isNew ? 'New entry' : 'Edit entry';
  const meta = 'slug' in entry ? `${weekRange(entry.dates)}, ${entry.year}` : 'Personal journal';
  return <form className="flex min-w-0 flex-col gap-5" aria-label={`Edit ${'slug' in entry ? `week ${entry.week}` : 'personal entry'}`} onSubmit={event => { event.preventDefault(); void save(); }}>
    <div className="flex flex-col gap-1.5"><Label>{meta}</Label><h2 className="m-0 text-[28px] font-light leading-tight md:text-[32px]">{heading}</h2></div>
    <fieldset disabled={busy} className="m-0 flex min-w-0 flex-col gap-5 border-0 p-0 disabled:opacity-60">
      {'id' in entry && <>
        <label className="ui-field"><span className="ui-label">Title</span><input ref={first} className="ui-field__input" required maxLength={200} value={entry.title} onChange={event => change({ title: event.target.value })} /></label>
        <div className="grid gap-5 sm:grid-cols-[180px_minmax(0,1fr)]">
          <label className="ui-field"><span className="ui-label">Date (blank for background)</span><input className="ui-field__input" type="date" value={entry.date} onChange={event => change({ date: event.target.value })} /></label>
          <label className="ui-field"><span className="ui-label">Summary</span><input className="ui-field__input" maxLength={1000} value={entry.description} onChange={event => change({ description: event.target.value })} /></label>
        </div>
      </>}
      <label className="ui-field"><span className="ui-label">Tags, comma separated</span><input className="ui-field__input" maxLength={2000} value={tags} onChange={event => { setTags(event.target.value); setDirty(true); setStatus('Unsaved changes'); }} /></label>
      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between gap-3">
          <span className="ui-label" id={`${kind}-body-label`}>Markdown</span>
          <div className="flex gap-2">
            <Button size="sm" aria-pressed={!preview} onClick={() => setPreview(false)}>Write</Button>
            <Button size="sm" aria-pressed={preview} onClick={() => setPreview(true)}>Preview</Button>
          </div>
        </div>
        {preview
          ? <div className={`min-h-72 border border-ink-30 p-4 ${markdownTone}`}><MarkdownPreview body={entry.body} /></div>
          : <textarea ref={'slug' in entry ? first : undefined} aria-labelledby={`${kind}-body-label`} className="min-h-80 w-full resize-y border border-ink-30 bg-transparent p-3 font-mono text-sm leading-relaxed text-ink placeholder:text-ink-60 focus:border-ink"
            maxLength={100000} value={entry.body} placeholder="Write your thoughts. Markdown, lists, links, and code blocks are welcome." onChange={event => change({ body: event.target.value })} />}
      </div>
      <div className="flex flex-wrap gap-2">
        <Button type="submit" variant="primary">{busy ? 'Saving…' : 'Save'}</Button>
        <Button onClick={cancel}>Cancel</Button>
        {!isNew && <Button variant="danger" className="ml-auto" onClick={() => void remove()}>Delete</Button>}
      </div>
    </fieldset>
    {error && <div role="alert" className="flex flex-col gap-3 border border-accent p-3 text-sm text-ink">
      <p className="m-0">{error}{conflict && ' Someone saved this entry after you opened it. Your text is still here.'}</p>
      {conflict && <Button size="sm" className="self-start" disabled={busy} onClick={() => void takeLatest()}>Keep my text and save over it</Button>}
    </div>}
    <p role="status" className="m-0 font-mono text-xs text-ink-60">{status}</p>
  </form>;
}

function withTags(entry: JournalEntry, tags: string): JournalEntry {
  return { ...entry, tags: [...new Set(tags.split(',').map(tag => tag.trim()).filter(Boolean))].slice(0, 30) };
}
function titleOf(entry: JournalEntry) {
  return 'slug' in entry ? `Week ${entry.week}` : entry.title || 'this entry';
}

/** Markdown rendered in ink on paper (MarkdownPreview's prose styles are the old zinc look). */
export const markdownTone = [
  '[&_.prose]:text-ink [&_p]:text-ink [&_li]:text-ink [&_ul]:text-ink [&_ol]:text-ink [&_strong]:text-ink [&_em]:text-ink [&_td]:text-ink [&_th]:text-ink',
  '[&_p]:font-light [&_li]:font-light [&_p]:text-[16px] [&_li]:text-[16px]',
  '[&_h1]:text-ink [&_h2]:text-ink [&_h3]:text-ink [&_h4]:text-ink [&_h5]:text-ink [&_h6]:text-ink [&_h1]:font-light [&_h2]:font-light [&_h3]:font-normal',
  '[&_h1]:!text-[28px] [&_h2]:!text-[22px] [&_h3]:!text-[17px] [&_h4]:!text-[15px]',
  '[&_a]:text-ink [&_a]:underline [&_a]:underline-offset-[3px] [&_a:hover]:text-ink-hover',
  '[&_code]:rounded-none [&_code]:bg-ink-07 [&_code]:text-ink [&_pre]:rounded-none [&_pre]:border-ink-30 [&_pre]:bg-transparent',
  '[&_blockquote]:border-ink-30 [&_blockquote]:text-ink-60 [&_hr]:border-ink-30 [&_th]:border-ink-30 [&_td]:border-ink-30',
].join(' ');
