import { useId, useState, type KeyboardEvent } from 'react';
import type { NoteTodo } from '../lib/workspace';

const field = 'w-full rounded-lg border border-zinc-700 bg-zinc-950 px-3 py-2 text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500';
const button = 'rounded-lg border border-zinc-700 px-3 py-2 text-sm hover:bg-zinc-800 disabled:opacity-50 disabled:cursor-not-allowed';

/** Keyboard state for a suggestion popup; Enter only picks when a row is highlighted. */
function useSuggestions(matches: string[], pick: (option: string) => void) {
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const listId = useId();
  function onKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      setOpen(true);
      if (!matches.length) return;
      const step = event.key === 'ArrowDown' ? 1 : -1;
      setActive(current => (current + step + matches.length) % matches.length);
    } else if (event.key === 'Enter' && open && matches[active]) {
      event.preventDefault();
      pick(matches[active]); setActive(-1);
    } else if (event.key === 'Escape' && open) {
      event.preventDefault();
      setOpen(false); setActive(-1);
    }
  }
  const list = open && matches.length > 0 && <ul id={listId} role="listbox" className="absolute z-20 mt-1 max-h-60 w-full overflow-y-auto rounded-lg border border-zinc-700 bg-zinc-900 py-1 shadow-lg">
    {matches.map((option, index) => <li key={option} id={`${listId}-${index}`} role="option" aria-selected={index === active}
      className={`cursor-pointer px-3 py-2 text-sm ${index === active ? 'bg-blue-600/30 text-blue-100' : 'text-zinc-200 hover:bg-zinc-800'}`}
      // Keep focus in the input so blur does not close the list before the click lands.
      onMouseDown={event => event.preventDefault()} onClick={() => { pick(option); setActive(-1); }}>{option}</li>)}
  </ul>;
  const inputProps = {
    role: 'combobox', 'aria-expanded': open && matches.length > 0, 'aria-controls': listId, 'aria-autocomplete': 'list' as const,
    'aria-activedescendant': open && active >= 0 ? `${listId}-${active}` : undefined,
    onFocus: () => setOpen(true), onBlur: () => { setOpen(false); setActive(-1); }, onKeyDown,
  };
  return { list, inputProps, open, setOpen, setActive };
}

/** Free-text input that suggests values already in use. */
export function Combobox({ label, value, options, onChange, required = false, maxLength }: { label: string; value: string; options: string[]; onChange: (value: string) => void; required?: boolean; maxLength?: number }) {
  // Once the value matches an option, show every option so switching is one click.
  const query = options.includes(value) ? '' : value.trim().toLowerCase();
  const matches = options.filter(option => option.toLowerCase().includes(query));
  const { list, inputProps, open, setOpen, setActive } = useSuggestions(matches, option => { onChange(option); setOpen(false); });
  return <label className="block text-sm">{label}<div className="relative mt-1">
    <input {...inputProps} className={`${field} pr-9`} value={value} required={required} maxLength={maxLength} autoComplete="off"
      onChange={e => { onChange(e.target.value); setOpen(true); setActive(-1); }} />
    <button type="button" tabIndex={-1} aria-label={`Show ${label.toLowerCase()} options`} className="absolute inset-y-0 right-0 px-3 text-zinc-400 hover:text-zinc-100"
      onMouseDown={event => event.preventDefault()} onClick={() => setOpen(!open)}>▾</button>
    {list}
  </div></label>;
}

