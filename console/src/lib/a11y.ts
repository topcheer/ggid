/**
 * Accessibility (a11y) utilities for common patterns.
 *
 * Usage:
 *   import { ariaLabel, ariaProps } from "@/lib/a11y";
 *   <button aria-label={ariaLabel("Close", t)} {...ariaProps.button}>...</button>
 */

/**
 * Returns the aria-label string, translating if t function is provided.
 */
export function ariaLabel(key: string, t?: (key: string) => string): string {
  if (t) return t(key);
  return key;
}

/**
 * Common aria attribute sets for different element types.
 */
export const ariaProps = {
  /** For icon-only buttons */
  iconButton: (label: string) => ({
    "aria-label": label,
    role: "button" as const,
  }),

  /** For toggle switches */
  switch: (checked: boolean, label: string) => ({
    role: "switch" as const,
    "aria-checked": checked,
    "aria-label": label,
    tabIndex: 0,
  }),

  /** For tab navigation */
  tab: (active: boolean, id: string) => ({
    role: "tab" as const,
    "aria-selected": active,
    "aria-controls": `${id}-panel`,
    id: `${id}-tab`,
    tabIndex: active ? 0 : -1,
  }),

  /** For tab panel */
  tabPanel: (id: string) => ({
    role: "tabpanel" as const,
    id: `${id}-panel`,
    "aria-labelledby": `${id}-tab`,
  }),

  /** For form fields */
  field: (label: string, required?: boolean) => ({
    "aria-label": label,
    "aria-required": required || undefined,
  }),

  /** For loading states */
  loading: (isLoading: boolean) => ({
    "aria-busy": isLoading,
    "aria-live": "polite" as const,
  }),

  /** For live regions (toasts, alerts) */
  live: () => ({
    "aria-live": "polite" as const,
    "aria-atomic": true,
  }),

  /** For modal dialogs */
  modal: (labelledBy: string) => ({
    role: "dialog" as const,
    "aria-modal": true,
    "aria-labelledby": labelledBy,
  }),

  /** For expandable sections */
  expandable: (expanded: boolean, controlId?: string) => ({
    "aria-expanded": expanded,
    "aria-controls": controlId,
  }),
};

/**
 * Keyboard handler for activating elements with Enter/Space.
 * Usage: onKeyDown={onEnterKey(onClick)}
 */
export function onEnterKey(fn: () => void) {
  return (e: React.KeyboardEvent) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      fn();
    }
  };
}
