// Run with: node --test extension/leetgrinder/test/
const test = require("node:test");
const assert = require("node:assert/strict");
const lib = require("../lib.js");

test("slugFromPath", () => {
  assert.equal(lib.slugFromPath("/problems/two-sum/"), "two-sum");
  assert.equal(lib.slugFromPath("/problems/two-sum"), "two-sum");
  assert.equal(lib.slugFromPath("/problems/two-sum/solutions/123/abc/"), "two-sum");
  assert.equal(lib.slugFromPath("/problems/Two_Sum/"), null);
  assert.equal(lib.slugFromPath("/problemset/"), null);
  assert.equal(lib.slugFromPath(""), null);
});

test("isAssistPath", () => {
  assert.ok(lib.isAssistPath("/problems/two-sum/solutions/"));
  assert.ok(lib.isAssistPath("/problems/two-sum/solutions/42/fast/"));
  assert.ok(lib.isAssistPath("/problems/two-sum/editorial/"));
  assert.ok(!lib.isAssistPath("/problems/two-sum/description/"));
  assert.ok(!lib.isAssistPath("/problems/two-sum/submissions/"));
  assert.ok(!lib.isAssistPath("/problems/solutions-count/"));
});

test("elapsedMinutes and inferOutcome", () => {
  const start = 1_000_000;
  assert.equal(lib.elapsedMinutes(start, start), 1);
  assert.equal(lib.elapsedMinutes(start, start + 25 * 60000), 25);
  assert.equal(lib.elapsedMinutes(start, start + 25.4 * 60000), 25);
  assert.equal(lib.elapsedMinutes(start, start + 10 * 3600000), 240);
  assert.equal(lib.elapsedMinutes(NaN, start), 1);
  assert.equal(lib.inferOutcome(1), "solved");
  assert.equal(lib.inferOutcome(25), "solved");
  assert.equal(lib.inferOutcome(26), "struggled");
});

test("shouldNudge", () => {
  const start = 0;
  assert.ok(!lib.shouldNudge({ startedAt: start, nudged: false }, 24 * 60000));
  assert.ok(lib.shouldNudge({ startedAt: start, nudged: false }, 25 * 60000));
  assert.ok(!lib.shouldNudge({ startedAt: start, nudged: true }, 60 * 60000));
  assert.ok(!lib.shouldNudge(null, 60 * 60000));
});

test("normalizeOrigin and originPattern", () => {
  assert.equal(lib.normalizeOrigin("https://career.example.com/leetgrinder"), "https://career.example.com");
  assert.equal(lib.normalizeOrigin(" https://career.example.com:8443 "), "https://career.example.com:8443");
  assert.equal(lib.normalizeOrigin("http://localhost:8080"), "http://localhost:8080");
  assert.equal(lib.normalizeOrigin("http://127.0.0.1:8080"), "http://127.0.0.1:8080");
  assert.equal(lib.normalizeOrigin("http://career.example.com"), null);
  assert.equal(lib.normalizeOrigin("https://user:pw@career.example.com"), null);
  assert.equal(lib.normalizeOrigin("javascript:alert(1)"), null);
  assert.equal(lib.normalizeOrigin("not a url"), null);
  assert.equal(lib.originPattern("https://career.example.com:8443"), "https://career.example.com/*");
  assert.equal(lib.originPattern("http://localhost:8080"), "http://localhost/*");
});

test("validToken", () => {
  assert.ok(lib.validToken("lg_" + "A".repeat(43)));
  assert.ok(!lib.validToken("lg_" + "A".repeat(42)));
  assert.ok(!lib.validToken("xx_" + "A".repeat(43)));
  assert.ok(!lib.validToken(undefined));
});

test("cleanAttempt", () => {
  const good = {
    id: "3f2c1a4e-8b9d-4c6e-9f0a-1b2c3d4e5f60",
    problemSlug: "two-sum",
    outcome: "solved",
    minutes: 18,
    assisted: false,
    notes: "hash map",
    isReview: true,
    extra: "dropped",
  };
  const cleaned = lib.cleanAttempt(good);
  assert.deepEqual(Object.keys(cleaned).sort(), ["assisted", "id", "isReview", "minutes", "notes", "outcome", "problemSlug"]);
  for (const bad of [
    { id: "nope" },
    { problemSlug: "../etc" },
    { outcome: "great" },
    { minutes: 0 },
    { minutes: 241 },
    { minutes: 1.5 },
    { minutes: "20" },
    { assisted: "yes" },
    { isReview: undefined },
    { notes: "a".repeat(2001) },
    { notes: "a\u0000b" },
  ]) {
    assert.equal(lib.cleanAttempt({ ...good, ...bad }), null, JSON.stringify(bad));
  }
  assert.ok(lib.cleanAttempt({ ...good, notes: "🙂".repeat(2000) }));
  assert.equal(lib.cleanAttempt(null), null);
});
