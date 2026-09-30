(() => {
  "use strict";
  const lib = globalThis.LeetgrinderLib;
  const form = document.getElementById("options");
  const originInput = document.getElementById("origin");
  const tokenInput = document.getElementById("token");
  const status = document.getElementById("status");

  function show(message, error = false) {
    status.textContent = message;
    status.className = error ? "error" : "";
  }

  ext.storage.local.get(["origin", "token"]).then(({ origin, token }) => {
    if (origin) originInput.value = origin;
    if (token) tokenInput.value = token;
  });

  form.addEventListener("submit", (event) => {
    event.preventDefault();
    const origin = lib.normalizeOrigin(originInput.value);
    const token = tokenInput.value.trim();
    if (!origin) return show("Enter an https:// origin, or http://localhost for development.", true);
    if (!lib.validToken(token)) return show("Paste the full token from the app. It starts with lg_.", true);
    // Firefox only allows permission prompts synchronously inside the click
    // handler, so request before any other await.
    const pattern = lib.originPattern(origin);
    ext.permissions.request({ origins: [pattern] }).then(
      async (granted) => {
        if (!granted) return show("Access to the app origin is required to log attempts.", true);
        const previous = (await ext.storage.local.get("origin")).origin;
        await ext.storage.local.set({ origin, token });
        originInput.value = origin;
        const previousOrigin = lib.normalizeOrigin(previous);
        if (previousOrigin && lib.originPattern(previousOrigin) !== pattern) {
          await ext.permissions.remove({ origins: [lib.originPattern(previousOrigin)] }).catch(() => {});
        }
        show("Saved. Use Test connection to check the token.");
      },
      () => show("The browser refused the permission request.", true),
    );
  });

  document.getElementById("test").addEventListener("click", async () => {
    show("Testing…");
    const res = await ext.runtime.sendMessage({ type: "test" }).catch(() => null);
    if (!res) return show("The extension's background worker did not answer.", true);
    if (res.ok && res.data && res.data.slug === "two-sum") return show("Connected. The token works.");
    show(res.error || lib.describeStatus(res.status, ""), true);
  });
})();
