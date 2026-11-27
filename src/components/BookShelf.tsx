import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { formatDate } from '../lib/format';
import {
  inProgress, lastUpdated, matchesBookSearch, progressMarks, shelves, sortCatalog, unitLabel,
  type Book, type BookStatus, type BookType, type ShelfData,
} from '../lib/books';
import { cn } from '../lib/utils';
import { Box, CloseIcon, ExternalIcon, IconButton, ItemTypeIcon, Label, SearchField, TickScale, buttonClassName } from './ui';

// The Bookshelf page (docs/redesign/pages/bookshelf.md). Desktop: SHELVES and TYPE filters on
// the left, the CATALOG table in the middle, the IN PROGRESS card on the right. Phone: title,
// search, the in-progress card, shelf and type chips, then list rows. Any row opens the item's
// details (chapters, log, dates); /books?id=<slug> opens it directly (the header search,
// public/ui/site-search.js, links book results there).

const TYPES: BookType[] = ['book', 'paper', 'course'];
const TYPE_LABEL: Record<BookType, string> = { book: 'Book', paper: 'Paper', course: 'Course' };
const STATUS_LABEL: Record<BookStatus, string> = { reading: 'Reading', paused: 'Paused', backlog: 'Backlog', reference: 'Reference', completed: 'Done' };
const PRIORITY_LABEL = { high: 'High', medium: 'Medium', low: 'Low' } as const;

// Catalog rows draw one tick per unit up to 12; longer items scale onto 12 ticks.
const CATALOG_TICKS = 12;
// The in-progress card draws one cell per unit up to 10; longer items scale onto 10 cells.
const CARD_CELLS = 10;

function useMedia(query: string): boolean {
  const [matches, setMatches] = useState(() => window.matchMedia(query).matches);
  useEffect(() => {
    const media = window.matchMedia(query);
    const update = () => setMatches(media.matches);
    update();
    media.addEventListener('change', update);
    return () => media.removeEventListener('change', update);
  }, [query]);
  return matches;
}

function authorsLine(book: Book): string {
  if (book.authors.length === 0) return '';
  if (book.authors.length <= 2) return book.authors.join(', ');
  return `${book.authors[0]} et al.`;
}

// Dates from the API are YYYY-MM-DD or ISO timestamps; log headings are free text, so fall
// back to the raw value.
function formatSafe(value: string): string {
  try { return formatDate(value); } catch { return value; }
}

function updatedLabel(book: Book): string {
  const value = lastUpdated(book);
  return value ? formatSafe(value) : '';
}

function manageHref(book: Book) {
  return `/manage/books?id=${encodeURIComponent(book.slug)}`;
}

function progressText(book: Book): string {
  return `${book.completed} of ${book.total} ${unitLabel(book.unit, book.total)}`;
}

// The ?id= deep link, kept in sync with the open details dialog.
function setIdParam(slug: string | null) {
  const url = new URL(window.location.href);
  if (slug) url.searchParams.set('id', slug); else url.searchParams.delete('id');
  window.history.replaceState(window.history.state, '', url);
}

