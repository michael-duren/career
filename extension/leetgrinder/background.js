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

// Timers live in storage.session so page reloads keep them but a browser
// restart clears them. Only this worker touches that storage area.
const timerKey = (slug) => `timer:${slug}`;

async function getTimer(slug) {
  const key = timerKey(slug);
  const stored = (await ext.storage.session.get(key))[key];
  if (stored && Number.isFinite(stored.startedAt)) return stored;
  const timer = { startedAt: Date.now(), assisted: false, nudged: false };
  await ext.storage.session.set({ [key]: timer });
  return timer;
}

async function updateTimer(slug, patch) {
  const timer = await getTimer(slug);
  if (patch.assisted === true) timer.assisted = true;
  if (patch.nudged === true) timer.nudged = true;
  await ext.storage.session.set({ [timerKey(slug)]: timer });
  return timer;
}

function fromLeetCode(sender) {
  return Boolean(sender.tab) && typeof sender.url === "string" && sender.url.startsWith("https://leetcode.com/");
}

function fromOptions(sender) {
  return !sender.tab && typeof sender.url === "string" && sender.url.startsWith(ext.runtime.getURL("options.html"));
}

async function handle(message, sender) {
  if (sender.id !== ext.runtime.id || !message || typeof message !== "object") {
    return { ok: false, status: 0, error: "Unexpected sender." };
  }
  if (message.type === "test") {
    if (!fromOptions(sender)) return { ok: false, status: 0, error: "Unexpected sender." };
    return api("GET", "/api/leetgrinder/problem/two-sum");
  }
  if (!fromLeetCode(sender)) return { ok: false, status: 0, error: "Unexpected sender." };
  const slug = message.slug;
  switch (message.type) {
    case "problem":
      if (!lib.validSlug(slug)) return { ok: false, status: 0, error: "Invalid problem." };
      return api("GET", `/api/leetgrinder/problem/${encodeURIComponent(slug)}`);
    case "timer:get":
      if (!lib.validSlug(slug)) return { ok: false, status: 0, error: "Invalid problem." };
      return { ok: true, status: 200, data: await getTimer(slug) };
    case "timer:update":
      if (!lib.validSlug(slug) || !message.patch || typeof message.patch !== "object") return { ok: false, status: 0, error: "Invalid timer update." };
      return { ok: true, status: 200, data: await updateTimer(slug, message.patch) };
    case "timer:reset":
      if (!lib.validSlug(slug)) return { ok: false, status: 0, error: "Invalid problem." };
      await ext.storage.session.remove(timerKey(slug));
      return { ok: true, status: 200 };
    case "attempt": {
      const attempt = lib.cleanAttempt(message.attempt);
      if (!attempt) return { ok: false, status: 400, error: "Check the attempt fields and try again." };
      return api("POST", "/api/leetgrinder/attempts", attempt);
    }
    default:
      return { ok: false, status: 0, error: "Unknown request." };
  }
}

ext.runtime.onMessage.addListener((message, sender, sendResponse) => {
  handle(message, sender).then(sendResponse, () => sendResponse({ ok: false, status: 0, error: "The extension hit an unexpected error." }));
  return true;
});
