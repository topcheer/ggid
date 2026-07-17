"use client";

import { useState, useCallback, useEffect } from "react";
import {
  AlertOctagon, Loader2, AlertCircle, X, RefreshCw, Clock,
  ShieldAlert, Check,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface BreakGlassRecord {
  id: string;
  user_id: string;
  username: string;
  reason: string;
  scope: string;
  duration_minutes: number;
  activated_at: string;
  expires_at: string;
  status: "active" | "expired";
  ip_address: string;
}

export default function BreakGlassPage() {
  const t = useTranslations();
  const [records, setRecords] = useState<BreakGlassRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState("");
  // Activate form
  const [showActivate, setShowActivate] = useState(false);
  const [reason, setReason] = useState("");
  const [scope, setScope] = useState("full");
  const [duration, setDuration] = useState(30);
  const [confirmText, setConfirmText] = useState("");
  const [activating, setActivating] = useState(false);

  const loadHistory = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/auth/break-glass/history", {
        headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      }).catch(() => null);
      if (res?.ok) {
        const d = await res.json();
        setRecords(d.records || d.history || []);
      }
    } catch { setError("Failed to load break-glass history"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadHistory(); }, [loadHistory]);

  const activate = async () => {
    if (!reason || confirmText !== "CONFIRM") return;
    setActivating(true);
    try {
      const res = await fetch("/api/v1/auth/break-glass/activate", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ reason, scope, duration_minutes: duration }),
      });
      if (res.ok) {
        setShowActivate(false); setReason(""); setConfirmText(""); setDuration(30); setScope("full");
        setSuccess("Break-glass access activated. All actions are being audited.");
        setTimeout(() => setSuccess(""), 5000);
        loadHistory();
      } else { setError("Failed to activate break-glass"); }
    } catch { setError("Network error"); }
    finally { setActivating(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const activeRecords = records.filter(r => r.status === "active");

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <AlertOctagon className="h-6 w-6 text-red-500" />
            Break-Glass Emergency Access
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Emergency privileged access for incident response. All actions are fully audited.</p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => setShowActivate(true)} className="flex items-center gap-2 rounded-lg bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700">
            <ShieldAlert className="h-4 w-4" /> Activate Break-Glass
          </button>
          <button onClick={loadHistory} disabled={loading} aria-label="Refresh" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
            <RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh
          </button>
        </div>
      </div>

      {/* Active break-glass warning */}
      {activeRecords.length > 0 && (
        <div role="alert" className="flex items-start gap-3 rounded-xl border border-red-300 bg-red-50 p-4 dark:border-red-800 dark:bg-red-950/30">
          <ShieldAlert className="h-5 w-5 shrink-0 text-red-600" />
          <div>
            <p className="text-sm font-bold text-red-800 dark:text-red-400">{activeRecords.length} active break-glass session(s)</p>
            {activeRecords.map(r => (
              <p key={r.id} className="mt-1 text-xs text-red-700 dark:text-red-500">
                {r.username} — {r.reason} — expires {r.expires_at ? new Date(r.expires_at).toLocaleString() : "—"}
              </p>
            ))}
          </div>
        </div>
      )}

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}
      {success && <div role="status" className="flex items-center gap-2 rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700 dark:bg-green-900/20 dark:text-green-400"><Check className="h-4 w-4 shrink-0" />{success}</div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-red-500" /></div> : records.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><AlertOctagon className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No break-glass activations recorded.</p></div></div>
      ) : (
        <div className={cardCls}>
          <h2 className="mb-4 text-sm font-semibold uppercase text-gray-400">Activation History</h2>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-900/50">
                <tr>
                  <th scope="col" className="px-4 py-3 text-left font-medium">User</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Reason</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Scope</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Duration</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Activated</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Expires</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-gray-800">
                {records.map(r => (
                  <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                    <td className="px-4 py-3 text-xs font-medium">{r.username || r.user_id}</td>
                    <td className="px-4 py-3 text-xs">{r.reason}</td>
                    <td className="px-4 py-3 text-xs font-mono">{r.scope}</td>
                    <td className="px-4 py-3 text-xs">{r.duration_minutes}min</td>
                    <td className="px-4 py-3 text-xs text-gray-500">{r.activated_at ? new Date(r.activated_at).toLocaleString() : "—"}</td>
                    <td className="px-4 py-3 text-xs text-gray-500">{r.expires_at ? new Date(r.expires_at).toLocaleString() : "—"}</td>
                    <td className="px-4 py-3"><span className={"px-2 py-0.5 rounded text-xs font-medium " + (r.status === "active" ? "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400" : "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400")}>{r.status}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Activate dialog */}
      {showActivate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowActivate(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-red-700 dark:text-red-400"><ShieldAlert className="h-5 w-5" /> Activate Break-Glass</h3>
            <div className="mt-3 flex items-start gap-2 rounded-lg bg-red-50 p-3 dark:bg-red-950/30">
              <AlertCircle className="h-4 w-4 shrink-0 text-red-600 mt-0.5" />
              <p className="text-xs text-red-700 dark:text-red-400">This grants emergency privileged access. ALL actions will be logged and reviewed by SOC. Use only for genuine incidents.</p>
            </div>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">Reason *</label><textarea aria-label="Break-glass reason" value={reason} onChange={e => setReason(e.target.value)} placeholder="Production outage — need admin access to restart services..." rows={3} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium">Scope</label><select aria-label="Break-glass scope" value={scope} onChange={e => setScope(e.target.value)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="full">Full Admin</option><option value="identity">Identity Only</option><option value="policy">Policy Only</option><option value="audit">Audit Only</option></select></div>
                <div><label className="text-sm font-medium">Duration (min)</label><input aria-label="Duration" type="number" min={5} max={120} value={duration} onChange={e => setDuration(parseInt(e.target.value) || 30)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              </div>
              <div><label className="text-sm font-medium">Type CONFIRM to proceed</label><input aria-label="Confirm text" type="text" value={confirmText} onChange={e => setConfirmText(e.target.value)} placeholder="CONFIRM" className="mt-1 w-full rounded-lg border border-red-300 dark:border-red-800 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowActivate(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button>
              <button onClick={activate} disabled={!reason || confirmText !== "CONFIRM" || activating} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">{activating ? <Loader2 className="h-4 w-4 animate-spin" /> : "Activate"}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
