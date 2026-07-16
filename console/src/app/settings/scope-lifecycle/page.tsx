"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { ShieldCheck, Clock, AlertTriangle } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface ScopeRequest {
  id: string;
  scope: string;
  requester: string;
  approver_chain: { approver: string; status: "pending" | "approved" | "rejected"; acted_at?: string }[];
  status: "pending" | "approved" | "rejected" | "expired";
  risk_level: "low" | "medium" | "high";
  requested_at: string;
  auto_expire_days: number;
  days_remaining: number;
}

const statusColors: Record<string, string> = {
  pending: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  approved: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  rejected: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  expired: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
};

const riskColors: Record<string, string> = {
  low: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function ScopeLifecyclePage() {
  const t = useTranslations();
  const [requests, setRequests] = useState<ScopeRequest[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/scope-lifecycle", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setRequests(d.requests || d || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const pending = requests.filter((r) => r.status === "pending").length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><ShieldCheck className="w-6 h-6 text-purple-500" />{t("scopeLifecycle.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">OAuth scope request approval workflow with risk assessment.</p>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Pending</span><p className="text-xl font-bold text-yellow-600 mt-1">{pending}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Approved</span><p className="text-xl font-bold text-green-600 mt-1">{requests.filter((r) => r.status === "approved").length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total</span><p className="text-xl font-bold mt-1">{requests.length}</p></div>
      </div>

      <div className="space-y-3">
        {requests.map((req) => (
          <div key={req.id} className="rounded-lg border dark:border-gray-800 p-4">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2"><span className="font-mono text-sm font-medium">{req.scope}</span><span className={`px-2 py-0.5 rounded text-xs ${riskColors[req.risk_level]}`}>{req.risk_level} risk</span></div>
              <span className={`px-2 py-0.5 rounded text-xs ${statusColors[req.status]}`}>{req.status}</span>
            </div>
            <div className="flex items-center gap-3 text-xs text-gray-500 mb-2"><span>Requested by: {req.requester}</span><span>at: {req.requested_at}</span></div>
            <div className="space-y-1 mb-2">{req.approver_chain.map((a, i) => (
              <div key={i} className="flex items-center gap-2 text-xs"><span className="font-mono text-gray-400">Step {i + 1}:</span><span>{a.approver}</span><span className={`px-1.5 py-0.5 rounded ${statusColors[a.status]}`}>{a.status}</span>{a.acted_at && <span className="text-gray-400">at {a.acted_at}</span>}</div>
            ))}</div>
            {req.status === "pending" && req.days_remaining >= 0 && (
              <div className="flex items-center gap-1 text-xs text-orange-600"><Clock className="w-3 h-3" /> Auto-expires in {req.days_remaining} days</div>
            )}
          </div>
        ))}
        {requests.length === 0 && !loading && <p className="text-sm text-gray-500 text-center py-8">No scope requests.</p>}
      </div>
    </div>
  );
}
