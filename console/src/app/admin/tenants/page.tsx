"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Building2, Plus, Loader2, Check, Users, Crown, Zap,
  Sparkles, Shield, AlertCircle, Trash2,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";
type TabId = "list" | "create";

interface Tenant {
  id: string; name: string; plan: string; user_count: number;
  name: string; slug?: string; plan: string; user_count: number; status: string; created: string; demo?: boolean;
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

  const [deleteTarget, setDeleteTarget] = useState<Tenant | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const res = await fetch(`${API_BASE}/api/v1/tenants`, { headers: { ...authHeader() } });
      if (res.ok) {
        const d = await res.json();
        setTenants(d.tenants || d.items || (Array.isArray(d) ? d : []));
        return;
      }
      setError("Failed to load tenants");
    } catch {
      setError("Network error");
    }
    setTenants([]);
  }, []);

  const handleDelete = async (tenant: Tenant) => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/tenants/${tenant.id}`, {
        method: "DELETE", headers: { ...authHeader() },
      });
      if (res.ok) {
        setTenants(prev => prev.filter(t => t.id !== tenant.id));
        setDeleteTarget(null);
      } else {
        setError("Failed to delete tenant");
      }
    } catch {
      setError("Network error");
    }
  };

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
            {error && <div className="mb-4 flex items-center gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-950"><AlertCircle className="h-4 w-4" /> {error}</div>}
            {tab === "list" && <TenantList tenants={tenants} onDelete={setDeleteTarget} />}
            {tab === "create" && <CreateTenant onCreated={() => { setTab("list"); load(); }} />}
          </>
        )}
      </div>

      {/* Delete confirmation */}
      {deleteTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 max-w-md w-full mx-4">
            <div className="flex items-center gap-3 mb-3">
              <div className="rounded-full bg-red-100 dark:bg-red-950 p-2"><Trash2 className="h-5 w-5 text-red-600" /></div>
              <h3 className="text-lg font-semibold">{t("common.delete")}</h3>
            </div>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
              Delete tenant <strong>{deleteTarget.name}</strong>? This action cannot be undone.
            </p>
            <div className="flex justify-end gap-2">
              <button onClick={() => setDeleteTarget(null)} className="rounded-lg border border-gray-300 dark:border-gray-700 px-4 py-2 text-sm">{t("common.cancel")}</button>
              <button onClick={() => handleDelete(deleteTarget)} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">{t("common.delete")}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// ============ Tenant List ============

function TenantList({ tenants, onDelete }: { tenants: Tenant[]; onDelete: (t: Tenant) => void }) {
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
            <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400 text-right">{t("common.actions")}</th>
          </tr></thead>
          <tbody>
            {tenants.map((t_item) => (
              <tr key={t_item.id} className="border-b border-gray-100 dark:border-gray-800/50 hover:bg-gray-50 dark:hover:bg-gray-800/30">
                <td className="py-3 px-4">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-gray-900 dark:text-white">{t_item.name}</span>
                    {t_item.demo && <span className="px-1.5 py-0.5 text-xs bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300 rounded">{t("tenants.list.demo")}</span>}
                  </div>
                  {t_item.slug && (
                    <a href={`https://${t_item.slug}.ggid-console.iot2.win`} target="_blank" rel="noopener noreferrer"
                       className="text-xs text-blue-500 hover:underline font-mono">{t_item.slug}.ggid-console.iot2.win</a>
                  )}
                </td>
                <td className="py-3 px-4"><code className="text-xs text-gray-400 font-mono">{t_item.id}</code></td>
                <td className="py-3 px-4">
                  <span className={`px-2 py-0.5 text-xs rounded-full capitalize ${planColors[t_item.plan] || planColors.free}`}>{t(`tenants.create.plan${t_item.plan.replace(/^./, (m: any) => m.toUpperCase())}`)}</span>
                </td>
                <td className="py-3 px-4 text-right">
                  <span className="text-sm font-medium text-gray-900 dark:text-white flex items-center justify-end gap-1"><Users className="w-3 h-3 text-gray-400" />{(t_item.user_count ?? 0).toLocaleString()}</span>
                </td>
                <td className="py-3 px-4">
                  <span className={`px-2 py-0.5 text-xs rounded-full ${statusColors[t_item.status] || statusColors.active}`}>{t(`tenants.list.status${t_item.status.replace(/^./, (m: any) => m.toUpperCase())}`)}</span>
                </td>
                <td className="py-3 px-4 text-xs text-gray-500">{t_item.created ? new Date(t_item.created).toLocaleDateString() : "—"}</td>
                <td className="py-3 px-4 text-right">
                  <button onClick={() => onDelete(t_item)} className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600" title={t("common.delete")}>
                    <Trash2 className="h-4 w-4" />
                  </button>
                </td>
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
  const [slug, setSlug] = useState("");
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
        body: JSON.stringify({ name, slug: slug || undefined, plan, admin_email: adminEmail }),
      });
      if (res.ok) {
        setMsg(t("tenants.create.created"));
        setTimeout(() => { setName(""); setSlug(""); setAdminEmail(""); setMsg(null); onCreated(); }, 1500);
        return;
      }
    } catch {
      setMsg(null);
      setSubmitting(false);
      setError("Network error — failed to create tenant");
      return;
    }
  };

  // Auto-generate slug from name (lowercase, hyphenated)
  const handleNameChange = (v: string) => {
    setName(v);
    if (!slug || slug === slugFromName(name)) {
      setSlug(slugFromName(v));
    }
  };
  const slugFromName = (s: string) => s.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 space-y-5">
      <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("tenants.create.title")}</h3>

      <div>
        <label className="block text-sm font-medium text-gray-900 dark:text-white mb-1">{t("tenants.create.name")}</label>
        <input type="text" value={name} onChange={(e) => handleNameChange(e.target.value)} placeholder={t("tenants.create.namePlaceholder")} autoFocus
          className="w-full px-4 py-2.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-900 dark:text-white mb-1">子域名 (Subdomain)</label>
        <p className="text-xs text-gray-500 mb-2">从组织名自动生成，可自定义修改。仅限小写字母、数字和连字符。</p>
        <div className="flex items-center gap-2">
          <input type="text" value={slug} onChange={(e) => setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
            placeholder="acme"
            className="flex-1 px-4 py-2.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white font-mono" />
          <span className="text-sm text-gray-500 whitespace-nowrap">.ggid-console.iot2.win</span>
        </div>
        {slug && (
          <p className="mt-1.5 text-xs text-green-600 dark:text-green-400 flex items-center gap-1">
            <span>✓</span>
            <span>租户访问地址: <span className="font-mono font-medium">{slug}.ggid-console.iot2.win</span></span>
          </p>
        )}
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
                <span className="text-sm font-bold text-gray-900 dark:text-white">{t(`tenants.create.plan${p.id.replace(/^./, (m: any) => m.toUpperCase())}`)}</span>
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
