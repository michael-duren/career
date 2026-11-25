import type { HTMLAttributes } from 'react';
import { cn } from '../../lib/utils';

// Label: mono 10px uppercase metadata text at the per-theme --ink-60 (4.5:1 on every theme).
// Not a form <label>; wrap or reference inputs with a real <label> for that.
export function Label({ as: Tag = 'span', className, ...props }: HTMLAttributes<HTMLElement> & { as?: 'span' | 'div' | 'p' | 'dt' | 'th' }) {
  return <Tag className={cn('ui-label', className)} {...props} />;
}
