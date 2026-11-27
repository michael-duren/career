import { useEffect, useRef, useState } from 'react';
import { MODULE_DESCRIPTIONS, MODULE_IDS, FOCUS_SUGGESTIONS, loadSettings, lockedBy, moduleById, saveSettings, type Settings } from '../lib/modules';
import {
  FOCUS_CARDS, SETUP_STEPS, connectionsFor, effectiveModules, finishPatch, goalDraftError, goalFromDraft, localDate, mcpURL, skipPatch,
  suggestedModules, switchesFrom, type GoalDraft, type ModuleSwitches, type SetupConnection,
} from '../lib/setup';
import { applyTheme } from '../lib/theme';
import { cn } from '../lib/utils';
import { HatchFill, Label, MeasurementLine, Switch } from './ui';
import { buttonClassName } from '../lib/ui';
import { ThemePicker } from './ThemePicker';

// First-run setup (docs/redesign/pages/setup.md; reference/setup/ARC-Setup-1..5.html and
// M-Setup-1..5.html): focus → modules → connect → theme → first goal. New users land here after
// login; Settings → General runs it again. Nothing but the theme (step 4 applies and saves it at
// once) is saved before Finish, which saves the modules and marks setup done. Skip saves every
// module on, the dark theme, and marks setup done.

const LAST = SETUP_STEPS.length - 1;
// ! beats the unlayered phone h1 rule in global.css (max-width: 639px).
const title = 'm-0 font-extralight !text-[32px] !leading-[1.05] md:!text-[50px] md:!leading-none';
const intro = 'm-0 mt-2 text-sm font-light text-ink-60 md:mt-2.5 md:text-base';
const errorText = (e: unknown, fallback: string) => (e instanceof Error && e.message ? e.message : fallback);

