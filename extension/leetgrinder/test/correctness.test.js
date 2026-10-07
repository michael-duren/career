const { test } = require('node:test');
const assert = require('node:assert/strict');
const lib = require('../lib.js');
const answers = { claim: 'Returns the target pair.', invariant: 'Seen contains earlier values.', initially: 'Seen is empty.', afterStep: 'Insert the current value.', therefore: 'All earlier values are represented.', termination: 'Every index has been checked.' };
const fields = { id: '3f2c1a4e-8b9d-4c6e-9f0a-1b2c3d4e5f60', problemSlug: 'two-sum', outcome: 'solved', minutes: 18, assisted: false, notes: '', isReview: false, timeComplexity: 'O(n)', spaceComplexity: 'O(n)', ...answers };
test('correctness answers survive building and outbox cleaning', () => {
  const built = lib.buildAttempt(fields, null, false);
  const cleaned = lib.cleanAttempt(JSON.parse(JSON.stringify(built)));
  for (const [key, value] of Object.entries(answers)) assert.equal(cleaned[key], value);
});
test('correctness answers are optional and bounded', () => {
  const built = lib.buildAttempt(fields, null, false);
  for (const key of Object.keys(answers)) {
    assert.equal(lib.cleanAttempt({ ...built, [key]: 'x'.repeat(2001) }), null);
    assert.equal(lib.cleanAttempt({ ...built, [key]: 7 }), null);
    assert.equal(lib.cleanAttempt({ ...built, [key]: 'a\u0000b' }), null);
    assert.ok(lib.cleanAttempt({ ...built, [key]: '😀'.repeat(2000) }));
  }
  const old = { ...fields };
  for (const key of Object.keys(answers)) delete old[key];
  assert.ok(lib.cleanAttempt(old));
});
