'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { cn } from '../lib/utils';
import { buildShelf, categoryColor, type Book, type BookType, type ShelfData, type BookStatus } from '../lib/books';

const SHELF_ORDER: BookType[] = ['book', 'course', 'paper'];
const SHELF_TABS: Record<BookType, { label: string; singular: string }> = {
  book: { label: 'Books', singular: 'book' },
  course: { label: 'Video courses', singular: 'video course' },
  paper: { label: 'Papers', singular: 'paper or article' },
};

const STATUS_META: Record<BookStatus, { label: string; classes: string }> = {
  reading: { label: 'Reading', classes: 'border-blue-500/50 bg-blue-500/10 text-blue-300' },
  paused: { label: 'Paused', classes: 'border-yellow-500/50 bg-yellow-500/10 text-yellow-300' },
  backlog: { label: 'Backlog', classes: 'border-zinc-600/60 bg-zinc-700/20 text-zinc-400' },
  reference: { label: 'Reference', classes: 'border-purple-500/50 bg-purple-500/10 text-purple-300' },
  completed: { label: 'Done', classes: 'border-green-500/50 bg-green-500/10 text-green-300' },
};

function ProgressBar({ percent, color }: { percent: number; color: string }) {
  return (
    <div className="h-1.5 w-full rounded-full bg-zinc-800 overflow-hidden">
      <div
        className="h-full rounded-full transition-all"
        style={{ width: `${Math.max(percent, percent > 0 ? 3 : 0)}%`, backgroundColor: color }}
      />
    </div>
  );
}

function StatusBadge({ status }: { status: BookStatus }) {
  const m = STATUS_META[status];
  return (
    <span className={cn('text-[10px] uppercase tracking-wide px-1.5 py-0.5 rounded border', m.classes)}>
      {m.label}
    </span>
  );
}

function CoverImage({ book, className }: { book: Book; className?: string }) {
  const [failed, setFailed] = useState(false);
  if (book.cover && !failed) {
    return (
      <img
        src={book.cover}
        alt={`${book.title} cover`}
        loading="lazy"
        onError={() => setFailed(true)}
        className={className}
      />
    );
  }
  // Fallback: gradient tile with the title — never a broken-image icon.
  return (
    <div
      className={cn('flex items-center justify-center p-2 text-center', className)}
      style={{ background: `linear-gradient(145deg, ${categoryColor(book.category)}33, #18181b)` }}
    >
      <span className="text-[10px] leading-tight font-semibold text-zinc-300 line-clamp-4">
        {book.title}
      </span>
    </div>
  );
}

function Stars({ rating }: { rating: number }) {
  return (
    <span className="text-yellow-400 text-xs" title={`${rating}/5`}>
      {'★'.repeat(Math.round(rating))}
      <span className="text-zinc-700">{'★'.repeat(5 - Math.round(rating))}</span>
    </span>
  );
}

function authorsLine(book: Book): string {
  if (book.authors.length === 0) return '';
  if (book.authors.length <= 2) return book.authors.join(' & ');
  return `${book.authors[0]} et al.`;
}

