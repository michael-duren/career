import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { runInNewContext } from 'node:vm';
import { join } from 'node:path';
import { DEFAULT_THEME, THEME_STORAGE_KEY, THEMES, applyTheme, isTheme, resolveTheme } from '../src/lib/theme.ts';

// The palette lives in public/ui/tokens.css (also linked by Go pages) and global.css imports it.
// Comments stripped so commented-out rules or braces in comments never count.
const read = (path: string) => readFileSync(new URL(path, import.meta.url), 'utf8');
const css = (read('../public/ui/tokens.css') + read('../src/styles/global.css')).replace(/\/\*[\s\S]*?\*\//g, '');

// The palette from docs/redesign/design-system.md ("Themes and tokens").
const SPEC = {
  dark: { paper: '#0C0C0B', ink: '#ECEAE3', accent: '#E0533D', 'ink-hover': '#FFFFFF' },
  vellum: { paper: '#F3F1EA', ink: '#1A1A17', accent: '#C2361E', 'ink-hover': '#000000' },
  cobalt: { paper: '#F7F7F4', ink: '#0047AB', accent: '#D6402A', 'ink-hover': '#002F73' },
  concrete: { paper: '#2C2D2F', ink: '#EFEDE6', accent: '#F2C230', 'ink-hover': '#FFFFFF' },
} as const;

// Per-theme label alpha, documented in docs/redesign/design-system.md.
const LABEL_ALPHA = { dark: 0.6, vellum: 0.64, cobalt: 0.78, concrete: 0.6 } as const;

function themeBlock(theme: string): Record<string, string> {
  const match = css.match(new RegExp(`\\[data-theme="${theme}"\\]\\s*\\{([^}]*)\\}`));
  assert.ok(match, `no [data-theme="${theme}"] block in public/ui/tokens.css`);
  return Object.fromEntries([...match[1].matchAll(/--([\w-]+):\s*([^;]+);/g)].map(([, name, value]) => [name, value.trim()]));
}

// Declarations of the CSS block whose selector list is exactly `selector` (whitespace-insensitive).
function blockFor(selector: string): Record<string, string> | undefined {
  const wanted = selector.replace(/\s+/g, '');
  for (const [, sel, body] of css.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    if (sel.replace(/^[\s\S]*;/, '').replace(/\s+/g, '') !== wanted) continue;
    return Object.fromEntries([...body.matchAll(/--([\w-]+):\s*([^;]+);/g)].map(([, name, value]) => [name, value.trim()]));
  }
  return undefined;
}

const rgb = (hex: string) => [1, 3, 5].map(i => parseInt(hex.slice(i, i + 2), 16));
function luminance([r, g, b]: number[]) {
  const [R, G, B] = [r, g, b].map(v => { const c = v / 255; return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4; });
  return 0.2126 * R + 0.7152 * G + 0.0722 * B;
}
function contrast(a: number[], b: number[]) {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}
// What the browser paints for rgba(ink, alpha) on a solid paper background.
const over = (ink: number[], paper: number[], alpha: number) => ink.map((v, i) => Math.round(alpha * v + (1 - alpha) * paper[i]));

test('theme names and fallback', () => {
  assert.deepEqual([...THEMES], ['dark', 'vellum', 'cobalt', 'concrete']);
  assert.equal(DEFAULT_THEME, 'dark');
  assert.ok(isTheme('cobalt'));
  assert.ok(!isTheme('light'));
  assert.ok(!isTheme(null));
  assert.equal(resolveTheme('vellum'), 'vellum');
  assert.equal(resolveTheme('Vellum'), 'dark');
  assert.equal(resolveTheme(undefined), 'dark');
});

test('applyTheme sets data-theme and remembers it, even without storage', () => {
  const saved = new Map<string, string>();
  const root = { dataset: {} as Record<string, string> } as unknown as HTMLElement;
  const original = Object.getOwnPropertyDescriptor(globalThis, 'localStorage');
  try {
    Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: { setItem: (key: string, value: string) => saved.set(key, value) } });
    assert.equal(applyTheme('concrete', root), true);
    assert.equal(root.dataset.theme, 'concrete');
    assert.equal(saved.get(THEME_STORAGE_KEY), 'concrete');

    Object.defineProperty(globalThis, 'localStorage', { configurable: true, get() { throw new Error('blocked'); } });
    assert.equal(applyTheme('vellum', root), false);
    assert.equal(root.dataset.theme, 'vellum');

    // An unknown value changes nothing, so a buggy caller cannot overwrite the saved choice.
    saved.set(THEME_STORAGE_KEY, 'cobalt');
    Object.defineProperty(globalThis, 'localStorage', { configurable: true, value: { setItem: (key: string, value: string) => saved.set(key, value) } });
    assert.equal(applyTheme('neon' as never, root), false);
    assert.equal(root.dataset.theme, 'vellum');
    assert.equal(saved.get(THEME_STORAGE_KEY), 'cobalt');
  } finally {
    if (original) Object.defineProperty(globalThis, 'localStorage', original);
    else delete (globalThis as { localStorage?: unknown }).localStorage;
  }
});

