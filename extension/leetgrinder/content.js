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

  // current is the problem on screen: {slug, lookup: Promise<lookup>, meta}.
  // A lookup is {status: "ok", info} or {status: "error", error} when the
  // app could not answer. meta is the problem's LeetCode metadata once read.
  let current = null;
  let lastPath = "";
  // ui is the mounted panel: {host, root, locked}. A locked panel holds an
  // attempt that may already be saved and must stay until it is retried.
  let ui = null;
  // capture is the latest validated submission on the open problem:
  // {slug, submissionId, status, lang, code} (see lib.cleanCapture).
  let capture = null;

  // captureFor picks the code that produced the result being logged: the
  // matching submission for a network-detected Accepted, the latest Accepted
  // one for the DOM fallback, or the latest of any status for the nudge.
  function captureFor(slug, submissionId) {
    if (!capture || capture.slug !== slug) return null;
    if (submissionId === undefined) return capture;
    const m = /^submission-(\d+)$/.exec(submissionId);
    if (m) return capture.submissionId === m[1] ? capture : null;
    return capture.status === "Accepted" ? capture : null;
  }

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
    return { status: "ok", info: res.data };
  }

  // readMetadata asks LeetCode's GraphQL API, same-origin, about the problem.
  // It resolves to cleaned metadata or null and never rejects.
  async function readMetadata(slug) {
    try {
      // Absolute: Firefox resolves relative content-script URLs against the extension.
      const res = await fetch(location.origin + "/graphql", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ operationName: "questionData", query: lib.GRAPHQL_QUERY, variables: { titleSlug: slug } }),
        credentials: "same-origin",
      });
      return res.ok ? lib.metadataFromGraphQL(await res.json()) : null;
    } catch {
      return null;
    }
  }

  // describe sends LeetCode's metadata to the app when it lacks the title or
  // topics, and keeps it to attach to attempts.
  async function describe(state, info) {
    if (state.meta !== undefined || (info.known && Array.isArray(info.topics) && info.topics.length > 0)) return;
    state.meta = null;
    const meta = await readMetadata(state.slug);
    if (!meta || current !== state) return;
    state.meta = meta;
    await send({ type: "metadata", slug: state.slug, metadata: meta });
  }

  const busy = () => Boolean(ui && ui.locked);

  // onPath runs on first load and on LeetCode's client-side navigation.
  async function onPath(path) {
    const slug = lib.slugFromPath(path);
    if (!slug) {
      current = null;
      if (!busy()) closeUI();
      removeBanner();
      return;
    }
    if (!current || current.slug !== slug) {
      if (!busy()) closeUI();
      removeBanner();
      current = { slug, lookup: lookup(slug) };
    }
    const state = current;
    const found = await state.lookup;
    // When the app is unreachable the timer still starts, so the minutes stay right.
    if (current !== state) return;
    showBanner(state, found);
    if (found.status === "ok") describe(state, found.info);
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
      if (found.status === "ok" && current === state) describe(state, found.info);
    }
    if (current !== state) return;
    // LeetCode re-renders the title; put the banner back if it was removed.
    showBanner(state, found);
    const res = await send({ type: "timer:get", slug: state.slug });
    if (found.status !== "ok" || !res.ok || !lib.shouldNudge(res.data, Date.now()) || ui || current !== state) return;
    await send({ type: "timer:update", slug: state.slug, patch: { nudged: true } });
    // An Accepted panel may have opened while the update was in flight.
    if (ui || current !== state) return;
    showNudge(state, found.info);
  }

  async function onAccepted(submissionId) {
    const state = current;
    if (!state || busy()) return;
    // Refresh so review status reflects anything logged since page load.
    state.lookup = lookup(state.slug);
    const found = await state.lookup;
    if (current !== state || busy()) return;
    showBanner(state, found);
    if (found.status === "ok") describe(state, found.info);
    // Accepted ends the nudge window even if the panel is dismissed or the
    // app is unreachable.
    const res = await send({ type: "timer:update", slug: state.slug, patch: { nudged: true } });
    if (current !== state || busy()) return;
    if (found.status === "error") {
      showError(`Accepted, but Leetgrinder could not be reached: ${found.error}`, submissionId);
      return;
    }
    const timer = res.ok ? res.data : { startedAt: Date.now(), assisted: false };
    const minutes = lib.elapsedMinutes(timer.startedAt, Date.now());
    showPanel(state, found.info, { outcome: lib.inferOutcome(minutes), minutes, assisted: Boolean(timer.assisted) }, captureFor(state.slug, submissionId));
  }

  window.addEventListener("message", (event) => {
    // Only same-window, same-origin messages from the page-world detector.
    // Page scripts could forge these, so they are validated strictly; they
    // only open the panel or offer code, and nothing is sent without a click.
    if (event.source !== window || event.origin !== location.origin) return;
    const data = event.data;
    if (!data || typeof data !== "object" || data.source !== DETECT_SOURCE) return;
    if (data.type === "submission") {
      const cleaned = lib.cleanCapture(data, current && current.slug);
      if (cleaned) capture = cleaned;
    } else if (data.type === "accepted" && typeof data.submissionId === "string" && data.submissionId.length <= 64) {
      onAccepted(data.submissionId);
    }
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
    select, input[type=number], input[type=text], textarea { box-sizing: border-box; width: 100%; padding: 6px 8px; border: 1px solid #303832;
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

  function showError(message, submissionId) {
    const root = mount();
    const retry = el("button", { type: "button", className: "primary", text: "Try again" });
    const dismiss = el("button", { type: "button", text: "Dismiss" });
    retry.addEventListener("click", () => {
      closeUI();
      onAccepted(submissionId);
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

  // ---- Banner --------------------------------------------------------------

  const BANNER_STYLE = `
    :host { all: initial; }
    .pill { display: inline-flex; flex-direction: column; gap: 0; max-width: 360px; margin: 6px 0; padding: 4px 10px;
      border: 1px solid #56704a; border-radius: 999px; background: #191d1b; color: #edf3ed; cursor: pointer;
      font: 12px/1.4 system-ui, -apple-system, "Segoe UI", sans-serif; text-align: left; }
    .pill:hover, .pill:focus-visible { border-color: #bee48a; outline: none; }
    .label { color: #bee48a; font-weight: 600; }
    .summary { color: #a7b4a9; }
    .fixed { position: fixed; top: 64px; right: 20px; z-index: 2147483646; }
  `;
  // banner is the mounted pill: {host, slug, text}.
  let banner = null;

  function removeBanner() {
    if (banner) banner.host.remove();
    banner = null;
  }

  // titleAnchor finds the problem title to sit under; LeetCode's markup
  // changes, so several selectors are tried before a fixed corner.
  function titleAnchor(slug) {
    for (const selector of [".text-title-large", '[data-cy="question-title"]', `a[href="/problems/${slug}/"]`]) {
      const node = document.querySelector(selector);
      if (node) return node;
    }
    return null;
  }

  // showBanner shows the problem's status from a lookup. It is hidden when
  // the extension is not set up or the app cannot be reached.
  function showBanner(state, found) {
    const view = found && found.status === "ok" ? lib.bannerState(found.info, Date.now()) : null;
    if (!view) {
      removeBanner();
      return;
    }
    const text = `${view.label}|${view.summary}`;
    const anchor = titleAnchor(state.slug);
    if (banner && banner.slug === state.slug && banner.text === text && banner.host.isConnected && banner.anchored === Boolean(anchor)) return;
    removeBanner();
    const host = el("div", { id: "leetgrinder-banner-root" });
    const root = host.attachShadow({ mode: "closed" });
    const pill = el("button", { type: "button", className: anchor ? "pill" : "pill fixed", title: "Open this problem's history in Leetgrinder" }, [
      el("span", { className: "label", text: view.label }),
      ...(view.summary ? [el("span", { className: "summary", text: view.summary })] : []),
    ]);
    pill.addEventListener("click", () => send({ type: "open-history", slug: state.slug }));
    root.append(el("style", { text: BANNER_STYLE }), pill);
    if (anchor) anchor.after(host);
    else document.documentElement.append(host);
    banner = { host, slug: state.slug, text, anchored: Boolean(anchor) };
  }

  function closeUI() {
    if (ui) ui.host.remove();
    ui = null;
  }

  function heading(state, info) {
    const title = el("h2", { text: info.title || (state.meta && state.meta.title) || state.slug });
    // The kind this attempt will count as today, as the app decides it.
    const label = lib.kindLabel(info.todayKind);
    if (label) title.append(el("span", { className: "badge", text: label }));
    const where = el("p", { className: "muted", text: "Leetgrinder" });
    return [title, where];
  }

  function showNudge(state, info) {
    const slug = state.slug;
    const root = mount();
    const unfinished = el("button", { type: "button", className: "primary", text: "Log as unfinished" });
    const later = el("button", { type: "button", text: "Keep going" });
    unfinished.addEventListener("click", async () => {
      const res = await send({ type: "timer:get", slug });
      const minutes = lib.elapsedMinutes(res.ok ? res.data.startedAt : Date.now(), Date.now());
      showPanel(state, info, { outcome: "unfinished", minutes, assisted: Boolean(res.ok && res.data.assisted) }, captureFor(slug));
    });
    later.addEventListener("click", closeUI);
    root.append(
      el("section", { className: "box", role: "dialog", "aria-label": "Leetgrinder time check" }, [
        ...heading(state, info),
        el("p", { text: `${lib.NUDGE_MINUTES} minutes on this problem. Log it as unfinished and look at a hint, or keep going.` }),
        el("div", { className: "actions" }, [later, unfinished]),
      ]),
    );
    unfinished.focus();
  }

  // complexityControl is a select of common classes plus "Other…", which
  // reveals a text field. value() is the stated complexity before
  // normalisation.
  function complexityControl(name, label) {
    const select = el("select", { name, "aria-label": `${label} complexity` }, [
      el("option", { value: "", text: "Choose…" }),
      ...lib.COMPLEXITIES.map((c) => el("option", { value: c, text: c })),
      el("option", { value: "other", text: "Other…" }),
    ]);
    const other = el("input", { type: "text", name: `${name}Other`, maxLength: lib.MAX_COMPLEXITY, placeholder: "O(m·n)", autocomplete: "off", spellcheck: false, "aria-label": `Other ${label.toLowerCase()} complexity` });
    other.hidden = true;
    select.addEventListener("change", () => {
      other.hidden = select.value !== "other";
      if (!other.hidden) other.focus();
    });
    return {
      node: el("label", {}, [label, select, other]),
      fields: [select, other],
      value: () => (select.value === "other" ? other.value : select.value),
    };
  }

  function showPanel(state, info, prefill, captured) {
    const slug = state.slug;
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
    const notes = el("textarea", { name: "notes", maxLength: lib.MAX_NOTES, placeholder: "What to remember next time" });
    const status = el("p", { className: "status", role: "status" });
    const submit = el("button", { type: "submit", className: "primary", text: "Log attempt" });
    const dismiss = el("button", { type: "button", text: "Dismiss" });
    const time = complexityControl("timeComplexity", "Time");
    const space = complexityControl("spaceComplexity", "Space");
    const required = el("p", { className: "muted" });
    const showRequired = () => {
      required.textContent = lib.needsComplexity(outcome.value) ? "Time and space complexity are required." : "Complexity is optional for unfinished attempts.";
    };
    showRequired();
    outcome.addEventListener("change", showRequired);
    const includeCode = el("input", { type: "checkbox", name: "includeCode", checked: Boolean(captured) });
    const codeRow = captured
      ? [el("label", { className: "check" }, [includeCode, `Code captured (${lib.languageLabel(captured.lang)}, ${lib.formatBytes(lib.utf8Bytes(captured.code))})`])]
      : [];
    const fields = [outcome, minutes, ...time.fields, ...space.fields, assisted, includeCode, notes];
    // After an ambiguous failure the server may have saved the entry, so the
    // fields lock and retries resend exactly the same attempt.
    let locked = null;

    const form = el("form", { className: "box", "aria-label": "Log this attempt to Leetgrinder" }, [
      ...heading(state, info),
      el("div", { className: "row" }, [el("label", {}, ["Outcome", outcome]), el("label", {}, ["Minutes", minutes])]),
      el("div", { className: "row" }, [time.node, space.node]),
      required,
      ...codeRow,
      el("label", { className: "check" }, [assisted, "Used a hint or solution"]),
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
      const attempt =
        locked ||
        lib.buildAttempt(
          {
            id,
            problemSlug: slug,
            outcome: outcome.value,
            minutes: Number(minutes.value),
            assisted: assisted.checked,
            notes: notes.value,
            // The app decides whether an attempt is a review.
            isReview: false,
            problem: state.meta || undefined,
            timeComplexity: time.value(),
            spaceComplexity: space.value(),
          },
          captured,
          includeCode.checked,
        );
      const problem = lib.attemptProblem(attempt);
      if (problem || !lib.cleanAttempt(attempt)) {
        status.className = "status error";
        status.textContent = problem || "The attempt is too large to send.";
        return;
      }
      const droppedCode = Boolean(captured && includeCode.checked && !attempt.code);
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
        const kind = lib.kindLabel(res.data && res.data.kind);
        const logged = kind ? `Logged to Leetgrinder as ${kind}.` : "Logged to Leetgrinder.";
        status.textContent = droppedCode ? `${logged} The code was too large to send.` : logged;
        // Refresh the banner with the new attempt.
        state.lookup = lookup(slug);
        state.lookup.then((found) => {
          if (current === state) showBanner(state, found);
        });
        // Start a fresh timer for the next attempt on this problem.
        await send({ type: "timer:restart", slug });
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
