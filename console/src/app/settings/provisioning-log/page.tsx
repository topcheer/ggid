"use client";

import { useState, useEffect, useCallback } from "react";
import { ListChecks, RefreshCw, Filter, ChevronDown } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface LogEvent {
  id: string;
  timestamp: string;
  user: string;
  source: "SCIM" | "JIT" | "manual";
  action: "create" | "update" | "disable" | "delete";
  target_app: string;
  status: "success" | "failed";
  error_detail: string | null;
}

const sourceColors: Record<string, string> = {
  SCIM: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  JIT: "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400",
  manual: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
};

export default function ProvisioningLogPage() {
  const t = useTranslations();

  const [events, setEvents] = useState<LogEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [filterSource, setFilterSource] = useState("");
  const [filterStatus, setFilterStatus] = useState("");
  const [expanded, setExpanded] = useState<string | null>(null);
  const [retrying, setRetrying] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try { const res = await fetch("/api/v1/identity/provisioning-log", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); setEvents(d.events || d || []); } }
    catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const retry = async (id: string) => {
    setRetrying(id);
    try { await fetch("/api/v1/identity/provisioning-log/" + id + "/retry", { method: "POST", headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); }
    catch { /* noop */ }
    finally { setRetrying(null); }
  };

  const filtered = events.filter((e) => { if (filterSource && e.source !== filterSource) return false; if (filterStatus && e.status !== filterStatus) return false; return true; });
  const failed = filtered.filter((e) => e.status === "failed").length;

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><ListChecks className="w-6 h-6 text-blue-500" /> {t("provisioningLog.title")}</h1><p className="text-sm text-gray-500 mt-1">Track user provisioning events across connected applications.</p></div>

      <div className="grid grid-cols-3 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total Events</span><p className="text-xl font-bold mt-1">{events.length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Failed</span><p className="text-xl font-bold text-red-600 mt-1">{failed}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Sources</span><p className="text-xl font-bold mt-1">{new Set(events.map((e) => e.source)).size}</p></div>
      </div>

      <div className="flex items-center gap-2">
        <Filter className="w-4 h-4 text-gray-400" />
        <select aria-label="Filter" value={filterSource} onChange={(e) => setFilterSource(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">All Sources</option><option value="SCIM">SCIM</option><option value="JIT">JIT</option><option value="manual">Manual</option></select>
        <select aria-label="Filter" value={filterStatus} onChange={(e) => setFilterStatus(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">All Status</option><option value="success">Success</option><option value="failed">Failed</option></select>
        <span className="text-sm text-gray-500">{filtered.length} events</span>
      </div>

      <div className="space-y-2">
        {filtered.map((e) => (
          <div key={e.id} className="rounded-lg border dark:border-gray-800 p-3">
            <div className="flex items-center justify-between cursor-pointer" onClick={() => e.status === "failed" && setExpanded(expanded === e.id ? null : e.id)}>
              <div className="flex items-center gap-3"><span className={"px-2 py-0.5 rounded text-xs " + sourceColors[e.source]}>{e.source}</span><span className={"px-2 py-0.5 rounded text-xs " + (e.status === "success" ? "bg-green-100 dark:bg-green-900/30 dark:text-green-400" : "bg-red-100 dark:bg-red-900/30 dark:text-red-400")}>{e.status}</span><span className="text-sm font-medium">{e.action}</span><span className="text-xs text-gray-500">{e.user} - {e.target_app}</span></div>
              <div className="flex items-center gap-2"><span className="text-xs text-gray-400">{e.timestamp}</span>{e.status === "failed" && <ChevronDown className={"w-4 h-4 text-gray-400 transition-transform " + (expanded === e.id ? "rotate-180" : "")} />}{e.status === "failed" && <button onClick={(ev) => { ev.stopPropagation(); retry(e.id); }} disabled={retrying === e.id} className="text-xs text-blue-600 hover:underline disabled:opacity-50 flex items-center gap-1"><RefreshCw className={"w-3 h-3 " + (retrying === e.id ? "animate-spin" : "")} /> Retry</button>}</div>
            </div>
            {expanded === e.id && e.error_detail && <div className="mt-2 pl-12 text-xs text-red-500 bg-red-50 dark:bg-red-900/10 rounded p-2">{e.error_detail}</div>}
          </div>
        ))}
        {filtered.length === 0 && !loading && <p className="text-sm text-gray-500 text-center py-8">No provisioning events.</p>}
      </div>
    </div>
  );
}
