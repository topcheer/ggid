"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { useConfirm } from "@/components/ConfirmDialog";
import { authHeader } from "@/lib/auth-helpers";
import {
  Megaphone, Plus, Users, KeyRound, Calendar, Mail, Check,
  Loader2, AlertCircle, Trash2, Eye, ChevronRight, ChevronLeft,
  Target, Send, Clock,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

interface Campaign {
  id: string;
  name: string;
  target_group: string;
  method: "passkey" | "webauthn" | "both";
  deadline: string;
  enrolled: number;
  target: number;
  status: "active" | "completed" | "draft" | "expired";
  created_at: string;
}

type TabId = "campaigns" | "create";
type CreateStep = 0 | 1 | 2 | 3;

const TARGET_GROUPS = [
  { value: "all", labelKey: "enrollmentCampaign.create.allUsers" },
  { value: "no_passkey", labelKey: "enrollmentCampaign.create.noPasskey" },
  { value: "admins", labelKey: "enrollmentCampaign.create.adminOnly" },
  { value: "custom", labelKey: "enrollmentCampaign.create.custom" },
];

const METHODS = [
  { value: "passkey", icon: KeyRound },
  { value: "webauthn", icon: KeyRound },
  { value: "both", icon: Users },
];

export default function EnrollmentCampaignPage() {
  const t = useTranslations();
  const [activeTab, setActiveTab] = useState<TabId>("campaigns");
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/enrollment/campaigns`, {
        headers: { ...authHeader() },
      });
      if (res.ok) {
        const data = await res.json();
        setCampaigns(Array.isArray(data) ? data : (data.campaigns || []));
      }
    } catch { /* empty state */ }
    setCampaigns([]);
    setLoading(false);
  }, []);

  useEffect(() => { load(); }, [load]);

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-800 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Megaphone className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white dark:text-white">{t("enrollmentCampaign.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 dark:text-gray-400 text-sm">{t("enrollmentCampaign.description")}</p>
        </div>

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {([
            { id: "campaigns" as TabId, label: t("enrollmentCampaign.tabs.campaigns"), icon: Megaphone },
            { id: "create" as TabId, label: t("enrollmentCampaign.tabs.create"), icon: Plus },
          ]).map((tab: any) => {
            const Icon = tab.icon;
            return (
              <button key={tab.id} onClick={() => setActiveTab(tab.id)}
                className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                  activeTab === tab.id ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm" : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
                }`}>
                <Icon className="w-4 h-4" />
                {tab.label}
              </button>
            );
          })}
        </div>

        {loading ? (
          <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>
        ) : activeTab === "campaigns" ? (
          <CampaignsList campaigns={campaigns} onRefresh={load} />
        ) : (
          <CreateCampaign onLaunched={() => { setActiveTab("campaigns"); load(); }} />
        )}
      </div>
    </div>
  );
}

// ============ Campaigns List ============

