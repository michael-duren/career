# Mobile app

Career Strategy can be installed from Safari’s Share menu → Add to Home Screen. Keep “Open as Web App” enabled if offered. Serve production over HTTPS; localhost also supports service workers for development.

The manifest and PNG icons are shared by the login and authenticated layouts. Standalone mode uses the same authenticated routes, with safe-area spacing around the notch and home indicator. Installation instructions appear in the navigation and disappear in standalone mode.

The service worker caches only the public offline page, manifest, and app icons. Private pages, API responses, credentials, and mutations are never added to Cache Storage. Offline navigation shows a retry screen; reconnecting and retrying loads the requested route through normal authentication. Existing editor draft recovery is unchanged. Offline editing/synchronization is not implemented.

When changing cached public resources, increment `CACHE` in `public/sw.js`. A new worker activates after existing app windows close, avoiding forced reloads while writing.

## Verification

Run `npm run dev -- --host 127.0.0.1 --port 4335`, then `node tests/pwa.browser.mjs`. The test uses the development login and a local Chromium/Brave binary (override with `TEST_BROWSER`), and `TEST_BASE_URL` can override the URL. It checks all page families and content routes at 375, 390, 430, and 844 pixels, navigation focus, search bounds, dialogs, login, cache contents, offline navigation, and reconnection. Screenshots are written to `/tmp/career-pwa-screenshots`.

Browser emulation does not replace checking installation, safe areas, and the software keyboard on a physical iPhone in Safari and standalone mode.
