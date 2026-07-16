"use client";
import { useState, useEffect, useCallback } from "react";
import { Server, Plug, RefreshCw, CheckCircle, XCircle, Play, Loader2, History, Activity, ChevronDown } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { LDAP_VENDOR_PRESETS } from "@/lib/ldap-vendor-presets";

interface LdapConfig {
  server_url: string; bind_dn: string; base_dn: string;
  user_filter: string; group_filter: string; start_tls: boolean;
  attribute_mapping: { ldap_attr: string; local_attr: string }[];
  sync_interval_minutes: number; auto_provision: boolean;
}

interface TestResult {
  status: "ok" | "failed"; latency_ms: number; users_found: number; groups_found: number; error?: string;
}

interface SyncStatus {
  status: "idle" | "syncing" | "success" | "failed";
  last_sync: { timestamp: string; users_synced: number; groups_synced: number; errors: string[] } | null;
}

interface SyncHistoryEntry {
  started_at: string; completed_at: string; status: string; users_synced: number; groups_synced: number;
}

const DEFAULT_CONFIG: LdapConfig = {
  server_url: "", bind_dn: "", base_dn: "",
  user_filter: "(objectClass=person)", group_filter: "(objectClass=groupOfNames)",
  start_tls: true,
  attribute_mapping: [
    { ldap_attr: "uid", local_attr: "username" },
    { ldap_attr: "mail", local_attr: "email" },
    { ldap_attr: "cn", local_attr: "display_name" },
  ],
  sync_interval_minutes: 360, auto_provision: true,
};

