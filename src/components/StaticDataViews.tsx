import { useEffect, useState } from 'react';
import { BookShelf } from './BookShelf';
import { CompanyBoard } from './CompanyBoard';
import MarkdownPreview from './MarkdownPreview';
import { buildShelf, type RawBook } from '../lib/books';
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

function usePagedEntries<T>(url: string) {
  const [data, setData] = useState<T[]>();
  const [error, setError] = useState('');
  useEffect(() => {
    const controller = new AbortController();
    let latestRequest = 0;
    const load = () => {
      const request = ++latestRequest;
      void (async () => {
        const entries: T[] = [];
        let offset: number | null = 0;
        while (offset !== null) {
          const response = await fetch(`${url}&offset=${offset}`, { cache: 'no-store', signal: controller.signal });
          if (expired(response)) return;
          const value = await response.json();
          if (!response.ok) throw new Error(value.error || 'Could not load this page.');
          const page: Page<T> = value;
          entries.push(...page.entries.map(item => item.entry));
          offset = page.nextOffset;
        }
        if (request === latestRequest) { setData(entries); setError(''); }
      })().catch(value => {
        if (request === latestRequest && value.name !== 'AbortError') setError(value instanceof Error ? value.message : 'Could not load this page.');
      });
    };
    load();
    window.addEventListener('workspace-saved', load);
    return () => { controller.abort(); window.removeEventListener('workspace-saved', load); };
  }, [url]);
  return { data, error };
}

function State({ error }: { error: string }) {
  return <p role={error ? 'alert' : 'status'} className={error ? 'text-accent' : 'text-ink-60'}>{error || 'Loading…'}</p>;
}

export function BooksView() {
  const { data, error } = usePagedEntries<RawBook>('/api/entries/book?limit=100&view=detail');
  return data ? <BookShelf data={buildShelf(data)} /> : <State error={error} />;
}

export function CompanyView() {
  const slug = decodeURIComponent(location.pathname.slice('/companies/'.length));
  const { data, error } = useJSON<Result<RawCompany>>(`/api/entries/company?id=${encodeURIComponent(slug)}`);
  if (!data) return <State error={error} />;
  return <CompanyBoard initialCompany={data.entry} initialRevision={data.revision} detailSlug={slug} />;
}


export function DocumentView({ id }: { id?: string }) {
  const rawPath = location.pathname.replace(/^\//, '').replace(/\/$/, '');
  const pathID = decodeURIComponent(rawPath.startsWith('documents/') ? rawPath.slice('documents/'.length) : (/^2026\/(languages|system-design|os-oss)$/.test(rawPath) ? `${rawPath}/index` : rawPath));
  const documentID = id ?? pathID;
  const { data, error } = useJSON<Result<Document>>(`/api/entries/document?id=${encodeURIComponent(documentID)}`);
  if (!data) return <State error={error} />;
  const page = data.entry;
  return <>{!/^# /m.test(page.body) ? <h1 className="mb-6 text-3xl font-bold">{page.title}</h1> : null}<MarkdownPreview body={page.body} /></>;
}

type AgentContext = { goals: Goal[]; rules: string[] };
export function AgentView() {
  const { data, error } = useJSON<AgentContext>('/api/agent-context');
  if (!data) return <State error={error} />;
  return <section className="space-y-4" aria-label="Current timeline goals"><h2 className="text-xl font-semibold">Current timeline goals ({data.goals.length})</h2>{data.goals.length === 0 ? <p className="text-zinc-400">No goals saved yet.</p> : null}{data.goals.map(goal => <article key={goal.id} className="rounded-lg border border-zinc-700 p-4"><h3 className="font-semibold">{goal.title}</h3><p className="mt-1 text-sm text-zinc-400">{goal.startDate} → {goal.endDate} · {goal.steps.filter(step => step.done).length}/{goal.steps.length} mini goals complete</p></article>)}</section>;
}
