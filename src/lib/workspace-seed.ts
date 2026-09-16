import { getCollection } from 'astro:content';
import bio from '../content/notes/bio.md?raw';
import { isEmptyStarterWeek, migrateJournals, type Workspace } from './workspace';

export async function seed(collections: typeof getCollection = getCollection): Promise<Workspace> {
  const [notes, weeks, books, companies, documents] = await Promise.all([
    collections('notes'), collections('progress'), collections('books'), collections('companies'), collections('docs'),
  ]);
  return {
    version: 2,
    catalogVersion: 1,
    notes: notes.filter(n => n.data.title || n.body?.trim()).map(n => ({ id: n.id, ...n.data, title: n.data.title || n.id.split('/').at(-1)!.replaceAll('-', ' '), description: n.data.description ?? '', body: n.body ?? '' })),
    weeks: weeks.map(w => ({ slug: w.id, ...w.data, body: w.body ?? '' })).filter(w => !isEmptyStarterWeek(w)),
    books: books.map(b => ({ slug: b.id, ...b.data, body: b.body ?? '' })),
    companies: companies.map(c => ({ slug: c.id, ...c.data, body: c.body ?? '' })),
    documents: documents.map(d => ({ id: d.id, ...d.data, description: d.data.description ?? '', tags: [], body: d.body ?? '' })),
  };
}

export async function seedWorkspace(collections: typeof getCollection = getCollection): Promise<Workspace> { return migrateJournals(await seed(collections), bio); }
