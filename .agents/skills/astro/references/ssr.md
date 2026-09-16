---
title: Server-Side Rendering (SSR)
priority: high
category: patterns
---

# Server-Side Rendering (SSR)

Enable SSR for dynamic, per-request rendering instead of build-time static generation.

## Dynamic Routes with SSR

```astro
---
// src/pages/products/[id].astro
export const prerender = false; // Enable SSR for this page

const { id } = Astro.params;

// Fetch data on each request (not at build time)
const response = await fetch(`https://api.example.com/products/${id}`);
const product = await response.json();

if (!product) {
  return Astro.redirect('/404');
}
---

<article>
  <h1>{product.name}</h1>
  <p>{product.description}</p>
  <p>Price: ${product.price}</p>
</article>
```

---

## Cookies & Headers

```astro
---
// src/pages/dashboard.astro
export const prerender = false;

// Read cookies
const token = Astro.cookies.get('auth-token');

if (!token) {
  return Astro.redirect('/login');
}

// Set cookies
Astro.cookies.set('last-visit', new Date().toISOString(), {
  httpOnly: true,
  secure: true,
  maxAge: 60 * 60 * 24 * 7, // 7 days
});

// Read headers
const userAgent = Astro.request.headers.get('user-agent');
const clientIp = Astro.clientAddress;
---

<h1>Dashboard</h1>
<p>Token: {token.value}</p>
<p>User Agent: {userAgent}</p>
<p>IP: {clientIp}</p>
```

---

## Mixed Rendering (Static + SSR)

> **Astro 5**: `output: 'hybrid'` was removed. Use `output: 'static'` with `prerender = false` on individual pages.

Mix static and SSR pages in the same project:

```javascript
// astro.config.mjs
export default defineConfig({
  output: 'static', // Default: all pages static
  adapter: vercel(), // Required for SSR
});
```

```astro
---
// src/pages/about.astro
// This page is static (pre-rendered at build time) - default behavior
// No need for prerender = true (it's the default)
---

---
// src/pages/dashboard.astro
// This page is SSR (rendered on each request)
export const prerender = false; // Opt-out of prerendering
---
```

---

## Astro 6.0 Integration Changes

> **Astro 6 Update**: Vite's new Environment API is now used for build configuration. Integration authors should note these changes:

### astro:build:setup Hook

```typescript
// Astro 5.x (deprecated)
hooks: {
  'astro:build:setup': ({ target, vite }) => {
    if (target === 'client') {
      vite.build.minify = false;
    }
  }
}

// Astro 6.0 (new)
hooks: {
  'astro:build:setup': ({ vite }) => {
    vite.environments.client.build.minify = false;
  }
}
```

### HMR Access Pattern

```typescript
// Astro 5.x (deprecated)
server.hot.send(event)

// Astro 6.0 (new)
server.environments.client.hot.send(event)
```

### entryPoints Removed

```typescript
// Astro 5.x (deprecated)
hooks: {
  'astro:build:ssr': (params) => {
    someLogic(params.entryPoints) // Now removed
  }
}
```

---

## Resources

- **SSR Guide**: https://docs.astro.build/en/guides/server-side-rendering/
