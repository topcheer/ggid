"use client";

import { useTranslations } from "@/lib/i18n";
import { useState, useEffect, useCallback } from "react";
import { ClipboardCheck, Plus, X, Bell } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface Campaign {
  id: string;
  name: string;
  scope: string;
  reviewers: string[];
  deadline: string;
  completion_pct: number;
  auto_revoke: boolean;
  reminders: boolean;
  status: "active" | "completed" | "overdue";
}

const statusColors: Record<string, string> = {
  active: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  completed: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  overdue: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

const scopes = ["All Users", "Admins Only", "Contractors", "External Partners", "Engineering"];

export default function AccessReviewCampaignsPage() {
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ name: "", scope: "All Users", reviewers: [] as string[], deadline: "", auto_revoke: false });
  const [reviewerInput, setReviewerInput] = useState("");

  const t = useTranslations();

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/access-review-campaigns", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setCampaigns(d.campaigns || d || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const create = async () => {
    if (!form.name) return;
    try { await fetch("/api/v1/policy/access-review-campaigns", { method: "POST", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(form) }); setShowCreate(false); setForm({ name: "", scope: "All Users", reviewers: [], deadline: "", auto_revoke: false }); fetchData(); }
    catch { /* noop */ }
  };

  const toggleReminders = async (id: string) => {
    setCampaigns(campaigns.map((c) => c.id === id ? { ...c, reminders: !c.reminders } : c));
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><ClipboardCheck className="w-6 h-6 text-blue-500" /> {t("accessReviewCampaigns.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("accessReviewCampaigns.subtitle")}</p></div>
        <button onClick={() => setShowCreate(true)} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 flex items-center gap-2"><Plus className="w-4 h-4" /> {t("accessReviewCampaigns.newCampaign")}</button>
      </div>

      <div className="space-y-3">
        {campaigns.map((c) => (
          <div key={c.id} className="rounded-lg border dark:border-gray-800 p-4">
            <div className="flex items-center justify-between mb-2">
              <div><span className="font-semibold">{c.name}</span><p className="text-xs text-gray-400 mt-0.5">{t("accessReviewCampaigns.scope")} {c.scope} - {t("accessReviewCampaigns.deadline")} {c.deadline}</p></div>
              <span className={`px-2 py-0.5 rounded text-xs ${statusColors[c.status]}`}>{c.status}</span>
            </div>
            <div className="flex items-center gap-3 mb-3">
              <div className="flex-1"><div className="flex items-center justify-between text-xs mb-1"><span className="text-gray-500">{t("accessReviewCampaigns.completion")}</span><span className="font-bold">{c.completion_pct.toFixed(0)}%</span></div><div className="w-full bg-gray-100 dark:bg-gray-800 rounded-full h-2 overflow-hidden"><div className="h-full bg-blue-500 rounded-full" style={{ width: `${c.completion_pct}%` }} /></div></div>
              <button onClick={() => toggleReminders(c.id)} className={`flex items-center gap-1 text-xs px-2 py-1 rounded ${c.reminders ? "text-orange-600 bg-orange-50 dark:bg-orange-900/20" : "text-gray-400"}`}><Bell className="w-3.5 h-3.5" /> {c.reminders ? t("accessReviewCampaigns.on") : t("accessReviewCampaigns.off")}</button>
              {c.auto_revoke && <span className="px-2 py-0.5 rounded text-xs bg-red-100 dark:bg-red-900/30 dark:text-red-400">{t("accessReviewCampaigns.autoRevoke")}</span>}
            </div>
            <div className="flex flex-wrap gap-1">{c.reviewers.map((r) => <span key={r} className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{r}</span>)}</div>
          </div>
        ))}
        {campaigns.length === 0 && !loading && <p className="text-sm text-gray-500 text-center py-8">{t("accessReviewCampaigns.noCampaigns")}</p>}
      </div>

      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800"><h3 className="font-semibold">{t("accessReviewCampaigns.newCampaign")}</h3><button onClick={() => setShowCreate(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button></div>
            <div className="px-6 py-4 space-y-3">
              <div><label className="text-sm font-medium">{t("accessReviewCampaigns.name")}</label><input aria-label="Q4 Access Review" type="text" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="Q4 Access Review" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <div><label className="text-sm font-medium">{t("accessReviewCampaigns.scope")}</label><select aria-label="form" value={form.scope} onChange={(e) => setForm({ ...form, scope: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm">{scopes.map((s) => <option key={s} value={s}>{s}</option>)}</select></div>
              <div><label className="text-sm font-medium">{t("accessReviewCampaigns.reviewers")}</label><div className="flex items-center gap-2 mt-1"><input aria-label="reviewer@example.com" type="text" value={reviewerInput} onChange={(e) => setReviewerInput(e.target.value)} onKeyDown={(e) => { if (e.key === "Enter" && reviewerInput) { setForm({ ...form, reviewers: [...form.reviewers, reviewerInput] }); setReviewerInput(""); } }} placeholder="reviewer@example.com" className="flex-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /><button onClick={() => { if (reviewerInput) { setForm({ ...form, reviewers: [...form.reviewers, reviewerInput] }); setReviewerInput(""); } }} className="px-3 py-2 rounded-lg bg-gray-100 dark:bg-gray-800 text-sm"><Plus className="w-4 h-4" /></button></div><div className="flex flex-wrap gap-1 mt-2">{form.reviewers.map((r) => <span key={r} className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400 font-mono">{r}<button onClick={() => setForm({ ...form, reviewers: form.reviewers.filter((x) => x !== r) })}><X className="w-3 h-3" /></button></span>)}</div></div>
              <div><label className="text-sm font-medium">Deadline</label><input aria-label="form" type="date" value={form.deadline} onChange={(e) => setForm({ ...form, deadline: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" /></div>
              <label className="flex items-center gap-2 text-sm"><input aria-label="Form" type="checkbox" checked={form.auto_revoke} onChange={(e) => setForm({ ...form, auto_revoke: e.target.checked })} className="rounded" /> {t("accessReviewCampaigns.autoRevokeAfterDeadline")}</label>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800"><button onClick={() => setShowCreate(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">{t("accessReviewCampaigns.cancel")}</button><button onClick={create} disabled={!form.name} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50" aria-label="Action">{t("accessReviewCampaigns.create")}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
