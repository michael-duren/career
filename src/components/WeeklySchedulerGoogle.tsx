import { useCallback, useEffect, useRef, useState } from 'react';
import { z } from 'zod';
import { SchedulerError, schedulerRequest } from '../lib/scheduler';
const statusSchema = z.object({ configured: z.boolean(), connected: z.boolean(), reconnectRequired: z.boolean(), lastRefresh: z.string().nullish(), error: z.string().optional(), revision: z.string(), calendarId: z.string().optional() });
const calendarsSchema = z.object({ calendars: z.array(z.object({ id: z.string(), summary: z.string(), selected: z.boolean() })).nullable().transform(v => v ?? []), revision: z.string() });
export function WeeklySchedulerGoogle({ week, refreshWeek, refreshGeneration }: { week: string; refreshWeek: () => void; refreshGeneration: number }) {
  const [status, setStatus] = useState<z.infer<typeof statusSchema> | null>(null);
  const [calendars, setCalendars] = useState<z.infer<typeof calendarsSchema> | null>(null);
  const [error, setError] = useState('');
  const [statusError, setStatusError] = useState('');
  const [working, setWorking] = useState(false);
  const pending = useRef(false);
  const statusSequence = useRef(0);
  const readStatus = useCallback(async () => {
    const sequence = ++statusSequence.current;
    try {
      const next = statusSchema.parse(await schedulerRequest('google/status'));
      if (sequence === statusSequence.current) { setStatus(next); setStatusError(''); }
    } catch (e) {
      if (sequence === statusSequence.current) setStatusError(e instanceof Error ? e.message : 'Could not read Google status.');
      throw e;
    }
  }, []);
  useEffect(() => { void readStatus().catch(() => {}); }, [readStatus, refreshGeneration]);
  const readCalendars = async (preserve = false) => {
    const next = calendarsSchema.parse(await schedulerRequest('google/calendars'));
    setCalendars(current => preserve && current ? { ...next, calendars: next.calendars.map(calendar => ({ ...calendar, selected: current.calendars.find(c => c.id === calendar.id)?.selected ?? calendar.selected })) } : next);
  };
  const perform = async (operation: () => Promise<void>, selection = false) => {
    if (pending.current) return;
    pending.current = true;
    statusSequence.current++;
    setWorking(true); setError('');
    try { await operation(); await readStatus(); }
    catch (e) {
      if (e instanceof SchedulerError) {
        try { await readStatus(); if (selection) await readCalendars(true); }
        catch { /* Keep the proposal if recovery is temporarily unavailable. */ }
      }
      setError(e instanceof Error ? e.message : 'Google request failed.');
    } finally { pending.current = false; setWorking(false); }
  };
  useEffect(() => {
    if (!status?.connected || status.reconnectRequired) return;
    const timer = window.setInterval(() => {
      if (document.visibilityState === 'visible') void perform(async () => { await schedulerRequest('google/refresh', { week }); refreshWeek(); });
    }, 300000);
    return () => clearInterval(timer);
  }, [week, status?.connected, status?.reconnectRequired, refreshWeek, readStatus]);
  return <details className="scheduler-google"><summary>Google Calendar · {status?.reconnectRequired ? 'Reconnect needed' : status?.connected ? 'Connected' : 'Not connected'}</summary><p>Accepted plans sync to a dedicated calendar. Changes made in Google are restored from this app; actual-time corrections stay here. Disconnecting leaves exported events in Google.</p>{status?.lastRefresh && Date.parse(status.lastRefresh) > 0 && <p>Availability refreshed: {new Date(status.lastRefresh).toLocaleString()}</p>}{(error || statusError || status?.error) && <p role="alert">{error || statusError || status?.error}</p>}{status && !status.configured && <p>Google connection is not configured on this server. Local scheduling is available.</p>}<div className="scheduler-actions">{status?.configured && (!status.connected || status.reconnectRequired) && <button disabled={working} onClick={() => void perform(async () => { const result = z.object({ url: z.string().url() }).parse(await schedulerRequest('google/connect', {})); window.location.assign(result.url); })}>{status.reconnectRequired ? 'Reconnect Google' : 'Connect Google'}</button>}{status?.connected && <><button disabled={working || status.reconnectRequired} onClick={() => void perform(async () => { await schedulerRequest('google/refresh', { week }); refreshWeek(); })}>Refresh availability</button><button disabled={working} onClick={() => void perform(async () => readCalendars())}>Select busy calendars</button><button disabled={working} onClick={() => void perform(async () => { await schedulerRequest('google/disconnect', { revision: status.revision }); setCalendars(null); refreshWeek(); })}>Disconnect</button></>}</div>{calendars && <form onSubmit={e => { e.preventDefault(); void perform(async () => { await schedulerRequest('google/calendars', { revision: calendars.revision, calendarIds: calendars.calendars.filter(c => c.selected).map(c => c.id) }); setCalendars(null); refreshWeek(); }, true); }}>{calendars.calendars.map(calendar => <label className="scheduler-check" key={calendar.id}><input disabled={working} type="checkbox" checked={calendar.selected} onChange={e => setCalendars({ ...calendars, calendars: calendars.calendars.map(c => c.id === calendar.id ? { ...c, selected: e.target.checked } : c) })} />{calendar.summary}</label>)}<button disabled={working}>Save calendar selection</button></form>}</details>;
}
