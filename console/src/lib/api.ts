"use client";

import { useEffect, useState, useCallback } from "react";

const API_BASE = process.env.NEXT_PUBLIC_GGID_API || "http://localhost:8080";
const TENANT_ID =
  process.env.NEXT_PUBLIC_TENANT_ID || "00000000-0000-0000-0000-000000000001";

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
    }
    setIsAuthenticated(false);
  };

  return { isAuthenticated, loading, logout };
}
