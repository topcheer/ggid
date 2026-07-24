"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  CalendarClock, Plus, History, Loader2, Check, Users,
  Shield, Clock, ChevronRight, AlertCircle,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";
type TabId = "schedules" | "create" | "history";

interface Schedule {
  id: string; name: string; scope: string; frequency: string;
  next_run: string; reviewer: string; enabled: boolean;
}

interface CampaignHistory {
  id: string; name: string; period: string; reviewer: string;
  reviewed: number; certified: number; revoked: number; status: string;
}

const FREQUENCIES = ["daily", "weekly", "monthly", "quarterly", "annual"];
const SCOPES = ["all_users", "by_department", "by_role", "service_accounts"];

const freqColors: Record<string, string> = {
  daily: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
  weekly: "bg-cyan-100 text-cyan-700 dark:bg-cyan-950 dark:text-cyan-300",
  monthly: "bg-purple-100 text-purple-700 dark:bg-purple-950 dark:text-purple-300",
  quarterly: "bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300",
  annual: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
};

export default function ReviewSchedulesPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("schedules");
  const [schedules, setSchedules] = useState<Schedule[]>([]);
  const [history, setHistory] = useState<CampaignHistory[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/identity/review-schedules`, { headers: { ...authHeader() } });
      if (res.ok) {
        const d = await res.json();
        const raw = d?.schedules ?? d?.items ?? d ?? [];
        setSchedules(Array.isArray(raw) ? raw : []);
      }
      else { setSchedules([]); }
    } catch { setSchedules([]); }
    setHistory([]);
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  const tabs: { id: TabId; label: string; icon: typeof CalendarClock }[] = [
    { id: "schedules", label: t("reviewSchedules.tabs.schedules"), icon: CalendarClock },
    { id: "create", label: t("reviewSchedules.tabs.create"), icon: Plus },
    { id: "history", label: t("reviewSchedules.tabs.history"), icon: History },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-800 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <CalendarClock className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white dark:text-white">{t("reviewSchedules.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 dark:text-gray-400 text-sm">{t("reviewSchedules.description")}</p>
        </div>

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {tabs.map(({ id, label, icon: Icon }) => (
            <button key={id} onClick={() => setTab(id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                tab === id ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm" : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
              }`}>
              <Icon className="w-4 h-4" />{label}
            </button>
          ))}
        </div>

        {loading ? (
          <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>
        ) : (
          <>
            {tab === "schedules" && <SchedulesTab schedules={schedules} setSchedules={setSchedules} />}
            {tab === "create" && <CreateTab onCreated={() => { setTab("schedules"); load(); }} />}
            {tab === "history" && <HistoryTab history={history} />}
          </>
        )}
      </div>
    </div>
  );
}

// ============ Schedules Tab ============

