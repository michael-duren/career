/*
 * Global search (docs/redesign/shell.md, "Global search"). Plain JS so the Astro layout
 * (src/components/SiteSearch.astro) and the Go-rendered Leetgrinder pages (internal/shell) share it;
 * both render the same .site-search markup.
 *
 * Desktop: typing in the header field opens a results panel under it. Phone (< 768px): the top bar's
 * search button (public/ui/shell.js) turns the same markup into a full-screen search page. Results
 * from GET /api/search (turned-off modules are already left out) are grouped by module with a count
 * per group, each a title and one line of context. With an empty field the panel shows recent searches.
 */

/** localStorage key for recent searches (src/lib/auth.ts signOut clears it; keep in sync). */
export const RECENT_KEY = 'career-recent-searches';
/** Recent searches kept and shown as chips. */
export const RECENT_LIMIT = 5;

/** Group heading per module id; '' holds documents, which belong to no module. */
export const GROUP_LABELS = { timeline: 'Goals', schedule: 'Schedule', leetgrinder: 'Problems', companies: 'Companies', people: 'People', bookshelf: 'Books', journal: 'Journal', notes: 'Notes', audio: 'Audio thoughts', '': 'Documents' };
/** Registry order of the modules search covers (src/lib/modules.ts; tests/site-search.test.ts checks it), for modules the saved order misses. */
export const REGISTRY_ORDER = ['timeline', 'schedule', 'leetgrinder', 'companies', 'people', 'bookshelf', 'journal', 'notes', 'audio'];

/** Where a result opens. */
export function resultHref(item) {
  const id = encodeURIComponent(item.id);
  const path = item.id.split('/').map(encodeURIComponent).join('/');
  switch (item.kind) {
    case 'goal': return `/timeline?goal=${id}`;
    case 'session': return '/weekly-scheduler';
    case 'problem': return `/leetgrinder/problem/${id}`;
    case 'run': return `/audio-thoughts?id=${id}`;
    case 'note': return `/notes/${path}`;
    case 'document': return `/documents/${path}`;
    case 'company': return `/companies/${path}`;
    case 'connection': return `/connections?id=${id}`;
    case 'book': return `/books?id=${id}`;
    case 'personal': return `/journal?tab=personal&id=${id}`;
    default: return `/journal?id=${id}`;
  }
}

