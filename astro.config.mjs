// @ts-check
import { defineConfig } from "astro/config";
import tailwindcss from "@tailwindcss/vite";
import mdx from "@astrojs/mdx";
import react from "@astrojs/react";
import remarkGfm from 'remark-gfm';
import rehypeSlug from 'rehype-slug';
import rehypeAutolinkHeadings from 'rehype-autolink-headings';

// https://astro.build/config
export default defineConfig({
  output: "static",
  integrations: [
    mdx(),
    react()
  ],
  vite: {
    // Route OAuth preflight through the application's origin policy, including in development.
    server: { cors: false },
    plugins: [tailwindcss()],
  },
  markdown: {
    remarkPlugins: [remarkGfm],
    rehypePlugins: [
      rehypeSlug,
      [rehypeAutolinkHeadings, {
        behavior: 'wrap',
        properties: {
          className: ['anchor']
        }
      }]
    ],
    shikiConfig: {
      theme: 'github-dark',
      wrap: true
    }
  }
});
