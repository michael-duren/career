import { useEffect, useMemo, useRef, useState } from 'react';
import { buildBoard, companyLogoUrl, matchesCompanySearch, sortCompanies, PRIORITIES, type Company, type Priority, type RawCompany } from '../lib/companies';
import { cn } from '../lib/utils';
import { Box, DeleteIcon, EditIcon, IconButton, Label, LogoSpot, SearchField, SegmentedTabs, TickScale, buttonClassName, type SegmentedTab } from './ui';

// The Companies page (docs/redesign/pages/companies.md): target companies as cards on desktop
// and rows on phones, with a priority filter, search, and edit/delete on every company. Steps,
// notes and connections live on each company's own page (/companies/<slug>).

type Filter = 'all' | Priority;
type EntryPage = { entries: { entry: RawCompany; revision: string }[]; nextOffset: number | null };

const PRIORITY_LABEL: Record<Priority, string> = { high: 'High', medium: 'Medium', low: 'Low' };

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

function companyPath(slug: string) {
  return `/companies/${slug.split('/').map(encodeURIComponent).join('/')}`;
}

export function CompanyGrid() {
  const [companies, setCompanies] = useState<RawCompany[]>();
  const [revisions, setRevisions] = useState<Record<string, string>>({});
  const [loadError, setLoadError] = useState('');
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');
  const [busy, setBusy] = useState(false);
  const [query, setQuery] = useState('');
  const [filter, setFilter] = useState<Filter>('all');
  const lock = useRef(false);
  const phone = useMedia('(max-width: 767px)');
  // At tablet widths the search field shares its row with the filter and both buttons.
  const tablet = useMedia('(min-width: 768px) and (max-width: 1023px)');

  useEffect(() => {
    const controller = new AbortController();
    let latest = 0;
    const load = () => {
      const request = ++latest;
      void (async () => {
        const entries: EntryPage['entries'] = [];
        for (let offset: number | null = 0; offset !== null;) {
          const response = await fetch(`/api/entries/company?limit=100&view=detail&offset=${offset}`, { cache: 'no-store', signal: controller.signal });
          if (response.status === 401) { location.assign(`/login?redirect=${encodeURIComponent(location.pathname + location.search)}`); return; }
          const value = await response.json();
          if (!response.ok) throw new Error(value.error || 'Could not load companies.');
          const page = value as EntryPage;
          entries.push(...page.entries);
          offset = page.nextOffset;
        }
        if (request !== latest) return;
        setCompanies(entries.map(item => item.entry));
        setRevisions(Object.fromEntries(entries.map(item => [item.entry.slug, item.revision])));
        setLoadError('');
      })().catch(value => {
        if (request === latest && value.name !== 'AbortError') setLoadError(value instanceof Error ? value.message : 'Could not load companies.');
      });
    };
    load();
    // Something else saved (the header's quick capture, another view): reload. A delete here
    // updates the list itself and fires the event while it holds the lock, so skip that one.
    const onSaved = () => { if (!lock.current) load(); };
    window.addEventListener('workspace-saved', onSaved);
    return () => { controller.abort(); window.removeEventListener('workspace-saved', onSaved); };
  }, []);

  const all = useMemo(() => sortCompanies(buildBoard(companies ?? []).categories.flatMap(group => group.companies)), [companies]);
  if (!companies) return <p role={loadError ? 'alert' : 'status'} className={loadError ? 'text-accent' : 'text-ink-60'}>{loadError || 'Loading…'}</p>;

  const featured = all.filter(company => company.featured).length;
  const shown = all.filter(company => (filter === 'all' || company.priority === filter) && matchesCompanySearch(company, query));
  const tabs: SegmentedTab<Filter>[] = [
    { value: 'all', label: phone ? 'All' : `All ${all.length}` },
    ...PRIORITIES.map(value => ({ value, label: phone && value === 'medium' ? 'Med' : PRIORITY_LABEL[value] })),
  ];

  async function remove(company: Company) {
    if (lock.current || !window.confirm(`Delete ${company.title}? This cannot be undone.`)) return;
    lock.current = true; setBusy(true); setError(''); setStatus('Deleting…');
    try {
      const revision = revisions[company.slug];
      if (!revision) throw new Error('This company changed elsewhere. Reload the page and try again.');
      const response = await fetch('/api/entries/company', { method: 'DELETE', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: company.slug, revision }) });
      const result = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(result.error || 'Could not delete. Please retry.');
      setCompanies(current => current?.filter(item => item.slug !== company.slug));
      setStatus(`${company.title} deleted.`);
      // Other views (header counts, the editor) refresh on this; the list above is already current.
      window.dispatchEvent(new Event('workspace-saved'));
    } catch (e) { setError(e instanceof Error ? e.message : 'Could not delete. Please retry.'); setStatus('Company not deleted.'); }
    finally { lock.current = false; setBusy(false); }
  }

  return (
    <div className="flex flex-col">
      <header className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div className="flex items-end justify-between gap-4">
          <div className="flex flex-col gap-1.5 md:gap-2">
            <Label>{all.length} tracked<span className="max-md:hidden"> · {featured} featured</span></Label>
            <h1 className="m-0 font-extralight !text-[40px] !leading-[0.9] md:!text-[52px]">Companies</h1>
          </div>
          <a href="/manage/companies?new=1" aria-label="Add company" title="Add company" className={cn(buttonClassName({ variant: 'primary', icon: true }), 'text-lg md:hidden')}>+</a>
        </div>
        <div className="flex flex-col gap-2.5 md:flex-row md:flex-wrap md:items-center">
          <SegmentedTabs tabs={tabs} value={filter} onChange={setFilter} label="Priority" controls="company-list"
            className="order-2 md:order-none max-md:grid max-md:w-full max-md:grid-cols-4 max-md:[&>button]:h-10 max-md:[&>button]:justify-center max-md:[&>button]:px-1 max-md:[&>button]:text-[10px]" />
          <SearchField label="Search companies" placeholder={tablet ? 'Search' : undefined} value={query} onChange={event => setQuery(event.target.value)} className="order-1 w-full md:order-none md:w-auto md:min-w-[160px] md:flex-1 xl:w-[220px] xl:flex-none" />
          <a href="/manage/companies" className={cn(buttonClassName(), 'max-md:hidden')}>Manage</a>
          <a href="/manage/companies?new=1" className={cn(buttonClassName({ variant: 'primary' }), 'max-md:hidden')}>+ Add company</a>
        </div>
      </header>

      <div className="mt-5 flex flex-wrap items-center gap-x-6 gap-y-2 max-md:hidden">
        <Label>Legend</Label>
        {PRIORITIES.map(value => <span key={value} className="flex items-center gap-2 text-xs font-light"><PriorityDot priority={value} />{PRIORITY_LABEL[value]}</span>)}
        <span className="flex items-center gap-2 text-xs font-light"><span aria-hidden="true" className="text-[10px]">◆</span>Featured</span>
        <span className="flex items-center gap-2 text-xs font-light"><TickScale count={3} filled={1} className="!h-2.5 gap-[2px] [&>span]:!h-2.5" />Checklist steps</span>
        <Label className="ml-auto">{shown.length} shown</Label>
      </div>

      {error && <p role="alert" className="mt-4 border border-accent p-3 text-sm text-accent">{error}</p>}
      {loadError && <p role="alert" className="mt-4 border border-accent p-3 text-sm text-accent">Couldn't refresh the list: {loadError} Reload to retry.</p>}
      <p role="status" className="mt-3 text-sm text-ink-60 empty:hidden">{status}</p>

      {all.length === 0 && <p className="mt-6 text-ink-60">No companies yet. <a href="/manage/companies?new=1" className="underline">Add your first company</a>.</p>}
      {all.length > 0 && shown.length === 0 && <p className="mt-6 text-ink-60">No companies match {query.trim() ? `"${query.trim()}"` : 'this filter'}.</p>}
      <ul id="company-list" aria-label="Companies" className={cn('m-0 list-none p-0', phone ? 'mt-2.5 flex flex-col' : 'mt-5 grid grid-cols-2 gap-x-6 gap-y-4 p-[3px] lg:grid-cols-3 xl:grid-cols-4')}>
        {shown.map(company => phone
          ? <CompanyRow key={company.slug} company={company} busy={busy} onDelete={() => void remove(company)} />
          : <CompanyCard key={company.slug} company={company} busy={busy} onDelete={() => void remove(company)} />)}
      </ul>
    </div>
  );
}

