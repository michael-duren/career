import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import Markdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import MarkdownPreview from './MarkdownPreview';
import { LogExplorer } from './LogExplorer';
import { JournalEditor, clearDraft, entryId, fetchEntry, readDraft, signInAgain, type JournalEntry, type JournalItem } from './JournalEditor';
import { Button, Label, MeasurementLine, SearchField, SegmentedTabs, buttonClassName } from './ui';
import { buildLogDocs, trackLabel, type WeekSummary } from '../lib/logs';
import { hoursText, journalCounts, latestEntryDate, parseWeekBody, personalDate, personalMatches, sectionEmpty, staleNote, weekBounds, weekHoursLabel, weekMatches, weekRange, type JournalSection } from '../lib/journal';
import { moduleEnabled } from '../lib/modules';
import { localDate } from '../lib/workspace';
import type { GoalHours } from '../lib/goal-progress';
import type { Goal } from '../lib/timeline';
import type { JournalWeek, PersonalJournalEntry } from '../lib/workspace';

const TABS = [
  { value: 'work', label: 'Work' },
  { value: 'personal', label: 'Personal' },
  { value: 'logs', label: 'Logs' },
] as const;
type Tab = (typeof TABS)[number]['value'];
const isTab = (value: string | null): value is Tab => TABS.some(tab => tab.value === value);

type Item<T> = { entry: T; revision: string };
/** Goal hours borrowed from the Schedule module; `on: false` while it (or Timeline) is turned off. */
type Hours = { on: false } | { on: true; goals: Goal[]; byGoal: Record<string, GoalHours>; error?: string };

async function listAll<T>(path: string): Promise<Item<T>[]> {
  const items: Item<T>[] = [];
  let offset = 0;
  do {
    const response = await fetch(`${path}${path.includes('?') ? '&' : '?'}limit=100&offset=${offset}`, { cache: 'no-store' });
    if (response.status === 401) signInAgain();
    const page = await response.json().catch(() => ({}));
    if (!response.ok) throw new Error(page.error || 'Request failed. Please retry.');
    items.push(...page.entries);
    offset = page.nextOffset ?? -1;
  } while (offset >= 0);
  return items;
}
async function loadHours(): Promise<Hours> {
  if (!await moduleEnabled('schedule')) return { on: false };
  try {
    const [goals, hours] = await Promise.all([fetch('/api/goals', { cache: 'no-store' }), fetch('/api/scheduler/goal-hours', { cache: 'no-store' })]);
    if (goals.status === 401 || hours.status === 401) signInAgain();
    // A turned-off module's API is 404: Schedule (or Timeline) was turned off after the settings check.
    if (goals.status === 404 || hours.status === 404) return { on: false };
    if (!goals.ok || !hours.ok) throw new Error(`goals ${goals.status}, goal hours ${hours.status}`);
    return { on: true, goals: (await goals.json()).goals ?? [], byGoal: (await hours.json()).goals ?? {} };
  } catch (e) {
    console.warn('Goal hours could not load.', e);
    return { on: true, goals: [], byGoal: {}, error: 'Goal hours could not load.' };
  }
}

const DESKTOP = '(min-width: 768px)';
function useDesktop() {
  const [desktop, setDesktop] = useState(() => window.matchMedia(DESKTOP).matches);
  useEffect(() => {
    const query = window.matchMedia(DESKTOP);
    const update = () => setDesktop(query.matches);
    query.addEventListener('change', update);
    return () => query.removeEventListener('change', update);
  }, []);
  return desktop;
}

function setUrlParam(name: string, value: string | null) {
  const url = new URL(location.href);
  if (value === null) url.searchParams.delete(name); else url.searchParams.set(name, value);
  history.replaceState(history.state, '', url);
}

