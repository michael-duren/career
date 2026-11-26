// The four Architect themes. The palette itself lives in public/ui/tokens.css; this
// module owns the theme names, the default, and where the chosen theme is remembered.
export const THEMES = ['dark', 'vellum', 'cobalt', 'concrete'] as const;
export type Theme = (typeof THEMES)[number];
export const DEFAULT_THEME: Theme = 'dark';

// Read by the inline script in ThemeHead.astro before first paint, so a saved theme
// never flashes the default first.
export const THEME_STORAGE_KEY = 'career-theme';

export function isTheme(value: unknown): value is Theme {
  return typeof value === 'string' && (THEMES as readonly string[]).includes(value);
}

export function resolveTheme(value: unknown): Theme {
  return isTheme(value) ? value : DEFAULT_THEME;
}

// Applies the page theme and remembers it for the next load (the saved key is global, so do
// not use this for a nested preview). Returns false when nothing was saved: an unknown value
// (ignored, so a buggy caller never overwrites the saved choice) or unavailable storage.
export function applyTheme(theme: Theme, root: HTMLElement = document.documentElement): boolean {
  if (!isTheme(theme)) {
    console.error(`applyTheme: unknown theme ${JSON.stringify(theme)}`);
    return false;
  }
  root.dataset.theme = theme;
  try {
    localStorage.setItem(THEME_STORAGE_KEY, theme);
    return true;
  } catch (error) {
    console.warn('applyTheme: theme not saved', error);
    return false;
  }
}
