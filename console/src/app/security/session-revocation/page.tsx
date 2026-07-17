"use client";

import { useState, useCallback, useEffect } from "react";
import {
  ShieldBan, Loader2, AlertCircle, X, RefreshCw, Ban, Clock,
  User, Globe, Smartphone, Search,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface RevokedSession {
  session_id: string;
  user_id: string;
  username: string;
  ip_address: string;
  device: string;
  revoked_at: string;
  revoked_by: string;
  reason: string;
  expires_at: string;
}

interface ActiveSession {
  session_id: string;
  user_id: string;
  username: string;
  ip_address: string;
  user_agent: string;
  created_at: string;
  last_active: string;
  risk_score: number;
}

export default function SessionRevocationPage() {
  const t = useTranslations();
  const [revoked, setRevoked] = useState<RevokedSession[]>([]);
  const [active, setActive] = useState<ActiveSession[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchUser, setSearchUser] = useState("");
  const [revokingId, setRevokingId] = useState<string | null>(null);
  const [revokeUser, setRevokeUser] = useState("");
  const [revokeReason, setRevokeReason] = useState("");
  const [showRevokeDialog, setShowRevokeDialog] = useState(false);
  const [success, setSuccess] = useState("");

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const headers = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [revokedRes, activeRes] = await Promise.all([
        fetch("/api/v1/auth/sessions/revoked", { headers }).catch(() => null),
        fetch("/api/v1/auth/sessions/active", { headers }).catch(() => null),
      ]);
      if (revokedRes?.ok) {
        const d = await revokedRes.json();
        setRevoked(d.sessions || d.revoked || []);
      }
      if (activeRes?.ok) {
        const d = await activeRes.json();
        setActive(d.sessions || d.active || []);
      }
    } catch {
      setError("Failed to load session data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const revokeSession = async (sessionId: string) => {
    setRevokingId(sessionId);
    try {
      const res = await fetch("/api/v1/auth/sessions/revoke", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ session_id: sessionId }),
      });
      if (res.ok) {
        setActive(prev => prev.filter(s => s.session_id !== sessionId));
        setSuccess("Session revoked successfully");
        setTimeout(() => setSuccess(""), 3000);
        loadData();
      }
    } catch {
      setError("Failed to revoke session");
    } finally {
      setRevokingId(null);
    }
  };

  const revokeAllUserSessions = async () => {
    if (!revokeUser) return;
    setRevokingId("user-" + revokeUser);
    try {
      const res = await fetch("/api/v1/auth/sessions/revoke-user", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ user_id: revokeUser, reason: revokeReason || "Manual revocation" }),
      });
      if (res.ok) {
        setSuccess(`All sessions revoked for ${revokeUser}`);
        setTimeout(() => setSuccess(""), 3000);
        setShowRevokeDialog(false);
        setRevokeUser("");
        setRevokeReason("");
        loadData();
      }
    } catch {
      setError("Failed to revoke user sessions");
    } finally {
      setRevokingId(null);
    }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const filteredActive = searchUser
    ? active.filter(s => s.username?.toLowerCase().includes(searchUser.toLowerCase()) || s.user_id.includes(searchUser))
    : active;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <ShieldBan className="h-6 w-6 text-red-500" />
            CAE Session Revocation
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Continuous Access Evaluation — view and revoke active/compromised sessions in real-time.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => setShowRevokeDialog(true)} className="flex items-center gap-2 rounded-lg bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700">
            <Ban className="h-4 w-4" /> Revoke User Sessions
          </button>
          <button onClick={loadData} disabled={loading} aria-label="Refresh sessions" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
            <RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh
          </button>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Success */}
      {success && (
        <div role="status" className="flex items-center gap-2 rounded-lg bg-green-50 px-4 py-3 text-sm text-green-700 dark:bg-green-900/20 dark:text-green-400">
          <ShieldBan className="h-4 w-4 shrink-0" />{success}
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-red-500" /></div>
      ) : (
        <>
          {/* Stats */}
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
            <div className={cardCls}>
              <div className="flex items-center gap-2"><User className="h-4 w-4 text-green-500" /><span className="text-xs font-semibold uppercase text-gray-400">Active</span></div>
              <p className="mt-2 text-2xl font-bold text-green-600">{active.length}</p>
            </div>
            <div className={cardCls}>
              <div className="flex items-center gap-2"><Ban className="h-4 w-4 text-red-500" /><span className="text-xs font-semibold uppercase text-gray-400">Revoked</span></div>
              <p className="mt-2 text-2xl font-bold text-red-600">{revoked.length}</p>
            </div>
            <div className={cardCls}>
              <div className="flex items-center gap-2"><Clock className="h-4 w-4 text-yellow-500" /><span className="text-xs font-semibold uppercase text-gray-400">High Risk</span></div>
              <p className="mt-2 text-2xl font-bold text-yellow-600">{active.filter(s => s.risk_score >= 70).length}</p>
            </div>
            <div className={cardCls}>
              <div className="flex items-center gap-2"><Globe className="h-4 w-4 text-blue-500" /><span className="text-xs font-semibold uppercase text-gray-400">Unique IPs</span></div>
              <p className="mt-2 text-2xl font-bold text-blue-600">{new Set(active.map(s => s.ip_address)).size}</p>
            </div>
          </div>

          {/* Active sessions */}
          <div className={cardCls}>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-sm font-semibold uppercase text-gray-400">Active Sessions</h2>
              <div className="relative">
                <Search className="absolute left-2 top-2.5 h-4 w-4 text-gray-400" />
                <input aria-label="Search sessions" type="text" value={searchUser} onChange={e => setSearchUser(e.target.value)} placeholder="Search user..." className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-8 pr-3 py-1.5 text-sm" />
              </div>
            </div>
            {filteredActive.length === 0 ? (
              <div className="py-8 text-center"><Smartphone className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No active sessions{searchUser ? " match your search" : ""}.</p></div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 dark:bg-gray-900/50">
                    <tr>
                      <th scope="col" className="px-4 py-3 text-left font-medium">User</th>
                      <th scope="col" className="px-4 py-3 text-left font-medium">IP</th>
                      <th scope="col" className="px-4 py-3 text-left font-medium">Device</th>
                      <th scope="col" className="px-4 py-3 text-left font-medium">Last Active</th>
                      <th scope="col" className="px-4 py-3 text-left font-medium">Risk</th>
                      <th scope="col" className="px-4 py-3 text-right font-medium">Action</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y dark:divide-gray-800">
                    {filteredActive.map(s => (
                      <tr key={s.session_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                        <td className="px-4 py-3"><span className="font-medium">{s.username || s.user_id}</span></td>
                        <td className="px-4 py-3 font-mono text-xs">{s.ip_address}</td>
                        <td className="px-4 py-3 text-xs text-gray-500 max-w-[200px] truncate">{s.user_agent}</td>
                        <td className="px-4 py-3 text-xs text-gray-500">{s.last_active ? new Date(s.last_active).toLocaleString() : "—"}</td>
                        <td className="px-4 py-3">
                          <span className={"px-2 py-0.5 rounded text-xs font-medium " + (s.risk_score >= 70 ? "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400" : s.risk_score >= 40 ? "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400" : "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400")}>
                            {s.risk_score ?? 0}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-right">
                          <button onClick={() => revokeSession(s.session_id)} disabled={revokingId === s.session_id} aria-label={`Revoke session for ${s.username}`} className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-50">
                            {revokingId === s.session_id ? <Loader2 className="h-4 w-4 animate-spin" /> : <Ban className="h-4 w-4" />}
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Recently revoked */}
          {revoked.length > 0 && (
            <div className={cardCls}>
              <h2 className="mb-4 text-sm font-semibold uppercase text-gray-400">Recently Revoked</h2>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 dark:bg-gray-900/50">
                    <tr>
                      <th scope="col" className="px-4 py-3 text-left font-medium">User</th>
                      <th scope="col" className="px-4 py-3 text-left font-medium">IP</th>
                      <th scope="col" className="px-4 py-3 text-left font-medium">Revoked At</th>
                      <th scope="col" className="px-4 py-3 text-left font-medium">By</th>
                      <th scope="col" className="px-4 py-3 text-left font-medium">Reason</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y dark:divide-gray-800">
                    {revoked.slice(0, 10).map(s => (
                      <tr key={s.session_id} className="opacity-70">
                        <td className="px-4 py-3 text-xs">{s.username || s.user_id}</td>
                        <td className="px-4 py-3 font-mono text-xs">{s.ip_address}</td>
                        <td className="px-4 py-3 text-xs text-gray-500">{s.revoked_at ? new Date(s.revoked_at).toLocaleString() : "—"}</td>
                        <td className="px-4 py-3 text-xs">{s.revoked_by || "system"}</td>
                        <td className="px-4 py-3 text-xs text-gray-500">{s.reason || "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      )}

      {/* Revoke all user sessions dialog */}
      {showRevokeDialog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowRevokeDialog(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Ban className="h-5 w-5 text-red-500" /> Revoke All User Sessions</h3>
            <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">This will immediately invalidate all active sessions for the specified user. They will need to re-authenticate.</p>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">User ID or Username</label><input aria-label="User to revoke" type="text" value={revokeUser} onChange={e => setRevokeUser(e.target.value)} placeholder="user-uuid or username" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">Reason (optional)</label><input aria-label="Revoke reason" type="text" value={revokeReason} onChange={e => setRevokeReason(e.target.value)} placeholder="Security incident..." className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowRevokeDialog(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button>
              <button onClick={revokeAllUserSessions} disabled={!revokeUser || !!revokingId} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">
                {revokingId ? <Loader2 className="h-4 w-4 animate-spin" /> : "Revoke All"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
