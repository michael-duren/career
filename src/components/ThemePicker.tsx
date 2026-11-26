import { useState } from 'react';
import { saveSettings } from '../lib/modules';
import { THEMES, applyTheme, resolveTheme, type Theme } from '../lib/theme';

// The theme picker shared by the phone More page (small swatches) and Settings → Theme (the large
// previews from setup step 4, reference/setup/ARC-Setup-4.html). Picking applies the theme to the
// page at once, remembers it for the next load, and saves it with PATCH /api/settings.

const LABELS: Record<Theme, string> = { dark: 'Dark', vellum: 'Vellum', cobalt: 'Cobalt', concrete: 'Concrete' };

export function ThemePicker({ size = 'small', className = '' }: { size?: 'small' | 'large'; className?: string }) {
  const [theme, setTheme] = useState<Theme>(() => resolveTheme(document.documentElement.dataset.theme));
  const [error, setError] = useState('');

  async function pick(next: Theme) {
    setTheme(next);
    setError('');
    applyTheme(next);
    try {
      await saveSettings({ theme: next });
    } catch (e) {
      // The theme is applied to this page; only the saved setting failed.
      setError(e instanceof Error ? e.message : 'Theme could not be saved. Try again.');
    }
  }

  const large = size === 'large';
  return (
    <fieldset className={`m-0 min-w-0 border-0 p-0 ${className}`}>
      <legend className={large ? 'ui-sr-only' : 'mb-3 p-0 text-[11px] font-medium tracking-[0.24em]'}>THEME</legend>
      <div className={large ? 'grid grid-cols-2 gap-x-4 gap-y-6 xl:grid-cols-4 xl:gap-[26px]' : 'grid grid-cols-4 gap-2.5'}>
        {THEMES.map(t => (
          <label key={t} className={`flex cursor-pointer flex-col outline-offset-2 outline-ink has-checked:outline has-focus-visible:outline-2 ${large ? 'gap-2.5 p-2' : 'items-center gap-1.5 p-1.5'}`}>
            <input type="radio" name="theme" value={t} checked={theme === t} onChange={() => void pick(t)} className="ui-sr-only" />
            {large ? (
              <span data-theme={t} aria-hidden="true" className="relative block h-[140px] overflow-hidden border border-ink-30 bg-paper md:h-[200px]">
                <span className="absolute top-[22%] right-[40%] left-[12%] border-t-2 border-ink" />
                <span className="absolute top-[42%] right-[20%] left-[12%] border-t border-ink-60" />
                <span className="absolute top-[58%] right-[55%] left-[12%] border-t border-ink-60" />
                <span className="ui-hatch--diagonal absolute bottom-[14%] left-[12%] h-[18%] w-[30%] border border-ink" />
                <span className="absolute right-[12%] bottom-[16%] size-2.5 rounded-full bg-accent" />
              </span>
            ) : (
              <span data-theme={t} aria-hidden="true" className="relative block h-11 w-full border border-ink-30 bg-paper">
                <span className="absolute top-[40%] left-[20%] h-px w-[60%] bg-ink" />
              </span>
            )}
            <span className={large ? 'text-[17px] font-light' : 'font-mono text-[9px] tracking-[0.1em]'}>{large ? LABELS[t] : LABELS[t].toUpperCase()}</span>
          </label>
        ))}
      </div>
      {error && <p role="alert" className="mt-3 text-sm text-accent">{error}</p>}
    </fieldset>
  );
}
