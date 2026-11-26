import { useEffect, useState } from 'react';
import { AuthService } from '../lib/auth';
import { DEFAULT_SETTINGS, MODULE_IDS, SETTINGS_CHANNEL, SETTINGS_EVENT, enabledModules, loadSettings, moreModules, saveSettings, settingsSchema, type Settings } from '../lib/modules';
import { THEMES, applyTheme, resolveTheme, type Theme } from '../lib/theme';

// The phone More page (docs/redesign/shell.md, reference/mobile/M-More.html): every enabled
// module the tab bar has no room for, with its count, then Settings, the theme picker and Sign out.

const THEME_LABELS: Record<Theme, string> = { dark: 'DARK', vellum: 'VELLUM', cobalt: 'COBALT', concrete: 'CONCRETE' };

const row = 'grid min-h-[60px] grid-cols-[34px_minmax(0,1fr)_auto_14px] items-center gap-3.5 border-b border-ink-14 text-ink no-underline hover:text-ink-hover';
const mark = 'flex size-[26px] items-center justify-center border border-ink-50 font-mono text-[11px]';
const heading = 'm-0 text-[11px] font-medium tracking-[0.24em]';

export function MorePage() {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [error, setError] = useState('');
  const [counts, setCounts] = useState<Record<string, number>>({});
  const [theme, setTheme] = useState<Theme>(() => resolveTheme(document.documentElement.dataset.theme));
  const [themeError, setThemeError] = useState('');
  const [signOutError, setSignOutError] = useState('');
  const [countsFailed, setCountsFailed] = useState(false);

  useEffect(() => {
    loadSettings().then(setSettings, (e: unknown) => {
      setError(e instanceof Error ? e.message : 'Settings could not be loaded. Try again.');
      setSettings(DEFAULT_SETTINGS);
    });
    fetch('/api/counts', { cache: 'no-store' })
      .then(response => response.ok ? response.json() : Promise.reject(new Error(`HTTP ${response.status}`)))
      .then((body: { counts?: Record<string, number> }) => setCounts(body.counts ?? {}))
      .catch((e: unknown) => { console.warn('Module counts could not be loaded.', e); setCountsFailed(true); });
    // Follow module changes saved elsewhere (saveSettings announces them), like the tab bar does.
    const update = (value: unknown) => {
      const parsed = settingsSchema.safeParse(value);
      if (parsed.success) setSettings(parsed.data);
    };
    const onEvent = (event: Event) => update((event as CustomEvent).detail);
    window.addEventListener(SETTINGS_EVENT, onEvent);
    const channel = typeof BroadcastChannel === 'function' ? new BroadcastChannel(SETTINGS_CHANNEL) : null;
    channel?.addEventListener('message', event => update(event.data));
    return () => { window.removeEventListener(SETTINGS_EVENT, onEvent); channel?.close(); };
  }, []);

  async function pickTheme(next: Theme) {
    setTheme(next);
    setThemeError('');
    applyTheme(next);
    try {
      await saveSettings({ theme: next });
    } catch (e) {
      // The theme is applied to this page; only the saved setting failed.
      setThemeError(e instanceof Error ? e.message : 'Theme could not be saved. Try again.');
    }
  }

  // Same steps as the header's SIGN OUT (public/ui/shell.js), but stays put and says so if the
  // logout request fails, since the session cookie would still be valid.
  async function signOut() {
    setSignOutError('');
    try {
      AuthService.clearAuth();
      localStorage.removeItem('career-shell-settings');
    } catch {
      // Storage blocked: the session cookie is what the logout request clears.
    }
    const response = await fetch('/api/auth/logout', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' }).catch(() => null);
    if (!response?.ok && response?.status !== 401) {
      setSignOutError('Sign out failed. Check your connection and try again.');
      return;
    }
    window.location.href = '/login';
  }

  const items = settings ? moreModules(settings) : [];
  const on = settings && !error ? enabledModules(settings).length : 0;

  return (
    <div className="flex flex-col">
      <h1 className="m-0 font-extralight !text-[44px] !leading-[0.9]">More</h1>
      {error && <p role="alert" className="mt-4 text-sm text-accent">{error}</p>}
      <nav aria-label="More pages" className="mt-[18px] flex flex-col" aria-busy={!settings}>
        {items.map(m => (
          <a key={m.id} href={m.routes[0]} className={row}>
            <span aria-hidden="true" className={mark}>{m.label[0]}</span>
            <span className="text-lg font-light">{m.label}</span>
            <span className="font-mono text-[11px] text-ink-60">{counts[m.id] ?? (countsFailed ? '–' : '')}</span>
            <span aria-hidden="true" className="text-sm text-ink-50">→</span>
          </a>
        ))}
        {settings && !items.length && <p className="py-4 text-sm text-ink-60">Every module you use is in the tab bar.</p>}
      </nav>

      <h2 className={`${heading} mt-[30px] mb-1`}>SETTINGS</h2>
      <a href="/settings/modules" className={row}>
        <span aria-hidden="true" className={mark}>
          <span className="relative h-2 w-3.5 rounded border border-ink"><span className="absolute top-px right-px size-1 rounded-full bg-ink" /></span>
        </span>
        <span className="text-lg font-light">Modules</span>
        <span className="font-mono text-[11px] text-ink-60">{settings && !error ? `${on} of ${MODULE_IDS.length} on` : ''}</span>
        <span aria-hidden="true" className="text-sm text-ink-50">→</span>
      </a>

      <fieldset className="m-0 mt-[26px] min-w-0 border-0 p-0">
        <legend className={`${heading} mb-3 p-0`}>THEME</legend>
        <div className="grid grid-cols-4 gap-2.5">
          {THEMES.map(t => (
            <label key={t} className="flex cursor-pointer flex-col items-center gap-1.5 p-1.5 outline-offset-2 outline-ink has-checked:outline has-focus-visible:outline-2">
              <input type="radio" name="theme" value={t} checked={theme === t} onChange={() => void pickTheme(t)} className="ui-sr-only" />
              <span data-theme={t} aria-hidden="true" className="relative block h-11 w-full border border-ink-30 bg-paper">
                <span className="absolute top-[40%] left-[20%] h-px w-[60%] bg-ink" />
              </span>
              <span className="font-mono text-[9px] tracking-[0.1em]">{THEME_LABELS[t]}</span>
            </label>
          ))}
        </div>
        {themeError && <p role="alert" className="mt-3 text-sm text-accent">{themeError}</p>}
      </fieldset>

      {signOutError && <p role="alert" className="mt-4 text-sm text-accent">{signOutError}</p>}
      <button type="button" onClick={() => void signOut()} className="mt-[26px] flex min-h-11 cursor-pointer items-center self-start border-0 bg-transparent p-0 font-mono text-[11px] tracking-[0.14em] text-ink-60 hover:text-ink">SIGN OUT</button>
    </div>
  );
}
