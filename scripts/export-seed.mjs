// Uses the installed Astro content loader/schema, never Netlify storage.
import { sync } from 'astro';
import { getViteConfig } from 'astro/config';
import { createServer } from 'vite';
import { mkdir, writeFile, chmod, readFile } from 'node:fs/promises';
import { parse } from 'devalue';
await sync({ logLevel: 'error' });
// Astro 6's sync command writes its schema-validated loader output here.
const store = parse(await readFile('node_modules/.astro/data-store.json', 'utf8'));
if (!(store instanceof Map)) throw new Error('Unsupported Astro content cache format');
const config = await getViteConfig({ server: { middlewareMode: true, hmr: false, watch: null, ws: false }, optimizeDeps: { noDiscovery: true } })({ command: 'build', mode: 'production' });
const server = await createServer(config);
try {
  const { seedWorkspace } = await server.ssrLoadModule('/src/lib/workspace-seed.ts');
  const data = await seedWorkspace(async name => [...(store.get(name)?.values() ?? [])]);
  for (const collection of ['books', 'companies', 'documents']) { if (!data[collection]?.length) throw new Error(`Seed collection ${collection} is empty; refusing incomplete export`); }
  await mkdir('.migration-private', { recursive: true, mode: 0o700 });
  await chmod('.migration-private', 0o700);
  const path = '.migration-private/seed-v2.json';
  await writeFile(path, JSON.stringify(data, null, 2) + '\n', { mode: 0o600 });
  await chmod(path, 0o600);
  console.log(`Seed written to ${path}`);
} finally { await server.close(); }
