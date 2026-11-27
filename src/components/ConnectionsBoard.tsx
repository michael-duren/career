import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { Avatar, Box, Button, Checkbox, HatchFill, Label, MeasurementLine, SearchField, SectionHeader, SegmentedTabs, buttonClassName, type SegmentedTab } from './ui';
import { moduleEnabled } from '../lib/modules';
import { CADENCES, CATCH_UP_GOAL_KEY, STAGE_LABELS, catchUpQueue, companySlug, employerKey, employers, isSnoozed, lastTouch, linkableGoals, linkedGoal, markCaughtUp, matchesQuery, newCompanyEntry, newConnection, nextDue, photoURL, relativeDays, trackedNames, withCatchUpLink, type CatchUpWeek, type Connection, type ConnectionImport, type ConnectionQueue, type ConnectionStage, type Employer } from '../lib/connections';
import type { Goal } from '../lib/timeline';
import { CatchUpBoard } from './CatchUpBoard';
import { formatDate } from '../lib/format';
import { localDate } from '../lib/workspace';
import { cn } from '../lib/utils';

// The People page (docs/redesign/pages/people.md; docs/redesign/reference/desktop/ARC-Connections.html,
// ARC-CatchUp.html and docs/redesign/reference/mobile/M-Connections.html): every person, the catch-up queue and where the
// network works, as tabs.

