import { useEffect, useRef, useState } from "react";
import { buildBoard, appendCompanyNote, type CompanyContact, type RawCompany, type Company, type CompanyStatus } from "../lib/companies";

import { toggleTask } from '../lib/checklist';
import { localDate } from '../lib/workspace';

const STATUS_LABEL: Record<CompanyStatus, string> = {
  not_started: "Not started",
  applied: "Applied",
  interviewing: "Interviewing",
  offer: "Offer",
  rejected: "Rejected",
  passed: "Passed",
};

const STATUS_DOT: Record<CompanyStatus, string> = {
  not_started: "bg-zinc-600",
  applied: "bg-blue-500",
  interviewing: "bg-amber-500",
  offer: "bg-emerald-500",
  rejected: "bg-red-500",
  passed: "bg-zinc-700",
};

interface Props {
  initialCompanies: RawCompany[];
  initialRevision: string | null;
  initialRevisions?: Record<string, string>;
  detailSlug?: string;
  skipInitialReload?: boolean;
}

export function CompanyBoard({ initialCompanies, initialRevision, initialRevisions = {}, detailSlug, skipInitialReload = false }: Props) {
  const [companies, setCompanies] = useState(initialCompanies);
  const [revision, setRevision] = useState(initialRevision);
  const [revisions, setRevisions] = useState<Record<string, string>>(initialRevisions);
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [contactDrafts, setContactDrafts] = useState<Record<string, CompanyContact | undefined>>({});
  const [busy, setBusy] = useState(false);
  const lock = useRef(false);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');
  const data = buildBoard(detailSlug ? companies.filter(company => company.slug === detailSlug) : companies);
  const Details = detailSlug ? 'section' : 'details';
  const Heading = detailSlug ? 'h1' : 'h2';
  useEffect(() => {
    const warn = (event: BeforeUnloadEvent) => { if (lock.current || Object.values(drafts).some(note => note.trim()) || Object.values(contactDrafts).some(Boolean)) event.preventDefault(); };
    window.addEventListener('beforeunload', warn);
    return () => window.removeEventListener('beforeunload', warn);
  }, [drafts, contactDrafts]);
  useEffect(() => { if (!skipInitialReload) void reload(); }, []);
  async function persist(slug: string, change: (company: RawCompany) => RawCompany, action: 'step' | 'note' | 'contact' | 'remove-contact' = 'step') {
    if (lock.current) return;
    lock.current = true; setBusy(true); setError(''); setStatus('Saving…');
    try {
      const company = companies.find(company => company.slug === slug);
      if (!company) throw new Error('Company not found. Reload the latest companies.');
      const response = await fetch('/api/entries/company', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ revision: revisions[slug] ?? revision, entry: change(company) }) });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || 'Could not save. Please retry.');
      const savedCompany = result.entry as RawCompany;
      setCompanies(current => current.map(item => item.slug === slug ? savedCompany : item)); setRevision(result.revision); setRevisions(current => ({ ...current, [slug]: result.revision }));
      if (action === 'note') setDrafts(previous => ({ ...previous, [slug]: '' }));
      if (action === 'contact') setContactDrafts(previous => ({ ...previous, [slug]: undefined }));
      setStatus(`${company.title}: ${action === 'note' ? 'note added' : action === 'step' ? 'step saved' : action === 'contact' ? 'contact saved' : 'contact removed'}.`);
      window.dispatchEvent(new Event('workspace-saved'));
    } catch (e) { setError(e instanceof Error ? e.message : 'Could not save. Please retry.'); setStatus('Change not saved. Your drafts are kept.'); }
    finally { lock.current = false; setBusy(false); }
  }
  async function reload() {
    if (lock.current) return;
    lock.current = true; setBusy(true); setError('');
    try {
      const path = detailSlug ? `/api/entries/company?id=${encodeURIComponent(detailSlug)}` : '/api/entries/company?limit=100&view=detail';
      const response = await fetch(path, { cache: 'no-store' });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || 'Could not load companies.');
      if (detailSlug) { setCompanies([result.entry]); setRevision(result.revision); setRevisions({ [detailSlug]: result.revision }); }
      else { setCompanies(result.entries.map((item: { entry: RawCompany }) => item.entry)); setRevisions(Object.fromEntries(result.entries.map((item: { entry: RawCompany; revision: string }) => [item.entry.slug, item.revision]))); }
      setStatus('Latest companies loaded. Drafts kept.');
    } catch (e) { setError(e instanceof Error ? e.message : 'Could not load companies.'); }
    finally { lock.current = false; setBusy(false); }
  }
  const [activeCategory, setActiveCategory] = useState('all');
  const visibleCategories = activeCategory === 'all' ? data.categories : data.categories.filter(group => group.label === activeCategory);
  const totalCompanies = data.categories.reduce((sum, group) => sum + group.companies.length, 0);

  return (
    <div className="space-y-6">
      {!detailSlug && <div className="flex flex-wrap items-center gap-4 text-sm text-zinc-400">
        <span>
          {totalCompanies} companies · {data.totals.applied} applied ·{" "}
          {data.totals.interviewing} interviewing
        </span>
      </div>}

      {error && <div role="alert" className="rounded-lg border border-red-800 p-3 text-sm text-red-200">{error} <button type="button" disabled={busy} className="underline disabled:opacity-50" onClick={() => void reload()}>Reload latest (keep drafts)</button></div>}
      <p role="status" className="text-sm text-zinc-400">{status}</p>
      {!detailSlug && <div className="flex flex-wrap gap-2">
        <button
          type="button"
          aria-pressed={activeCategory === "all"}
          onClick={() => setActiveCategory("all")}
          className={`px-3 py-1.5 rounded-md text-sm border transition-colors ${
            activeCategory === "all"
              ? "bg-zinc-100 text-zinc-900 border-zinc-100"
              : "bg-transparent text-zinc-300 border-zinc-700 hover:border-zinc-500"
          }`}
        >
          All categories
        </button>
        {data.categories.map((t) => (
          <button
            key={t.label}
            type="button"
            aria-pressed={activeCategory === t.label}
            onClick={() => setActiveCategory(t.label)}
            className={`px-3 py-1.5 rounded-md text-sm border transition-colors ${
              activeCategory === t.label
                ? "bg-zinc-100 text-zinc-900 border-zinc-100"
                : "bg-transparent text-zinc-300 border-zinc-700 hover:border-zinc-500"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>}

      {totalCompanies === 0 && <p className="text-zinc-400">{detailSlug ? 'This company is no longer available.' : 'No companies added yet.'}</p>}
      <div className={detailSlug ? 'space-y-6' : 'grid items-start gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4'}>
        {visibleCategories.flatMap(group => group.companies).map(company => (
          <article key={company.slug} className={`min-w-0 rounded-lg border border-zinc-800 bg-zinc-900 ${detailSlug ? 'p-5 sm:p-8' : 'p-3 transition-colors hover:border-zinc-600'}`}>
            <div className="flex items-center gap-3">
              <CompanyImage key={company.cover || company.slug} company={company} />
              <div className="min-w-0 flex-1">
                <Heading className={`${detailSlug ? 'text-3xl' : 'text-sm'} font-semibold leading-tight text-zinc-100`}>
                  {detailSlug ? company.title : <a href={`/companies/${company.slug.split('/').map(encodeURIComponent).join('/')}`} className="hover:text-blue-300 focus-visible:underline">{company.title}</a>}
                </Heading>
                <p className="mt-1 truncate text-[11px] text-zinc-500" title={company.category}>{company.category}</p>
                <span className="mt-1 flex items-center gap-1.5 text-[11px] text-zinc-400">
                  <span className={`h-1.5 w-1.5 rounded-full ${STATUS_DOT[company.status]}`} />
                  {STATUS_LABEL[company.status]}
                  {company.featured && <span className="ml-auto text-amber-300" title="Standout fit">★<span className="sr-only"> Standout fit</span></span>}
                </span>
              </div>
            </div>
            <div className="mt-3">
              <div className="flex justify-between gap-2 text-[10px] text-zinc-500">
                <label htmlFor={`progress-${company.slug}`}>{company.completed}/{company.steps.length} steps</label>
                <span>{company.priority} priority</span>
              </div>
              <progress id={`progress-${company.slug}`} className="mt-1 block h-1 w-full overflow-hidden rounded accent-blue-500" value={company.completed} max={company.steps.length || 1} />
            </div>
            <Details className={`mt-2 ${detailSlug ? 'text-sm' : 'text-xs'} text-zinc-400`}>
              {!detailSlug && <summary className="cursor-pointer py-1 hover:text-zinc-100">Details, notes & contacts</summary>}
              <div className="mt-2 space-y-3 border-t border-zinc-800 pt-3">
                <div className="flex flex-wrap gap-3 text-blue-400">
                  {!detailSlug && <a href={`/companies/${company.slug.split('/').map(encodeURIComponent).join('/')}`} className="hover:underline">Open company →</a>}
                  {company.url && <a href={company.url} target="_blank" rel="noreferrer" className="hover:underline">Careers ↗</a>}
                </div>
                {company.why && <p>{inlineLinks(company.why)}</p>}
                <div className="flex flex-wrap gap-1">{company.tags.map(tag => <span key={tag} className="rounded bg-zinc-800 px-1.5 py-0.5">{tag}</span>)}</div>
                {company.steps.length > 0 && <div><h3 className="mb-1 font-medium text-zinc-200">Application steps</h3><ul className="space-y-1">{company.steps.map((step) => <li key={step.index}><label className="flex cursor-pointer items-start gap-2"><input type="checkbox" className="mt-0.5 accent-blue-500" disabled={busy} checked={step.completed} onChange={event => { const checked = event.target.checked; void persist(company.slug, saved => ({ ...saved, body: toggleTask(saved.body, step.index, checked) })); }} /><span>{step.label}</span></label></li>)}</ul></div>}
                <h3 className="font-medium text-zinc-200">Notes</h3>
                {company.logEntries.length > 0 && <div><ul className="space-y-1">{company.logEntries.map((entry, i) => <li key={i} className="whitespace-pre-wrap break-words">{inlineLinks(entry)}</li>)}</ul></div>}
                <form className="space-y-2" onSubmit={event => { event.preventDefault(); const note = drafts[company.slug] ?? ''; if (note.trim()) void persist(company.slug, saved => ({ ...saved, body: appendCompanyNote(saved.body, note, localDate()) }), 'note'); }}>
                  <label className="block text-zinc-200">Add a note for {company.title}<textarea aria-label={`Add a note for ${company.title}`} required maxLength={5000} disabled={busy} className="mt-1 min-h-20 w-full rounded-lg border border-zinc-700 bg-zinc-950 p-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500" placeholder="A contact, an impression, or a reminder…" value={drafts[company.slug] ?? ''} onChange={event => setDrafts(previous => ({ ...previous, [company.slug]: event.target.value }))} /></label>
                  <button type="submit" disabled={busy || !drafts[company.slug]?.trim()} className="rounded-lg bg-blue-600 px-3 py-2 text-white hover:bg-blue-500 disabled:opacity-50">Add note</button>
                </form>
                <section aria-label={`Contacts for ${company.title}`} className="space-y-2 border-t border-zinc-800 pt-3">
                  <h3 className="font-medium text-zinc-200">Contacts</h3>
                  {!company.contacts?.length && <p>No contacts yet.</p>}
                  {(company.contacts ?? []).map(contact => <div key={contact.id} className="space-y-1 rounded-lg border border-zinc-700 p-2 break-words">
                    <p className="font-medium text-zinc-100">{contact.name}</p>
                    {contact.role && <p>{contact.role}</p>}
                    {contact.email && <a className="block text-blue-400 hover:underline" href={`mailto:${contact.email}`}>{contact.email}</a>}
                    {contact.url && /^https?:\/\//i.test(contact.url) && <a className="block text-blue-400 hover:underline" href={contact.url} target="_blank" rel="noreferrer">Profile ↗</a>}
                    {contact.notes && <p className="whitespace-pre-wrap">{contact.notes}</p>}
                    <div className="flex gap-3"><button type="button" disabled={busy} className="text-blue-400 hover:underline" onClick={() => { if (!contactDrafts[company.slug] || window.confirm('Discard the current contact draft?')) setContactDrafts(previous => ({ ...previous, [company.slug]: { ...contact } })); }}>Edit contact</button>
                    <button type="button" disabled={busy} className="text-red-300 hover:underline" onClick={() => { if (window.confirm(`Remove ${contact.name} from ${company.title}?`)) void persist(company.slug, saved => ({ ...saved, contacts: (saved.contacts ?? []).filter(item => item.id !== contact.id) }), 'remove-contact'); }}>Remove contact</button></div>
                  </div>)}
                  {contactDrafts[company.slug] ? <form className="space-y-2" onSubmit={event => {
                    event.preventDefault(); const contact = contactDrafts[company.slug]!;
                    void persist(company.slug, saved => ({ ...saved, contacts: [...(saved.contacts ?? []).filter(item => item.id !== contact.id), contact] }), 'contact');
                  }}>
                    <fieldset disabled={busy} className="space-y-2">
                      {(['name', 'role', 'email', 'url', 'notes'] as const).map(key => {
                        const contact = contactDrafts[company.slug]!;
                        const label = { name: 'Contact name', role: 'Role', email: 'Email', url: 'Profile URL', notes: 'Contact notes' }[key];
                        const props = { 'aria-label': label, className: 'mt-1 w-full rounded-lg border border-zinc-700 bg-zinc-950 p-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500', value: contact[key], maxLength: key === 'notes' ? 5000 : key === 'email' ? 254 : key === 'url' ? 2000 : 200, onChange: (event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => setContactDrafts(previous => ({ ...previous, [company.slug]: { ...contact, [key]: event.target.value } })) };
                        return <label className="block" key={key}>{label}{key === 'notes' ? <textarea {...props} /> : <input {...props} type={key === 'email' ? 'email' : key === 'url' ? 'url' : 'text'} required={key === 'name'} />}</label>;
                      })}
                      <div className="flex gap-3"><button type="submit" className="rounded-lg bg-blue-600 px-3 py-2 text-white">Save contact</button><button type="button" onClick={() => setContactDrafts(previous => ({ ...previous, [company.slug]: undefined }))}>Cancel contact</button></div>
                    </fieldset>
                  </form> : <button type="button" disabled={busy} className="text-blue-400 hover:underline" onClick={() => setContactDrafts(previous => ({ ...previous, [company.slug]: { id: crypto.randomUUID(), name: '', role: '', email: '', url: '', notes: '' } }))}>+ Add contact</button>}
                </section>
              </div>
            </Details>
          </article>
        ))}
      </div>
    </div>
  );
}

// Bundled brand icons work for existing saved companies without a data migration.
const bundledImages = new Set(["amazon", "anthropic", "baseten", "buoyant", "chronosphere", "clickhouse", "cloudflare", "cockroachdb", "datadog", "dynatrace", "fireworks-ai", "grafana-labs", "greptime", "groundcover", "hashicorp", "honeycomb", "isovalent", "modal", "mongodb", "netflix", "new-relic", "openai", "polar-signals", "redpanda", "scylladb", "tigerbeetle", "together-ai", "yugabyte"]);
function CompanyImage({ company }: { company: Company }) {
  const local = bundledImages.has(company.slug) ? `/images/companies/${company.slug}.png` : undefined;
  const [failed, setFailed] = useState<string[]>([]);
  const source = [company.cover, local].find(url => url && !failed.includes(url));
  const image = useRef<HTMLImageElement>(null);
  useEffect(() => {
    // An image can fail before React hydrates and attaches its error handler.
    if (source && image.current?.complete && !image.current.naturalWidth) {
      setFailed(previous => [...previous, source]);
    }
  }, [source]);
  return <div className="flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-white p-1.5 ring-1 ring-white/10">
    {source ? <img ref={image} src={source} alt={`${company.title} logo`} width={48} height={48} loading="lazy" decoding="async" className="h-full w-full object-contain" onError={() => setFailed(previous => [...previous, source])} />
      : <span className="text-lg font-bold text-zinc-700" aria-label={company.title}>{company.title.split(/\s+/).map(word => word[0]).join('').slice(0, 2).toUpperCase()}</span>}
  </div>;
}

// Render source links without injecting raw HTML from markdown.
function inlineLinks(text: string) {
  return text.split(/(\[[^\]]+\]\(https?:\/\/[^\s)]+\))/g).map((part, index) => {
    const link = part.match(/^\[([^\]]+)\]\((https?:\/\/[^\s)]+)\)$/);
    return link ? <a key={index} href={link[2]} target="_blank" rel="noreferrer" className="underline hover:text-zinc-200">{link[1]}</a> : part;
  });
}
