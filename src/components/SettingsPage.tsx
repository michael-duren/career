import { useEffect, useRef, useState, type KeyboardEvent, type PointerEvent, type ReactNode } from 'react';
import { AuthService, signOut } from '../lib/auth';
import {
  DEFAULT_SETTINGS, MODULE_DESCRIPTIONS, MODULE_IDS, SETTINGS_CHANNEL, SETTINGS_EVENT, appearsIn, enabledModules, isEnabled, loadSettings,
  lockedBy, moduleById, moduleStatus, moveModule, orderedModules, saveSettings, settingsSchema, tabBarHint, tabBarModules, type ModuleId, type Settings,
  type SettingsPatch,
} from '../lib/modules';
import { HatchFill, Label, Switch, buttonClassName } from './ui';
import { ThemePicker } from './ThemePicker';

// Settings (docs/redesign/pages/settings-modules.md; reference/desktop/ARC-Settings.html and
// reference/mobile/M-Modules.html). Desktop: a settings nav, then the section. Phone (< 768px):
// a back link to More instead of the nav. Changes save at once with PATCH /api/settings, and
// saveSettings() announces them so the header and tab bar update without a reload.

export type SettingsSection = 'general' | 'modules' | 'theme' | 'account';

const SECTIONS: { id: SettingsSection; label: string }[] = [
  { id: 'general', label: 'General' },
  { id: 'modules', label: 'Modules' },
  { id: 'theme', label: 'Theme' },
  { id: 'account', label: 'Account' },
];

const title = 'm-0 font-extralight !text-[40px] !leading-[0.9] md:!text-[46px]';
const intro = 'm-0 text-[13px] font-light leading-[1.45] text-ink-60';

export function SettingsPage({ section }: { section: SettingsSection }) {
  return (
    <div className="grid gap-x-12 md:grid-cols-[160px_minmax(0,1fr)] xl:grid-cols-[190px_minmax(0,1fr)]">
      <nav aria-label="Settings" className="hidden flex-col md:flex">
        <Label className="mb-2.5">Settings</Label>
        {SECTIONS.map(s => (
          <a
            key={s.id}
            href={`/settings/${s.id}`}
            aria-current={s.id === section ? 'page' : undefined}
            className={`flex min-h-9 items-center text-sm no-underline ${s.id === section ? 'border-l border-ink pl-2.5 text-ink' : 'pl-[11px] font-light text-ink-60 hover:text-ink'}`}
          >
            {s.label}
          </a>
        ))}
      </nav>
      <div className="min-w-0">
        <a href="/more" className="mb-1 -mt-2 flex min-h-11 items-center self-start font-mono text-[11px] tracking-[0.12em] text-ink-60 no-underline hover:text-ink md:hidden">← MORE</a>
        {section === 'general' && <GeneralSection />}
        {section === 'modules' && <ModulesSection />}
        {section === 'theme' && <ThemeSection />}
        {section === 'account' && <AccountSection />}
      </div>
    </div>
  );
}

function SectionHeading({ label, heading, children }: { label?: ReactNode; heading: string; children?: ReactNode }) {
  return (
    <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between md:gap-8">
      <div className="flex flex-col gap-1.5 md:gap-2">
        {label != null && <Label>{label}</Label>}
        <h1 className={title}>{heading}</h1>
      </div>
      {children}
    </div>
  );
}

// Quiet is drawn at --ink-60 like muted (the reference's 50% would miss contrast at 9.5px).
const statusTone = { accent: 'text-accent', muted: 'text-ink-60', quiet: 'text-ink-60' } as const;

