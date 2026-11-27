import type { ReactNode, SVGProps } from 'react';
import { cn } from '../../lib/utils';

// Thin line icons (1px stroke, currentColor), never filled. Decorative (aria-hidden): the
// control around an icon carries the accessible name (IconButton's label, for example).
export type IconProps = Omit<SVGProps<SVGSVGElement>, 'children'> & { size?: number };

function icon(name: string, paths: ReactNode, viewBox = 16) {
  function Icon({ size = 16, className, ...props }: IconProps) {
    return (
      <svg
        width={size}
        height={size}
        viewBox={`0 0 ${viewBox} ${viewBox}`}
        aria-hidden="true"
        focusable="false"
        className={cn('ui-icon', className)}
        {...props}
      >
        {paths}
      </svg>
    );
  }
  Icon.displayName = `${name}Icon`;
  return Icon;
}

export const SearchIcon = icon('Search', <><circle cx="6" cy="6" r="4.5" /><path d="M9.5 9.5 L13 13" /></>, 14);
export const SettingsIcon = icon('Settings', <><circle cx="8" cy="8" r="2.5" /><path d="M8 1 V3.5 M8 12.5 V15 M1 8 H3.5 M12.5 8 H15 M3 3 L4.8 4.8 M11.2 11.2 L13 13 M3 13 L4.8 11.2 M11.2 4.8 L13 3" /></>);
export const PlusIcon = icon('Plus', <path d="M8 2 V14 M2 8 H14" />);
export const CloseIcon = icon('Close', <path d="M3 3 L13 13 M13 3 L3 13" />);
export const EditIcon = icon('Edit', <path d="M10.5 2.5 L13.5 5.5 L5.5 13.5 H2.5 V10.5 Z M9 4 L12 7" />);
export const DeleteIcon = icon('Delete', <path d="M2.5 4 H13.5 M6 4 V2.5 H10 V4 M4 4 L4.8 13.5 H11.2 L12 4 M6.5 6.5 V11 M9.5 6.5 V11" />);
export const CheckIcon = icon('Check', <path d="M2.5 8.5 L6.5 12 L13.5 3.5" />);
export const ChevronLeftIcon = icon('ChevronLeft', <path d="M10 3 L5 8 L10 13" />);
export const ChevronRightIcon = icon('ChevronRight', <path d="M6 3 L11 8 L6 13" />);
export const ChevronDownIcon = icon('ChevronDown', <path d="M3 6 L8 11 L13 6" />);
export const ArrowRightIcon = icon('ArrowRight', <path d="M2 8 H14 M10 4 L14 8 L10 12" />);
export const ExternalIcon = icon('External', <path d="M9 2.5 H13.5 V7 M13.5 2.5 L7 9 M11.5 9.5 V13.5 H2.5 V4.5 H6.5" />);
export const MenuIcon = icon('Menu', <path d="M2 4 H14 M2 8 H14 M2 12 H14" />);
export const HomeIcon = icon('Home', <path d="M2 7.5 L8 2.5 L14 7.5 M3.5 6.5 V13.5 H12.5 V6.5" />);
export const CalendarIcon = icon('Calendar', <><rect x="2" y="3.5" width="12" height="10" /><path d="M2 6.5 H14 M5 2 V5 M11 2 V5" /></>);
export const ClockIcon = icon('Clock', <><circle cx="8" cy="8" r="6" /><path d="M8 4.5 V8 L10.5 9.5" /></>);
export const MicIcon = icon('Mic', <><rect x="6" y="2" width="4" height="7.5" rx="2" /><path d="M3.5 8 A4.5 4.5 0 0 0 12.5 8 M8 12.5 V14.5" /></>);
export const LinkIcon = icon('Link', <path d="M6.5 9.5 L9.5 6.5 M7 4.5 L8.5 3 A2.5 2.5 0 0 1 13 6.5 L11.5 8 M9 11.5 L7.5 13 A2.5 2.5 0 0 1 3 9.5 L4.5 8" />);
// Skipped: a circle with a slash (drawn in --accent by the caller).
export const SkippedIcon = icon('Skipped', <><circle cx="8" cy="8" r="7" /><path d="M3 13 L13 3" /></>);

// Bookshelf item types (book, paper, course). Used by the Bookshelf page and meant for search
// results too (global search, #172), so the same item reads the same everywhere. Drawn on an 18-unit grid (16x18 marks, centred).
export const BookIcon = icon('Book', <><rect x="2" y="1" width="14" height="16" /><path d="M5 1 V17" /></>, 18);
export const PaperIcon = icon('Paper', <path d="M2 1 H11 L16 6 V17 H2 Z M11 1 V6 H16" />, 18);
export const CourseIcon = icon('Course', <><rect x="2" y="3" width="14" height="12" /><path d="M7 6 L12 9 L7 12 Z" /></>, 18);
