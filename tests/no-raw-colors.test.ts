import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';

// Colors come from the theme tokens (public/ui/tokens.css: --paper, --ink, --ink-NN, --accent and
// the Tailwind utilities built on them), so every page works in all four themes. This guards the
// app's UI code against raw colors creeping back: hex values, rgb()/hsl()-style colors with
// literal values, Tailwind palette classes (zinc, blue, white…) and named colors in CSS.

const ROOT = new URL('..', import.meta.url).pathname;
const SCAN = ['src', 'public/ui', 'internal'];
const EXTENSIONS = /\.(tsx?|astro|css|m?js|templ|go)$/;
// Go tests hold color data as fixtures; *_templ.go is generated from the .templ files, which are scanned.
const SKIP = /(_test\.go|_templ\.go)$/;

// Allowed raw colors, each with the reason. `match` is tested against the offending text.
const ALLOWED: { file: string; match: RegExp; reason: string }[] = [
  { file: 'public/ui/tokens.css', match: /./, reason: 'the token file defines the palette' },
  { file: 'src/components/PwaHead.astro', match: /^#0C0C0B$/, reason: 'theme-color meta needs a literal; Dark --paper' },
  { file: 'src/components/dev/UiGallery.tsx', match: /^#(f60|fff)$/, reason: 'sample logo image (stands in for a fetched favicon)' },
  { file: 'src/lib/timeline.ts', match: /^#67e8f9$/, reason: 'DEFAULT_GOAL_COLOR: goal data the API requires, never drawn' },
  { file: 'internal/server/mcp_write.go', match: /^#67e8f9$/, reason: 'same default goal color for goals created over MCP' },
];

const PALETTE = 'slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose';
const UTILITY = String.raw`(?:bg|text|border(?:-[trblxyse])?|ring(?:-offset)?|outline|fill|stroke|from|via|to|divide|placeholder|accent|caret|decoration|shadow)`;
const NAMED = 'white|black|red|green|blue|gray|grey|silver|orange|yellow|purple|navy|maroon|teal|aqua|fuchsia|lime|olive|pink';

const PATTERNS: { name: string; re: RegExp }[] = [
  // A hex color: has a letter, or is 6/8 digits, or is a short all-digit one in a value position.
  // Issue references in comments ("#172", "(#181)") are all digits and not in a value position.
  { name: 'hex color', re: /(?<=[:'"=\[\s(,])#(?=[0-9a-fA-F]*[a-fA-F])[0-9a-fA-F]{3,8}\b|#[0-9]{6}(?:[0-9]{2})?\b|(?<=(?::\s*|['"=\[]))#[0-9]{3,4}\b/g },
  // rgb()/hsl()/oklch()… with literal channels; rgba(var(--ink-rgb), …) builds on a token and is fine.
  { name: 'color function', re: /\b(?:rgba?|hsla?|hwb|oklch|oklab|lab|lch)\((?!\s*var\()/g },
  { name: 'Tailwind palette class', re: new RegExp(String.raw`\b${UTILITY}-(?:${PALETTE})-\d{2,3}\b|\b${UTILITY}-(?:white|black)\b`, 'g') },
  { name: 'named color', re: new RegExp(String.raw`(?:(?:color|background(?:-color)?|border(?:-[a-z]+)?|outline|fill|stroke)\s*:\s*[^;"'}]*?\b(?:${NAMED})\b|\b(?:fill|stroke|color)=["'](?:${NAMED})["'])`, 'g') },
];

function files(dir: string): string[] {
  return readdirSync(join(ROOT, dir), { withFileTypes: true }).flatMap(entry => {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) return entry.name === 'node_modules' || entry.name === 'fonts' ? [] : files(path);
    return EXTENSIONS.test(entry.name) && !SKIP.test(entry.name) ? [path] : [];
  });
}

/** Every raw color in text, as "line: match (kind)". */
export function rawColors(file: string, text: string): string[] {
  const found: string[] = [];
  text.split('\n').forEach((line, i) => {
    for (const { name, re } of PATTERNS) {
      for (const match of line.matchAll(re)) {
        if (ALLOWED.some(allow => allow.file === file && allow.match.test(match[0]))) continue;
        found.push(`${file}:${i + 1}: ${match[0]} (${name})`);
      }
    }
  });
  return found;
}

test('the patterns catch raw colors and leave tokens and issue numbers alone', () => {
  const hits = (text: string) => rawColors('x.tsx', text).length;
  for (const bad of ['color: #fff;', "c = '#60a5fa'", 'bg-[#0C0C0B]', 'fill="#000"', 'background: #000000', 'hsl(${h} 60% 62%)', 'rgba(0, 0, 0, .5)',
    'className="bg-zinc-900"', 'hover:text-blue-400', 'border-l-zinc-700', 'text-white', 'ring-white/10', 'color: white', 'fill="black"']) {
    assert.ok(hits(bad) > 0, bad);
  }
  for (const good of ['// see #172 and (#181)', 'Week #12', 'rgba(var(--ink-rgb), 0.3)', 'color: var(--ink)', 'bg-paper text-ink-60 border-ink-30',
    'stroke="currentColor"', 'text-accent', 'href="#main-content"', 'whitespace-pre', 'ui-avatar--muted']) {
    assert.equal(hits(good), 0, good);
  }
});

test('no raw colors outside the token file', () => {
  const found = SCAN.flatMap(dir => files(dir)).flatMap(file => rawColors(relative(ROOT, join(ROOT, file)), readFileSync(join(ROOT, file), 'utf8')));
  assert.deepEqual(found, [], `Use the theme tokens (public/ui/tokens.css) instead:\n${found.join('\n')}`);
});

test('every allowance still matches something, so stale ones get removed', () => {
  for (const allow of ALLOWED) {
    const text = readFileSync(join(ROOT, allow.file), 'utf8');
    const used = PATTERNS.some(({ re }) => [...text.matchAll(re)].some(match => allow.match.test(match[0])));
    assert.ok(used, `${allow.file} (${allow.reason}) no longer needs its allowance`);
  }
});
