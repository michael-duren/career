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
  // Attempts saved by an older version carry neither field.
  const older = lib.cleanAttempt({ ...good, wantsReview: undefined, approach: undefined });
  assert.equal(older.wantsReview, false);
  assert.equal(older.approach, "");
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
