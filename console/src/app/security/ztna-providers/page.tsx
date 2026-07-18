"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Cloud, Shield, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2,
  Check, CheckCircle, XCircle, Activity, ArrowRight, Download, Copy,
  Zap, Eye, Globe, Terminal, Settings, Code, ChevronRight,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface Provider {
  id: string;
  type: "zscaler" | "cloudflare" | "twingate" | "tailscale";
  name: string;
  connected: boolean;
  config: Record<string, string>;
  last_sync: string | null;
  synced_users: number;
  synced_groups: number;
}

interface DeviceStatus {
  id: string;
  device_name: string;
  user: string;
  platform: string;
  trust_level: "trusted" | "compliant" | "unmanaged" | "blocked";
  compliance: { disk_encrypted: boolean; firewall: boolean; av_installed: boolean; os_updated: boolean };
  last_seen: string;
  issues: string[];
}

interface CAEPEvent {
  id: string;
  event_type: "session-revoked" | "credential-change" | "device-compliance-change";
  subject: string;
  actor: string;
  timestamp: string;
  details: string;
  provider: string;
}

const providerConfig = {
  zscaler: { name: "Zscaler ZPA", icon: Shield, color: "text-blue-500", fields: [{ key: "company_id", label: "Company ID" }, { key: "api_key", label: "API Key" }, { key: "zpa_cloud", label: "ZPA Cloud" }] },
  cloudflare: { name: "Cloudflare Access", icon: Cloud, color: "text-orange-500", fields: [{ key: "team_domain", label: "Team Domain" }, { key: "api_token", label: "API Token" }, { key: "account_id", label: "Account ID" }] },
  twingate: { name: "Twingate", icon: Globe, color: "text-indigo-500", fields: [{ key: "network", label: "Network Slug" }, { key: "api_key", label: "API Key" }] },
  tailscale: { name: "Tailscale", icon: Terminal, color: "text-purple-500", fields: [{ key: "tailnet", label: "Tailnet Name" }, { key: "api_key", label: "Auth Key" }] },
};

const trustColors: Record<string, string> = {
  trusted: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400",
  compliant: "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400",
  unmanaged: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400",
  blocked: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400",
};

const caepColors: Record<string, string> = {
  "session-revoked": "bg-red-50 text-red-600 dark:bg-red-950/20",
  "credential-change": "bg-yellow-50 text-yellow-600 dark:bg-yellow-950/20",
  "device-compliance-change": "bg-blue-50 text-blue-600 dark:bg-blue-950/20",
};

type Tab = "providers" | "devices" | "caep" | "scim" | "terraform";

