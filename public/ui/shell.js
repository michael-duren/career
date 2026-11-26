/*
 * App header behavior (docs/redesign/shell.md, "Desktop header"). Plain JS so the Astro
 * layout and the Go-rendered Leetgrinder pages load the same file (/ui/shell.js).
 *
 * The server renders Home plus a link for every module in registry order. This script:
 * - hides links to turned-off modules and puts the rest in the user's module order,
 *   from GET /api/settings (links stay as rendered if settings cannot load);
 * - redoes that without a reload when settings change: saveSettings() in
 *   src/lib/modules.ts fires SETTINGS_EVENT on window and posts on SETTINGS_CHANNEL
 *   for other tabs;
 * - moves links that don't fit into the MORE + menu;
 * - keeps the 12-hour clock current;
 * - signs out from the MORE + menu.
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

  // Pages built from a shared shell (a note, a company) don't know their final path at build time.
  if (!menu.querySelector('[aria-current]')) {
    const link = currentLink(items, location.pathname);
    link?.querySelector('a').setAttribute('aria-current', 'page');
  }

  function applySettings(settings) {
    const order = menuOrder(settings, ids);
    shown = [home, ...order.map(id => modules[ids.indexOf(id)])];
    for (const li of modules) li.hidden = !order.includes(li.dataset.moduleId);
    layout();
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
    localStorage.removeItem('auth_token');
    localStorage.removeItem('auth_username');
    await fetch('/api/auth/logout', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' }).catch(() => undefined);
    window.location.href = '/login';
  });

  new ResizeObserver(() => layout()).observe(header);
  document.fonts?.ready.then(layout);

  fetch('/api/settings', { cache: 'no-store' })
    .then(response => response.ok ? response.json() : Promise.reject(new Error(`HTTP ${response.status}`)))
    .then(applySettings)
    .catch(error => console.warn('Module settings could not be loaded; showing every link.', error));
  window.addEventListener(SETTINGS_EVENT, event => applySettings(event.detail));
  if (typeof BroadcastChannel === 'function') new BroadcastChannel(SETTINGS_CHANNEL).addEventListener('message', event => applySettings(event.data));

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