export function JournalHub() {
  const [params] = useState(() => new URLSearchParams(location.search));
  const [tab, setTabState] = useState<Tab>(() => { const value = params.get('tab'); return isTab(value) ? value : 'work'; });
  const [initialId] = useState(() => params.get('id'));
  const [visited, setVisited] = useState<Set<Tab>>(() => new Set([tab]));
  const [query, setQuery] = useState('');
  const [weeks, setWeeks] = useState<Item<JournalWeek>[] | null>(null);
  const [personal, setPersonal] = useState<Item<PersonalJournalEntry>[] | null>(null);
  const [hours, setHours] = useState<Hours | null>(null);
  const [error, setError] = useState('');
  const [newPersonal, setNewPersonal] = useState(0);
  const loads = useRef(0);
  const desktop = useDesktop();

  async function load() {
    const request = ++loads.current;
    try {
      const [nextWeeks, nextPersonal, nextHours] = await Promise.all([
        listAll<JournalWeek>('/api/entries/week?view=detail'), listAll<PersonalJournalEntry>('/api/entries/personal'), loadHours(),
      ]);
      if (request !== loads.current) return;
      setWeeks(nextWeeks); setPersonal(nextPersonal); setHours(nextHours); setError('');
    } catch (e) {
      if (request === loads.current) setError(e instanceof Error ? e.message : 'The journal could not load.');
    }
  }
  useEffect(() => {
    void load();
    const refresh = () => { void load(); };
    window.addEventListener('workspace-saved', refresh);
    return () => window.removeEventListener('workspace-saved', refresh);
  }, []);

  function setTab(next: Tab) {
    setTabState(next); setVisited(current => new Set(current).add(next));
    setUrlParam('tab', next === 'work' ? null : next);
    // An opened entry id belongs to the tab it was opened from.
    setUrlParam('id', null);
  }
  function newPersonalEntry() {
    if (tab !== 'personal') setTab('personal');
    setNewPersonal(count => count + 1);
  }

  const sortedWeeks = useMemo(() => [...(weeks ?? [])].sort((a, b) => b.entry.dates.localeCompare(a.entry.dates)), [weeks]);
  const stale = weeks ? staleNote(latestEntryDate(weeks.map(item => item.entry)), localDate()) : null;
  const panelId = 'journal-panel';

  return <div className="flex flex-col pb-12">
    <header className="flex flex-col gap-4 md:flex-row md:flex-wrap md:items-end md:justify-between">
      <div className="flex items-end justify-between gap-4">
        <div className="flex flex-col gap-2">
          <Label>{weeks && personal ? journalCounts(weeks.length, personal.length) : 'Loading…'}</Label>
          <h1 className="m-0 font-extralight !text-[40px] !leading-[0.9] md:!text-[52px]">Journal</h1>
        </div>
        <NewEntry tab={tab} onPersonal={newPersonalEntry} className="md:hidden" compact />
      </div>
      <div className="flex flex-col gap-2.5 md:flex-row md:items-center">
        {tab !== 'logs' && <SearchField label="Search journal" value={query} onChange={event => setQuery(event.target.value)} className="order-2 w-full md:order-1 md:w-[200px]" />}
        <SegmentedTabs tabs={TABS} value={tab} onChange={setTab} label="Journal" controls={panelId}
          className="order-1 max-md:grid max-md:w-full max-md:grid-cols-3 max-md:[&>button]:h-10 max-md:[&>button]:justify-center max-md:[&>button]:text-[10px] max-md:[&>button]:tracking-[0.12em] md:order-2" />
        <NewEntry tab={tab} onPersonal={newPersonalEntry} className="order-3 max-md:!hidden" />
      </div>
    </header>

    {error && <div role="alert" className="mt-6 flex flex-wrap items-center gap-3 border border-accent p-4 text-sm">{error}<Button size="sm" onClick={() => void load()}>Retry</Button></div>}
    {!error && (!weeks || !personal || !hours) && <p role="status" className="mt-8 font-mono text-xs text-ink-60">Loading your journal…</p>}

    {weeks && personal && hours && <div id={panelId} role="tabpanel" aria-label={TABS.find(item => item.value === tab)?.label} className="mt-5 md:mt-8">
      {visited.has('work') && <div hidden={tab !== 'work'}>
        <WorkPanel weeks={sortedWeeks} query={query} hours={hours} desktop={desktop} stale={stale} initialId={(params.get('tab') ?? 'work') === 'work' ? initialId : null}
          onSaved={saved => setWeeks(current => [...(current ?? []).filter(item => item.entry.slug !== saved.entry.slug), saved])}
          onDeleted={id => setWeeks(current => (current ?? []).filter(item => item.entry.slug !== id))} />
      </div>}
      {visited.has('personal') && <div hidden={tab !== 'personal'}>
        <PersonalPanel entries={personal} query={query} desktop={desktop} initialId={params.get('tab') === 'personal' ? initialId : null} newSignal={newPersonal}
          onSaved={saved => setPersonal(current => [...(current ?? []).filter(item => item.entry.id !== saved.entry.id), saved])}
          onDeleted={id => setPersonal(current => (current ?? []).filter(item => item.entry.id !== id))} />
      </div>}
      {visited.has('logs') && <div hidden={tab !== 'logs'}>
        <LogsPanel weeks={sortedWeeks} query={query} onQuery={setQuery} desktop={desktop} />
      </div>}
    </div>}
  </div>;
}

