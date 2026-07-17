"use client";
import { useState, useCallback, useEffect } from "react";
import {
  ShieldCheck, Loader2, AlertCircle, X, RefreshCw, Plus, Check,
  Ban, Clock, FileText, TrendingUp, ChevronRight, User,
  CheckCircle2, XCircle, AlertTriangle, Lock, Download,
  Eye, ArrowRight, Hash,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

/* ─── Types matching backend ─── */
interface ConsentRecord {
  id: string; user_id: string; client_id?: string; purpose: string;
  scopes: string[]; status: "active" | "withdrawn" | "expired";
  policy_version: string; granted_at: string; expires_at?: string | null;
  withdrawn_at?: string | null; withdrawn_reason?: string;
  created_at: string;
}
interface ConsentSummary {
  records: ConsentRecord[]; total: number; total_active: number;
  total_expired: number; total_revoked: number;
  by_purpose: Record<string, number>;
}

type Tab = "overview" | "registry" | "policies" | "dsr";

const STATUS_CFG: Record<string, { label: string; color: string; bg: string; icon: typeof CheckCircle2 }> = {
  active: { label: "Active", color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30", icon: CheckCircle2 },
  withdrawn: { label: "Withdrawn", color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30", icon: XCircle },
  expired: { label: "Expired", color: "text-gray-500", bg: "bg-gray-100 dark:bg-gray-800", icon: Clock },
};

const PURPOSE_COLORS = ["#6366f1", "#22c55e", "#f59e0b", "#ef4444", "#8b5cf6", "#06b6d4", "#ec4899"];

export default function ConsentPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("overview");
  const [data, setData] = useState<ConsentSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Grant form
  const [showGrant, setShowGrant] = useState(false);
  const [gUser, setGUser] = useState("");
  const [gPurpose, setGPurpose] = useState("marketing");
  const [gScopes, setGScopes] = useState("read:profile");
  const [gClient, setGClient] = useState("");
  const [granting, setGranting] = useState(false);

  // Withdraw
  const [confirmWithdraw, setConfirmWithdraw] = useState<ConsentRecord | null>(null);
  const [withdrawReason, setWithdrawReason] = useState("");

  // Filter
  const [fStatus, setFStatus] = useState("all");
  const [fUser, setFUser] = useState("");

  // DSR
  const [dsrType, setDsrType] = useState("access");
  const [dsrUser, setDsrUser] = useState("");
  const [dsrRequests, setDsrRequests] = useState<{ id: string; type: string; user_id: string; status: string; created_at: string; sla_days_left: number }[]>([]);

  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (fUser) params.set("user_id", fUser);
      if (fStatus !== "all") params.set("status", fStatus);
      const res = await fetch(`/api/v1/identity/consent/registry?${params}`, { headers: h }).catch(() => null);
      if (res?.ok) setData(await res.json());
      setError(null);
    } catch { setError(t("consent.loadError")); }
    finally { setLoading(false); }
  }, [fUser, fStatus]);

  useEffect(() => { loadData(); }, [loadData]);

  const grantConsent = async () => {
    if (!gUser || !gPurpose) return;
    setGranting(true);
    try {
      await fetch("/api/v1/identity/consent/registry", {
        method: "POST", headers: H,
        body: JSON.stringify({
          user_id: gUser, purpose: gPurpose,
          scopes: gScopes.split(",").map(s => s.trim()).filter(Boolean),
          client_id: gClient || undefined,
        }),
      });
      setShowGrant(false); setGUser(""); setGPurpose("marketing"); setGScopes("read:profile"); setGClient("");
      loadData();
    } catch { setError(t("consent.grantError")); }
    finally { setGranting(false); }
  };

  const withdrawConsent = async () => {
    if (!confirmWithdraw) return;
    setActionLoading(confirmWithdraw.id);
    try {
      const params = new URLSearchParams({ id: confirmWithdraw.id, reason: withdrawReason || t("consent.defaultWithdrawReason") });
      await fetch(`/api/v1/identity/consent/registry?${params}`, { method: "DELETE", headers: h });
      setConfirmWithdraw(null); setWithdrawReason("");
      loadData();
    } catch { setError(t("consent.withdrawError")); }
    finally { setActionLoading(null); }
  };

  const submitDSR = () => {
    if (!dsrUser) return;
    setDsrRequests(prev => [{
      id: `dsr-${Date.now()}`, type: dsrType, user_id: dsrUser,
      status: "processing", created_at: new Date().toISOString(), sla_days_left: 30,
    }, ...prev]);
    setDsrUser("");
  };

  const records = data?.records || [];
  const filtered = records;
  const purposes = Object.entries(data?.by_purpose || {});
  const totalPurpose = purposes.reduce((a, [, v]) => a + v, 0) || 1;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <ShieldCheck className="h-6 w-6 text-blue-500" /> {t("consent.title")}
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("consent.subtitle")}</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "overview" as Tab, label: t("consent.overview"), icon: TrendingUp },
          { id: "registry" as Tab, label: t("consent.registry"), icon: FileText },
          { id: "policies" as Tab, label: t("consent.privacyPolicies"), icon: Lock },
          { id: "dsr" as Tab, label: t("consent.dsrRequests"), icon: Download },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-blue-600 text-blue-600 dark:text-blue-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div> : (<>

      {/* ════ OVERVIEW ════ */}
      {tab === "overview" && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={card}>
              <div className="flex items-center justify-between">
                <div><p className="text-xs text-gray-400">{t("consent.totalConsents")}</p><p className="mt-1 text-2xl font-bold">{data?.total ?? 0}</p></div>
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/30"><FileText className="h-5 w-5 text-blue-500" /></div>
              </div>
            </div>
            <div className={card}>
              <div className="flex items-center justify-between">
                <div><p className="text-xs text-gray-400">{t("consent.active")}</p><p className="mt-1 text-2xl font-bold text-green-600">{data?.total_active ?? 0}</p></div>
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-green-100 dark:bg-green-900/30"><CheckCircle2 className="h-5 w-5 text-green-500" /></div>
              </div>
            </div>
            <div className={card}>
              <div className="flex items-center justify-between">
                <div><p className="text-xs text-gray-400">{t("consent.withdrawn")}</p><p className="mt-1 text-2xl font-bold text-red-600">{data?.total_revoked ?? 0}</p></div>
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-red-100 dark:bg-red-900/30"><XCircle className="h-5 w-5 text-red-500" /></div>
              </div>
            </div>
            <div className={card}>
              <div className="flex items-center justify-between">
                <div><p className="text-xs text-gray-400">{t("consent.expired")}</p><p className="mt-1 text-2xl font-bold text-gray-500">{data?.total_expired ?? 0}</p></div>
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-800"><Clock className="h-5 w-5 text-gray-400" /></div>
              </div>
            </div>
          </div>

          {/* Purpose breakdown */}
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Hash className="h-4 w-4" /> {t("consent.purposeBreakdown")}</h3>
            {purposes.length > 0 ? (
              <div className="space-y-2">
                {purposes.map(([purpose, count], i) => {
                  const pct = Math.round((count / totalPurpose) * 100);
                  return (
                    <div key={purpose} className="flex items-center gap-3">
                      <span className="w-28 text-xs font-mono text-gray-500">{purpose}</span>
                      <div className="flex-1 h-5 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-700">
                        <div className="h-full rounded-full" style={{ width: `${pct}%`, backgroundColor: PURPOSE_COLORS[i % PURPOSE_COLORS.length] }} />
                      </div>
                      <span className="w-16 text-right text-xs font-mono">{count}</span>
                    </div>
                  );
                })}
              </div>
            ) : <p className="text-sm text-gray-400">{t("consent.noData")}</p>}
          </div>

          {/* Recent activity */}
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Clock className="h-4 w-4" /> {t("consent.recentActivity")}</h3>
            <div className="space-y-2">
              {records.slice(0, 8).map(r => {
                const cfg = STATUS_CFG[r.status] || STATUS_CFG.active;
                const SIcon = cfg.icon;
                return (
                  <div key={r.id} className="flex items-center justify-between rounded-lg border p-2 dark:border-gray-700">
                    <div className="flex items-center gap-2">
                      <SIcon className={`h-4 w-4 ${cfg.color}`} />
                      <span className="text-xs font-mono text-gray-500">{r.user_id}</span>
                      <ChevronRight className="h-3 w-3 text-gray-300" />
                      <span className="text-xs font-medium">{r.purpose}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-gray-400">{new Date(r.granted_at).toLocaleDateString()}</span>
                      <span className={`px-1.5 py-0.5 rounded text-xs ${cfg.bg} ${cfg.color}`}>{cfg.label}</span>
                    </div>
                  </div>
                );
              })}
              {records.length === 0 && <p className="text-sm text-gray-400">{t("consent.noActivity")}</p>}
            </div>
          </div>
        </div>
      )}

      {/* ════ REGISTRY ════ */}
      {tab === "registry" && (
        <div>
          <div className="mb-4 flex items-center justify-between gap-4 flex-wrap">
            <div className="flex items-center gap-2">
              <select value={fStatus} onChange={e => setFStatus(e.target.value)} aria-label="Filter status" className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1.5 text-sm">
                <option value="all">{t("consent.allStatuses")}</option>
                <option value="active">{t("consent.active")}</option>
                <option value="withdrawn">{t("consent.withdrawn")}</option>
                <option value="expired">{t("consent.expired")}</option>
              </select>
              <input type="text" value={fUser} onChange={e => setFUser(e.target.value)} placeholder={t("consent.filterByUser")} className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-1.5 text-sm w-48" />
            </div>
            <button onClick={() => setShowGrant(true)} className="flex items-center gap-1 rounded-lg bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700">
              <Plus className="h-3 w-3" /> {t("consent.grantConsent")}
            </button>
          </div>

          {filtered.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><FileText className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("consent.noConsents")}</p></div></div>
          ) : (
            <div className="overflow-x-auto"><table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-800/50"><tr>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">{t("consent.user")}</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">{t("consent.purpose")}</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">{t("consent.scopes")}</th>
                <th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">{t("consent.status")}</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">{t("consent.granted")}</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">{t("consent.expires")}</th>
                <th scope="col" className="px-3 py-2 text-right text-xs font-medium text-gray-400">{t("consent.actions")}</th>
              </tr></thead>
              <tbody className="divide-y dark:divide-gray-800">
                {filtered.map(r => {
                  const cfg = STATUS_CFG[r.status] || STATUS_CFG.active;
                  const SIcon = cfg.icon;
                  return (
                    <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                      <td className="px-3 py-3 text-xs font-mono">{r.user_id}</td>
                      <td className="px-3 py-3 text-xs font-medium">{r.purpose}</td>
                      <td className="px-3 py-3"><div className="flex flex-wrap gap-1 max-w-xs">{(r.scopes || []).map(s => <span key={s} className="px-1.5 py-0.5 rounded bg-blue-100 dark:bg-blue-900/30 text-blue-600 text-xs font-mono">{s}</span>)}{(!r.scopes || r.scopes.length === 0) && <span className="text-xs text-gray-300">—</span>}</div></td>
                      <td className="px-3 py-3 text-center"><span className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}><SIcon className="h-3 w-3" /> {cfg.label}</span></td>
                      <td className="px-3 py-3 text-xs text-gray-400">{new Date(r.granted_at).toLocaleDateString()}</td>
                      <td className="px-3 py-3 text-xs text-gray-400">{r.expires_at ? new Date(r.expires_at).toLocaleDateString() : "—"}</td>
                      <td className="px-3 py-3 text-right">
                        {r.status === "active" && (
                          <button onClick={() => setConfirmWithdraw(r)} aria-label={t("consent.withdrawConsent")} className="rounded-lg p-1.5 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"><Ban className="h-3.5 w-3.5" /></button>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table></div>
          )}
        </div>
      )}

      {/* ════ PRIVACY POLICIES ════ */}
      {tab === "policies" && (
        <div className="space-y-6">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Lock className="h-4 w-4" /> {t("consent.policyVersions")}</h2>
            <div className="space-y-2">
              {[
                { version: "2.1", date: "2025-01-15", active: true, changes: t("consent.policyChangeV21"), reconsent: true },
                { version: "2.0", date: "2024-09-01", active: false, changes: t("consent.policyChangeV20"), reconsent: true },
                { version: "1.5", date: "2024-03-15", active: false, changes: t("consent.policyChangeV15"), reconsent: false },
                { version: "1.0", date: "2023-06-01", active: false, changes: t("consent.policyChangeV10"), reconsent: false },
              ].map(p => (
                <div key={p.version} className={`flex items-center justify-between rounded-lg border p-3 dark:border-gray-700 ${p.active ? "border-blue-300 dark:border-blue-700 bg-blue-50 dark:bg-blue-950/20" : ""}`}>
                  <div className="flex items-center gap-3">
                    <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${p.active ? "bg-blue-100 dark:bg-blue-900/30" : "bg-gray-100 dark:bg-gray-700"}`}><FileText className={`h-4 w-4 ${p.active ? "text-blue-500" : "text-gray-400"}`} /></div>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-sm">v{p.version}</span>
                        {p.active && <span className="px-1.5 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900/30 text-blue-600 font-medium">{t("consent.activePolicy")}</span>}
                        {p.reconsent && !p.active && <span className="flex items-center gap-0.5 px-1.5 py-0.5 rounded text-xs bg-amber-100 dark:bg-amber-900/30 text-amber-600"><AlertTriangle className="h-2.5 w-2.5" /> {t("consent.reconsentRequired")}</span>}
                      </div>
                      <p className="text-xs text-gray-400">{new Date(p.date).toLocaleDateString()} · {p.changes}</p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Eye className="h-4 w-4" /> {t("consent.reconsentImpact")}</h3>
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
              <div className="text-center"><p className="text-2xl font-bold text-amber-600">{data?.total_active ?? 0}</p><p className="text-xs text-gray-400">{t("consent.usersNeedReconsent")}</p></div>
              <div className="text-center"><p className="text-2xl font-bold">{records.filter(r => r.policy_version === "2.1").length}</p><p className="text-xs text-gray-400">{t("consent.onLatestVersion")}</p></div>
              <div className="text-center"><p className="text-2xl font-bold">{records.filter(r => r.policy_version !== "2.1").length}</p><p className="text-xs text-gray-400">{t("consent.onOlderVersion")}</p></div>
            </div>
          </div>
        </div>
      )}

      {/* ════ DSR REQUESTS ════ */}
      {tab === "dsr" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Download className="h-4 w-4" /> {t("consent.newDsr")}</h2>
            <div className="space-y-3">
              <div>
                <label className="text-sm font-medium">{t("consent.requestType")}</label>
                <select value={dsrType} onChange={e => setDsrType(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                  <option value="access">{t("consent.dsrAccess")}</option>
                  <option value="deletion">{t("consent.dsrDeletion")}</option>
                  <option value="portability">{t("consent.dsrPortability")}</option>
                </select>
              </div>
              <div>
                <label className="text-sm font-medium">{t("consent.user")}</label>
                <input type="text" value={dsrUser} onChange={e => setDsrUser(e.target.value)} placeholder="user:alice" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" />
              </div>
              <div className="rounded-lg bg-blue-50 dark:bg-blue-900/20 p-3 text-xs text-blue-600 dark:text-blue-400">
                <p className="flex items-center gap-1"><Clock className="h-3 w-3" /> {t("consent.slaNotice")}</p>
              </div>
              <button onClick={submitDSR} disabled={!dsrUser} className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
                <Download className="h-4 w-4" /> {t("consent.submitDsr")}
              </button>
            </div>
          </div>
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><FileText className="h-4 w-4" /> {t("consent.dsrHistory")} ({dsrRequests.length})</h2>
            {dsrRequests.length === 0 ? (
              <div className="py-8 text-center"><Download className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("consent.noDsr")}</p></div>
            ) : (
              <div className="space-y-2">
                {dsrRequests.map(d => (
                  <div key={d.id} className="rounded-lg border p-3 dark:border-gray-700">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/30"><User className="h-4 w-4 text-blue-500" /></div>
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="text-xs font-mono">{d.user_id}</span>
                            <span className="px-1.5 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900/30 text-blue-600 font-mono">{d.type}</span>
                          </div>
                          <p className="text-xs text-gray-400">{new Date(d.created_at).toLocaleString()}</p>
                        </div>
                      </div>
                      <div className="text-right">
                        <span className="text-xs font-medium text-amber-600">{d.sla_days_left} {t("consent.daysLeft")}</span>
                        <p className="text-xs text-gray-400">{t("consent.slaRemaining")}</p>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      </>)}

      {/* Grant dialog */}
      {showGrant && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowGrant(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-blue-500" /> {t("consent.grantConsent")}</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">{t("consent.user")}</label><input type="text" value={gUser} onChange={e => setGUser(e.target.value)} placeholder="user:alice" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" autoFocus /></div>
              <div><label className="text-sm font-medium">{t("consent.purpose")}</label>
                <select value={gPurpose} onChange={e => setGPurpose(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                  <option value="marketing">{t("consent.purposeMarketing")}</option>
                  <option value="analytics">{t("consent.purposeAnalytics")}</option>
                  <option value="third_party_share">{t("consent.purposeThirdParty")}</option>
                  <option value="essential">{t("consent.purposeEssential")}</option>
                </select>
              </div>
              <div><label className="text-sm font-medium">{t("consent.scopes")}</label><input type="text" value={gScopes} onChange={e => setGScopes(e.target.value)} placeholder="read:profile, write:data" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">{t("consent.clientIdOptional")}</label><input type="text" value={gClient} onChange={e => setGClient(e.target.value)} placeholder="web-console" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowGrant(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button>
              <button onClick={grantConsent} disabled={!gUser || granting} className="flex items-center gap-1 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
                {granting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} {t("consent.grant")}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Withdraw dialog */}
      {confirmWithdraw && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmWithdraw(null)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <div className="flex items-center gap-2"><AlertTriangle className="h-5 w-5 text-red-500" /><h3 className="text-lg font-semibold">{t("consent.withdrawTitle")}</h3></div>
            <p className="mt-3 text-sm text-gray-500 dark:text-gray-400">{t("consent.withdrawConfirm")}</p>
            <div className="mt-3"><label className="text-sm font-medium">{t("consent.reason")}</label><input type="text" value={withdrawReason} onChange={e => setWithdrawReason(e.target.value)} placeholder={t("consent.reasonPlaceholder")} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setConfirmWithdraw(null)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button>
              <button onClick={withdrawConsent} disabled={actionLoading === confirmWithdraw.id} className="flex items-center gap-1 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">
                {actionLoading === confirmWithdraw.id ? <Loader2 className="h-4 w-4 animate-spin" /> : <Ban className="h-4 w-4" />} {t("consent.withdraw")}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
