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
  return token ? { Authorization: `Bearer ${token}` } : {};
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