export function BookShelf({ data }: { data: ShelfData }) {
  const phone = useMedia('(max-width: 767px)');
  const [query, setQuery] = useState('');
  const [shelf, setShelf] = useState<string | null>(null);
  const [type, setType] = useState<BookType | null>(null);
  const [openSlug, setOpenSlug] = useState<string | null>(() => new URLSearchParams(window.location.search).get('id'));

  const all = data.all;
  const shelfList = useMemo(() => shelves(all), [all]);
  const current = useMemo(() => inProgress(all)[0] ?? null, [all]);
  const rows = useMemo(() => sortCatalog(all.filter(book =>
    (!shelf || book.category === shelf) && (!type || book.type === type) && matchesBookSearch(book, query))), [all, shelf, type, query]);
  const typeCounts = useMemo(() => Object.fromEntries(TYPES.map(t => [t, all.filter(book => book.type === t).length])) as Record<BookType, number>, [all]);
  const open = openSlug ? all.find(book => book.slug === openSlug) ?? null : null;

  // A shelf that no longer exists (its last item was deleted elsewhere) stops filtering.
  useEffect(() => { if (shelf && !shelfList.some(s => s.category === shelf)) setShelf(null); }, [shelf, shelfList]);

  function openBook(book: Book) { setOpenSlug(book.slug); setIdParam(book.slug); }
  function closeBook() { setOpenSlug(null); setIdParam(null); }

  const filtered = Boolean(shelf || type || query.trim());
  const empty = all.length === 0
    ? <p className="mt-6 text-ink-60">Nothing on the shelf yet. <a href="/manage/books?new=1" className="text-ink underline">Add a book, course or paper</a>.</p>
    : rows.length === 0 && filtered
      ? <p className="mt-6 text-ink-60">No items match {query.trim() ? `"${query.trim()}"` : 'this filter'}.</p>
      : null;

  const props = { all, rows, current, shelfList, typeCounts, shelf, setShelf, type, setType, query, setQuery, openBook, empty };
  return (
    <>
      {openSlug && !open && <p role="alert" className="mb-4 flex items-center justify-between gap-3 border border-accent p-3 text-sm text-accent">
        That item is no longer on the shelf.
        <button type="button" onClick={closeBook} className="border-0 bg-transparent p-0 font-mono text-[11px] uppercase tracking-[0.16em] text-ink underline">Dismiss</button>
      </p>}
      {phone ? <PhoneShelf {...props} /> : <DesktopShelf {...props} />}
      {open && <BookDetails book={open} onClose={closeBook} />}
    </>
  );
}

type ViewProps = {
  all: Book[];
  rows: Book[];
  current: Book | null;
  shelfList: { category: string; count: number }[];
  typeCounts: Record<BookType, number>;
  shelf: string | null;
  setShelf: (value: string | null) => void;
  type: BookType | null;
  setType: (value: BookType | null) => void;
  query: string;
  setQuery: (value: string) => void;
  openBook: (book: Book) => void;
  empty: ReactNode;
};

