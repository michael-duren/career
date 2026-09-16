# Astro Best Practices Skill

[![Install with Skills](https://img.shields.io/badge/skills-install-blue?style=for-the-badge)](https://skills.sh)
[![GitHub](https://img.shields.io/github/stars/BriantBR/astro-best-practices?style=for-the-badge)](https://github.com/BriantBR/astro-best-practices)

A comprehensive skill for building production-ready **Astro 6.x** applications with Islands Architecture, client directives, and modern web patterns.

> **🚀 Astro 6 Ready**: Updated with all Astro 6 breaking changes including Vite Environments API, legacy Content Collections removal, new HMR patterns, and `process.env` for environment variables.
>
> **AI-Powered Development**: This skill enhances AI coding assistants (Claude, Cursor, Windsurf) with expert Astro patterns and best practices.

## What This Skill Covers

- **Islands Architecture**: Zero JavaScript by default with selective hydration
- **Client Directives**: `client:load`, `client:idle`, `client:visible`, `client:media`, `client:only`
- **Multi-Framework Support**: Mix React, Vue, and Svelte in the same project
- **Content Collections**: Type-safe CMS with Content Layer API - Astro 6
- **View Transitions**: SPA-like navigation with `<ClientRouter />` - Astro 5+
- **Advanced Patterns**: SSR, middleware, image optimization, API endpoints
- **Real-World Examples**: Blogs, forms, authentication, i18n, SEO, dark mode

## Astro 6.0 Migration Guide

This skill includes comprehensive migration guidance for Astro 6.0:

### Breaking Changes Covered

- **Vite Environments API**: `astro:build:setup` hook uses `vite.environments` instead of `target`
- **HMR Access Pattern**: Changed from `server.hot.send()` to `server.environments.client.hot.send()`
- **Environment Variables**: Use `process.env` directly instead of `import.meta.env` transformation
- **Legacy Content Collections**: Removed entirely - must use Content Layer API
- **entryPoints removed**: Map removed from `astro:build:ssr` hook
- **Deprecated Astro.site/generator**: Use `import.meta.env.SITE` in `getStaticPaths()`
- **Image Metadata**: `emitESMImage()` replaced by `emitImageMetadata()`

### Experimental Flags Now Stable

- `liveContentCollections` is now stable as **Live Collections**

## Installation

Install this skill using the skills CLI:

```bash
npx skills add BriantBR/astro-best-practices
```

That's it! Your AI assistant will now have access to Astro best practices and patterns.

### Supported AI Assistants

- ✅ Claude (Anthropic)
- ✅ Cursor
- ✅ Windsurf
- ✅ Any agent supporting the skills.sh format

### Manual Installation

If you prefer to install manually, copy `SKILL.md` and the `references/` directory to your agent's skills directory.

## Usage

This skill is designed to be used with AI assistants that support the skills.sh format. Once installed, your assistant will automatically apply these best practices when working with Astro projects.

### When This Skill Activates

- Building content-focused sites
- Implementing partial hydration with islands
- Setting up multi-framework projects
- Optimizing performance with zero-JS by default
- Deploying static or SSR sites

## Contents

### Core Documentation

- **[SKILL.md](SKILL.md)**: Core principles, quick start, and reference index

### Feature References (Load on-demand)

**High Priority** (Common use cases):
- **[Content Collections](references/content-collections.md)**: Type-safe CMS for Markdown/MDX
- **[Image Optimization](references/images.md)**: Built-in Image and Picture components
- **[Server-Side Rendering](references/ssr.md)**: Dynamic routes, cookies, headers
- **[API Endpoints](references/api-endpoints.md)**: Server-side API routes, RSS, sitemap

**Medium Priority** (Specific features):
- **[Middleware](references/middleware.md)**: Request interception for auth, logging
- **[Pagination](references/pagination.md)**: Paginate large collections
- **[Environment & Config](references/environment-config.md)**: Environment variables, TypeScript helpers

**Low Priority** (Advanced):
- **[Custom Integrations](references/integrations.md)**: Extend Astro's build process

### Real-World Recipes

Complete examples for common scenarios:
- **[Blog Setup](references/recipe-blog.md)**: Full blog with tags, RSS, layouts
- **[Multi-Step Forms](references/recipe-forms.md)**: React form with API integration
- **[Authentication](references/recipe-auth.md)**: Cookie-based auth with protected routes
- **[SEO](references/recipe-seo.md)**: Open Graph, Twitter Cards, structured data
- **[i18n](references/recipe-i18n.md)**: Multi-language support
- **[Dark Mode](references/recipe-dark-mode.md)**: Theme toggle with localStorage

## Quick Start Example

Once installed, your AI assistant will automatically apply these patterns:

```astro
---
// Your AI will know to keep components static by default
import Header from '../components/Header.astro';
import Counter from '../components/Counter.jsx';
---

<html>
  <body>
    <!-- Static HTML (no JS) -->
    <Header />
    
    <!-- Only hydrate when needed -->
    <Counter client:idle />
  </body>
</html>
```

## Why Astro?

Astro is perfect for content-focused websites that need:
- **Speed**: Ship zero JavaScript by default
- **Flexibility**: Use any framework (React, Vue, Svelte) or mix them
- **Developer Experience**: Modern tooling with TypeScript support
- **SEO**: Server-side rendering with excellent Core Web Vitals

## Contributing

Found an issue or want to add more patterns? Contributions are welcome!

1. Fork the repo
2. Create a feature branch: `git checkout -b feature/new-pattern`
3. Commit your changes: `git commit -m "feat: add new pattern"`
4. Push to the branch: `git push origin feature/new-pattern`
5. Open a Pull Request

## Links

- 📦 [GitHub Repository](https://github.com/BriantBR/astro-best-practices)
- 🚀 [Skills.sh Ecosystem](https://skills.sh)
- 📚 [Astro Documentation](https://docs.astro.build)

## License

MIT

## Author

**BriantDev** - [@BriantBR](https://github.com/BriantBR)

---

<div align="center">
  
**[⭐ Star this repo](https://github.com/BriantBR/astro-best-practices)** if you find it useful!

Made with ❤️ for the Astro community

</div>
