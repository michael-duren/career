import { useId, useRef, useState, type KeyboardEvent } from 'react';
import type { NoteTodo } from '../lib/workspace';
import { Checkbox, buttonClassName } from './ui';

const field = 'w-full border border-solid border-ink-30 bg-paper px-3 py-2 text-ink placeholder:text-ink-60 focus:border-ink';
const button = buttonClassName();

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
  const list = open && matches.length > 0 && <ul id={listId} role="listbox" className="absolute z-20 mt-1 max-h-60 w-full overflow-y-auto border border-solid border-ink-30 bg-paper py-1">
    {matches.map((option, index) => <li key={option} id={`${listId}-${index}`} role="option" aria-selected={index === active}
      className={`cursor-pointer px-3 py-2 text-sm ${index === active ? 'bg-ink text-paper' : 'text-ink hover:bg-ink-05'}`}
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
  const inputId = `${inputProps['aria-controls']}-input`;
  // A separate label keeps the toggle button and options out of the input's accessible name.
  return <div className="block text-sm"><label htmlFor={inputId}>{label}</label><div className="relative mt-1">
    <input {...inputProps} id={inputId} className={`${field} pr-9`} value={value} required={required} maxLength={maxLength} autoComplete="off"
      onChange={e => { onChange(e.target.value); setOpen(true); setActive(-1); }} />
    <button type="button" tabIndex={-1} aria-label={`Show ${label.toLowerCase()} options`} className="absolute inset-y-0 right-0 border-0 bg-transparent px-3 text-ink-60 hover:text-ink"
      onMouseDown={event => event.preventDefault()} onClick={() => setOpen(!open)}>▾</button>
    {list}
  </div></div>;
}

/** Tag chips with suggestions; Enter or comma adds, Backspace on empty removes the last tag. */
export function TagInput({ value, options, onChange }: { value: string[]; options: string[]; onChange: (tags: string[]) => void }) {
  const [text, setText] = useState('');
  const query = text.trim().toLowerCase();
  const matches = options.filter(option => !value.includes(option) && option.toLowerCase().includes(query));
  const full = value.length >= 30;
  const add = (...tags: string[]) => {
    // Keep the typed text when the API limit is reached so nothing vanishes silently.
    if (full) return;
    const next = [...new Set([...value, ...tags.map(tag => tag.trim()).filter(Boolean)])].slice(0, 30);
    if (next.length !== value.length) onChange(next);
    setText('');
  };
  const { list, inputProps, open, setOpen, setActive } = useSuggestions(matches, add);
  return <div className="block text-sm"><label htmlFor={`${inputProps['aria-controls']}-input`}>Tags</label>
    <div className="relative mt-1">
      <div className={`${field} flex flex-wrap items-center gap-1.5 focus-within:border-ink`}>
        {value.map(tag => <span key={tag} className="flex items-center gap-1 border border-solid border-ink-30 py-0.5 pl-2 pr-0.5 font-mono text-[10px] uppercase tracking-[0.08em]">{tag}
          <button type="button" aria-label={`Remove tag ${tag}`} className="border-0 bg-transparent px-1 text-sm text-ink-60 hover:text-ink" onClick={() => onChange(value.filter(t => t !== tag))}>×</button></span>)}
        <input {...inputProps} id={`${inputProps['aria-controls']}-input`} className="min-w-32 flex-1 border-0 bg-transparent py-0.5 text-ink focus:outline-none focus-visible:outline-none" value={text} maxLength={80} autoComplete="off"
          placeholder={value.length ? '' : 'Type to search or add tags'}
          onChange={e => {
            const parts = e.target.value.split(',');
            if (parts.length > 1 && !full) { add(...parts.slice(0, -1)); setText(parts.at(-1) ?? ''); }
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
    {full && <p className="mt-1 text-xs text-accent">30 tag limit reached. Remove a tag to add another.</p>}
  </div>;
}

/** Small checklist kept apart from the note body; completed items stay hidden until asked for. */
export function NoteTodos({ todos, onChange }: { todos: NoteTodo[]; onChange: (todos: NoteTodo[]) => void | Promise<boolean> }) {
  const [text, setText] = useState('');
  const newTodo = useRef<HTMLInputElement>(null);
  const [showDone, setShowDone] = useState(false);
  const open = todos.filter(todo => !todo.done);
  const done = todos.filter(todo => todo.done);
  const add = async () => {
    const title = text.trim();
    if (!title || todos.length >= 200) return;
    setText('');
    if (await onChange([...todos, { id: crypto.randomUUID(), title, done: false }]) === false) setText(title);
  };
  // These actions unmount the focused control, so hand focus to the new-todo input.
  const replace = (next: NoteTodo[]) => { newTodo.current?.focus(); void onChange(next); };
  const toggle = (id: string, checked: boolean) => replace(todos.map(todo => todo.id === id ? { ...todo, done: checked } : todo));
  const row = (todo: NoteTodo) => <li key={todo.id} className="group flex min-h-11 items-center gap-3 border-0 border-b border-solid border-ink-14 text-sm last:border-b-0">
    <Checkbox checked={todo.done} aria-label={`Mark ${todo.title} ${todo.done ? 'not done' : 'done'}`} onChange={e => toggle(todo.id, e.target.checked)} />
    <span className={`flex-1 ${todo.done ? 'opacity-45 line-through' : ''}`}>{todo.title}</span>
    <button type="button" aria-label={`Delete ${todo.title}`} className="min-h-11 min-w-11 border-0 bg-transparent px-1 text-base text-ink-60 hover:text-accent" onClick={() => replace(todos.filter(t => t.id !== todo.id))}>×</button>
  </li>;
  return <fieldset className="space-y-2 border border-solid border-ink-30 px-4 pb-3 pt-2">
    <legend className="px-1 font-mono text-[10px] font-medium uppercase tracking-[0.16em] text-ink-60">Todos</legend>
    {open.length > 0 ? <ul className="m-0 list-none p-0">{open.map(row)}</ul> : <p className="text-xs text-ink-60">{done.length ? 'All done.' : 'No todos yet.'}</p>}
    <div className="flex gap-2">
      <input ref={newTodo} className={`${field} h-11 text-sm`} aria-label="New todo" placeholder="Add a todo" maxLength={500} value={text}
        onChange={e => setText(e.target.value)} onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); void add(); } }} />
      <button type="button" className={button} disabled={!text.trim()} onClick={() => void add()}>Add</button>
    </div>
    {done.length > 0 && <div className="space-y-1.5 border-0 border-t border-solid border-ink-14 pt-2">
      <div className="flex gap-3 text-xs text-ink-60">
        <button type="button" className="min-h-11 border-0 bg-transparent p-0 text-ink-60 hover:text-ink" aria-expanded={showDone} onClick={() => setShowDone(!showDone)}>{showDone ? 'Hide' : 'Show'} {done.length} completed</button>
        <button type="button" className="min-h-11 border-0 bg-transparent p-0 text-ink-60 hover:text-accent" onClick={() => replace(open)}>Clear completed</button>
      </div>
      {showDone && <ul className="m-0 list-none p-0">{done.map(row)}</ul>}
    </div>}
  </fieldset>;
}
