import { tickMarks } from '../../lib/ui';
import { cn } from '../../lib/utils';

// TickScale: a row of `count` ticks (every 5th taller) with the first `filled` drawn solid.
// Used for streaks, checklist steps and chapters. With a label it is an image with that
// accessible name ("2 of 29 days"); without one it is decorative and the value must be in text.
export function TickScale({ count, filled, label, className }: { count: number; filled: number; label?: string; className?: string }) {
  const ticks = tickMarks(count, filled);
  return (
    <span className={cn('ui-ticks', className)} {...(label ? { role: 'img', 'aria-label': label } : { 'aria-hidden': true })}>
      {ticks.map((tick, i) => (
        <span key={i} className={cn('ui-ticks__tick', tick.major && 'ui-ticks__tick--major', tick.on && 'ui-ticks__tick--on')} />
      ))}
    </span>
  );
}
