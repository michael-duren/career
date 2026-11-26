/*
 * Header search field and results panel (docs/redesign/shell.md, "Global search"). Plain JS so the
 * Astro layout (src/components/SiteSearch.astro) and the Go-rendered Leetgrinder pages
 * (internal/shell) share it; both render the same .site-search markup.
 */

function isEditable(target) {
  return target instanceof HTMLElement && (target.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(target.tagName));
}

function initSearch(root) {
  const input = root.querySelector('#site-search');
  const panel = document.querySelector('#search-panel');
  const status = document.querySelector('#search-status');
  const results = document.querySelector('#search-results');
  const kind = document.querySelector('#search-kind');
  kind.addEventListener('change', render);
  let timer;
  let request;
  let version = 0;
  const clear = document.querySelector('#search-clear');
  const labels = { run: 'Audio thoughts', note: 'Notes', document: 'Documents', company: 'Companies', connection: 'Connections', book: 'Books', personal: 'Personal journal', week: 'Work journal' };
  function close() { panel.hidden = true; input.setAttribute('aria-expanded', 'false'); request?.abort(); clearTimeout(timer); }
  clear.addEventListener('click', () => { input.value = ''; clear.hidden = true; input.focus(); void render(); });
  // "/" (outside inputs, selects and editable content) and Cmd/Ctrl+K jump to the search field.
  document.addEventListener('keydown', e => {
    if (e.defaultPrevented || e.isComposing) return;
    const slash = e.key === '/' && !e.metaKey && !e.ctrlKey && !e.altKey && !isEditable(e.target);
    if (slash || ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k')) { e.preventDefault(); input.focus(); input.select(); if (panel.hidden) void render(); }
  });

  async function render() {
    request?.abort();
    const current = ++version;
    const query = input.value.trim();
    clear.hidden = !input.value;
    input.setAttribute('aria-expanded', 'true');
    results.replaceChildren();
    panel.hidden = false;
    if (!query) { status.textContent = 'Find a note, book, company, or journal entry.'; return; }
    status.textContent = 'Searching…';
    request = new AbortController();
    let matches;
    try {
      const response = await fetch(`/api/search?q=${encodeURIComponent(query)}&kind=${encodeURIComponent(kind.value)}`, { cache: 'no-store', signal: request.signal });
      if (!response.ok) throw new Error(response.status === 401 ? 'Sign in again to search.' : 'Search could not load. Type again to retry.');
      matches = (await response.json()).results;
    } catch (error) { if (current !== version || request?.signal.aborted) return; status.textContent = error instanceof Error ? error.message : 'Search could not load. Try again.'; return; }
    if (current !== version || query !== input.value.trim() || !root.contains(document.activeElement) || panel.hidden) return;
    results.replaceChildren();
    status.textContent = matches.length ? `${matches.length} ${matches.length === 1 ? 'result' : 'results'}${matches.length === 50 ? ' (top matches)' : ''}` : 'No results. Try a different word.';
    for (const item of matches) {
      const li = document.createElement('li');
      const link = document.createElement('a');
      const encoded = item.id.split('/').map(encodeURIComponent).join('/');
      link.href = item.kind === 'run' ? `/audio-thoughts?id=${encodeURIComponent(item.id)}` : item.kind === 'note' ? `/notes/${encoded}` : item.kind === 'document' ? `/documents/${encoded}` : item.kind === 'company' ? `/companies/${encoded}` : item.kind === 'connection' ? `/connections?id=${encodeURIComponent(item.id)}` : item.kind === 'book' ? `/books?id=${encodeURIComponent(item.id)}` : item.kind === 'personal' ? `/journal?tab=personal&id=${encodeURIComponent(item.id)}` : `/journal?id=${encodeURIComponent(item.id)}`;
      link.className = 'search-result';
      const title = document.createElement('div');
      title.className = 'search-result-title';
      title.textContent = item.title;
      const badge = document.createElement('span');
      badge.className = 'search-result-kind';
      badge.textContent = labels[item.kind] || item.kind;
      const heading = document.createElement('div');
      heading.className = 'search-result-heading';
      heading.append(title, badge);
      const snippet = document.createElement('p');
      snippet.className = 'search-result-snippet';
      const start = Math.max(0, (item.body ?? '').toLowerCase().indexOf(query.toLowerCase()) - 40);
      snippet.textContent = (start ? '…' : '') + (item.body ?? '').slice(start, start + 160).replace(/[#*`\n]/g, ' ') + ((item.body ?? '').length > start + 160 ? '…' : '');
      link.append(heading, snippet);
      li.append(link);
      results.append(li);
    }
  }
  input.addEventListener('input', () => { clearTimeout(timer); request?.abort(); ++version; clear.hidden = !input.value; results.replaceChildren(); status.textContent = input.value.trim() ? 'Searching…' : 'Find a note, book, company, or journal entry.'; panel.hidden = false; input.setAttribute('aria-expanded', 'true'); timer = setTimeout(render, 180); });
  input.addEventListener('focus', () => { if (panel.hidden) void render(); });
  input.addEventListener('click', () => { if (panel.hidden) void render(); });
  root.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') { e.preventDefault(); input.focus(); close(); }
    const links = [...results.querySelectorAll('a')];
    const index = links.indexOf(document.activeElement);
    if (e.key === 'ArrowDown') { e.preventDefault(); links[Math.min(index + 1, links.length - 1)]?.focus(); }
    if (e.key === 'ArrowUp') { e.preventDefault(); if (index <= 0) input.focus(); else links[index - 1].focus(); }
    if (e.key === 'Enter' && document.activeElement === input) { e.preventDefault(); links[0]?.click(); }
  });
  document.addEventListener('pointerdown', e => { if (!root.contains(e.target)) close(); });
  root.addEventListener('focusout', e => { if (!root.contains(e.relatedTarget)) close(); });
}

const searchRoot = document.querySelector('.site-search');
if (searchRoot) initSearch(searchRoot);
