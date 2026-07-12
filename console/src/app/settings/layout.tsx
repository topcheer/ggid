// Force all settings pages to be dynamic (not statically prerendered)
// This prevents build-time fetch errors on pages with useEffect API calls
export const dynamic = 'force-dynamic';