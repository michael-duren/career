import type { CSSProperties, HTMLAttributes } from 'react';
import { goalPattern, type GoalPatternName, type HatchPattern } from '../../lib/ui';
import { cn } from '../../lib/utils';
import { MeasurementLine } from './MeasurementLine';

const hatchClass = (pattern: GoalPatternName) => (pattern === 'outline' ? '' : `ui-hatch--${pattern}`);

// HatchFill: a decorative area filled with the diagonal, crosshatch or dot pattern in ink.
export function HatchFill({ pattern = 'diagonal', className, ...props }: HTMLAttributes<HTMLSpanElement> & { pattern?: HatchPattern }) {
  return <span {...props} aria-hidden="true" className={cn(hatchClass(pattern), className)} />;
}

// GoalPattern: the strip (6-8px, left edge of a schedule block) or legend swatch that tells
// goals apart without color. Pass the goal's position (index) or an explicit pattern.
export function GoalPattern({ index, pattern, variant = 'strip', width, className, style }: {
  index?: number; pattern?: GoalPatternName; variant?: 'strip' | 'swatch'; width?: number; className?: string; style?: CSSProperties;
}) {
  const name = pattern ?? goalPattern(index ?? -1);
  const base = variant === 'strip' ? 'ui-goal-strip' : 'ui-goal-swatch';
  return (
    <span
      aria-hidden="true"
      data-pattern={name}
      className={cn(base, name === 'outline' ? `${base}--outline` : hatchClass(name), className)}
      style={variant === 'strip' && width ? { width, ...style } : style}
    />
  );
}

// Progress: a hatch fill sitting on a measurement line. value/max are clamped to 0..1 and
// exposed as a progressbar (0-100) named by label; the visible value belongs in nearby text.
export function Progress({ value, max = 1, pattern = 'diagonal', label, late = false, className }: {
  value: number; max?: number; pattern?: HatchPattern; label: string; late?: boolean; className?: string;
}) {
  const ratio = max > 0 && Number.isFinite(value) ? Math.min(1, Math.max(0, value / max)) : 0;
  return (
    <span role="progressbar" aria-label={label} aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.round(ratio * 100)} className={cn('ui-progress', className)}>
      <span className={cn('ui-progress__fill', hatchClass(pattern))} style={{ width: `${ratio * 100}%` }} />
      <MeasurementLine state={late ? 'late' : 'default'} />
    </span>
  );
}
