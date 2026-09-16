// Synthetic fixtures only. Capture the current TypeScript Markdown behavior for Go parity tests.
import { readFile, writeFile } from 'node:fs/promises';
import { createServer } from 'vite';
const server = await createServer({ configFile: false, server: { middlewareMode: true, hmr: false, watch: null, ws: false }, optimizeDeps: { noDiscovery: true } });
try {
  const data = JSON.parse(await readFile('tests/fixtures/migration/workspace-v2.json', 'utf8'));
  const { buildShelf } = await server.ssrLoadModule('/src/lib/books.ts');
  const { buildBoard } = await server.ssrLoadModule('/src/lib/companies.ts');
  const { checklist } = await server.ssrLoadModule('/src/lib/checklist.ts');
  const expected = {
    books: buildShelf(data.books).all.map(b => ({ slug: b.slug, chapters: b.chapters, log: b.log, completed: b.completed, total: b.total, percent: b.percent })),
    companies: buildBoard(data.companies).categories.flatMap(c => c.companies).map(c => ({ slug: c.slug, why: c.why, steps: c.steps, logEntries: c.logEntries })),
    checklists: Object.fromEntries([...data.books, ...data.companies].map(e => [e.slug, checklist(e.body)])),
    source: 'src/lib/books.ts, companies.ts, checklist.ts; synthetic workspace-v2.json',
  };
  await writeFile('tests/fixtures/migration/expected-projections.json', JSON.stringify(expected, null, 2) + '\n');
} finally { await server.close(); }
