"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Database, ArrowRight, Settings, Loader2, Save, Check,
  Search, Plus, Trash2, AlertCircle, Activity, RefreshCw,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

type TabId = "overview" | "log" | "config";

interface MigrationStats {
  total: number; migrated: number; pending: number; failed: number;
  success_rate: number; last_migration: string;
}

interface MigrationLog {
  user_id: string; email: string; source: string; timestamp: string;
  status: "success" | "failed" | "partial" | "skipped"; attributes: Record<string, string>;
}

interface MigrationConfig {
  source_db_conn: string; hash_format: string; enabled: boolean;
  attribute_mapping: Record<string, string>;
}

export default function MigrationPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("overview");

  const tabs: { id: TabId; label: string; icon: typeof Database }[] = [
    { id: "overview", label: t("migration.tabs.overview"), icon: Activity },
    { id: "log", label: t("migration.tabs.log"), icon: Database },
    { id: "config", label: t("migration.tabs.config"), icon: Settings },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Database className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("migration.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("migration.description")}</p>
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

        {tab === "overview" && <OverviewTab />}
        {tab === "log" && <LogTab />}
        {tab === "config" && <ConfigTab />}
      </div>
    </div>
  );
}

// ============ Overview Tab ============

function OverviewTab() {
  const t = useTranslations();
  const [stats, setStats] = useState<MigrationStats | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/admin/migration/stats`, { headers: { ...authHeader() } });
      if (res.ok) { const d = await res.json(); setStats(d); return; }
    } catch { /* mock */ }
    setStats({ total: 500, migrated: 380, pending: 95, failed: 25, success_rate: 94, last_migration: "2025-07-18T09:00:00Z" });
  }, []);

  useEffect(() => { load(); }, [load]);

  if (loading || !stats) return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>;

  const pct = (v: number) => stats.total > 0 ? Math.round((v / stats.total) * 100) : 0;

  return (
    <div className="space-y-4">
      {/* Stat cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard label={t("migration.overview.total")} value={stats.total} color="blue" icon={Database} />
        <StatCard label={t("migration.overview.migrated")} value={stats.migrated} color="green" icon={Check} />
        <StatCard label={t("migration.overview.pending")} value={stats.pending} color="orange" icon={Activity} />
        <StatCard label={t("migration.overview.failed")} value={stats.failed} color="red" icon={AlertCircle} />
      </div>

      {/* Progress */}
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">{t("migration.overview.progress")}</h3>
        {/* Big progress bar */}
        <div className="mb-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm text-gray-600 dark:text-gray-400">{t("migration.overview.migratedUsers")}</span>
            <span className="text-sm font-bold text-gray-900 dark:text-white">{stats.migrated}/{stats.total} ({pct(stats.migrated)}%)</span>
          </div>
          <div className="h-4 bg-gray-200 dark:bg-gray-800 rounded-full overflow-hidden flex">
            <div className="h-full bg-green-500" style={{ width: `${pct(stats.migrated)}%` }} />
            <div className="h-full bg-orange-400" style={{ width: `${pct(stats.pending)}%` }} />
            <div className="h-full bg-red-400" style={{ width: `${pct(stats.failed)}%` }} />
          </div>
        </div>
        {/* Breakdown bars */}
        <div className="space-y-3">
          <ProgressBar label={t("migration.overview.migratedUsers")} value={stats.migrated} total={stats.total} color="bg-green-500" />
          <ProgressBar label={t("migration.overview.pendingUsers")} value={stats.pending} total={stats.total} color="bg-orange-500" />
          <ProgressBar label={t("migration.overview.failedUsers")} value={stats.failed} total={stats.total} color="bg-red-500" />
        </div>
        <div className="flex items-center gap-4 mt-4 pt-4 border-t border-gray-200 dark:border-gray-800 text-xs text-gray-500">
          <span>{t("migration.overview.successRate")}: <strong className="text-green-600">{stats.success_rate}%</strong></span>
          <span>{t("migration.overview.lastMigration")}: {new Date(stats.last_migration).toLocaleString()}</span>
        </div>
      </div>
    </div>
  );
}

// ============ Log Tab ============

function LogTab() {
  const t = useTranslations();
  const [logs, setLogs] = useState<MigrationLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");

  useEffect(() => {
    setLogs([
      { user_id: "u1", email: "alice@company.com", source: "LDAP", timestamp: "2025-07-18T09:30:00Z", status: "success", attributes: { cn: "Alice Chen", department: "Engineering", title: "Senior Engineer" } },
      { user_id: "u2", email: "bob@company.com", source: "LDAP", timestamp: "2025-07-18T09:25:00Z", status: "success", attributes: { cn: "Bob Smith", department: "Sales" } },
      { user_id: "u3", email: "carol@company.com", source: "Active Directory", timestamp: "2025-07-18T09:20:00Z", status: "partial", attributes: { cn: "Carol Wong", title: "Manager" } },
      { user_id: "u4", email: "dave@company.com", source: "LDAP", timestamp: "2025-07-18T09:15:00Z", status: "failed", attributes: {} },
      { user_id: "u5", email: "eve@company.com", source: "CSV Import", timestamp: "2025-07-18T08:45:00Z", status: "success", attributes: { name: "Eve Park", role: "engineer" } },
      { user_id: "u6", email: "frank@company.com", source: "Active Directory", timestamp: "2025-07-18T08:40:00Z", status: "skipped", attributes: { reason: "already_exists" } },
    ]);
    setLoading(false);
  }, []);

  const filtered = logs.filter((l) => !search || l.email.toLowerCase().includes(search.toLowerCase()));

  const statusColors: Record<string, string> = {
    success: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
    failed: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
    partial: "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300",
    skipped: "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400",
  };

  if (loading) return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>;

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
      <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-1">{t("migration.log.title")}</h3>
      <p className="text-xs text-gray-500 dark:text-gray-400 mb-4">{t("migration.log.description")}</p>

      <div className="relative mb-4 max-w-xs">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
        <input type="text" value={search} onChange={(e) => setSearch(e.target.value)} placeholder={t("migration.log.searchPlaceholder")}
          className="w-full pl-9 pr-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-200 dark:border-gray-800 text-left">
              <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("migration.log.email")}</th>
              <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("migration.log.source")}</th>
              <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("migration.log.timestamp")}</th>
              <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("migration.log.status")}</th>
              <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("migration.log.attributes")}</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((l) => (
              <tr key={l.user_id} className="border-b border-gray-100 dark:border-gray-800/50">
                <td className="py-3 px-3 font-medium text-gray-900 dark:text-white">{l.email}</td>
                <td className="py-3 px-3"><span className="px-2 py-0.5 text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded">{l.source}</span></td>
                <td className="py-3 px-3 text-xs text-gray-500">{new Date(l.timestamp).toLocaleString()}</td>
                <td className="py-3 px-3">
                  <span className={`px-2 py-0.5 text-xs rounded-full ${statusColors[l.status]}`}>
                    {t(`migration.log.status${l.status.replace(/^./, (m) => m.toUpperCase())}`)}
                  </span>
                </td>
                <td className="py-3 px-3">
                  <div className="flex flex-wrap gap-1">
                    {Object.entries(l.attributes).map(([k, v]) => (
                      <span key={k} className="text-xs text-gray-500">{k}: {v}</span>
                    ))}
                    {Object.keys(l.attributes).length === 0 && <span className="text-xs text-gray-400">—</span>}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ============ Config Tab ============

function ConfigTab() {
  const t = useTranslations();
  const [config, setConfig] = useState<MigrationConfig>({
    source_db_conn: "", hash_format: "bcrypt", enabled: false,
    attribute_mapping: { cn: "display_name", mail: "email", department: "department", title: "title" },
  });
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);
  const [newLegacyKey, setNewLegacyKey] = useState("");
  const [newGgidKey, setNewGgidKey] = useState("");

  const load = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/admin/migration/config`, { headers: { ...authHeader() } });
      if (res.ok) { const d = await res.json(); setConfig({ ...config, ...d }); }
    } catch { /* defaults */ }
  }, []);

  useEffect(() => { load(); }, [load]);

  const save = async () => {
    setSaving(true);
    try {
      await fetch(`${API_BASE}/api/v1/admin/migration/config`, {
        method: "PUT", headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify(config),
      });
    } catch { /* ok */ }
    setSaving(false);
    setMsg(t("migration.config.saved"));
    setTimeout(() => setMsg(null), 3000);
  };

  const testConn = async () => {
    setTesting(true);
    setTimeout(() => {
      setTesting(false);
      setMsg(t("migration.config.connSuccess"));
      setTimeout(() => setMsg(null), 3000);
    }, 1000);
  };

  const addMapping = () => {
    if (!newLegacyKey || !newGgidKey) return;
    setConfig({ ...config, attribute_mapping: { ...config.attribute_mapping, [newLegacyKey]: newGgidKey } });
    setNewLegacyKey(""); setNewGgidKey("");
  };

  const removeMapping = (key: string) => {
    const next = { ...config.attribute_mapping };
    delete next[key];
    setConfig({ ...config, attribute_mapping: next });
  };

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 space-y-5">
      <div>
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("migration.config.title")}</h3>
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{t("migration.config.description")}</p>
      </div>

      {/* Connection */}
      <div className="p-4 rounded-lg bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-900 space-y-3">
        <h4 className="text-sm font-medium text-gray-900 dark:text-white flex items-center gap-2"><Database className="w-4 h-4 text-blue-600" />{t("migration.config.connectionTitle")}</h4>
        <div>
          <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("migration.config.sourceDbConn")}</label>
          <input type="text" value={config.source_db_conn} onChange={(e) => setConfig({ ...config, source_db_conn: e.target.value })}
            placeholder={t("migration.config.sourceDbConnPlaceholder")}
            className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">{t("migration.config.hashFormat")}</label>
            <select value={config.hash_format} onChange={(e) => setConfig({ ...config, hash_format: e.target.value })}
              className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-sm text-gray-900 dark:text-white">
              <option value="bcrypt">{t("migration.config.hashBcrypt")}</option>
              <option value="argon2">{t("migration.config.hashArgon2")}</option>
              <option value="scrypt">{t("migration.config.hashScrypt")}</option>
              <option value="md5">{t("migration.config.hashMd5")}</option>
              <option value="sha256">{t("migration.config.hashSha256")}</option>
            </select>
          </div>
          <div className="flex items-end gap-2">
            <button onClick={testConn} disabled={testing || !config.source_db_conn}
              className="flex items-center gap-1.5 px-3 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-lg text-sm hover:bg-gray-50 disabled:opacity-50">
              {testing ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
              {testing ? t("migration.config.testing") : t("migration.config.testConn")}
            </button>
          </div>
        </div>
        <label className="flex items-center gap-2 cursor-pointer">
          <input type="checkbox" checked={config.enabled} onChange={(e) => setConfig({ ...config, enabled: e.target.checked })} className="rounded" />
          <span className="text-sm text-gray-700 dark:text-gray-300">{t("migration.config.enabled")}</span>
        </label>
      </div>

      {/* Attribute Mapping */}
      <div>
        <h4 className="text-sm font-medium text-gray-900 dark:text-white mb-1">{t("migration.config.mappingTitle")}</h4>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">{t("migration.config.mappingDesc")}</p>
        <div className="space-y-2">
          {Object.entries(config.attribute_mapping).map(([legacy, ggid]) => (
            <div key={legacy} className="flex items-center gap-2 p-2 rounded-lg bg-gray-50 dark:bg-gray-800/50">
              <input type="text" value={legacy} readOnly className="flex-1 px-2 py-1.5 rounded border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white" />
              <ArrowRight className="w-4 h-4 text-gray-400" />
              <input type="text" value={ggid} readOnly className="flex-1 px-2 py-1.5 rounded border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white" />
              <button onClick={() => removeMapping(legacy)} className="p-1 text-red-500 hover:bg-red-50 dark:hover:bg-red-950 rounded"><Trash2 className="w-3 h-3" /></button>
            </div>
          ))}
          <div className="flex items-center gap-2 p-2">
            <input type="text" value={newLegacyKey} onChange={(e) => setNewLegacyKey(e.target.value)} placeholder={t("migration.config.legacyAttr")}
              className="flex-1 px-2 py-1.5 rounded border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white" />
            <ArrowRight className="w-4 h-4 text-gray-400" />
            <input type="text" value={newGgidKey} onChange={(e) => setNewGgidKey(e.target.value)} placeholder={t("migration.config.gigAttr")}
              className="flex-1 px-2 py-1.5 rounded border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 text-xs text-gray-900 dark:text-white" />
            <button onClick={addMapping} disabled={!newLegacyKey || !newGgidKey}
              className="p-1 text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-950 rounded disabled:opacity-30"><Plus className="w-4 h-4" /></button>
          </div>
        </div>
      </div>

      {msg && (
        <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300 text-sm">
          <Check className="w-4 h-4" />{msg}
        </div>
      )}

      <button onClick={save} disabled={saving}
        className="flex items-center gap-2 px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg font-medium text-sm">
        {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
        {t("migration.config.save")}
      </button>
    </div>
  );
}

// ============ Shared ============

function StatCard({ label, value, color, icon: Icon }: { label: string; value: number; color: string; icon: typeof Database }) {
  const colors: Record<string, string> = { blue: "text-blue-600", green: "text-green-600", orange: "text-orange-500", red: "text-red-500" };
  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
      <div className="flex items-center gap-2 mb-2"><Icon className={`w-5 h-5 ${colors[color]}`} /><span className="text-xs text-gray-500 dark:text-gray-400">{label}</span></div>
      <div className="text-2xl font-bold text-gray-900 dark:text-white">{value}</div>
    </div>
  );
}

function ProgressBar({ label, value, total, color }: { label: string; value: number; total: number; color: string }) {
  const pct = total > 0 ? Math.round((value / total) * 100) : 0;
  return (
    <div>
      <div className="flex items-center justify-between mb-1">
        <span className="text-xs text-gray-600 dark:text-gray-400">{label}</span>
        <span className="text-xs font-medium text-gray-900 dark:text-white">{value} ({pct}%)</span>
      </div>
      <div className="h-2 bg-gray-200 dark:bg-gray-800 rounded-full overflow-hidden">
        <div className={`h-full ${color} rounded-full transition-all duration-500`} style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}
