// The persistent outbox: attempts survive a closed tab, a reload and app
// downtime, and resend until the app accepts them.
const test = require("node:test");
const assert = require("node:assert/strict");
const lib = require("../lib.js");

// fakeStorage mimics ext.storage.local: values are copied in and out, so
// nothing is shared by reference between readers.
function fakeStorage() {
  const data = {};
  return {
    data,
    async get(key) {
      return key in data ? { [key]: structuredClone(data[key]) } : {};
    },
    async set(items) {
      for (const [k, v] of Object.entries(items)) data[k] = structuredClone(v);
    },
  };
}

// fakeApp answers posts from a script of results, recording each attempt.
function fakeApp(...answers) {
  const sent = [];
  const post = async (attempt) => {
    sent.push(attempt.id);
    const answer = answers.length > 1 ? answers.shift() : answers[0];
    return typeof answer === "function" ? answer(attempt) : answer;
  };
  return { sent, post };
}

const OK = { ok: true, status: 201, data: { kind: "new" } };
const DOWN = { ok: false, status: 0, error: "Could not reach the Leetgrinder app." };
const attempt = (n, extra = {}) => ({ id: `00000000-0000-4000-8000-00000000000${n}`, problemSlug: "two-sum", outcome: "solved", minutes: 20, ...extra });

test("an attempt is stored before it is sent, and removed once accepted", async () => {
  const storage = fakeStorage();
  const app = fakeApp((a) => {
    // A worker killed mid-request must still find the attempt.
    assert.deepEqual(storage.data.outbox[a.id].attempt, a);
    return OK;
  });
  const outbox = lib.createOutbox(storage, app.post);
  assert.deepEqual(await outbox.submit(attempt(1)), OK);
  assert.deepEqual(await outbox.summary(), { pending: 0, lastError: "", failed: [] });
});

test("an unsent attempt survives a worker restart and syncs later", async () => {
  const storage = fakeStorage();
  const res = await lib.createOutbox(storage, fakeApp(DOWN).post).submit(attempt(1));
  assert.equal(res.queued, true);
  assert.equal(res.status, 0);
  // The tab is closed and the worker restarts: a new outbox over the same
  // storage still holds the attempt.
  const app = fakeApp(OK);
  const restarted = lib.createOutbox(storage, app.post);
  assert.deepEqual(await restarted.summary(), { pending: 1, lastError: DOWN.error, failed: [] });
  assert.equal(await restarted.flush(), 1);
  assert.deepEqual(app.sent, [attempt(1).id]);
  assert.equal((await restarted.summary()).pending, 0);
});

test("retryable answers keep the attempt; final ones are returned and dropped", async () => {
  for (const status of [0, 401, 403, 408, 429, 500, 502, 503]) {
    assert.equal(lib.outboxRetryable(status), true, String(status));
  }
  for (const status of [400, 404, 409, 413, 415, 422]) {
    assert.equal(lib.outboxRetryable(status), false, String(status));
  }
  const storage = fakeStorage();
  const outbox = lib.createOutbox(storage, fakeApp({ ok: false, status: 503, error: "" }).post);
  const queued = await outbox.submit(attempt(1));
  assert.equal(queued.queued, true);
  assert.equal((await outbox.summary()).lastError, "The app answered with status 503.");
  // The panel handles a final answer itself (a 409 offers a new id), so the
  // outbox does not keep it.
  const conflict = { ok: false, status: 409, error: "Saved earlier with different values." };
  const res = await lib.createOutbox(storage, fakeApp(conflict).post).submit(attempt(2));
  assert.equal(res.queued, undefined);
  assert.equal(res.status, 409);
  const after = await outbox.summary();
  assert.equal(after.pending, 1);
  assert.deepEqual(after.failed, []);
});

test("flush resends oldest first, stops while the app is unreachable, and records tries", async () => {
  const storage = fakeStorage();
  // Stored in a different order from when they were queued.
  const clocks = { 1: 2000, 2: 3000, 3: 1000 };
  for (const n of [1, 2, 3]) {
    await lib.createOutbox(storage, fakeApp(DOWN).post, () => clocks[n]).submit(attempt(n));
  }
  const app = fakeApp(DOWN);
  assert.equal(await lib.createOutbox(storage, app.post).flush(), 0);
  // An unreachable app fails every attempt the same way, so one try is enough.
  assert.deepEqual(app.sent, [attempt(3).id]);
  assert.equal(storage.data.outbox[attempt(3).id].tries, 2);
  assert.equal(storage.data.outbox[attempt(1).id].tries, 1);
  const back = fakeApp(OK);
  assert.equal(await lib.createOutbox(storage, back.post).flush(), 3);
  assert.deepEqual(back.sent, [attempt(3).id, attempt(1).id, attempt(2).id]);
});

