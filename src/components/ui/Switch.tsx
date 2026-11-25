import type { ButtonHTMLAttributes } from 'react';
import { cn } from '../../lib/utils';

type Props = Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'onChange' | 'role' | 'aria-checked' | 'disabled'> & {
  checked: boolean;
  onCheckedChange?: (checked: boolean) => void;
  // locked: the switch cannot change (a dependency is off); drawn with a dashed track.
  locked?: boolean;
};

// Switch: a real <button role="switch"> (Space and Enter toggle it). Give it an accessible
// name with aria-label or aria-labelledby. The 44px button is the hit area around the 42x22 track.
export function Switch({ checked, onCheckedChange, locked = false, className, onClick, ...props }: Props) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={locked}
      className={cn('ui-switch', className)}
      onClick={event => {
        onClick?.(event);
        if (!event.defaultPrevented) onCheckedChange?.(!checked);
      }}
      {...props}
    >
      <span className="ui-switch__track" aria-hidden="true"><span className="ui-switch__knob" /></span>
    </button>
  );
}
