import { useEffect, useState } from 'react';
import { BookShelf } from './BookShelf';
import { CompanyBoard } from './CompanyBoard';
import { LogExplorer } from './LogExplorer';
import MarkdownPreview from './MarkdownPreview';
import { ProgressDashboard } from './ProgressDashboard';
import { buildShelf, type RawBook } from '../lib/books';
import { buildLogDocs, type WeekSummary } from '../lib/logs';
import { buildDashboard, TRACK_KEYS, type WeekEntry } from '../lib/progress';
import type { RawCompany } from '../lib/companies';
import type { Document } from '../lib/workspace';
import type { Goal } from '../lib/timeline';

type Result<T> = { entry: T; revision: string };
type Page<T> = { entries: Result<T>[]; nextOffset: number | null };

function expired(response: Response) {
  if (response.status !== 401) return false;
  location.assign(`/login?redirect=${encodeURIComponent(location.pathname + location.search)}`);
  return true;
}

function useJSON<T>(url: string) {
  const [data, setData] = useState<T>();
  const [error, setError] = useState('');
  useEffect(() => {
    const controller = new AbortController();
    const load = () => { void fetch(url, { cache: 'no-store', signal: controller.signal }).then(async response => {
      if (expired(response)) return;
      const value = await response.json();
      if (!response.ok) throw new Error(value.error || 'Could not load this page.');
      setData(value);
    }).catch(value => { if (value.name !== 'AbortError') setError(value instanceof Error ? value.message : 'Could not load this page.'); }); };
    load();
    window.addEventListener('workspace-saved', load);
    return () => { controller.abort(); window.removeEventListener('workspace-saved', load); };
  }, [url]);
  return { data, error };
}

function State({ error }: { error: string }) {
  return <p role={error ? 'alert' : 'status'} className={error ? 'text-red-300' : 'text-zinc-400'}>{error || 'Loading…'}</p>;
}

export function BooksView() {
  const { data, error } = useJSON<Page<RawBook>>('/api/entries/book?limit=100&view=detail');
  return data ? <BookShelf data={buildShelf(data.entries.map(item => item.entry))} /> : <State error={error} />;
}

export function CompaniesView({ detail = false }: { detail?: boolean }) {
  const slug = detail ? decodeURIComponent(location.pathname.slice('/companies/'.length)) : '';
  const url = slug ? `/api/entries/company?id=${encodeURIComponent(slug)}` : '/api/entries/company?limit=100&view=detail';
  const { data, error } = useJSON<Page<RawCompany> | Result<RawCompany>>(url);
  if (!data) return <State error={error} />;
  if ('entry' in data) return <CompanyBoard initialCompanies={[data.entry]} initialRevision={data.revision} initialRevisions={{ [slug]: data.revision }} detailSlug={slug} skipInitialReload />;
  return <CompanyBoard initialCompanies={data.entries.map(item => item.entry)} initialRevision={null} initialRevisions={Object.fromEntries(data.entries.map(item => [item.entry.slug, item.revision]))} skipInitialReload />;
}

function weeks(page: Page<WeekEntry>): WeekEntry[] { return page.entries.map(item => item.entry); }

export function LogsView() {
  const { data, error } = useJSON<Page<WeekEntry>>('/api/entries/week?limit=100&view=detail');
  if (!data) return <State error={error} />;
  const entries = weeks(data);
  const summaries: WeekSummary[] = entries.map(({ week, year, dates, hours, tags }) => ({ week, year, dates, hours, tags }));
  return <LogExplorer docs={buildLogDocs(entries)} weeks={summaries} />;
}

export function ProgressView({ compact = false }: { compact?: boolean }) {
  const { data, error } = useJSON<{ entries: WeekEntry[] }>('/api/summaries/progress');
  if (!data) return <State error={error} />;
  const dashboard = buildDashboard(data.entries);
  if (!compact) return <ProgressDashboard data={dashboard} />;
  return <section className="rounded-lg border border-zinc-800 bg-zinc-900 p-6 space-y-4"><h2 className="text-xl font-semibold">This week · W{dashboard.currentWeek}</h2><p className="text-zinc-400">{dashboard.currentDates} · {dashboard.currentHours}h logged</p><div className="grid grid-cols-2 gap-3">{dashboard.tracks.filter(track => TRACK_KEYS.some(key => key === track.key)).map(track => <div key={track.key} className="rounded bg-zinc-800 p-3"><h3 style={{ color: track.color }}>{track.name}</h3><p>{track.current}h</p></div>)}</div><a href="/progress" className="text-blue-400 hover:underline">Progress dashboard →</a></section>;
}

export function DocumentView({ id, home = false }: { id?: string; home?: boolean }) {
  const rawPath = location.pathname.replace(/^\//, '').replace(/\/$/, '');
  const pathID = decodeURIComponent(rawPath.startsWith('documents/') ? rawPath.slice('documents/'.length) : (/^2026\/(languages|system-design|os-oss)$/.test(rawPath) ? `${rawPath}/index` : rawPath));
  const documentID = id ?? pathID;
  const { data, error } = useJSON<Result<Document>>(`/api/entries/document?id=${encodeURIComponent(documentID)}`);
  if (!data) return <State error={error} />;
  const page = data.entry;
  return <>{home && <header><h1 className="text-3xl font-bold">{page.title}</h1><p className="text-zinc-400 mt-2">{page.description}</p><a className="mt-2 inline-block text-sm text-blue-400 hover:underline" href="/documents?id=index">Edit home page →</a></header>}{!home && <a className="mb-6 inline-block text-sm text-blue-400 hover:underline" href={`/documents?id=${encodeURIComponent(page.id)}`}>Edit this page →</a>}{!home && !/^# /m.test(page.body) ? <h1 className="mb-6 text-3xl font-bold">{page.title}</h1> : null}<MarkdownPreview body={page.body} /></>;
}

type AgentContext = { goals: Goal[]; rules: string[] };
export function AgentView() {
  const { data, error } = useJSON<AgentContext>('/api/agent-context');
  if (!data) return <State error={error} />;
  return <section className="space-y-4" aria-label="Current timeline goals"><h2 className="text-xl font-semibold">Current timeline goals ({data.goals.length})</h2>{data.goals.length === 0 ? <p className="text-zinc-400">No goals saved yet.</p> : null}{data.goals.map(goal => <article key={goal.id} className="rounded-lg border border-zinc-700 p-4"><h3 className="font-semibold">{goal.title}</h3><p className="mt-1 text-sm text-zinc-400">{goal.startDate} → {goal.endDate} · {goal.steps.filter(step => step.done).length}/{goal.steps.length} substeps complete</p></article>)}</section>;
}