interface CompanyOption { slug: string; title: string }
type Sort = 'name' | 'recent' | 'stale' | 'connected';
type Tab = 'people' | 'queue' | 'employers';
const TABS: readonly Tab[] = ['people', 'queue', 'employers'];
const PAGE = 100;
const EMPLOYER_PAGE = 30;
const SORT_LABELS: Record<Sort, string> = { name: 'A–Z', recent: 'Recently talked', stale: 'Longest since contact', connected: 'Recently connected' };
const select = 'h-[34px] min-w-0 border border-ink-30 bg-paper px-2 font-mono text-[11px] tracking-[0.06em] text-ink';
const textarea = 'w-full resize-y border border-ink-30 bg-transparent p-2.5 text-[13px] font-light text-ink placeholder:text-ink-60';
const panelId = 'connections-panel';

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
const linkedIn = (c: Connection) => (c.url && /^https?:\/\//i.test(c.url) ? c.url : undefined);
const shortDate = (date?: string) => (date ? formatDate(date) : '—');

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

function useMedia(query: string): boolean {
  const [matches, setMatches] = useState(() => window.matchMedia(query).matches);
  useEffect(() => {
    const media = window.matchMedia(query);
    const update = () => setMatches(media.matches);
    update();
    media.addEventListener('change', update);
    return () => media.removeEventListener('change', update);
  }, [query]);
  return matches;
}

function initialTab(): Tab {
  const tab = new URLSearchParams(location.search).get('tab');
  return TABS.includes(tab as Tab) ? tab as Tab : 'people';
}

export function ConnectionsBoard() {
  const today = localDate();
  // Below 768px is the phone layout; from 1280px the catch-up board has room for its side panel.
  const phone = useMedia('(max-width: 767px)');
  const wide = useMedia('(min-width: 1280px)');
  const [connections, setConnections] = useState<Connection[]>();
  const [revisions, setRevisions] = useState<Record<string, string>>({});
  const [companies, setCompanies] = useState<CompanyOption[]>([]);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');
  const [busy, setBusy] = useState(false);
  const lock = useRef(false);
  const [tab, setTab] = useState<Tab>(initialTab);
  const [query, setQuery] = useState('');
  const [employerQuery, setEmployerQuery] = useState('');
  const [companyFilter, setCompanyFilter] = useState('all');
  // Set from the Employers tab: only people who count toward that employer (see employerKey).
  const [employer, setEmployer] = useState<{ key: string; name: string }>();
  const [sort, setSort] = useState<Sort>('name');
  const [limit, setLimit] = useState(PAGE);
  const [employerLimit, setEmployerLimit] = useState(EMPLOYER_PAGE);
  const [selectedId, setSelectedId] = useState<string>();
  const [note, setNote] = useState<{ id: string; text: string }>();
  const [draft, setDraft] = useState<Connection>();
  const [importing, setImporting] = useState(false);
  const importPanel = useRef<HTMLDivElement>(null);
  // Tracked companies and the catch-up queue need the Companies module; People works without it.
  const [companiesOn, setCompaniesOn] = useState(true);
  const [companiesFailed, setCompaniesFailed] = useState(false);
  // The catch-up queue's "today" and week come from the server (the scheduler's time zone).
  const [queueInfo, setQueueInfo] = useState<{ today: string; week: CatchUpWeek }>();
  const [queueFailed, setQueueFailed] = useState(false);
  // Timeline goals for the queue's goal strip; undefined when the Timeline module is off or they failed to load.
  const [goals, setGoals] = useState<Goal[]>();
  const [goalRevisions, setGoalRevisions] = useState<Record<string, string>>({});

  async function load() {
    let on = true;
    try {
      on = await moduleEnabled('companies');
      setCompaniesOn(on);
      setCompaniesFailed(false);
      if (!on) setCompanyFilter('all');
      // A failed company list is reported on its own; connections still load.
      const companyList = on
        ? api<{ entries: { entry: CompanyOption }[] }>('/api/entries/company?limit=100').catch(error => {
          console.warn('Companies could not be loaded.', error);
          setCompaniesFailed(true);
          return { entries: [] as { entry: CompanyOption }[] };
        })
        : Promise.resolve({ entries: [] as { entry: CompanyOption }[] });
      // The queue and goals are extras: when they fail the rest of the page still works.
      const queueRequest = on
        ? api<ConnectionQueue>('/api/connections/queue').then(queue => { setQueueFailed(false); return queue; }).catch(error => {
          console.warn('The catch-up queue could not be loaded.', error);
          setQueueFailed(true);
          return undefined;
        })
        : Promise.resolve(undefined);
      const goalRequest = on ? moduleEnabled('timeline').then(timeline => timeline
        ? api<{ goals: Goal[]; revisions: Record<string, string> }>('/api/goals')
        : undefined).catch(error => {
        console.warn('Goals could not be loaded.', error);
        return undefined;
      }) : Promise.resolve(undefined);
      const [people, companyPage, queue, goalPage] = await Promise.all([
        api<{ connections: Connection[]; revisions: Record<string, string> }>('/api/connections'),
        companyList, queueRequest, goalRequest,
      ]);
      if (queue) setQueueInfo({ today: queue.today, week: queue.week });
      setGoals(goalPage?.goals);
      setGoalRevisions(goalPage?.revisions ?? {});
      const options = companyPage.entries.map(({ entry }) => ({ slug: entry.slug, title: entry.title })).sort((a, b) => a.title.localeCompare(b.title));
      setConnections(people.connections);
      setRevisions(people.revisions);
      setCompanies(options);
      return { ok: true, people: people.connections, options, on };
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load connections.');
      return { ok: false, people: [], options: [], on };
    }
  }

  useEffect(() => {
    void load().then(({ ok, people, options, on }) => {
      if (!ok) return;
      if (!on) setTab(current => current === 'queue' ? 'people' : current);
      // Deep links from search and the companies page.
      const params = new URLSearchParams(location.search);
      const company = options.find(c => c.slug === params.get('company'));
      if (company) setCompanyFilter(company.slug);
      const id = params.get('id');
      const target = id && people.find(c => c.id === id);
      if (target) {
        setTab('people');
        setSelectedId(target.id);
        setQuery(target.name);
      } else if (params.get('new') === '1') {
        setTab('people');
        setDraft(newConnection(company));
      } else if (params.get('import') === '1') {
        // From setup's "LinkedIn export" step.
        setTab('people');
        openImport();
      }
    });
  }, []);

  // Keep the open tab in the address so reloads and shared links return to it.
  useEffect(() => {
    const url = new URL(location.href);
    if (tab === 'people') url.searchParams.delete('tab');
    else url.searchParams.set('tab', tab);
    if (url.href !== location.href) history.replaceState(history.state, '', url);
  }, [tab]);

  const selected = connections?.find(c => c.id === selectedId);
  const noteText = note && note.id === selected?.id ? note.text : selected?.notes ?? '';
  const noteDirty = !!selected && noteText !== selected.notes;

  useEffect(() => {
    const warn = (event: BeforeUnloadEvent) => { if (lock.current || draft || noteDirty) event.preventDefault(); };
    window.addEventListener('beforeunload', warn);
    return () => window.removeEventListener('beforeunload', warn);
  }, [draft, noteDirty]);

  async function run(label: string, action: () => Promise<string>, failure = 'Change not saved.') {
    if (lock.current) return;
    lock.current = true; setBusy(true); setError(''); setStatus(`${label}…`);
    try {
      setStatus(await action());
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Change not saved. Please retry.');
      setStatus(failure);
    } finally { lock.current = false; setBusy(false); }
  }

  function stored(entry: Connection, revision: string) {
    setConnections(current => [...(current ?? []).filter(c => c.id !== entry.id), entry]);
    setRevisions(current => ({ ...current, [entry.id]: revision }));
    // The saved notes are now the panel's notes (panel actions save the typed text too).
    setNote(current => current?.id === entry.id ? undefined : current);
  }

  const save = (connection: Connection, message: string, closeDraft = false) => run('Saving', async () => {
    const { photo: _photo, ...entry } = connection;
    const result = await api<{ entry: Connection; revision: string }>('/api/entries/connection', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ entry, revision: revisions[connection.id] ?? null }),
    });
    // Keep the photo revision if the saved entry does not echo it back.
    stored({ ...(connection.photo ? { photo: connection.photo } : {}), ...result.entry }, result.revision);
    if (closeDraft) { setDraft(undefined); setSelectedId(result.entry.id); }
    return message;
  }, closeDraft ? 'Change not saved. Your draft is kept.' : 'Change not saved.');

  // Catch-up queue changes (docs/developer.md, Catch-up queue). Each returns the saved person.
  async function queueCall(path: string, method: string, body: Record<string, unknown>) {
    const result = await api<{ entry: Connection; revision: string; week?: CatchUpWeek }>(`/api/connections/queue${path}`, {
      method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body),
    });
    const before = connections?.find(c => c.id === result.entry.id);
    stored({ ...(before?.photo ? { photo: before.photo } : {}), ...result.entry }, result.revision);
    if (result.week) setQueueInfo(current => current && { ...current, week: result.week! });
    return result.entry;
  }

  const moveTo = (c: Connection, stage: ConnectionStage) => run('Saving', async () => {
    await queueCall('', 'POST', { id: c.id, revision: revisions[c.id], stage });
    return c.stage === stage ? `${c.name} is back in the queue.` : c.stage ? `${c.name} moved to ${STAGE_LABELS[stage]}.` : `${c.name} added to the queue.`;
  });

  // Resolves true once saved (the catch-up panel then drops the draft).
  async function unqueue(c: Connection): Promise<boolean> {
    let ok = false;
    await run('Saving', async () => {
      await queueCall('', 'DELETE', { id: c.id, revision: revisions[c.id] });
      ok = true;
      return `${c.name} removed from the queue.`;
    });
    return ok;
  }

  // Resolves true once saved, so the panel can clear the message draft.
  async function markSent(c: Connection): Promise<boolean> {
    let ok = false;
    await run('Saving', async () => {
      const entry = await queueCall('/sent', 'POST', { id: c.id, revision: revisions[c.id] });
      ok = true;
      return entry.stage ? `Sent. ${c.name} moved to ${STAGE_LABELS[entry.stage]}.` : `Sent. ${c.name} finished the queue.`;
    });
    return ok;
  }

  const snooze = (c: Connection, until: string) => run('Saving', async () => {
    await queueCall('/snooze', 'POST', { id: c.id, revision: revisions[c.id], until });
    return `${c.name} snoozed until ${formatDate(until)}.`;
  });

  async function setTarget(weeklyTarget: number): Promise<boolean> {
    let ok = false;
    await run('Saving', async () => {
      const { week } = await api<{ week: CatchUpWeek }>('/api/connections/queue/settings', {
        method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ weeklyTarget }),
      });
      setQueueInfo(current => current && { ...current, week });
      ok = true;
      return `Weekly target set to ${weeklyTarget}.`;
    });
    return ok;
  }

  // The strip shows one goal: linking another unlinks the old one first.
  const linkGoal = (goalId: string) => run('Saving', async () => {
    const all = goals ?? [];
    const saveGoal = async (goal: Goal) => {
      const result = await api<{ goal: Goal; revision: string }>('/api/goals', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ goal, revision: goalRevisions[goal.id] ?? null }),
      });
      setGoals(current => (current ?? []).map(g => g.id === result.goal.id ? result.goal : g));
      setGoalRevisions(current => ({ ...current, [result.goal.id]: result.revision }));
    };
    for (const goal of all.filter(g => g.id !== goalId && CATCH_UP_GOAL_KEY in g.metadata)) await saveGoal(withCatchUpLink(goal, false));
    const target = all.find(g => g.id === goalId);
    if (target) await saveGoal(withCatchUpLink(target, true));
    return target ? `Linked ${target.title} to the catch-up queue.` : 'The catch-up queue has no linked goal.';
  });

  const remove = (connection: Connection) => {
    if (!window.confirm(`Remove ${connection.name} from your connections? This also removes their photo and notes.`)) return;
    void run('Removing', async () => {
      await api('/api/entries/connection', { method: 'DELETE', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: connection.id, revision: revisions[connection.id] }) });
      setConnections(current => (current ?? []).filter(c => c.id !== connection.id));
      setDraft(undefined);
      setSelectedId(undefined);
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
    const { ok } = await load();
    return `${ok ? '' : 'The list could not be refreshed; reload to see the changes. '}Imported ${summary.parsed} rows: ${summary.created} new, ${summary.updated} updated, ${summary.unchanged} unchanged, ${summary.skipped} skipped. ${summary.linked} linked to companies; ${summary.messagesMatched} with message history.`;
  });

  // New companies pick up the unlinked people who work there (the server links them on save).
  const addCompanies = (names: string[]) => run('Adding companies', async () => {
    const taken = new Set(companies.map(c => c.slug));
    const planned = names.map(name => {
      const base = companySlug(name);
      if (!base) throw new Error(`"${name}" needs letters or digits to become a company.`);
      let slug = base;
      for (let n = 2; taken.has(slug); n++) slug = `${base}-${n}`;
      taken.add(slug);
      return { name, slug };
    });
    let added = 0;
    try {
      for (const { name, slug } of planned) {
        await api('/api/entries/company', {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ entry: newCompanyEntry(name, slug), revision: null }),
        });
        added++;
      }
    } catch (e) {
      const reason = e instanceof Error ? e.message : 'Request failed.';
      throw new Error(added ? `Added ${added} of ${planned.length} companies, then: ${reason}` : reason);
    } finally {
      if (added) await load();
    }
    return `Added ${added} ${added === 1 ? 'company' : 'companies'} to your board. People who work there are now linked.`;
  });

  function openImport() {
    setImporting(true);
    requestAnimationFrame(() => {
      importPanel.current?.scrollIntoView({ block: 'nearest' });
      importPanel.current?.querySelector('input')?.focus();
    });
  }

  function pick(id: string | undefined) {
    if (draft) return;
    if (noteDirty && !window.confirm(`Discard your unsaved notes for ${selected?.name}?`)) return;
    setSelectedId(id);
    setNote(undefined);
  }

  function edit(c: Connection) {
    if (noteDirty && !window.confirm(`Discard your unsaved notes for ${selected?.name}?`)) return;
    setNote(undefined);
    setTab('people');
    setSelectedId(c.id);
    setDraft({ ...c });
  }

  const companyTitle = useMemo(() => new Map(companies.map(c => [c.slug, c.title])), [companies]);
  const queueToday = queueInfo?.today ?? today;
  const due = useMemo(() => catchUpQueue(connections ?? [], today), [connections, today]);
  const queuedCount = useMemo(() => (connections ?? []).filter(c => !!c.stage && !isSnoozed(c, queueToday)).length, [connections, queueToday]);
  const board = useMemo(() => companiesOn ? trackedNames(companyTitle) : undefined, [companiesOn, companyTitle]);
  const network = useMemo(() => employers(connections ?? [], companiesOn ? companyTitle : undefined), [connections, companiesOn, companyTitle]);
  const visible = useMemo(() => {
    const list = (connections ?? []).filter(c => matchesQuery(c, query) && (!employer || employerKey(c, board) === employer.key) && (
      companyFilter === 'all' || (companyFilter === 'linked' ? !!c.companySlug : companyFilter === 'unlinked' ? !c.companySlug : c.companySlug === companyFilter)));
    const touched = (c: Connection) => lastTouch(c) ?? '';
    const compare: Record<Sort, (a: Connection, b: Connection) => number> = {
      name: (a, b) => a.name.localeCompare(b.name),
      recent: (a, b) => (b.lastContactedOn ?? '').localeCompare(a.lastContactedOn ?? '') || a.name.localeCompare(b.name),
      stale: (a, b) => touched(a).localeCompare(touched(b)) || a.name.localeCompare(b.name),
      connected: (a, b) => (b.connectedOn ?? '').localeCompare(a.connectedOn ?? '') || a.name.localeCompare(b.name),
    };
    return list.sort(compare[sort]);
  }, [connections, query, companyFilter, sort, employer, board]);

  if (!connections) return <p role={error ? 'alert' : 'status'} className={error ? 'text-accent' : 'text-ink-60'}>{error || 'Loading…'}</p>;

  const companyName = (c: Connection) => (c.companySlug && companiesOn ? companyTitle.get(c.companySlug) ?? c.companyName : c.companyName);
  const tracked = (c: Connection) => companiesOn && !!c.companySlug;
  const queueTab = companiesOn ? [{ value: 'queue' as const, label: `${phone ? 'Catch-up' : 'Catch-up queue'} · ${queuedCount}` }] : [];
  const tabs: SegmentedTab<Tab>[] = [{ value: 'people', label: 'People' }, ...queueTab, { value: 'employers', label: 'Employers' }];
  const showDetail = tab === 'people' && (!!draft || !!selected);
  const searchProps = tab === 'employers'
    ? { label: 'Search employers', value: employerQuery, onChange: (value: string) => { setEmployerQuery(value); setEmployerLimit(EMPLOYER_PAGE); } }
    : { label: 'Search people or roles', value: query, onChange: (value: string) => { setQuery(value); setLimit(PAGE); } };

  return (
    <div className="flex flex-col">
      <header className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div className="flex flex-col gap-4 md:flex-row md:items-end md:gap-9">
          <div className="flex flex-col gap-2">
            <Label>{connections.length} people · {network.length} employers</Label>
            <h1 className="m-0 font-extralight !text-[40px] !leading-[0.9] md:!text-[52px]">
              <span className="md:hidden">People</span><span className="hidden md:inline">Connections</span>
            </h1>
          </div>
          <SegmentedTabs tabs={tabs} value={tab} onChange={value => { setTab(value); }} label="View" controls={panelId}
            className="max-md:grid max-md:w-full max-md:grid-cols-[1fr_1.3fr_1fr] max-md:[&>button]:justify-center max-md:[&>button]:px-1 max-md:[&>button]:text-[10px] max-md:[&>button]:tracking-[0.1em]" />
        </div>
        <div className="flex gap-2.5">
          <SearchField label={searchProps.label} value={searchProps.value} onChange={event => searchProps.onChange(event.target.value)} className="w-full md:w-60" />
          <Button className="hidden md:inline-flex" aria-expanded={importing} onClick={() => importing ? setImporting(false) : openImport()}>Import LinkedIn</Button>
        </div>
      </header>

      {!companiesOn && <p className="mt-3 text-sm text-ink-60">The catch-up queue needs the Companies module, which is turned off in Settings → Modules.</p>}
      {error && <p role="alert" className="mt-4 border border-accent p-3 text-sm text-accent">{error}</p>}
      {companiesOn && queueFailed && <p role="alert" className="mt-4 border border-accent p-3 text-sm text-accent">The catch-up queue could not be loaded. Reload to retry.</p>}
      {companiesFailed && <p role="alert" className="mt-4 border border-accent p-3 text-sm text-accent">Companies could not be loaded, so company names and filters are missing. Reload to retry.</p>}
      <p role="status" className="mt-3 text-sm text-ink-60 empty:mt-0">{status}</p>

      {(importing || connections.length === 0) && <div ref={importPanel} className="mt-4"><ImportPanel busy={busy} onImport={importFiles} onClose={connections.length ? () => setImporting(false) : undefined} /></div>}

      <div id={panelId} role="tabpanel" aria-label={tabs.find(t => t.value === tab)?.label} className="mt-4 lg:mt-6">
        {tab === 'people' && <div className="grid gap-12 lg:grid-cols-[minmax(0,1fr)_300px]">
          <section aria-label="All people" className={cn('min-w-0 flex-col', showDetail ? 'hidden lg:flex' : 'flex')}>
            <div className="mb-3 grid grid-cols-2 gap-2 md:flex md:flex-wrap md:items-center">
              {companiesOn && <select aria-label="Filter by company" value={companyFilter} onChange={event => { setCompanyFilter(event.target.value); setLimit(PAGE); }} className={cn(select, 'w-full md:w-auto')}>
                <option value="all">All companies</option>
                <option value="linked">At tracked companies</option>
                <option value="unlinked">Not at tracked companies</option>
                {companies.map(c => <option key={c.slug} value={c.slug}>{c.title}</option>)}
              </select>}
              <select aria-label="Sort connections" value={sort} onChange={event => setSort(event.target.value as Sort)} className={cn(select, 'w-full md:w-auto')}>
                {Object.entries(SORT_LABELS).map(([value, label]) => <option key={value} value={value}>{label === 'A–Z' ? 'Name' : label}</option>)}
              </select>
              {employer && <Button size="sm" className="col-span-2" aria-label={`Clear employer filter: ${employer.name}`} onClick={() => { setEmployer(undefined); setLimit(PAGE); }}>{employer.name} ✕</Button>}
              <Button size="sm" className="ml-auto max-md:hidden" disabled={busy || !!draft} onClick={() => setDraft(newConnection())}>Add person</Button>
            </div>
            <div aria-hidden="true" className="hidden gap-4 border-b border-ink pb-1.5 md:grid md:grid-cols-[44px_minmax(0,1fr)_140px_110px] lg:grid-cols-[44px_minmax(0,1fr)_minmax(0,1fr)_140px_110px]">
              <span /><Label>Name</Label><Label className="hidden lg:block">Role</Label><Label>Company</Label><Label className="text-right">Last contact</Label>
            </div>
            <ul className="m-0 list-none p-0">
              {visible.slice(0, limit).map(c => {
                const on = c.id === selected?.id;
                return <li key={c.id}>
                  <button type="button" aria-pressed={on} disabled={!!draft} onClick={() => pick(on ? undefined : c.id)}
                    className={cn('grid !min-h-[62px] w-full cursor-pointer grid-cols-[38px_minmax(0,1fr)_auto] items-center gap-3 border-0 border-b border-solid border-ink-14 bg-transparent p-0 text-left text-ink hover:bg-ink-05 disabled:cursor-default md:!min-h-[46px] md:grid-cols-[44px_minmax(0,1fr)_140px_110px] md:gap-4 lg:grid-cols-[44px_minmax(0,1fr)_minmax(0,1fr)_140px_110px]', on && 'bg-ink-05')}>
                    <Avatar name={c.name} src={photoURL(c)} size={phone ? 34 : 30} muted={!on} />
                    <span className="flex min-w-0 flex-col">
                      <span className="truncate text-base font-light md:text-[15px]">{c.name}</span>
                      <span className="truncate text-xs text-ink-60 lg:hidden">
                        {c.role}<span className="md:hidden">{c.role && companyName(c) ? ' · ' : ''}{companyName(c)}</span>
                      </span>
                    </span>
                    <span className="hidden truncate text-[13px] text-ink-60 lg:block">{c.role}</span>
                    <span className="hidden min-w-0 items-center gap-2 text-[13px] font-light md:flex">
                      <span className="truncate">{companyName(c)}</span>
                      {tracked(c) && <BoardMark />}
                    </span>
                    <span title={c.lastContactedOn} className="text-right font-mono text-[10px] text-ink-60 md:text-[11px]">{phone ? (c.lastContactedOn ? formatDate(c.lastContactedOn).toUpperCase() : '') : shortDate(c.lastContactedOn)}</span>
                  </button>
                </li>;
              })}
              {visible.length === 0 && <li className="py-4 text-sm text-ink-60">No connections match.</li>}
            </ul>
            <div className="mt-2.5 flex flex-wrap items-center justify-between gap-3">
              <Label>Showing {Math.min(limit, visible.length)} of {visible.length} · {SORT_LABELS[sort]}</Label>
              {visible.length > limit && <Button size="sm" onClick={() => setLimit(limit + PAGE)}>Show {Math.min(PAGE, visible.length - limit)} more</Button>}
            </div>
          </section>

          <div className={cn('min-w-0 flex-col self-start', showDetail ? 'flex' : 'hidden lg:flex')}>
            {showDetail && <button type="button" onClick={() => { if (draft) setDraft(undefined); else pick(undefined); }}
              className="mb-4 flex min-h-11 cursor-pointer items-center self-start border-0 bg-transparent p-0 font-mono text-[11px] tracking-[0.14em] text-ink-60 hover:text-ink lg:hidden">← ALL PEOPLE</button>}
            <Box as="aside" aria-label={draft ? 'Edit person' : selected ? `About ${selected.name}` : 'Person details'} className="flex flex-col gap-2.5 p-5">
              {draft ? <ConnectionForm key={draft.id} draft={draft} isNew={!revisions[draft.id]} busy={busy} companies={companies} companiesOn={companiesOn} today={today}
                onChange={setDraft} onCancel={() => setDraft(undefined)}
                onSave={c => save(c, `${c.name} saved.`, true)} onRemove={remove} onPhoto={uploadPhoto} onRemovePhoto={removePhoto} />
                : selected ? <PersonDetail person={selected} today={today} busy={busy} companiesOn={companiesOn} companyName={companyName(selected)}
                  colleagues={connections.filter(c => !!employerKey(selected, board) && employerKey(c, board) === employerKey(selected, board)).length}
                  note={noteText} noteDirty={noteDirty} onNote={text => setNote({ id: selected.id, text })}
                  onSaveNote={() => void save({ ...selected, notes: noteText }, `Notes saved for ${selected.name}.`)}
                  onLog={() => void save(markCaughtUp({ ...selected, notes: noteText }, today), `Logged a catch-up with ${selected.name}.`)}
                  onQueue={() => void (selected.stage ? unqueue(selected) : moveTo(selected, 'to_contact'))}
                  onEdit={() => edit(selected)} />
                  : <p className="m-0 text-sm font-light text-ink-60">Pick someone to see their details and notes.</p>}
            </Box>
          </div>
        </div>}

        {tab === 'queue' && companiesOn && <CatchUpBoard connections={connections} today={queueToday} week={queueInfo?.week} query={query} busy={busy} drafting={!!draft}
          phone={phone} wide={wide} companyName={companyName} goal={goals && linkedGoal(goals)} goals={goals && linkableGoals(goals)} due={due}
          actions={{
            onMove: (c, stage) => void moveTo(c, stage),
            onRemove: unqueue,
            onSent: markSent,
            onSnooze: (c, until) => void snooze(c, until),
            onTarget: setTarget,
            onLink: goalId => void linkGoal(goalId),
            onCaughtUp: c => void save(markCaughtUp(c, today), `Logged a conversation with ${c.name}.`),
            onUnflag: c => void save({ ...c, queued: false }, `${c.name} is no longer flagged.`),
            onEdit: edit,
          }} />}

        {tab === 'employers' && <EmployersTab employers={network} query={employerQuery} limit={employerLimit} busy={busy} companiesOn={companiesOn}
          onMore={() => setEmployerLimit(employerLimit + EMPLOYER_PAGE)}
          onOpen={e => {
            setEmployer({ key: e.key, name: e.name });
            setCompanyFilter('all');
            setQuery('');
            setLimit(PAGE);
            setTab('people');
          }}
          canAdd={!companiesFailed} onAdd={addCompanies} />}
      </div>

      <div className="mt-5 grid grid-cols-2 gap-2.5 md:hidden">
        <Button aria-expanded={importing} onClick={() => importing ? setImporting(false) : openImport()}>Import LinkedIn</Button>
        <Button disabled={busy || !!draft} onClick={() => { setTab('people'); setDraft(newConnection()); }}>Add person</Button>
      </div>
    </div>
  );
}

