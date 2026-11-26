import { useRef, type KeyboardEvent } from 'react';
import { nextTabIndex } from '../../lib/ui';
import { cn } from '../../lib/utils';

export type SegmentedTab<T extends string> = { value: T; label: string };

// SegmentedTabs: a tablist of bordered segments; the selected one is solid ink. Keyboard per
// the WAI-ARIA tabs pattern: Tab enters at the selected tab, arrows move and select, Home/End
// jump. controls is the id of the panel the tabs switch (aria-controls on every tab).
export function SegmentedTabs<T extends string>({ tabs, value, onChange, label, controls, className }: {
  tabs: readonly SegmentedTab<T>[]; value: T; onChange: (value: T) => void; label: string; controls?: string; className?: string;
}) {
  const refs = useRef<(HTMLButtonElement | null)[]>([]);
  const selected = tabs.findIndex(tab => tab.value === value);
  // With no matching value nothing is selected, but the first tab stays reachable by Tab.
  const tabbable = Math.max(0, selected);

  function onKeyDown(event: KeyboardEvent<HTMLButtonElement>, index: number) {
    const next = nextTabIndex(event.key, index, tabs.length);
    if (next === null) return;
    event.preventDefault();
    refs.current[next]?.focus();
    onChange(tabs[next].value);
  }

  return (
    <div role="tablist" aria-label={label} className={cn('ui-tabs', className)}>
      {tabs.map((tab, index) => (
        <button
          key={tab.value}
          ref={element => { refs.current[index] = element; }}
          type="button"
          role="tab"
          aria-selected={index === selected}
          aria-controls={controls}
          tabIndex={index === tabbable ? 0 : -1}
          className="ui-tabs__tab"
          onClick={() => onChange(tab.value)}
          onKeyDown={event => onKeyDown(event, index)}
        >
          {tab.label}
        </button>
      ))}
    </div>
  );
}
