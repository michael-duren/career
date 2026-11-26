/*
 * App header behavior (docs/redesign/shell.md, "Desktop header" and "Mobile"). Plain JS so
 * the Astro layout and the Go-rendered Leetgrinder pages load the same file (/ui/shell.js).
 *
 * The server renders Home plus a link for every module in registry order. This script:
 * - hides links to turned-off modules and puts the rest in the user's module order,
 *   from GET /api/settings (links stay as rendered if settings cannot load);
 * - redoes that without a reload when settings change: saveSettings() in
 *   src/lib/modules.ts fires SETTINGS_EVENT on window and posts on SETTINGS_CHANNEL
 *   for other tabs;
 * - moves links that don't fit into the MORE + menu;
 * - keeps the 12-hour clock current;
 * - signs out from the MORE + menu;
 * - below 768px: fills the tab bar (Home, the first TAB_BAR_SIZE enabled modules in the
 *   user's order, More) and opens the search field from the top bar's search button.
 */

/** Window event and BroadcastChannel name carrying the saved settings (keep in sync with src/lib/modules.ts). */
export const SETTINGS_EVENT = 'career:settings';
export const SETTINGS_CHANNEL = 'career-settings';

/**
 * The module ids to show, in the user's order: stored order first, then any id it
 * misses in rendered (registry) order, keeping only enabled ones. `effective` from
 * the API already applies dependencies; without it the stored switch decides.
 */
export function menuOrder(settings, ids) {
  const enabled = id => (settings.effective ?? settings.modules ?? {})[id] !== false;
  const order = [...new Set([...(settings.moduleOrder ?? []), ...ids])].filter(id => ids.includes(id));
  return order.filter(enabled);
}

/** Module tabs between Home and More in the phone tab bar (src/lib/modules.ts tabBarModules agrees). */
export const TAB_BAR_SIZE = 3;

/** Last settings seen, so the next page draws the tab bar right before /api/settings answers. */
export const SETTINGS_CACHE_KEY = 'career-shell-settings';

/**
 * The tab marked current: Home or a module tab owning path; else More for /more, /settings
 * and module pages not in the bar (More lists them). Undefined for pages outside every module.
 */
export function currentTab(tabs, path) {
  const owner = currentLink(tabs.filter(tab => !tab.hidden), path);
  if (owner || !currentLink(tabs, path)) return owner;
  return tabs.find(tab => tab.dataset.moduleId === 'more');
}

/** Whether a settings payload (API, cache, event or channel) has the shape the menus read. */
export function validSettings(settings) {
  return !!settings && typeof settings === 'object' && Array.isArray(settings.moduleOrder)
    && !!settings.modules && typeof settings.modules === 'object'
    && (settings.effective === undefined || (!!settings.effective && typeof settings.effective === 'object'));
}

/** How many leading links fit in `available` px, with `gap` px between neighbours. */
export function fitCount(widths, available, gap) {
  let used = -gap;
  for (let i = 0; i < widths.length; i++) {
    used += gap + widths[i];
    if (used > available) return i;
  }
  return widths.length;
}

/** The menu item owning path: Home for "/", else the module whose data-routes match whole segments. */
export function currentLink(items, path) {
  const clean = path.replace(/\/+$/, '') || '/';
  return items.find(li => (li.dataset.routes ?? '').split(' ').filter(Boolean).some(route =>
    route === '/' ? clean === '/' : clean === route || clean.startsWith(`${route}/`)));
}

/** "5:20 PM", "12:05 AM" (same output as formatTime in src/lib/format.ts). */
export function formatClock(date) {
  const hour24 = date.getHours();
  return `${hour24 % 12 || 12}:${String(date.getMinutes()).padStart(2, '0')} ${hour24 < 12 ? 'AM' : 'PM'}`;
}

