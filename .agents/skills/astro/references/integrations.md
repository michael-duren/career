---
title: Custom Integrations
priority: low
category: patterns
---

# Custom Integrations

Create custom integrations to extend Astro's build process and add functionality.

## Basic Integration

```typescript
// integrations/my-integration.ts
import type { AstroIntegration } from 'astro';

export function myIntegration(): AstroIntegration {
  return {
    name: 'my-integration',
    hooks: {
      'astro:config:setup': ({ config, updateConfig, injectScript }) => {
        // Add custom Vite config
        updateConfig({
          vite: {
            plugins: [/* custom Vite plugins */],
          },
        });
        
        // Inject script into every page
        injectScript('page', 'console.log("Integration loaded");');
      },
      'astro:build:start': ({ buildConfig }) => {
        console.log('Build started:', buildConfig.outDir);
      },
      'astro:build:done': ({ dir, pages }) => {
        console.log(`Build complete: ${pages.length} pages in ${dir}`);
      },
    },
  };
}
```

```javascript
// astro.config.mjs
import { myIntegration } from './integrations/my-integration';

export default defineConfig({
  integrations: [myIntegration()],
});
```

---

## Available Hooks

| Hook | When It Runs |
|------|--------------|
| `astro:config:setup` | Before config finalization |
| `astro:config:done` | After config finalization |
| `astro:server:setup` | Dev server setup |
| `astro:server:start` | Dev server started |
| `astro:build:start` | Build started |
| `astro:build:setup` | Before build |
| `astro:build:done` | Build completed |

---

## Resources

- **Custom Integrations**: https://docs.astro.build/en/reference/integrations-reference/
