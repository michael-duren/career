import test from 'node:test';
import assert from 'node:assert/strict';
import { NEUTRAL_TONE, TAG_PALETTE, entryTone, tagChipClass, tagHash, tagTone } from '../src/lib/tag-colors.ts';
import { countWords, filterNotes, matchesNoteSearch, noteCardDate, noteStats, topicCounts, withoutTitleHeading } from '../src/lib/note-card.ts';

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

test('composed and decomposed spellings share a color', () => {
  assert.equal(tagHash('caf\u00e9'), tagHash('cafe\u0301'));
  assert.equal(tagTone('Caf\u00e9'), tagTone('cafe\u0301'));
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

test('note topics are counted and listed alphabetically', () => {
  assert.deepEqual(topicCounts([{ topic: 'SRE' }, { topic: 'Go' }, { topic: 'SRE' }]), [{ name: 'Go', count: 1 }, { name: 'SRE', count: 2 }]);
  assert.deepEqual(topicCounts([]), []);
});

test('notes filter by one topic and search, newest update first', () => {
  const note = (title: string, topic: string, updatedAt?: string, extra: { description?: string; tags?: string[] } = {}) => ({ title, topic, description: extra.description ?? '', tags: extra.tags ?? [], updatedAt });
  const notes = [
    note('Raft', 'Systems', '2026-11-01T00:00:00Z', { tags: ['consensus'] }),
    note('Indexes', 'Databases', '2026-11-20T00:00:00Z', { description: 'B-tree pages' }),
    note('Paxos', 'Systems', '2026-11-10T00:00:00Z'),
    note('Undated', 'Systems'),
  ];
  assert.deepEqual(filterNotes(notes, null, '').map(n => n.title), ['Indexes', 'Paxos', 'Raft', 'Undated']);
  assert.deepEqual(filterNotes(notes, 'Systems', '').map(n => n.title), ['Paxos', 'Raft', 'Undated']);
  // Search covers title, topic, summary and tags.
  assert.deepEqual(filterNotes(notes, null, 'CONSENSUS').map(n => n.title), ['Raft']);
  assert.deepEqual(filterNotes(notes, null, 'b-tree').map(n => n.title), ['Indexes']);
  assert.deepEqual(filterNotes(notes, null, 'databases').map(n => n.title), ['Indexes']);
  assert.deepEqual(filterNotes(notes, 'Databases', 'raft'), []);
  // Same update time: by title.
  assert.deepEqual(filterNotes([note('b', 'X', 'same'), note('a', 'X', 'same')], null, '').map(n => n.title), ['a', 'b']);
  assert.equal(matchesNoteSearch(notes[0], '   '), true);
});

test('note card dates are short, with the year only for past years', () => {
  const now = new Date(2026, 10, 27, 12);
  assert.equal(noteCardDate(new Date(2026, 10, 25, 12).toISOString(), now), 'Nov 25');
  assert.equal(noteCardDate(new Date(2025, 0, 3, 12).toISOString(), now), 'Jan 3 2025');
  assert.equal(noteCardDate(undefined, now), '');
  assert.equal(noteCardDate('nope', now), '');
});

test('the reader drops a leading heading that repeats the note title', () => {
  assert.equal(withoutTitleHeading('# Raft consensus\n\nLeader election.', 'Raft consensus'), 'Leader election.');
  assert.equal(withoutTitleHeading('\n# Raft consensus #\r\nBody', ' Raft consensus'), 'Body');
  assert.equal(withoutTitleHeading('# Raft consensus', 'Raft consensus'), '');
  // Different heading, a lower-level heading, or a heading further down stay.
  assert.equal(withoutTitleHeading('# Paxos\nBody', 'Raft consensus'), '# Paxos\nBody');
  assert.equal(withoutTitleHeading('## Raft consensus\nBody', 'Raft consensus'), '## Raft consensus\nBody');
  assert.equal(withoutTitleHeading('Intro\n# Raft consensus', 'Raft consensus'), 'Intro\n# Raft consensus');
  // A # that is part of the title is not a closing sequence.
  assert.equal(withoutTitleHeading('# C#\nBody', 'C#'), 'Body');
  assert.equal(withoutTitleHeading('# C# ##\nBody', 'C#'), 'Body');
});
