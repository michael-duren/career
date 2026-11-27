import { useCallback, useEffect, useRef, useState } from 'react';
import { Box, Button, Label, SegmentedTabs } from './ui';
import { AGENT_FORMATS, agentEndpoint, agentFileName, agentStats, padCount, previewText, type AgentFormat } from '../lib/agent-view';
import type { Goal } from '../lib/timeline';
import { localDate } from '../lib/workspace';

// The Agent context page (docs/redesign/pages/agent-context.md): the full output of
// GET /api/agent-context in Markdown or JSON, a short preview of it, and a copy button
// that copies all of it.

type Output = { markdown: string; json: string; goals: Goal[] };

const PANEL_ID = 'agent-context-preview';
const FAILED = 'Could not load agent context.';

/** Thrown when the session expired and the page is already on its way to /login. */
class SignedOut extends Error {}

async function fetchText(url: string, signal: AbortSignal): Promise<string> {
  const response = await fetch(url, { cache: 'no-store', signal });
  if (response.status === 401) {
    location.assign(`/login?redirect=${encodeURIComponent(location.pathname + location.search)}`);
    throw new SignedOut();
  }
  const text = await response.text();
  if (!response.ok) {
    let message = FAILED;
    try { const body = JSON.parse(text); if (typeof body?.error === 'string' && body.error) message = body.error; } catch { /* not JSON */ }
    throw new Error(message);
  }
  return text;
}

async function loadOutput(signal: AbortSignal): Promise<Output> {
  const [markdown, raw] = await Promise.all([fetchText(agentEndpoint('markdown'), signal), fetchText(agentEndpoint('json'), signal)]);
  let value: { goals?: unknown };
  try {
    value = JSON.parse(raw);
  } catch (e) {
    console.warn('Agent context JSON did not parse', e);
    throw new Error(FAILED);
  }
  const goals = Array.isArray(value?.goals) ? value.goals as Goal[] : [];
  // Shown and copied pretty-printed; the data is the API's.
  return { markdown, json: JSON.stringify(value, null, 2), goals };
}

/** Clipboard API first; the hidden-textarea fallback covers browsers that refuse it. */
async function copyText(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
    return;
  } catch (e) {
    console.warn('Clipboard write failed; trying the fallback', e);
  }
  const area = document.createElement('textarea');
  area.value = text;
  area.setAttribute('readonly', '');
  area.style.position = 'fixed';
  area.style.opacity = '0';
  const focused = document.activeElement as HTMLElement | null;
  document.body.append(area);
  let ok = false;
  try {
    area.select();
    area.setSelectionRange(0, text.length); // iOS Safari ignores select()
    ok = document.execCommand('copy');
  } finally {
    area.remove();
    focused?.focus();
  }
  if (!ok) throw new Error('Copy failed.');
}