function ModulesSection() {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [loadError, setLoadError] = useState('');
  const [saveError, setSaveError] = useState('');
  const [counts, setCounts] = useState<Record<string, number>>({});
  // Saves run one at a time, in order, so the last change wins on the server too. While any
  // are in flight, announced settings (our own saves echo back) don't overwrite newer local edits.
  const queue = useRef<Promise<unknown>>(Promise.resolve());
  const pending = useRef(0);
  // The latest settings shown, so two changes in one tick build on each other.
  const latest = useRef<Settings | null>(null);
  latest.current = settings;

  useEffect(() => {
    loadSettings().then(setSettings, (e: unknown) => setLoadError(e instanceof Error ? e.message : 'Settings could not be loaded. Try again.'));
    fetch('/api/counts?all=1', { cache: 'no-store' })
      .then(response => response.ok ? response.json() : Promise.reject(new Error(`HTTP ${response.status}`)))
      .then((body: { counts?: Record<string, number> }) => setCounts(body.counts ?? {}))
      .catch((e: unknown) => console.warn('Module counts could not be loaded; paused modules show no count.', e));
    const update = (value: unknown) => {
      const parsed = settingsSchema.safeParse(value);
      if (parsed.success && pending.current === 0) setSettings(parsed.data);
    };
    const onEvent = (event: Event) => update((event as CustomEvent).detail);
    window.addEventListener(SETTINGS_EVENT, onEvent);
    const channel = typeof BroadcastChannel === 'function' ? new BroadcastChannel(SETTINGS_CHANNEL) : null;
    channel?.addEventListener('message', event => update(event.data));
    return () => { window.removeEventListener(SETTINGS_EVENT, onEvent); channel?.close(); };
  }, []);

  function save(next: Settings, patch: SettingsPatch) {
    latest.current = next;
    setSettings(next);
    setSaveError('');
    pending.current++;
    const run = queue.current.then(() => saveSettings(patch));
    queue.current = run.catch(() => undefined);
    run.then(saved => {
      pending.current--;
      if (pending.current === 0) setSettings(saved);
    }, async (e: unknown) => {
      pending.current--;
      setSaveError(e instanceof Error ? e.message : 'Settings could not be saved. Try again.');
      // Show what is really saved again.
      if (pending.current === 0) {
        await loadSettings().then(setSettings, (reload: unknown) => {
          console.warn('Saved settings could not be reloaded after a failed save.', reload);
          setSaveError(`${e instanceof Error ? e.message : 'Settings could not be saved.'} Reload the page to see what is saved.`);
        });
      }
    });
  }

  function toggle(id: ModuleId) {
    const base = latest.current;
    if (!base) return;
    const on = base.modules[id] === false;
    save({ ...base, modules: { ...base.modules, [id]: on } }, { modules: { [id]: on } });
  }

  function reorder(order: ModuleId[]) {
    const base = latest.current;
    if (!base) return;
    save({ ...base, moduleOrder: order }, { moduleOrder: order });
  }

  if (loadError) return <><SectionHeading heading="Modules" /><p role="alert" className="mt-4 text-sm text-accent">{loadError}</p></>;
  const current = settings ?? DEFAULT_SETTINGS;
  const onCount = enabledModules(current).length;

  return (
    <div className="grid gap-x-12 gap-y-10 xl:grid-cols-[minmax(0,1fr)_300px]" aria-busy={!settings}>
      <section aria-labelledby="modules-heading" className="min-w-0">
        <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between md:gap-8">
          <div className="flex flex-col gap-1.5 md:gap-2">
            <Label>{settings ? `${onCount} of ${MODULE_IDS.length} on` : ' '}</Label>
            <h1 id="modules-heading" className={title}>Modules</h1>
          </div>
          <p className={`${intro} md:max-w-[300px] md:text-right`}>
            <span className="md:hidden">Off hides a module everywhere. Your data is kept.</span>
            <span className="hidden md:inline">Turning a module off hides it everywhere. Its data is kept, and comes back when you turn it on.</span>
          </p>
        </div>
        {saveError && <p role="alert" className="mt-4 text-sm text-accent">{saveError}</p>}
        <div aria-hidden="true" className="ui-label mt-5 hidden grid-cols-[52px_minmax(0,1fr)_200px] gap-[18px] border-b border-ink pb-1.5 md:grid xl:grid-cols-[52px_minmax(0,1fr)_260px]">
          <span /><span>Module</span><span>Appears in</span>
        </div>
        <ul className="m-0 mt-2.5 list-none p-0 md:mt-0">
          {settings && MODULE_IDS.map(id => {
            const m = moduleById(id);
            const on = isEnabled(settings, id);
            const locked = lockedBy(settings, id).length > 0;
            const status = moduleStatus(settings, id, counts);
            const nameId = `module-${id}-name`;
            const descId = `module-${id}-desc`;
            return (
              <li key={id} data-module-id={id} className={`grid min-h-16 grid-cols-[minmax(0,1fr)_52px] items-center gap-3.5 border-b border-ink-14 md:min-h-[50px] md:grid-cols-[52px_minmax(0,1fr)_200px] md:gap-[18px] xl:grid-cols-[52px_minmax(0,1fr)_260px] ${on ? '' : 'md:opacity-75'}`}>
                <Switch checked={on} locked={locked} onCheckedChange={() => toggle(id)} aria-labelledby={nameId} aria-describedby={descId} className="order-2 justify-self-end md:order-none md:justify-self-start" />
                <span className="flex min-w-0 flex-col gap-[3px] md:gap-0.5">
                  {/* Phone: name, description, status. Desktop: the status sits after the name. */}
                  <span className="contents md:flex md:flex-wrap md:items-baseline md:gap-x-2.5">
                    <span id={nameId} className="text-base font-light">{m.label}</span>
                    {status && <span className={`order-last font-mono text-[9.5px] tracking-[0.1em] uppercase md:order-none md:whitespace-nowrap ${statusTone[status.tone]}`}>{status.text}</span>}
                  </span>
                  <span id={descId} className="text-xs leading-[1.35] font-light text-ink-60 xl:truncate">{MODULE_DESCRIPTIONS[id]}{status ? <span className="ui-sr-only">. {status.text}</span> : null}</span>
                </span>
                <span className="hidden flex-wrap gap-1.5 md:flex" aria-label={`${m.label} appears in`}>
                  {appearsIn(m).map(place => (
                    <span key={place} className={`border px-[7px] py-[3px] font-mono text-[9.5px] tracking-[0.1em] ${on ? 'border-ink-50' : 'border-ink-14 text-ink-50 line-through'}`}>
                      {on ? place : <><span className="ui-sr-only">not in </span>{place}</>}
                    </span>
                  ))}
                </span>
              </li>
            );
          })}
        </ul>
      </section>
      {settings && <TabBarSection settings={settings} onReorder={reorder} />}
    </div>
  );
}