function DesktopShelf({ all, rows, current, shelfList, typeCounts, shelf, setShelf, type, setType, query, setQuery, openBook, empty }: ViewProps) {
  return (
    <div className="flex flex-col">
      <header className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div className="flex flex-col gap-2">
          <Label>{all.length} {all.length === 1 ? 'item' : 'books, courses & papers'}</Label>
          <h1 className="m-0 font-extralight !text-[52px] !leading-[0.9]">Bookshelf</h1>
        </div>
        <div className="flex flex-wrap items-center gap-2.5">
          <SearchField label="Search title or author" value={query} onChange={event => setQuery(event.target.value)} className="min-w-[180px] flex-1 lg:w-[240px] lg:flex-none" />
          <a href="/manage/books" className={buttonClassName()}>Manage</a>
          <a href="/manage/books?new=1" className={buttonClassName({ variant: 'primary' })}>+ Add item</a>
        </div>
      </header>

      <div className="mt-[30px] grid grid-cols-[170px_minmax(0,1fr)] gap-x-10 gap-y-8 xl:grid-cols-[190px_minmax(0,1fr)_290px] xl:gap-x-11">
        <nav aria-label="Shelves and types" className="row-span-2 flex flex-col xl:row-span-1">
          <h2 className="ui-section-header__title mb-2.5">Shelves</h2>
          <ShelfButton label="All" count={all.length} active={!shelf} onClick={() => setShelf(null)} />
          {shelfList.map(s => <ShelfButton key={s.category} label={s.category} count={s.count} active={shelf === s.category} onClick={() => setShelf(shelf === s.category ? null : s.category)} />)}
          <h2 className="ui-section-header__title mb-2.5 mt-6">Type</h2>
          {TYPES.map(t => (
            <button key={t} type="button" aria-pressed={type === t} onClick={() => setType(type === t ? null : t)}
              className={cn('flex min-h-8 items-center gap-2.5 border-0 bg-transparent p-0 text-left text-sm text-ink hover:text-ink-hover',
                type === t ? 'font-normal underline underline-offset-4' : 'font-light', type && type !== t && 'opacity-70')}>
              <ItemTypeIcon type={t} size={18} />{TYPE_LABEL[t]}
              <span className="ml-auto font-mono text-[10px] text-ink-60">{typeCounts[t]}</span>
            </button>
          ))}
        </nav>

        <aside aria-label="In progress" className="flex flex-col gap-3.5 max-xl:max-w-[440px] xl:col-start-3 xl:row-start-1">
          <Label>In progress</Label>
          <InProgressCard book={current} onOpen={openBook} />
        </aside>

        <section aria-labelledby="catalog-heading" className="flex min-w-0 flex-col xl:col-start-2 xl:row-start-1">
          <div className="mb-2 flex items-baseline justify-between">
            <h2 id="catalog-heading" className="ui-section-header__title">Catalog</h2>
            <Label>Sorted by priority</Label>
          </div>
          <div aria-hidden="true" className="ui-label grid grid-cols-[26px_minmax(0,1fr)_70px_110px] gap-3.5 border-b border-ink pb-1.5 xl:grid-cols-[26px_minmax(0,1fr)_150px_70px_110px]">
            <span /><span>Title</span><span className="max-xl:hidden">Author</span><span>Priority</span><span className="text-right">Progress</span>
          </div>
          {rows.length > 0 && <ul aria-label="Catalog" className="m-0 list-none p-0">
            {rows.map(book => (
              <li key={book.slug}>
                <button type="button" onClick={() => openBook(book)}
                  className={cn('grid min-h-10 w-full grid-cols-[26px_minmax(0,1fr)_70px_110px] items-center gap-3.5 border-0 border-b border-solid border-ink-14 bg-transparent px-0 py-1.5 text-left text-ink hover:bg-ink-05 xl:grid-cols-[26px_minmax(0,1fr)_150px_70px_110px]',
                    book.slug === current?.slug && 'bg-ink-05')}>
                  <span className="flex"><ItemTypeIcon type={book.type} size={18} /><span className="sr-only">{TYPE_LABEL[book.type]}: </span></span>
                  <span className="flex min-w-0 flex-col">
                    <span className={cn('truncate text-[15px] font-light', book.status === 'completed' && 'line-through opacity-45')}>{book.title}</span>
                    <span className="truncate text-xs text-ink-60 xl:hidden">{authorsLine(book)}</span>
                  </span>
                  <span className="truncate text-xs text-ink-60 max-xl:hidden">{authorsLine(book)}</span>
                  <Label className="!text-[9px]">{PRIORITY_LABEL[book.priority]}<span className="sr-only"> priority</span></Label>
                  <Progress book={book} />
                </button>
              </li>
            ))}
          </ul>}
          {empty}
        </section>
      </div>
    </div>
  );
}

