import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

// Focus ring utility
export const focusRing = cn(
  "focus-visible:outline-none focus-visible:ring-2",
  "focus-visible:ring-ring focus-visible:ring-offset-2",
);

// Disabled utility
export const disabled = "disabled:pointer-events-none disabled:opacity-50";