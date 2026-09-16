---
title: API Endpoints
priority: high
category: patterns
---

# API Endpoints

Create server-side API routes that handle GET, POST, PUT, DELETE requests.

## Basic API Route

```typescript
// src/pages/api/search.json.ts
import type { APIRoute } from 'astro';

export const GET: APIRoute = async ({ request }) => {
  const url = new URL(request.url);
  const query = url.searchParams.get('q');
  
  if (!query) {
    return new Response(JSON.stringify({ error: 'Missing query' }), {
      status: 400,
      headers: { 'Content-Type': 'application/json' },
    });
  }
  
  // Fetch from database or external API
  const results = await searchDatabase(query);
  
  return new Response(JSON.stringify({ results }), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  });
};

export const POST: APIRoute = async ({ request }) => {
  const data = await request.json();
  
  // Process data (save to DB, send email, etc.)
  const result = await saveToDatabase(data);
  
  return new Response(JSON.stringify({ success: true, id: result.id }), {
    status: 201,
    headers: { 'Content-Type': 'application/json' },
  });
};
```

---

## RSS Feed Generation

```typescript
// src/pages/rss.xml.ts
import rss from '@astrojs/rss';
import { getCollection } from 'astro:content';

export async function GET(context) {
  const posts = await getCollection('blog');
  
  return rss({
    title: 'My Blog',
    description: 'A blog about web development',
    site: context.site,
    items: posts.map((post) => ({
      title: post.data.title,
      description: post.data.description,
      pubDate: post.data.publishDate,
      link: `/blog/${post.slug}`,
    })),
    customData: '<language>en-us</language>',
  });
}
```

---

## Sitemap Generation

```bash
npx astro add sitemap
```

```javascript
// astro.config.mjs
import sitemap from '@astrojs/sitemap';

export default defineConfig({
  site: 'https://example.com', // Required for sitemap
  integrations: [sitemap()],
});
```

---

## Resources

- **API Endpoints Guide**: https://docs.astro.build/en/guides/endpoints/
