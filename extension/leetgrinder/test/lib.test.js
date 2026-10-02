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

test("timerExpired", () => {
  const min = 60000;
  const start = 1_000_000_000;
  assert.ok(lib.timerExpired(undefined, start));
  assert.ok(lib.timerExpired({ startedAt: "x" }, start));
  // Reloads and short absences keep the timer.
  assert.ok(!lib.timerExpired({ startedAt: start, lastSeenAt: start + 40 * min }, start + 60 * min));
  assert.ok(!lib.timerExpired({ startedAt: start }, start + 30 * min));
  // A page gone for over 30 minutes starts over, as does a days-old timer.
  assert.ok(lib.timerExpired({ startedAt: start, lastSeenAt: start + 10 * min }, start + 41 * min));
  assert.ok(lib.timerExpired({ startedAt: start }, start + 3 * 24 * 60 * min));
  // A tab left open all day (heartbeats continuing) still restarts.
  assert.ok(lib.timerExpired({ startedAt: start, lastSeenAt: start + 13 * 60 * min }, start + 13 * 60 * min));
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
    wantsReview: true,
    approach: "suboptimal",
    timeComplexity: "O(n)",
    spaceComplexity: "O(n^2)",
    extra: "dropped",
  };
  const cleaned = lib.cleanAttempt(good);
  assert.deepEqual(Object.keys(cleaned).sort(), [
    "approach",
    "assisted",
    "code",
    "codeLanguage",
    "id",
    "isReview",
    "minutes",
    "notes",
    "outcome",
    "problemSlug",
    "spaceComplexity",
    "timeComplexity",
    "wantsReview",
  ]);
  assert.equal(cleaned.wantsReview, true);
  assert.equal(cleaned.approach, "suboptimal");
  // Unset self-assessments are left out, for servers that predate them.
  for (const unset of [{ wantsReview: undefined, approach: undefined }, { wantsReview: false, approach: "" }]) {
    const plain = lib.cleanAttempt({ ...good, ...unset });
    assert.ok(!("wantsReview" in plain) && !("approach" in plain), JSON.stringify(unset));
  }
  assert.equal(cleaned.spaceComplexity, "O(n²)");
  assert.equal(cleaned.code, "");
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
    { wantsReview: "yes" },
    { approach: "brute-force" },
    { notes: "a".repeat(2001) },
    { notes: "a\u0000b" },
  ]) {
    assert.equal(lib.cleanAttempt({ ...good, ...bad }), null, JSON.stringify(bad));
  }
  assert.ok(lib.cleanAttempt({ ...good, notes: "🙂".repeat(2000) }));
  assert.equal(lib.cleanAttempt(null), null);
});

test("cleanMetadata", () => {
  const good = { number: 1, title: " Two Sum ", difficulty: "Easy", topics: [{ slug: "array", name: "Array" }, { slug: "array", name: "Array" }, { slug: "hash-table", name: "Hash Table" }] };
  assert.deepEqual(lib.cleanMetadata(good), { number: 1, title: "Two Sum", difficulty: "Easy", topics: [{ slug: "array", name: "Array" }, { slug: "hash-table", name: "Hash Table" }] });
  assert.deepEqual(lib.cleanMetadata({}), { number: 0, title: "", difficulty: "", topics: [] });
  for (const bad of [
    null,
    [],
    { number: -1 },
    { number: 1.5 },
    { number: 100001 },
    { difficulty: "easy" },
    { title: "x".repeat(201) },
    { title: "a\u0000b" },
    { topics: [{ slug: "Bad Slug", name: "x" }] },
    { topics: [{ slug: "array" }] },
    { topics: Array.from({ length: 21 }, (_, i) => ({ slug: `t${i}`, name: "T" })) },
  ]) {
    assert.equal(lib.cleanMetadata(bad), null, JSON.stringify(bad));
  }
});