export function SetupWizard() {
  const [step, setStep] = useState(0);
  const [focus, setFocus] = useState<string[]>([]);
  // The module switches once the user (or a re-run's saved settings) set them; null follows the focus picks.
  const [switches, setSwitches] = useState<ModuleSwitches | null>(null);
  const [goal, setGoal] = useState<GoalDraft>(() => ({ title: '', start: localDate(new Date()), end: '' }));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [loadError, setLoadError] = useState('');
  // One id per draft, so retrying Finish after a failed response saves the same goal again.
  const goalId = useRef(crypto.randomUUID());
  const heading = useRef<HTMLHeadingElement>(null);
  const firstRender = useRef(true);

  useEffect(() => {
    // A re-run starts from what is saved; a new install from the focus picks.
    loadSettings().then((saved: Settings) => {
      if (saved.setupCompletedAt) setSwitches(current => current ?? switchesFrom(saved.modules));
    }, (e: unknown) => setLoadError(errorText(e, 'Settings could not be loaded. Try again.')));
  }, []);

  // Each step moves focus to its heading, so keyboard and screen-reader users start at the top.
  useEffect(() => {
    if (firstRender.current) { firstRender.current = false; return; }
    heading.current?.focus();
    window.scrollTo({ top: 0 });
  }, [step]);

  const suggested = suggestedModules(focus);
  const on = switches ?? suggested;
  const effective = effectiveModules(on);
  const onIds = MODULE_IDS.filter(id => effective[id]);

  function go(next: number) {
    setError('');
    setStep(Math.max(0, Math.min(LAST, next)));
  }

  function toggleFocus(name: string) {
    setFocus(current => current.includes(name) ? current.filter(f => f !== name) : [...current, name]);
    // New picks bring new suggestions.
    setSwitches(null);
  }

  async function run(action: () => Promise<void>) {
    if (busy) return;
    setBusy(true);
    setError('');
    try {
      await action();
    } catch (e) {
      setError(errorText(e, 'Setup could not be saved. Try again.'));
      setBusy(false);
    }
  }

  const skip = () => run(async () => {
    await saveSettings(skipPatch(new Date()));
    applyTheme('dark');
    window.location.assign('/');
  });

  const finish = () => run(async () => {
    const problem = effective.timeline ? goalDraftError(goal) : '';
    if (problem) throw new Error(problem);
    // Modules first: the goal API needs Timeline on.
    await saveSettings(finishPatch(on, new Date()));
    const draft = effective.timeline ? goalFromDraft(goal, goalId.current, new Date()) : null;
    if (draft) {
      const response = await fetch('/api/goals', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ goal: draft, revision: null }) })
        .catch(() => { throw new Error('Setup is saved, but the goal could not be added. Check your connection and finish again.'); });
      const body: unknown = await response.json().catch(() => null);
      if (!response.ok) {
        const reason = body && typeof body === 'object' && 'error' in body && typeof body.error === 'string' ? body.error : `HTTP ${response.status}`;
        throw new Error(`Setup is saved, but the goal could not be added: ${reason}`);
      }
    }
    window.location.assign('/');
  });

  return (
    <div className="flex min-h-dvh flex-col bg-paper text-ink">
      <div className="mx-auto flex w-full max-w-[1280px] flex-1 flex-col px-[22px] pt-[22px] md:px-14 md:pt-10 xl:px-[120px]">
        <header className="flex h-7 items-center justify-between gap-4">
          <span className="text-[13px] font-medium tracking-[0.32em]">CAREER</span>
          <span className="ui-label hidden md:inline">Setup · step {step + 1} of {SETUP_STEPS.length}</span>
          <button type="button" disabled={busy} onClick={() => void skip()}
            className="-mr-2 inline-flex min-h-11 items-center border-0 bg-transparent px-2 font-mono text-[11px] tracking-[0.12em] text-ink-60 hover:text-ink disabled:opacity-50">
            SKIP SETUP
          </button>
        </header>
        <StepIndicator step={step} />

        <main className="mt-6 flex flex-1 flex-col pb-8 md:mt-[34px]">
          {loadError && <p role="alert" className="m-0 mb-4 text-sm text-accent">{loadError}</p>}
          {step === 0 && (
            <section aria-labelledby="setup-heading">
              <h1 id="setup-heading" ref={heading} tabIndex={-1} className={title}>What are you working on?</h1>
              <p className={intro}>Pick any that fit. We'll turn on the right modules, and you can change them later.</p>
              <fieldset className="m-0 mt-[18px] grid min-w-0 grid-cols-1 gap-2.5 border-0 p-0 md:mt-[30px] md:grid-cols-2 md:gap-[18px] xl:grid-cols-3">
                <legend className="ui-sr-only">Focus</legend>
                {FOCUS_CARDS.map(card => {
                  const picked = focus.includes(card.name);
                  return (
                    <label key={card.name} className={cn('relative flex cursor-pointer flex-col gap-1.5 border px-[18px] py-4 outline-offset-2 outline-ink has-focus-visible:outline-2',
                      picked ? 'border-ink bg-ink-05' : 'border-ink-30 hover:border-ink-60')}>
                      <input type="checkbox" checked={picked} onChange={() => toggleFocus(card.name)} className="ui-sr-only" />
                      {picked ? <HatchFill className="absolute top-3.5 right-3.5 size-4 border border-ink" /> : <span aria-hidden="true" className="absolute top-3.5 right-3.5 size-4 border border-ink" />}
                      <span className="pr-[26px] text-lg font-light md:text-[21px]">{card.name}</span>
                      <span className="text-[13px] font-light text-ink-60">{card.description}</span>
                      <span className="ui-label mt-1.5 !text-[9px]">{FOCUS_SUGGESTIONS[card.name].map(id => moduleById(id).label).join(' · ')}</span>
                    </label>
                  );
                })}
              </fieldset>
            </section>
          )}

          {step === 1 && (
            <section aria-labelledby="setup-heading">
              <div className="flex items-end justify-between gap-4">
                <h1 id="setup-heading" ref={heading} tabIndex={-1} className={title}>Your modules</h1>
                <Label className="shrink-0">{onIds.length} of {MODULE_IDS.length} on</Label>
              </div>
              <p className={intro}>{focus.length ? 'Suggested from what you picked. ' : ''}Anything off stays out of your menus until you turn it on in Settings.</p>
              <ul className="m-0 mt-3 grid list-none grid-cols-1 p-0 md:mt-[22px] md:grid-cols-2 md:gap-x-14">
                {MODULE_IDS.map(id => {
                  const m = moduleById(id);
                  const needs = lockedBy({ modules: on }, id);
                  const tag = needs.length ? `needs ${needs.join(' and ')}` : focus.length && suggested[id] ? 'suggested' : '';
                  return (
                    <li key={id} data-module-id={id} className="flex min-h-[58px] items-center gap-4 border-b border-ink-14">
                      <Switch checked={effective[id]} locked={needs.length > 0} aria-labelledby={`setup-${id}-name`} aria-describedby={`setup-${id}-desc`}
                        onCheckedChange={() => setSwitches({ ...on, [id]: !on[id] })} className="shrink-0" />
                      <span className="flex min-w-0 flex-1 flex-col gap-0.5 py-2">
                        <span className="flex flex-wrap items-baseline gap-x-2.5">
                          <span id={`setup-${id}-name`} className="text-base font-light">{m.label}</span>
                          {tag && <span className={cn('font-mono text-[9px] tracking-[0.1em] uppercase', needs.length ? 'text-accent' : 'text-ink-60')}>{tag}</span>}
                        </span>
                        <span id={`setup-${id}-desc`} className="text-xs font-light text-ink-60">{MODULE_DESCRIPTIONS[id]}{tag ? <span className="ui-sr-only">. {tag}</span> : null}</span>
                      </span>
                    </li>
                  );
                })}
              </ul>
            </section>
          )}

          {step === 2 && (
            <section aria-labelledby="setup-heading">
              <h1 id="setup-heading" ref={heading} tabIndex={-1} className={title}>Connect your tools</h1>
              <p className={intro}>Optional. Only what your modules use is shown. Skip any of these for now.</p>
              <ConnectList connections={connectionsFor(effective)} />
            </section>
          )}

          {step === 3 && (
            <section aria-labelledby="setup-heading">
              <h1 id="setup-heading" ref={heading} tabIndex={-1} className={title}>Pick a look</h1>
              <p className={intro}>Same drawings, different paper and ink. Change it any time.</p>
              <ThemePicker size="large" className="mt-5 md:mt-[30px]" />
            </section>
          )}

          {step === 4 && (
            <section aria-labelledby="setup-heading">
              <h1 id="setup-heading" ref={heading} tabIndex={-1} className={title}>Add your first goal</h1>
              {effective.timeline ? (
                <>
                  <p className={intro}>Something with an end date. You can break it into steps on the Timeline. Leave it blank to skip.</p>
                  <div className="mt-[26px] grid grid-cols-2 gap-x-4 gap-y-[22px] md:mt-10 md:max-w-[900px] md:grid-cols-[minmax(0,1fr)_200px_200px] md:gap-[30px]">
                    <label className="ui-field col-span-2 md:col-span-1">
                      <span className="ui-label">Goal</span>
                      <input className="ui-field__input lg:!text-[22px]" type="text" maxLength={200} placeholder="e.g. Finish NeetCode 150" value={goal.title} onChange={e => setGoal({ ...goal, title: e.target.value })} />
                    </label>
                    <label className="ui-field">
                      <span className="ui-label">Start</span>
                      <input className="ui-field__input lg:!text-[22px]" type="date" value={goal.start} onChange={e => setGoal({ ...goal, start: e.target.value })} />
                    </label>
                    <label className="ui-field">
                      <span className="ui-label">End</span>
                      <input className="ui-field__input lg:!text-[22px]" type="date" min={goal.start || undefined} value={goal.end} onChange={e => setGoal({ ...goal, end: e.target.value })} />
                    </label>
                  </div>
                </>
              ) : (
                <p className={intro}>Goals live on the Timeline, which you turned off. Turn it on in Settings → Modules when you want one.</p>
              )}
              <div className="mt-11 flex flex-col gap-1.5">
                <Label>Modules on</Label>
                <span className="text-[15px] font-light">{onIds.length ? onIds.map(id => moduleById(id).label).join(' · ') : 'None yet. Turn some on in Settings → Modules.'}</span>
              </div>
            </section>
          )}

          {error && <p role="alert" className="m-0 mt-6 text-sm text-accent">{error}</p>}
        </main>

        <footer className="sticky bottom-0 -mx-[22px] flex items-center justify-between border-t border-ink-30 bg-paper px-[22px] pt-[18px] pb-[max(24px,env(safe-area-inset-bottom))] md:static md:mx-0 md:px-0 md:pb-10">
          {step > 0 && <button type="button" className={buttonClassName()} disabled={busy} onClick={() => go(step - 1)}>← BACK</button>}
          <span className="ml-auto flex gap-2.5">
            {step < LAST
              ? <button type="button" className={buttonClassName({ variant: 'primary' })} disabled={busy} onClick={() => go(step + 1)}>CONTINUE →</button>
              : <button type="button" className={buttonClassName({ variant: 'primary' })} disabled={busy} onClick={() => void finish()}>FINISH<span className="hidden md:inline">&nbsp;· OPEN HOME</span> →</button>}
          </span>
        </footer>
      </div>
    </div>
  );
}

