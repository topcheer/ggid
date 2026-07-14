"use client";

import { useState, useEffect, useCallback } from "react";
import { FileWarning, RefreshCw, AlertTriangle, Clock, CheckCircle2 } from "lucide-react";

interface EvidenceItem {
  id: string;
  control_id: string;
  framework: string;
  evidence_type: string;
  collected_at: string;
  expires_at: string;
  days_remaining: number;
  status: "valid" | "expiring" | "expired";
}

export default function EvidenceExpiryPage() {
  const [items, setItems] = useState<EvidenceItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [refreshing, setRefreshing] = useState<Set<string>>(new Set());
  const [filterStatus, setFilterStatus] = useState("all");
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/audit/evidence-expiry", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) throw new Error(`Failed to load evidence: HTTP ${res.status}`);
      const data = await res.json();
      setItems(data.evidence || data || []);
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to load evidence expiry"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const refreshOne = async (id: string) => {
    setRefreshing((prev) => new Set(prev).add(id));
    try {
      const res = await fetch(`/api/v1/audit/evidence-expiry/${id}/refresh`, {
        method: "POST",
        headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      });
      if (!res.ok) throw new Error(`Refresh failed: HTTP ${res.status}`);
      setItems((prev) => prev.map((e) => e.id === id ? { ...e, status: "valid", days_remaining: 90, expires_at: new Date(Date.now() + 90 * 86400000).toISOString().split("T")[0] } : e));
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to refresh evidence"); }
    finally {
      setRefreshing((prev) => { const n = new Set(prev); n.delete(id); return n; });
    }
  };

  const batchRefresh = async () => {
    const expiring = items.filter((e) => e.status !== "valid");
    for (const e of expiring) {
      await refreshOne(e.id);
    }
  };

  const filtered = filterStatus === "all" ? items : items.filter((e) => e.status === filterStatus);
  const expired = items.filter((e) => e.status === "expired");
  const expiring = items.filter((e) => e.status === "expiring");
  const valid = items.filter((e) => e.status === "valid");

  const urgencyColor: Record<string, string> = {
    expired: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
    expiring: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400",
    valid: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  };

  const rowBorder: Record<string, string> = {
    expired: "border-l-4 border-red-500",
    expiring: "border-l-4 border-orange-500",
    valid: "",
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2"><FileWarning className="w-6 h-6 text-orange-500" /> Evidence Expiry</h1>
          <p className="text-sm text-gray-500 mt-1">Monitor and refresh expiring compliance evidence.</p>
        </div>
        {(expired.length > 0 || expiring.length > 0) && (
          <button onClick={batchRefresh} aria-label={`Batch refresh ${expired.length + expiring.length} evidence items`} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 flex items-center gap-2"><RefreshCw className="w-4 h-4" /> Batch Refresh ({expired.length + expiring.length})</button>
        )}
      </div>

      {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span className="flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> {error}</span><button onClick={fetchData} className="text-xs underline hover:text-red-700">Retry</button></div>}

      {<div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800" onClick={() => setFilterStatus(filterStatus === "valid" ? "all" : "valid")} role="button" aria-label="Filter by valid status" tabIndex={0} onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") setFilterStatus(filterStatus === "valid" ? "all" : "valid"); }}>
          <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Valid</span><CheckCircle2 className="w-5 h-5 text-green-400" /></div>
          <p className="text-2xl font-bold mt-1 text-green-600">{valid.length}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Expiring</span><Clock className="w-5 h-5 text-orange-400" /></div>
          <p className="text-2xl font-bold mt-1 text-orange-600">{expiring.length}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Expired</span><AlertTriangle className="w-5 h-5 text-red-400" /></div>
          <p className="text-2xl font-bold mt-1 text-red-600">{expired.length}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <span className="text-sm text-gray-500">Total</span>
          <p className="text-2xl font-bold mt-1">{items.length}</p>
        </div>
      </div>}

      {loading && <div className="rounded-lg border dark:border-gray-800 p-8 text-center"><div className="inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" /><div className="text-sm text-gray-500">Loading evidence...</div></div>}

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50">
            <tr><th className="px-4 py-3 text-left font-medium">Control</th><th className="px-4 py-3 text-left font-medium">Framework</th><th className="px-4 py-3 text-left font-medium">Type</th><th className="px-4 py-3 text-left font-medium">Collected</th><th className="px-4 py-3 text-left font-medium">Expires</th><th className="px-4 py-3 text-left font-medium">Days</th><th className="px-4 py-3 text-left font-medium">Status</th><th className="px-4 py-3 text-left font-medium">Action</th></tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {filtered.map((e) => (
              <tr key={e.id} className={"hover:bg-gray-50 dark:hover:bg-gray-900/30 " + rowBorder[e.status]}>
                <td className="px-4 py-3 font-medium">{e.control_id}</td>
                <td className="px-4 py-3 text-xs text-gray-500">{e.framework}</td>
                <td className="px-4 py-3 text-xs text-gray-500">{e.evidence_type}</td>
                <td className="px-4 py-3 text-xs text-gray-500">{e.collected_at}</td>
                <td className="px-4 py-3 text-xs text-gray-500">{e.expires_at}</td>
                <td className="px-4 py-3 text-xs font-bold">{e.days_remaining}</td>
                <td className="px-4 py-3"><span className={"px-2 py-0.5 rounded text-xs " + urgencyColor[e.status]}>{e.status}</span></td>
                <td className="px-4 py-3">
                  <button onClick={() => refreshOne(e.id)} disabled={refreshing.has(e.id)} aria-label={`Refresh evidence ${e.control_id}`} className="text-xs text-blue-600 hover:underline flex items-center gap-1 disabled:opacity-50"><RefreshCw className={"w-3 h-3 " + (refreshing.has(e.id) ? "animate-spin" : "")} /> {refreshing.has(e.id) ? "Refreshing..." : "Refresh"}</button>
                </td>
              </tr>
            ))}
            {filtered.length === 0 && !loading && <tr><td colSpan={8} className="px-4 py-8 text-center text-gray-500">No evidence items.</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}
