// leetcode-detect.js in a fake page: code capture for accepted and failed
// submissions, and how the content script's checks treat what it posts.
const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const lib = require("../lib.js");

function runDetector() {
  const posted = [];
  class FakeXHR {
    constructor() {
      this.listeners = {};
      this.responseType = "";
    }
    open(method, url) {
      this.method = method;
      this.url = url;
    }
    send() {}
    addEventListener(name, fn) {
      this.listeners[name] = fn;
    }
    respond(body) {
      this.responseText = JSON.stringify(body);
      if (this.listeners.load) this.listeners.load();
    }
  }
  const window = {
    location: { href: "https://leetcode.com/problems/two-sum/description/", origin: "https://leetcode.com" },
    postMessage: (msg, origin) => posted.push({ msg, origin }),
  };
  const document = { addEventListener() {}, querySelectorAll: () => [] };
  class MutationObserver {
    observe() {}
  }
  const context = vm.createContext({ window, document, MutationObserver, XMLHttpRequest: FakeXHR, URL, JSON, String, Number, Date, Promise, Set, Map, WeakSet, Element: class {}, setTimeout });
  vm.runInContext(fs.readFileSync(path.join(__dirname, "..", "leetcode-detect.js"), "utf8"), context);
  return { posted, FakeXHR };
}

// submitAndCheck submits code as LeetCode's editor does, then answers the
// submission's check with status.
function submitAndCheck(FakeXHR, { slug = "two-sum", id = 1234567, code = "class Solution: pass", lang = "python3", status }) {
  const submit = new FakeXHR();
  submit.open("POST", `/problems/${slug}/submit/`);
  submit.send(JSON.stringify({ lang, question_id: "1", typed_code: code }));
  submit.respond({ submission_id: id });
  const check = new FakeXHR();
  check.open("GET", `/submissions/detail/${id}/check/`);
  check.send();
  check.respond({ state: "SUCCESS", status_msg: status });
}

test("LeetCode: an accepted submission posts its code, then accepted", () => {
  const { posted, FakeXHR } = runDetector();
  submitAndCheck(FakeXHR, { status: "Accepted" });
  assert.deepEqual(posted.map((p) => p.msg.type), ["submission", "accepted"]);
  const [submission, accepted] = posted.map((p) => p.msg);
  assert.equal(accepted.submissionId, "submission-1234567");
  assert.deepEqual(lib.cleanCapture(submission, "two-sum"), { slug: "two-sum", submissionId: "1234567", status: "Accepted", lang: "python3", code: "class Solution: pass" });
});

test("LeetCode: failed submissions post their code without accepted", () => {
  const { posted, FakeXHR } = runDetector();
  let id = 100;
  for (const status of ["Wrong Answer", "Time Limit Exceeded", "Runtime Error", "Compile Error"]) {
    submitAndCheck(FakeXHR, { id: id++, status });
  }
  assert.deepEqual(posted.map((p) => p.msg.type), ["submission", "submission", "submission", "submission"]);
  // The nudge's unfinished panel offers the latest of them.
  const cleaned = posted.map((p) => lib.cleanCapture(p.msg, "two-sum"));
  assert.ok(cleaned.every(Boolean));
  assert.deepEqual(cleaned.map((c) => c.status), ["Wrong Answer", "Time Limit Exceeded", "Runtime Error", "Compile Error"]);
});

test("LeetCode: Run code and other problems are not captured for this one", () => {
  const { posted, FakeXHR } = runDetector();
  const run = new FakeXHR();
  run.open("POST", "/problems/two-sum/interpret_solution/");
  run.send(JSON.stringify({ lang: "python3", typed_code: "x" }));
  run.respond({ interpret_id: "runcode_1" });
  const check = new FakeXHR();
  check.open("GET", "/submissions/detail/runcode_1/check/");
  check.respond({ state: "SUCCESS", status_msg: "Accepted" });
  assert.equal(posted.length, 0);
  submitAndCheck(FakeXHR, { slug: "valid-anagram", status: "Accepted" });
  assert.equal(lib.cleanCapture(posted[0].msg, "two-sum"), null);
  assert.equal(lib.captureIssue(posted[0].msg, "two-sum"), "");
});

test("capture issues say why code was not captured", () => {
  const { posted, FakeXHR } = runDetector();
  submitAndCheck(FakeXHR, { id: 1, code: "x".repeat(lib.MAX_CODE_BYTES + 1), status: "Accepted" });
  const big = posted[0].msg;
  assert.equal(lib.cleanCapture(big, "two-sum"), null);
  assert.match(lib.captureIssue(big, "two-sum"), /over 64/);
  assert.match(lib.captureIssue({ ...big, code: "x", lang: "bad lang!" }, "two-sum"), /language/);
  assert.match(lib.captureIssue({ ...big, code: "" }, "two-sum"), /no code/);
  assert.equal(lib.captureIssue({ ...big, code: "x" }, "two-sum"), "");
  assert.match(lib.codeStatus(lib.captureIssue(big, "two-sum")), /^Code not captured: the code is over 64/);
  assert.match(lib.codeStatus(""), /^No code was captured/);
});