type ItemProps = { company: Company; busy: boolean; onDelete: () => void };

function CompanyCard({ company, busy, onDelete }: ItemProps) {
  return (
    <Box as="li" className="flex min-h-[158px] min-w-0 flex-col gap-2 px-3.5 py-3 hover:bg-ink-05">
      <div className="flex min-w-0 items-center gap-3">
        <LogoSpot name={company.title} src={companyLogoUrl(company.url)} lazy />
        <span className="flex min-w-0 flex-1 flex-col">
          <a href={companyPath(company.slug)} className="truncate text-[19px] font-light leading-[1.1] text-ink no-underline hover:underline">{company.title}</a>
          <Label className="truncate !text-[9px]" title={company.category}>{company.category}</Label>
        </span>
        <Marks company={company} className="mt-1 self-start" />
      </div>
      {company.why && <p className="m-0 line-clamp-2 text-xs font-light leading-[1.45] text-ink-60">{company.why}</p>}
      <div className="mt-auto flex items-center gap-2.5">
        <Steps company={company} />
        <ItemActions company={company} busy={busy} onDelete={onDelete} className="ml-auto" />
      </div>
    </Box>
  );
}

function CompanyRow({ company, busy, onDelete }: ItemProps) {
  return (
    <li className="grid grid-cols-[40px_minmax(0,1fr)_auto] items-center gap-3 border-b border-ink-14 py-3.5">
      <LogoSpot name={company.title} src={companyLogoUrl(company.url)} size={40} lazy />
      <a href={companyPath(company.slug)} className="flex min-w-0 flex-col gap-1 text-ink no-underline">
        <span className="flex min-w-0 items-center gap-2">
          <span className="truncate text-[17px] font-light">{company.title}</span>
          <Marks company={company} />
        </span>
        <span className="flex min-w-0 items-center gap-2">
          <Label className="truncate !text-[8.5px]">{company.category}</Label>
          <Steps company={company} compact />
        </span>
      </a>
      <ItemActions company={company} busy={busy} onDelete={onDelete} />
    </li>
  );
}

