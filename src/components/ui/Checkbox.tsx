import type { InputHTMLAttributes } from 'react';
import { cn } from '../../lib/utils';

type Props = Omit<InputHTMLAttributes<HTMLInputElement>, 'type' | 'size'> & {
  // next circles the box with the accent ring (the next item to do).
  next?: boolean;
  size?: 16 | 18;
};

// Checkbox: a native checkbox drawn as a small Box with an X when checked. The input is
// invisible but covers a 44px hit area, so keyboard, label and screen-reader behavior are
// native. Name it with aria-label or wrap it in a <label> with visible text.
export function Checkbox({ next = false, size = 16, className, ...props }: Props) {
  return (
    <span className={cn('ui-check ui-box ui-box--sm', size === 18 && 'ui-check--lg', className)}>
      <input type="checkbox" className="ui-check__input" {...props} />
      <svg className="ui-check__x" viewBox="0 0 16 16" aria-hidden="true" preserveAspectRatio="none">
        <path d="M0 0 L16 16 M16 0 L0 16" stroke="currentColor" strokeWidth="1" vectorEffect="non-scaling-stroke" />
      </svg>
      {next && <span className="ui-check__ring" aria-hidden="true" />}
    </span>
  );
}
