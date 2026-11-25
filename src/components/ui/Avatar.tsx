import type { CSSProperties } from 'react';
import { initials } from '../../lib/connections';
import { cn } from '../../lib/utils';

// Avatar: a 28-46px circle with a 1px border and the person's initials. Decorative by
// default (the name is usually beside it); pass labelled to expose the name instead.
export function Avatar({ name, size = 34, muted = false, labelled = false, className }: { name: string; size?: number; muted?: boolean; labelled?: boolean; className?: string }) {
  const clamped = Math.min(46, Math.max(28, size));
  const a11y = labelled ? { role: 'img', 'aria-label': name } : { 'aria-hidden': true as const };
  return (
    <span className={cn('ui-avatar', muted && 'ui-avatar--muted', className)} style={{ '--ui-avatar-size': `${clamped}px` } as CSSProperties} {...a11y}>
      {initials(name)}
    </span>
  );
}
