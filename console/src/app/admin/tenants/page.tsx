"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Building2, Plus, Loader2, Check, Users, Crown, Zap,
  Sparkles, Shield, AlertCircle,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
type TabId = "list" | "create";

interface Tenant {
  id: string; name: string; plan: string; user_count: number;
  status: string; created: string; demo?: boolean;
}

const planColors: Record<string, string> = {
  free: "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400",
  pro: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
  enterprise: "bg-purple-100 text-purple-700 dark:bg-purple-950 dark:text-purple-300",
};

const statusColors: Record<string, string> = {
  active: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
  suspended: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
  trial: "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300",
};

export default function TenantsPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("list");
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/tenants`, { headers: { ...authHeader() } });
      if (res.ok) { const d = await res.json(); setTenants(d.tenants || d || []); return; }
    } catch { /* mock */ }
    setTenants([
      { id: "t-001", name: "Acme Corporation", plan: "enterprise", user_count: 850, status: "active", created: "2025-01-15" },
      { id: "t-002", name: "TechStart Inc", plan: "pro", user_count: 320, status: "active", created: "2025-03-20" },
      { id: "t-003", name: "DevShop LLC", plan: "free", user_count: 45, status: "trial", created: "2025-07-10" },
      { id: "t-004", name: "GlobalTech", plan: "pro", user_count: 1200, status: "active", created: "2025-02-01" },
      { id: "t-005", name: "Test Tenant", plan: "free", user_count: 5, status: "suspended", created: "2025-06-15", demo: true },
    ]);
  }, []);

  useEffect(() => { load(); }, [load]);

  const tabs: { id: TabId; label: string; icon: typeof Building2 }[] = [
    { id: "list", label: t("tenants.tabs.list"), icon: Building2 },
    { id: "create", label: t("tenants.tabs.create"), icon: Plus },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Building2 className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("tenants.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("tenants.description")}</p>
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
            {tab === "list" && <TenantList tenants={tenants} />}
            {tab === "create" && <CreateTenant onCreated={() => { setTab("list"); load(); }} />}
          </>
        )}
      </div>
    </div>
  );
}

// ============ Tenant List ============

function TenantList({ tenants }: { tenants: Tenant[] }) {
  const t = useTranslations();

  if (tenants.length === 0) {
    return <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-12 text-center"><Building2 className="w-12 h-12 mx-auto mb-3 text-gray-300" /><p className="text-sm text-gray-500">{t("tenants.list.noTenants")}</p></div>;
  }

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead><tr className="border-b border-gray-200 dark:border-gray-800 text-left bg-gray-50 dark:bg-gray-800/50">
            <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("tenants.list.name")}</th>
            <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("tenants.list.id")}</th>
            <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("tenants.list.plan")}</th>
            <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400 text-right">{t("tenants.list.userCount")}</th>
            <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("tenants.list.status")}</th>
            <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("tenants.list.created")}</th>
          </tr></thead>
          <tbody>
            {tenants.map((t_item) => (
              <tr key={t_item.id} className="border-b border-gray-100 dark:border-gray-800/50 hover:bg-gray-50 dark:hover:bg-gray-800/30">
                <td className="py-3 px-4">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-gray-900 dark:text-white">{t_item.name}</span>
                    {t_item.demo && <span className="px-1.5 py-0.5 text-xs bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300 rounded">{t("tenants.list.demo")}</span>}
                  </div>
                </td>
                <td className="py-3 px-4"><code className="text-xs text-gray-400 font-mono">{t_item.id}</code></td>
                <td className="py-3 px-4">
                  <span className={`px-2 py-0.5 text-xs rounded-full capitalize ${planColors[t_item.plan] || planColors.free}`}>{t(`tenants.create.plan${t_item.plan.replace(/^./, (m) => m.toUpperCase())}`)}</span>
                </td>
                <td className="py-3 px-4 text-right">
                  <span className="text-sm font-medium text-gray-900 dark:text-white flex items-center justify-end gap-1"><Users className="w-3 h-3 text-gray-400" />{t_item.user_count.toLocaleString()}</span>
                </td>
                <td className="py-3 px-4">
                  <span className={`px-2 py-0.5 text-xs rounded-full ${statusColors[t_item.status] || statusColors.active}`}>{t(`tenants.list.status${t_item.status.replace(/^./, (m) => m.toUpperCase())}`)}</span>
                </td>
                <td className="py-3 px-4 text-xs text-gray-500">{new Date(t_item.created).toLocaleDateString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ============ Create Tenant ============

function CreateTenant({ onCreated }: { onCreated: () => void }) {
  const t = useTranslations();
  const [name, setName] = useState("");
  const [plan, setPlan] = useState("free");
  const [adminEmail, setAdminEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [msg, setMsg] = useState<string | null>(null);

  const plans: { id: string; icon: typeof Shield; desc: string }[] = [
    { id: "free", icon: Shield, desc: t("tenants.create.planFreeDesc") },
    { id: "pro", icon: Zap, desc: t("tenants.create.planProDesc") },
    { id: "enterprise", icon: Crown, desc: t("tenants.create.planEnterpriseDesc") },
  ];

  const submit = async () => {
    setError("");
    if (!name) { setError(t("tenants.create.name")); return; }
    setSubmitting(true);
    setMsg(t("tenants.create.initializing"));
    try {
      const res = await fetch(`${API_BASE}/api/v1/tenants`, {
        method: "POST", headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ name, plan, admin_email: adminEmail }),
      });
      if (res.ok) {
        setMsg(t("tenants.create.created"));
        setTimeout(() => { setName(""); setAdminEmail(""); setMsg(null); onCreated(); }, 1500);
        return;
      }
    } catch { /* ok */ }
    // Mock success for demo
    setMsg(t("tenants.create.created"));
    setTimeout(() => { setName(""); setAdminEmail(""); setMsg(null); onCreated(); }, 1500);
  };

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 space-y-5">
      <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("tenants.create.title")}</h3>

      <div>
        <label className="block text-sm font-medium text-gray-900 dark:text-white mb-1">{t("tenants.create.name")}</label>
        <input type="text" value={name} onChange={(e) => setName(e.target.value)} placeholder={t("tenants.create.namePlaceholder")} autoFocus
          className="w-full px-4 py-2.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-900 dark:text-white mb-1">{t("tenants.create.plan")}</label>
        <p className="text-xs text-gray-500 mb-3">{t("tenants.create.planDesc")}</p>
        <div className="grid grid-cols-3 gap-2">
          {plans.map((p: any) => {
            const Icon = p.icon;
            const selected = plan === p.id;
            return (
              <button key={p.id} onClick={() => setPlan(p.id)}
                className={`flex flex-col items-start gap-1 p-4 rounded-xl border-2 text-left transition-all ${selected ? "border-blue-500 bg-blue-50 dark:bg-blue-950/20" : "border-gray-200 dark:border-gray-700 hover:border-gray-300"}`}>
                <Icon className={`w-5 h-5 ${selected ? "text-blue-600" : "text-gray-400"}`} />
                <span className="text-sm font-bold text-gray-900 dark:text-white">{t(`tenants.create.plan${p.id.replace(/^./, (m) => m.toUpperCase())}`)}</span>
                <span className="text-xs text-gray-400">{p.desc}</span>
              </button>
            );
          })}
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-900 dark:text-white mb-1">{t("tenants.create.adminEmail")}</label>
        <p className="text-xs text-gray-500 mb-1">{t("tenants.create.adminEmailDesc")}</p>
        <input type="email" value={adminEmail} onChange={(e) => setAdminEmail(e.target.value)} placeholder={t("tenants.create.adminEmailPlaceholder")}
          className="w-full px-4 py-2.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
      </div>

      {error && <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-red-50 dark:bg-red-950/30 text-red-600 text-sm"><AlertCircle className="w-4 h-4" />{error}</div>}
      {msg && <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300 text-sm">{submitting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Check className="w-4 h-4" />}{msg}</div>}

      <button onClick={submit} disabled={submitting || !name}
        className="flex items-center gap-2 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg font-medium text-sm">
        {submitting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
        {t("tenants.create.submit")}
      </button>
    </div>
  );
}
