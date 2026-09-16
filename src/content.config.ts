import { defineCollection, z } from 'astro:content';
import { glob } from 'astro/loaders';
import { BOOK_CATEGORIES } from './lib/books';

const docsCollection = defineCollection({
  loader: glob({ pattern: '**/*.{md,mdx}', base: './src/content/docs', generateId: ({ entry }) => entry.replace(/\.mdx?$/, '') }),
  schema: z.object({
    title: z.string(),
    description: z.string().optional(),
  }),
});

const progressCollection = defineCollection({
  loader: glob({ pattern: '**/week-*.md', base: './src/content/progress' }),
  schema: z.object({
    week: z.number().int().positive(),
    year: z.number().int().default(2026),
    dates: z.string(),
    hours: z.record(z.string(), z.number().min(0)).default({}),
    tags: z.array(z.string()).default([]),
    targets: z.record(z.string(), z.number().min(0)).optional(),
  }),
});

// ---------------------------------------------------------------------------
// Books & courses reading tracker
//
// One markdown file per book/course under src/content/books/. Frontmatter holds
// the structured, metric-bearing fields; the body holds a "## Chapters" checklist
// (the schematic progress signal — checked boxes drive the progress bar) and a
// dated "## Log" of reading notes. Anything in this schema flows straight into
// the /books dashboard with no code change.
// ---------------------------------------------------------------------------
const booksCollection = defineCollection({
  loader: glob({ pattern: ['**/*.md', '!**/README.md'], base: './src/content/books' }),
  schema: z.object({
    title: z.string(),
    edition: z.string().optional(),
    authors: z.array(z.string()).default([]),
    category: z.enum(BOOK_CATEGORIES),
    // "book" or "course" — courses are tracked the same way (modules instead of chapters).
    type: z.enum(['book', 'course']).default('book'),
    url: z.string().url().optional(),
    cover: z.string().url().optional(),
    isbn: z.string().optional(),
    // Lifecycle. "reading" surfaces a book in the Currently Reading row.
    status: z
      .enum(['backlog', 'reading', 'paused', 'completed', 'reference'])
      .default('backlog'),
    // Exactly one book should set featured: true — it becomes the highlighted hero.
    featured: z.boolean().default(false),
    // Explicit progress override. If omitted, progress is computed from the
    // "## Chapters" checklist in the body (checked / total boxes).
    progress: z
      .object({
        unit: z.enum(['chapter', 'page', 'module', 'section', 'lecture']).default('chapter'),
        total: z.number().int().min(0),
        completed: z.number().int().min(0).default(0),
      })
      .optional(),
    started: z.string().optional(),
    finished: z.string().optional(),
    rating: z.number().min(0).max(5).optional(),
    priority: z.enum(['high', 'medium', 'low']).default('medium'),
    tags: z.array(z.string()).default([]),
  }),
});

const companiesCollection = defineCollection({
  loader: glob({ pattern: ['**/*.md', '!**/README.md'], base: './src/content/companies' }),
  schema: z.object({
    title: z.string(),
    category: z.string(),
    type: z.literal('company'),
    url: z.string().url(),
    cover: z.string().url().optional(),
    status: z.enum(['not_started', 'applied', 'interviewing', 'offer', 'rejected', 'passed']),
    featured: z.boolean(),
    priority: z.enum(['low', 'medium', 'high']),
    tags: z.array(z.string()),
  }),
});

const notesCollection = defineCollection({
  loader: glob({ pattern: ['**/*.md', '!**/README.md', '!bio.md'], base: './src/content/notes' }),
  schema: z.object({
    title: z.string().optional(),
    topic: z.string().default('General'),
    description: z.string().optional(),
    tags: z.array(z.string()).default([]),
  }),
});

export const collections = {
  notes: notesCollection,
  companies: companiesCollection,
  docs: docsCollection,
  progress: progressCollection,
  books: booksCollection,
};
