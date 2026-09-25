import { useEffect, useMemo, useRef, useState } from 'react';
import { ConnectionAvatar } from './ConnectionAvatar';
import { CADENCES, catchUpQueue, lastTouch, markCaughtUp, matchesQuery, newConnection, nextDue, relativeDays, type Connection, type ConnectionImport } from '../lib/connections';
import { localDate } from '../lib/workspace';

interface CompanyOption { slug: string; title: string }
type Sort = 'name' | 'recent' | 'stale' | 'connected';
const PAGE = 100;
const control = 'rounded-lg border border-zinc-700 bg-zinc-950 p-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500';
const input = `mt-1 w-full ${control}`;

async function api<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, { cache: 'no-store', ...init });
  if (response.status === 401) {
    location.assign(`/login?redirect=${encodeURIComponent(location.pathname + location.search)}`);
    throw new Error('Session expired.');
  }
  const result = await response.json();
  if (!response.ok) throw new Error(result.error || 'Request failed. Please retry.');
  return result as T;
}

const companyHref = (slug: string) => `/companies/${slug.split('/').map(encodeURIComponent).join('/')}`;

/** Center-crop to a 256px JPEG so stored photos stay small. */
async function squarePhoto(file: File): Promise<Blob> {
  const bitmap = await createImageBitmap(file);
  const side = Math.min(bitmap.width, bitmap.height);
  const canvas = document.createElement('canvas');
  canvas.width = canvas.height = 256;
  canvas.getContext('2d')!.drawImage(bitmap, (bitmap.width - side) / 2, (bitmap.height - side) / 2, side, side, 0, 0, 256, 256);
  bitmap.close();
  return new Promise((resolve, reject) => canvas.toBlob(blob => blob ? resolve(blob) : reject(new Error('Could not process that image.')), 'image/jpeg', 0.85));
}

