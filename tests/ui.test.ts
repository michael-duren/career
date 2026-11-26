import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { buttonClassName, cssUrl, goalPattern, monogram, nextTabIndex, tickMarks } from '../src/lib/ui.ts';

test('goal patterns follow goal order and fall back to an outline', () => {
  assert.deepEqual([0, 1, 2, 3, 9].map(goalPattern), ['diagonal', 'crosshatch', 'dots', 'outline', 'outline']);
  assert.equal(goalPattern(-1), 'outline');
  assert.equal(goalPattern(1.5), 'outline');
  assert.equal(goalPattern(Number.NaN), 'outline');
});

test('tick marks fill from the left and make every 5th tick major', () => {
  const ticks = tickMarks(11, 2);
  assert.equal(ticks.length, 11);
  assert.deepEqual(ticks.map(t => t.on), [true, true, false, false, false, false, false, false, false, false, false]);
  assert.deepEqual(ticks.flatMap((t, i) => (t.major ? [i] : [])), [0, 5, 10]);
  assert.equal(tickMarks(5, 9).filter(t => t.on).length, 5);
  assert.equal(tickMarks(5, -3).filter(t => t.on).length, 0);
  assert.deepEqual(tickMarks(-2, 1), []);
  assert.deepEqual(tickMarks(Number.NaN, 1), []);
  assert.equal(tickMarks(3, Number.NaN).filter(t => t.on).length, 0);
});

test('monogram uses the first two words, else the first two letters', () => {
  assert.equal(monogram('Jane Street'), 'JS');
  assert.equal(monogram('Ramp'), 'RA');
  assert.equal(monogram('  two  sigma investments '), 'TS');
  assert.equal(monogram('AT&T'), 'AT');
  assert.equal(monogram('Écrit Labs'), 'ÉL');
  assert.equal(monogram('X'), 'X');
  assert.equal(monogram(''), '?');
  assert.equal(monogram('—'), '?');
});

test('tab keys wrap with the arrows and jump with Home and End', () => {
  assert.equal(nextTabIndex('ArrowRight', 3, 4), 0);
  assert.equal(nextTabIndex('ArrowRight', 1, 4), 2);
  assert.equal(nextTabIndex('ArrowLeft', 0, 4), 3);
  assert.equal(nextTabIndex('Home', 2, 4), 0);
  assert.equal(nextTabIndex('End', 0, 4), 3);
  assert.equal(nextTabIndex('Enter', 0, 4), null);
  assert.equal(nextTabIndex('ArrowRight', 0, 0), null);
});

test('cssUrl escapes for a CSS string', () => {
  assert.equal(cssUrl('/logos/ramp.svg'), 'url("/logos/ramp.svg")');
  assert.equal(cssUrl('a"b\\c\nd'), 'url("a\\22 b\\5c c\\a d")');
});

test('button classes', () => {
  assert.equal(buttonClassName(), 'ui-btn');
  assert.equal(buttonClassName({ variant: 'primary' }), 'ui-btn ui-btn--primary');
  assert.equal(buttonClassName({ variant: 'danger', size: 'sm', icon: true }), 'ui-btn ui-btn--danger ui-btn--sm ui-btn--icon');
});

// The CSS layer is shared with Go pages, so it must stay on theme tokens: no raw colors.
test('primitives.css only uses theme tokens', () => {
  const css = readFileSync(new URL('../public/ui/primitives.css', import.meta.url), 'utf8').replace(/\/\*[\s\S]*?\*\//g, '');
  assert.doesNotMatch(css, /#[0-9a-f]{3,8}\b/i);
  assert.doesNotMatch(css, /rgba?\(\s*\d/);
  assert.doesNotMatch(css, /@apply|@theme|@tailwind/);
});
