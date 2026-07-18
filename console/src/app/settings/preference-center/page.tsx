"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Settings, Loader2, AlertCircle, X, RefreshCw, Plus, Check,
  Shield, Clock, ChevronRight, Download, Cookie, User,
  CheckCircle2, XCircle, AlertTriangle, Ban, FileText, Zap,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface DSRRequest { id: string; type: string; user_id: string; status: string; due_date: string; created_at: string; }

type Tab = "dsr" | "preferences" | "cookies";

const DSR_TYPES = [
  { value: "access", label: "Data Access (GDPR Art.15)" },
  { value: "erasure", label: "Right to Erasure (GDPR Art.17)" },
  { value: "portability", label: "Data Portability (GDPR Art.20)" },
  { value: "rectification", label: "Rectification (GDPR Art.16)" },
];

const STATUS_CFG: Record<string, { label: string; color: string; bg: string }> = {
  pending: { label: "Pending", color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30" },
  processing: { label: "Processing", color: "text-blue-600", bg: "bg-blue-100 dark:bg-blue-900/30" },
  completed: { label: "Completed", color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30" },
  rejected: { label: "Rejected", color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30" },
};

export default function PreferenceCenterPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("dsr");
  const [dsrRequests, setDsrRequests] = useState<DSRRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // DSR form
  const [dType, setDType] = useState("access");
  const [dUser, setDUser] = useState("");
  const [submitting, setSubmitting] = useState(false);

  // Preferences
  const [prefs, setPrefs] = useState({ marketing_email: false, security_alerts: true, product_updates: true, analytics_opt_out: false, data_sharing: false });
  const [savingPrefs, setSavingPrefs] = useState(false);

  // Cookies
  const [cookies, setCookies] = useState({ essential: true, functional: true, analytics: false, marketing: false });

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/audit/dsr", { headers: h }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setDsrRequests(d.dsr_requests || []); }
    } catch { setError(t("prefCenter.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const submitDSR = async () => {
    if (!dUser) return;
    setSubmitting(true);
    try {
      await fetch("/api/v1/audit/dsr", { method: "POST", headers: H, body: JSON.stringify({ type: dType, user_id: dUser }) });
      setDUser(""); loadData();
    } catch { setError(t("prefCenter.dsrError")); }
    finally { setSubmitting(false); }
  };

  const pending = dsrRequests.filter(d => d.status === "pending" || d.status === "processing");
  const completed = dsrRequests.filter(d => d.status === "completed");
  const overdue = dsrRequests.filter(d => new Date(d.due_date).getTime() < Date.now() && d.status !== "completed");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Settings className="h-6 w-6 text-teal-500" /> {t("prefCenter.title")}
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("prefCenter.subtitle")}</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "dsr" as Tab, label: `${t("prefCenter.dsr")} (${pending.length})`, icon: FileText },
          { id: "preferences" as Tab, label: t("prefCenter.preferences"), icon: Shield },
          { id: "cookies" as Tab, label: t("prefCenter.cookieConsent"), icon: Cookie },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-teal-600 text-teal-600 dark:text-teal-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-teal-500" /></div> : (<>

      {/* ════ DSR ════ */}
      {tab === "dsr" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div className="lg:col-span-1 space-y-4">
            <div className={card}>
              <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("prefCenter.newDsr")}</h3>
              <div className="space-y-3">
                <div><label className="text-sm font-medium">{t("prefCenter.requestType")}</label>
                  <select value={dType} onChange={e => setDType(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                    {DSR_TYPES.map(d => <option key={d.value} value={d.value}>{d.label}</option>)}
                  </select>
                </div>
                <div><label className="text-sm font-medium">{t("prefCenter.userId")}</label><input type="text" value={dUser} onChange={e => setDUser(e.target.value)} placeholder="user:alice" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
                <div className="rounded-lg bg-blue-50 dark:bg-blue-900/20 p-2 text-xs text-blue-600 dark:text-blue-400"><Clock className="inline h-3 w-3" /> {t("prefCenter.slaNotice")}</div>
                <button onClick={submitDSR} disabled={!dUser || submitting} className="flex items-center gap-2 w-full justify-center rounded-lg bg-teal-600 px-4 py-2 text-sm font-medium text-white hover:bg-teal-700 disabled:opacity-50">
                  {submitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />} {t("prefCenter.submit")}
                </button>
              </div>
            </div>
            <div className="grid grid-cols-3 gap-2">
              <div className={`${card} text-center !p-3`}><p className="text-lg font-bold text-yellow-600">{pending.length}</p><p className="text-xs text-gray-400">{t("prefCenter.pending")}</p></div>
              <div className={`${card} text-center !p-3`}><p className="text-lg font-bold text-green-600">{completed.length}</p><p className="text-xs text-gray-400">{t("prefCenter.completed")}</p></div>
              <div className={`${card} text-center !p-3`}><p className="text-lg font-bold text-red-600">{overdue.length}</p><p className="text-xs text-gray-400">{t("prefCenter.overdue")}</p></div>
            </div>
          </div>
          <div className="lg:col-span-2">
            <div className={card}>
              <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("prefCenter.dsrHistory")} ({dsrRequests.length})</h3>
              {dsrRequests.length === 0 ? (
                <div className="py-8 text-center"><FileText className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("prefCenter.noDsr")}</p></div>
              ) : (
                <div className="space-y-2">
                  {dsrRequests.sort((a: any, b: any) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()).map(d => {
                    const cfg = STATUS_CFG[d.status] || STATUS_CFG.pending;
                    const daysLeft = Math.max(0, Math.ceil((new Date(d.due_date).getTime() - Date.now()) / 86400000));
                    const isOverdue = daysLeft === 0 && d.status !== "completed";
                    return (
                      <div key={d.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                        <div className="flex items-center gap-3">
                          <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${cfg.bg}`}><User className={`h-4 w-4 ${cfg.color}`} /></div>
                          <div>
                            <div className="flex items-center gap-2">
                              <span className="text-xs font-mono">{d.user_id}</span>
                              <span className="px-1.5 py-0.5 rounded bg-teal-100 dark:bg-teal-900/30 text-teal-600 text-xs font-mono">{d.type}</span>
                            </div>
                            <p className="text-xs text-gray-400">{new Date(d.created_at).toLocaleDateString()}</p>
                          </div>
                        </div>
                        <div className="text-right">
                          <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}>{cfg.label}</span>
                          <p className={`text-xs mt-0.5 ${isOverdue ? "text-red-500 font-bold" : "text-gray-400"}`}>{daysLeft}d {t("prefCenter.left")}</p>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* ════ PREFERENCES ════ */}
      {tab === "preferences" && (
        <div className={card}>
          <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Shield className="h-4 w-4" /> {t("prefCenter.communicationPrefs")}</h3>
          <div className="space-y-3">
            {([
              ["marketing_email", t("prefCenter.marketingEmail"), t("prefCenter.marketingEmailDesc")],
              ["security_alerts", t("prefCenter.securityAlerts"), t("prefCenter.securityAlertsDesc")],
              ["product_updates", t("prefCenter.productUpdates"), t("prefCenter.productUpdatesDesc")],
              ["analytics_opt_out", t("prefCenter.analyticsOptOut"), t("prefCenter.analyticsOptOutDesc")],
              ["data_sharing", t("prefCenter.dataSharing"), t("prefCenter.dataSharingDesc")],
            ] as const).map(([key, label, desc]) => (
              <label key={key} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <div><span className="text-sm font-medium">{label}</span><p className="text-xs text-gray-400">{desc}</p></div>
                <button onClick={() => setPrefs(prev => ({ ...prev, [key]: !prev[key as keyof typeof prev] }))} aria-pressed={prefs[key as keyof typeof prefs]} aria-label={label}
                  className={`relative h-6 w-11 rounded-full transition ${prefs[key as keyof typeof prefs] ? "bg-teal-500" : "bg-gray-300 dark:bg-gray-700"}`}>
                  <span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition ${prefs[key as keyof typeof prefs] ? "left-5" : "left-0.5"}`} />
                </button>
              </label>
            ))}
          </div>
          <button onClick={() => { setSavingPrefs(true); setTimeout(() => setSavingPrefs(false), 800); }} className="mt-4 flex items-center gap-2 rounded-lg bg-teal-600 px-4 py-2 text-sm font-medium text-white hover:bg-teal-700">
            {savingPrefs ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} {t("prefCenter.savePrefs")}
          </button>
        </div>
      )}

      {/* ════ COOKIES ════ */}
      {tab === "cookies" && (
        <div className={card}>
          <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Cookie className="h-4 w-4" /> {t("prefCenter.cookieSettings")}</h3>
          <div className="space-y-3">
            {([
              ["essential", t("prefCenter.essential"), t("prefCenter.essentialDesc"), true],
              ["functional", t("prefCenter.functional"), t("prefCenter.functionalDesc"), false],
              ["analytics", t("prefCenter.analyticsCookies"), t("prefCenter.analyticsDesc"), false],
              ["marketing", t("prefCenter.marketingCookies"), t("prefCenter.marketingDesc"), false],
            ] as const).map(([key, label, desc, locked]) => (
              <div key={key} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                <div><div className="flex items-center gap-2"><span className="text-sm font-medium">{label}</span>{locked && <span className="px-1.5 py-0.5 rounded bg-gray-100 dark:bg-gray-700 text-xs text-gray-400">{t("prefCenter.required")}</span>}</div><p className="text-xs text-gray-400">{desc}</p></div>
                <button onClick={() => !locked && setCookies(prev => ({ ...prev, [key]: !prev[key as keyof typeof prev] }))} disabled={locked} aria-pressed={cookies[key as keyof typeof cookies]} aria-label={label}
                  className={`relative h-6 w-11 rounded-full transition ${cookies[key as keyof typeof cookies] ? "bg-teal-500" : "bg-gray-300 dark:bg-gray-700"} ${locked ? "opacity-50 cursor-not-allowed" : ""}`}>
                  <span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition ${cookies[key as keyof typeof cookies] ? "left-5" : "left-0.5"}`} />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      </>)}
    </div>
  );
}