export function ConnectionsBoard() {
  const today = localDate();
  const [connections, setConnections] = useState<Connection[]>();
  const [revisions, setRevisions] = useState<Record<string, string>>({});
  const [companies, setCompanies] = useState<CompanyOption[]>([]);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');
  const [busy, setBusy] = useState(false);
  const lock = useRef(false);
  const [query, setQuery] = useState('');
  const [companyFilter, setCompanyFilter] = useState('all');
  const [sort, setSort] = useState<Sort>('name');
  const [limit, setLimit] = useState(PAGE);
  const [draft, setDraft] = useState<Connection>();

  async function load() {
    try {
      const [people, companyPage] = await Promise.all([
        api<{ connections: Connection[]; revisions: Record<string, string> }>('/api/connections'),
        api<{ entries: { entry: CompanyOption }[] }>('/api/entries/company?limit=100'),
      ]);
      const options = companyPage.entries.map(({ entry }) => ({ slug: entry.slug, title: entry.title })).sort((a, b) => a.title.localeCompare(b.title));
      setConnections(people.connections);
      setRevisions(people.revisions);
      setCompanies(options);
      return { people: people.connections, options };
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load connections.');
      return { people: [], options: [] };
    }
  }

  useEffect(() => {
    void load().then(({ people, options }) => {
      // Deep links from search and the companies page.
      const params = new URLSearchParams(location.search);
      const company = options.find(c => c.slug === params.get('company'));
      if (company) setCompanyFilter(company.slug);
      const id = params.get('id');
      const target = id && people.find(c => c.id === id);
      if (target) {
        setDraft({ ...target });
        setQuery(target.name);
      } else if (params.get('new') === '1') {
        setDraft(newConnection(company));
      }
    });
  }, []);

  useEffect(() => {
    const warn = (event: BeforeUnloadEvent) => { if (lock.current || draft) event.preventDefault(); };
    window.addEventListener('beforeunload', warn);
    return () => window.removeEventListener('beforeunload', warn);
  }, [draft]);

  async function run(label: string, action: () => Promise<string>) {
    if (lock.current) return;
    lock.current = true; setBusy(true); setError(''); setStatus(`${label}…`);
    try {
      setStatus(await action());
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Change not saved. Please retry.');
      setStatus('Change not saved. Your draft is kept.');
    } finally { lock.current = false; setBusy(false); }
  }

  function stored(entry: Connection, revision: string) {
    setConnections(current => [...(current ?? []).filter(c => c.id !== entry.id), entry]);
    setRevisions(current => ({ ...current, [entry.id]: revision }));
  }

  const save = (connection: Connection, message: string, closeDraft = false) => run('Saving', async () => {
    const { photo: _photo, ...entry } = connection;
    const result = await api<{ entry: Connection; revision: string }>('/api/entries/connection', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ entry, revision: revisions[connection.id] ?? null }),
    });
    stored({ ...result.entry }, result.revision);
    if (closeDraft) setDraft(undefined);
    return message;
  });

  const remove = (connection: Connection) => {
    if (!window.confirm(`Remove ${connection.name} from your connections? This also removes their photo and notes.`)) return;
    void run('Removing', async () => {
      await api('/api/entries/connection', { method: 'DELETE', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: connection.id, revision: revisions[connection.id] }) });
      setConnections(current => (current ?? []).filter(c => c.id !== connection.id));
      setDraft(undefined);
      return `${connection.name} removed.`;
    });
  };

  const uploadPhoto = (connection: Connection, file: File) => run('Uploading photo', async () => {
    const form = new FormData();
    form.append('id', connection.id);
    form.append('photo', await squarePhoto(file), 'photo.jpg');
    const { photo } = await api<{ photo: string }>('/api/connections/photo', { method: 'POST', body: form });
    setConnections(current => (current ?? []).map(c => c.id === connection.id ? { ...c, photo } : c));
    setDraft(current => current?.id === connection.id ? { ...current, photo } : current);
    return `Photo saved for ${connection.name}.`;
  });

  const removePhoto = (connection: Connection) => run('Removing photo', async () => {
    await api('/api/connections/photo', { method: 'DELETE', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: connection.id }) });
    const clear = (c: Connection) => { const { photo: _photo, ...rest } = c; return rest; };
    setConnections(current => (current ?? []).map(c => c.id === connection.id ? clear(c) : c));
    setDraft(current => current?.id === connection.id ? clear(current) : current);
    return `Photo removed for ${connection.name}.`;
  });

  const importFiles = (form: HTMLFormElement) => run('Importing', async () => {
    const summary = await api<ConnectionImport>('/api/connections/import', { method: 'POST', body: new FormData(form) });
    form.reset();
    await load();
    return `Imported ${summary.parsed} rows: ${summary.created} new, ${summary.updated} updated, ${summary.unchanged} unchanged, ${summary.skipped} skipped. ${summary.linked} linked to companies; ${summary.messagesMatched} with message history.`;
  });

  const companyTitle = useMemo(() => new Map(companies.map(c => [c.slug, c.title])), [companies]);
  const queue = useMemo(() => catchUpQueue(connections ?? [], today), [connections, today]);
  const visible = useMemo(() => {
    const list = (connections ?? []).filter(c => matchesQuery(c, query) && (
      companyFilter === 'all' || (companyFilter === 'linked' ? !!c.companySlug : companyFilter === 'unlinked' ? !c.companySlug : c.companySlug === companyFilter)));
    const touched = (c: Connection) => lastTouch(c) ?? '';
    const compare: Record<Sort, (a: Connection, b: Connection) => number> = {
      name: (a, b) => a.name.localeCompare(b.name),
      recent: (a, b) => (b.lastContactedOn ?? '').localeCompare(a.lastContactedOn ?? '') || a.name.localeCompare(b.name),
      stale: (a, b) => touched(a).localeCompare(touched(b)) || a.name.localeCompare(b.name),
      connected: (a, b) => (b.connectedOn ?? '').localeCompare(a.connectedOn ?? '') || a.name.localeCompare(b.name),
    };
    return list.sort(compare[sort]);
  }, [connections, query, companyFilter, sort]);

  if (!connections) return <p role={error ? 'alert' : 'status'} className={error ? 'text-red-300' : 'text-zinc-400'}>{error || 'Loading…'}</p>;
  const linked = connections.filter(c => c.companySlug).length;

  return (
    <div className="space-y-8">
      <div className="flex flex-wrap items-center gap-4 text-sm text-zinc-400">
        <span>{connections.length} connections · {linked} at tracked companies · {queue.length} to catch up with</span>
        <button type="button" disabled={busy || !!draft} className="rounded-lg bg-blue-600 px-3 py-1.5 text-white hover:bg-blue-500 disabled:opacity-50" onClick={() => setDraft(newConnection())}>+ Add connection</button>
      </div>
      {error && <div role="alert" className="rounded-lg border border-red-800 p-3 text-sm text-red-200">{error}</div>}
      <p role="status" className="text-sm text-zinc-400">{status}</p>

      {draft && <ConnectionForm key={draft.id} draft={draft} isNew={!revisions[draft.id]} busy={busy} companies={companies} today={today}
        onChange={setDraft} onCancel={() => setDraft(undefined)}
        onSave={c => save(c, `${c.name} saved.`, true)} onRemove={remove} onPhoto={uploadPhoto} onRemovePhoto={removePhoto} />}

      <details className="rounded-lg border border-zinc-800 bg-zinc-900 p-4 text-sm" open={connections.length === 0}>
        <summary className="cursor-pointer font-medium text-zinc-200">Import from LinkedIn</summary>
        <div className="mt-3 space-y-3 text-zinc-400">
          <p>On LinkedIn open <em>Settings → Data privacy → Get a copy of your data</em>, pick <em>Connections</em> and <em>Messages</em>, then unzip the export. Re-importing updates names, roles and companies but keeps your notes, cadences and manual company links.</p>
          <form className="flex flex-wrap items-end gap-4" onSubmit={event => { event.preventDefault(); void importFiles(event.currentTarget); }}>
            <fieldset disabled={busy} className="contents">
              <label className="block text-zinc-200">Connections.csv <span className="text-zinc-500">(required)</span><input name="connections" type="file" accept=".csv,text/csv" required className="mt-1 block text-sm text-zinc-400 file:mr-3 file:rounded file:border-0 file:bg-zinc-700 file:px-3 file:py-1.5 file:text-zinc-100" /></label>
              <label className="block text-zinc-200">messages.csv <span className="text-zinc-500">(optional, sets last conversation)</span><input name="messages" type="file" accept=".csv,text/csv" className="mt-1 block text-sm text-zinc-400 file:mr-3 file:rounded file:border-0 file:bg-zinc-700 file:px-3 file:py-1.5 file:text-zinc-100" /></label>
              <button type="submit" className="rounded-lg bg-blue-600 px-3 py-2 text-white hover:bg-blue-500 disabled:opacity-50">Import</button>
            </fieldset>
          </form>
        </div>
      </details>

      <section aria-labelledby="queue-heading" className="space-y-3">
        <h2 id="queue-heading" className="text-xl font-semibold">Catch-up queue</h2>
        {queue.length === 0 ? <p className="text-sm text-zinc-400">Nobody to catch up with. Set a cadence or add someone to the queue.</p> :
          <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {queue.map(({ connection: c, overdueDays, reason }) => <li key={c.id} className="flex gap-3 rounded-lg border border-zinc-800 bg-zinc-900 p-3">
              <ConnectionAvatar connection={c} size={44} />
              <div className="min-w-0 flex-1 space-y-1 text-sm">
                <p className="truncate font-medium text-zinc-100">{c.name}</p>
                <p className="truncate text-zinc-400">{[c.role, c.companySlug ? companyTitle.get(c.companySlug) ?? c.companyName : c.companyName].filter(Boolean).join(' · ')}</p>
                <p className={overdueDays > 0 ? 'text-amber-300' : 'text-zinc-400'}>
                  {reason === 'manual' ? 'Queued' : overdueDays > 0 ? `${overdueDays}d overdue` : 'Due today'} · last talked {relativeDays(c.lastContactedOn, today).toLowerCase()}
                </p>
                <div className="flex flex-wrap gap-3 pt-1">
                  <button type="button" disabled={busy || draft?.id === c.id} className="text-emerald-400 hover:underline disabled:opacity-50" onClick={() => void save(markCaughtUp(c, today), `Logged a conversation with ${c.name}.`)}>Caught up today</button>
                  {reason === 'manual' && <button type="button" disabled={busy || draft?.id === c.id} className="text-zinc-300 hover:underline disabled:opacity-50" onClick={() => void save({ ...c, queued: false }, `${c.name} removed from the queue.`)}>Remove from queue</button>}
                  <button type="button" disabled={busy || !!draft} className="text-blue-400 hover:underline disabled:opacity-50" onClick={() => setDraft({ ...c })}>Edit</button>
                </div>
              </div>
            </li>)}
          </ul>}
      </section>

      <section aria-labelledby="all-heading" className="space-y-3">
        <h2 id="all-heading" className="text-xl font-semibold">All connections</h2>
        <div className="flex flex-wrap gap-3 text-sm">
          <input type="search" aria-label="Search connections" placeholder="Search name, role, company, notes…" value={query} onChange={event => { setQuery(event.target.value); setLimit(PAGE); }} className={`${control} w-full max-w-sm`} />
          <select aria-label="Filter by company" value={companyFilter} onChange={event => { setCompanyFilter(event.target.value); setLimit(PAGE); }} className={control}>
            <option value="all">All companies</option>
            <option value="linked">At tracked companies</option>
            <option value="unlinked">Not at tracked companies</option>
            {companies.map(c => <option key={c.slug} value={c.slug}>{c.title}</option>)}
          </select>
          <select aria-label="Sort connections" value={sort} onChange={event => setSort(event.target.value as Sort)} className={control}>
            <option value="name">Name</option>
            <option value="recent">Recently talked</option>
            <option value="stale">Longest since contact</option>
            <option value="connected">Recently connected</option>
          </select>
        </div>
        <p className="text-sm text-zinc-500">{visible.length} shown</p>
        <ul className="divide-y divide-zinc-800 rounded-lg border border-zinc-800">
          {visible.slice(0, limit).map(c => {
            const due = nextDue(c, today);
            return <li key={c.id} className="flex flex-wrap items-center gap-3 p-3 text-sm sm:flex-nowrap">
              <ConnectionAvatar connection={c} />
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium text-zinc-100">
                  {c.url ? <a href={c.url} target="_blank" rel="noreferrer" className="hover:underline">{c.name}</a> : c.name}
                  {c.queued && <span className="ml-2 rounded bg-amber-500/20 px-1.5 py-0.5 text-xs text-amber-300">queued</span>}
                </p>
                <p className="truncate text-zinc-400">
                  {c.role}{c.role && (c.companySlug || c.companyName) ? ' · ' : ''}
                  {c.companySlug ? <a href={companyHref(c.companySlug)} className="text-blue-400 hover:underline">{companyTitle.get(c.companySlug) ?? c.companyName}</a> : c.companyName}
                </p>
                {c.notes && <p className="truncate text-zinc-500">{c.notes}</p>}
              </div>
              <dl className="grid shrink-0 grid-cols-2 gap-x-4 text-xs text-zinc-400 sm:w-64">
                <dt>Last talked</dt><dd title={c.lastContactedOn} className="text-zinc-200">{relativeDays(c.lastContactedOn, today)}</dd>
                <dt>Connected</dt><dd className="text-zinc-200">{c.connectedOn ?? '—'}</dd>
                <dt>Next catch-up</dt><dd className={due && due <= today ? 'text-amber-300' : 'text-zinc-200'}>{due ? relativeDays(due, today).replace('ago', 'overdue') : '—'}</dd>
              </dl>
              <div className="flex shrink-0 justify-end gap-3 sm:w-28">
                <button type="button" disabled={busy || draft?.id === c.id} aria-pressed={c.queued} className="text-zinc-300 hover:underline disabled:opacity-50" onClick={() => void save({ ...c, queued: !c.queued }, c.queued ? `${c.name} removed from the queue.` : `${c.name} added to the queue.`)}>{c.queued ? 'Unqueue' : 'Queue'}</button>
                <button type="button" disabled={busy || !!draft} className="text-blue-400 hover:underline disabled:opacity-50" onClick={() => setDraft({ ...c })}>Edit</button>
              </div>
            </li>;
          })}
          {visible.length === 0 && <li className="p-3 text-sm text-zinc-400">No connections match.</li>}
        </ul>
        {visible.length > limit && <button type="button" className="text-sm text-blue-400 hover:underline" onClick={() => setLimit(limit + PAGE)}>Show {Math.min(PAGE, visible.length - limit)} more</button>}
      </section>
    </div>
  );
}