function TabBarSection({ settings, onReorder }: { settings: Settings; onReorder: (order: ModuleId[]) => void }) {
  const tabs = ['HOME', ...tabBarModules(settings).map(m => m.short), 'MORE'];
  const hint = tabBarHint(settings);
  return (
    <aside aria-labelledby="tab-bar-heading" className="flex min-w-0 flex-col gap-4 md:max-w-[420px] xl:max-w-none">
      <h2 id="tab-bar-heading" className="m-0 text-xs font-medium tracking-[0.24em]">
        <span className="md:hidden">TAB BAR PREVIEW</span><span className="hidden md:inline">PHONE TAB BAR</span>
      </h2>
      <p className={intro}>Home and More stay put. The middle three fill from your enabled modules in this order.</p>
      <ol aria-label="Phone tab bar preview" className="m-0 grid h-[58px] list-none grid-cols-5 border border-ink-50 p-0 md:h-[62px]">
        {tabs.map((label, i) => {
          const frame = i === 0 || i === tabs.length - 1;
          return (
            <li key={`${label}-${i}`} className={`flex min-w-0 flex-col items-center justify-center gap-1.5 font-mono text-[8.5px] tracking-[0.08em] uppercase ${frame ? 'text-ink-60' : ''} ${i > 0 ? 'border-l border-ink-14' : ''}`}>
              {frame ? <span aria-hidden="true" className="size-3.5 border border-ink" /> : <HatchFill className="size-3.5 border border-ink" />}
              {label}
            </li>
          );
        })}
      </ol>
      <OrderList settings={settings} onReorder={onReorder} />
      {hint && <p className="m-0 font-hand text-sm text-accent">{hint}</p>}
    </aside>
  );
}

