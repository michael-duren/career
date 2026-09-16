---
title: SEO Recipe
priority: medium
category: examples
---

# Recipe: SEO with Dynamic Metadata

Comprehensive SEO component with Open Graph, Twitter Cards, and structured data.

## SEO Component

```astro
---
// src/components/SEO.astro
interface Props {
  title: string;
  description: string;
  image?: string;
  article?: boolean;
  publishDate?: Date;
  author?: string;
}

const {
  title,
  description,
  image = '/default-og.jpg',
  article = false,
  publishDate,
  author,
} = Astro.props;

const canonicalURL = new URL(Astro.url.pathname, Astro.site);
const imageURL = new URL(image, Astro.site);
---

<!-- Primary Meta Tags -->
<title>{title}</title>
<meta name="title" content={title} />
<meta name="description" content={description} />
<link rel="canonical" href={canonicalURL} />

<!-- Open Graph / Facebook -->
<meta property="og:type" content={article ? 'article' : 'website'} />
<meta property="og:url" content={canonicalURL} />
<meta property="og:title" content={title} />
<meta property="og:description" content={description} />
<meta property="og:image" content={imageURL} />

{article && publishDate && (
  <meta property="article:published_time" content={publishDate.toISOString()} />
)}
{article && author && (
  <meta property="article:author" content={author} />
)}

<!-- Twitter -->
<meta property="twitter:card" content="summary_large_image" />
<meta property="twitter:url" content={canonicalURL} />
<meta property="twitter:title" content={title} />
<meta property="twitter:description" content={description} />
<meta property="twitter:image" content={imageURL} />

<!-- Additional Meta -->
<meta name="robots" content="index, follow" />
<meta name="googlebot" content="index, follow" />
```

---

## Usage

```astro
---
import SEO from '../components/SEO.astro';
---

<html>
  <head>
    <SEO 
      title="My Blog Post"
      description="An amazing blog post about Astro"
      image="/blog/post-image.jpg"
      article={true}
      publishDate={new Date('2024-01-15')}
      author="John Doe"
    />
  </head>
  <body>
    <!-- Content -->
  </body>
</html>
```

---

## JSON-LD Structured Data

```astro
---
// src/components/StructuredData.astro
interface Props {
  type: 'Article' | 'WebPage' | 'Organization';
  data: Record<string, any>;
}

const { type, data } = Astro.props;

const structuredData = {
  '@context': 'https://schema.org',
  '@type': type,
  ...data,
};
---

<script type="application/ld+json" set:html={JSON.stringify(structuredData)} />
```

**Usage for Article:**

```astro
<StructuredData
  type="Article"
  data={{
    headline: post.data.title,
    image: post.data.image,
    datePublished: post.data.publishDate.toISOString(),
    author: {
      '@type': 'Person',
      name: post.data.author,
    },
  }}
/>
```

---

## Resources

- **SEO Guide**: https://docs.astro.build/en/guides/seo/
