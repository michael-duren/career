// Background worker: the only code that holds the API token and talks to the
// Leetgrinder app. Chrome runs this file as a service worker; Firefox runs it
// as an event page after browser-shim.js and lib.js (see manifest.json).
if (typeof importScripts === "function") importScripts("browser-shim.js", "lib.js");

const lib = globalThis.LeetgrinderLib;
const REQUEST_TIMEOUT_MS = 15000;

async function config() {
  const { origin, token } = await ext.storage.local.get(["origin", "token"]);
  const normalized = lib.normalizeOrigin(origin);
  if (!normalized || !lib.validToken(token)) {
    return { error: "Set the app origin and API token in the extension options." };
  }
  const granted = await ext.permissions.contains({ origins: [lib.originPattern(normalized)] });
  if (!granted) {
    return { error: "Allow access to the app origin in the extension options." };
  }
  return { origin: normalized, token };
}

// api calls the app and resolves to {ok, status, data?, error?}. It never
// rejects, and never sends cookies or follows redirects.
async function api(method, path, body) {
  const cfg = await config();
  if (cfg.error) return { ok: false, status: 0, error: cfg.error };
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  try {
    const res = await fetch(cfg.origin + path, {
      method,
      headers: {
        Authorization: `Bearer ${cfg.token}`,
        Accept: "application/json",
        ...(body ? { "Content-Type": "application/json" } : {}),
      },
      body: body ? JSON.stringify(body) : undefined,
      credentials: "omit",
      redirect: "error",
      cache: "no-store",
      signal: controller.signal,
    });
    let data = null;
    try {
      data = await res.json();
    } catch {
      // Non-JSON bodies (proxies, HTML error pages) fall through to status.
    }
    if (res.ok) return { ok: true, status: res.status, data };
    const error = data && typeof data.error === "string" ? data.error : "";
    return { ok: false, status: res.status, error: lib.describeStatus(res.status, error) };
  } catch (err) {
    const message = err && err.name === "AbortError" ? "The app did not answer in time." : "Could not reach the Leetgrinder app.";
    return { ok: false, status: 0, error: message };
  } finally {
    clearTimeout(timer);
  }
}

// ---- Toolbar badge -------------------------------------------------------

// refreshBadge shows today's remaining goal: a number, a check when the goal
// is met, or a grey "?" when the app is not set up or cannot be reached.
async function refreshBadge() {
  const res = await api("GET", "/api/leetgrinder/today");
  const badge = lib.badgeState(res.ok ? res.data : null);
  await Promise.all([
    ext.action.setBadgeText({ text: badge.text }),
    ext.action.setBadgeBackgroundColor({ color: badge.color }),
    ext.action.setTitle({ title: badge.title }),
  ]).catch(() => {});
  return res;
}

// Confirmed attempts wait in storage.local until the app accepts them, so a
// closed tab, a reload or app downtime does not lose them.
const outbox = lib.createOutbox(ext.storage.local, (attempt) => api("POST", "/api/leetgrinder/attempts", attempt));

// syncOutbox resends waiting attempts; callers refresh the badge after it.
const syncOutbox = () => outbox.flush().catch(() => 0);

const BADGE_ALARM = "leetgrinder-today";
// Create the alarm once; recreating it on every worker start would restart
// its countdown.
ext.alarms.get(BADGE_ALARM).then((alarm) => alarm || ext.alarms.create(BADGE_ALARM, { periodInMinutes: 15 }));
ext.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === BADGE_ALARM) syncOutbox().then(refreshBadge);
});
ext.storage.onChanged.addListener((changes, area) => {
  if (area === "local" && (changes.origin || changes.token)) syncOutbox().then(refreshBadge);
});
ext.permissions.onAdded.addListener(() => syncOutbox().then(refreshBadge));
ext.permissions.onRemoved.addListener(() => refreshBadge());
// Chrome starts the worker at browser launch only for an onStartup
// listener; the top-level refresh below then runs once per worker start.
ext.runtime.onStartup.addListener(() => {});
syncOutbox().then(refreshBadge);

// Timers live in storage.session so page reloads keep them but a browser
// restart clears them. Only this worker touches that storage area.
const timerKey = (slug) => `timer:${slug}`;

// Timer changes are read-modify-write, so they run one at a time per slug.
const timerQueues = new Map();
function serialized(slug, fn) {
  const run = (timerQueues.get(slug) || Promise.resolve()).then(fn);
  const settled = run.catch(() => {});
  timerQueues.set(slug, settled);
  settled.then(() => {
    if (timerQueues.get(slug) === settled) timerQueues.delete(slug);
  });
  return run;
}

// readTimer returns the slug's timer, restarting it when it belongs to an
// earlier visit, and records that the problem page is open now.
async function readTimer(slug, patch = {}, rawSample = null, tabId = 0) {
  const key = timerKey(slug);
  const now = Date.now();
  const stored = (await ext.storage.session.get(key))[key];
  const fresh = lib.timerExpired(stored, now);
  let timer = fresh ? { startedAt: now, openedAt: now, activeMs: 0, tabs: {}, creditedTo: now, assisted: false, nudged: false } : stored;
  const sample = lib.validSample(rawSample, now);
  if (sample) timer = lib.creditActive(timer, sample, now, tabId);
  timer.lastSeenAt = now;
  if (patch.assisted === true) timer.assisted = true;
  if (patch.nudged === true) timer.nudged = true;
  await ext.storage.session.set({ [key]: timer });
  return timer;
}