// The step indicator: a measurement line with a dot per step (done filled, current hatched) and labels.
function StepIndicator({ step }: { step: number }) {
  return (
    <ol aria-label="Setup steps" className="relative m-0 mx-2 mt-[22px] h-11 list-none p-0 md:mx-1.5 md:mt-[26px]">
      <MeasurementLine className="absolute! top-2.5 right-0 left-0" />
      {SETUP_STEPS.map((s, i) => {
        const state = i < step ? 'done' : i === step ? 'current' : 'todo';
        const dot = 'absolute top-1 -left-1.5 box-border size-3 rounded-full border border-ink';
        return (
          <li key={s.label} aria-current={state === 'current' ? 'step' : undefined} className="absolute top-0 h-full w-0" style={{ left: `${(i / LAST) * 100}%` }}>
            {state === 'current' ? <HatchFill className={cn(dot, 'bg-paper')} /> : <span aria-hidden="true" className={cn(dot, state === 'done' ? 'bg-ink' : 'bg-paper')} />}
            <span className={cn('absolute top-6 font-mono text-[9.5px] tracking-[0.1em] whitespace-nowrap uppercase', i === 0 ? 'left-0' : i === LAST ? 'right-0' : 'left-0 -translate-x-1/2', state === 'current' ? 'text-ink' : 'text-ink-60')}>
              <span className="md:hidden">{s.short}</span><span className="hidden md:inline">{s.label}</span>
              {state !== 'current' && <span className="ui-sr-only">{state === 'done' ? ' (done)' : ' (to do)'}</span>}
            </span>
          </li>
        );
      })}
    </ol>
  );
}

