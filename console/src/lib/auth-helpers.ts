/**
 * Shared auth helpers for client-side API calls.
 *
 * On SSR/prerender, localStorage is unavailable — these helpers
 * safely return null/empty so fetch calls don't crash.
 */

export function getAuthToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("ggid_access_token");
}

/**
 * Build Authorization header object for fetch calls.
 * Returns empty object if no token (SSR or not logged in).
 *
 * Usage:
 *   const res = await fetch("/api/v1/...", {
 *     headers: { ...authHeader(), "Content-Type": "application/json" }
 *   });
 */
export function authHeader(): Record<string, string> {
  const token = getAuthToken();
  const headers: Record<string, string> = {};
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }
  // X-User-ID and X-Tenant-ID are needed by services that read identity
  // from headers (e.g., identity service /users/me).
  if (typeof window !== "undefined") {
    const uid = localStorage.getItem("ggid_user_id");
    if (uid) headers["X-User-ID"] = uid;
    const tid = localStorage.getItem("ggid_tenant_id");
    if (tid) headers["X-Tenant-ID"] = tid;
  }
  return headers;
}

/**
 * Check if we can make authenticated requests.
 * Use to guard fetch calls in useEffect.
 *
 * Usage:
 *   useEffect(() => {
 *     if (!isAuthenticated()) return;
 *     loadData();
 *   }, []);
 */
export function isAuthenticated(): boolean {
  return getAuthToken() !== null;
}