/** ◆: this person's employer is on the companies board. */
function BoardMark() {
  return <><span aria-hidden="true" className="text-[9px]">◆</span><span className="ui-sr-only">on your companies board</span></>;
}

function Stat({ value, label }: { value: ReactNode; label: ReactNode }) {
  return <span className="flex min-w-0 flex-col">
    <span className="truncate font-mono text-[13px]">{value}</span>
    <Label className="truncate !text-[9px]">{label}</Label>
  </span>;
}

function PersonDetail({ person, today, busy, companiesOn, companyName, colleagues, note, noteDirty, onNote, onSaveNote, onLog, onQueue, onEdit }: {
  person: Connection; today: string; busy: boolean; companiesOn: boolean; companyName: string; colleagues: number;
  note: string; noteDirty: boolean; onNote: (text: string) => void; onSaveNote: () => void; onLog: () => void; onQueue: () => void; onEdit: () => void;
}) {
  const due = nextDue(person, today);
  const link = linkedIn(person);
  return <>
    <Avatar name={person.name} src={photoURL(person)} size={46} />
    <h2 className="m-0 text-2xl leading-[1.1] font-light">{person.name}</h2>
    <p className="m-0 text-[13px] text-ink-60">
      {person.role}{person.role && companyName ? ' · ' : ''}
      {person.companySlug && companiesOn ? <a href={companyHref(person.companySlug)} className="text-ink-60 underline-offset-2 hover:text-ink">{companyName}</a> : companyName}
    </p>
    <div className="mt-1 grid grid-cols-2 gap-y-3 border-t border-ink-30 pt-3">
      <Stat value={<span title={person.lastContactedOn}>{shortDate(person.lastContactedOn)}</span>} label="Last contact" />
      {companyName ? <Stat value={colleagues} label={`At ${companyName}`} /> : <Stat value={shortDate(person.connectedOn)} label="Connected" />}
      {companyName && <Stat value={shortDate(person.connectedOn)} label="Connected" />}
      <Stat value={due ? <span className={due <= today ? 'text-accent' : undefined}>{relativeDays(due, today).replace('ago', 'overdue')}</span> : '—'} label="Next catch-up" />
    </div>
    {companiesOn && person.stage && <Label className="text-ink">In catch-up queue · {STAGE_LABELS[person.stage]}</Label>}
    <label className="mt-2 flex flex-col gap-1.5">
      <Label className="!text-[9px]">Notes</Label>
      <textarea rows={4} maxLength={5000} value={note} disabled={busy} onChange={event => onNote(event.target.value)} placeholder="What you talked about, what to follow up on…" className={textarea} />
    </label>
    {noteDirty && <Button size="sm" disabled={busy} onClick={onSaveNote}>Save notes</Button>}
    <div className="mt-1 flex gap-2">
      <Button variant="primary" className="flex-1" disabled={busy} onClick={onLog}>Log catch-up</Button>
      {link && <a href={link} target="_blank" rel="noreferrer" className={cn(buttonClassName(), 'flex-1')}>LinkedIn ↗</a>}
    </div>
    <div className="flex gap-2">
      <Button size="sm" className="flex-1" disabled={busy} onClick={onEdit}>Edit</Button>
      {companiesOn && <Button size="sm" className="flex-1" aria-pressed={!!person.stage} disabled={busy} onClick={onQueue}>{person.stage ? 'Unqueue' : 'Queue'}</Button>}
    </div>
  </>;
}

