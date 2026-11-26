// Architect drawing primitives (docs/redesign/design-system.md, "Drawing primitives").
// Styles live in public/ui/primitives.css; Go templ pages use the same classes.
export { Avatar } from './Avatar';
export { Box } from './Box';
export { Button, IconButton } from './Button';
export { Checkbox } from './Checkbox';
export { GoalPattern, HatchFill, Progress } from './HatchFill';
export * from './icons';
export { Label } from './Label';
export { LogoSpot } from './LogoSpot';
export { MeasurementLine, type MeasurementState } from './MeasurementLine';
export { NowLine, TodayMarker } from './NowLine';
export { SearchField } from './SearchField';
export { SectionHeader } from './SectionHeader';
export { SegmentedTabs, type SegmentedTab } from './SegmentedTabs';
export { Switch } from './Switch';
export { TickScale } from './TickScale';
export { buttonClassName, goalPattern } from '../../lib/ui';
