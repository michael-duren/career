import { checklist } from './checklist.ts';
import type { CollectionEntry } from 'astro:content';

export type CompanyStatus = CollectionEntry<'companies'>['data']['status'];
export type RawCompany = CollectionEntry<'companies'>['data'] & { slug: string; body: string };
export type Company = Omit<RawCompany, 'body'> & {
  why: string;
  steps: { index: number; label: string; completed: boolean }[];
  completed: number;
  logEntries: string[];
};
export interface CompanyBoardData {
  categories: { label: string; companies: Company[] }[];
  totals: Record<CompanyStatus, number>;
}
const categories = ['Observability / Infra', 'Cloud-Native Infra', 'Distributed Systems / Databases', 'Big Tech Infra / SRE', 'AI Infrastructure'];
const priority = { high: 0, medium: 1, low: 2 };

function section(body: string, heading: string): string {
  return body.match(new RegExp(`^## ${heading}\\s*\\r?\\n([\\s\\S]*?)(?=^## |$(?![\\s\\S]))`, 'm'))?.[1].trim() ?? '';
}

export function buildBoard(raws: RawCompany[]): CompanyBoardData {
  const groups = new Map<string, Company[]>();
  const totals: Record<CompanyStatus, number> = { not_started: 0, applied: 0, interviewing: 0, offer: 0, rejected: 0, passed: 0 };
  for (const { body, ...raw } of raws) {
    const steps = checklist(body).filter(task => task.section === 'steps').map(task => ({ index: task.index, label: task.label, completed: task.checked }));
    // Keep saved checklists in the same relationship-first order as new companies.
    // Retain source indices so toggles still update the correct Markdown line.
    const research = steps.findIndex(step => /^(Research team (&|and) open roles|Research the role)$/i.test(step.label));
    const relationship = steps.findIndex(step => step.label === 'Reach out and start building a relationship');
    if (research >= 0 && relationship > research) {
      const [step] = steps.splice(research, 1);
      steps.splice(relationship, 0, step);
    }
    const company: Company = { ...raw, why: section(body, 'Why'), steps, completed: steps.filter(step => step.completed).length,
      logEntries: section(body, 'Log').split(/\n(?=-\s+)/).filter(line => /^\s*-\s+/.test(line)).map(line => line.replace(/^\s*-\s+/, '').replace(/\n  /g, '\n').trim()) };
    const group = groups.get(raw.category) ?? [];
    group.push(company);
    groups.set(raw.category, group);
    totals[raw.status]++;
  }
  return { categories: Array.from(groups, ([label, companies]) => ({ label, companies: companies.sort((a, b) => priority[a.priority] - priority[b.priority] || a.title.localeCompare(b.title)) })).sort((a, b) => (categories.includes(a.label) ? categories.indexOf(a.label) : categories.length) - (categories.includes(b.label) ? categories.indexOf(b.label) : categories.length) || a.label.localeCompare(b.label)), totals };
}


/** Add a dated note to the Log section without changing company steps or other sections. */
export function appendCompanyNote(body: string, note: string, date: string): string {
  if (!note.trim()) return body;
  const item = `- ${date}: ${note.trim().replace(/\r?\n/g, '\n  ')}`;
  const heading = /^## Log[^\S\n]*\r?\n/im.exec(body);
  if (!heading) return `${body.trimEnd()}\n\n## Log\n\n${item}\n`;
  const at = heading.index + heading[0].length;
  return `${body.slice(0, at)}\n${item}\n${body.slice(at)}`;
}