/** The excerpt as one plain line: Markdown marks, list bullets, checkboxes and line breaks go. */
function plain(text) {
  return (text ?? '').replace(/\[[ xX]\]/g, ' ').replace(/[#*`>_]+/g, ' ').replace(/(^|\s)[-+](?=\s)/g, ' ').replace(/\s+/g, ' ').trim();
}

/**
 * One line under a result's title: its metadata (status, dates, role…) when the title matched,
 * else the text around the match.
 */
export function contextLine(item) {
  const excerpt = plain(item.excerpt);
  if ((item.titleMatch && item.context) || !excerpt) return item.context ?? '';
  return excerpt;
}

/**
 * Results grouped by module in the user's module order (then registry order, documents last), each
 * with its total matches: the API returns at most a few of each kind but counts them all in `totals`.
 */
export function groupResults(results, totals = {}, moduleOrder = []) {
  const order = [...new Set([...moduleOrder, ...REGISTRY_ORDER])];
  const groups = new Map();
  for (const item of results) {
    const id = item.module ?? '';
    if (!groups.has(id)) groups.set(id, { id, label: GROUP_LABELS[id] ?? id, results: [], kinds: new Set() });
    groups.get(id).results.push(item);
    groups.get(id).kinds.add(item.kind);
  }
  const rank = id => (id === '' ? order.length + 1 : order.includes(id) ? order.indexOf(id) : order.length);
  return [...groups.values()]
    .sort((a, b) => rank(a.id) - rank(b.id))
    .map(({ kinds, ...group }) => ({ ...group, total: Math.max(group.results.length, [...kinds].reduce((sum, kind) => sum + (totals[kind] ?? 0), 0)) }));
}

/** recent with query first, without repeats (ignoring case), at most RECENT_LIMIT. */
export function addRecent(recent, query) {
  const term = query.trim();
  if (!term) return recent;
  return [term, ...recent.filter(item => item.toLowerCase() !== term.toLowerCase())].slice(0, RECENT_LIMIT);
}

function readRecent() {
  try {
    const saved = JSON.parse(localStorage.getItem(RECENT_KEY) ?? '[]');
    return Array.isArray(saved) ? saved.filter(item => typeof item === 'string' && item.trim()).slice(0, RECENT_LIMIT) : [];
  } catch {
    return []; // Unreadable or blocked storage: no recent searches.
  }
}

function saveRecent(query) {
  try { localStorage.setItem(RECENT_KEY, JSON.stringify(addRecent(readRecent(), query))); } catch { /* blocked: not remembered */ }
}

/** The user's module order from the shell's settings cache (SETTINGS_CACHE_KEY in public/ui/shell.js). */
function savedModuleOrder() {
  try {
    const order = JSON.parse(localStorage.getItem('career-shell-settings') ?? 'null')?.moduleOrder;
    return Array.isArray(order) ? order : [];
  } catch {
    return [];
  }
}

function isEditable(target) {
  return target instanceof HTMLElement && (target.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(target.tagName));
}

function initSearch(root) {
  const input = root.querySelector('#site-search');
  const panel = root.querySelector('#search-panel');
  const status = root.querySelector('#search-status');
  const results = root.querySelector('#search-results');
  const recentBox = root.querySelector('#search-recent');
  const recentList = root.querySelector('#search-recent-list');
  const clear = root.querySelector('#search-clear');
  const EMPTY = 'Find a goal, note, problem, company or person.';
  let timer;
  let request;
  let version = 0;
  // On phones the search is a full-screen page while the top bar's search button has it open
  // (shell.js marks the header is-searching); it stays open until Cancel, the button or Escape.
  const fullScreen = () => !!root.closest('.is-searching') && matchMedia('(max-width: 767px)').matches;

  function close() {
    if (fullScreen()) return;
    panel.hidden = true; input.setAttribute('aria-expanded', 'false'); request?.abort(); clearTimeout(timer);
  }
  function open() { panel.hidden = false; input.setAttribute('aria-expanded', 'true'); }
  clear.addEventListener('click', () => { input.value = ''; clear.hidden = true; input.focus(); void render(); });
  // "/" (outside inputs, selects and editable content) and Cmd/Ctrl+K jump to the search field.
  document.addEventListener('keydown', e => {
    if (e.defaultPrevented || e.isComposing) return;
    const slash = e.key === '/' && !e.metaKey && !e.ctrlKey && !e.altKey && !isEditable(e.target);
    if (slash || ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k')) { e.preventDefault(); input.focus(); input.select(); if (panel.hidden) void render(); }
  });

  function showRecent(show) {
    const recent = show ? readRecent() : [];
    recentList.replaceChildren(...recent.map(term => {
      const chip = document.createElement('button');
      chip.type = 'button';
      chip.className = 'search-recent-chip';
      chip.textContent = term;
      chip.addEventListener('click', () => { input.value = term; input.focus(); void render(); });
      return chip;
    }));
    recentBox.hidden = recent.length === 0;
  }

  function resultLink(item, query) {
    const link = document.createElement('a');
    link.href = resultHref(item);
    link.className = 'search-result';
    const title = document.createElement('span');
    title.className = 'search-result-title';
    title.textContent = item.title;
    link.append(title);
    const line = contextLine(item);
    if (line) {
      const sub = document.createElement('span');
      sub.className = 'search-result-context';
      sub.textContent = line;
      link.append(sub);
    }
    link.addEventListener('click', () => saveRecent(query));
    return link;
  }

  function group(item, i, query) {
    const section = document.createElement('section');
    section.className = 'search-group';
    const head = document.createElement('div');
    head.className = 'search-group-head';
    const name = document.createElement('h2');
    name.id = `search-group-${i}`;
    name.textContent = item.label;
    const count = document.createElement('span');
    count.className = 'search-group-count';
    count.textContent = item.total > item.results.length ? `${item.results.length} of ${item.total}` : String(item.total);
    head.append(name, count);
    const list = document.createElement('ul');
    list.setAttribute('aria-labelledby', name.id);
    list.append(...item.results.map(result => {
      const li = document.createElement('li');
      li.append(resultLink(result, query));
      return li;
    }));
    section.append(head, list);
    return section;
  }

  async function render() {
    request?.abort();
    const current = ++version;
    const query = input.value.trim();
    clear.hidden = !input.value;
    results.replaceChildren();
    open();
    showRecent(!query);
    if (!query) { status.textContent = EMPTY; return; }
    if (query.length < 2) { status.textContent = 'Keep typing…'; return; }
    status.textContent = 'Searching…';
    request = new AbortController();
    let data;
    try {
      const response = await fetch(`/api/search?q=${encodeURIComponent(query)}`, { cache: 'no-store', signal: request.signal });
      if (!response.ok) throw new Error(response.status === 401 ? 'Sign in again to search.' : 'Search could not load. Type again to retry.');
      data = await response.json();
    } catch (error) { if (current !== version || request?.signal.aborted) return; status.textContent = error instanceof Error ? error.message : 'Search could not load. Try again.'; return; }
    if (current !== version || query !== input.value.trim() || panel.hidden) return;
    const groups = groupResults(data.results ?? [], data.totals ?? {}, savedModuleOrder());
    const total = groups.reduce((sum, item) => sum + item.total, 0);
    status.textContent = total ? `${total} ${total === 1 ? 'result' : 'results'} for “${query}”` : `No results for “${query}”. Try a different word.`;
    results.replaceChildren(...groups.map((item, i) => group(item, i, query)));
  }
  input.addEventListener('input', () => {
    clearTimeout(timer); request?.abort(); ++version;
    clear.hidden = !input.value; results.replaceChildren(); open();
    const query = input.value.trim();
    showRecent(!query);
    status.textContent = !query ? EMPTY : query.length < 2 ? 'Keep typing…' : 'Searching…';
    timer = setTimeout(render, 180);
  });
  input.addEventListener('focus', () => { if (panel.hidden) void render(); });
  input.addEventListener('click', () => { if (panel.hidden) void render(); });
  root.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') { e.preventDefault(); input.focus(); close(); return; }
    // Arrow keys move through the results, or the recent-search chips while the field is empty.
    const items = [...panel.querySelectorAll('.search-result, .search-recent-chip')];
    const index = items.indexOf(document.activeElement);
    if (e.key === 'ArrowDown') { e.preventDefault(); items[Math.min(index + 1, items.length - 1)]?.focus(); }
    if (e.key === 'ArrowUp') { e.preventDefault(); if (index <= 0) input.focus(); else items[index - 1].focus(); }
    if (e.key === 'Enter' && document.activeElement === input) { e.preventDefault(); results.querySelector('a')?.click(); }
  });
  document.addEventListener('pointerdown', e => { if (!root.contains(e.target)) close(); });
  root.addEventListener('focusout', e => { if (!root.contains(e.relatedTarget)) close(); });
}

if (typeof document !== 'undefined') {
  const searchRoot = document.querySelector('.site-search');
  if (searchRoot) initSearch(searchRoot);
}
