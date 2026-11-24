import { useRef, useState, type KeyboardEvent } from 'react';
import { BookHeart, ListFilter, NotebookPen } from 'lucide-react';
import { LogsView } from './StaticDataViews';
import { WorkspaceEditor } from './WorkspaceEditor';

const TABS = [
  { id: 'work', label: 'Work', icon: NotebookPen },
  { id: 'personal', label: 'Personal', icon: BookHeart },
  { id: 'logs', label: 'Logs', icon: ListFilter },
] as const;
type Tab = (typeof TABS)[number]['id'];
const isTab = (value: string | null): value is Tab => TABS.some(tab => tab.id === value);

export function JournalHub() {
  const [tab, setTabState] = useState<Tab>(() => { const value = new URLSearchParams(location.search).get('tab'); return isTab(value) ? value : 'work'; });
  const [visited, setVisited] = useState<Set<Tab>>(() => new Set([tab]));
  const tabRefs = useRef<Record<string, HTMLButtonElement | null>>({});
  function setTab(next: Tab, focus = false) {
    setTabState(next); setVisited(current => new Set(current).add(next));
    const url = new URL(location.href);
    if (next === 'work') url.searchParams.delete('tab'); else url.searchParams.set('tab', next);
    // An opened entry id belongs to the tab it was opened from.
    url.searchParams.delete('id');
    history.replaceState(history.state, '', url);
    if (focus) tabRefs.current[next]?.focus();
  }
  function onKey(event: KeyboardEvent) {
    const index = TABS.findIndex(item => item.id === tab);
    const move = event.key === 'ArrowRight' ? 1 : event.key === 'ArrowLeft' ? -1 : 0;
    if (event.key === 'Home') { event.preventDefault(); setTab(TABS[0].id, true); }
    else if (event.key === 'End') { event.preventDefault(); setTab(TABS[TABS.length - 1].id, true); }
    else if (move) { event.preventDefault(); setTab(TABS[(index + move + TABS.length) % TABS.length].id, true); }
  }
  return <div className="space-y-6">
    <header className="flex flex-wrap items-center justify-between gap-3">
      <h1 className="text-3xl font-bold">Journal</h1>
      <a href="/journal/new" className="rounded-lg border border-blue-500 bg-blue-600 px-4 py-2 text-sm font-semibold text-white hover:bg-blue-500">+ Add work entry</a>
    </header>
    <div role="tablist" aria-label="Journal sections" className="flex gap-1 overflow-x-auto border-b border-zinc-800" onKeyDown={onKey}>
      {TABS.map(item => <button key={item.id} ref={node => { tabRefs.current[item.id] = node; }} role="tab" id={`tab-${item.id}`} aria-controls={`panel-${item.id}`} aria-selected={tab === item.id} tabIndex={tab === item.id ? 0 : -1}
        className={`inline-flex shrink-0 items-center gap-2 border-b-2 px-4 py-2 text-sm font-medium ${tab === item.id ? 'border-blue-500 text-zinc-100' : 'border-transparent text-zinc-400 hover:text-zinc-200'}`} onClick={() => setTab(item.id)}>
        <item.icon className="size-4" aria-hidden />{item.label}
      </button>)}
    </div>
    {TABS.map(item => visited.has(item.id) && <div key={item.id} role="tabpanel" id={`panel-${item.id}`} aria-labelledby={`tab-${item.id}`} hidden={tab !== item.id} className="space-y-4">
      {item.id === 'work' && <><p className="text-sm"><a className="text-blue-400 underline" href="/api/agent-context?format=markdown&journal=work">Read work journal as Markdown</a></p><WorkspaceEditor kind="week" /></>}
      {item.id === 'personal' && <><p className="text-sm"><a className="text-blue-400 underline" href="/api/agent-context?format=markdown&journal=personal">Read personal journal as Markdown</a></p><WorkspaceEditor kind="personal" /></>}
      {item.id === 'logs' && <LogsView />}
    </div>)}
  </div>;
}
