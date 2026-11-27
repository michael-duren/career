import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { Button, Label, SearchIcon } from './ui';
import { shortDate, trackLabel, type LogDoc, type WeekSummary } from '../lib/logs';

const PAGE_SIZE = 25;

// ---------------------------------------------------------------------------
// The searchable work log: every daily entry (and each week's Blockers / Notes) is one record.
// Free-text search with highlighted matches, filters on the left, expandable records below.
// ---------------------------------------------------------------------------

interface Props {
  docs: LogDoc[];
  weeks: WeekSummary[];
  query: string;
  onQuery: (query: string) => void;
  /** Two columns from 768px; phones fold the filters into a disclosure. */
  desktop: boolean;
}

type FieldKey = 'year' | 'type' | 'week' | 'day' | 'track' | 'tag';

interface ActiveFilter {
  field: FieldKey;
  value: string;
}

const SECTION_LABEL: Record<string, string> = {
  day: 'What I did',
  blockers: 'Blockers',
  notes: 'Notes',
};

// --- text helpers -----------------------------------------------------------

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function queryTokens(query: string): string[] {
  return query.trim().split(/\s+/).filter(Boolean);
}

/** `text` with every query-word match underlined, as the reference marks matches. */
function Hl({ text, query }: { text: string; query: string }) {
  const tokens = queryTokens(query).map(escapeRegExp);
  if (!tokens.length) return <>{text}</>;
  const re = new RegExp(`(${tokens.join('|')})`, 'gi');
  // split() with a capture group puts the matches at odd indexes.
  return <>{text.split(re).map((part, i) => i % 2
    ? <mark key={i} className="bg-transparent font-normal text-ink underline decoration-1 underline-offset-[3px]">{part}</mark>
    : part)}</>;
}

// --- filtering --------------------------------------------------------------

function docMatchesQuery(doc: LogDoc, query: string): boolean {
  const tokens = queryTokens(query.toLowerCase());
  if (!tokens.length) return true;
  const hay = (
    doc.message +
    ' ' +
    doc.dateLabel +
    ' ' +
    doc.tags.join(' ') +
    ' w' +
    doc.week +
    ' ' +
    doc.section +
    ' ' +
    doc.activeTracks.join(' ')
  ).toLowerCase();
  return tokens.every((t) => hay.includes(t));
}

function docFieldValues(doc: LogDoc, field: FieldKey): string[] {
  switch (field) {
    case 'year':
      return [String(doc.year)];
    case 'week':
      return [`W${doc.week}`];
    case 'day':
      return doc.day ? [doc.day] : [];
    case 'tag':
      return doc.tags;
    case 'type':
      return [doc.section];
    case 'track':
      return doc.activeTracks;
  }
}

// Active filters group by field: OR within a field, AND across fields.
function docMatchesFilters(
  doc: LogDoc,
  groups: Map<FieldKey, Set<string>>,
  exceptField?: FieldKey,
): boolean {
  for (const [field, values] of groups) {
    if (field === exceptField) continue;
    if (values.size === 0) continue;
    const dv = docFieldValues(doc, field);
    if (!dv.some((v) => values.has(v))) return false;
  }
  return true;
}

const FACETS: { field: FieldKey; label: string }[] = [
  { field: 'year', label: 'Year' },
  { field: 'type', label: 'Section' },
  { field: 'week', label: 'Week' },
  { field: 'day', label: 'Day' },
  { field: 'track', label: 'Track' },
  { field: 'tag', label: 'Tags' },
];
const FACET_LABEL = Object.fromEntries(FACETS.map(({ field, label }) => [field, label])) as Record<FieldKey, string>;

