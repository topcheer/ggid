"use client";

import { useEffect, useState } from "react";

const API_BASE = process.env.NEXT_PUBLIC_GGID_API || "http://localhost:8080";
const TENANT_ID =
  process.env.NEXT_PUBLIC_TENANT_ID || "00000000-0000-0000-0000-000000000001";

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

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    "X-Tenant-ID": TENANT_ID,
    ...(options?.headers as Record<string, string>),
  };

  const resp = await fetch(`${API_BASE}${path}`, { ...options, headers });

  if (!resp.ok) {
    const text = await resp.text();
    throw new Error(`API ${resp.status}: ${text}`);
  }

  if (resp.status === 204) return {} as T;
  return resp.json();
}

export function useUsers() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = async () => {
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
  };

  useEffect(() => {
    refresh();
  }, []);

  return { users, loading, error, refresh };
}

export function useApi() {
  return { apiFetch, API_BASE, TENANT_ID };
}
