"use client";
import { useState, useCallback, useEffect } from "react";
import {
  ClipboardCheck, Loader2, AlertCircle, X, Plus, Check,
  ChevronRight, Clock, CheckCircle2, XCircle, Activity,
  Users, Calendar, Play, AlertTriangle,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface Campaign { id: string; name: string; scope: string; certifier: string; deadline: string; progress: number; status: "active" | "completed" | "overdue"; total: number; reviewed: number; }
interface CompletedReview { id: string; name: string; certifier: string; completed_at: string; approved: number; revoked: number; total: number; }

type Tab = "active" | "completed" | "create";

const STATUS_CFG: Record<string, { color: string; bg: string }> = {
  active: { color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30" },
  completed: { color: "text-blue-600", bg: "bg-blue-100 dark:bg-blue-900/30" },
  overdue: { color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30" },
};

const SAMPLE_ACTIVE: Campaign[] = [
  { id: "c1", name: "Q1 2025 Admin Access Review", scope: "role:admin", certifier: "security-team", deadline: "2025-02-15", progress: 65, status: "active", total: 24, reviewed: 16 },
  { id: "c2", name: "Engineering Sensitive Data Access", scope: "dept:engineering", certifier: "eng-manager", deadline: "2025-02-28", progress: 30, status: "active", total: 87, reviewed: 26 },
  { id: "c3", name: "Finance System Access Quarterly", scope: "app:finance", certifier: "fin-controller", deadline: "2025-01-20", progress: 90, status: "active", total: 15, reviewed: 14 },
];

const SAMPLE_COMPLETED: CompletedReview[] = [
  { id: "r1", name: "Q4 2024 Admin Access Review", certifier: "security-team", completed_at: "2024-12-28", approved: 22, revoked: 2, total: 24 },
  { id: "r2", name: "All-Hands Access Audit", certifier: "compliance", completed_at: "2024-11-15", approved: 341, revoked: 18, total: 359 },
  { id: "r3", name: "OAuth Client Review", certifier: "platform-team", completed_at: "2024-10-30", approved: 12, revoked: 4, total: 16 },
];

export default function AccessReviewsPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("active");
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [completed, setCompleted] = useState<CompletedReview[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Create wizard
  const [cName, setCName] = useState("");
  const [cScope, setCScope] = useState("role:admin");
  const [cCertifier, setCCertifier] = useState("");
  const [cFrequency, setCFrequency] = useState("quarterly");
  const [creating, setCreating] = useState(false);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policies/access-reviews", { headers: h }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setCampaigns(d.campaigns || d.items || SAMPLE_ACTIVE); setCompleted(d.completed || SAMPLE_COMPLETED); }
      else { setCampaigns(SAMPLE_ACTIVE); setCompleted(SAMPLE_COMPLETED); }
    } catch { setError(t("accessReviews.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const createCampaign = async () => {
    if (!cName) return;
    setCreating(true);
    try { await fetch("/api/v1/policies/access-reviews", { method: "POST", headers: H, body: JSON.stringify({ name: cName, scope: cScope, certifier: cCertifier, frequency: cFrequency }) }); setCName(""); setCCertifier(""); loadData(); setTab("active"); }
    catch { setError(t("accessReviews.createError")); }
    finally { setCreating(false); }
  };

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ClipboardCheck className="h-6 w-6 text-indigo-500" /> {t("accessReviews.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("accessReviews.subtitle")}</p></div>

      {error && (<div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>)}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["active", `${t("accessReviews.activeCampaigns")} (${campaigns.length})`, Play], ["completed", t("accessReviews.completedReviews"), CheckCircle2], ["create", t("accessReviews.createCampaign"), Plus]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (<>

      {/* ACTIVE */}
      {tab === "active" && (
        <div className="space-y-4">{campaigns.map(c => {
          const cfg = STATUS_CFG[c.status] || STATUS_CFG.active;
          const daysLeft = Math.ceil((new Date(c.deadline).getTime() - Date.now()) / 86400000);
          return (
            <div key={c.id} className={card}>
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center gap-3"><div className="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-100 dark:bg-indigo-900/30"><ClipboardCheck className="h-5 w-5 text-indigo-500" /></div><div><h3 className="font-semibold text-sm">{c.name}</h3><p className="text-xs text-gray-400">{t("accessReviews.scope")}: <code className="font-mono">{c.scope}</code> · {t("accessReviews.certifier")}: {c.certifier}</p></div></div>
                <div className="text-right"><span className={`px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}>{c.status}</span><p className={`mt-1 text-xs ${daysLeft < 7 ? "text-red-600 font-bold" : "text-gray-400"}`}>{daysLeft > 0 ? `${daysLeft}d left` : `${Math.abs(daysLeft)}d overdue`}</p></div>
              </div>
              <div className="flex items-center gap-3"><div className="flex-1 h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className={`h-full rounded-full ${c.progress >= 80 ? "bg-green-500" : c.progress >= 40 ? "bg-blue-500" : "bg-yellow-500"}`} style={{ width: `${c.progress}%` }} /></div><span className="text-xs font-mono w-12 text-right">{c.reviewed}/{c.total}</span></div>
              <div className="mt-2 flex items-center justify-between text-xs text-gray-400"><span><Users className="inline h-3 w-3" /> {c.total} {t("accessReviews.toReview")}</span><span><Calendar className="inline h-3 w-3" /> {t("accessReviews.deadline")}: {new Date(c.deadline).toLocaleDateString()}</span></div>
            </div>
          );
        })}</div>
      )}

      {/* COMPLETED */}
      {tab === "completed" && (
        <div className="space-y-2">{completed.map(r => (
          <div key={r.id} className={`${card} flex items-center justify-between !p-3`}>
            <div className="flex items-center gap-3"><div className="flex h-9 w-9 items-center justify-center rounded-lg bg-green-100 dark:bg-green-900/30"><CheckCircle2 className="h-4 w-4 text-green-500" /></div><div><span className="text-sm font-medium">{r.name}</span><p className="text-xs text-gray-400">{r.certifier} · {new Date(r.completed_at).toLocaleDateString()}</p></div></div>
            <div className="flex items-center gap-3 text-center"><div><p className="text-sm font-bold text-green-600">{r.approved}</p><p className="text-xs text-gray-400">{t("accessReviews.approved")}</p></div><div><p className="text-sm font-bold text-red-600">{r.revoked}</p><p className="text-xs text-gray-400">{t("accessReviews.revoked")}</p></div></div>
          </div>
        ))}</div>
      )}

      {/* CREATE */}
      {tab === "create" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Plus className="h-4 w-4" /> {t("accessReviews.newCampaign")}</h3>
            <div className="space-y-3">
              <div><label className="text-sm font-medium">{t("accessReviews.campaignName")}</label><input type="text" value={cName} onChange={e => setCName(e.target.value)} placeholder="Q1 2025 Access Review" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">{t("accessReviews.scope")}</label><select value={cScope} onChange={e => setCScope(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="role:admin">Admin Role</option><option value="dept:engineering">Engineering Dept</option><option value="app:finance">Finance App</option><option value="all">All Users</option></select></div>
              <div><label className="text-sm font-medium">{t("accessReviews.certifier")}</label><input type="text" value={cCertifier} onChange={e => setCCertifier(e.target.value)} placeholder="security-team" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">{t("accessReviews.frequency")}</label><div className="mt-1 flex gap-2">{["monthly", "quarterly", "annual"].map(f => <button key={f} onClick={() => setCFrequency(f)} aria-pressed={cFrequency === f} className={`rounded-lg border px-3 py-1.5 text-sm ${cFrequency === f ? "border-indigo-500 bg-indigo-50 dark:bg-indigo-950/30 text-indigo-600" : "border-gray-300 dark:border-gray-700"}`}>{f}</button>)}</div></div>
              <button onClick={createCampaign} disabled={!cName || creating} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />} {t("accessReviews.create")}</button>
            </div>
          </div>
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("accessReviews.howItWorks")}</h3>
            <div className="space-y-2 text-xs text-gray-500 dark:text-gray-400">
              <p className="flex items-start gap-2"><Check className="h-3.5 w-3.5 text-green-500 mt-0.5" /> {t("accessReviews.step1")}</p>
              <p className="flex items-start gap-2"><Check className="h-3.5 w-3.5 text-green-500 mt-0.5" /> {t("accessReviews.step2")}</p>
              <p className="flex items-start gap-2"><Check className="h-3.5 w-3.5 text-green-500 mt-0.5" /> {t("accessReviews.step3")}</p>
              <p className="flex items-start gap-2"><Check className="h-3.5 w-3.5 text-green-500 mt-0.5" /> {t("accessReviews.step4")}</p>
            </div>
          </div>
        </div>
      )}

      </>)}
    </div>
  );
}
