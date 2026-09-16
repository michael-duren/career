---
title: Authentication Recipe
priority: medium
category: examples
---

# Recipe: Authentication with Cookies

Cookie-based authentication with protected routes.

## Login Page

```astro
---
// src/pages/login.astro
import Layout from '../layouts/Layout.astro';

const error = Astro.url.searchParams.get('error');
---

<Layout title="Login">
  <h1>Login</h1>
  
  {error && <p class="error">{error}</p>}
  
  <form action="/api/auth/login" method="POST">
    <input type="email" name="email" placeholder="Email" required />
    <input type="password" name="password" placeholder="Password" required />
    <button type="submit">Login</button>
  </form>
</Layout>
```

---

## Login API Endpoint

```typescript
// src/pages/api/auth/login.ts
import type { APIRoute } from 'astro';

export const POST: APIRoute = async ({ request, cookies, redirect }) => {
  const formData = await request.formData();
  const email = formData.get('email') as string;
  const password = formData.get('password') as string;

  // Validate credentials (example - use proper auth library in production)
  if (email === 'user@example.com' && password === 'password') {
    // Set auth cookie
    cookies.set('auth-token', 'your-secure-token', {
      httpOnly: true,
      secure: true,
      sameSite: 'strict',
      maxAge: 60 * 60 * 24 * 7, // 7 days
      path: '/',
    });

    return redirect('/dashboard');
  }

  return redirect('/login?error=Invalid credentials');
};
```

---

## Protected Page

```astro
---
// src/pages/dashboard.astro
export const prerender = false; // Enable SSR

const token = Astro.cookies.get('auth-token');

if (!token) {
  return Astro.redirect('/login');
}

// Fetch user data
const user = await getUserFromToken(token.value);
---

<Layout title="Dashboard">
  <h1>Welcome, {user.name}!</h1>
  
  <form action="/api/auth/logout" method="POST">
    <button type="submit">Logout</button>
  </form>
</Layout>
```

---

## Logout Endpoint

```typescript
// src/pages/api/auth/logout.ts
import type { APIRoute } from 'astro';

export const POST: APIRoute = async ({ cookies, redirect }) => {
  cookies.delete('auth-token', { path: '/' });
  return redirect('/');
};
```

---

## Middleware Protection

```typescript
// src/middleware.ts
import { defineMiddleware } from 'astro:middleware';

export const onRequest = defineMiddleware(async (context, next) => {
  const token = context.cookies.get('auth-token');
  
  // Protect /dashboard and /admin routes
  if (context.url.pathname.startsWith('/dashboard') || 
      context.url.pathname.startsWith('/admin')) {
    if (!token) {
      return context.redirect('/login');
    }
  }
  
  return next();
});
```

---

## Resources

- See [middleware.md](middleware.md) for middleware patterns
- See [ssr.md](ssr.md) for cookies and headers
