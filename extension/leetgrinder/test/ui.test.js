// Badge, banner and popup helpers, from fixture API responses.
const test = require("node:test");
const assert = require("node:assert/strict");
const lib = require("../lib.js");
const today = require("./fixtures/today.json");

test("badgeState", () => {
  assert.deepEqual(lib.badgeState(today), { text: "2", color: "#2563eb", title: "Leetgrinder: 2 left for today's goal." });
  assert.equal(lib.badgeState({ ...today, met: true, remaining: 0 }).text, "✓");
  const off = lib.badgeState(null);
  assert.equal(off.text, "?");
  assert.match(off.title, /not connected/);
});

test("bannerState", () => {
  const now = new Date(2026, 9, 14, 12).getTime();
  const latest = { outcome: "struggled", minutes: 32, timeComplexity: "O(n log n)", spaceComplexity: "O(n)" };
  assert.deepEqual(lib.bannerState({ status: "new", latestAttempt: null }, now), { label: "New", summary: "" });
  assert.equal(lib.bannerState({ status: "due", recall: 0.62, latestAttempt: latest }, now).label, "Review due · recall 62%");
  assert.equal(lib.bannerState({ status: "due", todaysPick: true, recall: 0.62 }, now).label, "Today's review");
  assert.equal(lib.bannerState({ status: "due", flagReason: "Flagged: time complexity judged wrong" }, now).label, "Flagged: time complexity judged wrong");
  const notDue = lib.bannerState({ status: "notDue", nextDue: "2026-10-17", lastAttemptedAt: new Date(2026, 9, 11, 20).toISOString(), latestAttempt: latest }, now);
  assert.deepEqual(notDue, { label: "Reviewed 3 d ago · next due Oct 17", summary: "Struggled · 32 min · O(n log n)/O(n)" });
  assert.equal(lib.bannerState(null, now), null);
  assert.equal(lib.attemptSummary({ outcome: "unfinished", minutes: 25 }), "Unfinished · 25 min");
});

test("kindLabel and shortDate", () => {
  assert.equal(lib.kindLabel("review"), "Review");
  assert.equal(lib.kindLabel("practice"), "Practice");
  assert.equal(lib.kindLabel("constructor"), "");
  assert.equal(lib.shortDate("2026-01-05"), "Jan 5");
  assert.equal(lib.shortDate("nope"), "");
});

test("popupModel", () => {
  const model = lib.popupModel(today, "https://career.example.com");
  assert.equal(model.status, "2 to go");
  assert.deepEqual(model.progress, ["New 1/2", "Review 0/1", "Bonus 1"]);
  assert.equal(model.streak, "Streak 5 days");
  assert.deepEqual(model.picks, [{ title: "Two Sum", url: "https://leetcode.com/problems/two-sum/", reason: "Struggled 9 days ago · recall estimate 62%", done: false }]);
  // Entries with invalid slugs are dropped.
  assert.equal(model.due.length, 1);
  assert.equal(model.due[0].url, "https://leetcode.com/problems/lru-cache/");
  assert.equal(model.dashboard, "https://career.example.com/leetgrinder");
  assert.equal(lib.popupModel({ ...today, met: true, streak: 1 }, "").status, "Goal met");
  assert.equal(lib.popupModel({ ...today, streak: 1 }, "").streak, "Streak 1 day");
  const many = { ...today, due: Array.from({ length: 20 }, (_, i) => ({ slug: `p-${i}`, title: `P${i}` })) };
  assert.equal(lib.popupModel(many, "").due.length, 10);
});