export default function LdapSyncConfigPage() {
  const t = useTranslations();
  const [config, setConfig] = useState<LdapConfig>(DEFAULT_CONFIG);
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<TestResult | null>(null);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [syncStatus, setSyncStatus] = useState<SyncStatus>({ status: "idle", last_sync: null });
  const [syncing, setSyncing] = useState(false);
  const [history, setHistory] = useState<SyncHistoryEntry[]>([]);
  const [selectedVendor, setSelectedVendor] = useState("custom");

  const applyVendorPreset = (vendorId: string) => {
    setSelectedVendor(vendorId);
    const preset = LDAP_VENDOR_PRESETS.find(v => v.id === vendorId);
    if (!preset) return;
    const am = preset.config.attribute_mapping || {};
    const mapping = Object.entries(am).map(([ldap_attr, local_attr]) => ({ ldap_attr, local_attr }));
    setConfig(prev => ({
      ...prev,
      server_url: prev.server_url || preset.config.server_url,
      user_filter: preset.config.user_filter,
      group_filter: preset.config.group_filter,
      start_tls: preset.config.start_tls,
      attribute_mapping: mapping.length > 0 ? mapping : prev.attribute_mapping,
    }));
  };

  const loadData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const [configRes, statusRes, historyRes] = await Promise.all([
        fetch("/api/v1/identity/ldap/sync-config", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }).catch(() => null),
        fetch("/api/v1/identity/ldap/sync-status", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }).catch(() => null),
        fetch("/api/v1/identity/ldap/sync-history", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }).catch(() => null),
      ]);
      if (configRes?.ok) { const d = await configRes.json(); const c = d.config || d; if (c) setConfig(prev => ({ ...prev, ...c })); }
      if (statusRes?.ok) { const d = await statusRes.json(); setSyncStatus(d); }
      if (historyRes?.ok) { const d = await historyRes.json(); setHistory(d.history || []); }
    } catch { setError("Failed to load LDAP configuration"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const save = async () => {
    setSaving(true); setSaved(false);
    try {
      const res = await fetch("/api/v1/identity/ldap/sync-config", {
        method: "PUT", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ config }),
      });
      if (res.ok) setSaved(true);
    } catch { /* noop */ } finally { setSaving(false); }
  };

  const test = async () => {
    setTesting(true); setTestResult(null);
    try {
      const res = await fetch("/api/v1/identity/ldap/sync-config/test", {
        method: "POST", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ config }),
      });
      if (res.ok) { const d = await res.json(); setTestResult(d); }
      else { setTestResult({ status: "failed", latency_ms: 0, users_found: 0, groups_found: 0, error: `HTTP ${res.status}` }); }
    } catch (e) { setTestResult({ status: "failed", latency_ms: 0, users_found: 0, groups_found: 0, error: e instanceof Error ? e.message : "Connection failed" }); }
    finally { setTesting(false); }
  };

  const triggerSync = async () => {
    setSyncing(true);
    try {
      const res = await fetch("/api/v1/identity/ldap/sync", {
        method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
      });
      if (res.ok) { setSyncStatus({ status: "syncing", last_sync: null }); setTimeout(() => loadData(), 3000); }
    } catch { /* noop */ } finally { setSyncing(false); }
  };

  if (loading) return (<div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>);
  if (error) return (<div className="p-8"><div className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-950 dark:border-red-800 p-4"><p className="text-red-700 dark:text-red-400 text-sm font-medium">Error: {error}</p><button onClick={loadData} className="mt-2 px-4 py-1.5 rounded-lg bg-red-600 text-white text-sm hover:bg-red-700">Retry</button></div></div>);

  const statusColors: Record<string, string> = { idle: "bg-gray-100 text-gray-600", syncing: "bg-blue-100 text-blue-700 animate-pulse", success: "bg-green-100 text-green-700", failed: "bg-red-100 text-red-700" };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Server className="w-6 h-6 text-blue-500" /> {t("ldapSyncConfig.title")}</h1><p className="text-sm text-gray-500 mt-1">Configure LDAP server connection and sync settings.</p></div>
        <div className="flex gap-2">
          <button onClick={test} disabled={testing} aria-label="Test LDAP connection" className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm font-medium flex items-center gap-2"><Plug className={"w-4 h-4 " + (testing ? "animate-pulse" : "")} /> {testing ? "Testing..." : "Test"}</button>
          <button onClick={save} disabled={saving} aria-label="Save LDAP config" className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium disabled:opacity-50">{saving ? "Saving..." : "Save"}</button>
        </div>
      </div>

      {saved && <div className="rounded-lg border border-green-300 bg-green-50 dark:border-green-800 dark:bg-green-900/20 p-3 text-sm text-green-700 dark:text-green-400">Configuration saved.</div>}

      {testResult && (
        <div className={"rounded-lg border p-4 " + (testResult.status === "ok" ? "border-green-300 dark:border-green-800 bg-green-50 dark:bg-green-900/20" : "border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20")}>
          <div className="flex items-center gap-2 mb-2">{testResult.status === "ok" ? <CheckCircle className="w-5 h-5 text-green-500" /> : <XCircle className="w-5 h-5 text-red-500" />}<span className="text-sm font-medium">{testResult.status === "ok" ? "Connection successful" : "Connection failed"}</span></div>
          {testResult.status === "ok" && <div className="grid grid-cols-3 gap-4 text-sm"><div><span className="text-gray-500">Latency</span><p className="font-medium">{testResult.latency_ms}ms</p></div><div><span className="text-gray-500">Users Found</span><p className="font-medium">{testResult.users_found}</p></div><div><span className="text-gray-500">Groups Found</span><p className="font-medium">{testResult.groups_found}</p></div></div>}
          {testResult.error && <p className="text-sm text-red-600 mt-1">{testResult.error}</p>}
        </div>
      )}

      {/* Sync Status + Trigger */}
      <div className="rounded-lg border dark:border-gray-800 p-4">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-3">
            <Activity className="w-5 h-5 text-blue-500" />
            <div>
              <h3 className="text-sm font-semibold">Sync Status</h3>
              <span className={"inline-block mt-1 px-2 py-0.5 rounded text-xs font-medium " + (statusColors[syncStatus.status] || statusColors.idle)}>{syncStatus.status}</span>
            </div>
          </div>
          <button onClick={triggerSync} disabled={syncing || syncStatus.status === "syncing"} aria-label="Trigger LDAP sync" className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium flex items-center gap-2 disabled:opacity-50">
            {syncStatus.status === "syncing" ? <Loader2 className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />} Sync Now
          </button>
        </div>
        {syncStatus.last_sync && (
          <div className="grid grid-cols-3 gap-4 text-sm border-t dark:border-gray-700 pt-3">
            <div><span className="text-gray-500">Last Sync</span><p className="font-medium text-xs">{syncStatus.last_sync.timestamp}</p></div>
            <div><span className="text-gray-500">Users Synced</span><p className="font-medium">{syncStatus.last_sync.users_synced}</p></div>
            <div><span className="text-gray-500">Groups Synced</span><p className="font-medium">{syncStatus.last_sync.groups_synced}</p></div>
          </div>
        )}
        {syncStatus.last_sync?.errors && syncStatus.last_sync.errors.length > 0 && (
          <div className="mt-2 text-xs text-red-600">{syncStatus.last_sync.errors.length} errors during last sync</div>
        )}
      </div>

      {/* Vendor Preset Selector */}
      <div className="rounded-lg border dark:border-gray-800 p-4">
        <label className="text-sm font-medium">Directory Server Type</label>
        <select
          value={selectedVendor}
          onChange={(e) => applyVendorPreset(e.target.value)}
          className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"
        >
          {LDAP_VENDOR_PRESETS.map(v => (
            <option key={v.id} value={v.id}>{v.label}</option>
          ))}
        </select>
        {(() => { const p = LDAP_VENDOR_PRESETS.find(v => v.id === selectedVendor); return p ? <p className="mt-1.5 text-xs text-gray-500">{p.description}</p> : null; })()}
      </div>

      {/* Config form */}
      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3 max-w-lg">
        <div className="grid grid-cols-1 gap-3">
          <div><label className="text-sm font-medium">Server URL</label><input aria-label="ldap://dc01.example.com:389" type="text" value={config.server_url} onChange={(e) => setConfig({ ...config, server_url: e.target.value })} placeholder="ldap://dc01.example.com:389" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">Bind DN</label><input aria-label="cn=admin,dc=example,dc=com" type="text" value={config.bind_dn} onChange={(e) => setConfig({ ...config, bind_dn: e.target.value })} placeholder="cn=admin,dc=example,dc=com" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">Base DN</label><input aria-label="dc=example,dc=com" type="text" value={config.base_dn} onChange={(e) => setConfig({ ...config, base_dn: e.target.value })} placeholder="dc=example,dc=com" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">User Filter</label><input aria-label="config" type="text" value={config.user_filter} onChange={(e) => setConfig({ ...config, user_filter: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">Group Filter</label><input aria-label="config" type="text" value={config.group_filter} onChange={(e) => setConfig({ ...config, group_filter: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">Sync Interval (minutes)</label><input aria-label="config" type="number" value={config.sync_interval_minutes} onChange={(e) => setConfig({ ...config, sync_interval_minutes: parseInt(e.target.value) || 360 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
          <div className="flex items-center gap-4">
            <label className="flex items-center gap-2 text-sm"><input aria-label="Config" type="checkbox" checked={config.start_tls} onChange={(e) => setConfig({ ...config, start_tls: e.target.checked })} className="rounded" /> Start TLS</label>
            <label className="flex items-center gap-2 text-sm"><input aria-label="Config" type="checkbox" checked={config.auto_provision} onChange={(e) => setConfig({ ...config, auto_provision: e.target.checked })} className="rounded" /> Auto Provision</label>
          </div>
        </div>
      </div>

      {/* Attribute Mapping */}
      <div className="rounded-lg border dark:border-gray-800 p-4">
        <h3 className="text-sm font-semibold mb-2">Attribute Mapping</h3>
        <div className="overflow-x-auto"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-3 py-2 text-left font-medium">LDAP Attribute</th><th className="px-3 py-2 text-left font-medium">Local Attribute</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{config.attribute_mapping.map((m, i) => (<tr key={i}><td className="px-3 py-2"><input aria-label="Input field" type="text" value={m.ldap_attr} onChange={(e) => { const a = [...config.attribute_mapping]; a[i] = { ...m, ldap_attr: e.target.value }; setConfig({ ...config, attribute_mapping: a }); }} className="w-full px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs font-mono" /></td><td className="px-3 py-2"><input type="text" value={m.local_attr} onChange={(e) => { const a = [...config.attribute_mapping]; a[i] = { ...m, local_attr: e.target.value }; setConfig({ ...config, attribute_mapping: a }); }} className="w-full px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs font-mono" /></td></tr>))}</tbody></table></div>
      </div>

      {/* Sync History */}
      {history.length > 0 && (
        <div className="rounded-lg border dark:border-gray-800 p-4">
          <div className="flex items-center gap-2 mb-3"><History className="w-4 h-4 text-gray-500" /><h3 className="text-sm font-semibold">Sync History</h3></div>
          <div className="overflow-x-auto"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-3 py-2 text-left font-medium">Started</th><th className="px-3 py-2 text-left font-medium">Completed</th><th className="px-3 py-2 text-left font-medium">Status</th><th className="px-3 py-2 text-left font-medium">Users</th><th className="px-3 py-2 text-left font-medium">Groups</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{history.map((h, i) => (<tr key={i}><td className="px-3 py-2 text-xs text-gray-500">{h.started_at}</td><td className="px-3 py-2 text-xs text-gray-500">{h.completed_at}</td><td className="px-3 py-2"><span className={"px-2 py-0.5 rounded text-xs " + (h.status === "success" ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700")}>{h.status}</span></td><td className="px-3 py-2">{h.users_synced}</td><td className="px-3 py-2">{h.groups_synced}</td></tr>))}</tbody></table></div>
        </div>
      )}
    </div>
  );
}
