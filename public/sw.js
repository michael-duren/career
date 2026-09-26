// Public assets plus the data-free audio thoughts recorder shell may be cached.
// Private API responses, audio and entry HTML remain network-only. The recorder
// shell contains no saved entries; those are always loaded through private APIs.
// /running is the pre-rename path; offline it still opens the recorder shell.
const CACHE = 'career-public-v3';
const SHELL = '/audio-thoughts';
const ASSETS = ['/offline.html', '/manifest.webmanifest', '/apple-touch-icon.png', '/icons/icon-192.png', '/icons/icon-512.png'];
self.addEventListener('install', event => { event.waitUntil(caches.open(CACHE).then(cache => cache.addAll(ASSETS))); });
self.addEventListener('activate', event => {
  event.waitUntil((async () => { for (const name of await caches.keys()) if (name.startsWith('career-public-') && name !== CACHE) await caches.delete(name); await self.clients.claim(); })());
});
self.addEventListener('fetch', event => {
  const { request } = event; const url = new URL(request.url);
  if (request.method !== 'GET' || url.origin !== self.location.origin || url.pathname.startsWith('/api/')) return;
  if (request.mode === 'navigate') {
    const path = url.pathname.replace(/\/$/, ''), recorder = path === SHELL || path === '/running';
    event.respondWith((async () => {
      try {
        const response = await fetch(request);
        if (recorder && response.ok && !response.redirected && new URL(response.url).pathname.replace(/\/$/, '') === SHELL) await (await caches.open(CACHE)).put(SHELL, response.clone());
        return response;
      } catch { return (recorder && await caches.match(SHELL)) || await caches.match('/offline.html') || Response.error(); }
    })());
  } else if (ASSETS.includes(url.pathname) || url.pathname.startsWith('/_astro/')) {
    // Hashed Astro JS/CSS is public and needed by the offline recorder island.
    event.respondWith((async () => { const cached = await caches.match(request); if (cached) return cached; const response = await fetch(request); if (response.ok) await (await caches.open(CACHE)).put(request, response.clone()); return response; })());
  }
});
