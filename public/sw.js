// Public assets plus the data-free running recorder shell may be cached.
// Private API responses, audio and entry HTML remain network-only. The running
// shell contains no saved entries; those are always loaded through private APIs.
const CACHE = 'career-public-v2';
const ASSETS = ['/offline.html', '/manifest.webmanifest', '/apple-touch-icon.png', '/icons/icon-192.png', '/icons/icon-512.png'];
self.addEventListener('install', event => { event.waitUntil(caches.open(CACHE).then(cache => cache.addAll(ASSETS))); });
self.addEventListener('activate', event => {
  event.waitUntil((async () => { for (const name of await caches.keys()) if (name.startsWith('career-public-') && name !== CACHE) await caches.delete(name); await self.clients.claim(); })());
});
self.addEventListener('fetch', event => {
  const { request } = event; const url = new URL(request.url);
  if (request.method !== 'GET' || url.origin !== self.location.origin || url.pathname.startsWith('/api/')) return;
  if (request.mode === 'navigate') {
    const running = url.pathname.replace(/\/$/, '') === '/running';
    event.respondWith((async () => {
      try {
        const response = await fetch(request);
        if (running && response.ok && !response.redirected && new URL(response.url).pathname.replace(/\/$/, '') === '/running') await (await caches.open(CACHE)).put('/running', response.clone());
        return response;
      } catch { return (running && await caches.match('/running')) || await caches.match('/offline.html') || Response.error(); }
    })());
  } else if (ASSETS.includes(url.pathname) || url.pathname.startsWith('/_astro/')) {
    // Hashed Astro JS/CSS is public and needed by the offline recorder island.
    event.respondWith((async () => { const cached = await caches.match(request); if (cached) return cached; const response = await fetch(request); if (response.ok) await (await caches.open(CACHE)).put(request, response.clone()); return response; })());
  }
});
