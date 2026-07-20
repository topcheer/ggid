"use client";

import { useEffect, useState, useCallback } from "react";
import { API_BASE_URL, DEFAULT_TENANT_ID } from "./api-config";

const API_BASE = API_BASE_URL;

/** Get tenant ID from localStorage (set by subdomain resolver) or default */
function getTenantId(): string {
  if (typeof window !== "undefined") {
    return localStorage.getItem("ggid_tenant_id") || DEFAULT_TENANT_ID;
  }
  return DEFAULT_TENANT_ID;
}

const TENANT_ID = typeof window !== "undefined" ? getTenantId() : DEFAULT_TENANT_ID;

// Structured error from the GGID API
export interface ApiError extends Error {
  status: number;
  title: string;
  detail: string;
  requestId: string | null;
  code: string | null;
}

export function parseApiError(status: number, body: string): ApiError {
  let title = "Request Failed";
  let detail = body;
  let requestId: string | null = null;
  let code: string | null = null;

  // Try to parse structured error response
  try {
    const parsed = JSON.parse(body);
    title = parsed.error?.title || parsed.title || parsed.error?.code || title;
    detail = parsed.error?.detail || parsed.detail || parsed.error?.message || parsed.message || detail;
    requestId = parsed.request_id || parsed.error?.request_id || null;
    code = parsed.error?.code || parsed.code || null;
  } catch {
    // Not JSON — use raw text if short, else generic
    if (body.length > 200) detail = "Internal server error";
  }

  // Human-friendly status messages
  const statusMap: Record<number, string> = {
    400: "Bad Request",
    401: "Unauthorized",
    403: "Forbidden",
    404: "Not Found",
    409: "Conflict",
    422: "Validation Error",
    429: "Too Many Requests",
    500: "Internal Server Error",
    502: "Bad Gateway",
    503: "Service Unavailable",
  };
  if (title === "Request Failed" && statusMap[status]) {
    title = statusMap[status];
  }

  const err = new Error(`${title}: ${detail}`) as ApiError;
  err.status = status;
  err.title = title;
  err.detail = detail;
  err.requestId = requestId;
  err.code = code;
  return err;
}

export interface User {
  id: string;
  tenant_id: string;
  username: string;
  email: string;
  phone: string;
  status: string;
  email_verified: boolean;
  display_name: string;
  locale: string;
  timezone: string;
  created_at: string;
  updated_at: string;
}

export interface PageResult<T> {
  items?: T[];
  users?: T[];
  total?: number;
  total_count?: number;
}

// Get JWT from localStorage (set by login page)
function getAuthToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("ggid_access_token");
}

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    "X-Tenant-ID": TENANT_ID,
  };

  // Attach JWT if available
  const token = getAuthToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  if (options?.headers) {
    Object.assign(headers, options.headers);
  }

  const resp = await fetch(`${API_BASE}${path}`, { ...options, headers });

  if (resp.status === 401 && typeof window !== "undefined" && !path.includes("/auth/")) {
    // Try to refresh the token before giving up
    const refreshToken = localStorage.getItem("ggid_refresh_token");
    if (refreshToken) {
      try {
        const refreshResp = await fetch(`${API_BASE}/api/v1/auth/refresh`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ refresh_token: refreshToken }),
        });
        if (refreshResp.ok) {
          const tokens = await refreshResp.json();
          if (tokens.access_token) {
            localStorage.setItem("ggid_access_token", tokens.access_token);
            if (tokens.refresh_token) localStorage.setItem("ggid_refresh_token", tokens.refresh_token);
            // Update scopes from refreshed token
            try {
              const payload = JSON.parse(atob(tokens.access_token.split(".")[1]));
              const newScopes = payload.scopes || payload.roles || ["user"];
              localStorage.setItem("ggid_user_scopes", JSON.stringify(Array.isArray(newScopes) ? newScopes : [newScopes]));
            } catch {}
            // Retry the original request with the new token
            const retryHeaders = { ...headers };
            retryHeaders["Authorization"] = `Bearer ${tokens.access_token}`;
            const retryResp = await fetch(`${API_BASE}${path}`, { ...options, headers: retryHeaders });
            if (retryResp.status === 401) {
              // Still 401 after refresh — give up
              window.dispatchEvent(new CustomEvent("ggid:unauthorized"));
              throw parseApiError(401, "{\"detail\":\"Session expired\"}");
            }
            if (!retryResp.ok) {
              const text = await retryResp.text();
              throw parseApiError(retryResp.status, text);
            }
            if (retryResp.status === 204) return {} as T;
            return await retryResp.json() as T;
          }
        }
      } catch {
        // Refresh failed — fall through to logout
      }
    }
    // No refresh token or refresh failed — emit unauthorized
    window.dispatchEvent(new CustomEvent("ggid:unauthorized"));
    throw parseApiError(401, "{\"detail\":\"Session expired\"}");
  }

  if (!resp.ok) {
    const text = await resp.text();
    throw parseApiError(resp.status, text);
  }

  if (resp.status === 204) return {} as T;
  return resp.json();
}