function initHeader(header) {
  const menu = header.querySelector('[data-app-menu]');
  const moreItem = header.querySelector('[data-app-more]');
  const moreButton = moreItem.querySelector('button');
  const moreMenu = moreItem.querySelector('[data-app-more-menu]');
  const signOutItem = moreMenu.querySelector('[data-app-signout]').closest('li');
  const items = [...menu.querySelectorAll('li[data-module-id]')];
  const home = items.find(li => li.dataset.moduleId === 'home');
  const modules = items.filter(li => li !== home);
  const ids = modules.map(li => li.dataset.moduleId);
  let shown = items;
  const tabBar = header.querySelector('[data-app-tabbar]');
  const tabs = tabBar ? [...tabBar.querySelectorAll('[data-module-id]')] : [];
  const moduleTabs = tabs.filter(tab => ids.includes(tab.dataset.moduleId));

  // Pages built from a shared shell (a note, a company) don't know their final path at build time.
  if (!menu.querySelector('[aria-current]')) {
    const link = currentLink(items, location.pathname);
    link?.querySelector('a').setAttribute('aria-current', 'page');
  }

  function applySettings(settings) {
    if (!validSettings(settings)) return;
    const order = menuOrder(settings, ids);
    shown = [home, ...order.map(id => modules[ids.indexOf(id)])];
    for (const li of modules) li.hidden = !order.includes(li.dataset.moduleId);
    layout();
    layoutTabs(order.slice(0, TAB_BAR_SIZE));
  }

  // Show the first module tabs in the user's order between Home and More, and mark the current one.
  function layoutTabs(tabIds) {
    if (!tabBar) return;
    const more = tabs.find(tab => tab.dataset.moduleId === 'more');
    for (const tab of moduleTabs) tab.hidden = !tabIds.includes(tab.dataset.moduleId);
    for (const id of tabIds) {
      const tab = moduleTabs.find(t => t.dataset.moduleId === id);
      if (tab) tabBar.insertBefore(tab, more);
    }
    markCurrentTab();
  }

  function markCurrentTab() {
    for (const tab of tabs) tab.removeAttribute('aria-current');
    currentTab(tabs, location.pathname)?.setAttribute('aria-current', 'page');
  }

  // Put every shown link back in the bar in order, then move the ones that don't fit into MORE +.
  function layout() {
    menu.insertBefore(home, menu.firstChild);
    for (const li of shown) menu.insertBefore(li, moreItem);
    for (const li of items) if (!shown.includes(li)) menu.insertBefore(li, moreItem);
    const gap = parseFloat(getComputedStyle(menu).columnGap) || 0;
    const widths = shown.map(li => li.getBoundingClientRect().width);
    // MORE + always shows (it holds Sign out), so the links share what it leaves.
    const fit = fitCount(widths, menu.clientWidth - moreItem.getBoundingClientRect().width - gap, gap);
    const overflow = shown.slice(fit);
    for (const li of overflow) moreMenu.insertBefore(li, signOutItem);
    moreItem.classList.toggle('is-current', overflow.some(li => li.querySelector('[aria-current]')));
    header.dataset.ready = '';
  }

  function setOpen(open) {
    moreButton.setAttribute('aria-expanded', String(open));
    moreMenu.hidden = !open;
  }
  moreButton.addEventListener('click', () => setOpen(moreMenu.hidden));
  document.addEventListener('keydown', event => {
    if (event.key !== 'Escape' || moreMenu.hidden) return;
    const inside = moreItem.contains(document.activeElement);
    setOpen(false);
    if (inside) { event.preventDefault(); moreButton.focus(); }
  });
  // Close when focus lands elsewhere (Tab out). Not on focusout: a tap blurs the button with no
  // relatedTarget before the link's click, and closing then would swallow the click.
  document.addEventListener('focusin', event => { if (!moreItem.contains(event.target)) setOpen(false); });
  document.addEventListener('pointerdown', event => { if (!moreItem.contains(event.target)) setOpen(false); });

  moreMenu.querySelector('[data-app-signout]').addEventListener('click', async () => {
    try {
      localStorage.removeItem('auth_token');
      localStorage.removeItem('auth_username');
      localStorage.removeItem(SETTINGS_CACHE_KEY);
    } catch {
      // Storage blocked: the session cookie is what the logout request clears.
    }
    const response = await fetch('/api/auth/logout', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' }).catch(error => console.warn('Sign out request failed.', error));
    if (response && !response.ok) console.warn(`Sign out request failed: HTTP ${response.status}`);
    window.location.href = '/login';
  });

  // Phone top bar: the search button shows the search field under the bar and focuses it.
  const searchToggle = header.querySelector('[data-app-search-toggle]');
  const searchInput = header.querySelector('#site-search');
  function setSearching(open) {
    header.classList.toggle('is-searching', open);
    searchToggle?.setAttribute('aria-expanded', String(open));
    if (open) searchInput?.focus();
  }
  searchToggle?.addEventListener('click', () => setSearching(!header.classList.contains('is-searching')));
  document.addEventListener('keydown', event => {
    if (event.key !== 'Escape' || !header.classList.contains('is-searching')) return;
    setSearching(false);
    searchToggle?.focus();
  });

  new ResizeObserver(() => layout()).observe(header);
  document.fonts?.ready.then(layout);

  function remember(settings) {
    if (!validSettings(settings)) {
      console.warn('Ignored module settings with an unexpected shape.', settings);
      return;
    }
    applySettings(settings);
    try {
      localStorage.setItem(SETTINGS_CACHE_KEY, JSON.stringify({ modules: settings.modules, moduleOrder: settings.moduleOrder, effective: settings.effective }));
    } catch {
      // Storage blocked: the tab bar waits for /api/settings on every page.
    }
  }
  let cached = null;
  try {
    cached = JSON.parse(localStorage.getItem(SETTINGS_CACHE_KEY) ?? 'null');
  } catch {
    // Unreadable or blocked cache: keep the rendered tabs until settings load.
  }
  if (validSettings(cached)) {
    applySettings(cached);
  } else if (cached !== null) {
    try { localStorage.removeItem(SETTINGS_CACHE_KEY); } catch { /* blocked: nothing to clear */ }
  }
  // Mark the current tab among the tabs showing now (rendered, or from the cache).
  if (tabBar) markCurrentTab();

  fetch('/api/settings', { cache: 'no-store' })
    .then(response => response.ok ? response.json() : Promise.reject(new Error(`HTTP ${response.status}`)))
    .then(remember)
    .catch(error => console.warn('Module settings could not be loaded; the menu and tab bar keep their current links.', error));
  window.addEventListener(SETTINGS_EVENT, event => remember(event.detail));
  if (typeof BroadcastChannel === 'function') new BroadcastChannel(SETTINGS_CHANNEL).addEventListener('message', event => remember(event.data));

  const clock = header.querySelector('[data-app-clock]');
  function tick() {
    const now = new Date();
    clock.textContent = formatClock(now);
    clock.dateTime = `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`;
    setTimeout(tick, 60_000 - (now.getSeconds() * 1000 + now.getMilliseconds()));
  }
  tick();
}

if (typeof document !== 'undefined') {
  for (const header of document.querySelectorAll('[data-app-header]')) initHeader(header);
}
