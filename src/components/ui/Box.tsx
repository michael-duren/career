import type { HTMLAttributes } from 'react';
import { cn } from '../../lib/utils';

// Box: a 1px ink frame whose lines run 3px past each corner (2px when small).
// It draws with ::before/::after, so content inside must not rely on the box's own pseudo-elements.
export function Box({ small = false, as: Tag = 'div', className, ...props }: HTMLAttributes<HTMLElement> & { small?: boolean; as?: 'div' | 'section' | 'article' | 'span' | 'li' }) {
  return <Tag className={cn('ui-box', small && 'ui-box--sm', className)} {...props} />;
}