export function useUsers() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<PageResult<User>>("/api/v1/users");
      setUsers(data.users || data.items || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load users");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { users, loading, error, refresh };
}

export function useApi() {
  return { apiFetch, API_BASE, TENANT_ID };
}

// Check if user is authenticated
export type UserRole = "user" | "tenant_admin" | "platform_admin";

export function getUserScopes(): string[] {
  if (typeof window === "undefined") return ["user"];
  try {
    const raw = localStorage.getItem("ggid_user_scopes");
    return raw ? JSON.parse(raw) : ["user"];
  } catch {
    return ["user"];
  }
}

export function getUserRole(): UserRole {
  const scopes = getUserScopes();
  if (scopes.includes("platform:admin") || scopes.includes("admin")) return "platform_admin";
  if (scopes.includes("tenant:admin") || scopes.includes("manager")) return "tenant_admin";
  return "user";
}

export function useUserRole(): { role: UserRole; scopes: string[]; isPlatformAdmin: boolean; isTenantAdmin: boolean; isAdmin: boolean } {
  // Sync-read from localStorage so first render has correct scopes (no redirect flash)
  const [scopes, setScopes] = useState<string[]>(() => getUserScopes());

  const hasScope = (s: string) => scopes.includes(s);
  // Normalize scopes for matching: handle both role keys (e.g. "platform:admin")
  // and display names (e.g. "Platform Administrator") returned by some auth configs
  const lowerScopes = scopes.map((s) => s.toLowerCase());
  const hasRole = (...keys: string[]) => keys.some((k) => {
    if (scopes.includes(k)) return true;
    // Also check normalized forms: "platform:admin" matches "Platform Administrator"
    const normalized = k.replace(/[:_]/g, " ").toLowerCase();
    return lowerScopes.some((ls) => ls === normalized || ls === k.toLowerCase() || ls.includes(k.split(":").pop()!.toLowerCase()));
  });
  const role: UserRole = hasRole("platform:admin", "admin")
    ? "platform_admin"
    : hasRole("tenant:admin", "manager")
    ? "tenant_admin"
    : "user";

  return {
    role,
    scopes,
    isPlatformAdmin: role === "platform_admin",
    isTenantAdmin: role === "tenant_admin" || role === "platform_admin",
    isAdmin: role === "platform_admin",
  };
}

// ===== Dynamic Permission System =====

export async function fetchUserPermissions(): Promise<string[] | null> {
  try {
    const token = getAuthToken();
    if (!token) return null;
    const res = await fetch(`${API_BASE_URL}/api/v1/me/permissions`, {
      headers: { Authorization: `Bearer ${token}`, "X-Tenant-ID": DEFAULT_TENANT_ID },
    });
    if (!res.ok) return null;
    const data = await res.json();
    const perms = data.permissions || data.items || data;
    return Array.isArray(perms) ? perms : null;
  } catch {
    return null;
  }
}

export function getUserPermissions(): string[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem("ggid_user_permissions");
    return raw ? JSON.parse(raw) : [];
  } catch {
    return [];
  }
}

