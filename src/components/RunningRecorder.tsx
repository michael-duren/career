import { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { enqueueClip, queuedClips, syncClips, type QueuedClip } from '../lib/running-queue';
import { recordingStamp, waveformPoints } from '../lib/audio-thoughts';
import { cn } from '../lib/utils';
import { Button, Label, buttonClassName } from './ui';

const MAX_MS = 1800000, TICK_MS = 100, FLOOR_DB = -60;
const clock = (ms: number) => { const t = Math.floor(ms / 100); return `${String(Math.floor(t / 600)).padStart(2, '0')}:${String(Math.floor(t / 10) % 60).padStart(2, '0')}.${t % 10}`; };
const toMeter = (rms: number) => rms <= 0 ? 0 : Math.max(0, Math.min(1, (20 * Math.log10(rms) - FLOOR_DB) / -FLOOR_DB));

/** The live input level as a line (docs/redesign/pages/audio-thoughts.md): flat when silent, swinging with the voice. */
function Waveform({ history, level, recording }: { history: number[]; level: number; recording: boolean }) {
  return <div role="meter" aria-label="Microphone level" aria-valuemin={0} aria-valuemax={1} aria-valuenow={level} className="w-full max-w-[360px]">
    <svg aria-hidden="true" viewBox="0 0 360 44" preserveAspectRatio="none" className={cn('block h-11 w-full', recording ? 'text-ink' : 'text-ink-30')}>
      <polyline points={waveformPoints(history)} fill="none" stroke="currentColor" strokeWidth="1" vectorEffect="non-scaling-stroke" />
    </svg>
  </div>;
}

export default function RunningRecorder() {
  const [recording, setRecording] = useState(false), [starting, setStarting] = useState(false);
  const [elapsed, setElapsed] = useState(0), [waiting, setWaiting] = useState(0);
  const [level, setLevel] = useState(0);
  const [error, setError] = useState(''), [message, setMessage] = useState(''), [rescue, setRescue] = useState(''), [syncing, setSyncing] = useState(false);
  const recorder = useRef<MediaRecorder | null>(null), wake = useRef<WakeLockSentinel | null>(null);
  const audioContext = useRef<AudioContext | null>(null), timer = useRef<ReturnType<typeof setInterval> | undefined>(undefined);
  const pending = useRef<QueuedClip | null>(null), started = useRef(0);
  // Level history lives in a ref; the elapsed tick re-renders the waveform without copying the array.
  const history = useRef<number[]>([]);
  const count = () => void queuedClips().then(c => setWaiting(c.length)).catch(() => setError('Local audio storage unavailable. Keep this tab open and download your recording.'));
  const sync = () => { count(); setSyncing(true); void syncClips(count).catch(e => setMessage(e.message)).finally(() => setSyncing(false)); };
  async function holdScreen() { try { if (document.visibilityState === 'visible' && navigator.wakeLock) wake.current = await navigator.wakeLock.request('screen'); } catch { setMessage('Screen wake lock unavailable. Keep the screen on while recording.'); } }
  function release() { clearInterval(timer.current); void wake.current?.release(); wake.current = null; void audioContext.current?.close(); audioContext.current = null; setLevel(0); }
  async function save(clip: QueuedClip) {
    pending.current = clip;
    try { await enqueueClip(clip); pending.current = null; setRescue(current => { if (current) URL.revokeObjectURL(current); return ''; }); setError(''); setMessage('Take saved on this device.'); sync(); }
    catch { setError('Could not save audio locally. Download it now or retry saving before leaving this page.'); setRescue(URL.createObjectURL(clip.audio)); }
  }
  async function start() {
    setStarting(true); setError(''); setMessage('');
    let stream: MediaStream | undefined;
    try {
      if (!navigator.mediaDevices?.getUserMedia || typeof MediaRecorder === 'undefined') throw new Error('Recording unavailable in this browser. Import a voice memo instead.');
      stream = await navigator.mediaDevices.getUserMedia({ audio: { noiseSuppression: true, echoCancellation: true, autoGainControl: true } });
      const mimeType = ['audio/webm;codecs=opus', 'audio/mp4', 'audio/ogg;codecs=opus'].find(t => MediaRecorder.isTypeSupported(t));
      const next = new MediaRecorder(stream, { ...(mimeType ? { mimeType } : {}), audioBitsPerSecond: 24000 });
      recorder.current = next; const chunks: Blob[] = []; started.current = Date.now();
      const clientId = crypto.randomUUID(), recordedAt = new Date(started.current).toISOString();
      next.ondataavailable = event => { if (event.data.size) chunks.push(event.data); };
      next.onerror = () => { setError('Recording was interrupted. Check the saved audio; import a native voice memo if needed.'); if (next.state !== 'inactive') next.stop(); };
      next.onstop = () => {
        stream?.getTracks().forEach(track => track.stop()); release(); setRecording(false);
        // Idle draws a flat line again; the next take starts a fresh trace.
        history.current = [];
        const audio = new Blob(chunks, { type: next.mimeType || mimeType || 'audio/webm' });
        if (audio.size) void save({ clientId, recordedAt, durationMs: Math.min(MAX_MS, Date.now() - started.current), audio });
        else setError('No audio was captured. Try importing a native voice memo.');
      };
      next.start(1000); setRecording(true); setElapsed(0); history.current = []; navigator.vibrate?.(50); await holdScreen();
      // Stop can land while the wake lock request is pending; don't start metering a finished take.
      if (next.state === 'inactive') { release(); return; }
      let analyser: AnalyserNode | undefined;
      try { const ctx = new AudioContext(); audioContext.current = ctx; analyser = ctx.createAnalyser(); ctx.createMediaStreamSource(stream).connect(analyser); } catch { /* Recording still works without the meter. */ }
      const samples = analyser && new Float32Array(analyser.fftSize);
      timer.current = setInterval(() => {
        const now = Date.now(), ms = now - started.current; setElapsed(ms);
        if (analyser && samples) {
          analyser.getFloatTimeDomainData(samples);
          const rms = Math.sqrt(samples.reduce((sum, v) => sum + v * v, 0) / samples.length), value = toMeter(rms);
          history.current.push(value); setLevel(value);
        }
        if (ms >= MAX_MS || chunks.reduce((sum, c) => sum + c.size, 0) > 14 * 1024 * 1024) { if (next.state !== 'inactive') next.stop(); }
      }, TICK_MS);
    } catch (e) { stream?.getTracks().forEach(track => track.stop()); release(); setError((e as Error).message); }
    finally { setStarting(false); }
  }
  useEffect(() => {
    sync();
    const online = () => sync(); const visible = () => { if (recorder.current?.state === 'recording') void holdScreen(); else sync(); };
    const leaving = (e: BeforeUnloadEvent) => { if (recorder.current?.state === 'recording' || pending.current) e.preventDefault(); };
    const retry = setInterval(sync, 30000);
    // Deleting a thought can discard its unsent takes.
    window.addEventListener('workspace-saved', count);
    window.addEventListener('online', online); document.addEventListener('visibilitychange', visible); window.addEventListener('beforeunload', leaving);
    return () => { clearInterval(retry); window.removeEventListener('workspace-saved', count); window.removeEventListener('online', online); document.removeEventListener('visibilitychange', visible); window.removeEventListener('beforeunload', leaving); if (recorder.current?.state === 'recording') recorder.current.stop(); release(); };
  }, []);
  async function importFile(file: File) {
    if (file.size > 15 * 1024 * 1024 || !file.size) { setError('Choose an audio file between 1 byte and 15 MiB.'); return; }
    const mime = file.type.startsWith('audio/') ? file.type : ({ m4a: 'audio/mp4', mp3: 'audio/mpeg', wav: 'audio/wav', webm: 'audio/webm', ogg: 'audio/ogg', flac: 'audio/flac' } as Record<string, string>)[file.name.split('.').pop()?.toLowerCase() ?? ''];
    if (!mime) { setError('Choose a supported audio file.'); return; }
    const url = URL.createObjectURL(file);
    try {
      const audio = new Audio(url); const duration = await new Promise<number>((resolve, reject) => { audio.onloadedmetadata = () => resolve(audio.duration); audio.onerror = () => reject(new Error('This browser cannot read the audio duration. Convert the memo to MP3, M4A, or WAV first.')); setTimeout(() => reject(new Error('Could not read audio duration.')), 10000); });
      if (!Number.isFinite(duration) || duration > 1800) throw new Error('Import a clip no longer than 30 minutes with readable duration metadata.');
      await save({ clientId: crypto.randomUUID(), audio: new Blob([file], { type: mime }), recordedAt: new Date(file.lastModified || Date.now()).toISOString(), durationMs: Math.round(duration * 1000) });
    } catch (e) { setError((e as Error).message); } finally { URL.revokeObjectURL(url); }
  }
  const busy = recording || !!rescue;
  const root = useRef<HTMLDivElement>(null);
  useLayoutEffect(() => {
    // Finishing a take while reading a thought hides the recorder (and the record button disables itself),
    // so focus has already fallen to <body>; hand it to the note's Back button instead.
    const el = root.current, active = document.activeElement;
    if (el && !el.offsetParent && (el.contains(active) || active === document.body)) document.querySelector<HTMLButtonElement>('[data-thought-reading] button')?.focus();
  }, [busy, starting, error]);
  const prompt = starting ? 'OPENING MIC…' : recording ? 'TAP TO STOP' : 'TAP TO RECORD';
  // Stay visible over a thought being read while anything needs attention: opening the mic, a live take, an unsaved rescue,
  // or an error (data-recorder-active; the hide rule is in src/pages/audio-thoughts.astro).
  return <div ref={root} data-recorder data-recorder-active={busy || starting || !!error || undefined}>
    <section aria-label="Audio thoughts recorder" className="flex flex-col items-center gap-3.5 md:gap-[22px] md:pt-2.5">
      {/* The drawn target: outer circle, dashed inner circle and a crosshair around the record button. */}
      <div className="relative size-[180px] md:size-[240px]">
        <div aria-hidden="true" className="absolute inset-0 rounded-full border border-solid border-ink-30" />
        <div aria-hidden="true" className="absolute inset-[18px] rounded-full border border-dashed border-ink-30 md:inset-6" />
        <div aria-hidden="true" className="absolute -top-2 -bottom-2 left-1/2 border-0 border-l border-solid border-ink-14 md:-top-2.5 md:-bottom-2.5" />
        <div aria-hidden="true" className="absolute top-1/2 -right-2 -left-2 border-0 border-t border-solid border-ink-14 md:-right-2.5 md:-left-2.5" />
        <button type="button" aria-label="Record" aria-pressed={recording} title={recording ? 'Stop' : 'Record'} disabled={starting || !!rescue}
          onClick={() => recording ? recorder.current?.stop() : void start()}
          className="absolute inset-[46px] flex items-center justify-center rounded-full border border-solid border-ink bg-paper p-0 hover:bg-ink-05 disabled:cursor-not-allowed disabled:opacity-50 md:inset-[60px]">
          {recording
            ? <span className="size-6 bg-accent md:size-7" />
            : <span className={cn('size-9 rounded-full bg-accent md:size-11', starting && 'animate-pulse')} />}
        </button>
      </div>
      <span className="text-[13px] font-light tracking-[0.2em] md:text-[15px]">{prompt}</span>
      {(recording || elapsed > 0) && <p className={cn('m-0 font-mono text-sm tabular-nums', recording ? 'text-ink' : 'text-ink-60')} role="timer" aria-live="off" aria-label="Elapsed recording time">{clock(elapsed)} <span className="text-ink-60">/ 30:00</span></p>}
      <Waveform history={history.current} level={level} recording={recording} />
      <Label className="!text-[9px] md:hidden">{waiting} waiting to upload · works offline</Label>
      <div className="grid w-full grid-cols-2 border-0 border-t border-solid border-ink-30 pt-3.5 max-md:hidden">
        <span className="flex flex-col"><span className="font-mono text-lg">{waiting}</span><Label className="!text-[9px]">Waiting to upload</Label></span>
        <span className="flex flex-col"><span className="font-mono text-lg">Homelab</span><Label className="!text-[9px]">Transcriber</Label></span>
      </div>
      <div className="flex flex-wrap justify-center gap-2.5 md:self-stretch md:justify-start">
        <label className={cn(buttonClassName({ size: 'sm' }), 'cursor-pointer', busy && 'pointer-events-none opacity-50')} title="Import voice memo">
          Import memo
          <input className="ui-sr-only" type="file" accept="audio/*,.m4a,.webm,.wav,.mp3,.ogg,.flac" disabled={busy} onChange={e => { const f = e.target.files?.[0]; if (f) void importFile(f); e.target.value = ''; }} />
        </label>
        <Button size="sm" onClick={sync} disabled={syncing} title="Retry uploads">{syncing ? 'Syncing…' : 'Sync now'}</Button>
      </div>
      <span className="self-start font-hand text-sm text-accent max-md:hidden">works offline — uploads when you're back</span>
      <span className="ui-sr-only" role="status">{starting ? 'Opening microphone' : recording ? 'Recording' : 'Ready'}. {waiting ? `${waiting} waiting to upload` : 'All clips uploaded'}.</span>
      {(message || error || rescue) && <div className="flex w-full flex-col gap-2 text-sm">
        {message && <p role="status" className="m-0 text-ink-60">{message}</p>}
        {error && <p role="alert" className="m-0 border border-solid border-accent px-3 py-2 text-ink">{error}</p>}
        {rescue && <div className="flex flex-wrap gap-2"><a className={buttonClassName({ size: 'sm' })} href={rescue} download="audio-thought">Download unsaved audio</a><Button size="sm" onClick={() => pending.current && void save(pending.current)}>Retry local save</Button></div>}
      </div>}
      <p className="m-0 w-full text-xs font-light leading-relaxed text-ink-60">Keep the screen on while recording; screen lock can interrupt capture. Takes within 90 minutes of each other join the same thought.</p>
    </section>
  </div>;
}
const statusTone: Record<string, string> = { done: 'bg-ink', pending: 'bg-ink-30', transcribing: 'bg-ink animate-pulse', failed: 'bg-accent' };
type Clip = { clipId: string; status: string; transcript: string; error: string; recordedAt: string };
const clipStamp = (recordedAt: string) => { const { date, time } = recordingStamp(recordedAt); return date ? `${date} · ${time}` : ''; };
export function RunningClips({ noteId }: { noteId: string }) {
  const [clips, setClips] = useState<Clip[]>([]), [error, setError] = useState('');
  useEffect(() => {
    let live = true;
    const load = async () => { try { const response = await fetch(`/api/running/${encodeURIComponent(noteId)}/status`, { cache: 'no-store' }); if (!response.ok) throw new Error('Clip statuses unavailable.'); const data = await response.json(); if (live) { setClips(data.clips); setError(''); } } catch (e) { if (live) setError((e as Error).message); } };
    void load(); const timer = setInterval(load, 5000); return () => { live = false; clearInterval(timer); };
  }, [noteId]);
  // Thoughts typed in the editor have no takes; skip the empty panel.
  if (!clips.length && !error) return null;
  return <section className="mt-6 border-0 border-t border-solid border-ink-30 pt-4" aria-label="Audio takes">
    <h3 className="m-0 flex items-baseline justify-between gap-3 text-xs font-medium tracking-[0.24em]"><span>TAKES</span><Label>{clips.length} {clips.length === 1 ? 'take' : 'takes'}</Label></h3>
    {error && <p role="alert" className="mt-3 border border-solid border-accent px-3 py-2 text-sm text-ink">{error}</p>}
    <ol className="m-0 mt-2 list-none p-0">{clips.map((clip, i) => <li key={clip.clipId} className="flex flex-col gap-2 border-0 border-b border-solid border-ink-14 py-3.5">
      <div className="flex flex-wrap items-center justify-between gap-2 font-mono text-[11px]">
        <span>Take {i + 1} <span className="text-ink-60">· {clipStamp(clip.recordedAt)}</span></span>
        <span className="flex items-center gap-1.5 uppercase tracking-[0.08em] text-ink-60"><span className={`size-2 rounded-full ${statusTone[clip.status] ?? 'bg-ink-30'}`} />{clip.status}</span>
      </div>
      <audio className="h-9 w-full" controls preload="none" src={`/api/running/clips/${encodeURIComponent(noteId)}/${clip.clipId}/audio`} />
      {clip.error && <p className="m-0 text-xs text-accent">{clip.error}</p>}
      <div className="flex flex-wrap items-start justify-between gap-2">
        <details className="min-w-0 flex-1 text-sm"><summary className="cursor-pointer text-xs text-ink-60">Original transcript</summary><p className="mt-2 whitespace-pre-wrap font-light">{clip.transcript || <span className="text-ink-60 italic">Not transcribed yet.</span>}</p></details>
        <Button size="sm" disabled={clip.status === 'transcribing'} onClick={async () => { try { const r = await fetch(`/api/running/clips/${encodeURIComponent(noteId)}/${clip.clipId}/retranscribe`, { method: 'POST', headers: { 'Content-Type': 'application/json' } }); if (!r.ok) throw new Error('Could not queue transcription.'); setClips(current => current.map(c => c.clipId === clip.clipId ? { ...c, status: 'pending' } : c)); } catch (e) { setError((e as Error).message); } }}>Retranscribe</Button>
      </div>
    </li>)}</ol>
    <p className="m-0 mt-3 text-xs font-light text-ink-60">Retranscribing updates the original take transcript. Your edited text is preserved. Reload and reopen the thought to read newly appended transcripts.</p>
  </section>;
}
