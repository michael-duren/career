---
title: Blog Recipe
priority: high
category: examples
---

# Recipe: Blog with Content Collections

Complete blog setup with type-safe content, tags, and RSS feed.

## Directory Structure

```
src/
├── content/
│   └── blog/
│       ├── first-post.md
│       ├── second-post.mdx
│       └── third-post.md
├── layouts/
│   └── BlogLayout.astro
├── pages/
│   ├── blog/
│   │   ├── index.astro
│   │   ├── [slug].astro
│   │   └── tags/
│   │       └── [tag].astro
│   └── rss.xml.ts
└── content.config.ts    # Content collections config (Astro 5+)
```

---

## Blog Schema

```typescript
// src/content.config.ts
import { defineCollection } from 'astro:content';
import { glob } from 'astro/loaders';
import { z } from 'astro/zod';

const blog = defineCollection({
  loader: glob({ pattern: '**/*.md', base: './src/content/blog' }),
  schema: z.object({
    title: z.string(),
    description: z.string(),
    publishDate: z.coerce.date(),
    updatedDate: z.coerce.date().optional(),
    author: z.string(),
    tags: z.array(z.string()),
    image: z.string().optional(),
    draft: z.boolean().default(false),
  }),
});

export const collections = { blog };
```
src/
├── content/
│   ├── config.ts
│   └── blog/
│       ├── first-post.md
│       ├── second-post.mdx
│       └── third-post.md
├── layouts/
│   └── BlogLayout.astro
└── pages/
    ├── blog/
    │   ├── index.astro
    │   ├── [slug].astro
    │   └── tags/
    │       └── [tag].astro
    └── rss.xml.ts
```

---

## Blog Schema

```typescript
// src/content/config.ts
import { defineCollection, z } from 'astro:content';

const blog = defineCollection({
  schema: ({ image }) => z.object({
    title: z.string(),
    description: z.string(),
    publishDate: z.coerce.date(),
    updatedDate: z.coerce.date().optional(),
    author: z.string(),
    tags: z.array(z.string()),
    image: image().optional(),
    draft: z.boolean().default(false),
  }),
});

export const collections = { blog };
```

---

## Blog Index Page

```astro
---
// src/pages/blog/index.astro
import { getCollection } from 'astro:content';
import Layout from '../../layouts/Layout.astro';

const posts = await getCollection('blog', ({ data }) => !data.draft);
const sortedPosts = posts.sort((a, b) => 
  b.data.publishDate.getTime() - a.data.publishDate.getTime()
);

// Get all unique tags
const allTags = [...new Set(posts.flatMap(post => post.data.tags))];
---

<Layout title="Blog">
  <h1>Blog Posts</h1>
  
  <!-- Tag filter -->
  <nav>
    <a href="/blog">All</a>
    {allTags.map(tag => (
      <a href={`/blog/tags/${tag}`}>{tag}</a>
    ))}
  </nav>
  
  <!-- Posts list -->
  <section>
    {sortedPosts.map(post => (
      <article>
        <a href={`/blog/${post.slug}`}>
          <h2>{post.data.title}</h2>
          <time datetime={post.data.publishDate.toISOString()}>
            {post.data.publishDate.toLocaleDateString('en-US', {
              year: 'numeric',
              month: 'long',
              day: 'numeric',
            })}
          </time>
          <p>{post.data.description}</p>
        </a>
      </article>
    ))}
  </section>
</Layout>
```

---

## Single Blog Post

```astro
---
// src/pages/blog/[slug].astro
import { getCollection } from 'astro:content';
import { Image } from 'astro:assets';
import BlogLayout from '../../layouts/BlogLayout.astro';

export async function getStaticPaths() {
  const posts = await getCollection('blog');
  return posts.map(post => ({
    params: { slug: post.slug },
    props: { post },
  }));
}

const { post } = Astro.props;
const { Content, headings } = await post.render();
---

<BlogLayout 
  title={post.data.title}
  description={post.data.description}
  publishDate={post.data.publishDate}
  author={post.data.author}
>
  <article>
    <header>
      {post.data.image && (
        <Image 
          src={post.data.image} 
          alt={post.data.title}
          width={1200}
          height={630}
        />
      )}
      
      <h1>{post.data.title}</h1>
      
      <div class="meta">
        <time datetime={post.data.publishDate.toISOString()}>
          {post.data.publishDate.toLocaleDateString()}
        </time>
        <span>By {post.data.author}</span>
      </div>
      
      <ul class="tags">
        {post.data.tags.map(tag => (
          <li><a href={`/blog/tags/${tag}`}>{tag}</a></li>
        ))}
      </ul>
    </header>
    
    <!-- Table of contents -->
    {headings.length > 0 && (
      <aside class="toc">
        <h2>Table of Contents</h2>
        <ul>
          {headings.map(heading => (
            <li class={`level-${heading.depth}`}>
              <a href={`#${heading.slug}`}>{heading.text}</a>
            </li>
          ))}
        </ul>
      </aside>
    )}
    
    <!-- Rendered content -->
    <Content />
  </article>
</BlogLayout>
```

---

## Blog Layout

```astro
---
// src/layouts/BlogLayout.astro
import BaseLayout from './BaseLayout.astro';

interface Props {
  title: string;
  description: string;
  publishDate: Date;
  author: string;
}

const { title, description, publishDate, author } = Astro.props;
---

<BaseLayout title={title} description={description}>
  <slot />
  
  <!-- Share buttons (static, no JS) -->
  <aside class="share">
    <h3>Share this post</h3>
    <a 
      href={`https://twitter.com/intent/tweet?text=${encodeURIComponent(title)}&url=${Astro.url}`}
      target="_blank"
      rel="noopener"
    >
      Twitter
    </a>
    <a 
      href={`https://www.linkedin.com/sharing/share-offsite/?url=${Astro.url}`}
      target="_blank"
      rel="noopener"
    >
      LinkedIn
    </a>
  </aside>
</BaseLayout>

<style>
  article {
    max-width: 65ch;
    margin: 0 auto;
    padding: 2rem;
  }
  
  .meta {
    display: flex;
    gap: 1rem;
    color: #666;
  }
  
  .tags {
    display: flex;
    gap: 0.5rem;
    list-style: none;
    padding: 0;
  }
  
  .toc {
    background: #f5f5f5;
    padding: 1rem;
    border-radius: 8px;
    margin: 2rem 0;
  }
  
  .toc .level-2 { margin-left: 1rem; }
  .toc .level-3 { margin-left: 2rem; }
</style>
```

---

## Resources

- See [content-collections.md](content-collections.md) for schema details
