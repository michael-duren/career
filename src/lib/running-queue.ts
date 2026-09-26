export type QueuedClip = { clientId: string; audio: Blob; recordedAt: string; durationMs: number; noteId?: string };
// Keep the audio in durable browser storage until the server acknowledges its UUID.
function database(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open('career-running-clips-v1', 1);
    request.onupgradeneeded = () => request.result.createObjectStore('clips', { keyPath: 'clientId' });
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}
async function transaction<T>(mode: IDBTransactionMode, operation: (store: IDBObjectStore) => IDBRequest<T>): Promise<T> {
  const db = await database();
  return new Promise((resolve, reject) => {
    const tx = db.transaction('clips', mode); const req = operation(tx.objectStore('clips'));
    tx.oncomplete = () => { db.close(); resolve(req.result); };
    tx.onerror = tx.onabort = () => { db.close(); reject(tx.error ?? req.error ?? new Error('Local audio storage failed')); };
  });
}
export const enqueueClip = (clip: QueuedClip) => transaction('readwrite', s => s.put(clip));
export const queuedClips = () => transaction<QueuedClip[]>('readonly', s => s.getAll());
export const removeClip = (id: string) => transaction('readwrite', s => s.delete(id));
/**
 * Unsent takes that would land in a thought once uploaded: those addressed to it,
 * and unaddressed ones recorded within the server's grouping window of its clips.
 */
export function takesForThought(queued: QueuedClip[], noteId: string, clipTimes: string[], windowMs: number): QueuedClip[] {
  const times = clipTimes.map(t => Date.parse(t)).filter(t => !Number.isNaN(t));
  return queued.filter(clip => {
    if (clip.noteId) return clip.noteId === noteId;
    const at = Date.parse(clip.recordedAt);
    return !Number.isNaN(at) && times.some(t => Math.abs(at - t) <= windowMs);
  });
}
let syncing: Promise<void> | undefined;
export function syncClips(onChange: () => void): Promise<void> {
  if (syncing) return syncing;
  syncing = (async () => {
    const clips = (await queuedClips()).sort((a, b) => a.recordedAt.localeCompare(b.recordedAt));
    for (const clip of clips) {
      const form = new FormData(); form.set('audio', clip.audio, 'recording');
      form.set('clientId', clip.clientId); form.set('recordedAt', clip.recordedAt); form.set('durationMs', String(clip.durationMs));
      if (clip.noteId) form.set('noteId', clip.noteId);
      const response = await fetch('/api/running/clips', { method: 'POST', body: form, redirect: 'error', signal: AbortSignal.timeout(180000) });
      if (!response.ok) throw new Error(response.status === 401 ? 'Sign in to upload saved clips.' : (await response.json()).error || 'Upload failed; audio remains on this device.');
      const acknowledgement = await response.json();
      if (acknowledgement.clipId !== clip.clientId || typeof acknowledgement.noteId !== 'string' || !acknowledgement.noteId) throw new Error('Upload acknowledgement invalid; audio remains on this device.');
      await removeClip(clip.clientId); onChange();
      window.dispatchEvent(new Event('workspace-saved'));
    }
  })().finally(() => { syncing = undefined; });
  return syncing;
}
