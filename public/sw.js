// Only public app resources are cached. Authenticated HTML and APIs remain network-only.
const CACHE = 'career-public-v1';
const ASSETS = ['/offline.html', '/manifest.webmanifest', '/apple-touch-icon.png', '/icons/icon-192.png', '/icons/icon-512.png'];
self.addEventListener('install', event => {
  event.waitUntil(caches.open(CACHE).then(cache => cache.addAll(ASSETS)));
});
self.addEventListener('activate', event => {
  event.waitUntil((async () => {
    for (const name of await caches.keys()) {
      if (name.startsWith('career-public-') && name !== CACHE) await caches.delete(name);
    }
    await self.clients.claim();
  })());
});
self.addEventListener('fetch', event => {
  const { request } = event;
  const url = new URL(request.url);
  if (request.method !== 'GET' || url.origin !== self.location.origin) return;
  if (request.mode === 'navigate') {
    event.respondWith(fetch(request).catch(async () => (await caches.match('/offline.html')) || Response.error()));
  } else if (ASSETS.includes(url.pathname)) {
    event.respondWith(caches.match(request).then(cached => cached || fetch(request)));
  }
});
