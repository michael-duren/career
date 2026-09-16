import Fuse from 'fuse.js';
import type { Workspace } from './workspace';

export interface SearchEntry { title: string; href: string; kind: string; body: string }
const pages = [
  ['Bookshelf', '/books'], ['Companies', '/companies'], ['Home', '/'],
  ['Agent context', '/agents'], ['Personal journal', '/personal-journal'], ['Work journal', '/journal'], ['Logs', '/logs'], ['Notes', '/notes'],
  ['Pages', '/documents'], ['Progress', '/progress'], ['Timeline', '/timeline'],
];
const path = (id: string) => id.split('/').map(encodeURIComponent).join('/');
export function searchEntries(data: Workspace): SearchEntry[] {
  return [
    ...pages.map(([title, href]) => ({ title, href, kind: 'Navigation', body: '' })),
    ...data.documents.map(d => ({ title: d.title, href: d.id === 'index' ? '/' : `/documents/${path(d.id)}`, kind: 'Page', body: `${d.description} ${d.tags.join(' ')} ${d.body}` })),
    ...data.notes.map(n => ({ title: n.title, href: `/notes/${path(n.id)}`, kind: 'Note', body: `${n.topic} ${n.description} ${n.tags.join(' ')} ${n.body}` })),
    ...data.weeks.map(w => ({ title: `Week ${w.week} · ${w.dates}`, href: `/journal?id=${encodeURIComponent(w.slug)}`, kind: 'Work journal', body: `${w.tags.join(' ')} ${w.body}` })),
    ...(data.personalJournal ?? []).map(e => ({ title: e.title, href: `/personal-journal?id=${encodeURIComponent(e.id)}`, kind: 'Personal journal', body: `${e.date} ${e.description} ${e.tags.join(' ')} ${e.body}` })),
    ...data.books.map(b => ({ title: b.title, href: `/manage/books?id=${encodeURIComponent(b.slug)}`, kind: 'Book / course', body: `${b.authors.join(' ')} ${b.category} ${b.tags.join(' ')} ${b.body}` })),
    ...data.companies.map(c => ({ title: c.title, href: `/companies/${c.slug.split('/').map(encodeURIComponent).join('/')}`, kind: 'Company', body: `${c.category} ${c.tags.join(' ')} ${c.body}` })),
    ...(data.goals ?? []).map(g => ({ title: g.title, href: '/timeline', kind: 'Goal', body: `${g.notes.map(n => n.body).join(' ')} ${g.steps.map(s => s.title).join(' ')} ${Object.entries(g.metadata).flat().join(' ')}` })),
  ];
}
export function createSearch(entries: SearchEntry[]) {
  return new Fuse(entries, {
    keys: [{ name: 'title', weight: 3 }, { name: 'body', weight: 1 }],
    threshold: 0.35, ignoreLocation: true, ignoreFieldNorm: true,
    includeMatches: true,
  });
}