// Runs the pre-paint script from ThemeHead.astro the way Astro emits it (define:vars as consts).
function runThemeHead(getItem: () => string | null): string | undefined {
  const source = readFileSync(new URL('../src/components/ThemeHead.astro', import.meta.url), 'utf8');
  const body = source.match(/<script is:inline[^>]*>([\s\S]*?)<\/script>/)?.[1];
  assert.ok(body, 'no inline script in ThemeHead.astro');
  const documentElement = { dataset: {} as Record<string, string> };
  const keys: string[] = [];
  runInNewContext(`const themes = ${JSON.stringify(THEMES)}, fallback = ${JSON.stringify(DEFAULT_THEME)}, key = ${JSON.stringify(THEME_STORAGE_KEY)};${body}`, {
    document: { documentElement },
    localStorage: { getItem: (key: string) => { keys.push(key); return getItem(); } },
  });
  assert.deepEqual(keys, [THEME_STORAGE_KEY]);
  return documentElement.dataset.theme;
}

test('the pre-paint script agrees with resolveTheme', () => {
  for (const saved of [null, '', 'neon', 'Cobalt', ...THEMES]) assert.equal(runThemeHead(() => saved), resolveTheme(saved), String(saved));
  assert.equal(runThemeHead(() => { throw new Error('blocked'); }), DEFAULT_THEME);
});

test('every theme defines the spec palette', () => {
  for (const theme of THEMES) {
    const block = themeBlock(theme);
    for (const [token, value] of Object.entries(SPEC[theme])) assert.equal(block[token]?.toUpperCase(), value, `${theme} --${token}`);
    assert.deepEqual(block['ink-rgb'].split(',').map(Number), rgb(block.ink), `${theme} --ink-rgb matches --ink`);
    // Native controls and scrollbars follow the paper's lightness.
    const scheme = css.match(new RegExp(`\\[data-theme="${theme}"\\]\\s*\\{[^}]*color-scheme:\\s*(\\w+)`))?.[1];
    assert.equal(scheme, theme === 'vellum' || theme === 'cobalt' ? 'light' : 'dark', `${theme} color-scheme`);
  }
});

test('dark is also the default without a data-theme attribute', () => {
  const block = blockFor(':root,[data-theme="dark"]');
  assert.ok(block, 'no `:root, [data-theme="dark"]` block');
  for (const [token, value] of Object.entries(SPEC.dark)) assert.equal(block[token]?.toUpperCase(), value, `default --${token}`);
  assert.equal(block['ink-60-alpha'], '0.6');
});

// Only --ink-60 carries text; --ink-50 and below are borders, rules and tints.
test('labels at --ink-60 meet 4.5:1 on --paper in every theme', () => {
  for (const theme of THEMES) {
    const block = themeBlock(theme);
    const alpha = Number(block['ink-60-alpha']);
    assert.equal(alpha, LABEL_ALPHA[theme], `${theme} --ink-60-alpha (documented in design-system.md)`);
    const paper = rgb(block.paper);
    const ratio = contrast(over(rgb(block.ink), paper, alpha), paper);
    assert.ok(ratio >= 4.5, `${theme} --ink-60 contrast ${ratio.toFixed(2)}:1`);
  }
});