function ImportPanel({ busy, onImport, onClose }: { busy: boolean; onImport: (form: HTMLFormElement) => void; onClose?: () => void }) {
  const file = 'mt-1.5 block w-full text-sm text-ink-60 file:mr-3 file:h-[34px] file:cursor-pointer file:border file:border-solid file:border-ink-50 file:bg-transparent file:px-3 file:font-mono file:text-[11px] file:tracking-[0.14em] file:text-ink file:uppercase';
  return <Box as="section" aria-labelledby="import-heading" className="mb-2 flex flex-col gap-3 p-5 text-sm">
    <SectionHeader id="import-heading" title="Import from LinkedIn">
      {onClose && <button type="button" onClick={onClose} className="ml-auto min-h-11 cursor-pointer border-0 bg-transparent p-0 font-mono text-[11px] tracking-[0.14em] text-ink-60 hover:text-ink">CLOSE</button>}
    </SectionHeader>
    <p className="m-0 font-light text-ink-60">On LinkedIn open <em>Settings → Data privacy → Get a copy of your data</em>, pick <em>Connections</em> and <em>Messages</em>, then unzip the export. Re-importing updates names, roles and companies but keeps your notes, cadences and catch-up stages. A company link, or a deliberate unlink, only changes when that person's LinkedIn company changes.</p>
    <form className="flex flex-wrap items-end gap-4" onSubmit={event => { event.preventDefault(); onImport(event.currentTarget); }}>
      <fieldset disabled={busy} className="contents">
        <label className="block min-w-0"><Label>Connections.csv (required)</Label><input name="connections" type="file" accept=".csv,text/csv" required className={file} /></label>
        <label className="block min-w-0"><Label>messages.csv (optional, sets last conversation)</Label><input name="messages" type="file" accept=".csv,text/csv" className={file} /></label>
        <Button type="submit" variant="primary">Import</Button>
      </fieldset>
    </form>
  </Box>;
}