export function LogExplorer({ docs, weeks, query, onQuery, desktop }: Props) {
  const [filters, setFilters] = useState<ActiveFilter[]>([]);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [sortNewest, setSortNewest] = useState(true);
  const [page, setPage] = useState(1);

  // Map "W10" → "Jun 22" so week facets/filters read as dates.
  const weekDate = useMemo(() => {
    const m = new Map<string, string>();
    for (const w of weeks) m.set(`W${w.week}`, shortDate(w.dates));
    return m;
  }, [weeks]);

  const filterGroups = useMemo(() => {
    const m = new Map<FieldKey, Set<string>>();
    for (const f of filters) {
      if (!m.has(f.field)) m.set(f.field, new Set());
      m.get(f.field)!.add(f.value);
    }
    return m;
  }, [filters]);

  const isFilterActive = (field: FieldKey, value: string) =>
    filters.some((f) => f.field === field && f.value === value);

  const toggleFilter = (field: FieldKey, value: string) => {
    setFilters((prev) => {
      const exists = prev.some((f) => f.field === field && f.value === value);
      return exists
        ? prev.filter((f) => !(f.field === field && f.value === value))
        : [...prev, { field, value }];
    });
  };

  const valueLabel = (field: FieldKey, value: string) =>
    field === 'week' ? `${value} · ${weekDate.get(value) ?? ''}`.replace(/ · $/, '')
      : field === 'type' ? (SECTION_LABEL[value] ?? value)
        : field === 'track' ? trackLabel(value)
          : value;

  // Documents matching the full query + all filters.
  const results = useMemo(() => {
    const out = docs.filter(
      (d) => docMatchesQuery(d, query) && docMatchesFilters(d, filterGroups),
    );
    out.sort((a, b) =>
      sortNewest ? b.year - a.year || b.sortKey - a.sortKey : a.year - b.year || a.sortKey - b.sortKey,
    );
    return out;
  }, [docs, query, filterGroups, sortNewest]);

  // Reset to the first page whenever the result set changes.
  useEffect(() => {
    setPage(1);
  }, [query, filterGroups, sortNewest]);

  const totalPages = Math.max(1, Math.ceil(results.length / PAGE_SIZE));
  const safePage = Math.min(page, totalPages);
  const pageStart = (safePage - 1) * PAGE_SIZE;
  const pageDocs = results.slice(pageStart, pageStart + PAGE_SIZE);

  // Facet value counts: a field's counts reflect the query + every OTHER field's
  // filters (so selecting a value in one facet doesn't zero out its siblings).
  const facetData = useMemo(() => {
    const data = Object.fromEntries(FACETS.map(({ field }) => [field, []])) as unknown as Record<FieldKey, { value: string; count: number }[]>;
    for (const { field } of FACETS) {
      const counts = new Map<string, number>();
      for (const d of docs) {
        if (!docMatchesQuery(d, query)) continue;
        if (!docMatchesFilters(d, filterGroups, field)) continue;
        for (const v of docFieldValues(d, field)) {
          counts.set(v, (counts.get(v) || 0) + 1);
        }
      }
      data[field] = Array.from(counts.entries())
        .map(([value, count]) => ({ value, count }))
        .sort((a, b) => field === 'year' ? b.value.localeCompare(a.value) : b.count - a.count || a.value.localeCompare(b.value));
    }
    return data;
  }, [docs, query, filterGroups]);

  const toggleExpand = (id: string) =>
    setExpanded((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });

  const facets = FACETS.map(({ field, label }) => {
    const values = facetData[field];
    if (values.length === 0) return null;
    return (
      <Facet
        key={field}
        label={label}
        values={values}
        isActive={(v) => isFilterActive(field, v)}
        onToggle={(v) => toggleFilter(field, v)}
        renderValue={(v) => valueLabel(field, v)}
      />
    );
  });

  return (
    <div className="grid gap-6 md:grid-cols-[200px_minmax(0,1fr)] md:gap-8 xl:grid-cols-[220px_minmax(0,1fr)] xl:gap-14">
      {desktop && (
        <aside aria-label="Filters" className="flex min-w-0 flex-col gap-5">
          <h2 className="ui-section-header__title">Filters</h2>
          {facets}
        </aside>
      )}

      <section aria-label="Log entries" className="flex min-w-0 flex-col">
        <label className="flex h-[46px] items-center gap-2.5 border-b border-ink focus-within:outline focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-ink">
          <SearchIcon size={14} />
          <span className="ui-sr-only">Search logs</span>
          <input
            type="search"
            placeholder="Search logs (every word must match)"
            value={query}
            onChange={(e) => onQuery(e.target.value)}
            className="min-w-0 flex-1 border-0 bg-transparent text-[18px] font-light text-ink outline-none placeholder:text-ink-60 focus-visible:!outline-none"
          />
        </label>

        {!desktop && (
          <details className="mt-2 border-b border-ink-14 [&[open]>summary>span:last-child]:rotate-90">
            <summary className="flex min-h-11 cursor-pointer list-none items-center justify-between [&::-webkit-details-marker]:hidden">
              <span className="ui-section-header__title">Filters{filters.length ? ` · ${filters.length}` : ''}</span>
              <span aria-hidden="true" className="opacity-50 transition-transform">›</span>
            </summary>
            <div className="flex flex-col gap-5 pb-4 pt-1">{facets}</div>
          </details>
        )}

        <div className="mb-1.5 mt-3.5 flex flex-wrap items-center justify-between gap-2">
          <Label role="status">
            {results.length} match{results.length === 1 ? '' : 'es'}
            {filters.length > 0 || query ? ' (filtered)' : ''}
          </Label>
          <Button size="sm" onClick={() => setSortNewest((s) => !s)}>
            {sortNewest ? '↓ Newest first' : '↑ Oldest first'}
          </Button>
        </div>

        {/* Active filter chips */}
        {filters.length > 0 && (
          <div className="mb-2 flex flex-wrap items-center gap-2">
            {filters.map((f) => (
              <Button
                key={`${f.field}:${f.value}`}
                size="sm"
                onClick={() => toggleFilter(f.field, f.value)}
                aria-label={`Remove filter ${FACET_LABEL[f.field]}: ${valueLabel(f.field, f.value)}`}
                className="normal-case tracking-normal"
              >
                <span className="text-ink-60">{FACET_LABEL[f.field]}:</span>&nbsp;{valueLabel(f.field, f.value)}&nbsp;<span aria-hidden="true">×</span>
              </Button>
            ))}
            <button type="button" onClick={() => setFilters([])} className="ui-label cursor-pointer border-0 bg-transparent px-1 py-2 underline hover:text-ink">
              Clear all
            </button>
          </div>
        )}

        {/* Record list */}
        {results.length === 0 ? (
          <p className="m-0 border-b border-ink-14 py-8 text-sm font-light text-ink-60">No entries match your search.</p>
        ) : (
          <>
            <ul className="m-0 list-none p-0">
              {pageDocs.map((doc) => (
                <DocRow
                  key={doc.id}
                  doc={doc}
                  query={query}
                  open={expanded.has(doc.id)}
                  onToggle={() => toggleExpand(doc.id)}
                  onFilter={toggleFilter}
                />
              ))}
            </ul>

            {/* Pagination */}
            <div className="flex flex-wrap items-center justify-between gap-3 pt-3">
              <Label>
                {pageStart + 1}–{pageStart + pageDocs.length} of {results.length}
              </Label>
              {totalPages > 1 && (
                <div className="flex items-center gap-2">
                  <Button size="sm" disabled={safePage <= 1} onClick={() => setPage(safePage - 1)}>‹ Prev</Button>
                  <span className="px-2 font-mono text-xs">
                    {safePage} / {totalPages}
                  </span>
                  <Button size="sm" disabled={safePage >= totalPages} onClick={() => setPage(safePage + 1)}>Next ›</Button>
                </div>
              )}
            </div>
          </>
        )}
      </section>
    </div>
  );
}

