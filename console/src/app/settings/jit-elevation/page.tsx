"use client";

import { useState, useEffect, useCallback } from "react";
import { Zap, Clock, Check, X, AlertTriangle, User } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ElevationRequest {
  id: string;
  user_id: string;
  username: string;
  role: string;
  duration_minutes: number;
  justification: string;
  status: "pending" | "approved" | "active" | "rejected" | "expired";
  requested_at: string;
  expires_at: string;
  remaining_minutes: number;
}

const statusColors: Record<string, string> = {
  pending: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  approved: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  active: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  rejected: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  expired: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
};

export default function JITElevationPage() {
  const t = useTranslations();

  const [requests, setRequests] = useState<ElevationRequest[]>([]);
  const [loading, setLoading] = useState(false);
  const [form, setForm] = useState({ role: "", duration_minutes: 60, justification: "" });
  const [submitting, setSubmitting] = useState(false);
  const [actionId, setActionId] = useState<string | null>(null);

  const fetchRequests = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/jit-elevation", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setRequests(data.requests || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchRequests(); }, [fetchRequests]);

  const submitRequest = async () => {
    if (!form.role || !form.justification) return;
    setSubmitting(true);
    try {
      await fetch("/api/v1/policy/jit-elevation", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify(form),
      });
      setForm({ role: "", duration_minutes: 60, justification: "" });
      fetchRequests();
    } catch { /* noop */ }
    finally { setSubmitting(false); }
  };

  const approve = async (id: string) => {
    setActionId(id);
    try {
      await fetch(`/api/v1/policy/jit-elevation/${id}/approve`, { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      setRequests((prev) => prev.map((r) => r.id === id ? { ...r, status: "active" } : r));
    } catch { /* noop */ }
    finally { setActionId(null); }
  };

  const reject = async (id: string) => {
    setActionId(id);
    try {
      await fetch(`/api/v1/policy/jit-elevation/${id}/reject`, { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      setRequests((prev) => prev.map((r) => r.id === id ? { ...r, status: "rejected" } : r));
    } catch { /* noop */ }
    finally { setActionId(null); }
  };

  const pending = requests.filter((r) => r.status === "pending");
  const active = requests.filter((r) => r.status === "active");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Zap className="w-6 h-6 text-yellow-500" /> {t("jitElevation.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Request time-bound privileged access with approval workflow.</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Request form */}
        <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
          <h3 className="font-semibold flex items-center gap-2"><Zap className="w-4 h-4 text-yellow-500" /> Request Elevation</h3>
          <div>
            <label className="text-sm font-medium">Role</label>
            <input aria-label="admin" type="text" value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })} placeholder="admin" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm font-mono" />
          </div>
          <div>
            <label className="text-sm font-medium">Duration (minutes)</label>
            <input aria-label="form" type="number" value={form.duration_minutes} onChange={(e) => setForm({ ...form, duration_minutes: parseInt(e.target.value) || 60 })} min={5} max={480} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" />
          </div>
          <div>
            <label className="text-sm font-medium">Justification</label>
            <textarea aria-label="Need admin access to fix production incident..." value={form.justification} onChange={(e) => setForm({ ...form, justification: e.target.value })} rows={3} placeholder="Need admin access to fix production incident..." className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" />
          </div>
          <button onClick={submitRequest} disabled={submitting || !form.role || !form.justification} className="w-full px-4 py-2 rounded-lg bg-yellow-600 text-white text-sm font-medium hover:bg-yellow-700 disabled:opacity-50">{submitting ? "Submitting..." : "Request Elevation"}</button>
        </div>

        {/* Pending queue */}
        <div className="rounded-lg border dark:border-gray-800">
          <div className="px-4 py-3 border-b dark:border-gray-800"><h3 className="font-semibold flex items-center gap-2"><Clock className="w-4 h-4 text-yellow-500" /> Pending ({pending.length})</h3></div>
          <div className="divide-y dark:divide-gray-800 max-h-80 overflow-y-auto">
            {pending.map((r) => (
              <div key={r.id} className="px-4 py-3">
                <div className="flex items-center gap-2 mb-1">
                  <User className="w-3 h-3 text-gray-400" />
                  <span className="text-sm font-medium">{r.username}</span>
                  <span className="px-2 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400 font-mono">{r.role}</span>
                </div>
                <p className="text-xs text-gray-400 mb-1">{r.justification}</p>
                <div className="flex items-center gap-2 text-xs text-gray-400"><Clock className="w-3 h-3" />{r.duration_minutes}min · {r.requested_at}</div>
                <div className="flex items-center gap-2 mt-2">
                  <button onClick={() => approve(r.id)} disabled={actionId === r.id} className="px-3 py-1 rounded text-xs font-medium text-green-700 bg-green-50 dark:bg-green-900/20 hover:bg-green-100 flex items-center gap-1"><Check className="w-3 h-3" /> Approve</button>
                  <button onClick={() => reject(r.id)} disabled={actionId === r.id} className="px-3 py-1 rounded text-xs font-medium text-red-700 bg-red-50 dark:bg-red-900/20 hover:bg-red-100 flex items-center gap-1"><X className="w-3 h-3" /> Reject</button>
                </div>
              </div>
            ))}
            {pending.length === 0 && <p className="px-4 py-6 text-center text-sm text-gray-500">No pending requests.</p>}
          </div>
        </div>

        {/* Active elevations */}
        <div className="rounded-lg border dark:border-gray-800">
          <div className="px-4 py-3 border-b dark:border-gray-800"><h3 className="font-semibold flex items-center gap-2"><Zap className="w-4 h-4 text-green-500" /> Active ({active.length})</h3></div>
          <div className="divide-y dark:divide-gray-800 max-h-80 overflow-y-auto">
            {active.map((r) => (
              <div key={r.id} className="px-4 py-3">
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-sm font-medium">{r.username}</span>
                  <span className="px-2 py-0.5 rounded text-xs bg-green-100 dark:bg-green-900/30 dark:text-green-400 font-mono">{r.role}</span>
                </div>
                <div className="flex items-center justify-between mt-1">
                  <div className="relative w-16 h-16">
                    <svg viewBox="0 0 64 64" className="w-full h-full -rotate-90">
                      <circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={5} className="text-gray-200 dark:text-gray-800" />
                      <circle cx={32} cy={32} r={28} fill="none" stroke="#10b981" strokeWidth={5} strokeDasharray={`${(r.remaining_minutes / r.duration_minutes) * 176} 176`} strokeLinecap="round" />
                    </svg>
                    <div className="absolute inset-0 flex items-center justify-center"><span className="text-sm font-bold text-green-600">{r.remaining_minutes}m</span></div>
                  </div>
                  <span className="text-xs text-gray-400">Expires {r.expires_at}</span>
                </div>
              </div>
            ))}
            {active.length === 0 && <p className="px-4 py-6 text-center text-sm text-gray-500">No active elevations.</p>}
          </div>
        </div>
      </div>

      {/* Full history table */}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50">
            <tr>
              <th scope="col" className="px-4 py-3 text-left font-medium">User</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Role</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Duration</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Justification</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Status</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Requested</th>
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {requests.slice(0, 20).map((r) => (
              <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3 font-medium">{r.username}</td>
                <td className="px-4 py-3 font-mono text-xs">{r.role}</td>
                <td className="px-4 py-3">{r.duration_minutes}m</td>
                <td className="px-4 py-3 max-w-xs truncate" title={r.justification}>{r.justification}</td>
                <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColors[r.status]}`}>{r.status}</span></td>
                <td className="px-4 py-3 text-gray-500">{r.requested_at}</td>
              </tr>
            ))}
            {requests.length === 0 && !loading && <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">No elevation requests.</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}
