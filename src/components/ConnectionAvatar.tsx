import { useState } from 'react';
import { initials, photoURL, type Connection } from '../lib/connections';

const COLORS = ['bg-blue-700', 'bg-emerald-700', 'bg-amber-700', 'bg-rose-700', 'bg-violet-700', 'bg-cyan-700', 'bg-fuchsia-700', 'bg-lime-700'];

function color(name: string): string {
  let hash = 0;
  for (const char of name) hash = (hash * 31 + char.charCodeAt(0)) | 0;
  return COLORS[Math.abs(hash) % COLORS.length];
}

export function ConnectionAvatar({ connection, size = 40 }: { connection: Connection; size?: number }) {
  const source = photoURL(connection);
  const [failed, setFailed] = useState<string>();
  const style = { width: size, height: size, fontSize: size * 0.38 };
  if (source && failed !== source) {
    return <img src={source} alt="" width={size} height={size} loading="lazy" decoding="async" style={style} className="shrink-0 rounded-full object-cover ring-1 ring-white/10" onError={() => setFailed(source)} />;
  }
  return <span aria-hidden="true" style={style} className={`flex shrink-0 items-center justify-center rounded-full font-semibold text-white ring-1 ring-white/10 ${color(connection.name)}`}>{initials(connection.name)}</span>;
}
