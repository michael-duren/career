import { checklist, toggleTask } from '../lib/checklist';
import type { WorkspaceEntry } from '../lib/workspace';
import { BOOK_CATEGORIES } from '../lib/books';
import { TRACK_META } from '../lib/progress';
import { Combobox } from './NoteInputs';
import { isAutoTitle } from '../lib/audio-thoughts';

const field = 'mt-1 w-full rounded-lg border border-zinc-700 bg-zinc-950 px-3 py-2 text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500';
function Text({ label, value, onChange, required = false, type = 'text' }: { label: string; value?: string; onChange: (value: string) => void; required?: boolean; type?: string }) {
  return <label className="block text-sm">{label}<input className={field} type={type} value={value ?? ''} required={required} maxLength={1000} onChange={e => onChange(e.target.value)} /></label>;
}
function Select({ label, value, options, onChange }: { label: string; value: string; options: readonly string[]; onChange: (value: string) => void }) {
  return <label className="block text-sm">{label}<select className={field} value={value} onChange={e => onChange(e.target.value)}>{options.map(option => <option key={option} value={option}>{option.replaceAll('_', ' ')}</option>)}</select></label>;
}
export default function EntryFields({ entry, change, topics = [] }: { entry: WorkspaceEntry; change: (patch: Partial<WorkspaceEntry>) => void; topics?: string[] }) {
  if ('week' in entry) return <>
    <h2 className="text-lg font-semibold">Week {entry.week} · {entry.dates}</h2>
    <div className="grid grid-cols-2 gap-3">{[...new Set(['ostep', 'ebpf', 'database', 'systemDesign', ...Object.keys(entry.hours)])].map(key => <label className="block text-xs text-zinc-400" key={key}>{TRACK_META[key]?.name ?? key} hours<input className={field} type="number" min="0" max="168" step="0.25" value={entry.hours[key] ?? 0} onChange={e => change({ hours: { ...entry.hours, [key]: Number(e.target.value) } })} /></label>)}</div>
  </>;
  return <>
    {'runDate' in entry
      ? <label className="block text-sm">Title<input className={field} value={isAutoTitle(entry.title) ? '' : entry.title} autoFocus maxLength={200} placeholder="Leave blank to name it after the first few words" onChange={e => change({ title: e.target.value })} /></label>
      : <label className="block text-sm">Title<input className={field} value={entry.title} autoFocus required maxLength={200} onChange={e => change({ title: e.target.value })} /></label>}
    {'runDate' in entry && <><Text label="Date" type="date" value={entry.runDate} onChange={runDate => change({ runDate })} /><label className="block text-sm">Duration (minutes)<input className={field} type="number" min="0" max={1440} step="any" value={entry.durationMin ?? ''} onChange={e => change({ durationMin: e.target.value === '' ? undefined : Number(e.target.value) })} /></label></>}
    {'date' in entry && <Text label="Entry date (optional for background)" type="date" value={entry.date} onChange={date => change({ date })} />}
    {'topic' in entry && <Combobox label="Topic" value={entry.topic} options={topics} required maxLength={100} onChange={topic => change({ topic })} />}
    {'description' in entry && <Text label="Summary" value={entry.description} onChange={description => change({ description })} />}
    {'status' in entry && <>
      <div className="grid gap-3 sm:grid-cols-2">
        <Select label="Status" value={entry.status} options={entry.type === 'company' ? ['not_started', 'applied', 'interviewing', 'offer', 'rejected', 'passed'] : ['backlog', 'reading', 'paused', 'completed', 'reference']} onChange={status => change({ status } as Partial<WorkspaceEntry>)} />
        <Select label="Priority" value={entry.priority} options={['high', 'medium', 'low']} onChange={priority => change({ priority } as Partial<WorkspaceEntry>)} />
        {'authors' in entry ? <Select label="Category" value={entry.category} options={BOOK_CATEGORIES} onChange={category => change({ category } as Partial<WorkspaceEntry>)} /> : <Text label="Category" value={entry.category} required onChange={category => change({ category })} />}
        <Text label="Website" type="url" value={entry.url} required={entry.type === 'company'} onChange={url => change({ url })} />
      </div>
      <Text label="Cover image URL" type="url" value={entry.cover} onChange={cover => change({ cover })} />
      <label className="flex gap-2 text-sm"><input type="checkbox" checked={entry.featured} onChange={e => change({ featured: e.target.checked })} />Featured</label>
    </>}
    {'authors' in entry && <>
      <Select label="Type" value={entry.type} options={['book', 'course']} onChange={type => change({ type } as Partial<WorkspaceEntry>)} />
      <label className="block text-sm">Authors<textarea aria-label="Authors, one per line" className={field} value={entry.authors.join('\n')} onChange={e => change({ authors: e.target.value.split('\n') })} /></label>
      <div className="grid gap-3 sm:grid-cols-2"><Text label="Edition" value={entry.edition} onChange={edition => change({ edition })} /><Text label="ISBN" value={entry.isbn} onChange={isbn => change({ isbn })} /><Text label="Started" type="date" value={entry.started} onChange={started => change({ started })} /><Text label="Finished" type="date" value={entry.finished} onChange={finished => change({ finished })} /></div>
      <label className="block text-sm">Rating (0–5)<input className={field} type="number" min="0" max="5" step="0.5" value={entry.rating ?? ''} onChange={e => change({ rating: e.target.value === '' ? undefined : Number(e.target.value) })} /></label>
      <label className="flex gap-2 text-sm"><input type="checkbox" checked={!!entry.progress} onChange={e => change({ progress: e.target.checked ? { unit: 'page', completed: 0, total: 0 } : undefined })} />Enter a progress count instead of using the checklist</label>
      {entry.progress && <div className="grid gap-3 sm:grid-cols-3">
        <Select label="Unit" value={entry.progress.unit} options={['chapter', 'page', 'module', 'section', 'lecture']} onChange={unit => change({ progress: { ...entry.progress!, unit } } as Partial<WorkspaceEntry>)} />
        {(['completed', 'total'] as const).map(key => <label key={key} className="block text-sm">{key === 'completed' ? 'Completed' : 'Total'}<input className={field} type="number" min="0" max="100000" step="1" value={entry.progress![key]} onChange={e => change({ progress: { ...entry.progress!, [key]: Number(e.target.value) } })} /></label>)}
      </div>}
    </>}
  </>;
}

export function EntryChecklist({ body, onChange }: { body: string; onChange: (body: string) => void }) {
  const items = checklist(body);
  if (!items.length) return null;
  return <fieldset className="space-y-2 rounded-lg border border-zinc-700 p-3"><legend className="px-1 text-sm text-zinc-400">Checklist</legend>{items.map(item => <label key={item.index} className="flex items-start gap-2 text-sm"><input type="checkbox" className="mt-1" checked={item.checked} onChange={e => {
    onChange(toggleTask(body, item.index, e.target.checked));
  }} />{item.label}</label>)}</fieldset>;
}
