// Pure helpers behind the Architect drawing primitives (src/components/ui/,
// public/ui/primitives.css). Kept free of React so node --test can cover them.

// Goals have no color in the Architect UI; they are told apart by pattern, by goal order.
export const GOAL_PATTERNS = ['diagonal', 'crosshatch', 'dots'] as const;
export type HatchPattern = (typeof GOAL_PATTERNS)[number];
export type GoalPatternName = HatchPattern | 'outline';

// goalPattern maps a goal's position (0-based) to its pattern; goals past the third,
// and invalid positions, fall back to an outline.
export function goalPattern(index: number): GoalPatternName {
  return Number.isInteger(index) && index >= 0 && index < GOAL_PATTERNS.length ? GOAL_PATTERNS[index] : 'outline';
}

export type Tick = { on: boolean; major: boolean };

// tickMarks lays out a tick scale of `count` ticks with the first `filled` filled.
// Every 5th tick (0, 5, 10, ...) is a taller major tick. filled is clamped to 0..count.
export function tickMarks(count: number, filled: number): Tick[] {
  const total = Number.isFinite(count) ? Math.max(0, Math.floor(count)) : 0;
  const on = Number.isFinite(filled) ? Math.min(total, Math.max(0, Math.floor(filled))) : 0;
  return Array.from({ length: total }, (_, i) => ({ on: i < on, major: i % 5 === 0 }));
}

// monogram is the two-letter logo fallback: the first letters of the first two words
// ("Jane Street" -> "JS"), else the first two letters of a single word ("Ramp" -> "RA").
export function monogram(name: string): string {
  const words = name.replace(/[^\p{L}\p{N} ]/gu, ' ').split(/\s+/).filter(Boolean);
  if (words.length === 0) return '?';
  const letters = words.length > 1 ? [...words[0]][0] + [...words[1]][0] : [...words[0]].slice(0, 2).join('');
  return letters.toUpperCase();
}

// nextTabIndex implements the tablist keyboard pattern (WAI-ARIA APG, automatic
// activation): ArrowLeft/ArrowRight wrap around, Home/End jump to the ends.
// Returns null for keys the tablist does not handle.
export function nextTabIndex(key: string, current: number, count: number): number | null {
  if (count <= 0) return null;
  switch (key) {
    case 'ArrowRight': return (current + 1) % count;
    case 'ArrowLeft': return (current - 1 + count) % count;
    case 'Home': return 0;
    case 'End': return count - 1;
    default: return null;
  }
}

// cssUrl quotes a URL for CSS url(): backslashes, quotes and line breaks are escaped
// (CSS string escapes, which JSON escaping is not).
export function cssUrl(url: string): string {
  return `url("${url.replace(/[\\"\n\r\f]/g, ch => `\\${ch.charCodeAt(0).toString(16)} `)}")`;
}

export type ButtonVariant = 'primary' | 'secondary' | 'danger';
export type ButtonSize = 'md' | 'sm';

// buttonClassName returns the primitive classes for a button-looking element, so links
// (<a>) can share the Button look: buttonClassName({ variant: 'primary' }).
export function buttonClassName({ variant = 'secondary', size = 'md', icon = false }: { variant?: ButtonVariant; size?: ButtonSize; icon?: boolean } = {}): string {
  return ['ui-btn', variant !== 'secondary' && `ui-btn--${variant}`, size === 'sm' && 'ui-btn--sm', icon && 'ui-btn--icon'].filter(Boolean).join(' ');
}