function EmployersTab({ employers: all, query, limit, busy, companiesOn, canAdd, onMore, onOpen, onAdd }: {
  employers: Employer[]; query: string; limit: number; busy: boolean; companiesOn: boolean; canAdd: boolean;
  onMore: () => void; onOpen: (employer: Employer) => void; onAdd: (names: string[]) => void;
}) {
  const [picked, setPicked] = useState<string[]>([]);
  const q = query.trim().toLowerCase();
  const shown = q ? all.filter(e => e.name.toLowerCase().includes(q)) : all;
  const most = all[0]?.count ?? 1;
  const untracked = all.filter(e => !e.slug).slice(0, 5);
  // Names leave the list once added (or when they stop being untracked), so a failed add keeps them ticked.
  const chosen = picked.filter(name => untracked.some(e => e.name === name));
  const width = (n: number) => `${(n / most) * 100}%`;
  return <div className="grid gap-12 lg:grid-cols-[minmax(0,1fr)_300px]">
    <section aria-labelledby="employers-heading" className="flex min-w-0 flex-col">
      <SectionHeader id="employers-heading" title="Where your network works" meta="People per employer" className="mb-2 max-md:hidden" />
      <ul className="m-0 list-none p-0">
        {shown.slice(0, limit).map(e => <li key={e.key}>
          <button type="button" onClick={() => onOpen(e)} aria-label={`${e.name}: ${e.count} ${e.count === 1 ? 'person' : 'people'}${e.slug ? ', on your companies board' : ''}`}
            className="flex w-full cursor-pointer flex-col gap-[7px] border-0 border-b border-solid border-ink-14 bg-transparent px-0 py-2.5 text-left text-ink hover:bg-ink-05 md:grid md:min-h-[38px] md:grid-cols-[180px_minmax(0,1fr)_36px] md:items-center md:gap-4 md:py-0">
            <span className="flex min-w-0 items-center justify-between gap-2 md:justify-start">
              <span className="flex min-w-0 items-center gap-2">
                <span className="truncate text-[15px] font-light">{e.name}</span>
                {e.slug && <span aria-hidden="true" className="text-[9px]">◆</span>}
              </span>
              <span className="font-mono text-xs md:hidden">{e.count}</span>
            </span>
            <span className="relative block h-2 md:h-2.5">
              {e.slug
                ? <span className="absolute bottom-0 left-0 h-1.5 bg-ink md:h-[7px]" style={{ width: width(e.count) }} />
                : <HatchFill className="absolute bottom-0 left-0 h-1.5 md:h-[7px]" style={{ width: width(e.count) }} />}
              <MeasurementLine className="!absolute bottom-0 left-0" style={{ width: width(e.count) }} />
            </span>
            <span className="hidden text-right font-mono text-xs md:block">{e.count}</span>
          </button>
        </li>)}
        {shown.length === 0 && <li className="py-4 text-sm text-ink-60">{all.length ? 'No employers match.' : 'Import your LinkedIn connections to see where they work.'}</li>}
      </ul>
      <div className="mt-2.5 flex flex-wrap items-center justify-between gap-3">
        {companiesOn ? <Label>◆ On your companies board</Label> : <span />}
        {shown.length > limit && <Button size="sm" onClick={onMore}>Show {Math.min(EMPLOYER_PAGE, shown.length - limit)} more</Button>}
      </div>
    </section>
    {companiesOn && untracked.length > 0 && <aside aria-labelledby="untracked-heading" className="flex flex-col gap-2.5 text-sm font-light">
      <Label id="untracked-heading">Not on your board yet</Label>
      <ul className="m-0 flex list-none flex-col p-0">
        {untracked.map(e => <li key={e.key}>
          <label className="flex min-h-11 cursor-pointer items-center gap-3">
            <Checkbox checked={chosen.includes(e.name)} disabled={busy} onChange={event => setPicked(current => event.target.checked ? [...current, e.name] : current.filter(name => name !== e.name))} />
            <span>{e.name} · {e.count} {e.count === 1 ? 'person' : 'people'}</span>
          </label>
        </li>)}
      </ul>
      <Button className="mt-2.5" disabled={busy || !canAdd || chosen.length === 0} onClick={() => onAdd(chosen)}>Add to companies</Button>
    </aside>}
  </div>;
}

