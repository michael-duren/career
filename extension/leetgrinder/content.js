// Isolated-world content script on leetcode.com. It tracks the open problem,
// keeps its timer (via the background worker), and shows the confirm panel
// and the 25-minute nudge. It never talks to the app directly and builds all
// UI with DOM APIs and textContent, never HTML strings.
(() => {
  "use strict";
  if (globalThis.__leetgrinderContent) return;
  globalThis.__leetgrinderContent = true;

  const lib = globalThis.LeetgrinderLib;
  const DETECT_SOURCE = "leetgrinder-detect";

  // current is the problem on screen: {slug, lookup: Promise<lookup>}.
  // A lookup is {status: "in", info} for curriculum problems, {status: "out"}
  // for others, or {status: "error", error} when the app could not answer.
  let current = null;
  let lastPath = "";
  // ui is the mounted panel: {host, root, locked}. A locked panel holds an
  // attempt that may already be saved and must stay until it is retried.
  let ui = null;

  async function send(message) {
    try {
      const res = await ext.runtime.sendMessage(message);
      return res || { ok: false, status: 0, error: "No response from the extension." };
    } catch {
      return { ok: false, status: 0, error: "The extension was reloaded. Refresh this page." };
    }
  }

  async function lookup(slug) {
    const res = await send({ type: "problem", slug });
    if (!res.ok || !res.data) return { status: "error", error: res.error || lib.describeStatus(res.status, "") };
    return res.data.inCurriculum ? { status: "in", info: res.data } : { status: "out" };
  }

  const busy = () => Boolean(ui && ui.locked);

  // onPath runs on first load and on LeetCode's client-side navigation.
  async function onPath(path) {
    const slug = lib.slugFromPath(path);
    if (!slug) {
      current = null;
      if (!busy()) closeUI();
      return;
    }
    if (!current || current.slug !== slug) {
      if (!busy()) closeUI();
      current = { slug, lookup: lookup(slug) };
    }
    const state = current;
    const found = await state.lookup;
    // Problems outside the curriculum get no timer and no panel. When the app
    // is unreachable the timer still starts, so the minutes stay right.
    if (found.status === "out" || current !== state) return;
    await send({ type: "timer:get", slug });
    if (lib.isAssistPath(path)) await send({ type: "timer:update", slug, patch: { assisted: true } });
  }

  function watchLocation() {
    if (location.pathname === lastPath) return;
    lastPath = location.pathname;
    onPath(lastPath);
  }

  // tick runs every 30 seconds. Its timer:get doubles as the heartbeat that
  // keeps the open problem's timer from expiring (see lib.timerExpired).
  async function tick() {
    const state = current;
    if (!state) return;
    let found = await state.lookup;
    if (found.status === "error" && current === state && !ui) {
      state.lookup = lookup(state.slug);
      found = await state.lookup;
    }
    if (found.status === "out" || current !== state) return;
    const res = await send({ type: "timer:get", slug: state.slug });
    if (found.status !== "in" || !res.ok || !lib.shouldNudge(res.data, Date.now()) || ui || current !== state) return;
    await send({ type: "timer:update", slug: state.slug, patch: { nudged: true } });
    showNudge(state.slug, found.info);
  }

  async function onAccepted() {
    const state = current;
    if (!state || busy()) return;
    // Refresh so review status reflects anything logged since page load.
    state.lookup = lookup(state.slug);
    const found = await state.lookup;
    if (found.status === "out" || current !== state || busy()) return;
    if (found.status === "error") {
      showError(`Accepted, but Leetgrinder could not be reached: ${found.error}`);
      return;
    }
    const res = await send({ type: "timer:get", slug: state.slug });
    const timer = res.ok ? res.data : { startedAt: Date.now(), assisted: false };
    const minutes = lib.elapsedMinutes(timer.startedAt, Date.now());
    showPanel(state.slug, found.info, { outcome: lib.inferOutcome(minutes), minutes, assisted: Boolean(timer.assisted) });
  }

  window.addEventListener("message", (event) => {
    // Only same-window messages from the page-world detector. Page scripts
    // could forge this; it only opens the panel, which needs a click to send.
    if (event.source !== window || !event.data || event.data.source !== DETECT_SOURCE) return;
    if (event.data.type === "accepted") onAccepted();
  });

  // ---- UI ----------------------------------------------------------------

  const STYLE = `
    :host { all: initial; }
    .box { position: fixed; right: 20px; bottom: 20px; z-index: 2147483647; width: min(360px, calc(100vw - 40px));
      box-sizing: border-box; padding: 16px; border: 1px solid #56704a; border-radius: 8px; background: #191d1b;
      color: #edf3ed; font: 14px/1.5 system-ui, -apple-system, "Segoe UI", sans-serif; box-shadow: 0 8px 30px rgba(0,0,0,.45); }
    h2 { margin: 0 0 4px; font-size: 15px; }
    p { margin: 0 0 10px; }
    .muted { color: #a7b4a9; font-size: 12px; }
    .badge { display: inline-block; margin-left: 6px; padding: 0 6px; border-radius: 4px; background: #283423; color: #bee48a; font-size: 12px; }
    label { display: grid; gap: 4px; margin-bottom: 10px; font-size: 13px; }
    label.check { display: flex; gap: 8px; align-items: center; }
    select, input[type=number], textarea { box-sizing: border-box; width: 100%; padding: 6px 8px; border: 1px solid #303832;
      border-radius: 6px; background: #101312; color: #edf3ed; font: inherit; }
    textarea { min-height: 64px; resize: vertical; }
    .row { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
    .actions { display: flex; gap: 8px; justify-content: flex-end; }
    button { padding: 6px 12px; border: 1px solid #303832; border-radius: 6px; background: #101312; color: #edf3ed; font: inherit; cursor: pointer; }
    button.primary { border-color: #bee48a; background: #bee48a; color: #101312; font-weight: 600; }
    button:disabled { opacity: .6; cursor: default; }
    .status { min-height: 1.5em; margin: 8px 0 0; font-size: 13px; }
    .status.error { color: #ffb4a8; }
  `;

  function el(tag, props = {}, children = []) {
    const node = document.createElement(tag);
    for (const [key, value] of Object.entries(props)) {
      if (key === "text") node.textContent = value;
      else if (key in node) node[key] = value;
      else node.setAttribute(key, value);
    }
    for (const child of children) node.append(child);
    return node;
  }

  function mount() {
    closeUI();
    const host = el("div", { id: "leetgrinder-extension-root" });
    const root = host.attachShadow({ mode: "closed" });
    root.append(el("style", { text: STYLE }));
    document.documentElement.append(host);
    ui = { host, root, locked: false };
    return root;
  }

  function showError(message) {
    const root = mount();
    const retry = el("button", { type: "button", className: "primary", text: "Try again" });
    const dismiss = el("button", { type: "button", text: "Dismiss" });
    retry.addEventListener("click", () => {
      closeUI();
      onAccepted();
    });
    dismiss.addEventListener("click", closeUI);
    root.append(
      el("section", { className: "box", role: "alert" }, [
        el("h2", { text: "Leetgrinder" }),
        el("p", { className: "status error", text: message }),
        el("div", { className: "actions" }, [dismiss, retry]),
      ]),
    );
  }

  function closeUI() {
    if (ui) ui.host.remove();
    ui = null;
  }

  function heading(info) {
    const title = el("h2", { text: info.title });
    if (info.todaysReview) title.append(el("span", { className: "badge", text: "Today's review" }));
    const where = el("p", { className: "muted", text: `Leetgrinder · Session ${info.session} · Week ${info.week}` });
    return [title, where];
  }

  function showNudge(slug, info) {
    const root = mount();
    const unfinished = el("button", { type: "button", className: "primary", text: "Log as unfinished" });
    const later = el("button", { type: "button", text: "Keep going" });
    unfinished.addEventListener("click", async () => {
      const res = await send({ type: "timer:get", slug });
      const minutes = lib.elapsedMinutes(res.ok ? res.data.startedAt : Date.now(), Date.now());
      showPanel(slug, info, { outcome: "unfinished", minutes, assisted: Boolean(res.ok && res.data.assisted) });
    });
    later.addEventListener("click", closeUI);
    root.append(
      el("section", { className: "box", role: "dialog", "aria-label": "Leetgrinder time check" }, [
        ...heading(info),
        el("p", { text: `${lib.NUDGE_MINUTES} minutes on this problem. Log it as unfinished and look at a hint, or keep going.` }),
        el("div", { className: "actions" }, [later, unfinished]),
      ]),
    );
    unfinished.focus();
  }

  function showPanel(slug, info, prefill) {
    const root = mount();
    // One id per panel: retries of the same entry are idempotent on the server.
    const id = crypto.randomUUID();
    const outcome = el("select", { name: "outcome" }, [
      el("option", { value: "solved", text: "Solved" }),
      el("option", { value: "struggled", text: "Struggled" }),
      el("option", { value: "unfinished", text: "Unfinished" }),
    ]);
    outcome.value = prefill.outcome;
    const minutes = el("input", { type: "number", name: "minutes", min: 1, max: lib.MAX_MINUTES, step: 1, required: true, value: String(prefill.minutes) });
    const assisted = el("input", { type: "checkbox", name: "assisted", checked: prefill.assisted });
    const review = el("input", { type: "checkbox", name: "review", checked: Boolean(info.todaysReview && !info.reviewDone) });
    const notes = el("textarea", { name: "notes", maxLength: lib.MAX_NOTES, placeholder: "What to remember next time" });
    const status = el("p", { className: "status", role: "status" });
    const submit = el("button", { type: "submit", className: "primary", text: "Log attempt" });
    const dismiss = el("button", { type: "button", text: "Dismiss" });
    const fields = [outcome, minutes, assisted, review, notes];
    // After an ambiguous failure the server may have saved the entry, so the
    // fields lock and retries resend exactly the same attempt.
    let locked = null;

    const form = el("form", { className: "box", "aria-label": "Log this attempt to Leetgrinder" }, [
      ...heading(info),
      el("div", { className: "row" }, [el("label", {}, ["Outcome", outcome]), el("label", {}, ["Minutes", minutes])]),
      el("label", { className: "check" }, [assisted, "Used a hint or solution"]),
      ...(info.todaysReview ? [el("label", { className: "check" }, [review, "Count as today's review"])] : []),
      el("label", {}, ["Notes", notes]),
      el("div", { className: "actions" }, [dismiss, submit]),
      status,
    ]);
    dismiss.addEventListener("click", closeUI);
    form.addEventListener("keydown", (event) => {
      if (event.key === "Escape") closeUI();
      // Keep LeetCode's editor shortcuts from firing while typing here.
      event.stopPropagation();
    });
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const attempt = locked || {
        id,
        problemSlug: slug,
        outcome: outcome.value,
        minutes: Number(minutes.value),
        assisted: assisted.checked,
        notes: notes.value,
        isReview: Boolean(info.todaysReview && review.checked),
      };
      if (!lib.cleanAttempt(attempt)) {
        status.className = "status error";
        status.textContent = `Minutes must be a whole number from 1 to ${lib.MAX_MINUTES}.`;
        return;
      }
      submit.disabled = true;
      for (const f of fields) f.disabled = true;
      status.className = "status";
      status.textContent = "Saving…";
      // Until the outcome is known, a new Accepted must not replace this panel.
      const panel = ui;
      panel.locked = true;
      const res = await send({ type: "attempt", attempt });
      panel.locked = Boolean(locked);
      if (res.ok) {
        panel.locked = false;
        status.textContent = "Logged to Leetgrinder.";
        // Start a fresh timer for the next attempt on this problem.
        await send({ type: "timer:reset", slug });
        await send({ type: "timer:get", slug });
        setTimeout(() => {
          if (ui && ui.root === root) closeUI();
        }, 1500);
        return;
      }
      status.className = "status error";
      status.textContent = res.error || lib.describeStatus(res.status, "");
      if (res.status === 409) {
        // This id can never succeed; correct it in the app.
        panel.locked = false;
        return;
      }
      submit.disabled = false;
      if (res.status === 0 || res.status >= 500) {
        locked = attempt;
        panel.locked = true;
        submit.textContent = "Retry";
      } else if (!locked) {
        for (const f of fields) f.disabled = false;
      }
    });
    root.append(form);
    outcome.focus();
  }

  // LeetCode is a single-page app; poll the path to follow its navigation.
  watchLocation();
  setInterval(watchLocation, 1000);
  setInterval(tick, 30000);
})();