function FeaturedHero({ book }: { book: Book }) {
  const color = categoryColor(book.category);
  return (
    <div className="relative overflow-hidden rounded-xl border border-zinc-800 bg-gradient-to-br from-zinc-900 to-zinc-950">
      <div
        className="absolute inset-0 opacity-20 blur-2xl"
        style={{ background: `radial-gradient(circle at 20% 30%, ${color}, transparent 60%)` }}
      />
      <div className="relative flex flex-col sm:flex-row gap-6 p-6">
        <a
          href={book.url}
          target="_blank"
          rel="noreferrer"
          className="shrink-0 self-center sm:self-start"
        >
          <CoverImage
            book={book}
            className="w-32 h-44 sm:w-36 sm:h-52 object-cover rounded-lg shadow-2xl ring-1 ring-white/10"
          />
        </a>
        <div className="flex-1 min-w-0 space-y-3">
          <div className="flex items-center gap-2">
            <span className="text-[10px] uppercase tracking-widest font-semibold" style={{ color }}>
              ● Currently Reading
            </span>
            <StatusBadge status={book.status} />
          </div>
          <div>
            <h2 className="text-2xl font-bold leading-tight">{book.title}</h2>
            {book.edition && <span className="text-sm text-zinc-500">{book.edition}</span>}
            <p className="text-sm text-zinc-400 mt-0.5">{authorsLine(book)}</p>
          </div>

          {book.total > 0 && <div className="space-y-1.5 max-w-md">
            <div className="flex justify-between text-xs text-zinc-400">
              <span>
                {book.completed} / {book.total} {book.unit}s
              </span>
              <span className="font-semibold text-zinc-200">{book.percent}%</span>
            </div>
            <ProgressBar percent={book.percent} color={color} />
          </div>}

          {book.lastNote && (
            <div className="rounded-lg border border-zinc-800 bg-zinc-900/60 p-3 max-w-xl">
              <div className="text-[10px] uppercase tracking-wide text-zinc-500 mb-1">
                Latest note · {book.lastNote.date}
              </div>
              <p className="text-sm text-zinc-300 line-clamp-3">{book.lastNote.text}</p>
            </div>
          )}

          <div className="flex flex-wrap items-center gap-2 pt-1">
            <span
              className="text-[10px] uppercase tracking-wide px-2 py-0.5 rounded-full"
              style={{ backgroundColor: `${color}22`, color }}
            >
              {book.category}
            </span>
            {book.url && (
              <a
                href={book.url}
                target="_blank"
                rel="noreferrer"
                className="text-xs text-blue-400 hover:text-blue-300"
              >
                Open ↗
              </a>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function BookCard({ book, onOpen }: { book: Book; onOpen: (b: Book) => void }) {
  const color = categoryColor(book.category);
  return (
    <button
      onClick={() => onOpen(book)}
      className="group text-left flex gap-3 rounded-lg border border-zinc-800 bg-zinc-900 p-3 hover:border-zinc-700 hover:bg-zinc-900/80 transition-colors"
    >
      <CoverImage
        book={book}
        className="w-14 h-20 shrink-0 object-cover rounded ring-1 ring-white/5"
      />
      <div className="flex-1 min-w-0 flex flex-col">
        <div className="flex items-start justify-between gap-2">
          <h3 className="text-sm font-semibold leading-tight line-clamp-2 group-hover:text-blue-300">
            {book.title}
          </h3>
          <StatusBadge status={book.status} />
        </div>
        <p className="text-xs text-zinc-500 mt-0.5 truncate">{authorsLine(book)}</p>
        <div className="mt-auto pt-2 space-y-1">
          <div className="flex justify-between text-[10px] text-zinc-500">
            <span>{book.total > 0 ? `${book.completed}/${book.total} ${book.unit}s` : book.status === 'completed' ? 'Done' : book.status === 'reading' ? 'Reading' : 'Saved to read'}</span>
            <span>{book.rating ? <Stars rating={book.rating} /> : book.total > 0 ? `${book.percent}%` : null}</span>
          </div>
          {book.total > 0 && <ProgressBar percent={book.percent} color={color} />}
        </div>
      </div>
    </button>
  );
}

function BookModal({ book, onClose }: { book: Book; onClose: () => void }) {
  const color = categoryColor(book.category);
  const dialog = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    dialog.current?.showModal();
    const previous = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => { document.body.style.overflow = previous; };
  }, []);
  return (
    <dialog ref={dialog} aria-labelledby="book-modal-title" className="book-modal m-auto w-[calc(100%-24px)] max-w-2xl rounded-xl bg-zinc-950 text-zinc-100 backdrop:bg-black/70"
      onCancel={onClose} onClick={event => { if (event.target === event.currentTarget) onClose(); }}>

      <div
        className="relative max-h-[85dvh] w-full overflow-y-auto rounded-xl border border-zinc-800 bg-zinc-950 p-4 pt-16 sm:p-6 sm:pt-16"
        onClick={(e) => e.stopPropagation()}
      >
        <button
          onClick={onClose}
          className="absolute right-4 top-4 text-zinc-500 hover:text-white text-xl leading-none"
          aria-label="Close"
        >
          ×
        </button>
        <div className="flex flex-col gap-4 sm:flex-row">
          <CoverImage book={book} className="w-24 h-36 shrink-0 object-cover rounded-lg ring-1 ring-white/10" />
          <div className="min-w-0">
            <h2 id="book-modal-title" className="text-xl font-bold leading-tight">{book.title}</h2>
            <a className="mt-2 inline-block text-sm text-blue-400 hover:underline" href={`/manage/books?id=${encodeURIComponent(book.slug)}`}>Edit details & progress →</a>
            {book.edition && <div className="text-sm text-zinc-500">{book.edition}</div>}
            <p className="text-sm text-zinc-400 mt-1">{book.authors.join(', ')}</p>
            <div className="flex flex-wrap items-center gap-2 mt-2">
              <StatusBadge status={book.status} />
              <span
                className="text-[10px] uppercase tracking-wide px-2 py-0.5 rounded-full"
                style={{ backgroundColor: `${color}22`, color }}
              >
                {book.category}
              </span>
              {book.rating ? <Stars rating={book.rating} /> : null}
            </div>
          </div>
        </div>

        {book.total > 0 && <div className="mt-4 space-y-1.5">
          <div className="flex justify-between text-xs text-zinc-400">
            <span>
              {book.completed} / {book.total} {book.unit}s
            </span>
            <span className="font-semibold">{book.percent}%</span>
          </div>
          <ProgressBar percent={book.percent} color={color} />
        </div>}

        <div className="mt-4 grid grid-cols-2 sm:grid-cols-4 gap-2 text-xs">
          <Meta label="Started" value={book.started ?? '—'} />
          <Meta label="Finished" value={book.finished ?? '—'} />
          <Meta label="Priority" value={book.priority} />
          <Meta label="Type" value={book.type} />
        </div>

        {book.chapters.length > 0 && (
          <div className="mt-5">
            <h3 className="text-xs uppercase tracking-wide text-zinc-500 mb-2">
              {book.unit}s ({book.completed}/{book.chapters.length})
            </h3>
            <ul className="space-y-1">
              {book.chapters.map((c, i) => (
                <li
                  key={i}
                  className={cn(
                    'flex items-center gap-2 text-sm',
                    c.done ? 'text-zinc-400 line-through decoration-zinc-600' : 'text-zinc-300',
                  )}
                >
                  <span className={c.done ? 'text-green-400' : 'text-zinc-600'}>
                    {c.done ? '☑' : '☐'}
                  </span>
                  {c.label}
                </li>
              ))}
            </ul>
          </div>
        )}

        {book.log.length > 0 && (
          <div className="mt-5">
            <h3 className="text-xs uppercase tracking-wide text-zinc-500 mb-2">Log</h3>
            <div className="space-y-2">
              {book.log.map((n, i) => (
                <div key={i} className="rounded-lg border border-zinc-800 bg-zinc-900/60 p-2.5">
                  <div className="text-[10px] uppercase tracking-wide text-zinc-500 mb-0.5">
                    {n.date}
                  </div>
                  <p className="text-sm text-zinc-300">{n.text}</p>
                </div>
              ))}
            </div>
          </div>
        )}

        {book.url && (
          <a
            href={book.url}
            target="_blank"
            rel="noreferrer"
            className="mt-5 inline-block text-sm text-blue-400 hover:text-blue-300"
          >
            Open source ↗
          </a>
        )}
      </div>
    </dialog>
  );
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg bg-zinc-900 border border-zinc-800 px-2.5 py-1.5">
      <div className="text-[10px] uppercase tracking-wide text-zinc-500">{label}</div>
      <div className="text-zinc-300 capitalize truncate">{value}</div>
    </div>
  );
}

function StatCard({ label, value, accent }: { label: string; value: string; accent?: string }) {
  return (
    <div className="bg-zinc-900 rounded-lg p-3 border border-zinc-800">
      <div className="text-[10px] uppercase text-zinc-500 tracking-wide">{label}</div>
      <div className={cn('text-xl font-bold', accent ?? 'text-zinc-100')}>{value}</div>
    </div>
  );
}

interface Props {
  data: ShelfData;
}

export function BookShelf({ data }: Props) {
  const [open, setOpen] = useState<Book | null>(null);
  const [filter, setFilter] = useState<string>('all');
  const [activeType, setActiveType] = useState<BookType>('book');
  const shelf = useMemo(() => buildShelf(data.all.filter(book => book.type === activeType)), [data.all, activeType]);

  const visibleGroups = useMemo(
    () => (filter === 'all' ? shelf.groups : shelf.groups.filter((g) => g.category === filter)),
    [shelf.groups, filter],
  );

  const { stats } = shelf;
  const activeTab = SHELF_TABS[activeType];
  function selectTab(type: BookType) {
    setActiveType(type);
    setFilter('all');
    setOpen(null);
  }

  return (
    <div className="space-y-6">
      <div role="tablist" aria-label="Bookshelf types" className="flex flex-wrap gap-2 border-b border-zinc-800 pb-3">
        {SHELF_ORDER.map((type, index) => {
          const count = data.all.filter(book => book.type === type).length;
          return <button key={type} id={`shelf-tab-${type}`} type="button" role="tab" aria-controls="shelf-panel" aria-selected={activeType === type} tabIndex={activeType === type ? 0 : -1} onClick={() => selectTab(type)} onKeyDown={event => {
            const next = event.key === 'ArrowRight' ? SHELF_ORDER[(index + 1) % SHELF_ORDER.length] : event.key === 'ArrowLeft' ? SHELF_ORDER[(index - 1 + SHELF_ORDER.length) % SHELF_ORDER.length] : event.key === 'Home' ? SHELF_ORDER[0] : event.key === 'End' ? SHELF_ORDER[SHELF_ORDER.length - 1] : null;
            if (next) { event.preventDefault(); selectTab(next); document.getElementById(`shelf-tab-${next}`)?.focus(); }
          }} className={cn('rounded-lg px-4 py-2 text-sm font-medium focus-visible:outline-2 focus-visible:outline-blue-400', activeType === type ? 'bg-blue-600 text-white' : 'text-zinc-400 hover:bg-zinc-800 hover:text-white')}>{SHELF_TABS[type].label} <span className="ml-1 opacity-70">{count}</span></button>;
        })}
      </div>
      <div id="shelf-panel" role="tabpanel" aria-labelledby={`shelf-tab-${activeType}`} className="space-y-6">
      {shelf.all.length === 0 && <p className="rounded-lg border border-dashed border-zinc-700 p-6 text-sm text-zinc-400">No {activeTab.label.toLowerCase()} yet. <a className="text-blue-400 hover:underline" href="/manage/books">Add a {activeTab.singular} →</a></p>}
      {shelf.featured && <a className="inline-block text-sm text-blue-400 hover:underline" href={`/manage/books?id=${encodeURIComponent(shelf.featured.slug)}`}>Update {shelf.featured.title} →</a>}
      {/* Featured hero */}
      {shelf.featured && <FeaturedHero book={shelf.featured} />}

      {/* Stats */}
      {shelf.all.length > 0 && <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <StatCard label={activeTab.label} value={shelf.all.length.toString()} accent="text-blue-400" />
        <StatCard label="Reading now" value={stats.reading.toString()} accent="text-yellow-400" />
        <StatCard label="Completed" value={stats.completed.toString()} accent="text-green-400" />
        {stats.unitsTotal > 0 && <StatCard
          label="Units read"
          value={`${stats.unitsDone}/${stats.unitsTotal}`}
          accent="text-purple-400"
        />}
      </div>}

      {/* Category distribution bar */}
      {shelf.all.length > 0 && <div className="bg-zinc-900 rounded-lg p-4 border border-zinc-800">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-semibold text-zinc-300">By category</h2>
          {stats.unitsTotal > 0 && <span className="text-xs text-zinc-500">{stats.avgCompletion}% avg completion</span>}
        </div>
        <div className="flex flex-wrap gap-2">
          {stats.byCategory.map((c) => (
            <button
              key={c.category}
              onClick={() => setFilter(filter === c.category ? 'all' : c.category)}
              className={cn(
                'flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-full border transition-colors',
                filter === c.category
                  ? 'border-zinc-500 bg-zinc-800'
                  : 'border-zinc-800 hover:border-zinc-700',
              )}
            >
              <span className="w-2 h-2 rounded-full" style={{ backgroundColor: c.color }} />
              <span className="text-zinc-300">{c.category}</span>
              <span className="text-zinc-500">
                {c.done}/{c.count}
              </span>
            </button>
          ))}
          {filter !== 'all' && (
            <button
              onClick={() => setFilter('all')}
              className="text-xs px-2.5 py-1 rounded-full border border-zinc-800 text-zinc-400 hover:text-white"
            >
              clear ×
            </button>
          )}
        </div>
      </div>}

      {/* Currently reading row (excludes the hero) */}
      {filter === 'all' && shelf.currentlyReading.length > 0 && (
        <section>
          <h2 className="text-lg font-semibold mb-3">
            Also reading{' '}
            <span className="text-xs text-zinc-500 font-normal">
              ({shelf.currentlyReading.length})
            </span>
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {shelf.currentlyReading.map((b) => (
              <BookCard key={b.slug} book={b} onOpen={setOpen} />
            ))}
          </div>
        </section>
      )}

      {/* Category groups */}
      {visibleGroups.map((g) => (
        <section key={g.category}>
          <h2 className="text-lg font-semibold mb-3 flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: g.color }} />
            {g.category}
            <span className="text-xs text-zinc-500 font-normal">({g.books.length})</span>
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {g.books.map((b) => (
              <BookCard key={b.slug} book={b} onOpen={setOpen} />
            ))}
          </div>
        </section>
      ))}

      {open && <BookModal book={open} onClose={() => setOpen(null)} />}
      </div>
    </div>
  );
}
