"use client";

import { useState, useEffect, useCallback } from "react";
import { Siren, Check, X, Filter, AlertTriangle } from "lucide-react";

interface Alert {
  id: string;
  type: string;
  severity: "low" | "medium" | "high" | "critical";
  source: string;
  timestamp: string;
  affected_users: number;
  detail: string;
  status: "active" | "acknowledged" | "resolved";
}

const sevColors: Record<string, string> = {
  low: "border-l-gray-400", medium: "border-l-yellow-500", high: "border-l-orange-500", critical: "border-l-red-600",
};
const statusColors: Record<string, string> = {
  active: "text-red-600", acknowledged: "text-yellow-600", resolved: "text-green-600",
};

export default function SecurityAlertsPage() {
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [loading, setLoading] = useState(false);
  const [filterStatus, setFilterStatus] = useState("");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try { const res = await fetch("/api/v1/audit/security-alerts", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); setAlerts(d.alerts || d || []); } }
    catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const updateStatus = async (id: string, status: "acknowledged" | "resolved") => {
    setAlerts(alerts.map((a) => a.id === id ? { ...a, status } : a));
    try { await fetch("/api/v1/audit/security-alerts/" + id, { method: "PATCH", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify({ status }) }); } catch { /* noop */ }
  };

  const filtered = filterStatus ? alerts.filter((a) => a.status === filterStatus) : alerts;
  const activeCount = alerts.filter((a) => a.status === "active").length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Siren className="w-6 h-6 text-red-500" /> Security Alerts</h1><p className="text-sm text-gray-500 mt-1">Security alert monitoring with acknowledge and resolve workflows.</p></div>
        {activeCount > 0 && <span className="px-3 py-1 rounded-full text-xs font-medium bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400">{activeCount} active</span>}
      </div>

      <div className="flex items-center gap-2">
        <Filter className="w-4 h-4 text-gray-400" />
        <select value={filterStatus} onChange={(e) => setFilterStatus(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">All Status</option><option value="active">Active</option><option value="acknowledged">Acknowledged</option><option value="resolved">Resolved</option></select>
        <span className="text-sm text-gray-500">{filtered.length} alerts</span>
      </div>

      <div className="space-y-2">
        {filtered.map((a) => (
          <div key={a.id} className={"rounded-lg border-l-4 dark:border-gray-800 bg-white dark:bg-gray-900 p-3 " + sevColors[a.severity]}>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2"><AlertTriangle className={"w-4 h-4 " + (a.severity === "critical" ? "text-red-600" : a.severity === "high" ? "text-orange-500" : "text-gray-400")} /><span className="text-sm font-medium">{a.type}</span><span className={"text-xs font-medium " + statusColors[a.status]}>{a.status}</span></div>
              <span className="text-xs text-gray-400">{a.timestamp}</span>
            </div>
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{a.detail}</p>
            <div className="flex items-center justify-between mt-2"><span className="text-xs text-gray-500">Source: {a.source} - {a.affected_users} users affected</span>
              {a.status === "active" && <div className="flex gap-1"><button onClick={() => updateStatus(a.id, "acknowledged")} className="px-2 py-1 rounded text-xs bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400 text-yellow-700 flex items-center gap-1"><Check className="w-3 h-3" /> Ack</button><button onClick={() => updateStatus(a.id, "resolved")} className="px-2 py-1 rounded text-xs bg-green-100 dark:bg-green-900/30 dark:text-green-400 text-green-700 flex items-center gap-1"><Check className="w-3 h-3" /> Resolve</button></div>}
              {a.status === "acknowledged" && <button onClick={() => updateStatus(a.id, "resolved")} className="px-2 py-1 rounded text-xs bg-green-100 dark:bg-green-900/30 dark:text-green-400 text-green-700 flex items-center gap-1"><Check className="w-3 h-3" /> Resolve</button>}
            </div>
          </div>
        ))}
        {filtered.length === 0 && !loading && <p className="text-sm text-gray-500 text-center py-8">No alerts.</p>}
      </div>
    </div>
  );
}
