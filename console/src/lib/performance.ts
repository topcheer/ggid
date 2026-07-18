/**
 * Performance monitoring utilities for the Console.
 *
 * Usage:
 *   import { trackPageLoad, usePagePerformance } from "@/lib/performance";
 *
 *   // In any page component:
 *   usePagePerformance("Dashboard");
 */

import { useEffect } from "react";

/**
 * Track page load timing using the Performance API.
 * Logs to console in dev, can be sent to analytics in prod.
 */
export function trackPageLoad(pageName: string) {
  if (typeof window === "undefined") return;

  const nav = performance.getEntriesByType("navigation")[0] as PerformanceNavigationTiming | undefined;
  if (!nav) return;

  const metrics = {
    page: pageName,
    domContentLoaded: Math.round(nav.domContentLoadedEventEnd - nav.startTime),
    loadComplete: Math.round(nav.loadEventEnd - nav.startTime),
    ttfb: Math.round(nav.responseStart - nav.startTime),
    domInteractive: Math.round(nav.domInteractive - nav.startTime),
    transferSize: nav.transferSize,
  };

  if (process.env.NODE_ENV === "development") {
    console.debug(`[perf] ${pageName}:`, metrics);
  }

  // Store for potential analytics export
  const stored = JSON.parse(sessionStorage.getItem("ggid_perf") || "[]");
  stored.push({ ...metrics, timestamp: Date.now() });
  // Keep last 50 entries
  if (stored.length > 50) stored.shift();
  sessionStorage.setItem("ggid_perf", JSON.stringify(stored));
}

/**
 * Hook to track page performance on mount.
 */
export function usePagePerformance(pageName: string) {
  useEffect(() => {
    // Defer to next tick to ensure load is complete
    const timer = setTimeout(() => trackPageLoad(pageName), 0);
    return () => clearTimeout(timer);
  }, [pageName]);
}

/**
 * Get the bundle size estimate from the performance observer.
 */
export function getBundleMetrics(): { totalJS: number; chunkCount: number } | null {
  if (typeof window === "undefined") return null;

  const resources = performance.getEntriesByType("resource");
  const jsResources = resources.filter((r) => r.name.endsWith(".js"));

  return {
    totalJS: jsResources.reduce((sum, r) => sum + (r as PerformanceResourceTiming).transferSize, 0),
    chunkCount: jsResources.length,
  };
}

/**
 * Clear performance data.
 */
export function clearPerfData() {
  sessionStorage.removeItem("ggid_perf");
}