export default function ZTNAProvidersPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("providers");
  const [providers, setProviders] = useState<Provider[]>([]);
  const [devices, setDevices] = useState<DeviceStatus[]>([]);
  const [events, setEvents] = useState<CAEPEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Register provider
  const [showRegister, setShowRegister] = useState(false);
  const [newProviderType, setNewProviderType] = useState<Provider["type"]>("zscaler");
  const [newProviderName, setNewProviderName] = useState("");
  const [newConfig, setNewConfig] = useState<Record<string, string>>({});
  const [registering, setRegistering] = useState(false);
  // SCIM push
  const [scimProvider, setScimProvider] = useState("");
  const [scimPushUsers, setScimPushUsers] = useState(true);
  const [scimPushGroups, setScimPushGroups] = useState(true);
  const [pushing, setPushing] = useState(false);
  // Terraform
  const [tfProvider, setTfProvider] = useState("");
  const [copied, setCopied] = useState(false);
  // Actions
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [provRes, devRes, evRes] = await Promise.all([
        fetch("/api/v1/ztna/providers", { headers: h }).catch(() => null),
        fetch("/api/v1/ztna/devices?page_size=100", { headers: h }).catch(() => null),
        fetch("/api/v1/ztna/caep-events?page_size=50", { headers: h }).catch(() => null),
      ]);
      if (provRes?.ok) { const d = await provRes.json(); setProviders(d.providers || d.items || []); }
      if (devRes?.ok) { const d = await devRes.json(); setDevices(d.devices || d.items || []); }
      if (evRes?.ok) { const d = await evRes.json(); setEvents(d.events || d.items || []); }
    } catch { setError("Failed to load ZTNA data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const registerProvider = async () => {
    if (!newProviderName) return;
    setRegistering(true);
    try {
      await fetch("/api/v1/ztna/providers", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ type: newProviderType, name: newProviderName, config: newConfig }),
      });
      setShowRegister(false); setNewProviderName(""); setNewConfig({});
      loadData();
    } catch { setError("Failed to register provider"); }
    finally { setRegistering(false); }
  };

  const deleteProvider = async (id: string) => {
    setDeletingId(id);
    try {
      await fetch(`/api/v1/ztna/providers/${id}`, { method: "DELETE", headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID } });
      setProviders(prev => prev.filter(p => p.id !== id));
    } catch { setError("Failed to delete provider"); }
    finally { setDeletingId(null); }
  };

  const pushSCIM = async () => {
    if (!scimProvider) return;
    setPushing(true);
    try {
      await fetch(`/api/v1/ztna/providers/${scimProvider}/scim-push`, {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ push_users: scimPushUsers, push_groups: scimPushGroups }),
      });
    } catch { /* noop */ }
    finally { setPushing(false); }
  };

  const generateTF = (): string => {
    if (!tfProvider) return "# Select a provider to generate Terraform";
    const prov = providers.find(p => p.id === tfProvider);
    if (!prov) return "# Provider not found";
    const cfg = providerConfig[prov.type];
    const entries = Object.entries(prov.config || {}).map(([k, v]) => `  ${k} = "${v.includes("***") ? "REDACTED" : v}"`).join("\n");
    return `# ${cfg.name} — generated by GGID
terraform {
  required_providers {
    ${prov.type} = {
      source  = "${prov.type}/${prov.type}"
      version = "~> 1.0"
    }
  }
}

provider "${prov.type}" {
${entries}
}

# Application connector
resource "${prov.type}_application" "ggid_console" {
  name             = "GGID Console"
  domain           = "console.ggid.dev"
  health_check_url = "/healthz"
}
`;
  };

  const copyTF = async () => {
    await navigator.clipboard.writeText(generateTF());
    setCopied(true); setTimeout(() => setCopied(false), 3000);
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Cloud className="h-6 w-6 text-indigo-500" /> ZTNA Provider Integration</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Manage Zscaler/Cloudflare/Twingate/Tailscale integrations, device posture, CAEP events, and SCIM sync.</p>
        </div>
        <button onClick={loadData} disabled={loading} aria-label="Refresh" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"><RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "providers" as Tab, label: "Providers", icon: Settings },
          { id: "devices" as Tab, label: "Device Posture", icon: Shield },
          { id: "caep" as Tab, label: "CAEP Events", icon: Zap },
          { id: "scim" as Tab, label: "SCIM Push", icon: ArrowRight },
          { id: "terraform" as Tab, label: "Terraform Export", icon: Code },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id} className={"flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap " + (tab === tb.id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300")}><Icon className="h-4 w-4" /> {tb.label}</button>
        ); })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (<>

      {/* PROVIDERS */}
      {tab === "providers" && (<>
        <div className="flex justify-end"><button onClick={() => { setNewProviderType("zscaler"); setShowRegister(true); }} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Plus className="h-4 w-4" /> Add Provider</button></div>
        {providers.length === 0 ? <div className={cardCls}><div className="py-12 text-center"><Cloud className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No ZTNA providers connected.</p></div></div> : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">{providers.map(p => { const cfg = providerConfig[p.type]; const PIcon = cfg.icon; return (
            <div key={p.id} className={cardCls + " hover:shadow-md transition"}>
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3"><div className={"h-10 w-10 rounded-lg flex items-center justify-center bg-gray-100 dark:bg-gray-700"}><PIcon className={"h-5 w-5 " + cfg.color} /></div><div><h3 className="font-semibold text-gray-900 dark:text-white">{p.name}</h3><p className="text-xs text-gray-400">{cfg.name}</p></div></div>
                <div className="flex items-center gap-1"><span className={"h-2.5 w-2.5 rounded-full " + (p.connected ? "bg-green-500 animate-pulse" : "bg-gray-400")} /><span className="text-xs text-gray-400">{p.connected ? "Connected" : "Offline"}</span><button onClick={() => deleteProvider(p.id)} disabled={deletingId === p.id} aria-label="Delete" className="ml-2 rounded p-1 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20">{deletingId === p.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}</button></div>
              </div>
              <div className="mt-3 grid grid-cols-2 gap-2 text-center"><div><p className="text-xs text-gray-400">Synced Users</p><p className="text-lg font-bold text-indigo-600">{p.synced_users}</p></div><div><p className="text-xs text-gray-400">Synced Groups</p><p className="text-lg font-bold text-purple-600">{p.synced_groups}</p></div></div>
              {p.last_sync && <p className="mt-2 text-xs text-gray-400">Last sync: {new Date(p.last_sync).toLocaleString()}</p>}
            </div>
          ); })}</div>
        )}
      </>)}

      {/* DEVICE POSTURE */}
      {tab === "devices" && (
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
            <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Total</span><p className="mt-2 text-2xl font-bold">{devices.length}</p></div>
            <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Trusted</span><p className="mt-2 text-2xl font-bold text-green-600">{devices.filter(d => d.trust_level === "trusted").length}</p></div>
            <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Compliant</span><p className="mt-2 text-2xl font-bold text-blue-600">{devices.filter(d => d.trust_level === "compliant").length}</p></div>
            <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Issues</span><p className="mt-2 text-2xl font-bold text-red-600">{devices.filter(d => d.trust_level === "blocked" || d.trust_level === "unmanaged").length}</p></div>
          </div>
          <div className={cardCls}>
            {devices.length === 0 ? <div className="py-8 text-center"><Shield className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No devices registered.</p></div> : (
              <div className="overflow-x-auto"><table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th scope="col" className="px-3 py-2 text-left font-medium text-xs">Device</th><th scope="col" className="px-3 py-2 text-left font-medium text-xs">User</th><th scope="col" className="px-3 py-2 text-left font-medium text-xs">Platform</th><th scope="col" className="px-3 py-2 text-center font-medium text-xs">Trust</th><th scope="col" className="px-3 py-2 text-center font-medium text-xs">Checks</th><th scope="col" className="px-3 py-2 text-left font-medium text-xs">Issues</th></tr></thead>
                <tbody className="divide-y dark:divide-gray-800">{devices.map(d => (
                  <tr key={d.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                    <td className="px-3 py-2 text-xs font-medium">{d.device_name}</td>
                    <td className="px-3 py-2 text-xs">{d.user}</td>
                    <td className="px-3 py-2 text-xs text-gray-500">{d.platform}</td>
                    <td className="px-3 py-2 text-center"><span className={"px-2 py-0.5 rounded text-xs font-medium " + (trustColors[d.trust_level] || "")}>{d.trust_level}</span></td>
                    <td className="px-3 py-2"><div className="flex items-center justify-center gap-1">{[["E", d.compliance.disk_encrypted], ["F", d.compliance.firewall], ["A", d.compliance.av_installed], ["U", d.compliance.os_updated]].map(([label, ok]) => <span key={label as string} className={"h-5 w-5 rounded text-xs font-bold flex items-center justify-center " + (ok ? "bg-green-100 text-green-600 dark:bg-green-900/30" : "bg-red-100 text-red-600 dark:bg-red-900/30")} title={label as string}>{ok ? <Check className="h-3 w-3" /> : <X className="h-3 w-3" />}</span>)}</div></td>
                    <td className="px-3 py-2"><div className="flex flex-wrap gap-1">{d.issues?.map((iss: any, i: number) => <span key={i} className="px-1 py-0.5 rounded bg-red-50 text-red-600 dark:bg-red-950/20 text-xs">{iss}</span>)}</div></td>
                  </tr>
                ))}</tbody>
              </table></div>
            )}
          </div>
        </div>
      )}

      {/* CAEP EVENTS */}
      {tab === "caep" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> CAEP Event Stream</h2>
          {events.length === 0 ? <div className="py-8 text-center"><Zap className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No CAEP events.</p></div> : (
            <div className="space-y-2">{events.map(ev => (
              <div key={ev.id} className={"flex items-start gap-3 rounded-lg border p-3 " + (caepColors[ev.event_type] || "bg-gray-50 dark:bg-gray-900")}>
                <Zap className="h-4 w-4 mt-0.5 shrink-0 text-gray-400" />
                <div className="flex-1 min-w-0"><div className="flex items-center gap-2"><span className="font-mono text-xs font-medium">{ev.event_type}</span><span className="text-xs text-gray-500">{ev.subject}</span></div><p className="mt-0.5 text-xs text-gray-400">{ev.details}</p><div className="mt-1 flex items-center gap-2 text-xs text-gray-400"><span>by: {ev.actor}</span><span>·</span><span>{ev.provider}</span><span>·</span><span>{new Date(ev.timestamp).toLocaleString()}</span></div></div>
              </div>
            ))}</div>
          )}
        </div>
      )}

      {/* SCIM PUSH */}
      {tab === "scim" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><ArrowRight className="h-4 w-4" /> SCIM Outbound Push</h2>
            <div className="space-y-3">
              <div><label className="text-sm font-medium">Target Provider</label><select aria-label="SCIM provider" value={scimProvider} onChange={e => setScimProvider(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="">Select...</option>{providers.filter(p => p.connected).map(p => <option key={p.id} value={p.id}>{p.name}</option>)}</select></div>
              <div className="space-y-2"><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={scimPushUsers} onChange={e => setScimPushUsers(e.target.checked)} className="rounded" /> Push users</label><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={scimPushGroups} onChange={e => setScimPushGroups(e.target.checked)} className="rounded" /> Push groups</label></div>
              <button onClick={pushSCIM} disabled={!scimProvider || pushing} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{pushing ? <Loader2 className="h-4 w-4 animate-spin" /> : <ArrowRight className="h-4 w-4" />} Push Now</button>
            </div>
          </div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Activity className="h-4 w-4" /> Sync Status</h2>
            {providers.filter(p => p.connected).map(p => (
              <div key={p.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700 mb-2"><div><span className="font-medium text-sm">{p.name}</span><p className="text-xs text-gray-400">{p.synced_users} users · {p.synced_groups} groups</p></div><span className="text-xs text-gray-400">{p.last_sync ? new Date(p.last_sync).toLocaleString() : "Never synced"}</span></div>
            ))}
          </div>
        </div>
      )}

      {/* TERRAFORM EXPORT */}
      {tab === "terraform" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div className={cardCls + " lg:col-span-1"}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Code className="h-4 w-4" /> Select Provider</h2>
            <div className="space-y-2">{providers.map(p => <button key={p.id} onClick={() => setTfProvider(p.id)} aria-pressed={tfProvider === p.id} className={"flex w-full items-center gap-2 rounded-lg border p-3 text-left " + (tfProvider === p.id ? "border-indigo-500 bg-indigo-50 dark:bg-indigo-950/30" : "border-gray-200 dark:border-gray-700")}><Cloud className="h-4 w-4 text-gray-400" /><span className="text-sm font-medium">{p.name}</span><ChevronRight className="h-3 w-3 text-gray-400 ml-auto" /></button>)}</div>
          </div>
          <div className={cardCls + " lg:col-span-2"}>
            <div className="mb-3 flex items-center justify-between"><h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Code className="h-4 w-4" /> Terraform Configuration</h2><div className="flex gap-2"><button onClick={copyTF} disabled={!tfProvider} aria-label="Copy" className="rounded-lg border border-gray-300 px-2 py-1 text-xs dark:border-gray-700 disabled:opacity-50">{copied ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />} Copy</button><button onClick={() => { const blob = new Blob([generateTF()], { type: "text/plain" }); const url = URL.createObjectURL(blob); const a = document.createElement("a"); a.href = url; a.download = `ztna-${tfProvider}.tf`; a.click(); }} disabled={!tfProvider} className="rounded-lg border border-gray-300 px-2 py-1 text-xs dark:border-gray-700 disabled:opacity-50"><Download className="h-3 w-3" /></button></div></div>
            <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-xs text-green-400 font-mono max-h-96 overflow-y-auto">{generateTF()}</pre>
          </div>
        </div>
      )}

      </>)}

      {/* Register provider dialog */}
      {showRegister && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowRegister(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Cloud className="h-5 w-5 text-indigo-500" /> Add ZTNA Provider</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">Provider Type</label><div className="mt-2 grid grid-cols-2 gap-2">{Object.entries(providerConfig).map(([key, cfg]) => { const PIcon = cfg.icon; return <button key={key} onClick={() => { setNewProviderType(key as Provider["type"]); setNewConfig({}); }} aria-pressed={newProviderType === key} className={"flex items-center gap-2 rounded-lg border-2 p-3 " + (newProviderType === key ? "border-indigo-500 bg-indigo-50 dark:bg-indigo-950/30" : "border-gray-200 dark:border-gray-700")}><PIcon className={"h-4 w-4 " + cfg.color} /><span className="text-xs font-medium">{cfg.name}</span></button>; })}</div></div>
              <div><label className="text-sm font-medium">Name *</label><input aria-label="Provider name" type="text" value={newProviderName} onChange={e => setNewProviderName(e.target.value)} placeholder="Production Zscaler" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              {providerConfig[newProviderType].fields.map(f => <div key={f.key}><label className="text-sm font-medium">{f.label}</label><input aria-label={f.label} type={f.key.includes("key") || f.key.includes("token") ? "password" : "text"} value={newConfig[f.key] || ""} onChange={e => setNewConfig({ ...newConfig, [f.key]: e.target.value })} placeholder={f.label} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>)}
            </div>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setShowRegister(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button><button onClick={registerProvider} disabled={!newProviderName || registering} className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{registering ? <Loader2 className="h-4 w-4 animate-spin" /> : "Connect"}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