export function AgentContextView() {
  const [format, setFormat] = useState<AgentFormat>('markdown');
  const [output, setOutput] = useState<Output>();
  const [error, setError] = useState('');
  const [copied, setCopied] = useState(false);
  const [copyError, setCopyError] = useState('');
  const latest = useRef(0);
  const controller = useRef<AbortController | undefined>(undefined);
  const resetTimer = useRef(0);

  const load = useCallback(() => {
    const request = ++latest.current;
    controller.current?.abort();
    const current = new AbortController();
    controller.current = current;
    setError('');
    loadOutput(current.signal).then(value => {
      if (request !== latest.current) return;
      setOutput(value);
      setCopied(false);
    }).catch(e => {
      if (request !== latest.current || current.signal.aborted || e instanceof SignedOut) return;
      setError(e instanceof Error && e.message ? e.message : FAILED);
    });
  }, []);

  useEffect(() => {
    load();
    window.addEventListener('workspace-saved', load);
    return () => {
      controller.current?.abort();
      window.removeEventListener('workspace-saved', load);
      window.clearTimeout(resetTimer.current);
    };
  }, [load]);

  const text = output ? output[format] : '';
  const textRef = useRef(text);
  textRef.current = text;
  const preview = previewText(text);
  const stats = output ? agentStats(output.goals, localDate()) : undefined;

  function pick(next: AgentFormat) {
    setFormat(next);
    setCopied(false);
    setCopyError('');
  }

  async function copy() {
    if (!output) return;
    const copying = text;
    setCopyError('');
    try {
      await copyText(copying);
      // The format or the output may have changed while the clipboard prompt was open.
      if (copying !== textRef.current) return;
      setCopied(true);
      window.clearTimeout(resetTimer.current);
      resetTimer.current = window.setTimeout(() => setCopied(false), 2500);
    } catch (e) {
      console.warn('Copy failed', e);
      setCopied(false);
      setCopyError(`Copy failed. Open ${agentEndpoint(format)} and copy it from there.`);
    }
  }

  const tabs = <SegmentedTabs tabs={AGENT_FORMATS} value={format} onChange={pick} label="Format" controls={PANEL_ID}
    className="max-md:!grid max-md:!w-full max-md:grid-cols-2 [&>button]:!justify-center" />;
  const endpoint = agentEndpoint(format);

  return (
    <div>
      <header className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between md:gap-6">
        <div className="flex flex-col gap-1.5 md:gap-2">
          <Label className="md:hidden">{stats ? `${stats.goals} ${stats.goals === 1 ? 'goal' : 'goals'} · ${stats.steps} ${stats.steps === 1 ? 'step' : 'steps'}` : 'Agent context'}</Label>
          <Label className="max-md:hidden">Your goals, ready to hand to an AI assistant</Label>
          <h1 className="m-0 font-extralight !text-[40px] !leading-[0.9] md:!text-[52px]">Agent context</h1>
        </div>
        <div className="grid grid-cols-[minmax(0,1fr)_110px] gap-2.5 md:flex md:items-center">
          {tabs}
          <Button variant="primary" disabled={!output} onClick={copy}>{copied ? 'Copied ✓' : 'Copy'}</Button>
        </div>
      </header>
      <p role="status" className="sr-only">{copied ? `Copied the full ${format === 'markdown' ? 'Markdown' : 'JSON'} output.` : ''}</p>
      {copyError && <p role="alert" className="m-0 mt-3 text-[13px] text-accent">{copyError}</p>}

      {error && <div role="alert" className="mt-6 flex flex-wrap items-center gap-3 text-[14px] font-light">
        <span className="text-accent">{error}</span>
        <Button size="sm" onClick={load}>Retry</Button>
      </div>}

      <div className="mt-4 flex flex-col gap-6 md:mt-[30px] lg:grid lg:grid-cols-[300px_minmax(0,1fr)] lg:gap-14">
        <aside aria-label="About this output" className="flex flex-col gap-[18px] max-md:hidden">
          <dl className="m-0 grid grid-cols-4 gap-3.5 lg:grid-cols-2">
            {([['Goals', stats && padCount(stats.goals)], ['Active', stats && padCount(stats.active)], ['Steps', stats && padCount(stats.steps)], ['Characters', output && text.length.toLocaleString('en-US')]] as const).map(([label, value]) => (
              <div key={label} className="flex flex-col-reverse">
                <Label as="dt" className="!text-[9px]">{label}</Label>
                <dd className="m-0 font-mono text-[22px]">{value ?? '–'}</dd>
              </div>
            ))}
          </dl>
          {stats?.goals === 0 && <p className="m-0 text-[13px] font-light text-ink-60">No goals saved yet. <a href="/timeline" className="text-ink underline">Add them on the timeline</a>.</p>}
          <div className="flex flex-col gap-2 border-t border-ink-30 pt-4">
            <Label>Endpoint</Label>
            <a href={endpoint} className="border border-ink-30 px-3 py-2.5 font-mono text-[13px] text-ink no-underline hover:border-ink-50"><code>GET {endpoint.replace(/\?.*/, '')}{endpoint.includes('?') && <><wbr />{endpoint.slice(endpoint.indexOf('?'))}</>}</code></a>
            <span className="text-[13px] font-light leading-normal text-ink-60">Opens the full output (the page shows JSON pretty-printed). Your MCP server's get_career_overview starts from the same rules and goals.</span>
          </div>
          <div className="border-t border-ink-30 pt-4"><a href="/timeline" className="ui-label hover:text-ink">Edit timeline →</a></div>
        </aside>

        <Box as="section" id={PANEL_ID} role="tabpanel" aria-label={`${format === 'markdown' ? 'Markdown' : 'JSON'} preview`}
          className="relative mx-[3px] min-w-0 p-4 md:mx-0 md:px-6 md:py-5">
          <Label className="absolute right-[18px] top-3.5 !text-[9px] max-md:hidden">{agentFileName(format)}</Label>
          {output
            ? <>
                <pre className="m-0 whitespace-pre-wrap break-words font-mono text-[11.5px] leading-[1.65] text-ink [overflow-wrap:anywhere] md:pr-28 md:text-[12.5px] md:leading-[1.7]">{preview.text}</pre>
                {preview.truncated && <p className="m-0 mt-4 font-mono text-[11.5px] text-ink-60 md:text-[12.5px]">… {preview.hiddenChars.toLocaleString('en-US')} more characters. Copy takes all of it.</p>}
              </>
            : !error && <p role="status" className="m-0 text-[13px] text-ink-60">Loading…</p>}
        </Box>

        <div className="flex flex-wrap items-center justify-between gap-3 md:hidden">
          <a href={endpoint} className="ui-label !text-[9px] hover:text-ink">Also at GET {endpoint}</a>
          <a href="/timeline" className="ui-label !text-[9px] hover:text-ink">Edit timeline →</a>
        </div>
      </div>
    </div>
  );
}
