---
title: Middleware
priority: medium
category: patterns
---

# Middleware (Request Interception)

Middleware allows you to intercept requests and responses, useful for auth, logging, and custom headers.

## Basic Middleware

```typescript
// src/middleware.ts
import { defineMiddleware } from 'astro:middleware';

export const onRequest = defineMiddleware(async (context, next) => {
  // Add custom headers
  context.response.headers.set('X-Custom-Header', 'Hello');
  
  // Check authentication
  const token = context.cookies.get('auth-token');
  
  if (context.url.pathname.startsWith('/admin') && !token) {
    return context.redirect('/login');
  }
  
  // Log request
  console.log(`${context.request.method} ${context.url.pathname}`);
  
  // Continue to page/endpoint
  const response = await next();
  
  // Modify response (optional)
  return response;
});
```

---

## Storing Data in context.locals

Use `context.locals` to pass data from middleware to pages and API endpoints.

> **Astro 5+ Breaking Change**: You can no longer completely replace `context.locals`. Use `Object.assign()` instead:

```typescript
// ✅ Correct (Astro 5+)
Object.assign(context.locals, {
  user: await getUser(token),
  timestamp: Date.now(),
});

// ❌ Incorrect - Don't replace locals entirely
context.locals = {
  user: await getUser(token),
  timestamp: Date.now(),
};
```

> **Astro 6 Note**: Middleware API remains unchanged from Astro 5.

---

## Use Cases

| Use Case | Example |
|----------|---------|
| Authentication | Check auth tokens before protected routes |
| Logging | Log all requests with timestamps |
| Custom Headers | Add CORS, security headers |
| Rate Limiting | Limit API calls per user |
| Redirects | Redirect based on conditions |

---

## Resources

- **Middleware Guide**: https://docs.astro.build/en/guides/middleware/
