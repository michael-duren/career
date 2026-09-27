// Pure helpers shared by the background worker, content script, and options
// page. No browser APIs here so `node --test` can exercise them.
(function (root) {
  "use strict";

  const NUDGE_MINUTES = 25;
  const MAX_MINUTES = 240;
  const MAX_NOTES = 2000;
  const SLUG = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
  const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
  const TOKEN = /^lg_[A-Za-z0-9_-]{43}$/;
  const OUTCOMES = ["solved", "struggled", "unfinished"];

  // slugFromPath returns the problem slug for a LeetCode problem URL path.
  function slugFromPath(path) {
    const m = /^\/problems\/([^/]+)(?:\/|$)/.exec(path || "");
    return m && SLUG.test(m[1]) && m[1].length <= 100 ? m[1] : null;
  }

  // isAssistPath is true on a problem's Solutions or Editorial tab.
  function isAssistPath(path) {
    return /^\/problems\/[^/]+\/(?:solutions|editorial)(?:\/|$)/.test(path || "");
  }

  // elapsedMinutes rounds a timer to whole minutes within the API's range.
  function elapsedMinutes(startedAt, now) {
    const minutes = Math.round((now - startedAt) / 60000);
    if (!Number.isFinite(minutes)) return 1;
    return Math.min(MAX_MINUTES, Math.max(1, minutes));
  }

  // inferOutcome prefills the confirm panel after an Accepted submission.
  function inferOutcome(minutes) {
    return minutes <= NUDGE_MINUTES ? "solved" : "struggled";
  }

  // Open problem pages refresh lastSeenAt every 30 seconds. A timer whose
  // page has been gone this long, or that is older than a sitting, belongs to
  // an earlier visit and restarts.
  const TIMER_IDLE_MS = 30 * 60000;
  const TIMER_MAX_AGE_MS = 12 * 3600000;

  function timerExpired(timer, now) {
    if (!timer || !Number.isFinite(timer.startedAt)) return true;
    const seen = Number.isFinite(timer.lastSeenAt) ? timer.lastSeenAt : timer.startedAt;
    return now - seen > TIMER_IDLE_MS || now - timer.startedAt > TIMER_MAX_AGE_MS;
  }

  function shouldNudge(timer, now) {
    return Boolean(timer) && !timer.nudged && now - timer.startedAt >= NUDGE_MINUTES * 60000;
  }

  // normalizeOrigin accepts https origins, or plain http only on loopback,
  // so the API token never crosses the network unencrypted.
  function normalizeOrigin(input) {
    let url;
    try {
      url = new URL(String(input || "").trim());
    } catch {
      return null;
    }
    const loopback = ["localhost", "127.0.0.1"].includes(url.hostname);
    if (url.protocol !== "https:" && !(url.protocol === "http:" && loopback)) return null;
    if (url.username || url.password) return null;
    return url.origin;
  }

  // originPattern is the host permission match pattern for an origin.
  // Match patterns cannot carry ports and match every port on the host.
  function originPattern(origin) {
    const url = new URL(origin);
    return `${url.protocol}//${url.hostname}/*`;
  }

  function validToken(token) {
    return TOKEN.test(String(token || ""));
  }

  // cleanAttempt returns a copy with only the API's fields, or null when any
  // field is invalid. The background worker never forwards anything else.
  function cleanAttempt(a) {
    if (!a || typeof a !== "object") return null;
    const out = {
      id: a.id,
      problemSlug: a.problemSlug,
      outcome: a.outcome,
      minutes: a.minutes,
      assisted: a.assisted,
      notes: a.notes,
      isReview: a.isReview,
    };
    if (typeof out.id !== "string" || !UUID.test(out.id)) return null;
    if (typeof out.problemSlug !== "string" || !SLUG.test(out.problemSlug) || out.problemSlug.length > 100) return null;
    if (!OUTCOMES.includes(out.outcome)) return null;
    if (!Number.isInteger(out.minutes) || out.minutes < 1 || out.minutes > MAX_MINUTES) return null;
    if (typeof out.assisted !== "boolean" || typeof out.isReview !== "boolean") return null;
    if (typeof out.notes !== "string" || [...out.notes].length > MAX_NOTES || out.notes.includes("\u0000")) return null;
    return out;
  }

  // describeStatus turns an API result into a message for the learner.
  function describeStatus(status, error) {
    switch (status) {
      case 0:
        return error || "Could not reach the Leetgrinder app.";
      case 401:
        return "The app rejected the API token. Check the extension options.";
      case 409:
        return "This attempt was already saved with different values. Correct it in the app.";
      case 422:
        return "This problem is not in the Leetgrinder curriculum.";
      default:
        return error || `The app answered with status ${status}.`;
    }
  }

  const lib = {
    NUDGE_MINUTES,
    MAX_MINUTES,
    MAX_NOTES,
    slugFromPath,
    isAssistPath,
    elapsedMinutes,
    inferOutcome,
    shouldNudge,
    timerExpired,
    normalizeOrigin,
    originPattern,
    validToken,
    cleanAttempt,
    describeStatus,
    validSlug: (s) => typeof s === "string" && SLUG.test(s) && s.length <= 100,
  };
  root.LeetgrinderLib = lib;
  if (typeof module === "object" && module.exports) module.exports = lib;
})(globalThis);
