---
title: Pagination
priority: medium
category: patterns
---

# Pagination

Paginate large collections of content for better performance and UX.

## Basic Pagination

```astro
---
// src/pages/blog/[page].astro
import { getCollection } from 'astro:content';

export async function getStaticPaths({ paginate }) {
  const posts = await getCollection('blog');
  const sortedPosts = posts.sort((a, b) => 
    b.data.publishDate.getTime() - a.data.publishDate.getTime()
  );
  
  return paginate(sortedPosts, {
    pageSize: 10, // Posts per page
  });
}

const { page } = Astro.props;
---

<h1>Blog - Page {page.currentPage}</h1>

<!-- Posts for current page -->
{page.data.map((post) => (
  <article>
    <a href={`/blog/${post.slug}`}>
      <h2>{post.data.title}</h2>
    </a>
  </article>
))}

<!-- Pagination controls -->
<nav>
  {page.url.prev && <a href={page.url.prev}>Previous</a>}
  <span>Page {page.currentPage} of {page.lastPage}</span>
  {page.url.next && <a href={page.url.next}>Next</a>}
</nav>
```

---

## Page Object Properties

| Property | Type | Description |
|----------|------|-------------|
| `page.data` | Array | Items for current page |
| `page.currentPage` | number | Current page number (1-based) |
| `page.lastPage` | number | Total number of pages |
| `page.url.current` | string | Current page URL |
| `page.url.prev` | string \| undefined | Previous page URL |
| `page.url.next` | string \| undefined | Next page URL |

---

## Resources

- **Pagination Guide**: https://docs.astro.build/en/guides/pagination/
