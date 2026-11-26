import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, readdirSync } from 'node:fs';
import { DEFAULT_THEME, THEME_STORAGE_KEY, THEMES } from '../src/lib/theme.ts';

// Go-rendered pages (Leetgrinder) cannot use the Vite-bundled @fontsource files, so
// public/ui/fonts/ carries copies and public/ui/fonts.css declares them. They must stay
// identical to the packages Astro bundles, or the two halves of the app would drift.
const root = new URL('../', import.meta.url);
const read = (path: string) => readFileSync(new URL(path, root));
const COPIES: Record<string, string> = {
  'inter-tight-latin-wght-normal.woff2': 'node_modules/@fontsource-variable/inter-tight/files/',
  'ibm-plex-mono-latin-400-normal.woff2': 'node_modules/@fontsource/ibm-plex-mono/files/',
  'ibm-plex-mono-latin-500-normal.woff2': 'node_modules/@fontsource/ibm-plex-mono/files/',
  'architects-daughter-latin-400-normal.woff2': 'node_modules/@fontsource/architects-daughter/files/',
};

test('vendored fonts match the @fontsource packages byte for byte', () => {
  for (const [file, dir] of Object.entries(COPIES)) {
    assert.ok(read(`public/ui/fonts/${file}`).equals(read(dir + file)), `${file} differs from ${dir}${file}; copy it again`);
  }
});

test('fonts.css declares every vendored font and nothing else', () => {
  const css = read('public/ui/fonts.css').toString();
  const urls = [...css.matchAll(/url\(\/ui\/fonts\/([^)]+)\)/g)].map(m => m[1]).sort();
  assert.deepEqual(urls, Object.keys(COPIES).sort());
  const woff2 = readdirSync(new URL('public/ui/fonts/', root)).filter(f => f.endsWith('.woff2')).sort();
  assert.deepEqual(woff2, Object.keys(COPIES).sort());
  for (const family of ['Inter Tight Variable', 'IBM Plex Mono', 'Architects Daughter']) {
    assert.match(css, new RegExp(`font-family: "${family}"`));
  }
});

// The Leetgrinder layout's inline theme script mirrors ThemeHead.astro (src/lib/theme.ts).
test('the Go layout theme script uses the same themes, default and storage key', () => {
  const templ = read('internal/leetgrinder/views.templ').toString();
  const script = templ.match(/<script id="theme-init">([\s\S]*?)<\/script>/);
  assert.ok(script, 'views.templ has no theme-init script');
  assert.match(script[1], new RegExp(`\\[${THEMES.map(t => `"${t}"`).join(', ')}\\]`));
  assert.match(script[1], new RegExp(`"${THEME_STORAGE_KEY}"`));
  assert.match(script[1], new RegExp(`: "${DEFAULT_THEME}"`));
  // It must run before the first stylesheet so a saved theme never flashes the default.
  assert.ok(templ.indexOf('<script id="theme-init">') < templ.indexOf('<link rel="stylesheet"'), 'theme-init must come before the stylesheets');
});
