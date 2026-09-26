'use client';

import { useEffect, useMemo, useState } from 'react';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  BarController,
  Tooltip,
  Legend,
  type ChartOptions,
} from 'chart.js';
import { Bar } from 'react-chartjs-2';
import { cn } from '../lib/utils';
import { tagTone } from '../lib/tag-colors';
import {
  allHourKeys,
  shortDate,
  trackColor,
  trackLabel,
  type LogDoc,
  type WeekSummary,
} from '../lib/logs';

const PAGE_SIZE = 25;

ChartJS.register(CategoryScale, LinearScale, BarElement, BarController, Tooltip, Legend);

// ---------------------------------------------------------------------------
// A Kibana Discover–style explorer for the weekly journal. Free-text search
// (with match highlighting), field facets on the left, a time histogram up top,
// and expandable documents below.
// ---------------------------------------------------------------------------

interface Props {
  docs: LogDoc[];
  weeks: WeekSummary[];
}

type FieldKey = 'week' | 'day' | 'tag' | 'type' | 'track';

interface ActiveFilter {
  field: FieldKey;
  value: string;
}

const SECTION_LABEL: Record<string, string> = {
  day: 'daily',
  blockers: 'blocker',
  notes: 'note',
};

// --- text helpers -----------------------------------------------------------

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}

function escapeRegExp(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function queryTokens(query: string): string[] {
  return query.trim().split(/\s+/).filter(Boolean);
}

/** Escape `text`, then wrap any query-token matches in <mark>. */
function highlight(text: string, query: string): string {
  const esc = escapeHtml(text);
  const tokens = queryTokens(query).map(escapeRegExp);
  if (!tokens.length) return esc;
  const re = new RegExp(`(${tokens.join('|')})`, 'gi');
  return esc.replace(
    re,
    '<mark class="bg-yellow-400/30 text-yellow-100 rounded-sm px-0.5">$1</mark>',
  );
}

function Hl({ text, query }: { text: string; query: string }) {
  return <span dangerouslySetInnerHTML={{ __html: highlight(text, query) }} />;
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

// Active filters group by field: OR within a field, AND across fields (Kibana).
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
  { field: 'week', label: 'week' },
  { field: 'type', label: 'type' },
  { field: 'day', label: 'day' },
  { field: 'track', label: 'track' },
  { field: 'tag', label: 'tags' },
];

const chartOptions: ChartOptions<'bar'> = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { position: 'bottom', labels: { color: '#a1a1aa', boxWidth: 10, font: { size: 10 } } },
    tooltip: {
      backgroundColor: '#18181b',
      titleColor: '#fafafa',
      bodyColor: '#a1a1aa',
      borderColor: '#27272a',
      borderWidth: 1,
    },
  },
  scales: {
    x: { grid: { color: '#27272a' }, ticks: { color: '#71717a', font: { size: 10 } }, stacked: true },
    y: { grid: { color: '#27272a' }, ticks: { color: '#71717a', font: { size: 10 } }, stacked: true, beginAtZero: true },
  },
} as any;

