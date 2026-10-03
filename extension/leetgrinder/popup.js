// Toolbar popup: today's goal, streak, review picks and due list, read
// through the background worker, and attempts waiting to sync. Opening it
// resends waiting attempts; otherwise its only write is Discard.
(() => {
  "use strict";
  const lib = globalThis.LeetgrinderLib;
  const root = document.getElementById("popup");

  function el(tag, props = {}, children = []) {
    const node = document.createElement(tag);
    for (const [key, value] of Object.entries(props)) {
      if (key === "text") node.textContent = value;
      else node[key] = value;
    }
    for (const child of children) node.append(child);
    return node;
  }

  function list(items) {
    return el(
      "ul",
      { className: "list" },
      items.map((r) =>
        el("li", { className: r.done ? "done" : "" }, [
          el("a", { href: r.url, target: "_blank", rel: "noopener noreferrer", text: r.title }),
          ...(r.done ? [el("span", { className: "muted small", text: " · done" })] : []),
          ...(r.reason ? [el("div", { className: "muted small", text: r.reason })] : []),
        ]),
      ),
    );
  }

  // syncSection lists attempts waiting to sync and those the app rejected,
  // each with a Discard button.
  function syncSection(summary) {
    const lines = lib.outboxLines(summary);
    const failed = summary && Array.isArray(summary.failed) ? summary.failed : [];
    if (!lines.length && !failed.length) return [];
    const nodes = [el("h2", { text: "Sync" }), ...lines.map((text) => el("p", { className: "small", text }))];
    if (failed.length) {
      nodes.push(el("p", { className: "small error", text: "The app rejected these attempts. Log them again from the problem page if needed." }));
      nodes.push(
        el(
          "ul",
          { className: "list" },
          failed.map((f) => {
            const discard = el("button", { type: "button", className: "small", text: "Discard" });
            discard.addEventListener("click", async () => {
              discard.disabled = true;
              const res = await ext.runtime.sendMessage({ type: "outbox:discard", id: f.id }).catch(() => null);
              if (res && res.ok) sync.replaceChildren(...syncSection(res.outbox));
              else discard.disabled = false;
            });
            return el("li", {}, [el("span", { text: f.title }), el("div", { className: "muted small", text: f.error }), discard]);
          }),
        ),
      );
    }
    return nodes;
  }

  const sync = el("section", {});

  function render(model) {
    const children = [
      el("h1", { text: model.status }),
      el("ul", { className: "progress" }, [...model.progress, model.streak].map((t) => el("li", { text: t }))),
      el("h2", { text: "Today's review" }),
      model.picks.length ? list(model.picks) : el("p", { className: "muted small", text: "No review picked today." }),
    ];
    if (model.due.length) {
      children.push(el("h2", { text: `Also due (${model.dueCount})` }), list(model.due));
    }
    if (model.dashboard) {
      children.push(el("p", {}, [el("a", { href: model.dashboard, target: "_blank", rel: "noopener noreferrer", text: "Open the dashboard" })]));
    }
    root.replaceChildren(...children, sync);
  }

  ext.runtime
    .sendMessage({ type: "today" })
    .then((res) => {
      // Waiting attempts show even when the app cannot be reached.
      sync.replaceChildren(...syncSection(res && res.outbox));
      if (!res || !res.ok || !res.data) throw new Error((res && res.error) || lib.describeStatus(res ? res.status : 0, ""));
      render(lib.popupModel(res.data, res.origin));
    })
    .catch((err) => {
      root.replaceChildren(el("h1", { text: "Leetgrinder" }), el("p", { className: "error", text: err.message || "Could not load today." }), sync);
    });
})();
