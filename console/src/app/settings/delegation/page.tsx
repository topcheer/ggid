"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Users, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check,
  Shield, ChevronRight, Clock, Ban, Settings, Activity,
  CheckCircle2, XCircle, AlertTriangle, KeyRound, Lock,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface Delegation { id: string; delegator: string; delegate: string; scope: string; status: "active" | "expired" | "revoked"; expires_at: string; created_at: string; reason: string; }
interface DelegationConfig {
  max_delegation_depth: number; allowed_delegator_roles: string[]; delegation_expiry_hours: number;
  revocation_by_delegator: boolean; require_consent: boolean; audit_all_delegations: boolean;
  cascade_revoke_on_delegator_disable: boolean;
}

type Tab = "active" | "history" | "config";

const STATUS_CFG: Record<string, { label: string; color: string; bg: string }> = {
  active: { label: "Active", color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30" },
  expired: { label: "Expired", color: "text-gray-500", bg: "bg-gray-100 dark:bg-gray-800" },
  revoked: { label: "Revoked", color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30" },
};

export default function DelegationPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("active");
  const [delegations, setDelegations] = useState<Delegation[]>([]);
  const [config, setConfig] = useState<DelegationConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Create form
  const [showForm, setShowForm] = useState(false);
  const [dDelegator, setDDelegator] = useState("");
  const [dDelegate, setDDelegate] = useState("");
  const [dScope, setDScope] = useState("read:users");
  const [dReason, setDReason] = useState("");

  // Revoke
  const [confirmRevoke, setConfirmRevoke] = useState<string | null>(null);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [dRes, cRes] = await Promise.all([
        fetch("/api/v1/policies/delegations", { headers: h }).catch(() => null),
        fetch("/api/v1/policy/delegation/config", { headers: h }).catch(() => null),
      ]);
      if (dRes?.ok) { const d = await dRes.json(); setDelegations(d.delegations || []); }
      if (cRes?.ok) setConfig(await cRes.json());
    } catch { setError(t("delegation.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const createDelegation = async () => {
    if (!dDelegator || !dDelegate) return;
    setActionLoading("create");
    try {
      await fetch("/api/v1/policies/delegate", { method: "POST", headers: H, body: JSON.stringify({ delegator: dDelegator, delegate: dDelegate, scope: dScope, reason: dReason }) });
      setShowForm(false); setDDelegator(""); setDDelegate(""); setDReason("");
      loadData();
    } catch { setError(t("delegation.createError")); }
    finally { setActionLoading(null); }
  };

  const revokeDelegation = async (id: string) => {
    setActionLoading(`rvk-${id}`);
    try { await fetch(`/api/v1/policies/delegate?id=${id}`, { method: "DELETE", headers: h }); setConfirmRevoke(null); loadData(); }
    catch { setError(t("delegation.revokeError")); }
    finally { setActionLoading(null); }
  };

  const activeDelegations = delegations.filter(d => d.status === "active");
  const pastDelegations = delegations.filter(d => d.status !== "active");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Users className="h-6 w-6 text-purple-500" /> {t("delegation.title")}
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("delegation.subtitle")}</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "active" as Tab, label: `${t("delegation.active")} (${activeDelegations.length})`, icon: Shield },
          { id: "history" as Tab, label: t("delegation.history"), icon: Clock },
          { id: "config" as Tab, label: t("delegation.config"), icon: Settings },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-purple-600 text-purple-600 dark:text-purple-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-purple-500" /></div> : (<>

      {/* ════ ACTIVE ════ */}
      {tab === "active" && (
        <div>
          <div className="mb-4 flex items-center justify-between">
            <div className="grid grid-cols-3 gap-3">
              <div className="text-center"><p className="text-lg font-bold">{activeDelegations.length}</p><p className="text-xs text-gray-400">{t("delegation.activeCount")}</p></div>
              <div className="text-center"><p className="text-lg font-bold">{new Set(delegations.map(d => d.delegator)).size}</p><p className="text-xs text-gray-400">{t("delegation.delegators")}</p></div>
              <div className="text-center"><p className="text-lg font-bold">{new Set(delegations.map(d => d.delegate)).size}</p><p className="text-xs text-gray-400">{t("delegation.delegates")}</p></div>
            </div>
            <button onClick={() => setShowForm(true)} className="flex items-center gap-1 rounded-lg bg-purple-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-purple-700">
              <Plus className="h-3 w-3" /> {t("delegation.delegate")}
            </button>
          </div>
          {activeDelegations.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><Users className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("delegation.noActive")}</p></div></div>
          ) : (
            <div className="space-y-2">
              {activeDelegations.map(d => (
                <div key={d.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex items-center gap-3">
                    <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-purple-100 dark:bg-purple-900/30"><KeyRound className="h-4 w-4 text-purple-500" /></div>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-xs font-mono">{d.delegator}</span>
                        <ChevronRight className="h-3 w-3 text-gray-300" />
                        <span className="text-xs font-mono font-medium">{d.delegate}</span>
                      </div>
                      <div className="flex items-center gap-2 mt-0.5">
                        <span className="px-1.5 py-0.5 rounded bg-purple-100 dark:bg-purple-900/30 text-purple-600 text-xs font-mono">{d.scope}</span>
                        <span className="text-xs text-gray-400">{new Date(d.created_at).toLocaleDateString()}</span>
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={`px-1.5 py-0.5 rounded text-xs ${STATUS_CFG[d.status]?.bg} ${STATUS_CFG[d.status]?.color}`}>{STATUS_CFG[d.status]?.label}</span>
                    <button onClick={() => setConfirmRevoke(d.id)} disabled={actionLoading === `rvk-${d.id}`} aria-label="Revoke" className="rounded p-1 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20">
                      {actionLoading === `rvk-${d.id}` ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Ban className="h-3.5 w-3.5" />}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* ════ HISTORY ════ */}
      {tab === "history" && (
        <div className={card}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Clock className="h-4 w-4" /> {t("delegation.pastDelegations")}</h2>
          {pastDelegations.length === 0 ? (
            <div className="py-8 text-center"><Clock className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("delegation.noHistory")}</p></div>
          ) : (
            <div className="space-y-2">
              {pastDelegations.map(d => (
                <div key={d.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700 opacity-75">
                  <div className="flex items-center gap-3">
                    <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${STATUS_CFG[d.status]?.bg}`}>
                      {d.status === "revoked" ? <Ban className={`h-4 w-4 ${STATUS_CFG[d.status]?.color}`} /> : <Clock className={`h-4 w-4 ${STATUS_CFG[d.status]?.color}`} />}
                    </div>
                    <div>
                      <span className="text-xs font-mono">{d.delegator}</span>
                      <ChevronRight className="inline h-3 w-3 text-gray-300 mx-1" />
                      <span className="text-xs font-mono">{d.delegate}</span>
                      <p className="text-xs text-gray-400">{d.reason || "—"} · {new Date(d.created_at).toLocaleDateString()}</p>
                    </div>
                  </div>
                  <span className={`px-1.5 py-0.5 rounded text-xs ${STATUS_CFG[d.status]?.bg} ${STATUS_CFG[d.status]?.color}`}>{STATUS_CFG[d.status]?.label}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* ════ CONFIG ════ */}
      {tab === "config" && config && (
        <div className="space-y-4">
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Settings className="h-4 w-4" /> {t("delegation.policyConfig")}</h3>
            <div className="space-y-3">
              {([
                ["max_delegation_depth", t("delegation.maxDepth"), String(config.max_delegation_depth)],
                ["delegation_expiry_hours", t("delegation.expiryHours"), `${config.delegation_expiry_hours}h`],
                ["allowed_delegator_roles", t("delegation.allowedRoles"), config.allowed_delegator_roles.join(", ")],
              ] as const).map(([key, label, val]) => (
                <div key={key} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                  <span className="text-sm font-medium">{label}</span>
                  <span className="text-sm font-mono text-gray-500">{val}</span>
                </div>
              ))}
            </div>
          </div>
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("delegation.safeguards")}</h3>
            <div className="space-y-2">
              {([
                ["revocation_by_delegator", t("delegation.revocationByDelegator"), config.revocation_by_delegator],
                ["require_consent", t("delegation.requireConsent"), config.require_consent],
                ["audit_all_delegations", t("delegation.auditAll"), config.audit_all_delegations],
                ["cascade_revoke", t("delegation.cascadeRevoke"), config.cascade_revoke_on_delegator_disable],
              ] as const).map(([key, label, val]) => (
                <div key={key} className="flex items-center justify-between rounded-lg border p-2 dark:border-gray-700">
                  <span className="text-sm">{label}</span>
                  {val ? <CheckCircle2 className="h-4 w-4 text-green-500" /> : <XCircle className="h-4 w-4 text-gray-300" />}
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      </>)}

      {/* Create delegation modal */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-purple-500" /> {t("delegation.createDelegation")}</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">{t("delegation.delegator")}</label><input type="text" value={dDelegator} onChange={e => setDDelegator(e.target.value)} placeholder="user:alice" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" autoFocus /></div>
              <div><label className="text-sm font-medium">{t("delegation.delegate")}</label><input type="text" value={dDelegate} onChange={e => setDDelegate(e.target.value)} placeholder="user:bob" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">{t("delegation.scope")}</label>
                <select value={dScope} onChange={e => setDScope(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                  <option value="read:users">read:users</option><option value="write:users">write:users</option><option value="admin:orgs">admin:orgs</option><option value="read:audit">read:audit</option>
                </select>
              </div>
              <div><label className="text-sm font-medium">{t("delegation.reason")}</label><input type="text" value={dReason} onChange={e => setDReason(e.target.value)} placeholder="Vacation coverage" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button>
              <button onClick={createDelegation} disabled={!dDelegator || !dDelegate || actionLoading === "create"} className="rounded-lg bg-purple-600 px-4 py-2 text-sm font-medium text-white hover:bg-purple-700 disabled:opacity-50">
                {actionLoading === "create" ? <Loader2 className="h-4 w-4 animate-spin" /> : t("delegation.create")}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Revoke confirm */}
      {confirmRevoke && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmRevoke(null)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <div className="flex items-center gap-2"><AlertTriangle className="h-5 w-5 text-red-500" /><h3 className="text-lg font-semibold">{t("delegation.revokeTitle")}</h3></div>
            <p className="mt-3 text-sm text-gray-500 dark:text-gray-400">{t("delegation.revokeConfirm")}</p>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setConfirmRevoke(null)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button>
              <button onClick={() => revokeDelegation(confirmRevoke)} className="flex items-center gap-1 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"><Ban className="h-4 w-4" /> {t("delegation.revoke")}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
