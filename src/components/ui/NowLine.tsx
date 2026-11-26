import type { CSSProperties } from 'react';
import { formatTime, type TimeValue } from '../../lib/format';
import { cn } from '../../lib/utils';

// NowLine: a 1px accent line with a handwritten "now 5:20 PM". The parent positions it
// (usually absolutely at the current time); the label is real text for screen readers.
export function NowLine({ time, timeZone, className, style }: { time: TimeValue; timeZone?: string; className?: string; style?: CSSProperties }) {
  let label = '';
  try {
    label = formatTime(time, { timeZone });
  } catch {
    // A bad time or zone must not take the page down: draw the line without the label.
  }
  return (
    <div className={cn('ui-now', className)} style={style}>
      {label && <span className="ui-now__label">now {label}</span>}
    </div>
  );
}

// TodayMarker: the dash-dot accent vertical line for today on a date axis. Give it a height
// (or top/bottom) from the parent.
export function TodayMarker({ label = 'today', className, style }: { label?: string; className?: string; style?: CSSProperties }) {
  return (
    <div className={cn('ui-today', className)} style={style}>
      <span className="ui-today__label">{label}</span>
    </div>
  );
}
