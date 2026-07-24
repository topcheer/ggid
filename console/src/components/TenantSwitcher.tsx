"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { Building2, ChevronDown, Check, Loader2 } from "lucide-react";
import { API_BASE_URL } from "@/lib/api-config";
import { authHeader } from "@/lib/auth-helpers";

interface Tenant {
  id: string;
  name: string;
  slug?: string;
  plan?: string;
}

/**
 * Tenant switcher — allows platform admins to switch the active tenant context.
 * The selected tenant ID is stored in localStorage as "ggid_tenant_id" and
 * used by authHeader() to set X-Tenant-ID on all API requests.
 */
export function TenantSwitcher() {
  const [open, setOpen] = useState(false);
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [loading, setLoading] = useState(false);
  const [currentId, setCurrentId] = useState<string | null>(null);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setCurrentId(localStorage.getItem("ggid_tenant_id"));
  }, []);

  // Close on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const loadTenants = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE_URL}/api/v1/tenants`, {
        headers: { ...authHeader() },
      });
      if (res.ok) {
        const d = await res.json();
        setTenants(d.tenants || d.items || (Array.isArray(d) ? d : []));
      }
    } catch {
      // Silent fail — tenant switcher is optional
    }
    setLoading(false);
  }, []);

  const handleOpen = () => {
    if (!open && tenants.length === 0) loadTenants();
    setOpen(!open);
  };

  const switchTenant = (tenant: Tenant) => {
    localStorage.setItem("ggid_tenant_id", tenant.id);
    setCurrentId(tenant.id);
    setOpen(false);
    // Reload page to refresh all data with new tenant context
    window.location.reload();
  };

  const current = tenants.find(t => t.id === currentId);

  return (
    <div ref={ref} className="relative">
      <button
        onClick={handleOpen}
        className="flex items-center gap-2 rounded-lg border border-gray-200 dark:border-gray-700 px-3 py-1.5 text-sm hover:bg-gray-50 dark:hover:bg-gray-800 transition"
        title="Switch tenant context"
      >
        <Building2 className="h-4 w-4 text-gray-400" />
        <span className="hidden sm:inline max-w-[120px] truncate text-gray-700 dark:text-gray-300">
          {current?.name || (currentId ? currentId.slice(0, 8) + "..." : "Select Tenant")}
        </span>
        <ChevronDown className="h-3 w-3 text-gray-400" />
      </button>

      {open && (
        <div className="absolute right-0 z-50 mt-1 max-h-72 overflow-y-auto rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-gray-700 dark:bg-gray-900 min-w-[240px]">
          {loading ? (
            <div className="flex items-center justify-center py-4">
              <Loader2 className="h-4 w-4 animate-spin text-gray-400" />
            </div>
          ) : tenants.length === 0 ? (
            <div className="px-3 py-4 text-center text-xs text-gray-400">No tenants available</div>
          ) : (
            <>
              <div className="px-3 py-1 text-xs font-semibold uppercase text-gray-400 border-b border-gray-100 dark:border-gray-800 mb-1">
                Switch Tenant
              </div>
              {tenants.map(t => (
                <button
                  key={t.id}
                  onClick={() => switchTenant(t)}
                  className={`flex w-full items-center justify-between px-3 py-2 text-sm transition ${
                    t.id === currentId
                      ? "bg-brand-50 text-brand-700 dark:bg-brand-950/30 dark:text-brand-300"
                      : "text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-800"
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <Building2 className="h-3.5 w-3.5 text-gray-400" />
                    <div>
                      <div className="font-medium">{t.name}</div>
                      {t.slug && <div className="text-xs text-gray-400 font-mono">{t.slug}</div>}
                    </div>
                  </div>
                  {t.id === currentId && <Check className="h-4 w-4 text-brand-500" />}
                </button>
              ))}
            </>
          )}
        </div>
      )}
    </div>
  );
}