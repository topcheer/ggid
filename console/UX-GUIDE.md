# GGID Console UX Guide

## Design Principles

1. **Consistency** — Same patterns across all pages (cards, tables, forms)
2. **Feedback** — Every action shows loading → success/error state
3. **Safety** — Destructive operations require confirmation (ConfirmDialog)
4. **Accessibility** — ARIA labels, keyboard navigation, dark mode support

## Component Library

| Component | Location | Usage |
|-----------|----------|-------|
| EmptyState | `components/EmptyState.tsx` | No-data states with icon + CTA |
| LoadingState | `components/EmptyState.tsx` | Spinner during data fetch |
| ErrorState | `components/EmptyState.tsx` | Error with retry button |
| Toast | `components/Toast.tsx` | Success/error notifications (useToast hook) |
| ConfirmDialog | `components/ConfirmDialog.tsx` | Destructive action confirmation (useConfirm hook) |
| Pagination | `components/Pagination.tsx` | List page pagination |
| a11y | `lib/a11y.ts` | ariaLabel, ariaProps helpers |

## Page Structure

```
<div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
  <div className="max-w-5xl mx-auto">
    {/* Header with icon + title */}
    {/* Tab bar (if multi-tab) */}
    {/* Content cards */}
  </div>
</div>
```

## Color System

- Primary: `blue-600` / gradient `from-blue-600 to-purple-600`
- Success: `green-500/600`
- Warning: `yellow/amber-500`
- Danger: `red-500/600`
- Background: `gray-50` (light) / `gray-950` (dark)
- Cards: `white` (light) / `gray-900` (dark)

## i18n

- All UI strings via `t("key")` — never hardcode
- Keys in `messages/en.json` → flattened to `i18n-dicts.ts`
- Locales: en, zh, zh-TW, es, hi, fr, ar, pt, ru, de, ja, ko, tr, vi, id

## Form Patterns

1. **Validation**: inline error below field
2. **Submit**: disabled during loading
3. **Success**: toast notification + auto-dismiss (3s)
4. **Error**: toast with details + manual dismiss

## List Page Patterns

1. Search bar (top, full width)
2. Filter dropdowns (right-aligned)
3. Table with sortable headers
4. Pagination at bottom
5. Empty state when no results

## Dark Mode

- Use Tailwind `dark:` variants
- Test in both light/dark
- Dark-first pages use `bg-gray-950` base

## Mobile

- Hamburger menu for sidebar (md:hidden)
- Tables scroll horizontally (overflow-x-auto)
- Grid: 1 col mobile → 2-3 tablet → 4 desktop