// --- facet ------------------------------------------------------------------

function Facet({
  label,
  values,
  isActive,
  onToggle,
  renderValue,
}: {
  label: string;
  values: { value: string; count: number }[];
  isActive: (v: string) => boolean;
  onToggle: (v: string) => void;
  renderValue: (v: string) => string;
}) {
  const [open, setOpen] = useState(true);
  const [showAll, setShowAll] = useState(false);
  const shown = showAll ? values : values.slice(0, 8);
  return (
    <div className="flex flex-col">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
        className="ui-label flex min-h-8 cursor-pointer items-center justify-between border-0 bg-transparent p-0 text-left hover:text-ink"
      >
        <span>{label}</span>
        <span aria-hidden="true" className={`transition-transform ${open ? 'rotate-90' : ''}`}>›</span>
      </button>
      {open && (
        <div className="flex flex-col">
          {shown.map(({ value, count }) => {
            const active = isActive(value);
            return (
              <button
                type="button"
                key={value}
                onClick={() => onToggle(value)}
                aria-pressed={active}
                className={`flex min-h-9 w-full cursor-pointer items-center gap-2 border-0 bg-transparent py-1 pl-2 text-left text-sm font-light text-ink hover:bg-ink-05 ${active ? 'bg-ink-05 shadow-[inset_2px_0_0_var(--ink)]' : ''}`}
              >
                <span className="min-w-0 flex-1 truncate">{renderValue(value)}</span>
                <span className="pr-1 font-mono text-[10px] text-ink-60">{count}</span>
              </button>
            );
          })}
          {values.length > 8 && (
            <button
              type="button"
              onClick={() => setShowAll((s) => !s)}
              className="ui-label cursor-pointer border-0 bg-transparent py-2 pl-2 text-left hover:text-ink"
            >
              {showAll ? 'Show less' : `+ ${values.length - 8} more`}
            </button>
          )}
        </div>
      )}
    </div>
  );
}

