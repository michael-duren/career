---
title: Dark Mode Recipe
priority: low
category: examples
---

# Recipe: Dark Mode Toggle

Client-side dark mode with localStorage persistence.

## Theme Toggle Component

```astro
---
// src/components/ThemeToggle.astro
---

<button id="theme-toggle" aria-label="Toggle dark mode">
  <span class="sun">☀️</span>
  <span class="moon">🌙</span>
</button>

<script>
  const theme = (() => {
    if (typeof localStorage !== 'undefined' && localStorage.getItem('theme')) {
      return localStorage.getItem('theme');
    }
    if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
      return 'dark';
    }
    return 'light';
  })();

  if (theme === 'dark') {
    document.documentElement.classList.add('dark');
  } else {
    document.documentElement.classList.remove('dark');
  }

  window.localStorage.setItem('theme', theme);

  const handleToggle = () => {
    const element = document.documentElement;
    element.classList.toggle('dark');

    const isDark = element.classList.contains('dark');
    localStorage.setItem('theme', isDark ? 'dark' : 'light');
  };

  document
    .getElementById('theme-toggle')
    ?.addEventListener('click', handleToggle);
</script>

<style>
  #theme-toggle {
    background: transparent;
    border: none;
    cursor: pointer;
    font-size: 1.5rem;
  }

  .sun { display: none; }
  .moon { display: block; }

  :global(.dark) .sun { display: block; }
  :global(.dark) .moon { display: none; }
</style>
```

---

## Global Styles with Dark Mode

```css
/* src/styles/global.css */
:root {
  --bg: #ffffff;
  --text: #000000;
  --accent: #3b82f6;
}

.dark {
  --bg: #1a1a1a;
  --text: #ffffff;
  --accent: #60a5fa;
}

body {
  background: var(--bg);
  color: var(--text);
}

a {
  color: var(--accent);
}
```

---

## Prevent Flash of Unstyled Content (FOUC)

Add this inline script in `<head>` before any content:

```astro
---
// src/layouts/BaseLayout.astro
---
<html>
  <head>
    <script is:inline>
      const theme = (() => {
        if (typeof localStorage !== 'undefined' && localStorage.getItem('theme')) {
          return localStorage.getItem('theme');
        }
        if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
          return 'dark';
        }
        return 'light';
      })();
      
      if (theme === 'dark') {
        document.documentElement.classList.add('dark');
      }
    </script>
    
    <link rel="stylesheet" href="/styles/global.css" />
  </head>
  <body>
    <slot />
  </body>
</html>
```

**Note**: Use `is:inline` to prevent Astro from bundling/processing the script.

---

## Resources

- Client-side scripts run in browser, persist theme across page loads
