import { useEffect, useState, type CSSProperties } from 'react';
import { monogram } from '../../lib/ui';
import { cn } from '../../lib/utils';

// LogoSpot: a 36-40px square with the company logo painted in single-color ink (a CSS mask
// over --ink), or a two-letter monogram while it loads, without a src, or when it fails.
// The mask needs a same-origin or CORS-enabled src. Decorative by default (the name is
// usually next to it); pass labelled to expose "<name> logo" to screen readers.
export function LogoSpot({ name, src, size = 36, labelled = false, className }: { name: string; src?: string; size?: 36 | 38 | 40; labelled?: boolean; className?: string }) {
  const [loaded, setLoaded] = useState<string>();

  useEffect(() => {
    if (!src) return;
    let live = true;
    const image = new Image();
    image.onload = () => { if (live) setLoaded(src); };
    image.src = src;
    return () => { live = false; };
  }, [src]);

  const a11y = labelled ? { role: 'img', 'aria-label': `${name} logo` } : { 'aria-hidden': true as const };
  return (
    <span className={cn('ui-logo', className)} style={{ '--ui-logo-size': `${size}px` } as CSSProperties} {...a11y}>
      {src && loaded === src
        ? <span className="ui-logo__mark" style={{ '--ui-logo-src': `url(${JSON.stringify(src)})` } as CSSProperties} />
        : monogram(name)}
    </span>
  );
}
