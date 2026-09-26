import test from 'node:test';
import assert from 'node:assert/strict';
import { NEUTRAL_TONE, TAG_PALETTE, entryTone, tagChipClass, tagHash, tagTone } from '../src/lib/tag-colors.ts';
import { countWords, noteDate, noteStats } from '../src/lib/note-card.ts';

test('a tag always maps to the same palette color', () => {
  assert.equal(tagHash('systems'), tagHash('systems'));
  assert.equal(tagTone('systems'), tagTone('systems'));
  // Pinned so a palette or hash change that recolors saved notes is deliberate.
  assert.equal(tagHash(''), 0x811c9dc5);
  assert.equal(tagHash('a'), 0xe40c292c);
  assert.equal(tagTone('a'), TAG_PALETTE[0xe40c292c % TAG_PALETTE.length]);
});

test('case and surrounding space do not change a color', () => {
  assert.equal(tagTone(' Go '), tagTone('go'));
  assert.equal(tagHash('Distributed Systems'), tagHash('distributed systems'));
});

test('tags spread across the palette', () => {
  const tags = ['go', 'rust', 'systems', 'career', 'networking', 'os', 'interview', 'databases', 'ebpf', 'kubernetes', 'linux', 'reading', 'ideas', 'questions', 'observability', 'typescript'];
  const used = new Set(tags.map(tag => tagTone(tag).name));
  assert.ok(used.size >= 6, `only ${used.size} colors used`);
  for (const tag of tags) assert.ok(TAG_PALETTE.includes(tagTone(tag)));
});

test('untagged entries are neutral and tagged ones follow their first tag', () => {
  assert.equal(tagTone(''), NEUTRAL_TONE);
  assert.equal(tagTone('   '), NEUTRAL_TONE);
  assert.equal(tagTone(undefined), NEUTRAL_TONE);
  assert.equal(entryTone([]), NEUTRAL_TONE);
  assert.equal(entryTone(undefined), NEUTRAL_TONE);
  assert.equal(entryTone(['rust', 'go']), tagTone('rust'));
});

test('chip classes carry the tone and a pill shape', () => {
  const chip = tagChipClass('go');
  assert.ok(chip.includes(tagTone('go').chip));
  assert.ok(chip.includes('rounded-full'));
  const names = TAG_PALETTE.map(tone => tone.name);
  assert.equal(new Set(names).size, names.length);
  for (const tone of TAG_PALETTE) for (const key of ['card', 'chip', 'dot'] as const) assert.ok(tone[key].includes(tone.name));
});

test('note stats prefer the list summary and fall back to the body', () => {
  assert.equal(countWords('  one two\n\nthree  '), 3);
  assert.equal(countWords('   '), 0);
  assert.deepEqual(noteStats({ body: '', summary: { wordCount: 450, todoCount: 3, todoDone: 1 } }), { words: 450, minutes: 2, todos: 3, todosDone: 1 });
  assert.deepEqual(noteStats({ body: 'a b c', todos: [{ id: '1', title: 'x', done: true }, { id: '2', title: 'y', done: false }] }), { words: 3, minutes: 1, todos: 2, todosDone: 1 });
  assert.deepEqual(noteStats({ body: '' }), { words: 0, minutes: 0, todos: 0, todosDone: 0 });
});

test('note dates read relative for the last week', () => {
  const now = new Date(2026, 8, 26, 15);
  assert.equal(noteDate(undefined, now), '');
  assert.equal(noteDate('nope', now), '');
  assert.equal(noteDate(new Date(2026, 8, 26, 1).toISOString(), now), 'Today');
  assert.equal(noteDate(new Date(2026, 8, 25, 23).toISOString(), now), 'Yesterday');
  assert.equal(noteDate(new Date(2026, 8, 22, 12).toISOString(), now), '4 days ago');
  assert.equal(noteDate(new Date(2026, 7, 1, 12).toISOString(), now), new Date(2026, 7, 1, 12).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }));
});
