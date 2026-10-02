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

// Active time: simulate heartbeats from one or more tabs against creditActive.
// Each tab is {id, offset, visible(t), input(t) -> last input time or null}.
function simulate(startedAt, minutes, tabs) {
  let timer = { startedAt, activeMs: 0, tabs: {}, creditedTo: startedAt };
  const lastInput = {};
  const beats = [];
  for (const tab of tabs) {
    lastInput[tab.id] = 0;
    for (let t = startedAt + (tab.offset || 0); t <= startedAt + minutes * 60000; t += 30000) beats.push([t, tab]);
  }
  beats.sort((x, y) => x[0] - y[0]);
  for (const [t, tab] of beats) {
    const input = tab.input(t);
    if (input !== null) lastInput[tab.id] = Math.max(lastInput[tab.id], input);
    timer = lib.creditActive(timer, { visible: tab.visible(t), lastInputAt: lastInput[tab.id] }, t, tab.id);
  }
  return timer;
}

const T0 = Date.UTC(2026, 9, 2, 12);
const MIN = 60000;
const busy = { id: 1, visible: () => true, input: (t) => t };

test("active time accumulates while visible and in use", () => {
  const timer = simulate(T0, 20, [busy]);
  assert.equal(timer.activeMs, 20 * MIN);
  assert.equal(lib.activeMinutes(timer, T0 + 20 * MIN), 20);
});

test("active time excludes idle gaps beyond the idle threshold", () => {
  const work = 20 * MIN;
  const timer = simulate(T0, 90, [{ id: 1, visible: () => true, input: (t) => (t <= T0 + work ? t : null) }]);
  // Work plus the idle window after the last input, never the 70-minute lunch.
  assert.ok(timer.activeMs <= work + lib.ACTIVE_IDLE_MS);
  assert.ok(timer.activeMs >= work + lib.ACTIVE_IDLE_MS - 30000 - 30000);
  assert.ok(lib.activeMinutes(timer, T0 + 90 * MIN) <= 30);
  assert.equal(lib.elapsedMinutes(T0, T0 + 90 * MIN), 90);
});

test("active time excludes hidden time", () => {
  const timer = simulate(T0, 30, [{ id: 1, visible: (t) => t <= T0 + 10 * MIN || t > T0 + 25 * MIN, input: (t) => t }]);
  assert.ok(Math.abs(timer.activeMs - 15 * MIN) <= 60000);
});

test("a hidden twin tab does not swallow the visible tab's time", () => {
  const timer = simulate(T0, 20, [busy, { id: 2, offset: 15000, visible: () => false, input: () => null }]);
  assert.ok(timer.activeMs >= 20 * MIN - 60000, String(timer.activeMs));
});

test("two visible tabs count once", () => {
  const timer = simulate(T0, 20, [busy, { id: 2, offset: 15000, visible: () => true, input: (t) => t }]);
  assert.ok(timer.activeMs <= 20 * MIN);
  assert.ok(timer.activeMs >= 20 * MIN - 60000);
});

test("active time never exceeds time since the timer started", () => {
  const t = lib.creditActive({ startedAt: T0, activeMs: 5 * MIN, tabs: { 1: T0 + 10000 }, creditedTo: T0 }, { visible: true, lastInputAt: T0 + 20000 }, T0 + 20000, 1);
  assert.equal(t.activeMs, 20000);
});

test("a clock that goes backward credits nothing and resets the mark", () => {
  const timer = { startedAt: T0, activeMs: 60000, tabs: { 1: T0 + 10 * MIN }, creditedTo: T0 + 10 * MIN };
  const back = lib.creditActive(timer, { visible: true, lastInputAt: T0 + 5 * MIN }, T0 + 5 * MIN, 1);
  assert.equal(back.activeMs, 60000);
  assert.equal(back.tabs[1], T0 + 5 * MIN);
  assert.ok(back.creditedTo <= T0 + 5 * MIN);
  const next = lib.creditActive(back, { visible: true, lastInputAt: T0 + 5 * MIN + 30000 }, T0 + 5 * MIN + 30000, 1);
  assert.equal(next.activeMs, 90000);
});

test("one sample credits at most about one heartbeat", () => {
  const timer = lib.creditActive({ startedAt: T0, activeMs: 0, tabs: { 1: T0 }, creditedTo: T0 }, { visible: true, lastInputAt: T0 + 5 * MIN }, T0 + 5 * MIN, 1);
  assert.equal(timer.activeMs, lib.SAMPLE_MAX_CREDIT_MS);
  assert.equal(timer.tabs[1], T0 + 5 * MIN);
});

test("hidden and visible samples interleave", () => {
  let timer = { startedAt: T0, activeMs: 0, tabs: { 1: T0 }, creditedTo: T0 };
  // Hide flush counts the stretch up to now as visible.
  timer = lib.creditActive(timer, { visible: true, lastInputAt: T0 + 20000 }, T0 + 20000, 1);
  assert.equal(timer.activeMs, 20000);
  // Show: records the hidden stretch with no credit.
  timer = lib.creditActive(timer, { visible: false, lastInputAt: T0 + 20000 }, T0 + 5 * MIN, 1);
  assert.equal(timer.activeMs, 20000);
  timer = lib.creditActive(timer, { visible: true, lastInputAt: T0 + 5 * MIN + 30000 }, T0 + 5 * MIN + 30000, 1);
  assert.equal(timer.activeMs, 50000);
});

test("stale tab entries are pruned", () => {
  const timer = lib.creditActive({ startedAt: T0, activeMs: 0, tabs: { 7: T0, 1: T0 + 20 * MIN - 1000 }, creditedTo: T0 }, { visible: true, lastInputAt: T0 + 20 * MIN }, T0 + 20 * MIN, 1);
  assert.deepEqual(Object.keys(timer.tabs), ["1"]);
});

test("legacy timers without activeMs fall back to wall-clock time", () => {
  assert.equal(lib.activeMs({ startedAt: T0 }, T0 + 10 * MIN), 10 * MIN);
  assert.equal(lib.activeMinutes({ startedAt: T0 }, T0 + 10 * MIN), 10);
  const credited = lib.creditActive({ startedAt: T0 }, { visible: true, lastInputAt: T0 + 10 * MIN }, T0 + 10 * MIN, 1);
  assert.equal(credited.activeMs, 10 * MIN);
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