interface FormProps {
  draft: Connection; isNew: boolean; busy: boolean; companies: CompanyOption[]; today: string;
  onChange: (c: Connection) => void; onCancel: () => void; onSave: (c: Connection) => void;
  onRemove: (c: Connection) => void; onPhoto: (c: Connection, file: File) => void; onRemovePhoto: (c: Connection) => void;
}

function ConnectionForm({ draft, isNew, busy, companies, today, onChange, onCancel, onSave, onRemove, onPhoto, onRemovePhoto }: FormProps) {
  const set = <K extends keyof Connection>(key: K, value: Connection[K]) => onChange({ ...draft, [key]: value });
  const optional = (key: 'url' | 'connectedOn' | 'lastContactedOn', value: string) => {
    const { [key]: _old, ...rest } = draft;
    onChange(value ? { ...rest, [key]: value } : rest);
  };
  return (
    <form aria-label={isNew ? 'New connection' : `Edit ${draft.name}`} className="space-y-4 rounded-lg border border-blue-900 bg-zinc-900 p-4 text-sm"
      onSubmit={event => { event.preventDefault(); onSave(draft); }}>
      <h2 className="text-lg font-semibold">{isNew ? 'New connection' : `Edit ${draft.name}`}</h2>
      <fieldset disabled={busy} className="space-y-4">
        <div className="flex items-center gap-4">
          <ConnectionAvatar connection={draft} size={64} />
          {isNew ? <p className="text-zinc-500">Save first to add a photo.</p> : <div className="flex flex-wrap gap-3">
            <label className="cursor-pointer text-blue-400 hover:underline">{draft.photo ? 'Replace photo' : 'Upload photo'}
              <input type="file" accept="image/*" className="sr-only" onChange={event => { const file = event.target.files?.[0]; event.target.value = ''; if (file) onPhoto(draft, file); }} />
            </label>
            {draft.photo && <button type="button" className="text-red-300 hover:underline" onClick={() => onRemovePhoto(draft)}>Remove photo</button>}
          </div>}
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <label className="block text-zinc-200">Name<input required maxLength={200} value={draft.name} onChange={e => set('name', e.target.value)} className={input} /></label>
          <label className="block text-zinc-200">Role<input maxLength={200} value={draft.role} onChange={e => set('role', e.target.value)} className={input} /></label>
          <label className="block text-zinc-200">Company (as on LinkedIn)<input maxLength={200} value={draft.companyName} onChange={e => set('companyName', e.target.value)} className={input} /></label>
          <label className="block text-zinc-200">Tracked company
            <select value={draft.companySlug ?? ''} className={input} onChange={e => {
              const company = companies.find(c => c.slug === e.target.value);
              const { companySlug: _old, ...rest } = draft;
              onChange(company ? { ...rest, companySlug: company.slug, companyName: draft.companyName || company.title } : rest);
            }}>
              <option value="">Not linked</option>
              {companies.map(c => <option key={c.slug} value={c.slug}>{c.title}</option>)}
            </select>
          </label>
          <label className="block text-zinc-200">Email<input type="email" maxLength={254} value={draft.email} onChange={e => set('email', e.target.value)} className={input} /></label>
          <label className="block text-zinc-200">LinkedIn URL<input type="url" maxLength={2000} value={draft.url ?? ''} onChange={e => optional('url', e.target.value)} className={input} /></label>
          <label className="block text-zinc-200">Connected on<input type="date" value={draft.connectedOn ?? ''} onChange={e => optional('connectedOn', e.target.value)} className={input} /></label>
          <label className="block text-zinc-200">Last conversation
            <span className="flex gap-2"><input type="date" value={draft.lastContactedOn ?? ''} onChange={e => optional('lastContactedOn', e.target.value)} className={input} />
              <button type="button" className="mt-1 shrink-0 rounded-lg border border-zinc-700 px-2 text-zinc-300 hover:border-zinc-500" onClick={() => optional('lastContactedOn', today)}>Today</button></span>
          </label>
          <label className="block text-zinc-200">Catch up every
            <select value={draft.cadenceDays ?? ''} className={input} onChange={e => {
              const { cadenceDays: _old, ...rest } = draft;
              onChange(e.target.value ? { ...rest, cadenceDays: Number(e.target.value) } : rest);
            }}>
              <option value="">No cadence</option>
              {CADENCES.map(days => <option key={days} value={days}>{days < 365 ? `${days} days` : 'Year'}</option>)}
            </select>
          </label>
          <label className="block text-zinc-200">Tags <span className="text-zinc-500">(comma separated)</span>
            <input value={draft.tags.join(', ')} onChange={e => set('tags', e.target.value.split(',').map(t => t.trim()).filter(Boolean).slice(0, 30))} className={input} />
          </label>
        </div>
        <label className="flex items-center gap-2 text-zinc-200"><input type="checkbox" className="accent-blue-500" checked={draft.queued} onChange={e => set('queued', e.target.checked)} />In catch-up queue</label>
        <label className="block text-zinc-200">Notes<textarea maxLength={5000} value={draft.notes} onChange={e => set('notes', e.target.value)} className={`${input} min-h-24`} placeholder="How you met, what they work on, what to follow up on…" /></label>
        <div className="flex flex-wrap gap-3">
          <button type="submit" className="rounded-lg bg-blue-600 px-3 py-2 text-white hover:bg-blue-500 disabled:opacity-50">Save connection</button>
          <button type="button" className="rounded-lg border border-zinc-700 px-3 py-2 text-zinc-200 hover:border-zinc-500" onClick={onCancel}>Cancel</button>
          {!isNew && <button type="button" className="ml-auto text-red-300 hover:underline" onClick={() => onRemove(draft)}>Remove connection</button>}
        </div>
      </fieldset>
    </form>
  );
}
