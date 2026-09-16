---
title: Internationalization (i18n) Recipe
priority: medium
category: examples
---

# Recipe: Internationalization (i18n)

Multi-language support with translation files and dynamic routing.

## Directory Structure

```
src/
├── pages/
│   ├── index.astro           # Default (English)
│   ├── es/
│   │   └── index.astro       # Spanish
│   └── fr/
│       └── index.astro       # French
└── i18n/
    ├── en.json
    ├── es.json
    └── fr.json
```

---

## Translation Files

```json
// src/i18n/en.json
{
  "nav.home": "Home",
  "nav.about": "About",
  "nav.contact": "Contact",
  "hero.title": "Welcome to My Site",
  "hero.subtitle": "Building amazing web experiences"
}
```

```json
// src/i18n/es.json
{
  "nav.home": "Inicio",
  "nav.about": "Acerca de",
  "nav.contact": "Contacto",
  "hero.title": "Bienvenido a Mi Sitio",
  "hero.subtitle": "Construyendo experiencias web increíbles"
}
```

---

## Translation Helper

```typescript
// src/i18n/utils.ts
const translations = {
  en: () => import('./en.json').then(m => m.default),
  es: () => import('./es.json').then(m => m.default),
  fr: () => import('./fr.json').then(m => m.default),
};

export async function getTranslations(lang: keyof typeof translations) {
  return await translations[lang]();
}

export function getLangFromURL(url: URL) {
  const [, lang] = url.pathname.split('/');
  if (lang in translations) return lang as keyof typeof translations;
  return 'en';
}
```

---

## Multilingual Page

```astro
---
// src/pages/es/index.astro
import { getTranslations } from '../../i18n/utils';

const t = await getTranslations('es');
---

<html lang="es">
  <head>
    <title>{t['hero.title']}</title>
  </head>
  <body>
    <nav>
      <a href="/es">{t['nav.home']}</a>
      <a href="/es/about">{t['nav.about']}</a>
      <a href="/es/contact">{t['nav.contact']}</a>
    </nav>
    
    <main>
      <h1>{t['hero.title']}</h1>
      <p>{t['hero.subtitle']}</p>
    </main>
  </body>
</html>
```

---

## Language Switcher Component

```astro
---
// src/components/LanguageSwitcher.astro
import { getLangFromURL } from '../i18n/utils';

const currentLang = getLangFromURL(Astro.url);
const currentPath = Astro.url.pathname.replace(/^\/(en|es|fr)/, '') || '/';

const languages = [
  { code: 'en', name: 'English' },
  { code: 'es', name: 'Español' },
  { code: 'fr', name: 'Français' },
];
---

<select id="lang-switcher">
  {languages.map(lang => (
    <option 
      value={`/${lang.code}${currentPath}`}
      selected={lang.code === currentLang}
    >
      {lang.name}
    </option>
  ))}
</select>

<script>
  document.getElementById('lang-switcher')?.addEventListener('change', (e) => {
    const target = e.target as HTMLSelectElement;
    window.location.href = target.value;
  });
</script>
```

---

## Astro i18n Routing (Built-in)

Astro 5.x has experimental built-in i18n routing:

```javascript
// astro.config.mjs
export default defineConfig({
  i18n: {
    defaultLocale: 'en',
    locales: ['en', 'es', 'fr'],
    routing: {
      prefixDefaultLocale: false,
    },
  },
});
```

---

## Resources

- **i18n Routing**: https://docs.astro.build/en/guides/internationalization/
