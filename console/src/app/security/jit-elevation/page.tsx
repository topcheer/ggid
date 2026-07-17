"use client";

import { useState, useCallback, useEffect } from "react";
import {
  ShieldCheck, Loader2, AlertCircle, X, RefreshCw, Clock, Check, XCircle,
  ArrowUpCircle, Timer,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface ElevationRequest {
  id: string;
  user_id: string;
  username: string;
  requested_role: string;
  duration_minutes: number;
  reason: string;
  status: "pending" | "approved" | "denied" | "active" | "expired" | "revoked";
  created_at: string;
  approved_by: string | null;
  expires_at: string | null;
}

const statusColors: Record<string, string> = {
  pending: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400",
  approved: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  denied: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
  active: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
  expired: "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400",
  revoked: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
};

export default function JITElevationPage() {
  const t = useTranslations();
  const [requests, setRequests] = useState<ElevationRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState<"all" | "pending" | "active" | "history">("pending");
  // Request form
  const [showForm, setShowForm] = useState(false);
  const [reqRole, setReqRole] = useState("");
  const [reqDuration, setReqDuration] = useState(60);
  const [reqReason, setReqReason] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [actingId, setActingId] = useState<string | null>(null);

  const loadRequests = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/policies/jit-elevate", {
        headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      }).catch(() => null);
      if (res?.ok) {
        const d = await res.json();
        setRequests(d.requests || d.items || []);
      }
    } catch { setError("Failed to load elevation requests"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadRequests(); }, [loadRequests]);

  const submitRequest = async () => {
    if (!reqRole || !reqReason) return;
    setSubmitting(true);
    try {
      const res = await fetch("/api/v1/policies/jit-elevate", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ requested_role: reqRole, duration_minutes: reqDuration, reason: reqReason }),
      });
      if (res.ok) {
        setShowForm(false); setReqRole(""); setReqReason(""); setReqDuration(60);
        loadRequests();
      } else { setError("Failed to submit elevation request"); }
    } catch { setError("Network error"); }
    finally { setSubmitting(false); }
  };

  const approve = async (id: string) => {
    setActingId(id);
    try {
      await fetch(`/api/v1/policies/jit-elevate/${id}/approve`, {
        method: "POST", headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      });
      loadRequests();
    } catch { setError("Failed to approve"); }
    finally { setActingId(null); }
  };

  const deny = async (id: string) => {
    setActingId(id);
    try {
      await fetch(`/api/v1/policies/jit-elevate/${id}/deny`, {
        method: "POST", headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      });
      loadRequests();
    } catch { setError("Failed to deny"); }
    finally { setActingId(null); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const filtered = requests.filter(r => {
    if (tab === "all") return true;
    if (tab === "pending") return r.status === "pending";
    if (tab === "active") return r.status === "active";
    return ["approved", "denied", "expired", "revoked"].includes(r.status);
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <ArrowUpCircle className="h-6 w-6 text-indigo-500" />
            PAM JIT Privilege Elevation
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Just-In-Time privilege elevation with approval workflow and time-bound access.</p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => setShowForm(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700">
            <ArrowUpCircle className="h-4 w-4" /> Request Elevation
          </button>
          <button onClick={loadRequests} disabled={loading} aria-label="Refresh" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
            <RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh
          </button>
        </div>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Tabs */}
      <div className="flex gap-2">
        {(["pending", "active", "history", "all"] as const).map(tb => (
          <button key={tb} onClick={() => setTab(tb)} aria-pressed={tab === tb} className={"rounded-lg px-3 py-1.5 text-sm font-medium capitalize transition " + (tab === tb ? "bg-indigo-600 text-white" : "border border-gray-300 text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800")}>
            {tb} {tab === tb && `(${filtered.length})`}
          </button>
        ))}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div> : filtered.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><ShieldCheck className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No {tab} elevation requests.</p></div></div>
      ) : (
        <div className="space-y-3">
          {filtered.map(r => (
            <div key={r.id} className={cardCls}>
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-gray-900 dark:text-white">{r.username || r.user_id}</span>
                    <span className={"px-2 py-0.5 rounded text-xs font-medium " + (statusColors[r.status] || "")}>{r.status}</span>
                  </div>
                  <div className="mt-1 flex flex-wrap items-center gap-3 text-xs text-gray-500">
                    <span>Role: <span className="font-mono font-medium">{r.requested_role}</span></span>
                    <span className="flex items-center gap-1"><Timer className="h-3 w-3" /> {r.duration_minutes}min</span>
                    <span className="flex items-center gap-1"><Clock className="h-3 w-3" /> {r.created_at ? new Date(r.created_at).toLocaleString() : "—"}</span>
                    {r.expires_at && <span className="flex items-center gap-1"><Clock className="h-3 w-3" /> Expires: {new Date(r.expires_at).toLocaleString()}</span>}
                  </div>
                  <p className="mt-1 text-sm text-gray-600 dark:text-gray-400">{r.reason}</p>
                  {r.approved_by && <p className="mt-1 text-xs text-gray-400">Approved by: {r.approved_by}</p>}
                </div>
                {r.status === "pending" && (
                  <div className="flex items-center gap-2">
                    <button onClick={() => approve(r.id)} disabled={actingId === r.id} aria-label={`Approve ${r.requested_role} for ${r.username}`} className="rounded-lg bg-green-50 px-3 py-1.5 text-xs font-medium text-green-700 hover:bg-green-100 dark:bg-green-900/20 disabled:opacity-50 flex items-center gap-1">{actingId === r.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Check className="h-3.5 w-3.5" />} Approve</button>
                    <button onClick={() => deny(r.id)} disabled={actingId === r.id} aria-label={`Deny ${r.requested_role} for ${r.username}`} className="rounded-lg bg-red-50 px-3 py-1.5 text-xs font-medium text-red-700 hover:bg-red-100 dark:bg-red-900/20 disabled:opacity-50 flex items-center gap-1"><XCircle className="h-3.5 w-3.5" /> Deny</button>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Request dialog */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><ArrowUpCircle className="h-5 w-5 text-indigo-500" /> Request Privilege Elevation</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">Requested Role *</label><input aria-label="Requested role" type="text" value={reqRole} onChange={e => setReqRole(e.target.value)} placeholder="admin, security-admin..." className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">Duration (minutes) *</label><input aria-label="Duration minutes" type="number" min={5} max={480} value={reqDuration} onChange={e => setReqDuration(parseInt(e.target.value) || 60)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              <div><label className="text-sm font-medium">Reason *</label><textarea aria-label="Elevation reason" value={reqReason} onChange={e => setReqReason(e.target.value)} placeholder="Incident response IR-1234 requires admin access..." rows={3} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button>
              <button onClick={submitRequest} disabled={!reqRole || !reqReason || submitting} className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : "Submit"}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
