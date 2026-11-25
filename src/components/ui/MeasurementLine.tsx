import type { HTMLAttributes } from 'react';
import { cn } from '../../lib/utils';

export type MeasurementState = 'default' | 'plan' | 'late';

// MeasurementLine: a 1px line with slashed end ticks, for spans, durations and progress
// baselines. Decorative: put the measured value in nearby text (and "LATE" for late spans).
export function MeasurementLine({ state = 'default', className, ...props }: HTMLAttributes<HTMLSpanElement> & { state?: MeasurementState }) {
  return <span aria-hidden="true" className={cn('ui-dim block', state !== 'default' && `ui-dim--${state}`, className)} {...props} />;
}
