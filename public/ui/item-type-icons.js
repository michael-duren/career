/*
 * Line drawings for the bookshelf item types (book, paper, course), shared by the React icons
 * (src/components/ui/icons.tsx: BookIcon, PaperIcon, CourseIcon) and global search results
 * (public/ui/site-search.js), so an item reads the same everywhere. One path each on an
 * 18-unit grid (16x18 marks, centred), stroked in currentColor and never filled.
 */

/** viewBox size the paths are drawn on. */
export const ITEM_TYPE_VIEWBOX = 18;

/** Path data per item type. */
export const ITEM_TYPE_PATHS = {
  book: 'M2 1 H16 V17 H2 Z M5 1 V17',
  paper: 'M2 1 H11 L16 6 V17 H2 Z M11 1 V6 H16',
  course: 'M2 3 H16 V15 H2 Z M7 6 L12 9 L7 12 Z',
};
