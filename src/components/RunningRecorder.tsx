import { useEffect, useRef, useState } from 'react';
import { CloudUpload, Download, Mic, RefreshCw, Square, Upload } from 'lucide-react';
import { enqueueClip, queuedClips, syncClips, type QueuedClip } from '../lib/running-queue';
// Hardware-style transport key shared by the dock and the clip list.
// Shape, padding and face stay out of the base so per-use overrides never fight Tailwind's CSS order.
const keyBase = 'inline-flex items-center justify-center border border-black/70 text-xs font-medium text-zinc-200 shadow-[inset_0_1px_0_rgb(255_255_255/0.08),0_1px_2px_rgb(0_0_0/0.6)] active:translate-y-px disabled:pointer-events-none disabled:opacity-40';
const keyFace = 'bg-linear-to-b from-zinc-700 to-zinc-800 hover:from-zinc-600 hover:to-zinc-700';
const key = `${keyBase} ${keyFace} gap-1.5 rounded-md px-3`;
const dockKey = `${keyBase} ${keyFace} h-12 min-w-16 flex-col gap-0.5 rounded-md px-3 lg:h-11 lg:min-w-0 lg:flex-row lg:gap-1.5`;
const MAX_MS = 1800000, TICK_MS = 100, FLOOR_DB = -60;
const clock = (ms: number) => { const t = Math.floor(ms / 100); return `${String(Math.floor(t / 600)).padStart(2, '0')}:${String(Math.floor(t / 10) % 60).padStart(2, '0')}.${t % 10}`; };
const short = (s: number) => `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
const toMeter = (rms: number) => rms <= 0 ? 0 : Math.max(0, Math.min(1, (20 * Math.log10(rms) - FLOOR_DB) / -FLOOR_DB));
// Arrangement view grows in whole minutes so the ruler stays readable as a take gets longer.
const span = (ms: number) => Math.max(60, Math.ceil((ms / 1000 + 5) / 60) * 60);
const step = (seconds: number) => [5, 10, 15, 30, 60, 120, 300].find(s => seconds / s <= 8) ?? 300;
const codec = (mime: string) => mime.includes('opus') ? 'OPUS' : mime.includes('mp4') ? 'AAC' : mime.split('/')[1]?.split(';')[0]?.toUpperCase() || '—';

function Led({ on, color, label }: { on: boolean; color: 'red' | 'green' | 'amber'; label: string }) {
  const lit = { red: 'bg-red-500 shadow-[0_0_8px_rgb(239_68_68)]', green: 'bg-emerald-400 shadow-[0_0_8px_rgb(52_211_153)]', amber: 'bg-amber-400 shadow-[0_0_8px_rgb(251_191_36)]' }[color];
  return <span className="flex items-center gap-1.5 font-mono text-[10px] tracking-widest text-zinc-400 uppercase"><span className={`size-2 rounded-full ${on ? lit : 'bg-zinc-700'}`} />{label}</span>;
}

function Lane({ history, elapsed, recording }: { history: number[]; elapsed: number; recording: boolean }) {
  const canvas = useRef<HTMLCanvasElement>(null), [width, setWidth] = useState(0);
  const seconds = span(elapsed), tick = step(seconds);
  useEffect(() => {
    const el = canvas.current; if (!el) return;
    const observer = new ResizeObserver(([entry]) => setWidth(entry.contentRect.width)); observer.observe(el);
    return () => observer.disconnect();
  }, []);
  useEffect(() => {
    const el = canvas.current, ctx = el?.getContext('2d'); if (!el || !ctx || !width) return;
    const dpr = window.devicePixelRatio || 1, h = el.clientHeight;
    el.width = Math.round(width * dpr); el.height = Math.round(h * dpr); ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, width, h);
    const x = (ms: number) => ms / (seconds * 1000) * width;
    for (let s = 0; s <= seconds; s += tick / 2) { ctx.fillStyle = s % tick ? '#1c1c21' : '#26262c'; ctx.fillRect(Math.round(x(s * 1000)), 0, 1, h); }
    const header = 16, mid = header + (h - header) / 2, amp = (h - header) / 2 - 4;
    ctx.fillStyle = '#26262c'; ctx.fillRect(0, Math.round(mid), width, 1);
    if (!history.length) return;
    const end = x(elapsed);
    ctx.fillStyle = recording ? 'rgb(239 68 68 / 0.14)' : 'rgb(56 189 248 / 0.12)'; ctx.fillRect(0, 0, end, h);
    ctx.fillStyle = recording ? 'rgb(239 68 68 / 0.7)' : 'rgb(56 189 248 / 0.55)'; ctx.fillRect(0, 0, end, header);
    ctx.fillStyle = '#fafafa'; ctx.font = '600 10px ui-monospace, monospace'; ctx.textBaseline = 'middle';
    ctx.save(); ctx.beginPath(); ctx.rect(0, 0, end - 4, header); ctx.clip();
    ctx.fillText(recording ? 'Take · recording' : 'Take · saved', 6, header / 2 + 1); ctx.restore();
    ctx.fillStyle = recording ? '#fca5a5' : '#7dd3fc';
    const per = end / history.length;
    // Draw one bar per pixel column so long takes stay crisp instead of overdrawing.
    for (let px = 0; px < end; px++) {
      const from = Math.floor(px / per), to = Math.max(from + 1, Math.floor((px + 1) / per));
      let peak = 0; for (let i = from; i < to && i < history.length; i++) peak = Math.max(peak, history[i]);
      const bar = Math.max(1, peak * amp); ctx.fillRect(px, mid - bar, 1, bar * 2);
    }
    ctx.fillStyle = 'rgb(0 0 0 / 0.5)'; ctx.fillRect(end - 1, 0, 1, h);
  }, [history, history.length, elapsed, recording, seconds, tick, width]);
  const marks = Array.from({ length: Math.floor(seconds / tick) + 1 }, (_, i) => i * tick);
  const head = Math.min(100, elapsed / (seconds * 1000) * 100);
  return <div className="relative min-w-0 flex-1">
    <div className="relative h-6 border-b border-black/60 bg-zinc-900 font-mono text-[10px] text-zinc-500">
      {marks.map(s => <span key={s} className="absolute top-0 h-full border-l border-zinc-700 pl-1 leading-6" style={{ left: `${s / seconds * 100}%` }}>{s < seconds ? short(s) : ''}</span>)}
    </div>
    <div className="relative h-32 bg-[#101013] sm:h-36">
      <canvas ref={canvas} className="absolute inset-0 size-full" aria-hidden />
      {!history.length && !recording && <p className="absolute inset-0 flex items-center justify-center px-6 text-center text-xs text-zinc-500">Arm the mic and press record — the take draws here as you talk.</p>}
    </div>
    {(recording || history.length > 0) && <div className="pointer-events-none absolute inset-y-0 w-px bg-zinc-50 shadow-[0_0_6px_rgb(255_255_255/0.6)]" style={{ left: `${head}%` }}><span className="absolute -top-px -left-[5px] border-x-[5px] border-t-[7px] border-x-transparent border-t-zinc-50" /></div>}
  </div>;
}

function Meter({ level, peak, vertical }: { level: number; peak: number; vertical: boolean }) {
  // Segment colours follow the usual -60…0 dBFS meter: green, then amber above -12, red above -3.
  const segments = vertical ? 30 : 40, lit = Math.round(level * segments), hold = Math.round(peak * segments);
  const tone = (i: number) => { const db = FLOOR_DB + (i + 1) / segments * -FLOOR_DB; return db > -3 ? 'bg-red-500' : db > -12 ? 'bg-amber-400' : 'bg-emerald-400'; };
  return <div className={`flex gap-[2px] ${vertical ? 'h-full w-3 flex-col-reverse' : 'h-2 w-full'}`}>
    {Array.from({ length: segments }, (_, i) => <span key={i} className={`flex-1 rounded-[1px] ${i < lit || i === hold - 1 ? tone(i) : 'bg-zinc-800'}`} />)}
  </div>;
}

export default function RunningRecorder() {
  const [recording, setRecording] = useState(false), [starting, setStarting] = useState(false);
  const [elapsed, setElapsed] = useState(0), [waiting, setWaiting] = useState(0);
  const [level, setLevel] = useState(0), [peak, setPeak] = useState(0), [format, setFormat] = useState('');
  const [error, setError] = useState(''), [message, setMessage] = useState(''), [rescue, setRescue] = useState(''), [syncing, setSyncing] = useState(false);
  const recorder = useRef<MediaRecorder | null>(null), wake = useRef<WakeLockSentinel | null>(null);
  const audioContext = useRef<AudioContext | null>(null), timer = useRef<ReturnType<typeof setInterval> | undefined>(undefined);
  const pending = useRef<QueuedClip | null>(null), started = useRef(0), held = useRef({ value: 0, at: 0 });
  // Level history lives in a ref; the elapsed tick re-renders the lane without copying the array.
  const history = useRef<number[]>([]);
  const count = () => void queuedClips().then(c => setWaiting(c.length)).catch(() => setError('Local audio storage unavailable. Keep this tab open and download your recording.'));
  const sync = () => { count(); setSyncing(true); void syncClips(count).catch(e => setMessage(e.message)).finally(() => setSyncing(false)); };
  async function holdScreen() { try { if (document.visibilityState === 'visible' && navigator.wakeLock) wake.current = await navigator.wakeLock.request('screen'); } catch { setMessage('Screen wake lock unavailable. Keep the screen on while recording.'); } }
  function release() { clearInterval(timer.current); void wake.current?.release(); wake.current = null; void audioContext.current?.close(); audioContext.current = null; setLevel(0); setPeak(0); }
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
        const audio = new Blob(chunks, { type: next.mimeType || mimeType || 'audio/webm' });
        if (audio.size) void save({ clientId, recordedAt, durationMs: Math.min(MAX_MS, Date.now() - started.current), audio });
        else setError('No audio was captured. Try importing a native voice memo.');
      };
      next.start(1000); setRecording(true); setElapsed(0); history.current = []; setFormat(codec(next.mimeType || mimeType || '')); navigator.vibrate?.(50); await holdScreen();
      // Stop can land while the wake lock request is pending; don't start metering a finished take.
      if (next.state === 'inactive') { release(); return; }
      let analyser: AnalyserNode | undefined;
      try { const ctx = new AudioContext(); audioContext.current = ctx; analyser = ctx.createAnalyser(); ctx.createMediaStreamSource(stream).connect(analyser); } catch { /* Recording still works without the meter. */ }
      const samples = analyser && new Float32Array(analyser.fftSize); held.current = { value: 0, at: 0 };
      timer.current = setInterval(() => {
        const now = Date.now(), ms = now - started.current; setElapsed(ms);
        if (analyser && samples) {
          analyser.getFloatTimeDomainData(samples);
          const rms = Math.sqrt(samples.reduce((sum, v) => sum + v * v, 0) / samples.length), value = toMeter(rms);
          if (value >= held.current.value || now - held.current.at > 1500) held.current = { value, at: now };
          history.current.push(value); setLevel(value); setPeak(held.current.value);
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
    window.addEventListener('online', online); document.addEventListener('visibilitychange', visible); window.addEventListener('beforeunload', leaving);
    return () => { clearInterval(retry); window.removeEventListener('online', online); document.removeEventListener('visibilitychange', visible); window.removeEventListener('beforeunload', leaving); if (recorder.current?.state === 'recording') recorder.current.stop(); release(); };
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
  const busy = recording || !!rescue, db = level > 0 ? Math.round(FLOOR_DB + level * -FLOOR_DB) : null;
  return <><section aria-label="Audio thoughts recorder" className="mb-8 overflow-hidden rounded-xl border border-black bg-zinc-900 shadow-[0_12px_40px_rgb(0_0_0/0.5),inset_0_1px_0_rgb(255_255_255/0.05)]">
    <div className="flex flex-wrap items-stretch gap-3 border-b border-black bg-linear-to-b from-zinc-800 to-zinc-900 p-3 lg:flex-nowrap">
      <div className="fixed inset-x-0 bottom-0 z-20 border-t border-black bg-zinc-900/95 px-6 pt-3 pb-[max(1rem,env(safe-area-inset-bottom))] backdrop-blur lg:static lg:z-auto lg:border-0 lg:bg-transparent lg:p-0 lg:backdrop-blur-none">
        <div className="mx-auto flex max-w-sm items-center justify-between gap-2 lg:max-w-none lg:justify-start">
          <label className={`${dockKey} cursor-pointer ${busy ? 'pointer-events-none opacity-40' : ''}`} title="Import voice memo">
            <Upload className="size-4" /><span className="text-[10px] lg:text-xs">Import</span>
            <input className="sr-only" type="file" accept="audio/*,.m4a,.webm,.wav,.mp3,.ogg,.flac" disabled={busy} onChange={e => { const f = e.target.files?.[0]; if (f) void importFile(f); e.target.value = ''; }} />
          </label>
          <button className={`${keyBase} ${keyFace} size-11 rounded-md max-lg:hidden`} aria-label="Stop recording" title="Stop" disabled={!recording || starting} onClick={() => recorder.current?.stop()}><Square className="size-4 fill-current" /></button>
          <button className={`${keyBase} size-16 rounded-full lg:size-11 lg:rounded-md ${recording ? 'bg-linear-to-b from-red-900 to-red-950 ring-2 ring-red-500/70' : keyFace}`} aria-label="Record" aria-pressed={recording} title={recording ? 'Stop' : 'Record'} disabled={starting || !!rescue} onClick={() => recording ? recorder.current?.stop() : void start()}>
            {recording ? <Square className="size-6 fill-red-500 text-red-500 lg:hidden" /> : null}
            <span className={`rounded-full bg-red-600 ${recording ? 'hidden animate-pulse shadow-[0_0_12px_rgb(239_68_68)] lg:block lg:size-4' : 'size-8 lg:size-4'} ${starting ? 'animate-pulse' : ''}`} />
          </button>
          <button className={dockKey} onClick={sync} title="Retry uploads"><RefreshCw className={`size-4 ${syncing ? 'animate-spin' : ''}`} /><span className="text-[10px] lg:text-xs">Sync</span></button>
        </div>
      </div>
      <div className="flex min-w-0 flex-1 items-center justify-between gap-4 rounded-md border border-black bg-[#07100e] px-4 py-2 shadow-[inset_0_2px_8px_rgb(0_0_0/0.8)]">
        <div>
          <p className={`font-mono text-3xl leading-none tabular-nums sm:text-4xl ${recording ? 'text-teal-300 [text-shadow:0_0_10px_rgb(94_234_212/0.5)]' : 'text-teal-300/50'}`} role="timer" aria-live="off" aria-label="Elapsed recording time">{clock(elapsed)}</p>
          <p className="mt-1 font-mono text-[10px] tracking-widest text-teal-300/40 uppercase">min : sec · limit 30:00</p>
        </div>
        <dl className="hidden grid-cols-2 gap-x-4 gap-y-0.5 text-right font-mono text-[10px] tracking-wider text-teal-300/60 uppercase sm:grid">
          <dt className="text-teal-300/35">Codec</dt><dd>{format || '—'}</dd>
          <dt className="text-teal-300/35">Rate</dt><dd>24 kbps</dd>
          <dt className="text-teal-300/35">Input</dt><dd className="tabular-nums">{db === null ? '-∞' : db} dB</dd>
        </dl>
      </div>
      <div className="flex w-full items-center justify-between gap-4 rounded-md border border-black/60 bg-zinc-950/60 px-3 py-2 lg:w-auto lg:flex-col lg:items-start lg:justify-center">
        <Led on={recording} color="red" label={starting ? 'Arming' : recording ? 'Rec' : 'Idle'} />
        <Led on={waiting > 0} color="amber" label={waiting ? `${waiting} queued` : 'Queue clear'} />
        <span className="sr-only" role="status">{starting ? 'Opening microphone' : recording ? 'Recording' : 'Ready'}. {waiting ? `${waiting} waiting to upload` : 'All clips uploaded'}.</span>
      </div>
    </div>
    <div className="flex">
      <div className="hidden w-40 shrink-0 flex-col justify-between border-r border-black bg-zinc-800/80 p-3 sm:flex">
        <div>
          <p className="flex items-center gap-2 text-xs font-semibold text-zinc-200"><span className="h-3 w-1 rounded-full bg-red-500" />Voice</p>
          <p className="mt-1 flex items-center gap-1 font-mono text-[10px] text-zinc-500"><Mic className="size-3" />Mic · mono</p>
        </div>
        <div className="flex gap-1">
          <span className={`grid size-6 place-items-center rounded border border-black font-mono text-[10px] font-bold ${recording ? 'bg-red-600 text-white' : 'bg-zinc-700 text-zinc-400'}`} title="Record arm">R</span>
          <span className={`grid size-6 place-items-center rounded border border-black font-mono text-[10px] font-bold ${level > 0 ? 'bg-emerald-600 text-white' : 'bg-zinc-700 text-zinc-400'}`} title="Input monitoring">I</span>
        </div>
      </div>
      <Lane history={history.current} elapsed={elapsed} recording={recording} />
      <div className="hidden w-14 shrink-0 flex-col items-center gap-2 border-l border-black bg-zinc-800/80 py-2 lg:flex" role="meter" aria-label="Microphone level" aria-valuemin={0} aria-valuemax={1} aria-valuenow={level}>
        <span className="font-mono text-[9px] text-zinc-500">0 dB</span>
        <div className="flex-1"><Meter level={level} peak={peak} vertical /></div>
        <span className="font-mono text-[9px] text-zinc-500">-60</span>
      </div>
    </div>
    <div className="border-t border-black bg-zinc-800/80 px-3 py-2 lg:hidden" role="meter" aria-label="Microphone level" aria-valuemin={0} aria-valuemax={1} aria-valuenow={level}><Meter level={level} peak={peak} vertical={false} /></div>
    <div className="space-y-2 border-t border-black bg-zinc-950 px-4 py-3 text-xs">
      {message && <p role="status" className="flex items-center gap-2 text-zinc-300"><CloudUpload className="size-3.5 text-teal-400" />{message}</p>}
      {error && <p role="alert" className="rounded-md border border-red-900 bg-red-950/60 px-3 py-2 text-red-300">{error}</p>}
      {rescue && <div className="flex flex-wrap gap-2"><a className={`${key} h-9`} href={rescue} download="audio-thought"><Download className="size-4" />Download unsaved audio</a><button className={`${key} h-9`} onClick={() => pending.current && void save(pending.current)}>Retry local save</button></div>}
      <p className="leading-relaxed text-zinc-500">Keep the screen on while recording; screen lock can interrupt capture. Takes within 90 minutes of each other join the same thought.</p>
    </div>
  </section><div className="h-28 lg:hidden" aria-hidden /></>;
}
const status: Record<string, string> = { done: 'bg-emerald-400', pending: 'bg-amber-400', transcribing: 'bg-sky-400 animate-pulse', failed: 'bg-red-500' };
export function RunningClips({ noteId }: { noteId: string }) {
  const [clips, setClips] = useState<Array<{ clipId: string; status: string; transcript: string; error: string; recordedAt: string }>>([]), [error, setError] = useState('');
  useEffect(() => {
    let live = true;
    const load = async () => { try { const response = await fetch(`/api/running/${encodeURIComponent(noteId)}/status`, { cache: 'no-store' }); if (!response.ok) throw new Error('Clip statuses unavailable.'); const data = await response.json(); if (live) { setClips(data.clips); setError(''); } } catch (e) { if (live) setError((e as Error).message); } };
    void load(); const timer = setInterval(load, 5000); return () => { live = false; clearInterval(timer); };
  }, [noteId]);
  return <section className="mt-6 overflow-hidden rounded-xl border border-black bg-zinc-900" aria-label="Audio takes">
    <h3 className="flex items-center justify-between border-b border-black bg-linear-to-b from-zinc-800 to-zinc-900 px-4 py-2 font-mono text-[11px] tracking-widest text-zinc-400 uppercase"><span>Takes</span><span>{clips.length} {clips.length === 1 ? 'take' : 'takes'}</span></h3>
    {error && <p role="alert" className="border-b border-red-900 bg-red-950/60 px-4 py-2 text-sm text-red-300">{error}</p>}
    <ol className="divide-y divide-black">{clips.map((clip, i) => <li key={clip.clipId} className="flex">
      <span className="w-1 shrink-0 bg-sky-500/70" />
      <div className="min-w-0 flex-1 space-y-2 p-3">
        <div className="flex flex-wrap items-center justify-between gap-2 font-mono text-[11px]">
          <span className="text-zinc-300">Take {i + 1} <span className="text-zinc-500">· {new Date(clip.recordedAt).toLocaleString()}</span></span>
          <span className="flex items-center gap-1.5 tracking-wider text-zinc-400 uppercase"><span className={`size-2 rounded-full ${status[clip.status] ?? 'bg-zinc-600'}`} />{clip.status}</span>
        </div>
        <audio className="h-9 w-full" controls preload="none" src={`/api/running/clips/${encodeURIComponent(noteId)}/${clip.clipId}/audio`} />
        {clip.error && <p className="text-xs text-red-300">{clip.error}</p>}
        <div className="flex flex-wrap items-start justify-between gap-2">
          <details className="min-w-0 flex-1 text-sm"><summary className="cursor-pointer text-xs text-zinc-400">Original transcript</summary><p className="mt-2 whitespace-pre-wrap text-zinc-300">{clip.transcript || 'Waiting for local transcription.'}</p></details>
          <button className={`${key} h-8`} disabled={clip.status === 'transcribing'} onClick={async () => { try { const r = await fetch(`/api/running/clips/${encodeURIComponent(noteId)}/${clip.clipId}/retranscribe`, { method: 'POST', headers: { 'Content-Type': 'application/json' } }); if (!r.ok) throw new Error('Could not queue transcription.'); setClips(current => current.map(c => c.clipId === clip.clipId ? { ...c, status: 'pending' } : c)); } catch (e) { setError((e as Error).message); } }}><RefreshCw className="size-3.5" />Retranscribe</button>
        </div>
      </div>
    </li>)}</ol>
    <p className="border-t border-black px-4 py-2 text-xs text-zinc-500">Retranscribing updates the original take transcript. Your edited text is preserved. Reload and reopen the thought to read newly appended transcripts.</p>
  </section>;
}