/** + NEW ENTRY: a dated work entry (/journal/new), or a personal entry on the Personal tab. */
function NewEntry({ tab, onPersonal, className, compact = false }: { tab: Tab; onPersonal: () => void; className?: string; compact?: boolean }) {
  const label = tab === 'personal' ? 'New personal entry' : 'New work entry';
  const classes = `${buttonClassName({ variant: 'primary', icon: compact })} ${compact ? '!text-[18px]' : ''} ${className ?? ''}`;
  const text = compact ? '+' : '+ New entry';
  return tab === 'personal'
    ? <button type="button" className={classes} aria-label={compact ? label : undefined} onClick={onPersonal}>{text}</button>
    : <a className={classes} href="/journal/new" aria-label={compact ? label : undefined}>{text}</a>;
}

function MarkdownLink({ journal }: { journal: 'work' | 'personal' }) {
  return <a className="ui-label mt-4 inline-block self-start py-2 hover:text-ink" href={`/api/agent-context?format=markdown&journal=${journal}`}>Read as Markdown →</a>;
}

function StaleNote({ text, className = '' }: { text: string | null; className?: string }) {
  return text ? <p className={`m-0 font-hand text-sm text-accent ${className}`}>{text}</p> : null;
}

// ---------------------------------------------------------------------------
// Work

type Editing = { item: JournalItem<JournalEntry>; isNew: boolean; recovered?: boolean };

/** Confirms leaving an edit with unsaved changes. */
function useLeaveGuard() {
  const dirty = useRef(false);
  return {
    setDirty: (value: boolean) => { dirty.current = value; },
    confirmLeave: () => !dirty.current || window.confirm('Discard the unsaved draft?'),
  };
}

