// The four Architect themes. The palette itself lives in src/styles/global.css; this
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

// Applies a theme to the page and remembers it for the next load. Returns false when it
// could not be remembered (private mode, blocked site data), so a picker can say so.
// Unknown values (e.g. an untyped <select> value) apply and store the default instead.
export function applyTheme(value: Theme, root: HTMLElement = document.documentElement): boolean {
  const theme = resolveTheme(value);
  root.dataset.theme = theme;
  try {
    localStorage.setItem(THEME_STORAGE_KEY, theme);
    return true;
  } catch {
    return false;
  }
}
