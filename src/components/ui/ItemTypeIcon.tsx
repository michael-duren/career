import { BookIcon, CourseIcon, PaperIcon, type IconProps } from './icons';

const ICONS = { book: BookIcon, paper: PaperIcon, course: CourseIcon } as const;
export type ItemType = keyof typeof ICONS;

// ItemTypeIcon: the line icon for a bookshelf item type. Use it wherever a book, paper or
// course is listed (Bookshelf, search results) so the type reads the same everywhere.
export function ItemTypeIcon({ type, ...props }: IconProps & { type: ItemType }) {
  const Icon = ICONS[type] ?? BookIcon;
  return <Icon {...props} />;
}