// Featured diamond and priority dot, with their meaning in text for screen readers.
function Marks({ company, className }: { company: Company; className?: string }) {
  return (
    <span className={cn('flex flex-none items-center gap-2', className)}>
      {company.featured && <span title="Featured" className="text-[10px] leading-none"><span aria-hidden="true">◆</span><span className="sr-only">Featured</span></span>}
      <PriorityDot priority={company.priority} label />
    </span>
  );
}

// Filled (high), half-filled (medium) or empty (low) circle in ink. Also on the company page (CompanyBoard).
export function PriorityDot({ priority, label = false }: { priority: Priority; label?: boolean }) {
  return (
    <span title={label ? `${PRIORITY_LABEL[priority]} priority` : undefined} className={cn('inline-block size-[9px] flex-none rounded-full border border-solid border-ink',
      priority === 'high' && 'bg-ink', priority === 'medium' && 'bg-[linear-gradient(90deg,var(--ink)_50%,transparent_50%)]')}>
      {label && <span className="sr-only">{PRIORITY_LABEL[priority]} priority</span>}
    </span>
  );
}

function Steps({ company, compact = false }: { company: Company; compact?: boolean }) {
  if (company.steps.length === 0) return null;
  const label = `${company.completed} of ${company.steps.length} steps done`;
  return (
    <span className="flex flex-none items-center gap-2.5">
      <TickScale count={company.steps.length} filled={company.completed} label={label}
        className={cn('justify-start', compact ? '!h-[9px] gap-[2px] [&>span]:!h-[9px]' : '!h-3 gap-[3px] [&>span]:!h-3')} />
      {!compact && <span aria-hidden="true" className="font-mono text-[10px] text-ink-60">{company.completed}/{company.steps.length}</span>}
    </span>
  );
}

function ItemActions({ company, busy, onDelete, className }: ItemProps & { className?: string }) {
  return (
    <span className={cn('flex flex-none gap-1.5 md:gap-2', className)}>
      <a href={`/manage/companies?id=${encodeURIComponent(company.slug)}`} aria-label={`Edit ${company.title}`} title={`Edit ${company.title}`}
        className={cn(buttonClassName({ size: 'sm', icon: true }), 'border-ink-30')}><EditIcon size={13} /></a>
      <IconButton label={`Delete ${company.title}`} variant="danger" size="sm" disabled={busy} onClick={onDelete}><DeleteIcon size={13} /></IconButton>
    </span>
  );
}