// --- record row -------------------------------------------------------------

function DocRow({
  doc,
  query,
  open,
  onToggle,
  onFilter,
}: {
  doc: LogDoc;
  query: string;
  open: boolean;
  onToggle: () => void;
  onFilter: (field: FieldKey, value: string) => void;
}) {
  const preview = doc.message.replace(/\n+/g, ' · ');
  return (
    <li className="border-b border-ink-14">
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={open}
        className="grid w-full cursor-pointer grid-cols-[96px_minmax(0,1fr)] gap-4 border-0 bg-transparent py-3.5 text-left text-ink hover:bg-ink-05 sm:grid-cols-[120px_minmax(0,1fr)] sm:gap-5"
      >
        <span className="flex flex-col gap-1">
          <span className="font-mono text-xs uppercase">W{doc.week} · {shortDate(doc.weekDates)}</span>
          <Label className="!text-[9px]"><Hl text={doc.day || SECTION_LABEL[doc.section] || doc.section} query={query} /></Label>
        </span>
        <span className={`min-w-0 break-words text-[15px] font-light ${open ? '' : 'line-clamp-2'}`}>
          <Hl text={preview} query={query} />
        </span>
      </button>

      {open && (
        <div className="flex flex-col gap-3 pb-4 sm:pl-[140px]">
          {/* Parsed parts (the journal text) */}
          <div className="flex flex-col gap-2">
            {doc.parts.map((p, i) => (
              <div key={i} className="flex gap-2">
                {p.label && <Label className="mt-1 shrink-0">{p.label}</Label>}
                <p className="m-0 text-[15px] font-light leading-relaxed">
                  <Hl text={p.text} query={query} />
                </p>
              </div>
            ))}
          </div>

          {/* Field table */}
          <dl className="m-0 border-t border-ink-14 font-mono text-xs">
            <FieldRow name="week" value={`W${doc.week} · ${shortDate(doc.weekDates)}`} onClick={() => onFilter('week', `W${doc.week}`)} />
            <FieldRow name="dates" value={doc.weekDates} />
            {doc.day && <FieldRow name="day" value={doc.day} onClick={() => onFilter('day', doc.day)} />}
            <FieldRow name="section" value={SECTION_LABEL[doc.section] ?? doc.section} onClick={() => onFilter('type', doc.section)} />
            {doc.tags.length > 0 && <FieldRow name="tags" value={doc.tags.join(', ')} />}
            {Object.entries(doc.hours)
              .filter(([, v]) => (v || 0) > 0)
              .map(([k, v]) => (
                <FieldRow key={k} name={`hours.${k}`} value={`${v}h`} onClick={() => onFilter('track', k)} />
              ))}
          </dl>
          <a href={doc.slug ? `/journal?id=${encodeURIComponent(doc.slug)}` : '/journal'} className="ui-label self-start py-1 hover:text-ink">
            Open in journal →
          </a>
        </div>
      )}
    </li>
  );
}

function FieldRow({ name, value, onClick }: { name: string; value: string; onClick?: () => void }): ReactNode {
  return (
    <div className="flex items-start gap-3 border-b border-ink-14 py-1.5">
      <dt className="w-28 shrink-0 text-ink-60">{name}</dt>
      <dd className="m-0 min-w-0 break-words">
        {onClick ? (
          <button type="button" onClick={onClick} className="cursor-pointer border-0 bg-transparent p-0 text-left text-ink underline-offset-2 hover:underline" title="Filter on this value">
            {value}
          </button>
        ) : (
          value
        )}
      </dd>
    </div>
  );
}