test("an unreachable app, rejected token or refused origin stops the pass; other failures do not", async () => {
  for (const stop of [DOWN, { ok: false, status: 401, error: "" }, { ok: false, status: 403, error: "" }]) {
    const storage = fakeStorage();
    let clock = 0;
    for (const n of [1, 2, 3]) await lib.createOutbox(storage, fakeApp(DOWN).post, () => clock++).submit(attempt(n));
    const app = fakeApp(stop);
    await lib.createOutbox(storage, app.post).flush();
    assert.deepEqual(app.sent, [attempt(1).id], String(stop.status));
  }
  const storage = fakeStorage();
  let clock = 0;
  for (const n of [1, 2, 3]) await lib.createOutbox(storage, fakeApp(DOWN).post, () => clock++).submit(attempt(n));
  // A 503 or 429 may be about one attempt, so the rest are still tried.
  const app = fakeApp({ ok: false, status: 503, error: "" }, { ok: false, status: 429, error: "" }, OK);
  assert.equal(await lib.createOutbox(storage, app.post).flush(), 1);
  assert.deepEqual(app.sent, [attempt(1).id, attempt(2).id, attempt(3).id]);
  assert.equal((await lib.createOutbox(storage, app.post).summary()).pending, 2);
  const final = fakeApp({ ok: false, status: 422, error: "No." }, OK);
  assert.equal(await lib.createOutbox(storage, final.post).flush(), 1);
  assert.deepEqual(final.sent, [attempt(1).id, attempt(2).id]);
});

test("a 409 on a resend means the app already saved the attempt", async () => {
  const storage = fakeStorage();
  await lib.createOutbox(storage, fakeApp(DOWN).post).submit(attempt(1));
  const outbox = lib.createOutbox(storage, fakeApp({ ok: false, status: 409, error: "Saved earlier." }).post);
  await outbox.flush();
  assert.deepEqual(await outbox.summary(), { pending: 0, lastError: "", failed: [] });
});

test("a storage failure still sends the attempt, without queuing it", async () => {
  const broken = { get: async () => ({}), set: async () => { throw new Error("QUOTA_BYTES quota exceeded"); } };
  const app = fakeApp(OK);
  const error = console.error;
  console.error = () => {};
  try {
    assert.deepEqual(await lib.createOutbox(broken, app.post).submit(attempt(1)), OK);
    const down = await lib.createOutbox(broken, fakeApp(DOWN).post).submit(attempt(2));
    assert.equal(down.queued, undefined);
  } finally {
    console.error = error;
  }
  assert.deepEqual(app.sent, [attempt(1).id]);
});

test("the last error shown is the oldest waiting attempt's, which a pass tries first", async () => {
  const storage = fakeStorage();
  // Stored oldest last, so the summary must sort by queue time.
  let clock = 100;
  for (const n of [1, 2]) await lib.createOutbox(storage, fakeApp({ ok: false, status: 401, error: "Bad token." }).post, () => clock--).submit(attempt(n));
  await lib.createOutbox(storage, fakeApp(DOWN).post).flush();
  assert.equal((await lib.createOutbox(storage, fakeApp(DOWN).post).summary()).lastError, DOWN.error);
});

test("an attempt rejected during a flush stays as failed until discarded", async () => {
  const storage = fakeStorage();
  await lib.createOutbox(storage, fakeApp(DOWN).post).submit(attempt(1, { problem: { title: "Two Sum" } }));
  const outbox = lib.createOutbox(storage, fakeApp({ ok: false, status: 422, error: "Time complexity is required." }).post);
  assert.equal(await outbox.flush(), 0);
  assert.deepEqual(await outbox.summary(), { pending: 0, lastError: "", failed: [{ id: attempt(1).id, slug: "two-sum", title: "Two Sum", error: "Time complexity is required." }] });
  // A failed attempt is not resent.
  const app = fakeApp(OK);
  assert.equal(await lib.createOutbox(storage, app.post).flush(), 0);
  assert.deepEqual(app.sent, []);
  await outbox.discard(attempt(1).id);
  assert.deepEqual((await outbox.summary()).failed, []);
});