export function LogExplorer({ docs, weeks }: Props) {
  const [query, setQuery] = useState('');
  const [filters, setFilters] = useState<ActiveFilter[]>([]);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [histMode, setHistMode] = useState<'hits' | 'hours'>('hits');
  const [sortNewest, setSortNewest] = useState(true);
  const [page, setPage] = useState(1);

  const hourKeys = useMemo(() => allHourKeys(weeks), [weeks]);

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

  // Documents matching the full query + all filters.
  const results = useMemo(() => {
    const out = docs.filter(
      (d) => docMatchesQuery(d, query) && docMatchesFilters(d, filterGroups),
    );
    out.sort((a, b) =>
      sortNewest ? b.sortKey - a.sortKey : a.sortKey - b.sortKey,
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
    const data: Record<FieldKey, { value: string; count: number }[]> = {
      week: [],
      day: [],
      tag: [],
      type: [],
      track: [],
    };
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
        .sort((a, b) => b.count - a.count || a.value.localeCompare(b.value));
    }
    return data;
  }, [docs, query, filterGroups]);

  // Histogram over weeks (chronological), reactive to query + filters.
  const sortedWeeks = useMemo(
    () => [...weeks].sort((a, b) => a.week - b.week),
    [weeks],
  );

  const hitsPerWeek = useMemo(() => {
    const m = new Map<number, number>();
    for (const d of results) m.set(d.week, (m.get(d.week) || 0) + 1);
    return m;
  }, [results]);

  const chartData = useMemo(() => {
    const labels = sortedWeeks.map((w) => shortDate(w.dates));
    if (histMode === 'hits') {
      return {
        labels,
        datasets: [
          {
            label: 'matching entries',
            data: sortedWeeks.map((w) => hitsPerWeek.get(w.week) || 0),
            backgroundColor: '#60a5fa',
            borderRadius: 2,
          },
        ],
      } as any;
    }
    return {
      labels,
      datasets: hourKeys.map((k) => ({
        label: trackLabel(k),
        data: sortedWeeks.map((w) => w.hours[k] || 0),
        backgroundColor: trackColor(k),
        stack: 'hours',
      })),
    } as any;
  }, [sortedWeeks, hitsPerWeek, histMode, hourKeys]);

  const toggleExpand = (id: string) =>
    setExpanded((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });

  return (
    <div className="space-y-4">
      {/* Search bar */}
      <div className="flex flex-col sm:flex-row gap-2">
        <div className="relative flex-1">
          <svg
            className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            type="text"
            placeholder="Search the journal…  (space-separated terms, all must match)"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="w-full pl-9 pr-3 py-2 bg-zinc-900 border border-zinc-700 rounded font-mono text-sm focus:outline-none focus:border-blue-500"
          />
        </div>
        <button
          onClick={() => setSortNewest((s) => !s)}
          className="px-3 py-2 bg-zinc-900 border border-zinc-700 rounded text-xs text-zinc-300 hover:border-zinc-500 whitespace-nowrap"
        >
          {sortNewest ? '↓ Newest first' : '↑ Oldest first'}
        </button>
      </div>

      {/* Active filter chips */}
      {filters.length > 0 && (
        <div className="flex flex-wrap items-center gap-2">
          {filters.map((f) => (
            <button
              key={`${f.field}:${f.value}`}
              onClick={() => toggleFilter(f.field, f.value)}
              className="group flex items-center gap-1.5 px-2 py-1 bg-blue-600/15 border border-blue-500/40 rounded text-xs text-blue-300 hover:bg-blue-600/25"
            >
              <span className="opacity-60">{f.field}:</span>
              <span className="font-medium">
                {f.field === 'week' ? (weekDate.get(f.value) ?? f.value) : f.field === 'type' ? (SECTION_LABEL[f.value] ?? f.value) : f.value}
              </span>
              <span className="opacity-50 group-hover:opacity-100">✕</span>
            </button>
          ))}
          <button
            onClick={() => setFilters([])}
            className="text-xs text-zinc-500 hover:text-zinc-300 underline"
          >
            clear all
          </button>
        </div>
      )}

      <div className="flex flex-col lg:flex-row gap-4">
        {/* Field facets */}
        <aside className="lg:w-60 lg:flex-shrink-0 space-y-4">
          <div className="text-[10px] uppercase tracking-wide text-zinc-500 font-semibold">
            Fields
          </div>
          {FACETS.map(({ field, label }) => {
            const values = facetData[field];
            if (values.length === 0) return null;
            return (
              <Facet
                key={field}
                label={label}
                values={values}
                isActive={(v) => isFilterActive(field, v)}
                onToggle={(v) => toggleFilter(field, v)}
                renderValue={
                  field === 'type'
                    ? (v) => SECTION_LABEL[v] ?? v
                    : field === 'week'
                      ? (v) => weekDate.get(v) ?? v
                      : undefined
                }
                color={field === 'track' ? (v) => trackColor(v) : undefined}
              />
            );
          })}
        </aside>

        {/* Main column */}
        <div className="flex-1 min-w-0 space-y-3">
          {/* Histogram */}
          <div className="bg-zinc-900 rounded-lg border border-zinc-800 p-3">
            <div className="flex items-center justify-between mb-2">
              <div className="text-xs text-zinc-400">
                <span className="font-mono text-zinc-200">{results.length}</span> hits
                {filters.length > 0 || query ? ' (filtered)' : ''}
              </div>
              <div className="flex rounded border border-zinc-700 overflow-hidden text-[11px]">
                {(['hits', 'hours'] as const).map((m) => (
                  <button
                    key={m}
                    onClick={() => setHistMode(m)}
                    className={cn(
                      'px-2 py-1',
                      histMode === m
                        ? 'bg-blue-600/30 text-blue-200'
                        : 'text-zinc-400 hover:bg-zinc-800',
                    )}
                  >
                    {m === 'hits' ? 'Entries / wk' : 'Hours / wk'}
                  </button>
                ))}
              </div>
            </div>
            <div className="h-40">
              <Bar
                data={chartData}
                options={{
                  ...chartOptions,
                  onClick: (_evt: unknown, els: { index: number }[]) => {
                    if (!els.length) return;
                    const w = sortedWeeks[els[0].index];
                    if (w) toggleFilter('week', `W${w.week}`);
                  },
                } as any}
              />
            </div>
          </div>

          {/* Document list */}
          {results.length === 0 ? (
            <div className="bg-zinc-900 rounded-lg border border-zinc-800 p-8 text-center text-sm text-zinc-500">
              No entries match your search.
            </div>
          ) : (
            <>
              <div className="space-y-2">
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
              </div>

              {/* Pagination */}
              <div className="flex items-center justify-between gap-3 pt-1 text-xs text-zinc-500">
                <span>
                  {pageStart + 1}–{pageStart + pageDocs.length} of {results.length}
                </span>
                {totalPages > 1 && (
                  <div className="flex items-center gap-1">
                    <PageButton
                      label="‹ Prev"
                      disabled={safePage <= 1}
                      onClick={() => setPage(safePage - 1)}
                    />
                    <span className="px-2 font-mono text-zinc-400">
                      {safePage} / {totalPages}
                    </span>
                    <PageButton
                      label="Next ›"
                      disabled={safePage >= totalPages}
                      onClick={() => setPage(safePage + 1)}
                    />
                  </div>
                )}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function PageButton({
  label,
  disabled,
  onClick,
}: {
  label: string;
  disabled: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className="px-2 py-1 rounded border border-zinc-700 text-zinc-300 hover:border-zinc-500 disabled:opacity-40 disabled:cursor-not-allowed"
    >
      {label}
    </button>
  );
}

// --- facet ------------------------------------------------------------------

function Facet({
  label,
  values,
  isActive,
  onToggle,
  renderValue,
  color,
}: {
  label: string;
  values: { value: string; count: number }[];
  isActive: (v: string) => boolean;
  onToggle: (v: string) => void;
  renderValue?: (v: string) => string;
  color?: (v: string) => string;
}) {
  const [open, setOpen] = useState(true);
  const [showAll, setShowAll] = useState(false);
  const shown = showAll ? values : values.slice(0, 8);
  return (
    <div className="bg-zinc-900 rounded-lg border border-zinc-800">
      <button
        onClick={() => setOpen((o) => !o)}
        className="w-full flex items-center justify-between px-3 py-2 text-xs font-mono text-zinc-300 hover:bg-zinc-800/50"
      >
        <span>{label}</span>
        <span className={cn('transition-transform text-zinc-600', open && 'rotate-90')}>›</span>
      </button>
      {open && (
        <div className="px-2 pb-2 space-y-0.5">
          {shown.map(({ value, count }) => {
            const active = isActive(value);
            return (
              <button
                key={value}
                onClick={() => onToggle(value)}
                className={cn(
                  'w-full flex items-center gap-2 px-2 py-1 rounded text-xs text-left',
                  active ? 'bg-blue-600/20 text-blue-300' : 'text-zinc-400 hover:bg-zinc-800',
                )}
              >
                {color && (
                  <span
                    className="w-2 h-2 rounded-full flex-shrink-0"
                    style={{ backgroundColor: color(value) }}
                  />
                )}
                <span className="flex-1 truncate">{renderValue ? renderValue(value) : value}</span>
                <span className="font-mono text-[10px] text-zinc-600">{count}</span>
              </button>
            );
          })}
          {values.length > 8 && (
            <button
              onClick={() => setShowAll((s) => !s)}
              className="w-full text-left px-2 py-1 text-[10px] text-zinc-500 hover:text-zinc-300"
            >
              {showAll ? 'show less' : `+ ${values.length - 8} more`}
            </button>
          )}
        </div>
      )}
    </div>
  );
}

// --- document row -----------------------------------------------------------

const SECTION_BADGE: Record<string, string> = {
  day: 'bg-zinc-800 text-zinc-400',
  blockers: 'bg-red-500/15 text-red-300',
  notes: 'bg-blue-500/15 text-blue-300',
};

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
    <div className="bg-zinc-900 rounded-lg border border-zinc-800 overflow-hidden">
      <button
        onClick={onToggle}
        className="w-full flex items-start gap-3 p-3 text-left hover:bg-zinc-800/40"
      >
        <span className={cn('mt-0.5 text-zinc-600 transition-transform', open && 'rotate-90')}>›</span>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2 mb-1">
            <span className="font-mono text-xs text-blue-400">{shortDate(doc.weekDates)}</span>
            <span className="font-mono text-xs text-zinc-300">
              <Hl text={doc.dateLabel} query={query} />
            </span>
            <span className={cn('text-[10px] uppercase px-1.5 py-0.5 rounded', SECTION_BADGE[doc.section])}>
              {SECTION_LABEL[doc.section] ?? doc.section}
            </span>
            {doc.tags.map((t) => (
              <span key={t} className={cn('text-[10px] uppercase px-1.5 py-0.5 rounded ring-1 ring-inset', tagTone(t).chip)}>
                {t}
              </span>
            ))}
          </div>
          {!open && (
            <p className="text-xs text-zinc-500 truncate">
              <Hl text={preview} query={query} />
            </p>
          )}
        </div>
      </button>

      {open && (
        <div className="px-3 pb-3 border-t border-zinc-800 pt-3 space-y-3 text-sm">
          {/* Parsed parts (the journal text) */}
          <div className="space-y-2">
            {doc.parts.map((p, i) => (
              <div key={i} className="flex gap-2">
                {p.label && (
                  <span className="font-mono text-[11px] px-1.5 py-0.5 h-fit rounded bg-zinc-800 text-zinc-400 flex-shrink-0">
                    {p.label}
                  </span>
                )}
                <p className="text-zinc-300 leading-relaxed">
                  <Hl text={p.text} query={query} />
                </p>
              </div>
            ))}
          </div>

          {/* Field table — Kibana expanded-doc style */}
          <div className="rounded border border-zinc-800 divide-y divide-zinc-800 text-xs font-mono">
            <FieldRow name="week" value={`W${doc.week} · ${shortDate(doc.weekDates)}`} onClick={() => onFilter('week', `W${doc.week}`)} />
            <FieldRow name="dates" value={doc.weekDates} />
            {doc.day && <FieldRow name="day" value={doc.day} onClick={() => onFilter('day', doc.day)} />}
            <FieldRow name="type" value={doc.section} onClick={() => onFilter('type', doc.section)} />
            {doc.tags.length > 0 && (
              <FieldRow name="tags" value={doc.tags.join(', ')} />
            )}
            {Object.entries(doc.hours)
              .filter(([, v]) => (v || 0) > 0)
              .map(([k, v]) => (
                <FieldRow
                  key={k}
                  name={`hours.${k}`}
                  value={`${v}h`}
                  color={trackColor(k)}
                  onClick={() => onFilter('track', k)}
                />
              ))}
          </div>
          <div className="text-[10px] text-zinc-600 font-mono">
            <a href="/journal" className="text-blue-400 hover:underline">Edit in journal →</a>
          </div>
        </div>
      )}
    </div>
  );
}

function FieldRow({
  name,
  value,
  onClick,
  color,
}: {
  name: string;
  value: string;
  onClick?: () => void;
  color?: string;
}) {
  return (
    <div className="flex items-start gap-3 px-2 py-1.5">
      <span className="w-28 flex-shrink-0 text-zinc-500 flex items-center gap-1.5">
        {color && <span className="w-2 h-2 rounded-full" style={{ backgroundColor: color }} />}
        {name}
      </span>
      {onClick ? (
        <button
          onClick={onClick}
          className="text-left text-zinc-300 hover:text-blue-300 hover:underline"
          title="filter on this value"
        >
          {value}
        </button>
      ) : (
        <span className="text-zinc-300">{value}</span>
      )}
    </div>
  );
}
