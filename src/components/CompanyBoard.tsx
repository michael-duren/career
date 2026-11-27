import { useEffect, useRef, useState } from "react";
import { buildBoard, appendCompanyNote, companyLogoUrl, type RawCompany, type Company, type CompanyStatus } from "../lib/companies";
import { moduleEnabled } from '../lib/modules';
import { photoURL, relativeDays, type Connection } from '../lib/connections';
import { toggleTask } from '../lib/checklist';
import { localDate } from '../lib/workspace';
import { PriorityDot } from './CompanyGrid';
import { Avatar, Button, Checkbox, ExternalIcon, Label, LogoSpot, SectionHeader, buttonClassName } from './ui';

const STATUS_LABEL: Record<CompanyStatus, string> = {
  not_started: "Not started",
  applied: "Applied",
  interviewing: "Interviewing",
  offer: "Offer",
  rejected: "Rejected",
  passed: "Passed",
};

const PRIORITY_TEXT: Record<Company['priority'], string> = { high: 'High priority', medium: 'Medium priority', low: 'Low priority' };

interface Props {
  initialCompany: RawCompany;
  initialRevision: string;
  detailSlug: string;
}

// One company's page (/companies/<slug>): steps, notes, connections, edit and delete.
// The list of companies is CompanyGrid.
export function CompanyBoard({ initialCompany, initialRevision, detailSlug }: Props) {
  const [companies, setCompanies] = useState([initialCompany]);
  const [revisions, setRevisions] = useState<Record<string, string>>({ [detailSlug]: initialRevision });
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [connections, setConnections] = useState<Connection[]>();
  const [connectionsFailed, setConnectionsFailed] = useState(false);
  // Connections belong to the People module; while it is off they are not shown here.
  const [peopleOn, setPeopleOn] = useState(true);
  const [busy, setBusy] = useState(false);
  const lock = useRef(false);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');
  const data = buildBoard(companies.filter(company => company.slug === detailSlug));
  useEffect(() => {
    const warn = (event: BeforeUnloadEvent) => { if (lock.current || Object.values(drafts).some(note => note.trim())) event.preventDefault(); };
    window.addEventListener('beforeunload', warn);
    return () => window.removeEventListener('beforeunload', warn);
  }, [drafts]);
  useEffect(() => {
    const url = `/api/connections?company=${encodeURIComponent(detailSlug)}`;
    setConnectionsFailed(false);
    void moduleEnabled('people').then(on => {
      setPeopleOn(on);
      if (!on) return;
      return fetch(url, { cache: 'no-store' }).then(response => {
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return response.json();
      }).then(result => setConnections(result.connections)).catch(() => setConnectionsFailed(true));
    });
  }, [detailSlug]);
  async function persist(slug: string, change: (company: RawCompany) => RawCompany, action: 'step' | 'note' = 'step') {
    if (lock.current) return;
    lock.current = true; setBusy(true); setError(''); setStatus('Saving…');
    try {
      const company = companies.find(company => company.slug === slug);
      if (!company) throw new Error('Company not found. Reload the latest companies.');
      const response = await fetch('/api/entries/company', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ revision: revisions[slug], entry: change(company) }) });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || 'Could not save. Please retry.');
      const savedCompany = result.entry as RawCompany;
      setCompanies(current => current.map(item => item.slug === slug ? savedCompany : item)); setRevisions(current => ({ ...current, [slug]: result.revision }));
      if (action === 'note') setDrafts(previous => ({ ...previous, [slug]: '' }));
      setStatus(`${company.title}: ${action === 'note' ? 'note added' : 'step saved'}.`);
      window.dispatchEvent(new Event('workspace-saved'));
    } catch (e) { setError(e instanceof Error ? e.message : 'Could not save. Please retry.'); setStatus('Change not saved. Your drafts are kept.'); }
    finally { lock.current = false; setBusy(false); }
  }
  async function remove(slug: string) {
    const company = companies.find(company => company.slug === slug);
    if (lock.current || !company || !window.confirm(`Delete ${company.title}? This cannot be undone.`)) return;
    lock.current = true; setBusy(true); setError(''); setStatus('Deleting…');
    try {
      const response = await fetch('/api/entries/company', { method: 'DELETE', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: slug, revision: revisions[slug] }) });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || 'Could not delete. Please retry.');
      window.dispatchEvent(new Event('workspace-saved'));
      // Release the lock first so the beforeunload guard doesn't prompt on the way out.
      lock.current = false; window.location.assign('/companies');
    } catch (e) { setError(e instanceof Error ? e.message : 'Could not delete. Please retry.'); setStatus('Company not deleted.'); }
    finally { lock.current = false; setBusy(false); }
  }
  async function reload() {
    if (lock.current) return;
    lock.current = true; setBusy(true); setError('');
    try {
      const response = await fetch(`/api/entries/company?id=${encodeURIComponent(detailSlug)}`, { cache: 'no-store' });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || 'Could not load companies.');
      setCompanies([result.entry]); setRevisions({ [detailSlug]: result.revision });
      setStatus('Latest companies loaded. Drafts kept.');
    } catch (e) { setError(e instanceof Error ? e.message : 'Could not load companies.'); }
    finally { lock.current = false; setBusy(false); }
  }
  const visibleCompanies = data.categories.flatMap(group => group.companies);
  const totalCompanies = visibleCompanies.length;

  return (
    <div className="space-y-6">
      {error && <div role="alert" className="border border-solid border-accent p-3 text-sm text-ink">{error} <button type="button" disabled={busy} className="underline disabled:opacity-50" onClick={() => void reload()}>Reload latest (keep drafts)</button></div>}
      <p role="status" className="text-sm text-ink-60 empty:hidden">{status}</p>
      {totalCompanies === 0 && <p className="text-ink-60">This company is no longer available.</p>}
      {visibleCompanies.map(company => (
        <article key={company.slug} className="min-w-0 max-w-[920px] space-y-6">
          <header className="flex min-w-0 flex-wrap items-start gap-x-4 gap-y-3">
            <LogoSpot name={company.title} src={companyLogoUrl(company.url)} size={40} labelled />
            <div className="flex min-w-0 flex-1 flex-col gap-2">
              <Label className="truncate" title={company.category}>{company.category}</Label>
              <h1 className="m-0 font-extralight !text-[40px] !leading-[0.95] md:!text-[52px]">{company.title}</h1>
              <span className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm font-light text-ink-60">
                <span>{STATUS_LABEL[company.status]}</span>
                <span aria-hidden="true">·</span>
                <span className="inline-flex items-center gap-2"><PriorityDot priority={company.priority} />{PRIORITY_TEXT[company.priority]}</span>
                {company.featured && <><span aria-hidden="true">·</span><span className="inline-flex items-center gap-1.5"><span aria-hidden="true" className="text-[10px]">◆</span>Featured</span></>}
              </span>
            </div>
            <div className="flex w-full flex-none gap-2 sm:w-auto">
              <a href={`/manage/companies?id=${encodeURIComponent(company.slug)}`} aria-label={`Edit ${company.title}`} className={buttonClassName({ size: 'sm' })}>Edit</a>
              <Button variant="danger" size="sm" disabled={busy} aria-label={`Delete ${company.title}`} onClick={() => void remove(company.slug)}>Delete</Button>
            </div>
          </header>
          {(company.url || company.why || company.tags.length > 0) && <section className="space-y-3 border-t border-solid border-ink-14 pt-4 text-sm font-light">
            {company.url && <a href={company.url} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1.5 text-ink underline decoration-ink-50 underline-offset-2 hover:text-ink-hover">Careers<ExternalIcon size={12} /></a>}
            {company.why && <p className="m-0 leading-relaxed text-ink">{inlineLinks(company.why)}</p>}
            {company.tags.length > 0 && <div className="flex flex-wrap gap-2">{company.tags.map(tag => <span key={tag} className="border border-solid border-ink-30 px-2 py-0.5 font-mono text-[10px] uppercase tracking-[0.08em] text-ink">{tag}</span>)}</div>}
          </section>}
          {company.steps.length > 0 && <section aria-labelledby={`steps-${company.slug}`} className="space-y-3">
            <SectionHeader id={`steps-${company.slug}`} title="Application steps" meta={`${company.completed} of ${company.steps.length} done`} rule />
            <ul className="m-0 list-none space-y-1 p-0">{company.steps.map(step => <li key={step.index}>
              <label className="flex cursor-pointer items-center gap-3 text-sm font-light">
                <Checkbox disabled={busy} checked={step.completed} onChange={event => { const checked = event.target.checked; void persist(company.slug, saved => ({ ...saved, body: toggleTask(saved.body, step.index, checked) })); }} />
                <span className={step.completed ? 'text-ink-60 line-through' : undefined}>{step.label}</span>
              </label>
            </li>)}</ul>
          </section>}
          <section aria-labelledby={`notes-${company.slug}`} className="space-y-3">
            <SectionHeader id={`notes-${company.slug}`} title="Notes" rule />
            {company.logEntries.length > 0 && <ul className="m-0 list-none space-y-2 p-0 text-sm font-light">{company.logEntries.map((entry, i) => <li key={i} className="whitespace-pre-wrap break-words">{inlineLinks(entry)}</li>)}</ul>}
            <form className="space-y-2 pt-2" onSubmit={event => { event.preventDefault(); const note = drafts[company.slug] ?? ''; if (note.trim()) void persist(company.slug, saved => ({ ...saved, body: appendCompanyNote(saved.body, note, localDate()) }), 'note'); }}>
              <label className="block text-sm text-ink-60">Add a note for {company.title}<textarea aria-label={`Add a note for ${company.title}`} required maxLength={5000} disabled={busy} className="mt-1 min-h-20 w-full border border-solid border-ink-30 bg-paper p-2 text-sm text-ink placeholder:text-ink-60 focus:border-ink" placeholder="A contact, an impression, or a reminder…" value={drafts[company.slug] ?? ''} onChange={event => setDrafts(previous => ({ ...previous, [company.slug]: event.target.value }))} /></label>
              <Button type="submit" variant="primary" size="sm" disabled={busy || !drafts[company.slug]?.trim()}>Add note</Button>
            </form>
          </section>
          {peopleOn && <CompanyConnections company={company} connections={connections} failed={connectionsFailed} />}
        </article>
      ))}
    </div>
  );
}