function WorkPanel({ weeks, query, hours, desktop, stale, initialId, onSaved, onDeleted }: {
  weeks: Item<JournalWeek>[]; query: string; hours: Hours; desktop: boolean; stale: string | null; initialId: string | null;
  onSaved: (saved: Item<JournalWeek>) => void; onDeleted: (id: string) => void;
}) {
  const filtered = weeks.filter(item => weekMatches(item.entry, query));
  const [selectedId, setSelectedId] = useState<string | null>(initialId);
  // Phones expand one week inline; the requested (or newest) week starts open.
  const [openId, setOpenId] = useState<string | null | undefined>(undefined);
  const [editing, setEditing] = useState<Editing | null>(() => {
    const draft = readDraft<JournalWeek>('week');
    return draft ? { item: draft, isNew: !weeks.some(item => item.entry.slug === draft.entry.slug), recovered: true } : null;
  });
  const guard = useLeaveGuard();
  const missing = initialId && !weeks.some(item => item.entry.slug === initialId);
  const selected = filtered.find(item => item.entry.slug === selectedId) ?? (desktop ? filtered[0] : undefined);
  const open = openId === undefined ? (filtered.find(item => item.entry.slug === initialId) ?? filtered[0])?.entry.slug : openId;

  function pick(id: string) {
    if (editing && !guard.confirmLeave()) return;
    if (editing) { clearDraft('week'); setEditing(null); }
    setSelectedId(id); setEditError('');
    if (!desktop) setOpenId(open === id ? null : id);
    setUrlParam('id', id);
  }
  const [editError, setEditError] = useState('');
  // Edit from the entry's current saved revision, not the one the list was loaded with.
  const edit = (item: Item<JournalWeek>) => {
    setEditError('');
    fetchEntry<JournalWeek>('week', item.entry.slug)
      .then(latest => latest ? setEditing({ item: latest, isNew: false }) : setEditError('This week was deleted elsewhere.'))
      .catch(e => setEditError(e instanceof Error ? e.message : 'This week could not load.'));
  };
  const editor = editing && <JournalEditor key={entryId(editing.item.entry)} kind="week" item={editing.item} isNew={editing.isNew} recovered={editing.recovered} onDirtyChange={guard.setDirty}
    onSaved={saved => { onSaved(saved as Item<JournalWeek>); setEditing(null); setSelectedId(entryId(saved.entry)); }}
    onCancel={() => setEditing(null)}
    onDeleted={id => { onDeleted(id); setEditing(null); setSelectedId(null); setUrlParam('id', null); }} />;

  if (editor && !desktop) return editor;

  const list = <nav aria-label="Weeks" className="flex min-w-0 flex-col">
    <h2 className="ui-section-header__title mb-2 max-md:hidden">Weeks{yearsLabel(filtered)}</h2>
    {missing && <p role="alert" className="m-0 mb-2 text-sm text-ink-60">That week was not found. Choose another week.</p>}
    {filtered.length === 0 && <p className="m-0 py-3 text-sm font-light text-ink-60">{weeks.length ? 'No weeks match your search.' : 'No work weeks yet. Use + New entry to write the first one.'}</p>}
    {filtered.map(item => {
      const week = item.entry, current = desktop ? selected?.entry.slug === week.slug : open === week.slug;
      return <div key={week.slug} className="border-b border-ink-14">
        <button type="button" onClick={() => pick(week.slug)}
          aria-current={desktop && current ? 'true' : undefined} aria-expanded={desktop ? undefined : current}
          className={`grid w-full cursor-pointer items-center gap-3 border-0 bg-transparent text-left text-ink hover:bg-ink-05 ${desktop ? `min-h-11 grid-cols-[44px_minmax(0,1fr)] pl-3 ${current ? 'bg-ink-05 shadow-[inset_2px_0_0_var(--ink)]' : ''}` : 'min-h-[52px] grid-cols-[50px_minmax(0,1fr)_14px] pl-0'}`}>
          <span className="font-mono text-xs">W{week.week}</span>
          <span className={`font-light opacity-75 ${desktop ? 'text-[13px]' : 'text-sm'}`}>{weekRange(week.dates)}</span>
          {!desktop && <span aria-hidden="true" className="opacity-50">{current ? '−' : '+'}</span>}
        </button>
        {!desktop && current && <div className="flex min-w-0 flex-col gap-1.5 pb-4 pl-[62px]">
          <WeekSections week={week} compact />
          {editError && <p role="alert" className="m-0 text-sm">{editError}</p>}
          <Button size="sm" className="mt-2 self-start" onClick={() => edit(item)} aria-label={`Edit week ${week.week}`}>Edit</Button>
        </div>}
      </div>;
    })}
    <MarkdownLink journal="work" />
  </nav>;

  if (!desktop) return <div className="flex flex-col"><StaleNote text={stale} className="mb-2" />{list}</div>;
  return <div className="grid gap-8 md:grid-cols-[200px_minmax(0,1fr)] xl:grid-cols-[250px_minmax(0,1fr)] xl:gap-14">
    {list}
    <article className="relative flex min-h-[420px] min-w-0 flex-col border-l border-ink-14 pl-6 xl:pl-10">
      {editError && !editor && <p role="alert" className="m-0 mb-3 text-sm">{editError}</p>}
      {editor || (selected ? <WeekArticle item={selected} hours={hours} onEdit={() => edit(selected)} /> : <p className="m-0 text-[15px] font-light text-ink-60">Choose a week to read it.</p>)}
      <StaleNote text={stale} className="mt-auto self-end pt-8" />
    </article>
  </div>;
}

