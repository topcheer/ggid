/**
 * Centralized API configuration.
 * Reads from environment variables with safe defaults.
 */

export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL ||
  "";

export const DEFAULT_TENANT_ID =
  process.env.NEXT_PUBLIC_TENANT_ID ||
  "";

/**
 * Extract tenant slug from subdomain.
 * Pattern: <slug>.ggid-console.iot2.win → slug
 * - default.ggid-console.iot2.win → "default"
 * - acme.ggid-console.iot2.win → "acme"
 * - ggid-console.iot2.win (no subdomain) → "" (use default)
 * - localhost → "" (development)
 */
export function getTenantSlugFromSubdomain(): string {
  if (typeof window === "undefined") return "";
  const hostname = window.location.hostname;
  
  // localhost or IP — no subdomain
  if (hostname === "localhost" || /^\d+\.\d+\.\d+\.\d+$/.test(hostname)) {
    return "";
  }
  
  const parts = hostname.split(".");
  // *.ggid-console.iot2.win → parts = ["slug", "ggid-console", "iot2", "win"]
  // ggid-console.iot2.win → parts = ["ggid-console", "iot2", "win"]
  if (parts.length >= 4 && parts[1] === "ggid-console") {
    const slug = parts[0];
    // Skip "www" and exact domain match
    if (slug === "www" || slug === "ggid-console") return "";
    return slug;
  }
  return "";
}

/**
 * Get the effective tenant slug.
 * Priority: URL ?tenant= param > subdomain > "default" (for non-subdomain access)
 */
export function getEffectiveTenantSlug(): string {
  if (typeof window === "undefined") return "default";
  // URL param override
  const params = new URLSearchParams(window.location.search);
  const urlTenant = params.get("tenant");
  if (urlTenant) return urlTenant;
  
  // Subdomain
  const subdomain = getTenantSlugFromSubdomain();
  if (subdomain) return subdomain;
  
  // No subdomain → default tenant
  return "default";
}

/**
 * Resolve tenant slug to UUID via API.
 * Returns null if slug not found or API unavailable.
 */
let cachedTenantResolve: { slug: string; id: string; ts: number } | null = null;

export async function resolveTenantSlug(slug: string): Promise<string | null> {
  if (!slug || slug === "default") return DEFAULT_TENANT_ID;
  
  // Cache for 5 minutes
  if (cachedTenantResolve && cachedTenantResolve.slug === slug && Date.now() - cachedTenantResolve.ts < 300000) {
    return cachedTenantResolve.id;
  }
  
  try {
    const resp = await fetch(`${API_BASE_URL}/api/v1/tenants/resolve?slug=${encodeURIComponent(slug)}`, {
      headers: { "Content-Type": "application/json" },
    });
    if (!resp.ok) return null;
    const data = await resp.json();
    const id = data.id || data.tenant_id || data.tenantId;
    if (id) {
      cachedTenantResolve = { slug, id, ts: Date.now() };
      return id;
    }
    return null;
  } catch {
    return null;
  }
}

export function buildUrl(path: string): string {
  if (path.startsWith("http")) return path;
  return `${API_BASE_URL}${path}`;
}

/**
 * Health check hook — polls API healthz endpoint.
 */
export interface HealthResult {
  online: boolean;
  latencyMs: number | null;
}

export async function checkApiHealth(): Promise<boolean> {
  const result = await checkApiHealthDetailed();
  return result.online;
}

export async function checkApiHealthDetailed(): Promise<HealthResult> {
  try {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 5000);
    const start = performance.now();
    const resp = await fetch(`${API_BASE_URL}/healthz`, {
      signal: controller.signal,
      headers: { "X-Tenant-ID": DEFAULT_TENANT_ID },
    });
    clearTimeout(timeout);
    const latencyMs = Math.round(performance.now() - start);
    return { online: resp.ok, latencyMs };
  } catch {
    return { online: false, latencyMs: null };
  }
}