/** Tag chips with suggestions; Enter or comma adds, Backspace on empty removes the last tag. */
export function TagInput({ value, options, onChange }: { value: string[]; options: string[]; onChange: (tags: string[]) => void }) {
  const [text, setText] = useState('');
  const query = text.trim().toLowerCase();
  const matches = options.filter(option => !value.includes(option) && option.toLowerCase().includes(query));
  const add = (...tags: string[]) => {
    const next = [...new Set([...value, ...tags.map(tag => tag.trim()).filter(Boolean)])].slice(0, 30);
    if (next.length !== value.length) onChange(next);
    setText('');
  };
  const { list, inputProps, open, setOpen, setActive } = useSuggestions(matches, add);
  return <div className="block text-sm"><label htmlFor={`${inputProps['aria-controls']}-input`}>Tags</label>
    <div className="relative mt-1">
      <div className={`${field} flex flex-wrap items-center gap-1.5 focus-within:ring-2 focus-within:ring-blue-500`}>
        {value.map(tag => <span key={tag} className="flex items-center gap-1 rounded-full bg-zinc-800 py-0.5 pl-2.5 pr-1 text-xs text-zinc-200">{tag}
          <button type="button" aria-label={`Remove tag ${tag}`} className="rounded-full px-1 text-zinc-400 hover:bg-zinc-700 hover:text-zinc-100" onClick={() => onChange(value.filter(t => t !== tag))}>×</button></span>)}
        <input {...inputProps} id={`${inputProps['aria-controls']}-input`} className="min-w-32 flex-1 bg-transparent py-0.5 focus:outline-none focus-visible:outline-none" value={text} maxLength={80} autoComplete="off"
          placeholder={value.length ? '' : 'Type to search or add tags'}
          onChange={e => {
            const parts = e.target.value.split(',');
            if (parts.length > 1) { add(...parts.slice(0, -1)); setText(parts.at(-1) ?? ''); }
            else setText(e.target.value);
            setOpen(true); setActive(-1);
          }}
          onKeyDown={e => {
            inputProps.onKeyDown(e);
            if (e.defaultPrevented) return;
            if (e.key === 'Enter' && text.trim()) { e.preventDefault(); add(text); }
            else if (e.key === 'Backspace' && !text && value.length) onChange(value.slice(0, -1));
          }}
          // Commit a half-typed tag so it is not lost when the user moves on to Save.
          onBlur={() => { inputProps.onBlur(); if (text.trim()) add(text); }} />
      </div>
      {list}
    </div>
  </div>;
}

/** Small checklist kept apart from the note body; completed items stay hidden until asked for. */
export function NoteTodos({ todos, onChange, disabled = false }: { todos: NoteTodo[]; onChange: (todos: NoteTodo[]) => void; disabled?: boolean }) {
  const [text, setText] = useState('');
  const [showDone, setShowDone] = useState(false);
  const open = todos.filter(todo => !todo.done);
  const done = todos.filter(todo => todo.done);
  const add = () => {
    if (!text.trim() || todos.length >= 200) return;
    onChange([...todos, { id: crypto.randomUUID(), title: text.trim(), done: false }]);
    setText('');
  };
  const toggle = (id: string, checked: boolean) => onChange(todos.map(todo => todo.id === id ? { ...todo, done: checked } : todo));
  const row = (todo: NoteTodo) => <li key={todo.id} className="group flex items-start gap-2 text-sm">
    <input type="checkbox" className="mt-1" checked={todo.done} disabled={disabled} aria-label={`Mark ${todo.title} ${todo.done ? 'not done' : 'done'}`} onChange={e => toggle(todo.id, e.target.checked)} />
    <span className={`flex-1 ${todo.done ? 'text-zinc-500 line-through' : ''}`}>{todo.title}</span>
    <button type="button" disabled={disabled} aria-label={`Delete ${todo.title}`} className="px-1 text-zinc-500 opacity-0 hover:text-red-300 focus-visible:opacity-100 group-hover:opacity-100" onClick={() => onChange(todos.filter(t => t.id !== todo.id))}>×</button>
  </li>;
  return <fieldset className="space-y-2 rounded-lg border border-zinc-700 p-3">
    <legend className="px-1 text-sm text-zinc-400">Todos</legend>
    {open.length > 0 ? <ul className="space-y-1.5">{open.map(row)}</ul> : <p className="text-xs text-zinc-500">{done.length ? 'All done.' : 'No todos yet.'}</p>}
    <div className="flex gap-2">
      <input className={`${field} py-1.5 text-sm`} aria-label="New todo" placeholder="Add a todo" maxLength={500} value={text} disabled={disabled}
        onChange={e => setText(e.target.value)} onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); add(); } }} />
      <button type="button" className={`${button} py-1.5`} disabled={disabled || !text.trim()} onClick={add}>Add</button>
    </div>
    {done.length > 0 && <div className="space-y-1.5 border-t border-zinc-800 pt-2">
      <div className="flex gap-3 text-xs text-zinc-400">
        <button type="button" className="hover:text-zinc-100" aria-expanded={showDone} onClick={() => setShowDone(!showDone)}>{showDone ? 'Hide' : 'Show'} {done.length} completed</button>
        <button type="button" className="hover:text-red-300" disabled={disabled} onClick={() => onChange(open)}>Clear completed</button>
      </div>
      {showDone && <ul className="space-y-1.5">{done.map(row)}</ul>}
    </div>}
  </fieldset>;
}
