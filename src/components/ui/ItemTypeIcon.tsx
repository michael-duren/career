import type { BookType } from '../../lib/books';
import { BookIcon, CourseIcon, PaperIcon, type IconProps } from './icons';

export type ItemType = BookType;
const ICONS: Record<ItemType, typeof BookIcon> = { book: BookIcon, paper: PaperIcon, course: CourseIcon };

// ItemTypeIcon: the line icon for a bookshelf item type. Use it wherever a book, paper or
// course is listed (Bookshelf now; search results once they carry the type, #172).
export function ItemTypeIcon({ type, ...props }: IconProps & { type: ItemType }) {
  const Icon = ICONS[type] ?? BookIcon;
  return <Icon {...props} />;
}
