/**
 * Centralized API configuration.
 * Reads from environment variables with safe defaults.
 */

export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL ||
  process.env.NEXT_PUBLIC_GGID_API ||
  "http://localhost:8080";

export const DEFAULT_TENANT_ID =
  process.env.NEXT_PUBLIC_TENANT_ID ||
  "00000000-0000-0000-0000-000000000001";

export function buildUrl(path: string): string {
  if (path.startsWith("http")) return path;
  return `${API_BASE_URL}${path}`;
}

/**
 * Health check hook — polls API healthz endpoint.
 */
export async function checkApiHealth(): Promise<boolean> {
  try {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 3000);
    const resp = await fetch(`${API_BASE_URL}/healthz`, {
      signal: controller.signal,
      headers: { "X-Tenant-ID": DEFAULT_TENANT_ID },
    });
    clearTimeout(timeout);
    return resp.ok;
  } catch {
    return false;
  }
}
