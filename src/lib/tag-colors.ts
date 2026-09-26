/**
 * Deterministic tag colors. A tag hashes into a fixed palette so it gets the
 * same color on every card, chip, and page. Class strings are written out in
 * full so Tailwind can find them.
 */
export interface TagTone {
  /** Palette name, useful for tests and data attributes. */
  name: string;
  /** Left accent border and a faint tint for a card. */
  card: string;
  /** Pill for the tag itself. */
  chip: string;
  /** Small solid swatch. */
  dot: string;
}

export const TAG_PALETTE: readonly TagTone[] = [
  { name: 'sky', card: 'border-l-sky-500 bg-sky-500/[0.06] hover:bg-sky-500/10', chip: 'bg-sky-500/15 text-sky-300 ring-sky-500/30', dot: 'bg-sky-400' },
  { name: 'emerald', card: 'border-l-emerald-500 bg-emerald-500/[0.06] hover:bg-emerald-500/10', chip: 'bg-emerald-500/15 text-emerald-300 ring-emerald-500/30', dot: 'bg-emerald-400' },
  { name: 'amber', card: 'border-l-amber-500 bg-amber-500/[0.06] hover:bg-amber-500/10', chip: 'bg-amber-500/15 text-amber-300 ring-amber-500/30', dot: 'bg-amber-400' },
  { name: 'rose', card: 'border-l-rose-500 bg-rose-500/[0.06] hover:bg-rose-500/10', chip: 'bg-rose-500/15 text-rose-300 ring-rose-500/30', dot: 'bg-rose-400' },
  { name: 'violet', card: 'border-l-violet-500 bg-violet-500/[0.06] hover:bg-violet-500/10', chip: 'bg-violet-500/15 text-violet-300 ring-violet-500/30', dot: 'bg-violet-400' },
  { name: 'teal', card: 'border-l-teal-500 bg-teal-500/[0.06] hover:bg-teal-500/10', chip: 'bg-teal-500/15 text-teal-300 ring-teal-500/30', dot: 'bg-teal-400' },
  { name: 'orange', card: 'border-l-orange-500 bg-orange-500/[0.06] hover:bg-orange-500/10', chip: 'bg-orange-500/15 text-orange-300 ring-orange-500/30', dot: 'bg-orange-400' },
  { name: 'fuchsia', card: 'border-l-fuchsia-500 bg-fuchsia-500/[0.06] hover:bg-fuchsia-500/10', chip: 'bg-fuchsia-500/15 text-fuchsia-300 ring-fuchsia-500/30', dot: 'bg-fuchsia-400' },
  { name: 'lime', card: 'border-l-lime-500 bg-lime-500/[0.06] hover:bg-lime-500/10', chip: 'bg-lime-500/15 text-lime-300 ring-lime-500/30', dot: 'bg-lime-400' },
  { name: 'indigo', card: 'border-l-indigo-400 bg-indigo-500/[0.06] hover:bg-indigo-500/10', chip: 'bg-indigo-500/15 text-indigo-300 ring-indigo-500/30', dot: 'bg-indigo-400' },
];

export const NEUTRAL_TONE: TagTone = { name: 'neutral', card: 'border-l-zinc-700 bg-zinc-900/40 hover:bg-zinc-900', chip: 'bg-zinc-800 text-zinc-300 ring-zinc-700', dot: 'bg-zinc-500' };

/** Case and surrounding space do not change a tag's color. */
export function normalizeTag(tag: string): string {
  return tag.trim().toLowerCase();
}

/** 32-bit FNV-1a over UTF-16 code units: small, fast, and stable across runs. */
export function tagHash(tag: string): number {
  let hash = 0x811c9dc5;
  const text = normalizeTag(tag);
  for (let i = 0; i < text.length; i++) {
    hash ^= text.charCodeAt(i);
    hash = Math.imul(hash, 0x01000193) >>> 0;
  }
  return hash;
}

export function tagTone(tag: string | null | undefined): TagTone {
  if (!tag || !normalizeTag(tag)) return NEUTRAL_TONE;
  return TAG_PALETTE[tagHash(tag) % TAG_PALETTE.length];
}

/** An entry takes the color of its first tag, so related entries group visually. */
export function entryTone(tags: readonly string[] | undefined): TagTone {
  return tagTone(tags?.[0]);
}

/** Classes for a tag pill; callers add size and spacing. */
export function tagChipClass(tag: string): string {
  return `rounded-full ring-1 ring-inset ${tagTone(tag).chip}`;
}