function PhoneShelf({ all, rows, current, shelfList, typeCounts, shelf, setShelf, type, setType, query, setQuery, openBook, empty }: ViewProps) {
  return (
    <div className="flex flex-col">
      <header className="flex items-end justify-between gap-4">
        <div className="flex flex-col gap-1.5">
          <Label>{all.length} {all.length === 1 ? 'item' : 'items'}</Label>
          <h1 className="m-0 font-extralight !text-[40px] !leading-[0.9]">Bookshelf</h1>
        </div>
        <a href="/manage/books?new=1" aria-label="Add item" title="Add item" className={cn(buttonClassName({ variant: 'primary', icon: true }), 'text-lg')}>+</a>
      </header>
      <SearchField label="Search title or author" value={query} onChange={event => setQuery(event.target.value)} className="mt-4" />

      <Label className="mt-[22px]">In progress</Label>
      <div className="mx-[3px] mt-2.5"><InProgressCard book={current} onOpen={openBook} compact /></div>

      <div role="group" aria-label="Shelves and types" className="-mx-[22px] mt-[22px] flex gap-2 overflow-x-auto px-[22px] pb-1">
        <Chip label="All" active={!shelf} onClick={() => setShelf(null)} />
        {shelfList.map(s => <Chip key={s.category} label={s.category} active={shelf === s.category} onClick={() => setShelf(shelf === s.category ? null : s.category)} />)}
        <span aria-hidden="true" className="mx-1 w-px flex-none self-stretch bg-ink-30" />
        {TYPES.filter(t => typeCounts[t] > 0).map(t => (
          <Chip key={t} label={TYPE_LABEL[t]} icon={<ItemTypeIcon type={t} size={14} />} active={type === t} onClick={() => setType(type === t ? null : t)} />
        ))}
      </div>

      {rows.length > 0 && <ul aria-label="Catalog" className="m-0 mt-2 list-none p-0">
        {rows.map(book => (
          <li key={book.slug}>
            <button type="button" onClick={() => openBook(book)}
              className="grid min-h-[58px] w-full grid-cols-[22px_minmax(0,1fr)_auto] items-center gap-3 border-0 border-b border-solid border-ink-14 bg-transparent px-0 py-2 text-left text-ink">
              <span className="flex"><ItemTypeIcon type={book.type} size={18} /><span className="sr-only">{TYPE_LABEL[book.type]}: </span></span>
              <span className="flex min-w-0 flex-col">
                <span className={cn('truncate text-[15px] font-light', book.status === 'completed' && 'line-through opacity-45')}>{book.title}</span>
                <span className="truncate text-xs text-ink-60">{authorsLine(book)}</span>
              </span>
              <Label className="!text-[8.5px]">{PRIORITY_LABEL[book.priority]}<span className="sr-only"> priority</span></Label>
            </button>
          </li>
        ))}
      </ul>}
      {empty}
    </div>
  );
}

function ShelfButton({ label, count, active, onClick }: { label: string; count: number; active: boolean; onClick: () => void }) {
  return (
    <button type="button" aria-pressed={active} onClick={onClick}
      className={cn('flex min-h-8 items-center gap-2 border-0 border-l border-solid bg-transparent py-0 pl-2.5 pr-0 text-left text-sm text-ink hover:text-ink-hover',
        active ? 'border-ink font-normal' : 'border-transparent font-light opacity-70')}>
      <span className="min-w-0 flex-1 truncate">{label}</span>
      <span className="font-mono text-[10px] text-ink-60">{count}</span>
    </button>
  );
}

function Chip({ label, icon, active, onClick }: { label: string; icon?: ReactNode; active: boolean; onClick: () => void }) {
  return (
    <button type="button" aria-pressed={active} onClick={onClick}
      className={cn('flex h-11 flex-none items-center gap-2 border border-solid px-3 font-mono text-[10px] font-medium uppercase tracking-[0.1em]',
        active ? 'border-ink bg-ink text-paper' : 'border-ink-30 bg-transparent text-ink')}>
      {icon}{label}
    </button>
  );
}

// Catalog progress: a tick per unit (every 5th taller), or a dash when the item has no units.
function Progress({ book }: { book: Book }) {
  const marks = progressMarks(book.completed, book.total, CATALOG_TICKS);
  if (marks.count === 0) return <span className="text-right font-mono text-[10px] text-ink-60"><span aria-hidden="true">—</span><span className="sr-only">No progress tracked</span></span>;
  return <TickScale count={marks.count} filled={marks.filled} label={progressText(book)} className="ml-auto !h-3 justify-end gap-[3px] [&>span]:!h-3" />;
}

// Hatched cells for finished units, empty cells for the rest.
function Cells({ book, compact = false }: { book: Book; compact?: boolean }) {
  const marks = progressMarks(book.completed, book.total, CARD_CELLS);
  return (
    <span role="img" aria-label={progressText(book)} className={cn('grid', compact ? 'gap-[3px]' : 'gap-1')} style={{ gridTemplateColumns: `repeat(${marks.count}, minmax(0, 1fr))` }}>
      {Array.from({ length: marks.count }, (_, i) => (
        <span key={i} className={cn('box-border border border-solid', compact ? 'h-2.5' : 'h-3.5', i < marks.filled ? 'ui-hatch--diagonal border-ink' : 'border-ink-30')} />
      ))}
    </span>
  );
}

