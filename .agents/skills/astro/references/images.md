---
title: Image Optimization
priority: high
category: patterns
---

# Image Optimization

Astro provides built-in image optimization with automatic WebP conversion and responsive sizes.

## Built-in Image Component

```astro
---
import { Image } from 'astro:assets';
import heroImage from '../assets/hero.jpg'; // Local import
---

<!-- ✅ Optimized: auto WebP, responsive sizes -->
<Image 
  src={heroImage} 
  alt="Hero image"
  width={1200}
  height={630}
  format="webp"
  quality={80}
/>

<!-- Remote images -->
<Image 
  src="https://example.com/image.jpg"
  alt="Remote image"
  width={800}
  height={600}
  inferSize
/>
```

---

## Picture Component (Art Direction)

Use Picture for different images at different breakpoints:

```astro
---
import { Picture } from 'astro:assets';
import desktopImage from '../assets/hero-desktop.jpg';
import mobileImage from '../assets/hero-mobile.jpg';
---

<Picture 
  src={desktopImage}
  formats={['avif', 'webp']}
  alt="Responsive hero"
  widths={[400, 800, 1200]}
  sizes="(max-width: 768px) 100vw, 1200px"
  fallbackFormat="jpg"
/>
```

---

## Lazy Load Images Below Fold

```astro
---
import { Image } from 'astro:assets';
import heroImage from '../assets/hero.jpg';
import belowFoldImage from '../assets/below-fold.jpg';
---

<!-- Above fold: eager loading -->
<Image 
  src={heroImage} 
  alt="Hero" 
  loading="eager"
  fetchpriority="high"
/>

<!-- Below fold: lazy loading -->
<Image 
  src={belowFoldImage} 
  alt="Below fold" 
  loading="lazy"
/>
```

---

## Preload Critical Assets

```astro
<head>
  <link rel="preload" as="image" href="/hero.jpg" />
  <link rel="preload" as="font" href="/fonts/inter.woff2" type="font/woff2" crossorigin />
</head>
```

---

## Resources

- **Image Optimization Guide**: https://docs.astro.build/en/guides/images/
