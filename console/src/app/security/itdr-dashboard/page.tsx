"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import {
  Shield, Loader2, AlertCircle, X, RefreshCw, Activity,
  AlertTriangle, Filter, Clock,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface Detection {
  id: string;
  type: string;
  severity: "critical" | "high" | "medium" | "low";
  source: string;
  timestamp: string;
  affected_users: number;
  status: "active" | "acknowledged" | "resolved";
  mitre_techniques: string[];
  description: string;
}

interface Stats {
  total: number;
  critical: number;
  high: number;
  medium: number;
  low: number;
  acknowledged: number;
  resolved: number;
}

const sevColors: Record<string, string> = {
  critical: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400 border-red-300 dark:border-red-800",
  high: "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400 border-orange-300 dark:border-orange-800",
  medium: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400 border-yellow-300 dark:border-yellow-800",
  low: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400 border-blue-300 dark:border-blue-800",
};

const statusColors: Record<string, string> = {
  active: "bg-red-50 text-red-600 dark:bg-red-950/30 dark:text-red-400",
  acknowledged: "bg-yellow-50 text-yellow-600 dark:bg-yellow-950/30 dark:text-yellow-400",
  resolved: "bg-green-50 text-green-600 dark:bg-green-950/30 dark:text-green-400",
};

export default function ITDRDashboardPage() {
  const t = useTranslations();
  const [detections, setDetections] = useState<Detection[]>([]);
  const [stats, setStats] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filterSeverity, setFilterSeverity] = useState<string>("all");
  const [filterStatus, setFilterStatus] = useState<string>("all");
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [actingId, setActingId] = useState<string | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const loadData = useCallback(async () => {
    try {
      const headers = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [detRes, statsRes] = await Promise.all([
        fetch("/api/v1/audit/itdr/detections?page_size=50", { headers }).catch(() => null),
        fetch("/api/v1/audit/itdr/stats?window=24h", { headers }).catch(() => null),
      ]);
      if (detRes?.ok) {
        const d = await detRes.json();
        setDetections(d.detections || d.items || []);
      }
      if (statsRes?.ok) {
        const s = await statsRes.json();
        setStats({
          total: s.total || 0,
          critical: s.critical || 0,
          high: s.high || 0,
          medium: s.medium || 0,
          low: s.low || 0,
          acknowledged: s.acknowledged || 0,
          resolved: s.resolved || 0,
        });
      }
    } catch { /* keep previous data */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => {
    loadData();
    if (autoRefresh) {
      intervalRef.current = setInterval(loadData, 10000); // Refresh every 10s
    }
    return () => { if (intervalRef.current) clearInterval(intervalRef.current); };
  }, [loadData, autoRefresh]);

  const acknowledge = async (id: string) => {
    setActingId(id);
    try {
      await fetch(`/api/v1/audit/itdr/detections/${id}/acknowledge`, {
        method: "POST", headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      });
      setDetections(prev => prev.map(d => d.id === id ? { ...d, status: "acknowledged" } : d));
    } catch { setError("Failed to acknowledge"); }
    finally { setActingId(null); }
  };

  const resolve = async (id: string) => {
    setActingId(id);
    try {
      await fetch(`/api/v1/audit/itdr/detections/${id}/resolve`, {
        method: "POST", headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      });
      setDetections(prev => prev.map(d => d.id === id ? { ...d, status: "resolved" } : d));
    } catch { setError("Failed to resolve"); }
    finally { setActingId(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const filtered = detections.filter(d => {
    if (filterSeverity !== "all" && d.severity !== filterSeverity) return false;
    if (filterStatus !== "all" && d.status !== filterStatus) return false;
    return true;
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Activity className="h-6 w-6 text-red-500" />
            ITDR Real-Time Dashboard
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Live threat detections with auto-refresh (10s interval).
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => setAutoRefresh(!autoRefresh)} aria-pressed={autoRefresh} className={"flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium " + (autoRefresh ? "bg-green-50 text-green-700 dark:bg-green-950/30 dark:text-green-400" : "border border-gray-300 text-gray-600 dark:border-gray-700 dark:text-gray-300")}>
            <Activity className={"h-4 w-4 " + (autoRefresh ? "animate-pulse" : "")} /> {autoRefresh ? "Live" : "Paused"}
          </button>
          <button onClick={loadData} aria-label="Refresh now" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
            <RefreshCw className="h-4 w-4" /> Refresh
          </button>
        </div>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Stats */}
      {stats && (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4 lg:grid-cols-7">
          <div className={cardCls + " text-center"}><p className="text-xs font-semibold uppercase text-gray-400">Total</p><p className="mt-1 text-2xl font-bold">{stats.total}</p></div>
          <div className={cardCls + " text-center"}><p className="text-xs font-semibold uppercase text-red-400">Critical</p><p className="mt-1 text-2xl font-bold text-red-600">{stats.critical}</p></div>
          <div className={cardCls + " text-center"}><p className="text-xs font-semibold uppercase text-orange-400">High</p><p className="mt-1 text-2xl font-bold text-orange-600">{stats.high}</p></div>
          <div className={cardCls + " text-center"}><p className="text-xs font-semibold uppercase text-yellow-400">Medium</p><p className="mt-1 text-2xl font-bold text-yellow-600">{stats.medium}</p></div>
          <div className={cardCls + " text-center"}><p className="text-xs font-semibold uppercase text-blue-400">Low</p><p className="mt-1 text-2xl font-bold text-blue-600">{stats.low}</p></div>
          <div className={cardCls + " text-center"}><p className="text-xs font-semibold uppercase text-yellow-400">Ack'd</p><p className="mt-1 text-2xl font-bold text-yellow-600">{stats.acknowledged}</p></div>
          <div className={cardCls + " text-center"}><p className="text-xs font-semibold uppercase text-green-400">Resolved</p><p className="mt-1 text-2xl font-bold text-green-600">{stats.resolved}</p></div>
        </div>
      )}

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-2">
        <Filter className="h-4 w-4 text-gray-400" />
        <select aria-label="Filter severity" value={filterSeverity} onChange={e => setFilterSeverity(e.target.value)} className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-sm">
          <option value="all">All Severities</option>
          <option value="critical">Critical</option>
          <option value="high">High</option>
          <option value="medium">Medium</option>
          <option value="low">Low</option>
        </select>
        <select aria-label="Filter status" value={filterStatus} onChange={e => setFilterStatus(e.target.value)} className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1 text-sm">
          <option value="all">All Status</option>
          <option value="active">Active</option>
          <option value="acknowledged">Acknowledged</option>
          <option value="resolved">Resolved</option>
        </select>
        <span className="text-xs text-gray-400">{filtered.length} detections</span>
      </div>

      {/* Detection list */}
      {loading && detections.length === 0 ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-red-500" /></div> : filtered.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><Shield className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No threats detected. System is clean.</p></div></div>
      ) : (
        <div className="space-y-2">
          {filtered.map(d => (
            <div key={d.id} className={"rounded-xl border p-4 " + sevColors[d.severity]}>
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <AlertTriangle className={"h-4 w-4 " + (d.severity === "critical" ? "text-red-600" : d.severity === "high" ? "text-orange-600" : "text-yellow-600")} />
                    <span className="font-medium text-gray-900 dark:text-white">{d.type}</span>
                    <span className={"px-2 py-0.5 rounded text-xs font-medium " + sevColors[d.severity]}>{d.severity}</span>
                    <span className={"px-2 py-0.5 rounded text-xs " + statusColors[d.status]}>{d.status}</span>
                  </div>
                  {d.description && <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">{d.description}</p>}
                  <div className="mt-2 flex flex-wrap items-center gap-3 text-xs text-gray-500">
                    <span>Source: {d.source}</span>
                    {d.affected_users > 0 && <span>Users: {d.affected_users}</span>}
                    <span className="flex items-center gap-1"><Clock className="h-3 w-3" /> {d.timestamp ? new Date(d.timestamp).toLocaleString() : "—"}</span>
                    {d.mitre_techniques?.length > 0 && d.mitre_techniques.map(m => <span key={m} className="px-1.5 py-0.5 rounded font-mono bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400">{m}</span>)}
                  </div>
                </div>
                {d.status === "active" && (
                  <div className="flex items-center gap-2">
                    <button onClick={() => acknowledge(d.id)} disabled={actingId === d.id} aria-label={`Acknowledge ${d.type}`} className="rounded-lg bg-yellow-50 px-3 py-1.5 text-xs font-medium text-yellow-700 hover:bg-yellow-100 dark:bg-yellow-950/30 disabled:opacity-50">
                      {actingId === d.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : "Acknowledge"}
                    </button>
                    <button onClick={() => resolve(d.id)} disabled={actingId === d.id} aria-label={`Resolve ${d.type}`} className="rounded-lg bg-green-50 px-3 py-1.5 text-xs font-medium text-green-700 hover:bg-green-100 dark:bg-green-950/30 disabled:opacity-50">Resolve</button>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
