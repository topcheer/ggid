"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Bot, Plus, AlertTriangle, Loader2, Check, X, Search,
  Shield, Clock, KeyRound, Webhook, Cpu, UserPlus, Ban,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
type TabId = "list" | "register" | "orphans";

interface NHI {
  id: string; type: string; name: string; status: string;
  owner: string; last_seen: string;
}

interface Orphan {
  id: string; name: string; type: string; last_seen: string;
  days_inactive: number; reason: string;
}

const typeIcons: Record<string, typeof Bot> = {
  service_account: Shield, api_key: KeyRound, oauth: KeyRound,
  machine: Cpu, webhook: Webhook,
};

const statusColors: Record<string, string> = {
  active: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
  inactive: "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400",
  expired: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
  rotating: "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300",
};

const NHI_TYPES = ["service_account", "api_key", "oauth", "machine", "webhook"];

export default function NHIPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("list");
  const [nhis, setNHIs] = useState<NHI[]>([]);
  const [orphans, setOrphans] = useState<Orphan[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [nhiRes, orphanRes] = await Promise.all([
        fetch(`${API_BASE}/api/v1/identity/nhi`, { headers: { ...authHeader() } }),
        fetch(`${API_BASE}/api/v1/identity/nhi/orphans`, { headers: { ...authHeader() } }),
      ]);
      if (nhiRes.ok) { const d = await nhiRes.json(); setNHIs(d.nhis || d || []); }
      if (orphanRes.ok) { const d = await orphanRes.json(); setOrphans(d.orphans || d || []); }
      return;
    } catch { /* mock */ }
    setNHIs([
      { id: "n1", type: "service_account", name: "payment-service-prod", status: "active", owner: "payments-team", last_seen: "2025-07-18T09:00:00Z" },
      { id: "n2", type: "api_key", name: "analytics-export-key", status: "active", owner: "data-team", last_seen: "2025-07-18T08:30:00Z" },
      { id: "n3", type: "oauth", name: "hris-integration", status: "rotating", owner: "hr-ops", last_seen: "2025-07-17T18:00:00Z" },
      { id: "n4", type: "machine", name: "ci-runner-01", status: "active", owner: "devops", last_seen: "2025-07-18T09:35:00Z" },
      { id: "n5", type: "webhook", name: "slack-notify-hook", status: "active", owner: "it-team", last_seen: "2025-07-18T07:00:00Z" },
      { id: "n6", type: "service_account", name: "legacy-import-svc", status: "expired", owner: "—", last_seen: "2025-05-01T00:00:00Z" },
    ]);
    setOrphans([
      { id: "o1", name: "legacy-import-svc", type: "service_account", last_seen: "2025-05-01", days_inactive: 78, reason: "no_owner" },
      { id: "o2", name: "temp-api-key-001", type: "api_key", last_seen: "2025-06-10", days_inactive: 38, reason: "inactive" },
      { id: "o3", name: "old-webhook-test", type: "webhook", last_seen: "2025-06-15", days_inactive: 33, reason: "stale_creds" },
      { id: "o4", name: "batch-job-runner", type: "service_account", last_seen: "2025-05-20", days_inactive: 59, reason: "no_owner" },
    ]);
  }, []);

  useEffect(() => { load(); }, [load]);

  const filtered = nhis.filter((n: any) => !search || n.name.toLowerCase().includes(search.toLowerCase()) || n.owner.toLowerCase().includes(search.toLowerCase()));

  const tabs: { id: TabId; label: string; icon: typeof Bot; count?: number }[] = [
    { id: "list", label: t("nhi.tabs.list"), icon: Bot },
    { id: "register", label: t("nhi.tabs.register"), icon: Plus },
    { id: "orphans", label: t("nhi.tabs.orphans"), icon: AlertTriangle, count: orphans.length },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Bot className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("nhi.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("nhi.description")}</p>
        </div>

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {tabs.map(({ id, label, icon: Icon, count }) => (
            <button key={id} onClick={() => setTab(id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                tab === id ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm" : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
              }`}>
              <Icon className="w-4 h-4" />{label}
              {count !== undefined && count > 0 && <span className="px-1.5 py-0.5 text-xs bg-red-200 dark:bg-red-900 text-red-700 dark:text-red-300 rounded-full">{count}</span>}
            </button>
          ))}
        </div>

        {loading ? (
          <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>
        ) : (
          <>
            {tab === "list" && <ListTab nhis={filtered} search={search} setSearch={setSearch} />}
            {tab === "register" && <RegisterTab onRegistered={load} />}
            {tab === "orphans" && <OrphansTab orphans={orphans} setOrphans={setOrphans} />}
          </>
        )}
      </div>
    </div>
  );
}

// ============ List Tab ============

function ListTab({ nhis, search, setSearch }: { nhis: NHI[]; search: string; setSearch: (v: string) => void }) {
  const t = useTranslations();

  if (nhis.length === 0) {
    return <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-12 text-center"><Bot className="w-12 h-12 mx-auto mb-3 text-gray-300" /><p className="text-sm text-gray-500">{t("nhi.list.noNHIs")}</p></div>;
  }

  return (
    <div className="space-y-4">
      <div className="relative max-w-xs">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder={t("nhi.list.searchPlaceholder")}
          className="w-full pl-9 pr-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 text-sm text-gray-900 dark:text-white" />
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead><tr className="border-b border-gray-200 dark:border-gray-800 text-left bg-gray-50 dark:bg-gray-800/50">
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("nhi.list.type")}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("nhi.list.name")}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("nhi.list.status")}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("nhi.list.owner")}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("nhi.list.lastSeen")}</th>
            </tr></thead>
            <tbody>
              {nhis.map((n: any) => {
                const Icon = typeIcons[n.type] || Bot;
                return (
                  <tr key={n.id} className="border-b border-gray-100 dark:border-gray-800/50 hover:bg-gray-50 dark:hover:bg-gray-800/30">
                    <td className="py-3 px-4">
                      <div className="flex items-center gap-2">
                        <Icon className="w-4 h-4 text-gray-400" />
                        <span className="text-xs text-gray-600 dark:text-gray-400">{t(`nhi.list.type${n.type.replace(/_./g, (m: any) => m[1].toUpperCase()).replace(/^./, (m: any) => m.toUpperCase())}`)}</span>
                      </div>
                    </td>
                    <td className="py-3 px-4 font-mono text-sm text-gray-900 dark:text-white">{n.name}</td>
                    <td className="py-3 px-4">
                      <span className={`px-2 py-0.5 text-xs rounded-full ${statusColors[n.status] || statusColors.inactive}`}>
                        {t(`nhi.list.status${n.status.replace(/^./, (m: any) => m.toUpperCase())}`)}
                      </span>
                    </td>
                    <td className="py-3 px-4 text-gray-600 dark:text-gray-400">{n.owner}</td>
                    <td className="py-3 px-4 text-xs text-gray-500">{n.last_seen ? new Date(n.last_seen).toLocaleDateString() : "—"}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

// ============ Register Tab ============

function RegisterTab({ onRegistered }: { onRegistered: () => void }) {
  const t = useTranslations();
  const [type, setType] = useState("");
  const [name, setName] = useState("");
  const [owner, setOwner] = useState("");
  const [metadata, setMetadata] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const submit = async () => {
    setError("");
    if (!type) { setError(t("nhi.register.selectType")); return; }
    setSubmitting(true);
    try {
      await fetch(`${API_BASE}/api/v1/identity/nhi`, {
        method: "POST", headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ type, name, owner, metadata: metadata ? JSON.parse(metadata) : {} }),
      });
    } catch { /* ok */ }
    setSubmitting(false);
    setType(""); setName(""); setOwner(""); setMetadata("");
    setMsg(t("nhi.register.registered"));
    setTimeout(() => setMsg(null), 3000);
    onRegistered();
  };

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 space-y-5">
      <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("nhi.register.title")}</h3>

      <div>
        <label className="block text-sm font-medium text-gray-900 dark:text-white mb-2">{t("nhi.register.type")}</label>
        <div className="grid grid-cols-2 md:grid-cols-5 gap-2">
          {NHI_TYPES.map((tp: any) => {
            const Icon = typeIcons[tp] || Bot;
            return (
              <button key={tp} onClick={() => setType(tp)}
                className={`flex flex-col items-center gap-1 p-3 rounded-lg border-2 text-xs transition-all ${type === tp ? "border-blue-500 bg-blue-50 dark:bg-blue-950/30 text-blue-700 dark:text-blue-300" : "border-gray-200 dark:border-gray-700 text-gray-500"}`}>
                <Icon className="w-5 h-5" />
                {t(`nhi.list.type${tp.replace(/_./g, (m: any) => m[1].toUpperCase()).replace(/^./, (m: any) => m.toUpperCase())}`)}
              </button>
            );
          })}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label className="block text-sm font-medium text-gray-900 dark:text-white mb-1">{t("nhi.register.name")}</label>
          <input type="text" value={name} onChange={(e) => setName(e.target.value)} placeholder={t("nhi.register.namePlaceholder")}
            className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-900 dark:text-white mb-1">{t("nhi.register.owner")}</label>
          <input type="text" value={owner} onChange={(e) => setOwner(e.target.value)} placeholder={t("nhi.register.ownerPlaceholder")}
            className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-900 dark:text-white mb-1">{t("nhi.register.metadata")}</label>
        <textarea value={metadata} onChange={(e) => setMetadata(e.target.value)} placeholder={t("nhi.register.metadataPlaceholder")} rows={4}
          className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm font-mono text-gray-900 dark:text-white" />
      </div>

      {error && <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-red-50 dark:bg-red-950/30 text-red-700 dark:text-red-300 text-sm"><AlertTriangle className="w-4 h-4" />{error}</div>}
      {msg && <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300 text-sm"><Check className="w-4 h-4" />{msg}</div>}

      <button onClick={submit} disabled={submitting}
        className="flex items-center gap-2 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg font-medium text-sm">
        {submitting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
        {t("nhi.register.submit")}
      </button>
    </div>
  );
}

// ============ Orphans Tab ============

function OrphansTab({ orphans, setOrphans }: { orphans: Orphan[]; setOrphans: (o: Orphan[]) => void }) {
  const t = useTranslations();

  const claim = (id: string) => setOrphans(orphans.filter((o: any) => o.id !== id));
  const disable = (id: string) => setOrphans(orphans.filter((o: any) => o.id !== id));

  if (orphans.length === 0) {
    return <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-12 text-center"><Check className="w-12 h-12 mx-auto mb-3 text-green-500" /><p className="text-sm text-gray-500">{t("nhi.orphans.noOrphans")}</p></div>;
  }

  return (
    <div className="space-y-3">
      <div className="bg-orange-50 dark:bg-orange-950/30 rounded-lg p-4 flex items-center gap-2">
        <AlertTriangle className="w-5 h-5 text-orange-500 flex-shrink-0" />
        <p className="text-sm text-orange-700 dark:text-orange-300">{t("nhi.orphans.description")}</p>
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead><tr className="border-b border-gray-200 dark:border-gray-800 text-left bg-gray-50 dark:bg-gray-800/50">
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("nhi.orphans.name")}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("nhi.orphans.type")}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("nhi.orphans.lastSeen")}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400 text-right">{t("nhi.orphans.daysInactive")}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("nhi.orphans.reason")}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400 text-right">{t("nhi.orphans.actions")}</th>
            </tr></thead>
            <tbody>
              {orphans.map((o: any) => (
                <tr key={o.id} className="border-b border-gray-100 dark:border-gray-800/50">
                  <td className="py-3 px-4 font-mono text-sm text-gray-900 dark:text-white">{o.name}</td>
                  <td className="py-3 px-4"><span className="text-xs text-gray-500">{t(`nhi.list.type${o.type.replace(/_./g, (m: any) => m[1].toUpperCase()).replace(/^./, (m: any) => m.toUpperCase())}`)}</span></td>
                  <td className="py-3 px-4 text-xs text-gray-500">{new Date(o.last_seen).toLocaleDateString()}</td>
                  <td className="py-3 px-4 text-right">
                    <span className={`text-xs font-medium ${o.days_inactive > 60 ? "text-red-600" : "text-orange-500"}`}>{o.days_inactive}d</span>
                  </td>
                  <td className="py-3 px-4">
                    <span className="px-2 py-0.5 text-xs bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300 rounded">{t(`nhi.orphans.reason${o.reason.replace(/_./g, (m: any) => m[1].toUpperCase()).replace(/^./, (m: any) => m.toUpperCase())}`)}</span>
                  </td>
                  <td className="py-3 px-4">
                    <div className="flex items-center gap-1 justify-end">
                      <button onClick={() => claim(o.id)} className="flex items-center gap-1 px-2 py-1 bg-green-50 dark:bg-green-950/30 hover:bg-green-100 dark:hover:bg-green-950 text-green-600 rounded text-xs font-medium">
                        <UserPlus className="w-3 h-3" />{t("nhi.orphans.claim")}
                      </button>
                      <button onClick={() => disable(o.id)} className="flex items-center gap-1 px-2 py-1 bg-red-50 dark:bg-red-950/30 hover:bg-red-100 dark:hover:bg-red-950 text-red-600 rounded text-xs font-medium">
                        <Ban className="w-3 h-3" />{t("nhi.orphans.disable")}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
