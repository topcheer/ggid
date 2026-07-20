"use client";
import { useState, useEffect, useCallback } from "react";
import { Globe, Loader2, AlertCircle, Shield, Activity, Filter } from "lucide-react";
import { usePageTitle } from "@/lib/usePageTitle";
import { authHeader } from "@/lib/auth-helpers";
import { API_BASE_URL } from "@/lib/api-config";

const API_BASE = API_BASE_URL;

interface AuditEvent { id: string; event_type: string; actor_id: string; actor_name: string; tenant_id: string; action: string; result: string; created_at: string; resource_type: string; ip_address: string; }

export default function GlobalAuditDashboard() {
  usePageTitle("Global Audit Dashboard");
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [tenantFilter, setTenantFilter] = useState("");

  const load = useCallback(async () => {
    setLoading(true); setError("");
    try {
      const params = new URLSearchParams();
      if (tenantFilter) params.set("tenant_id", tenantFilter);
      params.set("page_size", "50");
      const res = await fetch(`${API_BASE}/api/v1/audit/events?${params}`, { headers: { ...authHeader() } });
      if (res.ok) { const d = await res.json(); setEvents(d.events || d.items || (Array.isArray(d) ? d : [])); }
    } catch { setError("Failed to load audit events"); }
    setLoading(false);
  }, [tenantFilter]);

  useEffect(() => { load(); }, [load]);

  // Unique tenant IDs for filter
  const tenantIds = [...new Set(events.map(e => e.tenant_id).filter(Boolean))];

  if (loading) return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>;

  return (
    <div className="p-6">
      <h1 className="mb-1 text-2xl font-bold text-gray-900 dark:text-white">Global Audit Dashboard</h1>
      <p className="mb-4 text-sm text-gray-500">Cross-tenant audit events from all tenants.</p>

      {error && <div className="mb-4 flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950"><AlertCircle className="h-4 w-4 shrink-0" /> {error}</div>}

      <div className="mb-4 flex items-center gap-3">
        <Filter className="h-4 w-4 text-gray-400" />
        <select value={tenantFilter} onChange={e => setTenantFilter(e.target.value)} className="rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-700 dark:bg-gray-800">
          <option value="">All Tenants</option>
          {tenantIds.map(tid => <option key={tid} value={tid}>{tid.substring(0, 8)}</option>)}
        </select>
      </div>

      <div className="mb-4 grid grid-cols-3 gap-4">
        <div className="rounded-lg border border-gray-200 p-4 dark:border-gray-800"><div className="flex items-center gap-2"><Activity className="h-5 w-5 text-blue-500" /></div><p className="mt-2 text-2xl font-bold">{events.length}</p><p className="text-xs text-gray-500">Total Events</p></div>
        <div className="rounded-lg border border-gray-200 p-4 dark:border-gray-800"><div className="flex items-center gap-2"><Globe className="h-5 w-5 text-green-500" /></div><p className="mt-2 text-2xl font-bold">{tenantIds.length}</p><p className="text-xs text-gray-500">Tenants</p></div>
        <div className="rounded-lg border border-gray-200 p-4 dark:border-gray-800"><div className="flex items-center gap-2"><Shield className="h-5 w-5 text-red-500" /></div><p className="mt-2 text-2xl font-bold">{events.filter(e => e.result === "failed" || e.result === "denied").length}</p><p className="text-xs text-gray-500">Failed/Denied</p></div>
      </div>

      <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-800">
        <table className="w-full">
          <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-800 dark:bg-gray-900">
            <tr><th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Time</th><th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Event</th><th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Actor</th><th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Tenant</th><th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Result</th><th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">IP</th></tr>
          </thead>
          <tbody className="divide-y divide-gray-100 dark:divide-gray-800">
            {events.slice(0, 50).map(e => (
              <tr key={e.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                <td className="px-4 py-3 text-xs text-gray-500">{e.created_at ? new Date(e.created_at).toLocaleString() : "—"}</td>
                <td className="px-4 py-3 text-sm">{e.event_type || e.action || "—"}</td>
                <td className="px-4 py-3 text-sm">{e.actor_name || e.actor_id?.substring(0, 8) || "system"}</td>
                <td className="px-4 py-3 text-xs"><span className="rounded-full bg-indigo-50 px-2 py-0.5 text-xs text-indigo-600 dark:bg-indigo-950">{e.tenant_id?.substring(0, 8) || "—"}</span></td>
                <td className="px-4 py-3"><span className={`rounded-full px-2 py-0.5 text-xs font-medium ${e.result === "success" ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}>{e.result}</span></td>
                <td className="px-4 py-3 text-xs text-gray-400">{e.ip_address?.replace(/\/\d+$/, "") || "—"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
