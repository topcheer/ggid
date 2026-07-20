"use client";
import { useState, useEffect, useCallback } from "react";
import { ShieldAlert, Loader2, AlertCircle, Activity, Zap, Globe } from "lucide-react";
import { usePageTitle } from "@/lib/usePageTitle";
import { authHeader } from "@/lib/auth-helpers";
import { API_BASE_URL } from "@/lib/api-config";

const API_BASE = API_BASE_URL;

interface Threat { id: string; type: string; severity: string; tenant_id: string; description: string; detected_at: string; status: string; }

export default function GlobalThreatsPage() {
  usePageTitle("Global Threat Dashboard");
  const [threats, setThreats] = useState<Threat[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true); setError("");
    try {
      const res = await fetch(`${API_BASE}/api/v1/security/threats`, { headers: { ...authHeader() } });
      if (res.ok) { const d = await res.json(); setThreats(d.threats || d.items || d.incidents || (Array.isArray(d) ? d : [])); }
    } catch { setError("Failed to load threats"); }
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const severityColors: Record<string, string> = { critical: "bg-red-100 text-red-700", high: "bg-orange-100 text-orange-700", medium: "bg-yellow-100 text-yellow-700", low: "bg-blue-100 text-blue-700" };

  if (loading) return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>;

  return (
    <div className="p-6">
      <h1 className="mb-1 text-2xl font-bold text-gray-900 dark:text-white">Global Threat Dashboard</h1>
      <p className="mb-4 text-sm text-gray-500">Security threats and incidents across all tenants.</p>

      {error && <div className="mb-4 flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950"><AlertCircle className="h-4 w-4 shrink-0" /> {error}</div>}

      <div className="mb-4 grid grid-cols-4 gap-4">
        <div className="rounded-lg border border-gray-200 p-4 dark:border-gray-800"><ShieldAlert className="h-5 w-5 text-red-500" /><p className="mt-2 text-2xl font-bold">{threats.length}</p><p className="text-xs text-gray-500">Total Threats</p></div>
        <div className="rounded-lg border border-gray-200 p-4 dark:border-gray-800"><Zap className="h-5 w-5 text-orange-500" /><p className="mt-2 text-2xl font-bold">{threats.filter(t => t.severity === "critical" || t.severity === "high").length}</p><p className="text-xs text-gray-500">Critical/High</p></div>
        <div className="rounded-lg border border-gray-200 p-4 dark:border-gray-800"><Activity className="h-5 w-5 text-blue-500" /><p className="mt-2 text-2xl font-bold">{threats.filter(t => t.status === "open").length}</p><p className="text-xs text-gray-500">Open</p></div>
        <div className="rounded-lg border border-gray-200 p-4 dark:border-gray-800"><Globe className="h-5 w-5 text-green-500" /><p className="mt-2 text-2xl font-bold">{new Set(threats.map(t => t.tenant_id)).size}</p><p className="text-xs text-gray-500">Tenants Affected</p></div>
      </div>

      {threats.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-8 text-center dark:border-gray-800 dark:bg-gray-900"><ShieldAlert className="mx-auto mb-3 h-12 w-12 text-gray-300" /><p className="text-sm text-gray-500">No active threats detected.</p></div>
      ) : (
        <div className="space-y-3">
          {threats.map(t => (
            <div key={t.id} className="flex items-center justify-between rounded-xl border border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-900">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-red-50 dark:bg-red-950"><ShieldAlert className="h-5 w-5 text-red-500" /></div>
                <div>
                  <p className="text-sm font-medium text-gray-900 dark:text-white">{t.type || t.description || "Unknown Threat"}</p>
                  <p className="text-xs text-gray-500">{t.description} - Tenant: {t.tenant_id?.substring(0, 8) || "—"}</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${severityColors[t.severity] || severityColors.low}`}>{t.severity}</span>
                <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${t.status === "open" ? "bg-red-100 text-red-700" : "bg-green-100 text-green-700"}`}>{t.status}</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
