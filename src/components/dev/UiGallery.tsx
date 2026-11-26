import { useEffect, useState, type ReactNode } from 'react';
import {
  Avatar, Box, Button, buttonClassName, Checkbox, GoalPattern, HatchFill, IconButton, Label, LogoSpot, MeasurementLine,
  NowLine, Progress, SearchField, SectionHeader, SegmentedTabs, Switch, TickScale, TodayMarker,
} from '../ui';
import * as Icons from '../ui/icons';
import { resolveTheme, THEMES, type Theme } from '../../lib/theme';
import { cn } from '../../lib/utils';

// Dev-only gallery (/dev/ui) of every drawing primitive in every state, for visual review.
// The theme picker previews a theme on this page without saving it.

const slug = (text: string) => `ui-${text.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')}`;

function Panel({ title, meta, children, className }: { title: string; meta?: string; children: ReactNode; className?: string }) {
  return (
    <section aria-labelledby={slug(title)} className={cn('flex min-w-0 flex-col gap-4', className)}>
      <SectionHeader id={slug(title)} title={title} meta={meta} />
      <div className="border-t border-ink-30" />
      {children}
    </section>
  );
}

function Specimen({ name, children, className }: { name: string; children: ReactNode; className?: string }) {
  return (
    <div className={cn('flex min-w-0 flex-col gap-3', className)}>
      <Label>{name}</Label>
      <div className="flex min-w-0 flex-wrap items-center gap-6">{children}</div>
    </div>
  );
}

// A sample logo (any colors); LogoSpot paints its shape in ink.
const SAMPLE_LOGO = `data:image/svg+xml,${encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path fill="#f60" d="M12 2 22 20H2Z"/><circle cx="12" cy="14" r="3" fill="#fff"/></svg>')}`;

const SCHEDULE = [
  { title: 'LeetCode', time: '7:15 AM – 8:45 AM' },
  { title: 'System design', time: '9 AM – 10:30 AM' },
  { title: 'Alloy', time: '4 PM – 5 PM' },
  { title: 'Fourth goal', time: '6 PM – 7 PM' },
];