function CompanyConnections({ company, connections: people = [], failed }: { company: Company; connections?: Connection[]; failed: boolean }) {
  const today = localDate();
  const slug = encodeURIComponent(company.slug);
  const link = 'text-ink underline decoration-ink-50 underline-offset-2 hover:text-ink-hover';
  return <section aria-label={`Connections at ${company.title}`} className="space-y-3 text-sm font-light">
    <SectionHeader title="Connections" meta={people.length > 0 ? String(people.length) : undefined} rule />
    {people.length === 0 && <p className="text-ink-60">{failed ? "Couldn't load connections." : 'No connections here yet.'}</p>}
    <ul className="m-0 list-none p-0">
      {people.slice(0, 8).map(person => <li key={person.id}>
        <a href={`/connections?id=${encodeURIComponent(person.id)}`} className="flex items-center gap-3 border-b border-solid border-ink-14 py-2 text-ink no-underline hover:bg-ink-05">
          <Avatar name={person.name} src={photoURL(person)} size={28} />
          <span className="min-w-0 flex-1">
            <span className="block truncate">{person.name}</span>
            {person.role && <span className="block truncate text-ink-60">{person.role}</span>}
          </span>
          <span className="shrink-0 font-mono text-[11px] text-ink-60" title="Last conversation">{relativeDays(person.lastContactedOn, today)}</span>
        </a>
      </li>)}
    </ul>
    <div className="flex flex-wrap gap-4">
      {people.length > 8 && <a className={link} href={`/connections?company=${slug}`}>All {people.length} connections →</a>}
      <a className={link} href={`/connections?new=1&company=${slug}`}>+ Add connection</a>
    </div>
  </section>;
}

// Render source links without injecting raw HTML from markdown.
function inlineLinks(text: string) {
  return text.split(/(\[[^\]]+\]\(https?:\/\/[^\s)]+\))/g).map((part, index) => {
    const link = part.match(/^\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)$/);
    return link ? <a key={index} href={link[2]} target="_blank" rel="noreferrer" className="underline decoration-ink-50 hover:text-ink-hover">{link[1]}</a> : part;
  });
}