test("discard keeps attempts still waiting", async () => {
  const storage = fakeStorage();
  const outbox = lib.createOutbox(storage, fakeApp(DOWN).post);
  await outbox.submit(attempt(1));
  await outbox.discard(attempt(1).id);
  assert.equal((await outbox.summary()).pending, 1);
});

test("concurrent flushes share one pass and concurrent submits keep every attempt", async () => {
  const storage = fakeStorage();
  let release;
  const gate = new Promise((r) => (release = r));
  const app = fakeApp(async () => {
    await gate;
    return OK;
  });
  const waiting = lib.createOutbox(storage, fakeApp(DOWN).post);
  await Promise.all([1, 2, 3].map((n) => waiting.submit(attempt(n))));
  assert.equal((await waiting.summary()).pending, 3);
  const outbox = lib.createOutbox(storage, app.post);
  const first = outbox.flush();
  const second = outbox.flush();
  assert.equal(first, second);
  release();
  assert.equal(await first, 3);
  assert.equal(app.sent.length, 3);
});

test("a resubmitted id replaces its entry; a full outbox refuses new attempts", async () => {
  const storage = fakeStorage();
  let clock = 100;
  const outbox = lib.createOutbox(storage, fakeApp(DOWN).post, () => clock++);
  await outbox.submit(attempt(1));
  await outbox.submit(attempt(1));
  assert.equal((await outbox.summary()).pending, 1);
  // Retry now keeps the queue position and starts counting tries again.
  assert.equal(storage.data.outbox[attempt(1).id].queuedAt, 100);
  assert.equal(storage.data.outbox[attempt(1).id].tries, 1);
  // Resubmitting an attempt marked failed clears the mark.
  storage.data.outbox[attempt(1).id].failed = "No.";
  await outbox.submit(attempt(1));
  assert.equal(storage.data.outbox[attempt(1).id].failed, undefined);
  for (let i = 1; i < lib.MAX_OUTBOX; i++) {
    await outbox.submit({ ...attempt(1), id: `10000000-0000-4000-8000-${String(i).padStart(12, "0")}` });
  }
  assert.equal((await outbox.summary()).pending, lib.MAX_OUTBOX);
  const full = await outbox.submit(attempt(2));
  assert.equal(full.ok, false);
  assert.equal(full.queued, undefined);
  assert.match(full.error, /already waiting to sync/);
  // Retrying one already waiting still works.
  assert.equal((await outbox.submit(attempt(1))).queued, true);
  // A rejected attempt sent again counts as new, so it cannot exceed the cap.
  storage.data.outbox[attempt(1).id].failed = "No.";
  for (let i = 1; i < lib.MAX_OUTBOX; i++) storage.data.outbox[`10000000-0000-4000-8000-${String(i).padStart(12, "0")}`].failed = undefined;
  storage.data.outbox[attempt(3).id] = { attempt: attempt(3), queuedAt: 1, tries: 0 };
  assert.match((await outbox.submit(attempt(1))).error, /already waiting to sync/);
  // Rejected attempts do not count toward the cap.
  delete storage.data.outbox[attempt(3).id];
  assert.equal((await outbox.submit(attempt(2))).queued, true);
});

test("outboxLines, queuedMessage and validAttemptId", () => {
  assert.deepEqual(lib.outboxLines({ pending: 0, failed: [] }), []);
  assert.deepEqual(lib.outboxLines({ pending: 1, lastError: "" }), ["1 attempt waiting to sync."]);
  assert.deepEqual(lib.outboxLines({ pending: 2, lastError: "Could not reach the Leetgrinder app." }), ["2 attempts waiting to sync. Last try: Could not reach the Leetgrinder app."]);
  assert.deepEqual(lib.outboxLines(undefined), []);
  assert.match(lib.queuedMessage(DOWN), /^Could not reach the Leetgrinder app\. The attempt is saved in the extension and syncs automatically/);
  assert.ok(lib.validAttemptId(attempt(1).id));
  assert.ok(!lib.validAttemptId("../x"));
});