const getTimer = (slug, sample, tabId) => serialized(slug, () => readTimer(slug, {}, sample, tabId));

const updateTimer = (slug, patch, sample, tabId) => serialized(slug, () => readTimer(slug, patch, sample, tabId));

// restartTimer starts timing the next attempt after one is logged. It is
// already nudged: the learner just finished and needs no "Log as unfinished".
const restartTimer = (slug) =>
  serialized(slug, async () => {
    const now = Date.now();
    const timer = { startedAt: now, openedAt: now, lastSeenAt: now, activeMs: 0, tabs: {}, creditedTo: now, assisted: false, nudged: true };
    await ext.storage.session.set({ [timerKey(slug)]: timer });
    return timer;
  });

// fromProblemSite is true for content scripts on LeetCode or NeetCode.
function fromProblemSite(sender) {
  return Boolean(sender.tab) && lib.siteOf(sender.url) !== null;
}

// The options page opens in a tab, so sender.tab is set there too; its
// extension URL is what identifies it (content scripts report the page URL).
function fromOptions(sender) {
  return typeof sender.url === "string" && sender.url.startsWith(ext.runtime.getURL("options.html"));
}

function fromPopup(sender) {
  return typeof sender.url === "string" && sender.url.startsWith(ext.runtime.getURL("popup.html"));
}

async function handle(message, sender) {
  if (sender.id !== ext.runtime.id || !message || typeof message !== "object") {
    return { ok: false, status: 0, error: "Unexpected sender." };
  }
  if (message.type === "test") {
    if (!fromOptions(sender)) return { ok: false, status: 0, error: "Unexpected sender." };
    return api("GET", "/api/leetgrinder/problem/two-sum");
  }
  if (message.type === "today") {
    if (!fromPopup(sender)) return { ok: false, status: 0, error: "Unexpected sender." };
    // Resending runs beside the badge read, so a slow app delays the popup
    // by one request, not a pass plus a request.
    const [, res] = await Promise.all([syncOutbox(), refreshBadge()]);
    const cfg = await config();
    return { ...res, origin: cfg.origin || "", outbox: await outbox.summary() };
  }
  if (message.type === "outbox:discard") {
    if (!fromPopup(sender) || !lib.validAttemptId(message.id)) return { ok: false, status: 0, error: "Unexpected sender." };
    await outbox.discard(message.id);
    return { ok: true, status: 200, outbox: await outbox.summary() };
  }
  if (!fromProblemSite(sender)) return { ok: false, status: 0, error: "Unexpected sender." };
  const slug = message.slug;
  switch (message.type) {
    case "problem":
      if (!lib.validSlug(slug)) return { ok: false, status: 0, error: "Invalid problem." };
      return api("GET", `/api/leetgrinder/problem/${encodeURIComponent(slug)}`);
    case "timer:get":
      if (!lib.validSlug(slug)) return { ok: false, status: 0, error: "Invalid problem." };
      return { ok: true, status: 200, data: await getTimer(slug, message.sample, sender.tab?.id ?? 0) };
    case "timer:update":
      if (!lib.validSlug(slug) || !message.patch || typeof message.patch !== "object") return { ok: false, status: 0, error: "Invalid timer update." };
      return { ok: true, status: 200, data: await updateTimer(slug, message.patch, message.sample, sender.tab?.id ?? 0) };
    case "timer:restart":
      if (!lib.validSlug(slug)) return { ok: false, status: 0, error: "Invalid problem." };
      return { ok: true, status: 200, data: await restartTimer(slug) };
    case "metadata": {
      const meta = lib.cleanMetadata(message.metadata);
      if (!lib.validSlug(slug) || !meta || !meta.title) return { ok: false, status: 0, error: "Invalid problem details." };
      return api("PUT", `/api/leetgrinder/problem/${encodeURIComponent(slug)}`, meta);
    }
    case "attempt": {
      const attempt = lib.cleanAttempt(message.attempt);
      if (!attempt) return { ok: false, status: 400, error: "Check the attempt fields and try again." };
      const res = await outbox.submit(attempt);
      if (res.ok) refreshBadge();
      return res;
    }
    case "open-history": {
      if (!lib.validSlug(slug)) return { ok: false, status: 0, error: "Invalid problem." };
      const cfg = await config();
      if (cfg.error) return { ok: false, status: 0, error: cfg.error };
      await ext.tabs.create({ url: `${cfg.origin}/leetgrinder/problem/${encodeURIComponent(slug)}` });
      return { ok: true, status: 200 };
    }
    default:
      return { ok: false, status: 0, error: "Unknown request." };
  }
}

ext.runtime.onMessage.addListener((message, sender, sendResponse) => {
  handle(message, sender).then(sendResponse, () => sendResponse({ ok: false, status: 0, error: "The extension hit an unexpected error." }));
  return true;
});