function ConnectList({ connections }: { connections: SetupConnection[] }) {
  const [copied, setCopied] = useState('');
  const [copyError, setCopyError] = useState('');

  async function copy() {
    const url = mcpURL(window.location.origin);
    setCopyError('');
    try {
      await navigator.clipboard.writeText(url);
      setCopied(url);
    } catch {
      setCopyError(`Copy failed. The MCP URL is ${url}`);
    }
  }

  if (!connections.length) return <p className="m-0 mt-6 py-5 text-[15px] font-light text-ink-60">Nothing to connect for the modules you picked.</p>;
  const cta = 'h-10 px-3 text-[10px] md:h-11 md:w-full md:px-5 md:text-[11px] row-span-2 md:row-span-1';
  return (
    <>
      <ul className="m-0 mt-6 flex list-none flex-col p-0">
        {connections.map(c => (
          <li key={c.name} className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3.5 gap-y-1.5 border-b border-ink-14 py-3.5 md:min-h-[72px] md:grid-cols-[220px_minmax(0,1fr)_160px] md:gap-6 md:py-2">
            <span className="flex flex-col gap-[3px]">
              <span className="text-lg font-light">{c.name}</span>
              <span className="ui-label !text-[9px]">For {moduleById(c.module).label}</span>
            </span>
            {c.copy
              ? <button type="button" className={cn(buttonClassName(), cta, 'md:order-last')} onClick={() => void copy()} aria-label={`Copy the MCP URL for ${c.name}`}>{copied ? 'COPIED' : c.cta}</button>
              : <a href={c.href} target="_blank" rel="noopener" className={cn(buttonClassName(), cta, 'md:order-last')} aria-label={`${c.cta.toLowerCase()} ${c.name} (opens in a new tab)`}>{c.cta}</a>}
            <span className="text-[13px] font-light text-ink-60 md:text-sm">{c.description}</span>
          </li>
        ))}
      </ul>
      <p aria-live="polite" className={cn('m-0 mt-3 text-sm', copyError ? 'text-accent' : 'text-ink-60', !copyError && !copied && 'ui-sr-only')}>
        {copyError || (copied ? `Copied ${copied}` : '')}
      </p>
    </>
  );
}
