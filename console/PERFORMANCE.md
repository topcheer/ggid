# GGID Console Performance Report

## Bundle Size Analysis

| Metric | Value | Notes |
|--------|-------|-------|
| Total static | 14 MB | All JS chunks |
| Total chunks | 853 | Turbopack output |
| Largest chunk | 3.3 MB | Contains i18n dicts (15 languages) |
| 2nd largest | 368 KB | Shared components |
| 3rd largest | 368 KB | React framework |

## Top 5 Chunks

| Rank | Size | Content | Optimization |
|------|------|---------|-------------|
| 1 | 3.3 MB | i18n-dicts.ts (15 languages × ~2000 keys each) | Load language on demand (see below) |
| 2 | 368 KB | Shared UI components (sidebar, tables, forms) | Already optimized |
| 3 | 368 KB | React + Next.js runtime | Framework minimum |
| 4 | 276 KB | Auth + API client modules | Acceptable |
| 5 | 224 KB | Settings hub + page modules | Acceptable |

## Optimization Status

### Done ✅
- `experimental.optimizePackageImports: ['lucide-react']` — tree-shakes 694 icon imports
- `output: 'standalone'` — minimal Docker image
- Per-route code splitting (Turbopack automatic)
- No custom fonts loaded (system font stack)
- No unoptimized images (2 files use next/image)

### Opportunities

1. **Lazy-load i18n dicts (biggest win: -2.5MB initial load)**
   - Currently all 15 languages bundled in one chunk
   - Fix: dynamic `import()` based on `locale` state
   - Would reduce initial JS from ~3.3MB to ~500KB

2. **Reduce icon imports**
   - 694 files import from lucide-react
   - optimizePackageImports helps but barrel imports still pulled
   - Fix: use specific imports `import { Users } from "lucide-react/Users"`

3. **Route-level prefetch tuning**
   - Next.js prefetches linked routes aggressively
   - Fix: add `prefetch={false}` to sidebar links for non-critical pages

## First Load Estimates

| Page | Estimated JS | Notes |
|------|-------------|-------|
| /login | ~400 KB | Shared + i18n |
| /dashboard | ~600 KB | Shared + charts |
| /users | ~500 KB | Shared + table |
| /settings | ~500 KB | Shared + card grid |
| /docs | ~450 KB | Shared + code blocks |

## Recommendations

1. **Priority 1**: Lazy-load i18n dicts (biggest single optimization)
2. **Priority 2**: Monitor real user metrics (usePagePerformance hook already in place)
3. **Priority 3**: Consider webpack instead of Turbopack for production (better tree-shaking)

## Monitoring

- `lib/performance.ts` tracks TTFB, DOMContentLoaded, loadComplete per page
- Data stored in sessionStorage, ready for analytics export
- `usePagePerformance("PageName")` hook on key pages
