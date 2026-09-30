// NeetCode support: slug mapping and the page-world detector.
const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
require("../neetcode-slugs.js");
const lib = require("../lib.js");

test("siteOf", () => {
  assert.equal(lib.siteOf("https://leetcode.com/problems/two-sum/"), "leetcode");
  assert.equal(lib.siteOf("https://neetcode.io/problems/two-integer-sum/question"), "neetcode");
  assert.equal(lib.siteOf("http://neetcode.io/"), null);
  assert.equal(lib.siteOf("https://neetcode.io.evil.example/"), null);
  assert.equal(lib.siteOf("https://neetcode.io:8443/"), null);
  assert.equal(lib.siteOf("not a url"), null);
});

test("problemFromPath maps NeetCode slugs to LeetCode", () => {
  assert.deepEqual(lib.problemFromPath("/problems/two-integer-sum/question", "neetcode"), {
    slug: "two-sum",
    metadata: { number: 1, title: "Two Sum", difficulty: "Easy", topics: [] },
  });
  assert.equal(lib.slugFromPath("/problems/duplicate-integer", "neetcode"), "contains-duplicate");
  // NeetCode-only or unknown problems are not tracked.
  assert.equal(lib.slugFromPath("/problems/not-a-real-problem/question", "neetcode"), null);
  assert.equal(lib.slugFromPath("/problems/constructor/", "neetcode"), null);
  assert.equal(lib.slugFromPath("/roadmap", "neetcode"), null);
  assert.equal(lib.neetcodeProblem("__proto__"), null);
  // LeetCode paths are unchanged.
  assert.deepEqual(lib.problemFromPath("/problems/two-sum/", "leetcode"), { slug: "two-sum", metadata: null });
});

test("isAssistPath on NeetCode", () => {
  assert.ok(lib.isAssistPath("/problems/two-integer-sum/solution", "neetcode"));
  assert.ok(!lib.isAssistPath("/problems/two-integer-sum/question", "neetcode"));
  assert.ok(!lib.isAssistPath("/problems/two-integer-sum/solutions/", "neetcode"));
});

test("every slug table entry is valid", () => {
  const table = require("../neetcode-slugs.js");
  assert.ok(Object.keys(table).length > 100);
  for (const nc of Object.keys(table)) assert.ok(lib.neetcodeProblem(nc), nc);
});

// runDetector loads neetcode-detect.js against fake XHR and fetch and
// returns the messages it posts.
function runDetector() {
  const posted = [];
  class FakeXHR {
    constructor() {
      this.listeners = {};
      this.responseType = "";
    }
    open() {}
    send() {}
    addEventListener(type, fn) {
      this.listeners[type] = fn;
    }
    respond(body) {
      this.responseText = JSON.stringify(body);
      if (this.listeners.load) this.listeners.load();
    }
  }
  const window = {
    location: { href: "https://neetcode.io/problems/two-integer-sum/question", origin: "https://neetcode.io" },
    postMessage: (msg, origin) => posted.push({ msg, origin }),
  };
  const context = vm.createContext({ window, XMLHttpRequest: FakeXHR, URL, JSON, String, Date, Promise });
  vm.runInContext(fs.readFileSync(path.join(__dirname, "..", "neetcode-detect.js"), "utf8"), context);
  return { posted, FakeXHR };
}

// Trimmed from real neetcode.io responses (2026-09-30): Submit answers one
// result object, Run an array of them.
const SUBMIT_ACCEPTED = { data: { token: "e8064668-613a-4a47-b279-0d5a206d774d", language_id: 1, status: { id: 3, description: "Accepted" }, test_case_count: 32, correct_test_case_count: 32 } };
const RUN_ACCEPTED = { data: [{ token: "8e04706e-d772-4577-bf5a-be826d193db3", status: { id: 3, description: "Accepted" }, test_case_count: 1 }] };

function submit(FakeXHR, url, sent, response) {
  const xhr = new FakeXHR();
  xhr.open("POST", url);
  xhr.send(JSON.stringify({ data: sent }));
  xhr.respond(response);
}

test("detector reports an accepted NeetCode submission", () => {
  const { posted, FakeXHR } = runDetector();
  const sent = { problemId: "two-integer-sum", rawCode: "class Solution: pass", lang: "python" };
  submit(FakeXHR, "https://neetcode.io/api/executeCodeFunctionHttp", sent, SUBMIT_ACCEPTED);
  assert.equal(posted.length, 2);
  const [submission, accepted] = posted.map((p) => p.msg);
  assert.equal(posted[0].origin, "https://neetcode.io");
  assert.equal(submission.type, "submission");
  assert.equal(accepted.type, "accepted");
  assert.equal(accepted.submissionId, `submission-${submission.submissionId}`);
  // Mapped to the LeetCode slug, the content script accepts the capture.
  const cleaned = lib.cleanCapture({ ...submission, slug: lib.neetcodeProblem(submission.slug).slug }, "two-sum");
  assert.deepEqual(cleaned, { slug: "two-sum", submissionId: submission.submissionId, status: "Accepted", lang: "python", code: "class Solution: pass" });
});

test("detector ignores runs, failures and other calls", () => {
  const { posted, FakeXHR } = runDetector();
  const sent = { problemId: "two-integer-sum", rawCode: "x", lang: "python" };
  submit(FakeXHR, "/api/executeCodeFunctionHttp", sent, { data: { status: { description: "Wrong Answer" } } });
  assert.deepEqual(posted.map((p) => p.msg.type), ["submission"]);
  submit(FakeXHR, "/api/runCodeFunctionHttp", sent, RUN_ACCEPTED);
  // A Run-shaped reply on the submit path is not a verdict either.
  submit(FakeXHR, "/api/executeCodeFunctionHttp", sent, RUN_ACCEPTED);
  submit(FakeXHR, "https://evil.example/api/executeCodeFunctionHttp", sent, { data: { status: { description: "Accepted" } } });
  submit(FakeXHR, "/api/executeCodeFunctionHttp", { problemId: 1, rawCode: "x", lang: "python" }, { data: { status: { description: "Accepted" } } });
  assert.equal(posted.length, 1);
  // "Accepted" with failing test cases is not a pass.
  submit(FakeXHR, "/api/executeCodeFunctionHttp", sent, { data: { status: { description: "Accepted" }, test_case_count: 32, correct_test_case_count: 31 } });
  assert.deepEqual(posted.slice(1).map((p) => p.msg.type), ["submission"]);
});
