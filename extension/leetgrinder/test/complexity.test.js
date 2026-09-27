// Complexity validation, page-world message validation, and payload building.
const test = require("node:test");
const assert = require("node:assert/strict");
const lib = require("../lib.js");
// Shared with internal/leetgrinder/complexity_test.go.
const vectors = require("./complexity-vectors.json");

test("normalizeComplexity matches the shared vectors", () => {
  assert.ok(vectors.length >= 20);
  for (const v of vectors) {
    const got = lib.normalizeComplexity(v.in);
    if (v.error) assert.equal(got, null, JSON.stringify(v.in));
    else assert.equal(got, v.out, JSON.stringify(v.in));
  }
  for (const c of lib.COMPLEXITIES) assert.equal(lib.normalizeComplexity(c), c);
  assert.equal(lib.normalizeComplexity(undefined), null);
  assert.equal(lib.normalizeComplexity(3), null);
});

const capture = {
  source: "leetgrinder-detect",
  type: "submission",
  slug: "two-sum",
  submissionId: "1234567",
  status: "Accepted",
  lang: "python3",
  code: "class Solution:\n    pass\n",
};

test("cleanCapture accepts only the exact message shape", () => {
  assert.deepEqual(lib.cleanCapture(capture, "two-sum"), {
    slug: "two-sum",
    submissionId: "1234567",
    status: "Accepted",
    lang: "python3",
    code: capture.code,
  });
  assert.ok(lib.cleanCapture({ ...capture, status: "Wrong Answer" }, "two-sum"));
  assert.ok(lib.cleanCapture({ ...capture, code: "x".repeat(lib.MAX_CODE_BYTES) }, "two-sum"));
  for (const [bad, slug] of [
    [{}, "two-sum"],
    [capture, "three-sum"],
    [capture, null],
    [{ ...capture, extra: 1 }, "two-sum"],
    [{ ...capture, source: "page" }, "two-sum"],
    [{ ...capture, type: "accepted" }, "two-sum"],
    [{ ...capture, submissionId: 1234567 }, "two-sum"],
    [{ ...capture, submissionId: "12a" }, "two-sum"],
    [{ ...capture, submissionId: "1".repeat(21) }, "two-sum"],
    [{ ...capture, status: 1 }, "two-sum"],
    [{ ...capture, status: "x".repeat(65) }, "two-sum"],
    [{ ...capture, status: "Accepted\n" }, "two-sum"],
    [{ ...capture, lang: "python 3" }, "two-sum"],
    [{ ...capture, lang: "" }, "two-sum"],
    [{ ...capture, lang: "x".repeat(33) }, "two-sum"],
    [{ ...capture, code: "" }, "two-sum"],
    [{ ...capture, code: ["x"] }, "two-sum"],
    [{ ...capture, code: "a\u0000b" }, "two-sum"],
    // 64 KiB of three-byte characters is over the byte cap.
    [{ ...capture, code: "界".repeat(lib.MAX_CODE_BYTES / 2) }, "two-sum"],
  ]) {
    assert.equal(lib.cleanCapture(bad, slug), null, JSON.stringify(bad).slice(0, 120));
  }
  assert.equal(lib.cleanCapture(null, "two-sum"), null);
  assert.equal(lib.cleanCapture([capture], "two-sum"), null);
});

const fields = {
  id: "3f2c1a4e-8b9d-4c6e-9f0a-1b2c3d4e5f60",
  problemSlug: "two-sum",
  outcome: "solved",
  minutes: 18,
  assisted: false,
  notes: "",
  isReview: false,
  timeComplexity: "O(nlogn)",
  spaceComplexity: "O(n)",
};

test("buildAttempt attaches code only when kept, matching, and within the body limit", () => {
  const captured = lib.cleanCapture(capture, "two-sum");
  const withCode = lib.buildAttempt(fields, captured, true);
  assert.equal(withCode.code, capture.code);
  assert.equal(withCode.codeLanguage, "python3");
  assert.equal(withCode.timeComplexity, "O(n log n)");
  assert.equal(lib.attemptProblem(withCode), "");
  assert.deepEqual(lib.cleanAttempt(withCode), withCode);

  const optedOut = lib.buildAttempt(fields, captured, false);
  assert.equal(optedOut.code, "");
  assert.equal(optedOut.codeLanguage, "");
  assert.equal(lib.buildAttempt(fields, { ...captured, slug: "three-sum" }, true).code, "");
  assert.equal(lib.buildAttempt(fields, null, true).code, "");

  // 64 KiB of newlines escapes to 128 KiB of JSON: sent without the code.
  const huge = { ...captured, code: "\n".repeat(lib.MAX_CODE_BYTES) };
  assert.equal(lib.buildAttempt(fields, huge, true).code, "");
  const big = { ...captured, code: "x".repeat(lib.MAX_CODE_BYTES) };
  assert.equal(lib.buildAttempt(fields, big, true).code.length, lib.MAX_CODE_BYTES);
});

test("attemptProblem requires complexity for solved and struggled only", () => {
  const base = lib.buildAttempt(fields, null, false);
  assert.equal(lib.attemptProblem(base), "");
  for (const bad of [
    { timeComplexity: "" },
    { spaceComplexity: "  " },
    { outcome: "struggled", timeComplexity: "" },
  ]) {
    assert.match(lib.attemptProblem({ ...base, ...bad }), /required/, JSON.stringify(bad));
    assert.equal(lib.cleanAttempt({ ...base, ...bad }), null);
  }
  assert.match(lib.attemptProblem({ ...base, timeComplexity: "fast" }), /O\(/);
  assert.match(lib.attemptProblem({ ...base, outcome: "unfinished", timeComplexity: "n" }), /O\(/);
  const unfinished = { ...base, outcome: "unfinished", timeComplexity: "", spaceComplexity: "" };
  assert.equal(lib.attemptProblem(unfinished), "");
  assert.ok(lib.cleanAttempt(unfinished));
  assert.match(lib.attemptProblem({ ...base, code: "x", codeLanguage: "" }), /code/);
  assert.match(lib.attemptProblem({ ...base, code: "", codeLanguage: "cpp" }), /code/);
  assert.match(lib.attemptProblem({ ...base, minutes: 0 }), /Minutes/);
});

test("languageLabel and formatBytes", () => {
  assert.equal(lib.languageLabel("python3"), "Python3");
  assert.equal(lib.languageLabel("cpp"), "C++");
  assert.equal(lib.languageLabel("mysql"), "mysql");
  assert.equal(lib.languageLabel("constructor"), "constructor");
  assert.equal(lib.formatBytes(512), "512 B");
  assert.equal(lib.formatBytes(1229), "1.2 KB");
  assert.equal(lib.utf8Bytes("界"), 3);
});

test("describeStatus uses the app's message for 422 and explains 413", () => {
  assert.equal(lib.describeStatus(422, "time and space complexity are required."), "time and space complexity are required.");
  assert.match(lib.describeStatus(422, ""), /curriculum/);
  assert.match(lib.describeStatus(413, ""), /too large/);
});
