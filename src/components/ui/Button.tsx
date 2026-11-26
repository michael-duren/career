import type { ButtonHTMLAttributes } from 'react';
import { buttonClassName, type ButtonSize, type ButtonVariant } from '../../lib/ui';
import { cn } from '../../lib/utils';

type Props = ButtonHTMLAttributes<HTMLButtonElement> & { variant?: ButtonVariant; size?: ButtonSize };

// Button: 44px mono uppercase button. secondary (default) is outlined, primary is solid ink,
// danger is the accent (delete). For a link that looks like a button use buttonClassName().
export function Button({ variant = 'secondary', size = 'md', type = 'button', className, ...props }: Props) {
  return <button type={type} className={cn(buttonClassName({ variant, size }), className)} {...props} />;
}

// IconButton: a square Button (44px, or 34px with size="sm" on desktop; 44px below 1024px) holding a line icon.
// label is required: it is the button's accessible name and tooltip.
export function IconButton({ label, variant = 'secondary', size = 'md', type = 'button', className, ...props }: Omit<Props, 'aria-label'> & { label: string }) {
  return <button type={type} aria-label={label} title={label} className={cn(buttonClassName({ variant, size, icon: true }), className)} {...props} />;
}