/** Map permission keys to nav group/item visibility */
export const NAV_PERMISSION_MAP: Record<string, string[]> = {
  // Group: OVERVIEW — always visible
  "/dashboard": [],
  "/sessions": [],
  "/access-requests": [],

  // Group: IDENTITY
  "/users": ["users:read"],
  "/roles": ["roles:read"],
  "/organizations": ["orgs:read"],
  "/organizations/analytics": ["orgs:read"],
  "/settings/nhi": ["identity:read"],
  "/settings/migration": ["identity:read"],
  "/settings/attribute-mapping": ["identity:read"],
  "/settings/import-wizard": ["identity:write"],
  "/settings/import-monitor": ["identity:read"],
  "/settings/review-schedules": ["identity:read"],

  // Group: SECURITY
  "/security/session-detail": ["security:read"],
  "/security/cae-monitor": ["security:read"],
  "/security/privileged-activity": ["security:read"],
  "/security/risk-score": ["security:read"],
  "/security/posture": ["security:read"],
  "/settings/conditional-access": ["security:read"],
  "/settings/security-policy": ["security:read"],
  "/settings/password-migration": ["security:read"],
  "/settings/password-strength": ["security:read"],
  "/settings/password-policy": ["security:read"],
  "/settings/enrollment-campaign": ["security:read"],
  "/settings/passkey-management": ["security:read"],
  "/settings/mfa": ["security:read"],

  // Group: GOVERNANCE
  "/settings/sod-matrix": ["governance:read"],
  "/settings/delegations": ["governance:read"],
  "/policies": ["policies:read"],

  // Group: AUDIT
  "/audit": ["audit:read"],
  "/audit/explorer": ["audit:read"],
  "/audit/ccm": ["audit:read"],

  // Group: APPLICATIONS
  "/oauth-clients": ["oauth:read"],
  "/webhooks": ["webhooks:read"],
  "/api-keys": ["apikeys:read"],
  "/settings/scim": ["provisioning:read"],
  "/settings/ldap-config": ["provisioning:read"],
  "/settings/ldap-sync-config": ["provisioning:read"],

  // Group: SETTINGS
  "/settings": ["settings:read"],
  "/settings/branding": ["settings:read"],
  "/settings/feature-flags": ["settings:write"],

  // Group: ADMIN
  "/admin/tenants": ["tenants:read"],

  // Group: HELP — always visible
  "/docs": [],
  "/monitoring": [],
};

export function useUserPermissions(): {
  permissions: string[];
  hasPermission: (key: string) => boolean;
  loading: boolean;
} {
  const [permissions, setPermissions] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Try cached permissions first (instant render)
    const cached = getUserPermissions();
    if (cached.length > 0) {
      setPermissions(cached);
      setLoading(false);
    }
    // Fetch fresh from API
    fetchUserPermissions().then(perms => {
      if (perms && perms.length > 0) {
        localStorage.setItem("ggid_user_permissions", JSON.stringify(perms));
        setPermissions(perms);
      }
      setLoading(false);
    });
  }, []);

  const { isPlatformAdmin } = useUserRole();
  const hasPermission = (key: string): boolean => {
    if (isPlatformAdmin) return true; // Admin sees everything
    // If no dynamic permissions loaded, fall back to scope-based (legacy)
    if (permissions.length === 0) return true; // fallback: show all (legacy behavior)
    return permissions.includes(key);
  };

  return { permissions, hasPermission, loading };
}

export function useAuth() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const token = getAuthToken();
    setIsAuthenticated(!!token);
    setLoading(false);
  }, []);

  const logout = () => {
    if (typeof window !== "undefined") {
      localStorage.removeItem("ggid_access_token");
      localStorage.removeItem("ggid_refresh_token");
      localStorage.removeItem("ggid_session_id");
      localStorage.removeItem("ggid_tenant_id");
      localStorage.removeItem("ggid_user_id");
      localStorage.removeItem("ggid_user_name");
      localStorage.removeItem("ggid_user_email");
      localStorage.removeItem("ggid_user_scopes");
      localStorage.removeItem("ggid_user_permissions");
    }
    setIsAuthenticated(false);
  };

  return { isAuthenticated, loading, logout };
}