function SchedulesTab({ schedules, setSchedules }: { schedules: Schedule[]; setSchedules: (s: Schedule[]) => void }) {
  const t = useTranslations();

  const toggle = async (id: string) => {
    const s = schedules.find((x: any) => x.id === id);
    if (!s) return;
    try {
      await fetch(`${API_BASE}/api/v1/identity/review-schedules/${id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ enabled: !s.enabled }),
      });
      load();
    } catch {
      setSchedules(schedules.map((s: any) => s.id === id ? { ...s, enabled: !s.enabled } : s));
    }
  };

  if (schedules.length === 0) {
    return <div className="bg-white dark:bg-gray-800 dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 dark:border-gray-800 p-12 text-center"><CalendarClock className="w-12 h-12 mx-auto mb-3 text-gray-300" /><p className="text-sm text-gray-500">{t("reviewSchedules.schedules.noSchedules")}</p></div>;
  }

  return (
    <div className="space-y-2">
      {schedules.map((s: any) => (
        <div key={s.id} className="bg-white dark:bg-gray-800 dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 dark:border-gray-800 p-4">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-2">
              <Users className="w-5 h-5 text-gray-400" />
              <span className="text-sm font-medium text-gray-900 dark:text-white dark:text-white">{s.name}</span>
            </div>
            <div className="flex items-center gap-3">
              <span className={`px-2 py-0.5 text-xs rounded-full ${freqColors[s.frequency] || freqColors.monthly}`}>
                {t(`reviewSchedules.schedules.frequency${(s.frequency || "monthly").replace(/^./, (m: any) => m.toUpperCase())}`)}
              </span>
              <button onClick={() => toggle(s.id)} className={`relative w-10 h-6 rounded-full transition-colors ${s.enabled ? "bg-blue-600" : "bg-gray-300 dark:bg-gray-600"}`}>
                <span className={`absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform ${s.enabled ? "translate-x-4" : ""}`} />
              </button>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-4 text-xs text-gray-500">
            <span><Shield className="w-3 h-3 inline mr-1" />{t("reviewSchedules.schedules.scope")}: <span className="text-gray-700 dark:text-gray-300 dark:text-gray-300">{s.scope}</span></span>
            <span><Clock className="w-3 h-3 inline mr-1" />{t("reviewSchedules.schedules.nextRun")}: <span className="text-gray-700 dark:text-gray-300 dark:text-gray-300">{s.next_run ? new Date(s.next_run).toLocaleDateString() : "—"}</span></span>
            <span><Users className="w-3 h-3 inline mr-1" />{t("reviewSchedules.schedules.reviewer")}: <span className="text-gray-700 dark:text-gray-300 dark:text-gray-300">{s.reviewer}</span></span>
          </div>
        </div>
      ))}
    </div>
  );
}

// ============ Create Tab ============

function CreateTab({ onCreated }: { onCreated: () => void }) {
  const t = useTranslations();
  const [name, setName] = useState("");
  const [scope, setScope] = useState("");
  const [frequency, setFrequency] = useState("");
  const [reviewer, setReviewer] = useState("");
  const [startDate, setStartDate] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const submit = async () => {
    setError("");
    if (!scope) { setError(t("reviewSchedules.create.selectScope")); return; }
    if (!frequency) { setError(t("reviewSchedules.create.selectFrequency")); return; }
    setSubmitting(true);
    try {
      await fetch(`${API_BASE}/api/v1/identity/review-schedules`, {
        method: "POST", headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ name, scope, frequency, reviewer, start_date: startDate }),
      });
    } catch { /* ok */ }
    setSubmitting(false);
    onCreated();
  };

  return (
    <div className="bg-white dark:bg-gray-800 dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 dark:border-gray-800 p-6 space-y-5">
      <h3 className="text-sm font-semibold text-gray-900 dark:text-white dark:text-white">{t("reviewSchedules.create.title")}</h3>

      <div>
        <label className="block text-sm font-medium text-gray-900 dark:text-white dark:text-white mb-1">{t("reviewSchedules.create.name")}</label>
        <input type="text" value={name} onChange={(e) => setName(e.target.value)} placeholder={t("reviewSchedules.create.namePlaceholder")}
          className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 dark:bg-gray-800 text-sm text-gray-900 dark:text-white dark:text-white" />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-900 dark:text-white dark:text-white mb-1">{t("reviewSchedules.create.scope")}</label>
        <p className="text-xs text-gray-500 mb-2">{t("reviewSchedules.create.scopeDesc")}</p>
        <div className="grid grid-cols-2 gap-2">
          {SCOPES.map((s: any) => (
            <button key={s} onClick={() => setScope(s)}
              className={`flex items-center gap-2 p-3 rounded-lg border-2 text-sm transition-all ${scope === s ? "border-blue-500 bg-blue-50 dark:bg-blue-950/30 text-blue-700 dark:text-blue-300" : "border-gray-200 dark:border-gray-700 text-gray-600 dark:text-gray-400"}`}>
              {scope === s && <Check className="w-3 h-3" />}
              {t(`reviewSchedules.create.scope${s.replace(/_./g, (m: any) => m[1].toUpperCase()).replace(/^./, (m: any) => m.toUpperCase())}`)}
            </button>
          ))}
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-900 dark:text-white dark:text-white mb-1">{t("reviewSchedules.create.frequency")}</label>
        <p className="text-xs text-gray-500 mb-2">{t("reviewSchedules.create.frequencyDesc")}</p>
        <div className="flex flex-wrap gap-2">
          {FREQUENCIES.map((f: any) => (
            <button key={f} onClick={() => setFrequency(f)}
              className={`px-3 py-1.5 rounded-lg text-sm font-medium border-2 transition-all ${frequency === f ? "border-blue-500 " + (freqColors[f] || "") : "border-gray-200 dark:border-gray-700 text-gray-500"}`}>
              {t(`reviewSchedules.schedules.frequency${f.replace(/^./, (m: any) => m.toUpperCase())}`)}
            </button>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-gray-900 dark:text-white dark:text-white mb-1">{t("reviewSchedules.create.reviewer")}</label>
          <p className="text-xs text-gray-500 mb-1">{t("reviewSchedules.create.reviewerDesc")}</p>
          <input type="email" value={reviewer} onChange={(e) => setReviewer(e.target.value)} placeholder={t("reviewSchedules.create.reviewerPlaceholder")}
            className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 dark:bg-gray-800 text-sm text-gray-900 dark:text-white dark:text-white" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-900 dark:text-white dark:text-white mb-1">{t("reviewSchedules.create.startDate")}</label>
          <input type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)}
            min={new Date().toISOString().split("T")[0]}
            className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 dark:bg-gray-800 text-sm text-gray-900 dark:text-white dark:text-white" />
        </div>
      </div>

      {error && <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-red-50 dark:bg-red-950/30 text-red-700 dark:text-red-300 text-sm"><AlertCircle className="w-4 h-4" />{error}</div>}

      <button onClick={submit} disabled={submitting}
        className="flex items-center gap-2 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg font-medium text-sm">
        {submitting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
        {t("reviewSchedules.create.submit")}
      </button>
    </div>
  );
}

// ============ History Tab ============

function HistoryTab({ history }: { history: CampaignHistory[] }) {
  const t = useTranslations();

  const statusColors: Record<string, string> = {
    complete: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
    partial: "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300",
    overdue: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
  };

  if (history.length === 0) {
    return <div className="bg-white dark:bg-gray-800 dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 dark:border-gray-800 p-12 text-center"><History className="w-12 h-12 mx-auto mb-3 text-gray-300" /><p className="text-sm text-gray-500">{t("reviewSchedules.history.noHistory")}</p></div>;
  }

  return (
    <div className="bg-white dark:bg-gray-800 dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 dark:border-gray-800 overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-gray-200 dark:border-gray-700 dark:border-gray-800 text-left bg-gray-50 dark:bg-gray-800 dark:bg-gray-800/50">
            <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400">{t("reviewSchedules.history.campaign")}</th>
            <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400">{t("reviewSchedules.history.period")}</th>
            <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400">{t("reviewSchedules.history.reviewer")}</th>
            <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400 text-right">{t("reviewSchedules.history.reviewed")}</th>
            <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400 text-right">{t("reviewSchedules.history.certified")}</th>
            <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400 text-right">{t("reviewSchedules.history.revoked")}</th>
            <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400 dark:text-gray-400">{t("reviewSchedules.history.status")}</th>
          </tr></thead>
          <tbody>
            {history.map((h: any) => (
              <tr key={h.id} className="border-b border-gray-100 dark:border-gray-800/50">
                <td className="py-3 px-4 font-medium text-gray-900 dark:text-white dark:text-white">{h.name}</td>
                <td className="py-3 px-4 text-gray-600 dark:text-gray-400 dark:text-gray-400">{h.period}</td>
                <td className="py-3 px-4 text-gray-600 dark:text-gray-400 dark:text-gray-400">{h.reviewer}</td>
                <td className="py-3 px-4 text-right text-gray-900 dark:text-white dark:text-white">{h.reviewed}</td>
                <td className="py-3 px-4 text-right text-green-600 font-medium">{h.certified}</td>
                <td className="py-3 px-4 text-right text-red-600 font-medium">{h.revoked}</td>
                <td className="py-3 px-4">
                  <span className={`px-2 py-0.5 text-xs rounded-full ${statusColors[h.status] || statusColors.complete}`}>
                    {t(`reviewSchedules.history.status${h.status.replace(/^./, (m: any) => m.toUpperCase())}`)}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