function LineBook({ width, height }: { width: number; height: number }) {
  return (
    <svg aria-hidden="true" focusable="false" width={width} height={height} viewBox="0 0 64 84" className="ui-icon">
      <rect x="1" y="1" width="62" height="82" vectorEffect="non-scaling-stroke" />
      <path d="M8 1 V83" vectorEffect="non-scaling-stroke" />
      <path d="M18 22 H54 M18 30 H46" strokeOpacity="0.5" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}

function InProgressCard({ book, onOpen, compact = false }: { book: Book | null; onOpen: (book: Book) => void; compact?: boolean }) {
  if (!book) {
    return <p className="m-0 text-sm font-light text-ink-60">Nothing in progress. Tick a chapter in <a href="/manage/books" className="text-ink underline">Manage</a> to start one.</p>;
  }
  const left = book.total - book.completed;
  const updated = updatedLabel(book);
  if (compact) {
    return (
      <Box className="p-0">
        <button type="button" onClick={() => onOpen(book)} className="grid w-full grid-cols-[44px_minmax(0,1fr)] gap-3.5 border-0 bg-transparent px-4 py-3.5 text-left text-ink">
          <LineBook width={44} height={58} />
          <span className="flex min-w-0 flex-col gap-1.5">
            <span className="text-[17px] font-light leading-[1.2]">{book.title}</span>
            <Cells book={book} compact />
            <span className="flex justify-between gap-2">
              <Label className="!text-[9px]">{book.completed} / {book.total} {unitLabel(book.unit, book.total)}</Label>
              {updated && <Label className="!text-[9px]">{updated}</Label>}
            </span>
          </span>
        </button>
      </Box>
    );
  }
  return (
    <>
      <Box className="flex flex-col gap-2 p-[18px]">
        <LineBook width={64} height={84} />
        <button type="button" onClick={() => onOpen(book)} className="border-0 bg-transparent p-0 text-left text-[22px] font-light leading-[1.15] text-ink hover:underline">{book.title}</button>
        <span className="text-xs text-ink-60">{authorsLine(book)}{book.edition ? ` · ${book.edition}` : ''}</span>
        <div className="mt-2 flex flex-col gap-1.5">
          <div className="flex justify-between">
            <Label className="!text-[9px]">{unitLabel(book.unit, 2)}</Label>
            <span className="font-mono text-xs">{book.completed} / {book.total}</span>
          </div>
          <Cells book={book} />
        </div>
        {updated && <Label className="mt-1.5 !text-[9px]">Last updated {updated}</Label>}
        <div className="mt-1.5 flex gap-2">
          <a href={manageHref(book)} className={cn(buttonClassName({ variant: 'primary' }), 'flex-1 px-2')}>Log reading</a>
          {book.url
            ? <a href={book.url} target="_blank" rel="noreferrer" className={cn(buttonClassName(), 'flex-1 px-2')}>Open ↗<span className="sr-only"> (opens in a new tab)</span></a>
            : <button type="button" onClick={() => onOpen(book)} className={cn(buttonClassName(), 'flex-1 px-2')}>Details</button>}
        </div>
      </Box>
      <span className="ui-note">{left} {unitLabel(book.unit, left)} to go</span>
    </>
  );
}

function BookDetails({ book, onClose }: { book: Book; onClose: () => void }) {
  const dialog = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    // Return focus to whatever opened the dialog (a row, the card) when it closes.
    const opener = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    dialog.current?.showModal();
    const previous = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => { document.body.style.overflow = previous; opener?.focus(); };
  }, []);
  const updated = updatedLabel(book);
  const done = book.chapters.filter(chapter => chapter.done).length;
  // Progress may be tracked in pages while the checklist lists chapters: name the checklist by the
  // item type's default unit in that case.
  const checklistUnit = book.progress ? (book.type === 'course' ? 'module' : book.type === 'paper' ? 'section' : 'chapter') : book.unit;
  return (
    <dialog ref={dialog} aria-labelledby="book-details-title" onCancel={onClose} onClick={event => { if (event.target === event.currentTarget) onClose(); }}
      className="m-auto w-[calc(100%-32px)] max-w-2xl border-0 bg-transparent p-[3px] text-ink backdrop:bg-paper/80">
      <Box className="max-h-[85dvh] overflow-y-auto bg-paper p-5 md:p-7">
        <div className="flex items-start gap-4">
          <span className="mt-1 flex"><ItemTypeIcon type={book.type} size={22} /></span>
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <Label>{TYPE_LABEL[book.type]} · {book.category} · {STATUS_LABEL[book.status]}</Label>
            <h2 id="book-details-title" className="m-0 text-[26px] font-extralight leading-[1.1]">{book.title}</h2>
            {(book.authors.length > 0 || book.edition) && <p className="m-0 text-sm text-ink-60">{[book.authors.join(', '), book.edition].filter(Boolean).join(' · ')}</p>}
          </div>
          <IconButton label="Close" size="sm" onClick={onClose}><CloseIcon size={13} /></IconButton>
        </div>

        {book.total > 0 && <div className="mt-5 flex flex-col gap-1.5">
          <div className="flex justify-between">
            <Label>{unitLabel(book.unit, 2)}</Label>
            <span className="font-mono text-xs">{book.completed} / {book.total} · {book.percent}%</span>
          </div>
          <Cells book={book} />
        </div>}

        <dl className="m-0 mt-5 grid grid-cols-2 gap-x-5 gap-y-3 sm:grid-cols-4">
          <Meta label="Started" value={book.started ? formatSafe(book.started) : '—'} />
          <Meta label="Finished" value={book.finished ? formatSafe(book.finished) : '—'} />
          <Meta label="Priority" value={PRIORITY_LABEL[book.priority]} />
          {book.rating ? <Meta label="Rating" value={`${book.rating} / 5`} /> : <Meta label="Updated" value={updated || '—'} />}
        </dl>

        {book.chapters.length > 0 && <section className="mt-6">
          <h3 className="ui-section-header__title mb-2">{unitLabel(checklistUnit, 2)} <span className="font-mono text-[10px] font-normal tracking-normal text-ink-60">{done} / {book.chapters.length}</span></h3>
          <ul className="m-0 list-none p-0">
            {book.chapters.map(chapter => (
              <li key={chapter.index} className={cn('flex min-h-8 items-center gap-3 border-b border-ink-14 text-sm font-light', chapter.done && 'line-through opacity-45')}>
                <span aria-hidden="true" className="relative inline-block size-3 flex-none border border-solid border-ink">
                  {chapter.done && <svg viewBox="0 0 12 12" className="ui-icon absolute inset-0 size-full"><path d="M0 0 L12 12 M12 0 L0 12" /></svg>}
                </span>
                <span className="sr-only">{chapter.done ? 'Done: ' : 'Not done: '}</span>{chapter.label}
              </li>
            ))}
          </ul>
        </section>}

        {book.log.length > 0 && <section className="mt-6">
          <h3 className="ui-section-header__title mb-2">Log</h3>
          <ul className="m-0 flex list-none flex-col gap-3 p-0">
            {book.log.map((note, i) => (
              <li key={`${note.date}-${i}`} className="flex flex-col gap-1 border-b border-ink-14 pb-3">
                <Label>{formatSafe(note.date)}</Label>
                <p className="m-0 text-sm font-light">{note.text}</p>
              </li>
            ))}
          </ul>
        </section>}

        <div className="mt-6 flex flex-wrap gap-2">
          <a href={manageHref(book)} className={buttonClassName({ variant: 'primary' })}>Edit details &amp; progress</a>
          {book.url && <a href={book.url} target="_blank" rel="noreferrer" className={cn(buttonClassName(), 'gap-2')}>Open source <ExternalIcon size={12} /><span className="sr-only"> (opens in a new tab)</span></a>}
        </div>
      </Box>
    </dialog>
  );
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-1">
      <Label as="dt">{label}</Label>
      <dd className="m-0 truncate font-mono text-xs">{value}</dd>
    </div>
  );
}
