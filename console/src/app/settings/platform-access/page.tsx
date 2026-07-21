"use client";
import { useState, useEffect, useCallback } from "react";
import {
  Shield, ShieldCheck, ShieldX, Loader2, CheckCircle2, XCircle,
  Clock, User, AlertCircle, Plus, Copy, History,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { usePageTitle } from "@/lib/usePageTitle";
import { authHeader } from "@/lib/auth-helpers";
import { DEFAULT_TENANT_ID } from "@/lib/api-config";
import { API_BASE_URL } from "@/lib/api-config";

interface Consent {
  id: string;
  tenant_id: string;
  granted_to: string;
  granted_by: string;
  scope: string;
  expires_at: string | null;
  revoked_at: string | null;
  reason: string;
  created_at: string;
}

interface ImpersonationLog {
  id: string;
  impersonator_id: string;
  target_user_id: string | null;
  reason: string;
  started_at: string;
  ended_at: string | null;
  ip_address: string;
}

const API_BASE = API_BASE_URL;

export default function PlatformAccessPage() {
  usePageTitle("Platform Access");
  const t = useTranslations();
  const [consents, setConsents] = useState<Consent[]>([]);
  const [logs, setLogs] = useState<ImpersonationLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showGrant, setShowGrant] = useState(false);
  const [granting, setGranting] = useState(false);

  // Grant form
  const [scope, setScope] = useState("support");
  const [expiresIn, setExpiresIn] = useState("24h");
  const [reason, setReason] = useState("");
  const [success, setSuccess] = useState("");

  const tenantId = typeof window !== "undefined" ? localStorage.getItem("ggid_tenant_id") || DEFAULT_TENANT_ID : DEFAULT_TENANT_ID;

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [consentRes, logRes] = await Promise.all([
        fetch(`${API_BASE}/api/v1/tenants/${tenantId}/access`, { headers: { ...authHeader() } }),
        fetch(`${API_BASE}/api/v1/tenants/${tenantId}/access/logs`, { headers: { ...authHeader() } }).catch(() => null),
      ]);

      if (consentRes.ok) {
        const d = await consentRes.json();
        setConsents(d.consents || d.items || (Array.isArray(d) ? d : []));
      }

      if (logRes && logRes.ok) {
        const d = await logRes.json();
        setLogs(d.logs || d.sessions || d.items || (Array.isArray(d) ? d : []));
      }
    } catch {
      setError("Failed to load platform access data");
    }
    setLoading(false);
  }, [tenantId]);

  useEffect(() => { load(); }, [load]);

  const handleGrant = async () => {
    if (!reason.trim()) { setError("Reason is required"); return; }
    setGranting(true);
    setError("");
    try {
      const expiresAt = expiresIn !== "never" ? new Date(Date.now() + parseDuration(expiresIn)).toISOString() : null;
      const res = await fetch(`${API_BASE}/api/v1/tenants/${tenantId}/access/grant`, {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ scope, expires_at: expiresAt, reason }),
      });
      if (res.ok) {
        setSuccess("Platform access granted successfully.");
        setTimeout(() => setSuccess(""), 3000);
        setShowGrant(false);
        setReason("");
        await load();
      } else {
        const d = await res.json().catch(() => ({}));
        setError(d.error?.message || "Failed to grant access");
      }
    } catch {
      setError("Network error");
    }
    setGranting(false);
  };

  const handleRevoke = async (consentId: string) => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/tenants/${tenantId}/access/${consentId}`, {
        method: "DELETE", headers: { ...authHeader() },
      });
      if (res.ok) {
        await load();
      } else {
        setError("Failed to revoke access");
      }
    } catch {
      setError("Network error");
    }
  };

  const copyRequestLink = () => {
    const url = `${window.location.origin}/admin/impersonate?tenant=${tenantId}`;
    navigator.clipboard.writeText(url);
    setSuccess("Request link copied to clipboard.");
    setTimeout(() => setSuccess(""), 3000);
  };

  if (loading) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl p-6">
      <h1 className="mb-1 text-2xl font-bold text-gray-900 dark:text-white dark:text-white">Platform Access</h1>
      <p className="mb-6 text-sm text-gray-500">
        Manage platform administrator access to your tenant. All access is logged and audited.
      </p>

      {error && (
        <div className="mb-4 flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950">
          <AlertCircle className="h-4 w-4 shrink-0" /> {error}
        </div>
      )}
      {success && (
        <div className="mb-4 flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950">
          <CheckCircle2 className="h-4 w-4 shrink-0" /> {success}
        </div>
      )}

      {/* Status Card */}
      <div className="mb-6 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6 dark:border-gray-800 dark:bg-gray-900">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {consents.filter(c => !c.revoked_at).length > 0 ? (
              <ShieldCheck className="h-8 w-8 text-green-500" />
            ) : (
              <ShieldX className="h-8 w-8 text-gray-400" />
            )}
            <div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white dark:text-white">
                {consents.filter(c => !c.revoked_at).length > 0 ? "Access Granted" : "No Active Access"}
              </h2>
              <p className="text-sm text-gray-500">
                {consents.filter(c => !c.revoked_at).length} active consent(s)
              </p>
            </div>
          </div>
          <div className="flex gap-2">
            <button onClick={copyRequestLink} className="flex items-center gap-1.5 rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm dark:border-gray-700">
              <Copy className="h-4 w-4" /> Copy Request Link
            </button>
            <button onClick={() => setShowGrant(true)} className="flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700">
              <Plus className="h-4 w-4" /> Grant Access
            </button>
          </div>
        </div>
      </div>

      {/* Grant Modal */}
      {showGrant && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-md rounded-xl bg-white dark:bg-gray-800 p-6 dark:bg-gray-900 mx-4">
            <h3 className="mb-4 text-lg font-semibold">Grant Platform Access</h3>
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300 dark:text-gray-300">Scope</label>
                <select value={scope} onChange={e => setScope(e.target.value)} className="mt-1 w-full rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm dark:border-gray-700 dark:bg-gray-800">
                  <option value="support">Support (read-only)</option>
                  <option value="audit">Audit (read audit logs)</option>
                  <option value="full">Full (read/write)</option>
                </select>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300 dark:text-gray-300">Expires In</label>
                <select value={expiresIn} onChange={e => setExpiresIn(e.target.value)} className="mt-1 w-full rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm dark:border-gray-700 dark:bg-gray-800">
                  <option value="1h">1 hour</option>
                  <option value="4h">4 hours</option>
                  <option value="24h">24 hours</option>
                  <option value="7d">7 days</option>
                  <option value="never">Never (manual revoke only)</option>
                </select>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300 dark:text-gray-300">Reason <span className="text-red-500">*</span></label>
                <textarea value={reason} onChange={e => setReason(e.target.value)} required rows={3} placeholder="e.g., Investigating user login issues reported on 2024-01-15" className="mt-1 w-full rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm dark:border-gray-700 dark:bg-gray-800" />
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => setShowGrant(false)} className="rounded-lg border border-gray-300 dark:border-gray-600 px-4 py-2 text-sm dark:border-gray-700">Cancel</button>
              <button onClick={handleGrant} disabled={granting || !reason.trim()} className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
                {granting ? <Loader2 className="h-4 w-4 animate-spin" /> : "Grant Access"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Active Consents */}
      <div className="mb-6 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 dark:border-gray-800 dark:bg-gray-900">
        <div className="border-b border-gray-200 dark:border-gray-700 px-6 py-4 dark:border-gray-800">
          <h3 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">
            <Shield className="h-4 w-4" /> Active Consents
          </h3>
        </div>
        <div className="divide-y divide-gray-100 dark:divide-gray-800">
          {consents.filter(c => !c.revoked_at).length === 0 ? (
            <p className="px-6 py-8 text-center text-sm text-gray-400">No active consents. Grant access to allow platform administrators to help with your tenant.</p>
          ) : (
            consents.filter(c => !c.revoked_at).map(c => (
              <div key={c.id} className="flex items-center justify-between px-6 py-4">
                <div className="flex items-center gap-3">
                  <div className={`rounded-full px-2 py-0.5 text-xs font-medium ${c.scope === "full" ? "bg-red-100 text-red-700 dark:bg-red-950" : c.scope === "audit" ? "bg-amber-100 text-amber-700 dark:bg-amber-950" : "bg-blue-100 text-blue-700 dark:bg-blue-950"}`}>
                    {c.scope}
                  </div>
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-white dark:text-white">{c.granted_to}</p>
                    <p className="text-xs text-gray-500">{c.reason}</p>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  {c.expires_at && (
                    <span className="flex items-center gap-1 text-xs text-gray-400">
                      <Clock className="h-3 w-3" /> {new Date(c.expires_at).toLocaleDateString()}
                    </span>
                  )}
                  <button onClick={() => handleRevoke(c.id)} className="rounded-lg border border-red-300 px-3 py-1 text-xs text-red-600 hover:bg-red-50 dark:border-red-800">
                    Revoke
                  </button>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Revoked Consents */}
      {consents.filter(c => c.revoked_at).length > 0 && (
        <div className="mb-6 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 dark:border-gray-800 dark:bg-gray-900">
          <div className="border-b border-gray-200 dark:border-gray-700 px-6 py-4 dark:border-gray-800">
            <h3 className="text-sm font-semibold uppercase text-gray-400">Revoked Consents</h3>
          </div>
          <div className="divide-y divide-gray-100 dark:divide-gray-800">
            {consents.filter(c => c.revoked_at).map(c => (
              <div key={c.id} className="flex items-center gap-3 px-6 py-3 opacity-60">
                <XCircle className="h-4 w-4 text-red-400" />
                <span className="text-sm text-gray-600 dark:text-gray-400 dark:text-gray-400">{c.granted_to} — revoked {c.revoked_at ? new Date(c.revoked_at).toLocaleDateString() : ""}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Access Logs */}
      <div className="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 dark:border-gray-800 dark:bg-gray-900">
        <div className="border-b border-gray-200 dark:border-gray-700 px-6 py-4 dark:border-gray-800">
          <h3 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400">
            <History className="h-4 w-4" /> Access History
          </h3>
        </div>
        <div className="divide-y divide-gray-100 dark:divide-gray-800">
          {logs.length === 0 ? (
            <p className="px-6 py-8 text-center text-sm text-gray-400">No platform access sessions recorded.</p>
          ) : (
            logs.map(l => (
              <div key={l.id} className="flex items-center justify-between px-6 py-3">
                <div className="flex items-center gap-3">
                  <User className="h-4 w-4 text-gray-400" />
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-white dark:text-white">{l.impersonator_id.substring(0, 8)}</p>
                    <p className="text-xs text-gray-500">{l.reason}</p>
                  </div>
                </div>
                <div className="text-right text-xs text-gray-400">
                  <p>{l.started_at ? new Date(l.started_at).toLocaleString() : "—"}</p>
                  <p>{l.ip_address || ""}</p>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

function parseDuration(d: string): number {
  const m = d.match(/^(\d+)([hdm])$/);
  if (!m) return 86400000; // default 24h
  const n = parseInt(m[1]);
  const unit = m[2];
  return n * (unit === "h" ? 3600000 : unit === "d" ? 86400000 : 60000);
}
