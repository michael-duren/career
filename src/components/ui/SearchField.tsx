import type { InputHTMLAttributes } from 'react';
import { cn } from '../../lib/utils';
import { SearchIcon } from './icons';

type Props = Omit<InputHTMLAttributes<HTMLInputElement>, 'type' | 'size'> & {
  // label is the accessible name (visually hidden); placeholder defaults to it.
  label: string;
  // size "sm" is the 34px header field; default is 44px.
  size?: 'md' | 'sm';
  // hint shows a keyboard shortcut at the right, e.g. "/".
  hint?: string;
};

// SearchField: a bordered search input with a thin magnifier. The whole field is a <label>,
// so clicking anywhere focuses the input.
export function SearchField({ label, size = 'md', hint, placeholder, className, ...props }: Props) {
  return (
    <label className={cn('ui-search', size === 'sm' && 'ui-search--sm', className)}>
      <SearchIcon size={size === 'sm' ? 13 : 14} />
      <span className="ui-sr-only">{label}</span>
      <input type="search" className="ui-search__input" placeholder={placeholder ?? label} {...props} />
      {hint && <kbd className="ui-search__hint" aria-hidden="true">{hint}</kbd>}
    </label>
  );
}