test('alpha steps resolve per theme root and map into Tailwind', () => {
  // On [data-theme] too, so a nested themed element (a theme preview) resolves its own alphas.
  const steps = blockFor(':root,[data-theme]');
  assert.ok(steps, 'no `:root, [data-theme]` block for the alpha steps');
  assert.equal(steps['ink-60'], 'rgba(var(--ink-rgb), var(--ink-60-alpha))');
  for (const [step, alpha] of [['50', '0.5'], ['30', '0.3'], ['14', '0.14'], ['07', '0.07'], ['05', '0.05']]) assert.equal(steps[`ink-${step}`], `rgba(var(--ink-rgb), ${alpha})`, `--ink-${step}`);

  const tailwind = blockFor('@theme inline');
  assert.ok(tailwind, 'no `@theme inline` block');
  for (const token of ['paper', 'ink', 'accent', 'ink-hover', 'ink-60', 'ink-50', 'ink-30', 'ink-14', 'ink-07', 'ink-05']) assert.equal(tailwind[`color-${token}`], `var(--${token})`, `--color-${token}`);
  const body = css.match(/\bbody\s*\{([^}]*)\}/)?.[1] ?? '';
  assert.match(body, /\bbg-paper\b/);
  assert.match(body, /\btext-ink\b/);

  const fonts = blockFor('@theme');
  assert.ok(fonts, 'no `@theme` font block');
  assert.match(fonts['font-sans'], /^"Inter Tight Variable"/);
  assert.match(fonts['font-mono'], /^"IBM Plex Mono"/);
  assert.match(fonts['font-hand'], /^"Architects Daughter"/);
});

test('fonts are self-hosted, never loaded from a font CDN', () => {
  for (const pkg of ['@fontsource-variable/inter-tight', '@fontsource/ibm-plex-mono', '@fontsource/architects-daughter']) assert.match(css, new RegExp(`@import "${pkg}/`));
  const files = (dir: string): string[] => readdirSync(dir).flatMap(name => {
    if (name === 'node_modules' || name.startsWith('.')) return [];
    const path = join(dir, name);
    return statSync(path).isDirectory() ? files(path) : [path];
  });
  const root = new URL('..', import.meta.url).pathname;
  const cdn = /fonts\.(googleapis|gstatic|google)\.com|rsms\.me\/inter|unpkg\.com\/@?fontsource|fonts\.bunny\.net|use\.typekit\.net|p\.typekit\.net|cdn\.jsdelivr\.net\/npm\/@?fontsource/;
  for (const file of ['src', 'public', 'internal', 'extension'].flatMap(dir => files(join(root, dir)))) {
    if (!/\.(astro|[cm]?[jt]sx?|css|html|templ|go|svg|webmanifest|json)$/.test(file) || file.endsWith('_templ.go')) continue;
    assert.doesNotMatch(readFileSync(file, 'utf8'), cdn, file);
  }
});

// Inactive segmented tabs use max(0.7, --ink-60-alpha) (public/ui/primitives.css): 0.7 as in the
// reference, raised where a theme's label alpha is higher, so the text keeps 4.5:1 everywhere.
test('inactive segmented tab text meets 4.5:1 on every theme', () => {
  const primitives = readFileSync(new URL('../public/ui/primitives.css', import.meta.url), 'utf8');
  assert.match(primitives, /\.ui-tabs__tab \{[^}]*color: rgba\(var\(--ink-rgb\), max\(0\.7, var\(--ink-60-alpha\)\)\);/);
  for (const theme of THEMES) {
    const block = themeBlock(theme);
    const alpha = Math.max(0.7, Number(block['ink-60-alpha']));
    const ratio = contrast(over(rgb(block.ink), rgb(block.paper), alpha), rgb(block.paper));
    assert.ok(ratio >= 4.5, `${theme}: ${ratio.toFixed(2)}:1`);
  }
});
