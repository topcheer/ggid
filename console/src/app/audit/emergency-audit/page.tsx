"use client";

import { useState, useEffect, useCallback } from "react";
import { AlertTriangle, Calendar, Clock, Activity, Zap, Shield } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface BreakGlassRecord {
  id: string;
  requester: string;
  reason: string;
  scope: string;
  activated_at: string;
  deactivated_at: string | null;
  duration_minutes: number;
  actions_taken: string[];
  status: "active" | "closed" | "reviewed";
}

const statusColors: Record<string, string> = {
  active: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  closed: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  reviewed: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
};

export default function EmergencyAccessAuditPage() {
  const t = useTranslations();

  const [records, setRecords] = useState<BreakGlassRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const params = startDate && endDate ? `?start=${startDate}&end=${endDate}` : "";
      const res = await fetch(`/api/v1/audit/emergency-access${params}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setRecords(data.records || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [startDate, endDate]);

  useEffect(() => {
    const end = new Date();
    const start = new Date(); start.setDate(start.getDate() - 30);
    setStartDate(start.toISOString().split("T")[0]);
    setEndDate(end.toISOString().split("T")[0]);
  }, []);

  useEffect(() => {
    if (startDate && endDate) fetchData();
  }, [startDate, endDate, fetchData]);

  const activeCount = records.filter((r) => r.status === "active").length;
  const totalActions = records.reduce((s, r) => s + r.actions_taken.length, 0);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Zap className="w-6 h-6 text-red-500" /> {t("auditEmergencyAudit.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Break-glass usage records and post-incident review tracking.</p>
      </div>

      {/* Date filter */}
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2">
          <Calendar className="w-4 h-4 text-gray-400" />
          <input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
          <span className="text-gray-400">to</span>
          <input type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" />
        </div>
        <button onClick={fetchData} disabled={loading} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">{loading ? "Loading..." : "Refresh"}</button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Total Events</span><Activity className="w-5 h-5 text-gray-400" /></div>
          <p className="text-2xl font-bold mt-1">{records.length}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Active</span><AlertTriangle className="w-5 h-5 text-red-400" /></div>
          <p className="text-2xl font-bold mt-1 text-red-600">{activeCount}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Total Actions</span><Activity className="w-5 h-5 text-gray-400" /></div>
          <p className="text-2xl font-bold mt-1">{totalActions}</p>
        </div>
        <div className="rounded-lg border p-4 dark:border-gray-800">
          <div className="flex items-center justify-between"><span className="text-sm text-gray-500">Avg Duration</span><Clock className="w-5 h-5 text-gray-400" /></div>
          <p className="text-2xl font-bold mt-1">{records.length > 0 ? Math.round(records.reduce((s, r) => s + r.duration_minutes, 0) / records.length) : 0}<span className="text-base text-gray-400">m</span></p>
        </div>
      </div>

      {/* Records timeline */}
      <div className="space-y-4">
        {records.map((r) => (
          <div key={r.id} className="rounded-lg border dark:border-gray-800 overflow-hidden">
            <div className="px-4 py-3 border-b dark:border-gray-800 bg-gray-50 dark:bg-gray-900/30">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Shield className="w-4 h-4 text-gray-400" />
                  <span className="font-semibold">{r.requester}</span>
                  <span className={`px-2 py-0.5 rounded text-xs ${statusColors[r.status]}`}>{r.status}</span>
                </div>
                <div className="flex items-center gap-3 text-xs text-gray-400">
                  <span className="flex items-center gap-1"><Clock className="w-3 h-3" /> {r.duration_minutes}m</span>
                  <span>{r.activated_at}</span>
                </div>
              </div>
            </div>
            <div className="px-4 py-3 space-y-2 text-sm">
              <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
                <div><span className="text-xs text-gray-400">Reason</span><p className="text-sm">{r.reason}</p></div>
                <div><span className="text-xs text-gray-400">Scope</span><p className="text-sm font-mono">{r.scope}</p></div>
                <div><span className="text-xs text-gray-400">Deactivated</span><p className="text-sm text-gray-500">{r.deactivated_at || "Still active"}</p></div>
              </div>
              {r.actions_taken.length > 0 && (
                <div>
                  <span className="text-xs text-gray-400">Actions Taken ({r.actions_taken.length})</span>
                  <div className="flex flex-wrap gap-1 mt-1">
                    {r.actions_taken.map((a, i) => (
                      <span key={i} className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{a}</span>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        ))}
        {records.length === 0 && !loading && (
          <p className="text-sm text-gray-500 text-center py-8">No emergency access events found.</p>
        )}
      </div>
    </div>
  );
}
