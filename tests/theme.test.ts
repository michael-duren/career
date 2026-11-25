import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { DEFAULT_THEME, THEMES, isTheme, resolveTheme } from '../src/lib/theme.ts';

const css = readFileSync(new URL('../src/styles/global.css', import.meta.url), 'utf8');

// The palette from docs/redesign/design-system.md ("Themes and tokens").
const SPEC = {
  dark: { paper: '#0C0C0B', ink: '#ECEAE3', accent: '#E0533D', 'ink-hover': '#FFFFFF' },
  vellum: { paper: '#F3F1EA', ink: '#1A1A17', accent: '#C2361E', 'ink-hover': '#000000' },
  cobalt: { paper: '#F7F7F4', ink: '#0047AB', accent: '#D6402A', 'ink-hover': '#002F73' },
  concrete: { paper: '#2C2D2F', ink: '#EFEDE6', accent: '#F2C230', 'ink-hover': '#FFFFFF' },
} as const;

function themeBlock(theme: string): Record<string, string> {
  const match = css.match(new RegExp(`\\[data-theme="${theme}"\\]\\s*\\{([^}]*)\\}`));
  assert.ok(match, `no [data-theme="${theme}"] block in global.css`);
  return Object.fromEntries([...match[1].matchAll(/--([\w-]+):\s*([^;]+);/g)].map(([, name, value]) => [name, value.trim()]));
}

// Declarations of the CSS block whose selector list is exactly `selector` (whitespace-insensitive).
function blockFor(selector: string): Record<string, string> | undefined {
  const wanted = selector.replace(/\s+/g, '');
  for (const [, sel, body] of css.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    if (sel.replace(/\/\*[\s\S]*?\*\//g, '').replace(/^[\s\S]*;/, '').replace(/\s+/g, '') !== wanted) continue;
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

test('every theme defines the spec palette', () => {
  for (const theme of THEMES) {
    const block = themeBlock(theme);
    for (const [token, value] of Object.entries(SPEC[theme])) assert.equal(block[token]?.toUpperCase(), value, `${theme} --${token}`);
    assert.deepEqual(block['ink-rgb'].split(',').map(Number), rgb(block.ink), `${theme} --ink-rgb matches --ink`);
  }
});

test('dark is also the default without a data-theme attribute', () => {
  assert.equal(blockFor(':root,[data-theme="dark"]')?.paper, SPEC.dark.paper);
});

// Only --ink-60 carries text; --ink-50 and below are borders, rules and tints.
test('labels at --ink-60 meet 4.5:1 on --paper in every theme', () => {
  for (const theme of THEMES) {
    const block = themeBlock(theme);
    const alpha = Number(block['ink-60-alpha']);
    assert.ok(alpha >= 0.6 && alpha <= 1, `${theme} --ink-60-alpha ${alpha}`);
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
  assert.match(css, /body\s*\{\s*@apply bg-paper text-ink;/);
});

test('fonts are self-hosted, never loaded from a font CDN', () => {
  for (const pkg of ['@fontsource-variable/inter-tight', '@fontsource/ibm-plex-mono', '@fontsource/architects-daughter']) assert.match(css, new RegExp(`@import "${pkg}/`));
  const files = (dir: string): string[] => readdirSync(dir).flatMap(name => {
    const path = join(dir, name);
    return statSync(path).isDirectory() ? files(path) : [path];
  });
  for (const file of [...files('src'), ...files('public'), ...files('internal/leetgrinder')]) {
    if (!/\.(astro|tsx?|css|html|templ|js|mjs)$/.test(file)) continue;
    assert.doesNotMatch(readFileSync(file, 'utf8'), /fonts\.(googleapis|gstatic)\.com/, file);
  }
});
