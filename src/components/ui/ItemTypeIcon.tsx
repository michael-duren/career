import type { BookType } from '../../lib/books';
import { BookIcon, CourseIcon, PaperIcon, type IconProps } from './icons';

export type ItemType = BookType;
const ICONS: Record<ItemType, typeof BookIcon> = { book: BookIcon, paper: PaperIcon, course: CourseIcon };

// ItemTypeIcon: the line icon for a bookshelf item type. Use it wherever a book, paper or
// course is listed in React. Global search draws the same paths (public/ui/item-type-icons.js).
export function ItemTypeIcon({ type, ...props }: IconProps & { type: ItemType }) {
  const Icon = ICONS[type] ?? BookIcon;
  return <Icon {...props} />;
}