function yearsLabel(weeks: Item<JournalWeek>[]) {
  const years = [...new Set(weeks.map(item => item.entry.year))].sort();
  return years.length ? ` · ${years.length > 2 ? `${years[0]}–${years.at(-1)}` : years.join(', ')}` : '';
}

function WeekArticle({ item, hours, onEdit }: { item: Item<JournalWeek>; hours: Hours; onEdit: () => void }) {
  const week = item.entry;
  const start = weekBounds(week.dates)?.start ?? '';
  const goals = hours.on ? hours.goals.flatMap(goal => {
    const value = hours.byGoal[goal.id]?.weekHours[start] ?? 0;
    return value > 0 ? [{ id: goal.id, title: goal.title, hours: value }] : [];
  }) : [];
  const legacy = Object.entries(week.hours ?? {}).filter(([, value]) => value > 0);
  const goalTotal = goals.reduce((sum, goal) => sum + goal.hours, 0);
  const legacyTotal = legacy.reduce((sum, [, value]) => sum + value, 0);
  return <>
    <div className="flex items-end justify-between gap-4">
      <div className="flex flex-col gap-1.5"><Label>{weekRange(week.dates)}, {week.year}</Label><h2 className="m-0 text-[32px] font-light leading-tight">Week {week.week}</h2></div>
      <Button onClick={onEdit} aria-label={`Edit week ${week.week}`}>Edit</Button>
    </div>
    <div className="mt-4 flex items-center gap-3">
      <MeasurementLine className="flex-1" /><Label className="!text-[9px]">{hours.on && hours.error ? 'Hours unavailable' : weekHoursLabel(goalTotal, legacyTotal)}</Label><MeasurementLine className="flex-1" />
    </div>
    <div className="mt-2 flex flex-col gap-1">
      {hours.on && hours.error && <p role="alert" className="m-0 text-xs text-ink-60">{hours.error}</p>}
      {goals.length > 0 && <Label as="p" className="m-0">{goals.map(goal => `${goal.title} ${hoursText(goal.hours)} h`).join(' · ')}</Label>}
      {legacy.length > 0 && <Label as="p" className="m-0">Earlier tracks: {legacy.map(([key, value]) => `${trackLabel(key)} ${hoursText(value)} h`).join(' · ')}</Label>}
      {hours.on && <a className="ui-label self-start py-1 hover:text-ink" href="/weekly-scheduler">Log work →</a>}
    </div>
    <WeekSections week={week} />
  </>;
}

/** What I did (by day), Blockers and Notes as ruled lists; anything richer renders as Markdown. */
function WeekSections({ week, compact = false }: { week: JournalWeek; compact?: boolean }) {
  const sections = parseWeekBody(week.body ?? '', weekBounds(week.dates)?.start ?? null);
  if (compact) {
    const [what, ...rest] = sections;
    const empty = rest.filter(sectionEmpty);
    return <>
      {sectionEmpty(what) ? <span className="text-[15px] font-light opacity-60">Nothing noted.</span> : <SectionBody section={what} compact />}
      {rest.filter(section => !sectionEmpty(section)).map((section, index) => <div key={index} className="mt-2 flex flex-col gap-1.5"><Label className="!text-[9px] !text-ink">{section.title || 'More'}</Label><SectionBody section={section} compact /></div>)}
      {empty.length > 0 && <Label className="mt-2 !text-[9px]">{empty.map(section => `${section.title} · none`).join(' · ')}</Label>}
    </>;
  }
  return <>{sections.map((section, index) => <section key={index} aria-label={section.title || 'More'}>
    <h3 className="ui-label mb-2.5 mt-7 !text-ink">{section.title || 'More'}</h3>
    {sectionEmpty(section) ? <span className="text-sm font-light opacity-45">Nothing noted.</span> : <SectionBody section={section} />}
  </section>)}</>;
}

