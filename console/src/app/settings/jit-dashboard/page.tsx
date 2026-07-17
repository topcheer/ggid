"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Zap, Loader2, AlertCircle, X, RefreshCw, Plus, Check, Clock,
  Shield, Activity, User, ChevronRight, Ban, KeyRound, FileClock,
  CheckCircle2, XCircle, AlertTriangle, TrendingUp,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface JITRequest {
  id: string; user_id: string; role_id: string; reason: string;
  status: "pending" | "approved" | "rejected" | "expired" | "revoked";
  requested_at: string; decided_at: string | null;
  decided_by: string | null; expires_at: string | null;
  duration_minutes: number;
}

type Tab = "requests" | "active" | "submit" | "history";

const STATUS_CFG: Record<string, { label: string; color: string; bg: string; icon: typeof Clock }> = {
  pending: { label: "Pending", color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30", icon: Clock },
  approved: { label: "Approved", color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30", icon: CheckCircle2 },
  rejected: { label: "Rejected", color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30", icon: XCircle },
  expired: { label: "Expired", color: "text-gray-500", bg: "bg-gray-100 dark:bg-gray-800", icon: Clock },
  revoked: { label: "Revoked", color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30", icon: Ban },
};

export default function JITDashboardPage() {
  const [tab, setTab] = useState<Tab>("requests");
  const [requests, setRequests] = useState<JITRequest[]>([]);
  const [active, setActive] = useState<JITRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Submit form
  const [sUser, setSUser] = useState("");
  const [sRole, setSRole] = useState("");
  const [sReason, setSReason] = useState("");
  const [sDuration, setSDuration] = useState(60);
  const [submitting, setSubmitting] = useState(false);

  // Filter
  const [fStatus, setFStatus] = useState("all");

  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [rRes, aRes] = await Promise.all([
        fetch("/api/v1/policies/jit/requests", { headers: h }).catch(() => null),
        fetch("/api/v1/policies/jit/active", { headers: h }).catch(() => null),
      ]);
      if (rRes?.ok) { const d = await rRes.json(); setRequests(d.requests || d || []); }
      if (aRes?.ok) { const d = await aRes.json(); setActive(d.requests || d.elevations || []); }
      setError(null);
    } catch { setError("Failed to load JIT data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  // Auto-refresh active every 15s for TTL countdown
  useEffect(() => {
    if (tab !== "active") return;
    const timer = setInterval(async () => {
      const res = await fetch("/api/v1/policies/jit/active", { headers: h }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setActive(d.requests || d.elevations || []); }
    }, 15000);
    return () => clearInterval(timer);
  }, [tab]);

  const submitRequest = async () => {
    if (!sUser || !sRole) return;
    setSubmitting(true);
    try {
      await fetch("/api/v1/policies/jit/request", {
        method: "POST", headers: H,
        body: JSON.stringify({ user_id: sUser, role_id: sRole, reason: sReason, duration_minutes: sDuration }),
      });
      setSUser(""); setSRole(""); setSReason(""); setSDuration(60);
      setTab("requests");
      loadData();
    } catch { setError("Failed to submit request"); }
    finally { setSubmitting(false); }
  };

  const actOn = async (id: string, action: "approve" | "reject" | "revoke") => {
    setActionLoading(`${action}-${id}`);
    try {
      await fetch(`/api/v1/policies/jit/requests/${id}/${action}`, { method: "POST", headers: H });
      loadData();
    } catch { setError(`Failed to ${action} request`); }
    finally { setActionLoading(null); }
  };

  const fmtTTL = (expiresAt: string | null) => {
    if (!expiresAt) return "—";
    const ms = new Date(expiresAt).getTime() - Date.now();
    if (ms <= 0) return "expired";
    const m = Math.floor(ms / 60000); const s = Math.floor((ms % 60000) / 1000);
    return `${m}m ${s}s`;
  };

  const filteredRequests = requests.filter(r => fStatus === "all" || r.status === fStatus);
  const pendingCount = requests.filter(r => r.status === "pending").length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Zap className="h-6 w-6 text-amber-500" /> JIT Provisioning Dashboard
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Just-in-time access elevation — request, approve, and monitor time-limited role grants.
        </p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "requests" as Tab, label: `Requests (${pendingCount > 0 ? pendingCount + " pending" : "0"})`, icon: FileClock },
          { id: "active" as Tab, label: "Active Elevations", icon: Activity },
          { id: "submit" as Tab, label: "New Request", icon: Plus },
          { id: "history" as Tab, label: "History", icon: TrendingUp },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-amber-600 text-amber-600 dark:text-amber-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-amber-500" /></div> : (<>

      {/* ════ REQUESTS ════ */}
      {tab === "requests" && (
        <div>
          <div className="mb-4 flex items-center gap-2">
            <select value={fStatus} onChange={e => setFStatus(e.target.value)} aria-label="Filter status" className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1.5 text-sm">
              <option value="all">All Status</option>
              <option value="pending">Pending</option>
              <option value="approved">Approved</option>
              <option value="rejected">Rejected</option>
              <option value="expired">Expired</option>
              <option value="revoked">Revoked</option>
            </select>
          </div>
          {filteredRequests.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><FileClock className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No JIT requests found.</p></div></div>
          ) : (
            <div className="space-y-2">
              {filteredRequests.map(r => {
                const cfg = STATUS_CFG[r.status] || STATUS_CFG.pending;
                const SIcon = cfg.icon;
                return (
                  <div key={r.id} className={`${card} flex items-center justify-between`}>
                    <div className="flex items-center gap-3">
                      <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${cfg.bg}`}><SIcon className={`h-5 w-5 ${cfg.color}`} /></div>
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-sm">{r.user_id}</span>
                          <ChevronRight className="h-3 w-3 text-gray-300" />
                          <span className="px-1.5 py-0.5 rounded bg-amber-100 dark:bg-amber-900/30 text-amber-600 text-xs font-mono">{r.role_id}</span>
                          <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}>{cfg.label}</span>
                        </div>
                        <p className="text-xs text-gray-400 mt-0.5">{r.reason || "No reason provided"}</p>
                        <p className="text-xs text-gray-400">{new Date(r.requested_at).toLocaleString()} · {r.duration_minutes}m duration</p>
                      </div>
                    </div>
                    {r.status === "pending" && (
                      <div className="flex gap-1">
                        <button onClick={() => actOn(r.id, "approve")} disabled={actionLoading === `approve-${r.id}`}
                          aria-label={"Approve request for " + r.user_id}
                          className="flex items-center gap-1 rounded-lg bg-green-600 px-2 py-1 text-xs font-medium text-white hover:bg-green-700 disabled:opacity-50">
                          {actionLoading === `approve-${r.id}` ? <Loader2 className="h-3 w-3 animate-spin" /> : <Check className="h-3 w-3" />} Approve
                        </button>
                        <button onClick={() => actOn(r.id, "reject")} disabled={actionLoading === `reject-${r.id}`}
                          aria-label={"Reject request for " + r.user_id}
                          className="flex items-center gap-1 rounded-lg bg-red-600 px-2 py-1 text-xs font-medium text-white hover:bg-red-700 disabled:opacity-50">
                          {actionLoading === `reject-${r.id}` ? <Loader2 className="h-3 w-3 animate-spin" /> : <X className="h-3 w-3" />} Reject
                        </button>
                      </div>
                    )}
                    {r.status === "approved" && r.expires_at && (
                      <span className="text-xs text-gray-400">Expires: {fmtTTL(r.expires_at)}</span>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* ════ ACTIVE ELEVATIONS ════ */}
      {tab === "active" && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={card + " text-center"}><Activity className="mx-auto h-5 w-5 text-green-400" /><p className="mt-2 text-2xl font-bold">{active.length}</p><p className="text-xs text-gray-400">Active Elevations</p></div>
            <div className={card + " text-center"}><Clock className="mx-auto h-5 w-5 text-amber-400" /><p className="mt-2 text-2xl font-bold">{active.filter(a => fmtTTL(a.expires_at) !== "expired").length}</p><p className="text-xs text-gray-400">Still Valid</p></div>
            <div className={card + " text-center"}><User className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-2 text-2xl font-bold">{new Set(active.map(a => a.user_id)).size}</p><p className="text-xs text-gray-400">Unique Users</p></div>
            <div className={card + " text-center"}><Shield className="mx-auto h-5 w-5 text-purple-400" /><p className="mt-2 text-2xl font-bold">{new Set(active.map(a => a.role_id)).size}</p><p className="text-xs text-gray-400">Roles Elevated</p></div>
          </div>
          {active.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><Activity className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No active elevations. Auto-refresh every 15s.</p></div></div>
          ) : (
            <div className="space-y-2">
              {active.map(a => {
                const expired = a.expires_at ? new Date(a.expires_at).getTime() < Date.now() : false;
                return (
                  <div key={a.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                    <div className="flex items-center gap-3">
                      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-green-100 dark:bg-green-900/30"><KeyRound className="h-4 w-4 text-green-500" /></div>
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium">{a.user_id}</span>
                          <ChevronRight className="h-3 w-3 text-gray-300" />
                          <span className="px-1.5 py-0.5 rounded bg-amber-100 dark:bg-amber-900/30 text-amber-600 text-xs font-mono">{a.role_id}</span>
                        </div>
                        <p className="text-xs text-gray-400">Approved · {new Date(a.decided_at || a.requested_at).toLocaleString()}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-3">
                      <div className="text-right">
                        <p className={`text-xs font-bold ${expired ? "text-red-600" : "text-gray-500"}`}>{fmtTTL(a.expires_at)}</p>
                        <p className="text-xs text-gray-400">remaining</p>
                      </div>
                      {!expired && (
                        <button onClick={() => actOn(a.id, "revoke")} disabled={actionLoading === `revoke-${a.id}`}
                          aria-label={"Revoke elevation for " + a.user_id}
                          className="flex items-center gap-1 rounded-lg bg-red-600 px-2 py-1 text-xs font-medium text-white hover:bg-red-700 disabled:opacity-50">
                          {actionLoading === `revoke-${a.id}` ? <Loader2 className="h-3 w-3 animate-spin" /> : <Ban className="h-3 w-3" />} Revoke
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* ════ SUBMIT ════ */}
      {tab === "submit" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Plus className="h-4 w-4" /> Request JIT Elevation</h2>
            <div className="space-y-3">
              <div>
                <label className="text-sm font-medium">User ID</label>
                <input type="text" value={sUser} onChange={e => setSUser(e.target.value)} placeholder="user:alice" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
              </div>
              <div>
                <label className="text-sm font-medium">Role ID</label>
                <input type="text" value={sRole} onChange={e => setSRole(e.target.value)} placeholder="admin" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
              </div>
              <div>
                <label className="text-sm font-medium">Reason</label>
                <textarea value={sReason} onChange={e => setSReason(e.target.value)} placeholder="Emergency production access for incident INC-1234" rows={3} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" />
              </div>
              <div>
                <label className="text-sm font-medium">Duration (minutes)</label>
                <div className="mt-1 flex gap-2">
                  {[30, 60, 120, 240, 480].map(d => (
                    <button key={d} onClick={() => setSDuration(d)} aria-pressed={sDuration === d}
                      className={`rounded-lg border px-3 py-1.5 text-sm ${sDuration === d ? "border-amber-500 bg-amber-50 dark:bg-amber-950/30 text-amber-600" : "border-gray-300 dark:border-gray-700"}`}>
                      {d < 60 ? `${d}m` : `${d / 60}h`}
                    </button>
                  ))}
                </div>
              </div>
              <button onClick={submitRequest} disabled={!sUser || !sRole || submitting}
                className="flex items-center gap-2 rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white hover:bg-amber-700 disabled:opacity-50">
                {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />} Submit Request
              </button>
            </div>
          </div>
          <div className={card}>
            <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Shield className="h-4 w-4" /> JIT Policy</h2>
            <div className="space-y-2 text-xs text-gray-500 dark:text-gray-400">
              <div className="flex items-start gap-2"><Check className="h-3.5 w-3.5 text-green-500 mt-0.5" /><span>Requests require approval from a designated approver</span></div>
              <div className="flex items-start gap-2"><Check className="h-3.5 w-3.5 text-green-500 mt-0.5" /><span>Elevations auto-expire at the specified duration</span></div>
              <div className="flex items-start gap-2"><Check className="h-3.5 w-3.5 text-green-500 mt-0.5" /><span>All actions are audit-logged with reason</span></div>
              <div className="flex items-start gap-2"><Check className="h-3.5 w-3.5 text-green-500 mt-0.5" /><span>Active elevations can be revoked at any time</span></div>
              <div className="flex items-start gap-2"><AlertTriangle className="h-3.5 w-3.5 text-amber-500 mt-0.5" /><span>Maximum duration: 8 hours (480 minutes)</span></div>
            </div>
          </div>
        </div>
      )}

      {/* ════ HISTORY ════ */}
      {tab === "history" && (
        <div className={card}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><TrendingUp className="h-4 w-4" /> Elevation History</h2>
          {requests.length === 0 ? (
            <div className="py-8 text-center"><TrendingUp className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No history yet.</p></div>
          ) : (
            <div className="space-y-2">
              {requests.filter(r => r.status !== "pending").sort((a, b) =>
                new Date(b.requested_at).getTime() - new Date(a.requested_at).getTime()
              ).map(r => {
                const cfg = STATUS_CFG[r.status] || STATUS_CFG.pending;
                const SIcon = cfg.icon;
                return (
                  <div key={r.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                    <div className="flex items-center gap-3">
                      <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${cfg.bg}`}><SIcon className={`h-4 w-4 ${cfg.color}`} /></div>
                      <div>
                        <span className="text-sm font-medium">{r.user_id}</span>
                        <span className="mx-1 text-gray-300">→</span>
                        <span className="px-1.5 py-0.5 rounded bg-amber-100 dark:bg-amber-900/30 text-amber-600 text-xs font-mono">{r.role_id}</span>
                        <p className="text-xs text-gray-400 mt-0.5">{new Date(r.requested_at).toLocaleString()}</p>
                      </div>
                    </div>
                    <div className="text-right">
                      <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}>{cfg.label}</span>
                      {r.decided_by && <p className="text-xs text-gray-400 mt-0.5">by {r.decided_by}</p>}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      </>)}
    </div>
  );
}
