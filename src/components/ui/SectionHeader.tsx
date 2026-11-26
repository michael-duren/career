import type { ReactNode } from 'react';
import { cn } from '../../lib/utils';
import { Label } from './Label';

// SectionHeader: an uppercase heading on the left and an optional mono label (meta) on the
// right. rule draws the 1px ink line used under table headers.
export function SectionHeader({ title, meta, level = 2, rule = false, id, className, children }: {
  title: ReactNode; meta?: ReactNode; level?: 2 | 3 | 4; rule?: boolean; id?: string; className?: string; children?: ReactNode;
}) {
  const Heading = `h${level}` as const;
  return (
    <div className={cn('ui-section-header', rule && 'ui-section-header--rule', className)}>
      <Heading id={id} className="ui-section-header__title">{title}</Heading>
      {children}
      {meta != null && <Label className="ui-section-header__meta">{meta}</Label>}
    </div>
  );
}