test("metadataFromGraphQL", () => {
  const json = { data: { question: { questionFrontendId: "146", title: "LRU Cache", difficulty: "Medium", topicTags: [{ slug: "design", name: "Design" }] } } };
  assert.deepEqual(lib.metadataFromGraphQL(json), { number: 146, title: "LRU Cache", difficulty: "Medium", topics: [{ slug: "design", name: "Design" }] });
  // Non-numeric ids (contest problems) leave the number unknown.
  json.data.question.questionFrontendId = "LCP 01";
  assert.equal(lib.metadataFromGraphQL(json).number, 0);
  assert.equal(lib.metadataFromGraphQL({ data: { question: null } }), null);
  assert.equal(lib.metadataFromGraphQL({ errors: [] }), null);
  assert.equal(lib.metadataFromGraphQL({ data: { question: { title: "", difficulty: "Easy" } } }), null);
  assert.ok(lib.GRAPHQL_QUERY.includes("topicTags"));
});

test("attempts carry optional problem metadata", () => {
  const fields = { id: "3f2504e0-4f89-41d3-9a0c-0305e82c3301", problemSlug: "lru-cache", outcome: "unfinished", minutes: 30, assisted: false, notes: "", isReview: false, wantsReview: false, approach: "", timeComplexity: "", spaceComplexity: "" };
  const withMeta = lib.buildAttempt({ ...fields, problem: { number: 146, title: "LRU Cache", difficulty: "Medium", topics: [] } }, null, false);
  assert.equal(withMeta.problem.title, "LRU Cache");
  assert.equal(lib.cleanAttempt(withMeta).problem.number, 146);
  assert.equal(lib.buildAttempt(fields, null, false).problem, undefined);
  assert.equal(lib.cleanAttempt(lib.buildAttempt(fields, null, false)).problem, undefined);
  assert.equal(lib.cleanAttempt({ ...lib.buildAttempt(fields, null, false), problem: { difficulty: "Trivial" } }), null);
  // Invalid metadata is left off rather than blocking the attempt.
  assert.equal(lib.buildAttempt({ ...fields, problem: { number: -1 } }, null, false).problem, undefined);
});

// Active time: simulate the 30-second heartbeat against creditActive.
function simulate(lib, { startedAt, minutes, visibleAt, inputAt }) {
  let timer = { startedAt, activeMs: 0, lastSampleAt: startedAt };
  let lastInput = startedAt;
  for (let t = startedAt + 30000; t <= startedAt + minutes * 60000; t += 30000) {
    const input = inputAt(t);
    if (input !== null) lastInput = Math.max(lastInput, input);
    timer = lib.creditActive(timer, { visible: visibleAt(t), lastInputAt: lastInput }, t);
  }
  return timer;
}

test("active time accumulates while visible and in use", () => {
  const start = Date.UTC(2026, 9, 2, 12);
  const timer = simulate(lib, { startedAt: start, minutes: 20, visibleAt: () => true, inputAt: (t) => t });
  assert.equal(timer.activeMs, 20 * 60000);
  assert.equal(lib.activeMinutes(timer, start + 20 * 60000), 20);
});

test("active time excludes idle gaps beyond the idle threshold", () => {
  const start = Date.UTC(2026, 9, 2, 12);
  // 20 minutes of work, then a 70-minute lunch with no input (tab still visible).
  const work = 20 * 60000;
  const timer = simulate(lib, { startedAt: start, minutes: 90, visibleAt: () => true, inputAt: (t) => (t <= start + work ? t : null) });
  // Work plus at most the idle window after the last input.
  assert.equal(timer.activeMs, work + lib.ACTIVE_IDLE_MS);
  const minutes = lib.activeMinutes(timer, start + 90 * 60000);
  assert.equal(minutes, 25);
  assert.equal(lib.elapsedMinutes(start, start + 90 * 60000), 90);
});

