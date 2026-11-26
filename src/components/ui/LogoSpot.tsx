import { useEffect, useMemo, useState, type CSSProperties } from 'react';
import { cssUrl, monogram, resolveUrl } from '../../lib/ui';
import { cn } from '../../lib/utils';

type LogoState = 'monogram' | 'loading' | 'logo' | 'failed';

// LogoSpot: a 36-40px square with the company logo painted in single-color ink (a CSS mask
// over --ink), or a two-letter monogram while it loads, without a usable src, or when it
// fails. The mask needs a same-origin or CORS-enabled src. data-logo-state tells which one
// is shown. Decorative by default (the name is usually next to it); pass labelled to expose
// "<name> logo" to screen readers.
export function LogoSpot({ name, src, size = 36, labelled = false, className }: { name: string; src?: string; size?: 36 | 38 | 40; labelled?: boolean; className?: string }) {
  // Absolute, so the probe and the CSS mask (which resolves against the stylesheet) agree.
  // A malformed src resolves to undefined and shows the monogram.
  const url = useMemo(() => (src && typeof document !== 'undefined' ? resolveUrl(src, document.baseURI) : undefined), [src]);
  const [result, setResult] = useState<{ url: string; ok: boolean }>();

  useEffect(() => {
    if (!url) return;
    let live = true;
    // Probe in CORS mode like the CSS mask does, so a logo the mask cannot use (cross-origin
    // without CORS headers) fails here too and the monogram stays.
    const image = new Image();
    image.crossOrigin = 'anonymous';
    image.onload = () => { if (live) setResult({ url, ok: true }); };
    image.onerror = () => { if (live) setResult({ url, ok: false }); };
    image.src = url;
    return () => { live = false; };
  }, [url]);

  const state: LogoState = !url ? 'monogram' : result?.url !== url ? 'loading' : result.ok ? 'logo' : 'failed';
  const a11y = labelled ? { role: 'img', 'aria-label': `${name} logo` } : { 'aria-hidden': true as const };
  return (
    <span className={cn('ui-logo', className)} data-logo-state={state} style={{ '--ui-logo-size': `${size}px` } as CSSProperties} {...a11y}>
      {state === 'logo' && url
        ? <span className="ui-logo__mark" style={{ '--ui-logo-src': cssUrl(url) } as CSSProperties} />
        : monogram(name)}
    </span>
  );
}