export default function UiGallery() {
  const [theme, setTheme] = useState<Theme>('dark');
  const [modules, setModules] = useState({ timeline: true, leet: false });
  const [steps, setSteps] = useState([true, false, false]);
  const [priority, setPriority] = useState<'all' | 'high' | 'medium' | 'low'>('all');
  const [query, setQuery] = useState('');

  // Follow the page theme: the saved choice it loaded with, or data-theme changed in devtools.
  useEffect(() => {
    const root = document.documentElement;
    const sync = () => setTheme(resolveTheme(root.dataset.theme));
    sync();
    const observer = new MutationObserver(sync);
    observer.observe(root, { attributes: true, attributeFilter: ['data-theme'] });
    return () => observer.disconnect();
  }, []);

  function pickTheme(next: Theme) {
    document.documentElement.dataset.theme = next;
  }

  return (
    <main className="mx-auto flex max-w-[1400px] flex-col gap-12 px-[22px] py-10 md:px-14">
      <header className="flex flex-wrap items-end gap-6">
        <div className="flex flex-col gap-2">
          <Label>Dev only · not linked from the app</Label>
          <h1 className="m-0 text-[40px] font-extralight leading-[0.9] md:text-[52px]">Drawing primitives</h1>
        </div>
        <SegmentedTabs className="md:ml-auto" label="Theme" tabs={THEMES.map(value => ({ value, label: value }))} value={theme} onChange={pickTheme} />
      </header>

      <div className="grid gap-x-14 gap-y-12 lg:grid-cols-2">
        <Panel title="Box" meta="3px corner overshoot">
          <Specimen name="Default · small (2px)">
            <Box className="flex h-20 w-40 items-center justify-center text-sm font-light">Box</Box>
            <Box small className="h-4 w-4" />
            <Box small className="h-[18px] w-[18px]" />
          </Specimen>
        </Panel>

        <Panel title="Measurement line" meta="spans and durations">
          <Specimen name="Default · planned · late" className="[&>div]:flex-col [&>div]:items-stretch">
            <div className="flex items-center gap-3"><MeasurementLine className="flex-1" /><Label>5:00 AM — 15.5 hours — 8:30 PM</Label><MeasurementLine className="flex-1" /></div>
            <div className="flex flex-col gap-2"><Label>90 min · planned</Label><MeasurementLine state="plan" className="w-40" /></div>
            <div className="flex flex-col gap-2"><Label className="text-accent">Late</Label><MeasurementLine state="late" className="w-40" /></div>
          </Specimen>
        </Panel>

        <Panel title="Hatch fill" meta="diagonal · crosshatch · dots">
          <Specimen name="Fills">
            <HatchFill pattern="diagonal" className="h-14 w-24 border border-ink" />
            <HatchFill pattern="crosshatch" className="h-14 w-24 border border-ink" />
            <HatchFill pattern="dots" className="h-14 w-24 border border-ink" />
          </Specimen>
          <Specimen name="Progress on a measurement line" className="[&>div]:flex-col [&>div]:items-stretch">
            {[[0, 'Not started'], [0.4, '40%'], [1, 'Done']].map(([value, text]) => (
              <div key={String(text)} className="grid grid-cols-[minmax(0,1fr)_90px] items-center gap-5">
                <Progress value={Number(value)} label={`Progress ${text}`} />
                <Label>{text}</Label>
              </div>
            ))}
            <div className="grid grid-cols-[minmax(0,1fr)_90px] items-center gap-5">
              <Progress value={0.7} late pattern="crosshatch" label="Progress 70%, late" />
              <Label className="text-accent">70% · Late</Label>
            </div>
          </Specimen>
        </Panel>

        <Panel title="Goal patterns" meta="by goal order">
          <Specimen name="Schedule blocks">
            <div className="grid w-full grid-cols-2 gap-3 sm:grid-cols-4">
              {SCHEDULE.map((block, index) => (
                <div key={block.title} className="flex h-16 min-w-0 border border-ink">
                  <GoalPattern index={index} />
                  <span className="flex min-w-0 flex-col px-[5px] py-[2px]">
                    <span className="truncate text-[12px]">{block.title}</span>
                    <span className="font-mono text-[8.5px] text-ink-60">{block.time}</span>
                  </span>
                </div>
              ))}
            </div>
          </Specimen>
          <Specimen name="Legend">
            {SCHEDULE.map((block, index) => (
              <span key={block.title} className="flex items-center gap-2 text-[13px] font-light"><GoalPattern index={index} variant="swatch" />{block.title}</span>
            ))}
          </Specimen>
        </Panel>

        <Panel title="Tick scale" meta="streaks · steps · chapters">
          <Specimen name="Streak 2 of record 29" className="[&>div]:flex-col [&>div]:items-stretch [&>div]:gap-1">
            <TickScale count={30} filled={2} label="2 day streak, record 29" />
            <div className="flex justify-between"><Label>0</Label><Label>Days</Label><Label>29</Label></div>
          </Specimen>
          <Specimen name="Checklist 2 / 5 · empty · full">
            <TickScale count={5} filled={2} className="w-16" label="2 of 5 steps" />
            <TickScale count={10} filled={0} className="w-28" label="0 of 10 chapters" />
            <TickScale count={6} filled={6} className="w-20" label="6 of 6 steps" />
          </Specimen>
        </Panel>

        <Panel title="Switch" meta='role="switch"'>
          <Specimen name="Interactive">
            <span className="flex items-center gap-2"><Switch checked={modules.timeline} onCheckedChange={on => setModules(m => ({ ...m, timeline: on }))} aria-labelledby="sw-timeline" /><span id="sw-timeline" className="text-[15px] font-light">Timeline</span></span>
            <span className="flex items-center gap-2"><Switch checked={modules.leet} onCheckedChange={on => setModules(m => ({ ...m, leet: on }))} aria-labelledby="sw-leet" /><span id="sw-leet" className="text-[15px] font-light">Leetgrinder</span></span>
          </Specimen>
          <Specimen name="On · off · locked on · locked off">
            <Switch checked aria-label="On" />
            <Switch checked={false} aria-label="Off" />
            <Switch checked locked aria-label="Locked on" />
            <Switch checked={false} locked aria-label="Locked off" />
          </Specimen>
        </Panel>

        <Panel title="Checkbox" meta="X · accent ring for next">
          <Specimen name="Goal sessions" className="[&>div]:flex-col [&>div]:items-stretch [&>div]:gap-0">
            {['Kth Largest in Stream', 'DDIA chapter 5', 'Alloy model check'].map((title, index) => {
              const next = steps.findIndex(done => !done) === index;
              return (
                <label key={title} className={cn('flex min-h-[54px] items-center gap-5 border-b border-ink-14 pl-3', steps[index] && 'line-through opacity-45', next && 'text-[17px]')}>
                  <Checkbox checked={steps[index]} next={next} onChange={event => setSteps(s => s.map((v, i) => (i === index ? event.target.checked : v)))} />
                  <span className="font-light">{title}</span>
                </label>
              );
            })}
          </Specimen>
          <Specimen name="Unchecked · checked · next · disabled · 18px">
            <Checkbox aria-label="Unchecked" />
            <Checkbox aria-label="Checked" defaultChecked />
            <Checkbox aria-label="Next" next />
            <Checkbox aria-label="Disabled" disabled />
            <Checkbox aria-label="Large" size={18} defaultChecked />
          </Specimen>
        </Panel>

        <Panel title="Segmented tabs" meta="arrows · home · end">
          <SegmentedTabs label="Priority" controls="ui-tab-panel" value={priority} onChange={setPriority}
            tabs={[{ value: 'all', label: 'All 61' }, { value: 'high', label: 'High' }, { value: 'medium', label: 'Medium' }, { value: 'low', label: 'Low' }]} />
          <p id="ui-tab-panel" role="tabpanel" className="m-0 text-[13px] font-light">Showing {priority} priority companies.</p>
        </Panel>

        <Panel title="Button" meta="44px · mono 11px">
          <Specimen name="Primary · secondary · danger · disabled · link">
            <Button variant="primary">Save goal</Button>
            <Button>Cancel</Button>
            <Button variant="danger">Delete</Button>
            <Button disabled>Disabled</Button>
            <a href="#ui-button" className={buttonClassName({ variant: 'secondary' })}>Link</a>
          </Specimen>
          <Specimen name="Icon 44 · icon 34 · danger icon">
            <IconButton label="Add"><Icons.PlusIcon /></IconButton>
            <IconButton label="Settings" size="sm"><Icons.SettingsIcon size={15} /></IconButton>
            <IconButton label="Edit" size="sm"><Icons.EditIcon size={14} /></IconButton>
            <IconButton label="Delete" variant="danger" size="sm"><Icons.DeleteIcon size={14} /></IconButton>
          </Specimen>
        </Panel>

        <Panel title="Search field" meta="44px · 34px header">
          <SearchField label="Search companies" value={query} onChange={event => setQuery(event.target.value)} />
          <SearchField label="Search everything" size="sm" hint="/" className="w-[210px] max-w-full" />
        </Panel>

        <Panel title="Section header · label" meta="type">
          <SectionHeader title="Goal sessions" />
          <SectionHeader title="Active goals" meta="4 goals" />
          <SectionHeader title="Company" meta="Steps" rule level={3} />
          <Specimen name="Labels and notes">
            <Label>Week 48 · America / Chicago</Label>
            <Label className="text-accent">Late</Label>
            <span className="ui-note">next in 25 min</span>
          </Specimen>
        </Panel>

        <Panel title="Now line · today" meta="accent">
          <div className="relative h-36 border-l border-ink-14">
            {[0, 1, 2, 3].map(row => <div key={row} className="absolute inset-x-0 border-t border-ink-07" style={{ top: `${row * 33}%` }} />)}
            <NowLine time={17 * 60 + 20} className="absolute inset-x-0" style={{ top: '58%' }} />
            <TodayMarker className="absolute top-0 bottom-0" style={{ left: '70%' }} />
          </div>
        </Panel>

        <Panel title="Logo spot · avatar" meta="ink only">
          <Specimen name="Monogram · logo (masked) · missing logo · 40px">
            <LogoSpot name="Jane Street" />
            <LogoSpot name="Ramp" />
            <LogoSpot name="Delta" src={SAMPLE_LOGO} />
            <LogoSpot name="Broken Logo" src="data:image/png;base64,AAAA" />
            <LogoSpot name="Datadog" size={40} />
          </Specimen>
          <Specimen name="28 · 34 · 46 · muted">
            <Avatar name="Priya Natarajan" size={28} />
            <Avatar name="Priya Natarajan" />
            <Avatar name="Priya Natarajan" size={46} />
            <Avatar name="Sam Ortiz" muted />
          </Specimen>
        </Panel>

        <Panel title="Line icons" meta="1px · currentColor" className="lg:col-span-2">
          <div className="grid grid-cols-3 gap-4 sm:grid-cols-5 lg:grid-cols-10">
            {Object.entries(Icons).map(([name, Icon]) => (
              <div key={name} className="flex flex-col items-center gap-2 py-2">
                <Icon size={20} />
                <span className="font-mono text-[9px] tracking-[0.1em] text-ink-60">{name.replace(/Icon$/, '')}</span>
              </div>
            ))}
          </div>
        </Panel>
      </div>
    </main>
  );
}