test("active time excludes hidden time", () => {
  const start = Date.UTC(2026, 9, 2, 12);
  const timer = simulate(lib, {
    startedAt: start,
    minutes: 30,
    visibleAt: (t) => t <= start + 10 * 60000 || t > start + 25 * 60000,
    inputAt: (t) => t,
  });
  assert.equal(timer.activeMs, 15 * 60000);
});

test("one sample credits a bounded amount after a long gap", () => {
  const start = Date.UTC(2026, 9, 2, 12);
  const timer = lib.creditActive({ startedAt: start, activeMs: 0, lastSampleAt: start }, { visible: true, lastInputAt: start + 20 * 60000 }, start + 20 * 60000);
  assert.equal(timer.activeMs, lib.SAMPLE_MAX_CREDIT_MS);
  assert.equal(timer.lastSampleAt, start + 20 * 60000);
});

test("a hidden sample advances the clock without credit", () => {
  const start = Date.UTC(2026, 9, 2, 12);
  let timer = { startedAt: start, activeMs: 1000, lastSampleAt: start };
  timer = lib.creditActive(timer, { visible: false, lastInputAt: start }, start + 60000);
  assert.equal(timer.activeMs, 1000);
  timer = lib.creditActive(timer, { visible: true, lastInputAt: start + 90000 }, start + 90000);
  assert.equal(timer.activeMs, 1000 + 30000);
});

test("legacy timers without activeMs fall back to wall-clock time", () => {
  const start = Date.UTC(2026, 9, 2, 12);
  assert.equal(lib.activeMs({ startedAt: start }, start + 600000), 600000);
  assert.equal(lib.activeMinutes({ startedAt: start }, start + 600000), 10);
  const credited = lib.creditActive({ startedAt: start }, { visible: true, lastInputAt: start + 600000 }, start + 600000);
  assert.equal(credited.activeMs, 600000);
});

test("validSample", () => {
  assert.deepEqual(lib.validSample({ visible: true, lastInputAt: 5 }, 10), { visible: true, lastInputAt: 5 });
  assert.equal(lib.validSample({ visible: true, lastInputAt: 50 }, 10).lastInputAt, 10);
  assert.equal(lib.validSample({ visible: "yes", lastInputAt: 5 }, 10), null);
  assert.equal(lib.validSample({ visible: true }, 10), null);
  assert.equal(lib.validSample(null, 10), null);
});

test("prefill and outcome come from active time", () => {
  const start = Date.UTC(2026, 9, 2, 12);
  const timer = { startedAt: start, activeMs: 20 * 60000 };
  const now = start + 90 * 60000;
  const minutes = lib.activeMinutes(timer, now);
  assert.equal(minutes, 20);
  assert.equal(lib.inferOutcome(minutes), "solved");
  assert.equal(lib.inferOutcome(lib.elapsedMinutes(start, now)), "struggled");
  assert.ok(lib.showOpenTime(20, 90));
  assert.ok(!lib.showOpenTime(20, 20));
});

test("nudge uses active time", () => {
  const start = Date.UTC(2026, 9, 2, 12);
  const open = start + 90 * 60000;
  assert.ok(!lib.shouldNudge({ startedAt: start, activeMs: 20 * 60000, nudged: false }, open));
  assert.ok(lib.shouldNudge({ startedAt: start, activeMs: 25 * 60000, nudged: false }, start + 26 * 60000));
  assert.ok(!lib.shouldNudge({ startedAt: start, activeMs: 40 * 60000, nudged: true }, open));
});

test("timer expiry ignores active time", () => {
  const start = Date.UTC(2026, 9, 2, 12);
  const min = 60000;
  assert.ok(!lib.timerExpired({ startedAt: start, activeMs: 0, lastSeenAt: start + 40 * min }, start + 60 * min));
  assert.ok(lib.timerExpired({ startedAt: start, activeMs: 99 * min, lastSeenAt: start + 10 * min }, start + 41 * min));
  assert.ok(lib.timerExpired({ startedAt: start, activeMs: 0, lastSeenAt: start + 13 * 60 * min }, start + 13 * 60 * min));
});