function SectionBody({ section, compact = false }: { section: JournalSection; compact?: boolean }) {
  if (section.rich) return <MarkdownPreview ink body={section.markdown} />;
  return <>{section.groups.filter(group => group.items.length || !compact).map((group, index) => <div key={index} className="flex min-w-0 flex-col">
    {group.label && <Label className={`!text-[9px] ${compact ? '!text-ink' : 'mb-2'} ${index > 0 ? 'mt-3' : ''}`}>{group.label}</Label>}
    {compact
      ? group.items.map((text, i) => <span key={i} className="break-words text-[15px] font-light leading-normal">— <InlineMarkdown text={text} /></span>)
      : <ul className="m-0 flex list-none flex-col p-0">{group.items.map((text, i) => <li key={i} className="flex gap-3.5 border-b border-ink-14 py-1.5 text-[17px] font-light leading-normal"><span aria-hidden="true" className="opacity-40">—</span><span className="min-w-0 break-words"><InlineMarkdown text={text} /></span></li>)}</ul>}
  </div>)}</>;
}

const INLINE = ['strong', 'em', 'code', 'a', 'del'];
/** One list item's inline Markdown (bold, italics, code, links); block wrappers are dropped and raw HTML is not rendered. */
function InlineMarkdown({ text }: { text: string }) {
  return <Markdown remarkPlugins={[remarkGfm]} allowedElements={INLINE} unwrapDisallowed
    components={{ a: ({ node: _node, ...props }) => <a {...props} className="text-ink underline underline-offset-[3px] hover:text-ink-hover" />, code: ({ node: _node, ...props }) => <code {...props} className="bg-ink-07 px-1 font-mono text-[0.9em]" /> }}>{text}</Markdown>;
}

// ---------------------------------------------------------------------------
// Personal

function LockDrawing() {
  return <svg aria-hidden="true" width="28" height="34" viewBox="0 0 28 34" fill="none" stroke="currentColor" strokeWidth="1"><rect x="1" y="14" width="26" height="19" /><path d="M6 14 V9 a8 8 0 0 1 16 0 V14" /></svg>;
}

