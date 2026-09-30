// Toolbar popup: today's goal, streak, review picks and due list, read
// through the background worker. It never writes anything.
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
    root.replaceChildren(...children);
  }

  ext.runtime
    .sendMessage({ type: "today" })
    .then((res) => {
      if (!res || !res.ok || !res.data) throw new Error((res && res.error) || lib.describeStatus(res ? res.status : 0, ""));
      render(lib.popupModel(res.data, res.origin));
    })
    .catch((err) => {
      root.replaceChildren(el("h1", { text: "Leetgrinder" }), el("p", { className: "error", text: err.message || "Could not load today." }));
    });
})();