function CampaignsList({ campaigns, onRefresh }: { campaigns: Campaign[]; onRefresh: () => void }) {
  const t = useTranslations();

  const statusColors: Record<string, string> = {
    active: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
    completed: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
    draft: "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400",
    expired: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm("Delete this campaign?")) return;
    try {
      await fetch(`${API_BASE}/api/v1/auth/enrollment/campaigns/${id}`, { method: "DELETE", headers: { ...authHeader() } });
    } catch { /* ok */ }
    onRefresh();
  };

  if (campaigns.length === 0) {
    return (
      <div className="bg-white dark:bg-gray-800 dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 dark:border-gray-800 p-12 text-center">
        <Megaphone className="w-12 h-12 mx-auto mb-3 text-gray-300" />
        <p className="text-sm text-gray-500 dark:text-gray-400">{t("enrollmentCampaign.campaigns.noCampaigns")}</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {campaigns.map((c: any) => {
        const pct = c.target > 0 ? Math.round((c.enrolled / c.target) * 100) : 0;
        return (
          <div key={c.id} className="bg-white dark:bg-gray-800 dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 dark:border-gray-800 p-4">
            <div className="flex items-center justify-between mb-3">
              <div>
                <h3 className="text-sm font-semibold text-gray-900 dark:text-white dark:text-white">{c.name}</h3>
                <div className="flex items-center gap-3 mt-1 text-xs text-gray-500">
                  <span className="flex items-center gap-1"><Target className="w-3 h-3" />{t(`enrollmentCampaign.create.${c.target_group === "no_passkey" ? "noPasskey" : c.target_group === "all" ? "allUsers" : c.target_group === "admins" ? "adminOnly" : "custom"}`)}</span>
                  <span className="flex items-center gap-1"><KeyRound className="w-3 h-3" />{c.method}</span>
                  <span className="flex items-center gap-1"><Calendar className="w-3 h-3" />{new Date(c.deadline).toLocaleDateString()}</span>
                </div>
              </div>
              <span className={`px-2.5 py-0.5 text-xs rounded-full ${statusColors[c.status]}`}>
                {t(`enrollmentCampaign.campaigns.status${c.status.replace(/^./, (m: any) => m.toUpperCase())}`)}
              </span>
            </div>

            {/* Progress */}
            <div className="mb-2">
              <div className="flex items-center justify-between mb-1">
                <span className="text-xs text-gray-500">{t("enrollmentCampaign.campaigns.progress")}</span>
                <span className="text-xs font-medium text-gray-900 dark:text-white dark:text-white">
                  {c.enrolled}/{c.target} ({pct}%)
                </span>
              </div>
              <div className="h-2 bg-gray-200 dark:bg-gray-800 rounded-full overflow-hidden">
                <div className="h-full bg-green-500 rounded-full transition-all duration-500" style={{ width: `${pct}%` }} />
              </div>
            </div>

            {/* Actions */}
            <div className="flex items-center gap-2 mt-2">
              <button className="flex items-center gap-1 px-2 py-1 text-xs text-blue-600 hover:underline">
                <Eye className="w-3 h-3" />
                {t("enrollmentCampaign.campaigns.viewDetails")}
              </button>
              {c.status === "active" && (
                <button onClick={() => handleDelete(c.id)} className="flex items-center gap-1 px-2 py-1 text-xs text-red-600 hover:underline">
                  <Trash2 className="w-3 h-3" />
                  {t("enrollmentCampaign.campaigns.deleteCampaign")}
                </button>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}

// ============ Create Campaign Wizard ============

function CreateCampaign({ onLaunched }: { onLaunched: () => void }) {
  const t = useTranslations();
  const [step, setStep] = useState<CreateStep>(0);
  const [name, setName] = useState("");
  const [targetGroup, setTargetGroup] = useState("no_passkey");
  const [method, setMethod] = useState<"passkey" | "webauthn" | "both">("passkey");
  const [deadline, setDeadline] = useState("");
  const [sendEmail, setSendEmail] = useState(true);
  const [launching, setLaunching] = useState(false);
  const [error, setError] = useState("");

  const steps = [
    { title: t("enrollmentCampaign.create.step1"), icon: Target },
    { title: t("enrollmentCampaign.create.step2"), icon: KeyRound },
    { title: t("enrollmentCampaign.create.step3"), icon: Calendar },
    { title: t("enrollmentCampaign.create.step4"), icon: Check },
  ];

  const launch = async () => {
    if (!name || !deadline) { setError("Please fill all fields"); return; }
    setLaunching(true);
    try {
      await fetch(`${API_BASE}/api/v1/auth/enrollment/campaigns`, {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ name, target_group: targetGroup, method, deadline, send_email: sendEmail }),
      });
      onLaunched();
    } catch {
      onLaunched(); // still close
    } finally {
      setLaunching(false);
    }
  };

  const canNext = () => {
    if (step === 0) return !!targetGroup;
    if (step === 1) return !!method;
    if (step === 2) return !!deadline;
    return true;
  };

  return (
    <div className="bg-white dark:bg-gray-800 dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-700 dark:border-gray-800 p-6">
      {/* Stepper */}
      <div className="flex items-center gap-2 mb-8">
        {steps.map((s: any, i: number) => {
          const Icon = s.icon;
          const isActive = step === i;
          const isPast = step > i;
          return (
            <div key={i} className="flex items-center gap-2 flex-1">
              {i > 0 && <div className={`h-0.5 flex-1 ${isPast ? "bg-green-500" : "bg-gray-200 dark:bg-gray-700"}`} />}
              <div className={`flex items-center gap-2 ${i === steps.length - 1 ? "" : "flex-1"}`}>
                <div className={`w-8 h-8 rounded-full flex items-center justify-center text-xs font-medium ${
                  isActive ? "bg-blue-600 text-white" : isPast ? "bg-green-500 text-white" : "bg-gray-200 dark:bg-gray-700 text-gray-400"
                }`}>
                  {isPast ? <Check className="w-4 h-4" /> : <Icon className="w-4 h-4" />}
                </div>
                <span className={`text-xs hidden sm:inline ${isActive ? "text-blue-600 font-medium" : "text-gray-500"}`}>{s.title}</span>
              </div>
            </div>
          );
        })}
      </div>

      {/* Step Content */}
      {error && <div className="flex items-center gap-2 px-4 py-2 mb-4 rounded-lg bg-red-50 dark:bg-red-950/30 text-red-700 dark:text-red-300 text-sm"><AlertCircle className="w-4 h-4" />{error}</div>}

      {step === 0 && (
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-semibold text-gray-900 dark:text-white dark:text-white mb-2">{t("enrollmentCampaign.create.campaignName")}</label>
            <input type="text" value={name} onChange={(e) => setName(e.target.value)}
              placeholder={t("enrollmentCampaign.create.campaignNamePlaceholder")}
              className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 dark:bg-gray-800 text-sm text-gray-900 dark:text-white dark:text-white" />
          </div>
          <div>
            <label className="block text-sm font-semibold text-gray-900 dark:text-white dark:text-white mb-2">{t("enrollmentCampaign.create.targetGroup")}</label>
            <div className="grid grid-cols-2 gap-2">
              {TARGET_GROUPS.map((g: any) => (
                <button key={g.value} onClick={() => setTargetGroup(g.value)}
                  className={`p-3 rounded-lg border-2 text-left text-sm transition-all ${
                    targetGroup === g.value ? "border-blue-500 bg-blue-50 dark:bg-blue-950/30 text-blue-700 dark:text-blue-300" : "border-gray-200 dark:border-gray-700 hover:border-gray-300 text-gray-700 dark:text-gray-300"
                  }`}>
                  {t(g.labelKey)}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {step === 1 && (
        <div className="space-y-3">
          <label className="block text-sm font-semibold text-gray-900 dark:text-white dark:text-white">{t("enrollmentCampaign.create.method")}</label>
          {METHODS.map((m: any) => {
            const Icon = m.icon;
            const active = method === m.value;
            return (
              <button key={m.value} onClick={() => setMethod(m.value as typeof method)}
                className={`w-full flex items-center gap-3 p-4 rounded-lg border-2 text-left transition-all ${
                  active ? "border-blue-500 bg-blue-50 dark:bg-blue-950/30" : "border-gray-200 dark:border-gray-700 hover:border-gray-300"
                }`}>
                <Icon className={`w-5 h-5 ${active ? "text-blue-600" : "text-gray-400"}`} />
                <div>
                  <div className="text-sm font-medium text-gray-900 dark:text-white dark:text-white">
                    {t(`enrollmentCampaign.create.method${m.value.replace(/^./, (c: any) => c.toUpperCase())}`)}
                  </div>
                </div>
                {active && <Check className="w-5 h-5 text-blue-600 ml-auto" />}
              </button>
            );
          })}
        </div>
      )}

      {step === 2 && (
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-semibold text-gray-900 dark:text-white dark:text-white mb-1">{t("enrollmentCampaign.create.deadline")}</label>
            <p className="text-xs text-gray-500 dark:text-gray-400 mb-2">{t("enrollmentCampaign.create.deadlineDesc")}</p>
            <input type="date" value={deadline} onChange={(e) => setDeadline(e.target.value)}
              min={new Date().toISOString().split("T")[0]}
              className="w-full md:w-64 px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-600 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 dark:bg-gray-800 text-sm text-gray-900 dark:text-white dark:text-white" />
          </div>
          <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={sendEmail} onChange={(e) => setSendEmail(e.target.checked)} className="rounded" />
            <span className="text-sm text-gray-700 dark:text-gray-300 dark:text-gray-300 flex items-center gap-1">
              <Mail className="w-4 h-4" />
              {t("enrollmentCampaign.create.sendEmail")}
            </span>
          </label>
        </div>
      )}

      {step === 3 && (
        <div className="space-y-3">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-white dark:text-white mb-3">{t("enrollmentCampaign.create.reviewTitle")}</h3>
          <ReviewRow label={t("enrollmentCampaign.create.campaignName")} value={name} />
          <ReviewRow label={t("enrollmentCampaign.create.targetGroup")} value={t(`enrollmentCampaign.create.${targetGroup === "no_passkey" ? "noPasskey" : targetGroup === "all" ? "allUsers" : targetGroup === "admins" ? "adminOnly" : "custom"}`)} />
          <ReviewRow label={t("enrollmentCampaign.create.method")} value={t(`enrollmentCampaign.create.method${method.replace(/^./, (c: any) => c.toUpperCase())}`)} />
          <ReviewRow label={t("enrollmentCampaign.create.deadline")} value={deadline} />
          <ReviewRow label={t("enrollmentCampaign.create.sendEmail")} value={sendEmail ? "Yes" : "No"} />
        </div>
      )}

      {/* Navigation */}
      <div className="flex items-center justify-between mt-6 pt-4 border-t border-gray-200 dark:border-gray-700 dark:border-gray-800">
        {step > 0 ? (
          <button onClick={() => setStep(step - 1)} className="flex items-center gap-1.5 px-4 py-2 bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 dark:text-gray-300 rounded-lg text-sm font-medium">
            <ChevronLeft className="w-4 h-4" />
            {t("enrollmentCampaign.create.back")}
          </button>
        ) : <div />}

        {step < 3 ? (
          <button onClick={() => setStep(step + 1)} disabled={!canNext()}
            className="flex items-center gap-1.5 px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium">
            {t("enrollmentCampaign.create.next")}
            <ChevronRight className="w-4 h-4" />
          </button>
        ) : (
          <button onClick={launch} disabled={launching}
            className="flex items-center gap-2 px-6 py-2 bg-green-600 hover:bg-green-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium">
            {launching ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
            {t("enrollmentCampaign.create.confirmLaunch")}
          </button>
        )}
      </div>
    </div>
  );
}

function ReviewRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between py-2 border-b border-gray-100 dark:border-gray-800/50">
      <span className="text-xs text-gray-500 dark:text-gray-400">{label}</span>
      <span className="text-sm font-medium text-gray-900 dark:text-white dark:text-white">{value}</span>
    </div>
  );
}