// The module order, rearranged by dragging a row's handle (mouse, pen or touch) or with the
// arrow keys on a focused handle. Turned-off modules keep their place, struck through.
function OrderList({ settings, onReorder }: { settings: Settings; onReorder: (order: ModuleId[]) => void }) {
  const order = orderedModules(settings).map(m => m.id);
  const [drag, setDrag] = useState<{ id: ModuleId; order: ModuleId[] } | null>(null);
  const [announcement, setAnnouncement] = useState('');
  const list = useRef<HTMLOListElement>(null);
  const shown = drag?.order ?? order;
  // A keyboard move re-renders the list; keep focus on the moved module's handle.
  const focusId = useRef<ModuleId | null>(null);
  useEffect(() => {
    if (!focusId.current) return;
    const handle = list.current?.querySelector<HTMLButtonElement>(`[data-module-id="${focusId.current}"] button`);
    focusId.current = null;
    if (handle && document.activeElement !== handle) handle.focus();
  });

  function announce(next: ModuleId[], id: ModuleId) {
    setAnnouncement(`${moduleById(id).label} moved to position ${next.indexOf(id) + 1} of ${next.length}.`);
  }

  function onKeyDown(event: KeyboardEvent<HTMLButtonElement>, id: ModuleId) {
    const from = order.indexOf(id);
    const to = { ArrowUp: from - 1, ArrowDown: from + 1, Home: 0, End: order.length - 1 }[event.key];
    if (to === undefined) return;
    event.preventDefault();
    if (to < 0 || to >= order.length || to === from) return;
    const next = moveModule(order, from, to);
    focusId.current = id;
    onReorder(next);
    announce(next, id);
  }

  function onPointerDown(event: PointerEvent<HTMLButtonElement>, id: ModuleId) {
    if (event.button !== 0) return;
    event.currentTarget.setPointerCapture(event.pointerId);
    setDrag({ id, order });
  }

  function onPointerMove(event: PointerEvent<HTMLButtonElement>) {
    if (!drag || !list.current) return;
    const rows = [...list.current.children] as HTMLElement[];
    // The row under the pointer's height takes the dragged module.
    let to = rows.findIndex(row => event.clientY < row.getBoundingClientRect().bottom);
    if (to === -1) to = rows.length - 1;
    const from = drag.order.indexOf(drag.id);
    if (to !== from) setDrag({ id: drag.id, order: moveModule(drag.order, from, to) });
  }

  function onPointerUp() {
    if (!drag) return;
    const { id, order: next } = drag;
    setDrag(null);
    if (next.join() !== order.join()) {
      onReorder(next);
      announce(next, id);
    }
  }

  return (
    <>
      <ol ref={list} aria-label="Module order" className="m-0 mt-1 flex list-none flex-col p-0">
        {shown.map((id, i) => {
          const m = moduleById(id);
          const on = isEnabled(settings, id);
          return (
            <li key={id} data-module-id={id} className={`flex min-h-11 items-center gap-3 border-b border-ink-14 text-sm font-light ${drag?.id === id ? 'bg-ink-05' : ''}`}>
              <span aria-hidden="true" className="font-mono text-[10px] text-ink-60">{String(i + 1).padStart(2, '0')}</span>
              <span className={on ? '' : 'text-ink-50 line-through'}>{m.label}{on ? null : <span className="ui-sr-only"> (off)</span>}</span>
              <button
                type="button"
                aria-label={`Move ${m.label}, position ${i + 1} of ${shown.length}`}
                aria-describedby="module-order-help"
                onKeyDown={event => onKeyDown(event, id)}
                onPointerDown={event => onPointerDown(event, id)}
                onPointerMove={onPointerMove}
                onPointerUp={onPointerUp}
                onPointerCancel={() => setDrag(null)}
                onLostPointerCapture={() => setDrag(null)}
                className="ml-auto flex min-h-11 min-w-11 cursor-grab touch-none items-center justify-center border-0 bg-transparent p-0 font-mono text-[11px] text-ink-50 select-none hover:text-ink active:cursor-grabbing"
              >
                <span aria-hidden="true">⋮⋮</span>
              </button>
            </li>
          );
        })}
      </ol>
      <p id="module-order-help" className="ui-sr-only">Drag, or use the up and down arrow keys, to move a module. The first three turned-on modules fill the phone tab bar.</p>
      <p aria-live="polite" className="ui-sr-only">{announcement}</p>
    </>
  );
}

// General holds only "run setup again" for now (docs/redesign/pages/setup.md).
function GeneralSection() {
  return (
    <section className="flex flex-col">
      <SectionHeading heading="General" />
      <div className="mt-[30px] flex flex-col items-start gap-3 border-t border-ink-14 pt-5">
        <h2 className="m-0 text-xs font-medium tracking-[0.24em]">FIRST-RUN SETUP</h2>
        <p className={intro}>Walk through focus, modules, connections, theme and a first goal again. Your data is kept; Finish saves the modules you pick.</p>
        <a href="/setup" className={buttonClassName()}>RUN SETUP AGAIN</a>
      </div>
    </section>
  );
}

function ThemeSection() {
  return (
    <section className="flex flex-col">
      <SectionHeading heading="Theme">
        <p className={`${intro} md:max-w-[300px] md:text-right`}>Same drawings, different paper and ink. Change it any time.</p>
      </SectionHeading>
      <ThemePicker size="large" className="mt-[30px]" />
    </section>
  );
}

function AccountSection() {
  const [username] = useState(() => { try { return AuthService.getUsername(); } catch { return null; } });
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  async function onSignOut() {
    setError('');
    setBusy(true);
    if (!await signOut()) {
      setBusy(false);
      setError('Sign out failed. Check your connection and try again.');
    }
  }

  return (
    <section className="flex flex-col">
      <SectionHeading heading="Account" label={username ? `Signed in as ${username}` : undefined} />
      <div className="mt-[30px] flex flex-col items-start gap-3 border-t border-ink-14 pt-5">
        <p className={intro}>Signing out ends this session on this device. Your data stays saved.</p>
        <button type="button" className="ui-btn" disabled={busy} onClick={() => void onSignOut()}>SIGN OUT</button>
        {error && <p role="alert" className="m-0 text-sm text-accent">{error}</p>}
      </div>
    </section>
  );
}
