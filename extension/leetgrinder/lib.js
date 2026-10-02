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
  // APPROACHES are the learner's own verdicts on their solution; "" is not stated.
  const APPROACHES = ["", "optimal", "suboptimal"];

  // Complexity rules mirror internal/leetgrinder/complexity.go; both run
  // test/complexity-vectors.json.
  const COMPLEXITIES = ["O(1)", "O(log n)", "O(√n)", "O(n)", "O(n log n)", "O(n²)", "O(n³)", "O(2ⁿ)", "O(n!)"];
  const MAX_COMPLEXITY = 40;
  const SPELLINGS = [
    ["nlogn", "n log n"],
    ["logn", "log n"],
    ["n^2", "n²"],
    ["n^3", "n³"],
    ["2^n", "2ⁿ"],
  ];
  const CONTROL = /[\u0000-\u001f\u007f-\u009f]/;
  const MAX_CODE_BYTES = 64 * 1024;
  // The app's API body limit. Payloads over it are refused before sending.
  const MAX_BODY_BYTES = 96 * 1024;
  const CODE_LANGUAGE = /^[A-Za-z0-9_+#.-]{1,32}$/;
  const SUBMISSION_ID = /^[0-9]{1,20}$/;
  const MAX_STATUS = 64;
  // Problem metadata limits mirror internal/leetgrinder/catalog.go.
  const MAX_TITLE = 200;
  const MAX_TOPICS = 20;
  const MAX_TOPIC = 60;
  const MAX_NUMBER = 100000;
  const DIFFICULTIES = ["", "Easy", "Medium", "Hard"];
  const LANGUAGE_LABELS = {
    c: "C",
    cpp: "C++",
    csharp: "C#",
    dart: "Dart",
    elixir: "Elixir",
    go: "Go",
    erlang: "Erlang",
    golang: "Go",
    java: "Java",
    javascript: "JavaScript",
    kotlin: "Kotlin",
    php: "PHP",
    python: "Python",
    python3: "Python3",
    racket: "Racket",
    ruby: "Ruby",
    rust: "Rust",
    scala: "Scala",
    swift: "Swift",
    typescript: "TypeScript",
  };

  // normalizeComplexity returns the canonical spelling of a stated
  // complexity, "" when nothing is stated, or null when it is not O(...) in
  // at most 40 characters.
  function normalizeComplexity(input) {
    if (typeof input !== "string") return null;
    let s = input
      .split(/[ \t\n\r\f\v]+/)
      .filter(Boolean)
      .join(" ");
    if (s === "") return "";
    if (s.startsWith("o(")) s = "O(" + s.slice(2);
    for (const [from, to] of SPELLINGS) s = s.split(from).join(to);
    if ([...s].length > MAX_COMPLEXITY || s.length < 4 || !s.startsWith("O(") || !s.endsWith(")") || CONTROL.test(s)) return null;
    return s;
  }

  const needsComplexity = (outcome) => outcome === "solved" || outcome === "struggled";

  function utf8Bytes(s) {
    return new TextEncoder().encode(s).length;
  }

  function validCode(code, lang) {
    if (typeof code !== "string" || typeof lang !== "string") return false;
    if (code === "") return lang === "";
    return CODE_LANGUAGE.test(lang) && !code.includes("\u0000") && utf8Bytes(code) <= MAX_CODE_BYTES;
  }

  // cleanCapture checks a page-world "submission" message. Page scripts can
  // forge these, so only exactly this shape, for the problem on screen, with
  // bounded sizes, is accepted; the learner still sees and confirms the code.
  function cleanCapture(data, slug) {
    if (!data || typeof data !== "object" || Array.isArray(data)) return null;
    if (Object.keys(data).sort().join(",") !== "code,lang,slug,source,status,submissionId,type") return null;
    if (data.source !== "leetgrinder-detect" || data.type !== "submission") return null;
    if (typeof data.slug !== "string" || data.slug !== slug || !SLUG.test(data.slug)) return null;
    if (typeof data.submissionId !== "string" || !SUBMISSION_ID.test(data.submissionId)) return null;
    if (typeof data.status !== "string" || data.status.length > MAX_STATUS || CONTROL.test(data.status)) return null;
    if (typeof data.code !== "string" || data.code === "" || !validCode(data.code, data.lang)) return null;
    return { slug: data.slug, submissionId: data.submissionId, status: data.status, lang: data.lang, code: data.code };
  }

  function languageLabel(lang) {
    return Object.prototype.hasOwnProperty.call(LANGUAGE_LABELS, lang) ? LANGUAGE_LABELS[lang] : lang;
  }

  function formatBytes(n) {
    return n < 1024 ? `${n} B` : `${(n / 1024).toFixed(1)} KB`;
  }

  // cleanMetadata returns {number, title, difficulty, topics:[{slug,name}]}
  // with only valid, bounded values, or null when the shape is wrong. The
  // app validates the same rules again.
  function cleanMetadata(m) {
    if (!m || typeof m !== "object" || Array.isArray(m)) return null;
    const title = typeof m.title === "string" ? m.title.trim() : "";
    const number = m.number === undefined ? 0 : m.number;
    const difficulty = m.difficulty === undefined ? "" : m.difficulty;
    if (!Number.isInteger(number) || number < 0 || number > MAX_NUMBER) return null;
    if ([...title].length > MAX_TITLE || CONTROL.test(title)) return null;
    if (!DIFFICULTIES.includes(difficulty)) return null;
    const topics = [];
    const seen = new Set();
    for (const t of Array.isArray(m.topics) ? m.topics : []) {
      if (!t || typeof t.slug !== "string" || typeof t.name !== "string") return null;
      const name = t.name.trim();
      if (!SLUG.test(t.slug) || t.slug.length > MAX_TOPIC || [...name].length > MAX_TOPIC || CONTROL.test(name)) return null;
      if (!seen.has(t.slug)) {
        seen.add(t.slug);
        topics.push({ slug: t.slug, name });
      }
    }
    if (topics.length > MAX_TOPICS) return null;
    return { number, title, difficulty, topics };
  }

  // metadataFromGraphQL reads LeetCode's question(titleSlug) response, or
  // returns null when it names no problem or has an unexpected shape.
  function metadataFromGraphQL(json) {
    const q = json && json.data && json.data.question;
    if (!q || typeof q !== "object") return null;
    const id = /^[0-9]{1,6}$/.test(String(q.questionFrontendId || "")) ? Number(q.questionFrontendId) : 0;
    const topics = Array.isArray(q.topicTags) ? q.topicTags.map((t) => ({ slug: t && t.slug, name: t && t.name })) : [];
    const m = cleanMetadata({ number: id, title: q.title, difficulty: q.difficulty, topics });
    return m && m.title ? m : null;
  }

  // GRAPHQL_QUERY asks LeetCode for one problem's metadata.
  const GRAPHQL_QUERY = "query questionData($titleSlug: String!) { question(titleSlug: $titleSlug) { questionFrontendId title difficulty topicTags { slug name } } }";

  // attemptProblem explains why an attempt cannot be sent, or returns "".
  function attemptProblem(a) {
    if (!a || typeof a !== "object") return "Invalid attempt.";
    if (typeof a.id !== "string" || !UUID.test(a.id)) return "Invalid attempt id.";
    if (typeof a.problemSlug !== "string" || !SLUG.test(a.problemSlug) || a.problemSlug.length > 100) return "Invalid problem.";
    if (!OUTCOMES.includes(a.outcome)) return "Choose an outcome.";
    if (!Number.isInteger(a.minutes) || a.minutes < 1 || a.minutes > MAX_MINUTES) return `Minutes must be a whole number from 1 to ${MAX_MINUTES}.`;
    if (typeof a.assisted !== "boolean" || typeof a.isReview !== "boolean") return "Invalid attempt.";
    // Both are optional, so the payload works with servers that predate them.
    if ((a.wantsReview !== undefined && typeof a.wantsReview !== "boolean") || (a.approach !== undefined && !APPROACHES.includes(a.approach))) return "Invalid attempt.";
    if (typeof a.notes !== "string" || [...a.notes].length > MAX_NOTES || a.notes.includes("\u0000")) return `Keep notes to ${MAX_NOTES} characters or fewer.`;
    const time = normalizeComplexity(a.timeComplexity);
    const space = normalizeComplexity(a.spaceComplexity);
    if (time === null || space === null) return "Write complexity like O(m·n): start with O( and end with ), in 40 characters or fewer.";
    if (needsComplexity(a.outcome) && (time === "" || space === "")) return "Choose the time and space complexity. Both are required for solved and struggled attempts.";
    if (!validCode(a.code, a.codeLanguage)) return "The captured code is invalid or over 64 KB. Leave it out and try again.";
    if (a.problem !== undefined && !cleanMetadata(a.problem)) return "Invalid problem details.";
    return "";
  }

  function bodyTooLarge(a) {
    return utf8Bytes(JSON.stringify(a)) > MAX_BODY_BYTES;
  }

  // buildAttempt assembles the API payload from the panel's fields. The code
  // is attached only when the learner kept it, it belongs to this problem,
  // and the whole body fits the API's limit.
  function buildAttempt(fields, capture, includeCode) {
    const a = {
      id: fields.id,
      problemSlug: fields.problemSlug,
      outcome: fields.outcome,
      minutes: fields.minutes,
      assisted: fields.assisted,
      notes: fields.notes,
      isReview: fields.isReview,
      timeComplexity: normalizeComplexity(fields.timeComplexity) ?? fields.timeComplexity,
      spaceComplexity: normalizeComplexity(fields.spaceComplexity) ?? fields.spaceComplexity,
      code: "",
      codeLanguage: "",
    };
    // Unset self-assessments are left out, so servers without them still
    // accept the attempt.
    if (fields.wantsReview) a.wantsReview = true;
    if (fields.approach) a.approach = fields.approach;
    const meta = cleanMetadata(fields.problem);
    if (meta) a.problem = meta;
    if (includeCode && capture && capture.slug === fields.problemSlug) {
      const withCode = { ...a, code: capture.code, codeLanguage: capture.lang };
      if (!bodyTooLarge(withCode)) return withCode;
    }
    return a;
  }

  // siteOf names the problem site a page URL belongs to: "leetcode",
  // "neetcode", or null.
  function siteOf(url) {
    let u;
    try {
      u = new URL(String(url || ""));
    } catch {
      return null;
    }
    if (u.protocol !== "https:" || u.port) return null;
    if (u.hostname === "leetcode.com") return "leetcode";
    if (u.hostname === "neetcode.io") return "neetcode";
    return null;
  }

  function pathSlug(path) {
    const m = /^\/problems\/([^/]+)(?:\/|$)/.exec(path || "");
    return m && SLUG.test(m[1]) && m[1].length <= 100 ? m[1] : null;
  }

  // neetcodeProblem maps a NeetCode problem slug to the LeetCode problem it
  // mirrors: {slug, metadata}, or null for NeetCode-only problems. The app
  // only knows LeetCode slugs. The table is neetcode-slugs.js.
  function neetcodeProblem(ncSlug) {
    const table = root.LeetgrinderNeetCodeSlugs;
    if (!table || typeof ncSlug !== "string" || !Object.prototype.hasOwnProperty.call(table, ncSlug)) return null;
    const [slug, number, title, difficulty] = table[ncSlug];
    const metadata = cleanMetadata({ number, title, difficulty, topics: [] });
    return validSlug(slug) && metadata ? { slug, metadata } : null;
  }

  // problemFromPath returns the problem at a URL path on the given site as
  // {slug, metadata}: its LeetCode slug, and on NeetCode the metadata known
  // from the slug table (null on LeetCode, which is asked instead).
  function problemFromPath(path, site = "leetcode") {
    const slug = pathSlug(path);
    if (!slug) return null;
    if (site === "leetcode") return { slug, metadata: null };
    return site === "neetcode" ? neetcodeProblem(slug) : null;
  }

  // slugFromPath returns the LeetCode slug of the problem at a URL path.
  function slugFromPath(path, site = "leetcode") {
    const p = problemFromPath(path, site);
    return p ? p.slug : null;
  }

  // isAssistPath is true on a problem's Solutions or Editorial tab
  // (LeetCode), or its Solution tab (NeetCode).
  function isAssistPath(path, site = "leetcode") {
    const tabs = site === "neetcode" ? /^\/problems\/[^/]+\/solution(?:\/|$)/ : /^\/problems\/[^/]+\/(?:solutions|editorial)(?:\/|$)/;
    return tabs.test(path || "");
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

  // cleanAttempt returns a copy with only the API's fields and normalised
  // complexities, or null when any field is invalid or the body would be
  // over the API's limit. The background worker never forwards anything else.
  function cleanAttempt(a) {
    if (!a || typeof a !== "object") return null;
    const text = (v) => (v === undefined ? "" : v);
    const out = {
      id: a.id,
      problemSlug: a.problemSlug,
      outcome: a.outcome,
      minutes: a.minutes,
      assisted: a.assisted,
      notes: a.notes,
      isReview: a.isReview,
      timeComplexity: text(a.timeComplexity),
      spaceComplexity: text(a.spaceComplexity),
      code: text(a.code),
      codeLanguage: text(a.codeLanguage),
    };
    if (a.problem !== undefined) out.problem = a.problem;
    if (a.wantsReview !== undefined && a.wantsReview !== false) out.wantsReview = a.wantsReview;
    if (a.approach !== undefined && a.approach !== "") out.approach = a.approach;
    if (attemptProblem(out)) return null;
    if (out.problem !== undefined) out.problem = cleanMetadata(out.problem);
    out.timeComplexity = normalizeComplexity(out.timeComplexity);
    out.spaceComplexity = normalizeComplexity(out.spaceComplexity);
    return bodyTooLarge(out) ? null : out;
  }

  // describeStatus turns an API result into a message for the learner.
  function describeStatus(status, error) {
    switch (status) {
      case 0:
        return error || "Could not reach the Leetgrinder app.";
      case 401:
        return "The app rejected the API token. Check the extension options.";
      case 409:
        return "Saved earlier with different values. Check the history, or save this as a new attempt.";
      case 413:
        return "The attempt is too large to save. Leave the code out and try again.";
      case 422:
        return error || "The app could not accept this attempt.";
      default:
        return error || `The app answered with status ${status}.`;
    }
  }

  // ---- Badge, banner and popup (pure, so they can be tested) -------------

  const OUTCOME_LABELS = { solved: "Solved", struggled: "Struggled", unfinished: "Unfinished" };
  const KIND_LABELS = { new: "New", review: "Review", practice: "Practice" };
  const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

  const DISCARD_PROMPT = "This attempt may not be saved yet. Discard it?";

  // panelCloses decides whether a close request ends the log panel. A locked
  // panel holds an attempt that may not be saved: Escape never closes it and
  // Dismiss needs the learner's confirmation, which confirmDiscard supplies.
  function panelCloses(trigger, locked, confirmDiscard) {
    if (!locked) return true;
    if (trigger === "escape") return false;
    return Boolean(confirmDiscard && confirmDiscard(DISCARD_PROMPT));
  }

  function kindLabel(kind) {
    return Object.prototype.hasOwnProperty.call(KIND_LABELS, kind) ? KIND_LABELS[kind] : "";
  }

  function percent(f) {
    return `${Math.round((Number(f) || 0) * 100)}%`;
  }

  // shortDate turns "2026-10-14" into "Oct 14".
  function shortDate(iso) {
    const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(String(iso || ""));
    return m ? `${MONTHS[Number(m[2]) - 1]} ${Number(m[3])}` : "";
  }

  // daysAgo counts whole days between an ISO instant and now, both in local time.
  function daysAgo(iso, now) {
    const then = new Date(iso);
    if (Number.isNaN(then.getTime())) return null;
    const start = (d) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
    return Math.max(0, Math.round((start(new Date(now)) - start(then)) / 86400000));
  }

  // badgeState is the toolbar badge for a /today response, or for an error
  // when today is null.
  function badgeState(today) {
    if (!today || typeof today !== "object") {
      return { text: "?", color: "#71717a", title: "Leetgrinder: not connected. Check the extension options." };
    }
    if (today.met) return { text: "✓", color: "#16a34a", title: "Leetgrinder: today's goal is met." };
    const n = Math.max(0, Number(today.remaining) || 0);
    return { text: String(n), color: "#2563eb", title: `Leetgrinder: ${n} left for today's goal.` };
  }

  // bannerState is the on-page pill for a problem status response.
  function bannerState(info, now) {
    if (!info || typeof info !== "object") return null;
    let label;
    if (info.todaysPick && info.attemptedToday) label = "Today's review · done";
    else if (info.todaysPick) label = "Today's review";
    else if (info.status === "due" && info.flagReason) label = info.flagReason;
    else if (info.status === "due") label = `Review due · recall ${percent(info.recall)}`;
    else if (info.status === "notDue") {
      const days = daysAgo(info.lastAttemptedAt, now);
      const when = days === null ? "Reviewed" : days === 0 ? "Reviewed today" : `Reviewed ${days} d ago`;
      label = info.nextDue ? `${when} · next due ${shortDate(info.nextDue)}` : when;
    } else label = "New";
    return { label, summary: attemptSummary(info.latestAttempt) };
  }

  // attemptSummary reads "Struggled · 32 min · O(n log n)/O(n)".
  function attemptSummary(a) {
    if (!a || typeof a !== "object") return "";
    const parts = [OUTCOME_LABELS[a.outcome] || "Attempt"];
    if (Number.isInteger(a.minutes)) parts.push(`${a.minutes} min`);
    if (a.timeComplexity || a.spaceComplexity) parts.push(`${a.timeComplexity || "?"}/${a.spaceComplexity || "?"}`);
    return parts.join(" · ");
  }

  // popupModel shapes a /today response for the popup: progress lines, the
  // picks and the first ten due problems with LeetCode links.
  function popupModel(today, origin) {
    const item = (r) => ({
      title: String(r.title || r.slug),
      url: `https://leetcode.com/problems/${encodeURIComponent(r.slug)}/`,
      reason: String(r.reason || ""),
      done: Boolean(r.done),
    });
    const goal = today.goal || {};
    const done = today.done || {};
    return {
      status: today.met ? "Goal met" : `${Number(today.remaining) || 0} to go`,
      progress: [`New ${done.new || 0}/${goal.new || 0}`, `Review ${done.review || 0}/${goal.review || 0}`, `Bonus ${done.bonus || 0}`],
      streak: `Streak ${today.streak || 0} ${today.streak === 1 ? "day" : "days"}`,
      picks: (Array.isArray(today.picks) ? today.picks : []).filter((r) => r && validSlug(r.slug)).map(item),
      due: (Array.isArray(today.due) ? today.due : []).filter((r) => r && validSlug(r.slug)).slice(0, 10).map(item),
      dueCount: Number(today.dueCount) || 0,
      dashboard: origin ? `${origin}/leetgrinder` : "",
    };
  }

  function validSlug(s) {
    return typeof s === "string" && SLUG.test(s) && s.length <= 100;
  }

  const lib = {
    NUDGE_MINUTES,
    MAX_MINUTES,
    MAX_NOTES,
    siteOf,
    neetcodeProblem,
    problemFromPath,
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
    COMPLEXITIES,
    MAX_COMPLEXITY,
    MAX_CODE_BYTES,
    MAX_BODY_BYTES,
    normalizeComplexity,
    needsComplexity,
    utf8Bytes,
    cleanCapture,
    languageLabel,
    formatBytes,
    attemptProblem,
    buildAttempt,
    cleanMetadata,
    metadataFromGraphQL,
    GRAPHQL_QUERY,
    validSlug,
    kindLabel,
    panelCloses,
    DISCARD_PROMPT,
    APPROACHES,
    shortDate,
    daysAgo,
    badgeState,
    bannerState,
    attemptSummary,
    popupModel,
  };
  root.LeetgrinderLib = lib;
  if (typeof module === "object" && module.exports) module.exports = lib;
})(globalThis);
