import { useState, type CSSProperties } from 'react';
import { initials } from '../../lib/connections';
import { cn } from '../../lib/utils';

// Avatar: a 28-46px circle with a 1px border and the person's initials, or their photo (src)
// once it loads; a photo that fails falls back to the initials. Decorative by default (the
// name is usually beside it); pass labelled to expose the name instead.
export function Avatar({ name, src, size = 34, muted = false, labelled = false, className }: { name: string; src?: string; size?: number; muted?: boolean; labelled?: boolean; className?: string }) {
  const [failed, setFailed] = useState<string>();
  const clamped = Number.isFinite(size) ? Math.min(46, Math.max(28, size)) : 34;
  const style = { '--ui-avatar-size': `${clamped}px` } as CSSProperties;
  const classes = cn('ui-avatar', muted && 'ui-avatar--muted', className);
  if (src && failed !== src) {
    return <img src={src} alt={labelled ? name : ''} width={clamped} height={clamped} loading="lazy" decoding="async" style={style} className={cn(classes, 'ui-avatar--photo')} onError={() => setFailed(src)} />;
  }
  const a11y = labelled ? { role: 'img', 'aria-label': name } : { 'aria-hidden': true as const };
  return (
    <span className={classes} style={style} {...a11y}>
      {initials(name)}
    </span>
  );
}
