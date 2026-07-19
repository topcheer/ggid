"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Gauge, Loader2, AlertCircle, X, RefreshCw, Save, TrendingUp,
  Check, ChevronRight, Zap, Database, Users, Key, Activity,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface Quota { name: string; label: string; current: number; limit: number; unit: string; icon: typeof Users; }
interface PlanTier { name: string; price: string; limits: Record<string, number>; features: string[]; current: boolean; popular?: boolean; }

type Tab = "overview" | "trends" | "plans";

export default function QuotasPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [plan, setPlan] = useState("pro");
  const [saving, setSaving] = useState(false);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const quotas: Quota[] = [
    { name: "users", label: t("quotas.users"), current: 347, limit: 500, unit: "", icon: Users },
    { name: "api_keys", label: t("quotas.apiKeys"), current: 12, limit: 50, unit: "", icon: Key },
    { name: "sessions", label: t("quotas.sessions"), current: 89, limit: 1000, unit: "", icon: Activity },
    { name: "storage", label: t("quotas.storage"), current: 4.2, limit: 50, unit: "GB", icon: Database },
    { name: "api_calls", label: t("quotas.apiCalls"), current: 128420, limit: 500000, unit: "/mo", icon: Zap },
    { name: "webhooks", label: t("quotas.webhooks"), current: 8, limit: 25, unit: "", icon: Activity },
  ];

  const plans: PlanTier[] = [
    { name: "Free", price: "$0", limits: { users: 50, api_keys: 5, storage: 1, api_calls: 10000 }, features: ["Up to 50 users", "Basic MFA", "Community support"], current: false },
    { name: "Pro", price: "$299/mo", limits: { users: 500, api_keys: 50, storage: 50, api_calls: 500000 }, features: ["Up to 500 users", "Advanced MFA + WebAuthn", "Priority support", "DLP + SOAR", "Custom branding"], current: true, popular: true },
    { name: "Enterprise", price: "Custom", limits: { users: 999999, api_keys: 999, storage: 1000, api_calls: 9999999 }, features: ["Unlimited users", "Dedicated support", "On-premise option", "SOC2 + FedRAMP", "Custom SLAs"], current: false },
  ];

  // Trend data (daily)
  const trendDays = Array.from({ length: 14 }, (_, i) => i + 1);
  const apiTrend: number[] = [];
  const userTrend: number[] = [];
  const maxApi = Math.max(...apiTrend, 1);

  useEffect(() => { setLoading(false); }, []);

  const saveLimits = async () => {
    setSaving(true); try { await fetch(`/api/v1/quotas/${TENANT_ID}`, { method: "PUT", headers: H, body: JSON.stringify({ plan }) }); } catch { /* noop */ } finally { setSaving(false); }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Gauge className="h-6 w-6 text-blue-500" /> {t("quotas.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("quotas.subtitle")}</p></div>
        <button onClick={saveLimits} disabled={saving} className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} {t("quotas.save")}</button>
      </div>

      {error && (<div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>)}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "overview" as Tab, label: t("quotas.overview"), icon: Gauge },
          { id: "trends" as Tab, label: t("quotas.usageTrends"), icon: TrendingUp },
          { id: "plans" as Tab, label: t("quotas.planManagement"), icon: Check },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-blue-600 text-blue-600 dark:text-blue-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {tb.label}</button>
        );})}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div> : (<>

      {/* OVERVIEW */}
      {tab === "overview" && (
        <div className="space-y-6">
          <div className={`${card} flex items-center justify-between`}>
            <div><p className="text-xs text-gray-400">{t("quotas.currentPlan")}</p><p className="text-2xl font-bold text-blue-600">Pro</p></div>
            <button onClick={() => setTab("plans")} className="flex items-center gap-1 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700">{t("quotas.upgrade")} <ChevronRight className="h-4 w-4" /></button>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">{quotas.map(q => {
            const QIcon = q.icon; const pct = Math.min(Math.round((q.current / q.limit) * 100), 100); const isNear = pct >= 80;
            return (
              <div key={q.name} className={card}>
                <div className="flex items-center justify-between mb-2"><div className="flex items-center gap-2"><QIcon className={`h-4 w-4 ${isNear ? "text-orange-500" : "text-gray-400"}`} /><span className="text-sm font-medium">{q.label}</span></div><span className={`text-xs font-mono ${isNear ? "text-orange-600 font-bold" : "text-gray-400"}`}>{pct}%</span></div>
                <div className="h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className={`h-full rounded-full ${isNear ? "bg-orange-500" : pct >= 50 ? "bg-blue-500" : "bg-green-500"}`} style={{ width: `${pct}%` }} /></div>
                <p className="mt-1 text-xs text-gray-400"><span className="font-bold">{q.current.toLocaleString()}{q.unit}</span> / {q.limit.toLocaleString()}{q.unit} {q.unit === "/mo" ? "/mo" : ""}</p>
              </div>
            );
          })}</div>
        </div>
      )}

      {/* TRENDS */}
      {tab === "trends" && (
        <div className="space-y-6">
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> {t("quotas.apiCallsChart")}</h3>
            <svg width="100%" viewBox="0 0 420 100" className="overflow-visible"><polyline points={apiTrend.map((v: any, i: number) => `${i * 30},${90 - (v / maxApi) * 70}`).join(" ")} fill="none" stroke="#3b82f6" strokeWidth="2" strokeLinejoin="round" />{apiTrend.map((v: any, i: number) => <circle key={i} cx={i * 30} cy={90 - (v / maxApi) * 70} r="2" fill="#3b82f6" />)}</svg>
          </div>
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Users className="h-4 w-4" /> {t("quotas.userGrowth")}</h3>
            <div className="flex items-end gap-1 h-24">{userTrend.map((v: any, i: number) => { const maxU = Math.max(...userTrend); return <div key={i} className="flex-1 rounded-t bg-blue-300 dark:bg-blue-800" style={{ height: `${(v / maxU) * 100}%` }} />; })}</div>
          </div>
        </div>
      )}

      {/* PLANS */}
      {tab === "plans" && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">{plans.map(p => (
          <div key={p.name} className={`${card} ${p.current ? "border-blue-500 dark:border-blue-400" : ""} ${p.popular ? "ring-2 ring-blue-500" : ""}`}>
            <div className="flex items-center justify-between mb-2"><h3 className="text-lg font-bold">{p.name}</h3>{p.current && <span className="px-2 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900/30 text-blue-600 font-medium">{t("quotas.current")}</span>}</div>
            <p className="text-2xl font-bold mb-3">{p.price}</p>
            <ul className="space-y-1 mb-4">{p.features.map((f: any, i: number) => <li key={i} className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400"><Check className="h-3 w-3 text-green-500 shrink-0" /> {f}</li>)}</ul>
            <button onClick={() => setPlan(p.name.toLowerCase())} disabled={p.current} className={`w-full rounded-lg px-4 py-2 text-sm font-medium ${p.current ? "bg-gray-100 dark:bg-gray-700 text-gray-400" : "bg-blue-600 text-white hover:bg-blue-700"}`}>{p.current ? t("quotas.currentPlan") : t("quotas.switchTo")}</button>
          </div>
        ))}</div>
      )}

      </>)}
    </div>
  );
}
