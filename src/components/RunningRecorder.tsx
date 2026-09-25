import { useEffect, useRef, useState } from 'react';
import { Download, RefreshCw, Upload } from 'lucide-react';
import { enqueueClip, queuedClips, syncClips, type QueuedClip } from '../lib/running-queue';
const button = 'min-h-12 rounded-lg border border-zinc-600 px-4 py-3 disabled:opacity-50';
const dockButton = 'flex size-14 flex-col items-center justify-center gap-1 rounded-full bg-zinc-800 text-zinc-200 hover:bg-zinc-700 disabled:opacity-40 lg:size-16';
const clock = (seconds: number) => `${Math.floor(seconds / 60)}:${String(seconds % 60).padStart(2, '0')}`;
export default function RunningRecorder() {
  const [recording, setRecording] = useState(false), [starting, setStarting] = useState(false);
  const [seconds, setSeconds] = useState(0), [levels, setLevels] = useState<number[]>([]), [waiting, setWaiting] = useState(0);
  const [error, setError] = useState(''), [message, setMessage] = useState(''), [rescue, setRescue] = useState('');
  const recorder = useRef<MediaRecorder | null>(null), wake = useRef<WakeLockSentinel | null>(null);
  const audioContext = useRef<AudioContext | null>(null), timer = useRef<ReturnType<typeof setInterval> | undefined>(undefined);
  const pending = useRef<QueuedClip | null>(null), started = useRef(0);
  const count = () => void queuedClips().then(c => setWaiting(c.length)).catch(() => setError('Local audio storage unavailable. Keep this tab open and download your recording.'));
  const sync = () => { count(); void syncClips(count).catch(e => setMessage(e.message)); };
  async function holdScreen() { try { if (document.visibilityState === 'visible' && navigator.wakeLock) wake.current = await navigator.wakeLock.request('screen'); } catch { setMessage('Screen wake lock unavailable. Keep the screen on while recording.'); } }
  function release() { clearInterval(timer.current); void wake.current?.release(); wake.current = null; void audioContext.current?.close(); audioContext.current = null; }
  async function save(clip: QueuedClip) {
    pending.current = clip;
    try { await enqueueClip(clip); pending.current = null; setRescue(current => { if (current) URL.revokeObjectURL(current); return ''; }); setError(''); setMessage('Audio saved on this device.'); sync(); }
    catch { setError('Could not save audio locally. Download it now or retry saving before leaving this page.'); setRescue(URL.createObjectURL(clip.audio)); }
  }
  async function start() {
    setStarting(true); setError(''); setMessage('');
    let stream: MediaStream | undefined;
    try {
      if (!navigator.mediaDevices?.getUserMedia || typeof MediaRecorder === 'undefined') throw new Error('Recording unavailable in this browser. Import a voice memo below.');
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
        if (audio.size) void save({ clientId, recordedAt, durationMs: Math.min(1800000, Date.now() - started.current), audio });
        else setError('No audio was captured. Try importing a native voice memo.');
      };
      next.start(1000); setRecording(true); setSeconds(0); setLevels([]); navigator.vibrate?.(50); await holdScreen();
      let analyser: AnalyserNode | undefined;
      try { const ctx = new AudioContext(); audioContext.current = ctx; analyser = ctx.createAnalyser(); ctx.createMediaStreamSource(stream).connect(analyser); } catch { /* Recording still works without the meter. */ }
      timer.current = setInterval(() => {
        const elapsed = Date.now() - started.current; setSeconds(Math.floor(elapsed / 1000));
        if (analyser) { const samples = new Uint8Array(analyser.fftSize); analyser.getByteTimeDomainData(samples); const level = Math.min(1, Math.sqrt(samples.reduce((sum, v) => sum + ((v - 128) / 128) ** 2, 0) / samples.length) * 4); setLevels(current => [...current.slice(-47), level]); }
        if (elapsed >= 1800000 || chunks.reduce((sum, c) => sum + c.size, 0) > 14 * 1024 * 1024) { if (next.state !== 'inactive') next.stop(); }
      }, 250);
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
  const bars = [...Array(Math.max(0, 48 - levels.length)).fill(0), ...levels];
  return <section className="mb-8 lg:grid lg:grid-cols-[1fr_16rem] lg:overflow-hidden lg:rounded-2xl lg:border lg:border-zinc-800 lg:bg-zinc-900/60">
    <div className="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/60 p-5 lg:rounded-none lg:border-0 lg:bg-transparent lg:p-8">
      <div className="flex items-center justify-between text-sm">
        <span className={`flex items-center gap-2 font-medium ${recording ? 'text-red-400' : 'text-zinc-400'}`}><span className={`size-2 rounded-full ${recording ? 'animate-pulse bg-red-500' : 'bg-zinc-600'}`} />{starting ? 'Opening microphone…' : recording ? 'Recording' : 'Ready'}</span>
        <span className="text-zinc-400" role="status">{waiting ? `${waiting} waiting to upload` : 'All clips uploaded'}</span>
      </div>
      <p className={`text-center font-mono text-6xl font-light tabular-nums tracking-tight lg:text-7xl ${recording ? 'text-zinc-50' : 'text-zinc-500'}`} role="timer" aria-live="off">{clock(seconds)}</p>
      <div className="flex h-16 items-center justify-center gap-[3px]" role="meter" aria-label="Microphone level" aria-valuemin={0} aria-valuemax={1} aria-valuenow={levels.at(-1) ?? 0}>
        {bars.map((v, i) => <span key={i} className={`w-1 rounded-full transition-[height] duration-200 ${recording ? 'bg-red-400' : 'bg-zinc-700'}`} style={{ height: `${Math.max(4, v * 64)}px` }} />)}
      </div>
      {message && <p role="status" className="text-center text-sm text-zinc-300">{message}</p>}
      {error && <p role="alert" className="rounded-lg bg-red-950/60 px-3 py-2 text-sm text-red-300">{error}</p>}
      {rescue && <div className="flex flex-wrap gap-3"><a className={`${button} inline-flex items-center gap-2`} href={rescue} download="running-recording"><Download className="size-4" />Download unsaved audio</a><button className={button} onClick={() => pending.current && void save(pending.current)}>Retry local save</button></div>}
      <p className="text-xs leading-relaxed text-zinc-500">Keep the screen on and stop before pocketing your phone; screen lock can interrupt recording. Clips join the nearest run within 90 minutes.</p>
    </div>
    <div className="fixed inset-x-0 bottom-0 z-20 border-t border-zinc-800 bg-zinc-950/95 px-8 pt-4 pb-[max(1.25rem,env(safe-area-inset-bottom))] backdrop-blur lg:static lg:z-auto lg:flex lg:flex-col lg:items-center lg:justify-center lg:gap-6 lg:border-t-0 lg:border-l lg:bg-zinc-900 lg:p-8">
      <div className="mx-auto flex max-w-sm items-center justify-between lg:contents">
        <label className={`${dockButton} cursor-pointer lg:order-2 ${recording || rescue ? 'pointer-events-none opacity-40' : ''}`} title="Import voice memo">
          <Upload className="size-5" /><span className="text-[10px]">Import</span>
          <input className="sr-only" type="file" accept="audio/*,.m4a,.webm,.wav,.mp3,.ogg,.flac" disabled={recording || !!rescue} onChange={e => { const f = e.target.files?.[0]; if (f) void importFile(f); e.target.value = ''; }} />
        </label>
        <button className="group flex size-20 items-center justify-center rounded-full border-4 border-zinc-200 disabled:opacity-50 lg:order-1 lg:size-28" aria-label={recording ? 'Stop recording' : 'Start recording'} disabled={starting || !!rescue} onClick={() => recording ? recorder.current?.stop() : void start()}>
          <span className={`bg-red-600 transition-all duration-200 group-hover:bg-red-500 ${recording ? 'size-8 rounded-md lg:size-10' : 'size-16 rounded-full lg:size-24'} ${starting ? 'animate-pulse' : ''}`} />
        </button>
        <button className={`${dockButton} lg:order-3`} onClick={sync} title="Retry uploads"><RefreshCw className="size-5" /><span className="text-[10px]">Sync</span></button>
      </div>
    </div>
    <div className="h-32 lg:hidden" aria-hidden />
  </section>;
}
export function RunningClips({ noteId }: { noteId: string }) {
  const [clips, setClips] = useState<Array<{ clipId: string; status: string; transcript: string; error: string; recordedAt: string }>>([]), [error, setError] = useState('');
  useEffect(() => {
    let live = true;
    const load = async () => { try { const response = await fetch(`/api/running/${encodeURIComponent(noteId)}/status`, { cache: 'no-store' }); if (!response.ok) throw new Error('Clip statuses unavailable.'); const data = await response.json(); if (live) { setClips(data.clips); setError(''); } } catch (e) { if (live) setError((e as Error).message); } };
    void load(); const timer = setInterval(load, 5000); return () => { live = false; clearInterval(timer); };
  }, [noteId]);
  return <section className="mt-6 space-y-3" aria-label="Run audio clips"><h3>{clips.length} segments</h3>{error && <p role="alert">{error}</p>}{clips.map(clip => <div key={clip.clipId} className="rounded border border-zinc-700 p-3 space-y-2">
    <p>{new Date(clip.recordedAt).toLocaleString()} · {clip.status}</p><audio className="w-full" controls preload="none" src={`/api/running/clips/${encodeURIComponent(noteId)}/${clip.clipId}/audio`} />
    {clip.error && <p>{clip.error}</p>}<details><summary>Original clip transcript</summary><p className="whitespace-pre-wrap">{clip.transcript || 'Waiting for local transcription.'}</p></details>
    <button className={button} disabled={clip.status === 'transcribing'} onClick={async () => { try { const r = await fetch(`/api/running/clips/${encodeURIComponent(noteId)}/${clip.clipId}/retranscribe`, { method: 'POST', headers: { 'Content-Type': 'application/json' } }); if (!r.ok) throw new Error('Could not queue transcription.'); setClips(current => current.map(c => c.clipId === clip.clipId ? { ...c, status: 'pending' } : c)); } catch (e) { setError((e as Error).message); } }}>Retranscribe</button>
  </div>)}<p className="text-sm text-zinc-400">Retranscribing updates the original clip transcript. Your edited run text is preserved. Reload and reopen the run to read newly appended transcripts.</p></section>;
}
