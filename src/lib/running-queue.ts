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
const queuedClip = (id: string) => transaction<QueuedClip | undefined>('readonly', s => s.get(id));
/**
 * Unsent takes that would land in a thought once uploaded: those addressed to it,
 * and unaddressed ones the server would group into it. The server groups each take
 * with the nearest clip in the window, including takes from the same backlog, so a
 * matched take pulls in further takes within the window of it.
 */
export function takesForThought(queued: QueuedClip[], noteId: string, clipTimes: string[], windowMs: number): QueuedClip[] {
  const times = clipTimes.map(t => Date.parse(t)).filter(t => !Number.isNaN(t));
  const matched = new Set(queued.filter(clip => clip.noteId === noteId));
  for (let grew = true; grew;) {
    grew = false;
    for (const clip of queued) {
      const at = Date.parse(clip.recordedAt);
      if (clip.noteId || matched.has(clip) || Number.isNaN(at) || !times.some(t => Math.abs(at - t) <= windowMs)) continue;
      matched.add(clip); times.push(at); grew = true;
    }
  }
  return queued.filter(clip => matched.has(clip));
}
let syncing: Promise<void> | undefined;
/** Settles once any upload in progress has finished, successfully or not. */
export const syncSettled = () => (syncing ?? Promise.resolve()).catch(() => {});
export function syncClips(onChange: () => void): Promise<void> {
  if (syncing) return syncing;
  syncing = (async () => {
    const snapshot = (await queuedClips()).sort((a, b) => a.recordedAt.localeCompare(b.recordedAt));
    for (const { clientId } of snapshot) {
      // Re-read each take: deleting its thought may have discarded it since the snapshot.
      const clip = await queuedClip(clientId);
      if (!clip) continue;
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