function PersonalPanel({ entries, query, desktop, initialId, newSignal, onSaved, onDeleted }: {
  entries: Item<PersonalJournalEntry>[]; query: string; desktop: boolean; initialId: string | null; newSignal: number;
  onSaved: (saved: Item<PersonalJournalEntry>) => void; onDeleted: (id: string) => void;
}) {
  const [editing, setEditing] = useState<Editing | null>(() => {
    const draft = readDraft<PersonalJournalEntry>('personal');
    return draft ? { item: draft, isNew: !entries.some(item => item.entry.id === draft.entry.id), recovered: true } : null;
  });
  // Private until asked: a link straight to an entry, a recovered draft or + New entry opens it.
  const [unlocked, setUnlocked] = useState(() => !!initialId || !!editing);
  const [selectedId, setSelectedId] = useState<string | null>(initialId);
  const [details, setDetails] = useState<Record<string, Item<PersonalJournalEntry> | 'missing'>>({});
  const [error, setError] = useState('');
  const guard = useLeaveGuard();
  const sorted = [...entries].sort((a, b) => (b.entry.date || '').localeCompare(a.entry.date || '') || (b.entry.updatedAt ?? '').localeCompare(a.entry.updatedAt ?? ''));
  const filtered = sorted.filter(item => personalMatches(item.entry, query));
  const selected = filtered.find(item => item.entry.id === selectedId) ?? (desktop ? filtered[0] : undefined);
  const missing = initialId && !entries.some(item => item.entry.id === initialId);
  // Starts at 0 so a + New entry clicked while the journal was still loading still opens the editor.
  const handled = useRef(0);

  useEffect(() => {
    if (newSignal === handled.current) return;
    handled.current = newSignal;
    if (editing && !guard.confirmLeave()) return;
    clearDraft('personal');
    setUnlocked(true);
    setEditing({ item: { entry: { id: crypto.randomUUID(), title: '', date: localDate(), description: '', tags: [], body: '' }, revision: null }, isNew: true });
  }, [newSignal]);

  // Lists carry no body: fetch the full entry (and its current revision) when one is shown.
  const shown = unlocked && selected ? selected.entry.id : null;
  useEffect(() => {
    if (!shown || details[shown]) return;
    let live = true;
    fetch(`/api/entries/personal?id=${encodeURIComponent(shown)}`, { cache: 'no-store' })
      .then(async response => {
        if (response.status === 401) signInAgain();
        if (response.status === 404) return 'missing' as const;
        const result = await response.json().catch(() => ({}));
        if (!response.ok) throw new Error(result.error || 'This entry could not load.');
        return result as Item<PersonalJournalEntry>;
      })
      .then(detail => { if (live) { setDetails(current => ({ ...current, [shown]: detail })); setError(''); } })
      .catch(e => { if (live) setError(e instanceof Error ? e.message : 'This entry could not load.'); });
    return () => { live = false; };
  }, [shown, details]);
  // A save anywhere can change any entry; drop cached bodies so they reload.
  const firstEntries = useRef(entries);
  useEffect(() => { if (entries !== firstEntries.current) setDetails({}); }, [entries]);

  function pick(id: string) {
    if (editing && !guard.confirmLeave()) return;
    if (editing) { clearDraft('personal'); setEditing(null); }
    setSelectedId(!desktop && selected?.entry.id === id ? null : id);
    setUrlParam('id', id);
  }

  if (!unlocked) return <button type="button" onClick={() => setUnlocked(true)}
    className="flex w-full cursor-pointer flex-col items-center gap-2.5 border-0 bg-transparent py-16 text-ink hover:bg-ink-05 md:py-28">
    <LockDrawing />
    <span className="text-lg font-light md:text-xl">{entries.length} personal entr{entries.length === 1 ? 'y' : 'ies'}</span>
    <Label>Private · {desktop ? 'open to read' : 'tap to open'}</Label>
  </button>;

  const editor = editing && <JournalEditor key={entryId(editing.item.entry)} kind="personal" item={editing.item} isNew={editing.isNew} recovered={editing.recovered} onDirtyChange={guard.setDirty}
    onSaved={saved => { onSaved(saved as Item<PersonalJournalEntry>); setEditing(null); setSelectedId(entryId(saved.entry)); }}
    onCancel={() => setEditing(null)}
    onDeleted={id => { onDeleted(id); setEditing(null); setSelectedId(null); setUrlParam('id', null); }} />;
  if (editor && !desktop) return editor;

  const reader = (item: Item<PersonalJournalEntry>, compact: boolean): ReactNode => {
    const detail = details[item.entry.id];
    if (detail === 'missing') return <p className="m-0 text-sm text-ink-60">This entry was deleted elsewhere.</p>;
    if (!detail) return error ? <p role="alert" className="m-0 text-sm">{error}</p> : <p className="m-0 font-mono text-xs text-ink-60">Opening…</p>;
    const entry = detail.entry;
    const editButton = <Button size={compact ? 'sm' : 'md'} className={compact ? 'mt-2 self-start' : ''} onClick={() => setEditing({ item: detail, isNew: false })} aria-label={`Edit ${entry.title}`}>Edit</Button>;
    return <>
      {!compact && <div className="flex items-end justify-between gap-4">
        <div className="flex min-w-0 flex-col gap-1.5"><Label>{personalDate(entry.date)}</Label><h2 className="m-0 break-words text-[32px] font-light leading-tight">{entry.title}</h2></div>
        {editButton}
      </div>}
      {entry.description && <p className={`m-0 font-light text-ink-60 ${compact ? 'text-sm' : 'mt-3 text-[15px]'}`}>{entry.description}</p>}
      {entry.tags.length > 0 && <Label as="p" className="m-0 mt-2">{entry.tags.join(' · ')}</Label>}
      <div className={`min-w-0 ${compact ? 'mt-1' : 'mt-6'}`}><MarkdownPreview ink body={entry.body} /></div>
      {compact && editButton}
    </>;
  };

  const list = <nav aria-label="Personal entries" className="flex min-w-0 flex-col">
    <h2 className="ui-section-header__title mb-2 max-md:hidden">Entries · {entries.length}</h2>
    {missing && <p role="alert" className="m-0 mb-2 text-sm text-ink-60">That entry was not found. Choose another entry.</p>}
    {filtered.length === 0 && <p className="m-0 py-3 text-sm font-light text-ink-60">{entries.length ? 'No entries match your search.' : 'No personal entries yet. Use + New entry to write one.'}</p>}
    {filtered.map(item => {
      const entry = item.entry, current = selected?.entry.id === entry.id;
      return <div key={entry.id} className="border-b border-ink-14">
        <button type="button" onClick={() => pick(entry.id)}
          aria-current={desktop && current ? 'true' : undefined} aria-expanded={desktop ? undefined : current}
          className={`grid w-full cursor-pointer items-center gap-3 border-0 bg-transparent text-left text-ink hover:bg-ink-05 ${desktop ? `min-h-11 grid-cols-[64px_minmax(0,1fr)] py-2 pl-3 ${current ? 'bg-ink-05 shadow-[inset_2px_0_0_var(--ink)]' : ''}` : 'min-h-[52px] grid-cols-[64px_minmax(0,1fr)_14px] py-2'}`}>
          <span className="font-mono text-xs uppercase">{personalDate(entry.date)}</span>
          <span className="min-w-0 break-words text-sm font-light">{entry.title || 'Untitled'}</span>
          {!desktop && <span aria-hidden="true" className="opacity-50">{current ? '−' : '+'}</span>}
        </button>
        {!desktop && current && <div className="flex min-w-0 flex-col gap-2 pb-4 pl-[76px]">{reader(item, true)}</div>}
      </div>;
    })}
    <MarkdownLink journal="personal" />
  </nav>;

  if (!desktop) return list;
  return <div className="grid gap-8 md:grid-cols-[220px_minmax(0,1fr)] xl:grid-cols-[280px_minmax(0,1fr)] xl:gap-14">
    {list}
    <article className="flex min-h-[420px] min-w-0 flex-col border-l border-ink-14 pl-6 xl:pl-10">
      {editor || (selected ? reader(selected, false) : <p className="m-0 text-[15px] font-light text-ink-60">Choose an entry to read it.</p>)}
    </article>
  </div>;
}

// ---------------------------------------------------------------------------
// Logs

function LogsPanel({ weeks, query, onQuery, desktop }: { weeks: Item<JournalWeek>[]; query: string; onQuery: (query: string) => void; desktop: boolean }) {
  const entries = useMemo(() => weeks.map(item => item.entry), [weeks]);
  const docs = useMemo(() => buildLogDocs(entries), [entries]);
  const summaries: WeekSummary[] = useMemo(() => entries.map(({ week, year, dates, hours, tags }) => ({ week, year, dates, hours, tags })), [entries]);
  return <LogExplorer docs={docs} weeks={summaries} query={query} onQuery={onQuery} desktop={desktop} />;
}
