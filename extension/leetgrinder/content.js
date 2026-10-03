// Isolated-world content script on leetcode.com and neetcode.io. It tracks
// the open problem (always by its LeetCode slug, which is what the app knows),
// keeps its timer (via the background worker), and shows the confirm panel
// and the 25-minute nudge. It never talks to the app directly and builds all
// UI with DOM APIs and textContent, never HTML strings.
(() => {
  "use strict";
  if (globalThis.__leetgrinderContent) return;
  globalThis.__leetgrinderContent = true;

  const lib = globalThis.LeetgrinderLib;
  const DETECT_SOURCE = "leetgrinder-detect";
  // "leetcode" or "neetcode"; NeetCode pages use their own slugs, mapped
  // to LeetCode's by lib.problemFromPath.
  const site = lib.siteOf(location.href);

  // current is the problem on screen: {slug, lookup: Promise<lookup>, meta,
  // known}. A lookup is {status: "ok", info} or {status: "error", error}
  // when the app could not answer. meta is the problem's LeetCode metadata
  // once read; known is the metadata NeetCode's slug table has, if any.
  let current = null;
  let lastPath = "";
  // ui is the mounted panel: {host, root, locked, panel, saved}. A locked
  // panel holds an attempt that may already be saved and must stay until it
  // is retried. panel is how a log panel was opened ("accepted", "nudge" or
  // "manual"), unset for other panels; saved is set once its attempt is
  // saved and the panel is only waiting to close.
  let ui = null;
  // captureProblem is why the latest submission on the open problem was not
  // captured, or "" (see lib.captureIssue).
  let captureProblem = "";
  // capture is the latest validated submission on the open problem:
  // {slug, submissionId, status, lang, code} (see lib.cleanCapture). It is
  // cleared once logged with an attempt, so a later attempt never carries it.
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

  // Active time: the page counts as in use while it is visible and the
  // learner pressed a key, moved the mouse or scrolled recently. Timer
  // messages carry that sample; the background worker credits the time.
  // Starts at 0: a reload or navigation earns no active time until real input.
  let lastInputAt = 0;
  for (const name of ["keydown", "pointerdown", "mousemove", "wheel", "scroll", "touchstart"]) {
    window.addEventListener(
      name,
      (event) => {
        if (event.isTrusted) lastInputAt = Date.now();
      },
      { capture: true, passive: true },
    );
  }
  const activitySample = (visible = document.visibilityState === "visible") => ({ visible, lastInputAt });
  // Hiding flushes the time up to now as active; showing again records a
  // hidden stretch that earns nothing.
  document.addEventListener("visibilitychange", () => {
    if (!current) return;
    const nowVisible = document.visibilityState === "visible";
    send({ type: "timer:get", slug: current.slug, sample: activitySample(!nowVisible) });
  });

  async function send(message) {
    if (message.type.startsWith("timer:") && message.type !== "timer:restart" && !message.sample) message = { ...message, sample: activitySample() };
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
  // On NeetCode it uses the slug table instead; the app fetches topics
  // itself. It resolves to cleaned metadata or null and never rejects.
  async function readMetadata(state) {
    if (site !== "leetcode") return state.known || null;
    const slug = state.slug;
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
  // topics (on NeetCode, which has no topics, only when it lacks the
  // problem), and keeps it to attach to attempts.
  async function describe(state, info) {
    const described = info.known && (site !== "leetcode" || (Array.isArray(info.topics) && info.topics.length > 0));
    if (state.meta !== undefined || described) return;
    state.meta = null;
    const meta = await readMetadata(state);
    if (!meta || current !== state) return;
    state.meta = meta;
    await send({ type: "metadata", slug: state.slug, metadata: meta });
  }

  const busy = () => Boolean(ui && ui.locked);

  // onPath runs on first load and on the site's client-side navigation.
  async function onPath(path) {
    const problem = lib.problemFromPath(path, site);
    const slug = problem && problem.slug;
    if (!slug) {
      current = null;
      if (!busy()) closeUI();
      removeBanner();
      return;
    }
    if (!current || current.slug !== slug) {
      if (!busy()) closeUI();
      removeBanner();
      current = { slug, lookup: lookup(slug), known: problem.metadata };
      captureProblem = "";
    }
    const state = current;
    const found = await state.lookup;
    // When the app is unreachable the timer still starts, so the minutes stay right.
    if (current !== state) return;
    showBanner(state, found);
    if (found.status === "ok") describe(state, found.info);
    await send({ type: "timer:get", slug });
    if (lib.isAssistPath(path, site)) await send({ type: "timer:update", slug, patch: { assisted: true } });
  }

  function watchLocation() {
    // The title often renders after the status arrives; move a corner
    // banner under it as soon as it appears.
    if (banner && !banner.anchored && current && current.slug === banner.slug && titleAnchor(current.slug)) {
      current.lookup.then((found) => {
        if (current && banner && !banner.anchored) showBanner(current, found);
      });
    }
    if (location.pathname === lastPath) return;
    lastPath = location.pathname;
    onPath(lastPath);
  }

  // tick runs every 30 seconds. Its timer:get doubles as the heartbeat that
  // keeps the open problem's timer from expiring (see lib.timerExpired).
  async function tick() {
    const state = current;
    if (!state) return;
    // Send the heartbeat first: a slow or retried lookup must not delay the
    // sample the background worker credits active time from.
    const heartbeat = send({ type: "timer:get", slug: state.slug });
    let found = await state.lookup;
    if (found.status === "error" && current === state && !ui) {
      state.lookup = lookup(state.slug);
      found = await state.lookup;
      if (found.status === "ok" && current === state) describe(state, found.info);
    }
    if (current !== state) return;
    // The site re-renders the title; put the banner back if it was removed.
    showBanner(state, found);
    const res = await heartbeat;
    if (found.status !== "ok" || !res.ok || !lib.shouldNudge(res.data, Date.now()) || ui || current !== state) return;
    await send({ type: "timer:update", slug: state.slug, patch: { nudged: true } });
    // An Accepted panel may have opened while the update was in flight.
    if (ui || current !== state) return;
    showNudge(state, found.info);
  }

  // manualOpen reports a log panel opened from the banner, whose typed or
  // pasted input an Accepted submission must not replace.
  const manualOpen = () => Boolean(ui && ui.panel === "manual" && !ui.saved);

  async function onAccepted(submissionId) {
    const state = current;
    if (!state || busy() || manualOpen()) return;
    // Refresh so review status reflects anything logged since page load.
    state.lookup = lookup(state.slug);
    const found = await state.lookup;
    if (current !== state || busy() || manualOpen()) return;
    showBanner(state, found);
    if (found.status === "ok") describe(state, found.info);
    // Accepted ends the nudge window even if the panel is dismissed or the
    // app is unreachable.
    const res = await send({ type: "timer:update", slug: state.slug, patch: { nudged: true } });
    if (current !== state || busy() || manualOpen()) return;
    if (found.status === "error") {
      showError(`Accepted, but Leetgrinder could not be reached: ${found.error}`, submissionId);
      return;
    }
    showPanel(state, found.info, lib.prefillFrom(res.ok ? res.data : null, Date.now()), captureFor(state.slug, submissionId), "accepted");
  }

  // forgetCapture drops the captured submission once an attempt carrying its
  // code is saved or queued, so a later attempt does not reuse it.
  function forgetCapture(attempt) {
    if (capture && attempt.code && attempt.code === capture.code && attempt.problemSlug === capture.slug) capture = null;
  }

  // logManually opens the log panel from the banner, for an attempt solved
  // elsewhere or not submitted, prefilled from the problem timer. The
  // latest captured submission, if any, is offered as with the nudge.
  async function logManually(state) {
    // An open log panel keeps what was typed in it; the button does nothing.
    const blocked = () => current !== state || busy() || Boolean(ui && ui.panel);
    if (blocked()) return;
    const found = await state.lookup;
    if (blocked() || found.status !== "ok") return;
    const res = await send({ type: "timer:get", slug: state.slug });
    if (blocked()) return;
    showPanel(state, found.info, lib.prefillFrom(res.ok ? res.data : null, Date.now()), captureFor(state.slug), "manual");
  }

  window.addEventListener("message", (event) => {
    // Only same-window, same-origin messages from the page-world detector.
    // Page scripts could forge these, so they are validated strictly; they
    // only open the panel or offer code, and nothing is sent without a click.
    if (event.source !== window || event.origin !== location.origin) return;
    const data = event.data;
    if (!data || typeof data !== "object" || data.source !== DETECT_SOURCE) return;
    if (data.type === "submission") {
      // NeetCode's detector reports NeetCode's slug.
      const nc = site === "neetcode" ? lib.neetcodeProblem(data.slug) : null;
      const message = site === "neetcode" ? { ...data, slug: nc ? nc.slug : "" } : data;
      const cleaned = lib.cleanCapture(message, current && current.slug);
      if (cleaned) {
        capture = cleaned;
        captureProblem = "";
      } else {
        const issue = lib.captureIssue(message, current && current.slug);
        if (issue) captureProblem = issue;
      }
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
    details { margin-bottom: 10px; }
    summary { margin-bottom: 6px; cursor: pointer; }
    textarea[name=pastedCode] { min-height: 96px; font: 12px/1.4 ui-monospace, SFMono-Regular, Menlo, monospace; white-space: pre; }
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
    .bar { display: inline-flex; gap: 6px; align-items: center; }
    .bar.fixed { position: fixed; top: 64px; right: 20px; z-index: 2147483646; }
    .log { margin: 6px 0; padding: 4px 10px; border: 1px solid #56704a; border-radius: 999px; background: #191d1b; color: #bee48a;
      cursor: pointer; font: 600 12px/1.4 system-ui, -apple-system, "Segoe UI", sans-serif; }
    .log:hover, .log:focus-visible { border-color: #bee48a; outline: none; }
  `;
  // banner is the mounted pill: {host, slug, text}.
  let banner = null;

  function removeBanner() {
    if (banner) banner.host.remove();
    banner = null;
  }

  // titleAnchor finds the problem title to sit under; the sites' markup
  // changes, so several selectors are tried before a fixed corner.
  function titleAnchor(slug) {
    const selectors = site === "neetcode" ? [".problem-title-row"] : [".text-title-large", '[data-cy="question-title"]', `a[href="/problems/${slug}/"]`];
    for (const selector of selectors) {
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
    const pill = el("button", { type: "button", className: "pill", title: "Open this problem's history in Leetgrinder" }, [
      el("span", { className: "label", text: view.label }),
      ...(view.summary ? [el("span", { className: "summary", text: view.summary })] : []),
    ]);
    pill.addEventListener("click", () => send({ type: "open-history", slug: state.slug }));
    // Logs an attempt without a submission, such as one solved elsewhere.
    const log = el("button", { type: "button", className: "log", text: "Log attempt", title: "Log an attempt on this problem to Leetgrinder" });
    log.addEventListener("click", () => logManually(state));
    root.append(el("style", { text: BANNER_STYLE }), el("div", { className: anchor ? "bar" : "bar fixed" }, [pill, log]));
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
      showPanel(state, info, lib.prefillFrom(res.ok ? res.data : null, Date.now(), "unfinished"), captureFor(slug), "nudge");
    });
    later.addEventListener("click", closeUI);
    root.append(
      el("section", { className: "box", role: "dialog", "aria-label": "Leetgrinder time check" }, [
        ...heading(state, info),
        el("p", { text: `${lib.NUDGE_MINUTES} active minutes on this problem. Log it as unfinished and look at a hint, or keep going.` }),
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

  function showPanel(state, info, prefill, captured, opened) {
    const slug = state.slug;
    const root = mount();
    ui.panel = opened;
    // One id per panel: retries of the same entry are idempotent on the server.
    let id = crypto.randomUUID();
    const outcome = el("select", { name: "outcome" }, [
      el("option", { value: "solved", text: "Solved" }),
      el("option", { value: "struggled", text: "Struggled" }),
      el("option", { value: "unfinished", text: "Unfinished" }),
    ]);
    outcome.value = prefill.outcome;
    const minutes = el("input", { type: "number", name: "minutes", min: 1, max: lib.MAX_MINUTES, step: 1, required: true, value: String(prefill.minutes) });
    const assisted = el("input", { type: "checkbox", name: "assisted", checked: prefill.assisted });
    const approach = el("select", { name: "approach" }, [
      el("option", { value: "", text: "Not stated" }),
      el("option", { value: "optimal", text: "Reached the optimal solution" }),
      el("option", { value: "suboptimal", text: "Took a simpler approach for time" }),
    ]);
    const wantsReview = el("input", { type: "checkbox", name: "wantsReview", checked: false });
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
      : [el("p", { className: "muted", text: lib.codeStatus(captureProblem) })];
    // Code written elsewhere (an IDE, a whiteboard transcript) can be pasted;
    // it replaces any captured code.
    const pasteLang = el("select", { name: "pasteLanguage", "aria-label": "Pasted code language" }, [
      el("option", { value: "", text: "Choose…" }),
      ...lib.PASTE_LANGUAGES.map((l) => el("option", { value: l, text: lib.languageLabel(l) })),
    ]);
    // No preselection: pasted code is often in another language than the
    // captured submission, so the learner always chooses.
    const pasteCode = el("textarea", { name: "pastedCode", spellcheck: false, placeholder: "Paste your solution", "aria-label": "Pasted code" });
    const pasteRow = el("details", {}, [
      el("summary", { className: "muted", text: captured ? "Paste code instead" : "Paste code" }),
      el("label", {}, ["Language", pasteLang]),
      el("label", {}, ["Code", pasteCode]),
    ]);
    const fields = [outcome, minutes, ...time.fields, ...space.fields, approach, assisted, wantsReview, includeCode, pasteLang, pasteCode, notes];
    // After an ambiguous failure the server may have saved the entry, so the
    // fields lock and retries resend exactly the same attempt.
    let locked = null;
    // droppedCode is set when the captured code was too large to send.
    let droppedCode = false;
    // queued is set while the outbox holds this panel's attempt, and
    // restarted once the timer was restarted for it.
    let queued = false;
    let restarted = false;

    const form = el("form", { className: "box", "aria-label": "Log this attempt to Leetgrinder" }, [
      ...heading(state, info),
      el("div", { className: "row" }, [el("label", {}, ["Outcome", outcome]), el("label", {}, ["Minutes", minutes])]),
      ...(lib.showOpenTime(prefill.minutes, prefill.openMinutes) ? [el("p", { className: "muted", text: `Active ${prefill.minutes} min (open ${prefill.openMinutes} min)` })] : []),
      el("div", { className: "row" }, [time.node, space.node]),
      required,
      ...codeRow,
      pasteRow,
      el("label", {}, ["Approach", approach]),
      el("label", { className: "check" }, [assisted, "Used a hint or solution"]),
      el("label", { className: "check" }, [wantsReview, "Review this again soon"]),
      el("label", {}, ["Notes", notes]),
      el("div", { className: "actions" }, [dismiss, submit]),
      status,
    ]);
    const closePanel = (trigger) => {
      // A saved panel waiting to close is not holding anything unsaved.
      if (lib.panelCloses(trigger, Boolean(ui && ui.locked && !ui.saved), (message) => window.confirm(message))) closeUI();
    };
    dismiss.addEventListener("click", () => closePanel("dismiss"));
    form.addEventListener("keydown", (event) => {
      if (event.key === "Escape") closePanel("escape");
      // Keep the site's editor shortcuts from firing while typing here.
      event.stopPropagation();
    });
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const pasted = locked ? { capture: null } : lib.pastedCode(pasteCode.value, pasteLang.value, slug);
      if (pasted.error) {
        status.className = "status error";
        status.textContent = pasted.error;
        return;
      }
      const { source, withCode } = lib.codeSource(pasted.capture, captured, includeCode.checked);
      const attempt =
        locked ||
        lib.buildAttempt(
          {
            id,
            problemSlug: slug,
            outcome: outcome.value,
            minutes: Number(minutes.value),
            assisted: assisted.checked,
            wantsReview: wantsReview.checked,
            approach: approach.value,
            notes: notes.value,
            // The app decides whether an attempt is a review.
            isReview: false,
            problem: state.meta || undefined,
            timeComplexity: time.value(),
            spaceComplexity: space.value(),
          },
          source,
          withCode,
        );
      const problem = lib.attemptProblem(attempt);
      if (problem || !lib.cleanAttempt(attempt)) {
        status.className = "status error";
        status.textContent = problem || "The attempt is too large to send.";
        return;
      }
      if (pasted.capture && !attempt.code) {
        status.className = "status error";
        status.textContent = "The pasted code is too large to send. Shorten it or leave it out.";
        return;
      }
      // A retry resends the locked attempt, so it keeps the first note.
      if (!locked) droppedCode = Boolean(source && withCode && !attempt.code);
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
        panel.locked = panel.saved = true;
        forgetCapture(attempt);
        const kind = lib.kindLabel(res.data && res.data.kind);
        const logged = kind ? `Logged to Leetgrinder as ${kind}.` : "Logged to Leetgrinder.";
        status.textContent = droppedCode ? `${logged} The code was too large to send.` : logged;
        // Refresh the banner with the new attempt.
        state.lookup = lookup(slug);
        state.lookup.then((found) => {
          if (current === state) showBanner(state, found);
        });
        // Start a fresh timer for the next attempt on this problem, unless
        // queuing it already did. The panel stays locked until then, so a
        // new panel cannot reuse the minutes just logged.
        if (!restarted) {
          const restart = await send({ type: "timer:restart", slug });
          if (!restart.ok) status.textContent += " The timer did not restart; check the minutes on your next log.";
        }
        panel.locked = false;
        setTimeout(() => {
          if (ui && ui.root === root) closeUI();
        }, 1500);
        return;
      }
      status.className = "status error";
      status.textContent = res.error || lib.describeStatus(res.status, "");
      if (res.queued) {
        // The extension keeps the attempt and resends it until the app
        // accepts it, so the panel can close (Escape included) without
        // losing it. Retry now resends the same attempt.
        locked = attempt;
        panel.locked = false;
        submit.disabled = false;
        submit.textContent = "Retry now";
        status.className = "status";
        status.textContent = lib.queuedMessage(res);
        forgetCapture(attempt);
        if (!restarted) await send({ type: "timer:restart", slug });
        queued = restarted = true;
        return;
      }
      if (queued && res.status !== 409 && !lib.outboxRetryable(res.status)) {
        // A queued attempt was rejected for good on Retry now. The outbox
        // dropped it, so unlock the fields to fix and save it again, under a
        // new id so a resend already under way cannot touch the new entry.
        id = crypto.randomUUID();
        locked = null;
        queued = false;
        panel.locked = false;
        submit.disabled = false;
        submit.textContent = "Log attempt";
        for (const f of fields) f.disabled = false;
        return;
      }
      if (res.status === 409) {
        // This id can never succeed. Keep what was typed and let the learner
        // save it under a new id.
        id = crypto.randomUUID();
        locked = null;
        queued = false;
        panel.locked = false;
        submit.disabled = false;
        submit.textContent = "Log attempt";
        for (const f of fields) f.disabled = false;
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

  // Both sites are single-page apps; poll the path to follow navigation.
  watchLocation();
  setInterval(watchLocation, 1000);
  setInterval(tick, 30000);
})();