interface FormProps {
  draft: Connection; isNew: boolean; busy: boolean; companies: CompanyOption[]; companiesOn: boolean; today: string;
  onChange: (c: Connection) => void; onCancel: () => void; onSave: (c: Connection) => void;
  onRemove: (c: Connection) => void; onPhoto: (c: Connection, file: File) => void; onRemovePhoto: (c: Connection) => void;
}

function Field({ label, className, children }: { label: ReactNode; className?: string; children: ReactNode }) {
  return <label className={cn('ui-field !gap-1', className)}><Label>{label}</Label>{children}</label>;
}

function ConnectionForm({ draft, isNew, busy, companies, companiesOn, today, onChange, onCancel, onSave, onRemove, onPhoto, onRemovePhoto }: FormProps) {
  const set = <K extends keyof Connection>(key: K, value: Connection[K]) => onChange({ ...draft, [key]: value });
  const optional = (key: 'url' | 'connectedOn' | 'lastContactedOn', value: string) => {
    const { [key]: _old, ...rest } = draft;
    onChange(value ? { ...rest, [key]: value } : rest);
  };
  const input = 'ui-field__input !h-10 !text-[15px]';
  const link = 'cursor-pointer border-0 bg-transparent p-0 font-mono text-[11px] tracking-[0.14em] uppercase';
  return (
    <form aria-label={isNew ? 'New connection' : `Edit ${draft.name}`} className="flex flex-col gap-4 text-sm"
      onSubmit={event => { event.preventDefault(); onSave(draft); }}>
      <h2 className="m-0 text-2xl leading-[1.1] font-light">{isNew ? 'New connection' : `Edit ${draft.name}`}</h2>
      <fieldset disabled={busy} className="m-0 flex min-w-0 flex-col gap-4 border-0 p-0">
        <div className="flex items-center gap-4">
          <Avatar name={draft.name || '?'} src={photoURL(draft)} size={46} />
          {isNew ? <p className="m-0 text-xs text-ink-60">Save first to add a photo.</p> : <div className="flex flex-wrap gap-x-4 gap-y-1">
            <label className={cn(link, 'flex min-h-11 items-center text-ink hover:text-ink-hover')}>{draft.photo ? 'Replace photo' : 'Upload photo'}
              <input type="file" accept="image/*" className="ui-sr-only" onChange={event => { const file = event.target.files?.[0]; event.target.value = ''; if (file) onPhoto(draft, file); }} />
            </label>
            {draft.photo && <button type="button" className={cn(link, 'min-h-11 text-accent')} onClick={() => onRemovePhoto(draft)}>Remove photo</button>}
          </div>}
        </div>
        <Field label="Name"><input required maxLength={200} value={draft.name} onChange={e => set('name', e.target.value)} className={input} /></Field>
        <Field label="Role"><input maxLength={200} value={draft.role} onChange={e => set('role', e.target.value)} className={input} /></Field>
        <Field label="Company (as on LinkedIn)"><input maxLength={200} value={draft.companyName} onChange={e => set('companyName', e.target.value)} className={input} /></Field>
        {companiesOn && <Field label="Tracked company">
          <select value={draft.companySlug ?? ''} className={cn(input, 'bg-paper')} onChange={e => {
            const company = companies.find(c => c.slug === e.target.value);
            const { companySlug: _old, ...rest } = draft;
            onChange(company ? { ...rest, companySlug: company.slug, companyName: draft.companyName || company.title } : rest);
          }}>
            <option value="">Not linked</option>
            {companies.map(c => <option key={c.slug} value={c.slug}>{c.title}</option>)}
          </select>
        </Field>}
        <Field label="Email"><input type="email" maxLength={254} value={draft.email} onChange={e => set('email', e.target.value)} className={input} /></Field>
        <Field label="LinkedIn URL"><input type="url" maxLength={2000} value={draft.url ?? ''} onChange={e => optional('url', e.target.value)} className={input} /></Field>
        <Field label="Connected on"><input type="date" value={draft.connectedOn ?? ''} onChange={e => optional('connectedOn', e.target.value)} className={input} /></Field>
        <div className="flex items-end gap-2">
          <Field label="Last conversation" className="flex-1"><input type="date" value={draft.lastContactedOn ?? ''} onChange={e => optional('lastContactedOn', e.target.value)} className={input} /></Field>
          <Button size="sm" onClick={() => optional('lastContactedOn', today)}>Today</Button>
        </div>
        <Field label="Catch up every">
          <select value={draft.cadenceDays ?? ''} className={cn(input, 'bg-paper')} onChange={e => {
            const { cadenceDays: _old, ...rest } = draft;
            onChange(e.target.value ? { ...rest, cadenceDays: Number(e.target.value) } : rest);
          }}>
            <option value="">No cadence</option>
            {CADENCES.map(days => <option key={days} value={days}>{days < 365 ? `${days} days` : 'Year'}</option>)}
          </select>
        </Field>
        <Field label="Tags (comma separated)">
          <input value={draft.tags.join(', ')} onChange={e => set('tags', e.target.value.split(',').map(t => t.trim()).filter(Boolean).slice(0, 30))} className={input} />
        </Field>
        {companiesOn && <label className="flex min-h-11 cursor-pointer items-center gap-3"><Checkbox checked={draft.queued} onChange={e => set('queued', e.target.checked)} />Flag for a catch-up</label>}
        <label className="flex flex-col gap-1.5"><Label>Notes</Label>
          <textarea rows={4} maxLength={5000} value={draft.notes} onChange={e => set('notes', e.target.value)} className={textarea} placeholder="How you met, what they work on, what to follow up on…" />
        </label>
        <div className="flex flex-wrap gap-2">
          <Button type="submit" variant="primary" className="flex-1">Save</Button>
          <Button className="flex-1" onClick={onCancel}>Cancel</Button>
        </div>
        {!isNew && <Button variant="danger" onClick={() => onRemove(draft)}>Remove connection</Button>}
      </fieldset>
    </form>
  );
}
