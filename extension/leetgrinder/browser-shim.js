// Firefox exposes promise-based `browser`; Chrome MV3's `chrome` returns
// promises too. Everything else in the extension uses `ext`.
globalThis.ext = globalThis.browser ?? globalThis.chrome;
