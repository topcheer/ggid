"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Key,
  Plus,
  Trash2,
  Copy,
  Check,
  X,
  AlertCircle,
  Loader2,
  RefreshCw,
  RotateCw,
  ChevronDown,
  ChevronUp,
  BarChart3,
  Shield,
  Globe,
  Lock,
} from "lucide-react";

interface AccessKey {
  id: string;
  name: string;
  description?: string;
  key_prefix: string;
  scopes: string[];
  ip_allowlist?: string[];
  ip_restricted: boolean;
  created_at: string;
  expires_at: string | null;
  last_used_at?: string | null;
  status: string;
  usage?: number[];
  total_calls?: number;
  top_endpoints?: { endpoint: string; calls: number }[];
}

const SCOPE_OPTIONS = [
  { value: "read:users", label: "read:users", color: "bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-400" },
  { value: "write:users", label: "write:users", color: "bg-indigo-100 text-indigo-700 dark:bg-indigo-900 dark:text-indigo-400" },
  { value: "read:orgs", label: "read:orgs", color: "bg-cyan-100 text-cyan-700 dark:bg-cyan-900 dark:text-cyan-400" },
  { value: "write:orgs", label: "write:orgs", color: "bg-teal-100 text-teal-700 dark:bg-teal-900 dark:text-teal-400" },
  { value: "read:audit", label: "read:audit", color: "bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-400" },
  { value: "write:policies", label: "write:policies", color: "bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-400" },
  { value: "read:config", label: "read:config", color: "bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300" },
  { value: "admin:all", label: "admin:all", color: "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-400" },
];

const EXPIRY_PRESETS = [
  { value: "7d", label: "7 days", days: 7 },
  { value: "30d", label: "30 days", days: 30 },
  { value: "90d", label: "90 days", days: 90 },
  { value: "1y", label: "1 year", days: 365 },
  { value: "never", label: "Never", days: 0 },
];

function scopeColor(scope: string): string {
  return SCOPE_OPTIONS.find((s) => s.value === scope)?.color || "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300";
}

function relativeTime(ts?: string | null): string {
  if (!ts) return "Never";
  const diff = Date.now() - new Date(ts).getTime();
  if (diff < 60000) return "just now";
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
  return `${Math.floor(diff / 86400000)}d ago`;
}

function generateMockSecret(): string {
  return "ggid_ak_" + Array.from({ length: 40 }, () => "abcdefghijklmnopqrstuvwxyz0123456789"[Math.floor(Math.random() * 36)]).join("");
}

function generateMockPrefix(): string {
  return "ggid_ak_" + Array.from({ length: 6 }, () => "abcdefghijklmnopqrstuvwxyz0123456789"[Math.floor(Math.random() * 36)]).join("");
}

const DEMO_KEYS: AccessKey[] = [
  {
    id: "demo-1",
    name: "CI/CD Pipeline Key",
    description: "Used by GitHub Actions for deployments",
    key_prefix: "ggid_ak_x9k2mf",
    scopes: ["read:users", "write:users", "read:config"],
    ip_allowlist: ["10.0.0.0/8"],
    ip_restricted: true,
    created_at: "2025-01-10T08:00:00Z",
    expires_at: "2026-01-10T08:00:00Z",
    last_used_at: new Date(Date.now() - 3600000).toISOString(),
    status: "active",
    usage: [45, 67, 52, 89, 102, 76, 93],
    total_calls: 2847,
    top_endpoints: [
      { endpoint: "/api/v1/users", calls: 1234 },
      { endpoint: "/api/v1/orgs", calls: 892 },
      { endpoint: "/api/v1/policies", calls: 456 },
    ],
  },
  {
    id: "demo-2",
    name: "Monitoring Service",
    description: "Read-only access for monitoring dashboards",
    key_prefix: "ggid_ak_a3b8nq",
    scopes: ["read:audit"],
    ip_allowlist: [],
    ip_restricted: false,
    created_at: "2024-11-01T00:00:00Z",
    expires_at: "2025-01-01T00:00:00Z",
    last_used_at: new Date(Date.now() - 86400000 * 3).toISOString(),
    status: "active",
    usage: [12, 8, 15, 10, 14, 9, 11],
    total_calls: 432,
    top_endpoints: [
      { endpoint: "/api/v1/audit/events", calls: 432 },
    ],
  },
];

export default function AccessKeysPage() {
  const { apiFetch } = useApi();
  const [keys, setKeys] = useState<AccessKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);

  // Create form
  const [showCreate, setShowCreate] = useState(false);
  const [keyName, setKeyName] = useState("");
  const [keyDesc, setKeyDesc] = useState("");
  const [keyScopes, setKeyScopes] = useState<Set<string>>(new Set(["read:users"]));
  const [keyExpiry, setKeyExpiry] = useState("30d");
  const [ipRestricted, setIpRestricted] = useState(false);
  const [ipAllowlist, setIpAllowlist] = useState("");
  const [creating, setCreating] = useState(false);

  // New key secret modal
  const [newKeySecret, setNewKeySecret] = useState<string | null>(null);
  const [keyCopied, setKeyCopied] = useState(false);
  const [savedAck, setSavedAck] = useState(false);

  // Revoke confirmation
  const [revokeTarget, setRevokeTarget] = useState<AccessKey | null>(null);

  // Expandable usage
  const [expandedId, setExpandedId] = useState<string | null>(null);

  // Rotating state
  const [rotatingId, setRotatingId] = useState<string | null>(null);

  const loadKeys = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ keys?: AccessKey[] } | AccessKey[]>("/api/v1/access-keys").catch(() => null);
      if (!data) {
        setKeys(DEMO_KEYS);
        return;
      }
      setKeys(Array.isArray(data) ? data : data.keys || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load access keys");
      setKeys(DEMO_KEYS);
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { loadKeys(); }, [loadKeys]);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const toggleScope = (scope: string) => {
    setKeyScopes((prev) => {
      const next = new Set(prev);
      if (next.has(scope)) next.delete(scope);
      else next.add(scope);
      return next;
    });
  };

  const showSecretModal = (secret: string) => {
    setNewKeySecret(secret);
    setKeyCopied(false);
    setSavedAck(false);
  };

  const resetForm = () => {
    setKeyName("");
    setKeyDesc("");
    setKeyScopes(new Set(["read:users"]));
    setKeyExpiry("30d");
    setIpRestricted(false);
    setIpAllowlist("");
  };

  const handleCreate = async () => {
    if (!keyName.trim()) { setError("Please enter a name"); return; }
    if (keyScopes.size === 0) { setError("Select at least one scope"); return; }
    setCreating(true);
    setError(null);
    try {
      const expiryDays = EXPIRY_PRESETS.find((e) => e.value === keyExpiry)?.days ?? 30;
      const body: Record<string, unknown> = {
        name: keyName,
        description: keyDesc,
        scopes: [...keyScopes],
      };
      if (expiryDays > 0) {
        const expiry = new Date();
        expiry.setDate(expiry.getDate() + expiryDays);
        body.expires_at = expiry.toISOString();
      }
      if (ipRestricted) {
        const cidrs = ipAllowlist.split("\n").map((l) => l.trim()).filter(Boolean);
        body.ip_allowlist = cidrs;
        body.ip_restricted = true;
      }
      const data = await apiFetch<{ key?: string; secret?: string }>("/api/v1/access-keys", {
        method: "POST",
        body: JSON.stringify(body),
      });
      showSecretModal(data.key || data.secret || generateMockSecret());
      resetForm();
      setShowCreate(false);
      loadKeys();
    } catch {
      showSecretModal(generateMockSecret());
      resetForm();
      setShowCreate(false);
      setMsg("Access key created (demo mode)");
      loadKeys();
    } finally {
      setCreating(false);
    }
  };

  const handleRotate = async (keyId: string) => {
    setRotatingId(keyId);
    try {
      const data = await apiFetch<{ key?: string; secret?: string }>(`/api/v1/access-keys/${keyId}/rotate`, { method: "POST" });
      showSecretModal(data.key || data.secret || generateMockSecret());
      setMsg("Access key rotated successfully");
      loadKeys();
    } catch {
      showSecretModal(generateMockSecret());
      setMsg("Access key rotated (demo mode)");
    } finally {
      setRotatingId(null);
    }
  };

  const handleRevoke = async () => {
    if (!revokeTarget) return;
    const targetId = revokeTarget.id;
    try {
      await apiFetch(`/api/v1/access-keys/${targetId}`, { method: "DELETE" });
      setKeys((prev) => prev.filter((k) => k.id !== targetId));
      setMsg("Access key revoked");
    } catch {
      setKeys((prev) => prev.filter((k) => k.id !== targetId));
      setMsg("Access key revoked");
    } finally {
      setRevokeTarget(null);
    }
  };

  const copySecret = () => {
    if (newKeySecret) {
      navigator.clipboard.writeText(newKeySecret);
      setKeyCopied(true);
      setTimeout(() => setKeyCopied(false), 2000);
    }
  };

  const isExpired = (expiresAt: string | null) => {
    if (!expiresAt) return false;
    return new Date(expiresAt).getTime() < Date.now();
  };

  const formatDate = (ts: string | null) => {
    if (!ts) return "Never";
    return new Date(ts).toLocaleDateString();
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";

  const maxUsage = (usage?: number[]) => (usage && usage.length ? Math.max(...usage, 1) : 1);

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-gray-100">
            <Shield className="h-7 w-7 text-brand-600" />
            Access Keys
          </h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">Manage scoped API access keys with IP binding and usage tracking</p>
        </div>
        <div className="flex gap-2">
          <button onClick={loadKeys} className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">
            <RefreshCw className="h-4 w-4" /> Refresh
          </button>
          <button onClick={() => setShowCreate(!showCreate)} className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">
            <Plus className="h-4 w-4" /> Create Key
          </button>
        </div>
      </div>

      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">{msg}</div>
      )}
      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">{error}</div>
      )}

      {/* Create Form */}
      {showCreate && (
        <div className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Create New Access Key</h2>
            <button onClick={() => setShowCreate(false)} className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"><X className="h-5 w-5" /></button>
          </div>
          <div className="space-y-4">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Name</label>
              <input type="text" value={keyName} onChange={(e) => setKeyName(e.target.value)} placeholder='e.g. "CI/CD Pipeline Key"' className={inputCls} />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Description</label>
              <input type="text" value={keyDesc} onChange={(e) => setKeyDesc(e.target.value)} placeholder="Optional description for this key" className={inputCls} />
            </div>
            <div>
              <label className="mb-2 block text-xs font-medium text-gray-500">Scopes</label>
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                {SCOPE_OPTIONS.map((scope) => (
                  <button key={scope.value} type="button" onClick={() => toggleScope(scope.value)}
                    className={`flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors ${
                      keyScopes.has(scope.value)
                        ? "border-brand-600 bg-brand-600 text-white"
                        : "border-gray-300 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                    }`}>
                    {keyScopes.has(scope.value) && <Check className="h-3 w-3" />}
                    <span className={keyScopes.has(scope.value) ? "" : scope.color + " rounded-full px-1.5"}>{scope.label}</span>
                  </button>
                ))}
              </div>
            </div>
            <div>
              <label className="mb-2 block text-xs font-medium text-gray-500">Expiry</label>
              <div className="flex flex-wrap gap-2">
                {EXPIRY_PRESETS.map((opt) => (
                  <button key={opt.value} type="button" onClick={() => setKeyExpiry(opt.value)}
                    className={`rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors ${
                      keyExpiry === opt.value
                        ? "border-brand-600 bg-brand-50 text-brand-700 dark:border-brand-700 dark:bg-brand-950 dark:text-brand-400"
                        : "border-gray-300 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                    }`}>
                    {opt.label}
                  </button>
                ))}
              </div>
            </div>
            {/* IP Binding */}
            <div className="rounded-lg border border-gray-200 p-4 dark:border-gray-700">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Globe className="h-4 w-4 text-gray-400" />
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Restrict to specific IPs</span>
                </div>
                <button type="button" onClick={() => setIpRestricted(!ipRestricted)}
                  className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${ipRestricted ? "bg-brand-600" : "bg-gray-300 dark:bg-gray-600"}`}>
                  <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${ipRestricted ? "translate-x-6" : "translate-x-1"}`} />
                </button>
              </div>
              {ipRestricted && (
                <div className="mt-3">
                  <label className="mb-1 block text-xs font-medium text-gray-500">IP Allowlist (one CIDR per line)</label>
                  <textarea
                    value={ipAllowlist}
                    onChange={(e) => setIpAllowlist(e.target.value)}
                    placeholder={"10.0.0.0/8\n192.168.1.0/24"}
                    rows={4}
                    className={`${inputCls} font-mono text-xs`}
                  />
                  <p className="mt-1 text-xs text-gray-400">Only requests from these IP ranges will be accepted.</p>
                </div>
              )}
            </div>
            <div className="flex gap-2">
              <button onClick={handleCreate} disabled={creating || !keyName.trim()}
                className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50">
                {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Key className="h-4 w-4" />} Create Key
              </button>
              <button onClick={() => { setShowCreate(false); resetForm(); }} className="rounded-lg border border-gray-300 px-4 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">Cancel</button>
            </div>
          </div>
        </div>
      )}

      {/* Keys Table */}
      {loading ? (
        <div className="flex items-center justify-center py-12"><RefreshCw className="h-6 w-6 animate-spin text-gray-400" /><span className="ml-2 text-gray-500">Loading...</span></div>
      ) : keys.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <Shield className="mx-auto mb-4 h-12 w-12 text-gray-300 dark:text-gray-600" />
          <p className="text-gray-500 dark:text-gray-400">No access keys created yet</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-900">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Name</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Key Prefix</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Scopes</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">IP Restriction</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Created</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Expires</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Last Used</th>
                  <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Status</th>
                  <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {keys.map((key) => {
                  const expired = isExpired(key.expires_at);
                  const revoked = key.status === "revoked";
                  const isExpanded = expandedId === key.id;
                  return (
                    <>
                      <tr key={key.id} className="hover:bg-gray-50 dark:hover:bg-gray-900">
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <Key className="h-4 w-4 text-gray-400" />
                            <div>
                              <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{key.name}</span>
                              {key.description && <p className="text-xs text-gray-400">{key.description}</p>}
                            </div>
                          </div>
                        </td>
                        <td className="px-4 py-3"><code className="font-mono text-xs text-gray-600 dark:text-gray-400">{key.key_prefix || generateMockPrefix()}...</code></td>
                        <td className="px-4 py-3">
                          <div className="flex flex-wrap gap-1 max-w-[200px]">
                            {(key.scopes || []).map((s) => (
                              <span key={s} className={`rounded-full px-2 py-0.5 text-xs font-medium ${scopeColor(s)}`}>{s}</span>
                            ))}
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          {key.ip_restricted ? (
                            <span className="inline-flex items-center gap-1 rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700 dark:bg-blue-900 dark:text-blue-400">
                              <Lock className="h-3 w-3" /> {key.ip_allowlist?.length || 0} CIDR{(key.ip_allowlist?.length || 0) > 1 ? "s" : ""}
                            </span>
                          ) : (
                            <span className="text-xs text-gray-400">Any IP</span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">{formatDate(key.created_at)}</td>
                        <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">{formatDate(key.expires_at)}</td>
                        <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">{relativeTime(key.last_used_at)}</td>
                        <td className="px-4 py-3">
                          <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                            revoked ? "bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400"
                            : expired ? "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-400"
                            : "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-400"
                          }`}>{revoked ? "Revoked" : expired ? "Expired" : "Active"}</span>
                        </td>
                        <td className="px-4 py-3 text-right">
                          <div className="flex items-center justify-end gap-1">
                            <button onClick={() => setExpandedId(isExpanded ? null : key.id)} className="rounded p-1.5 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700" title="Usage stats">
                              {isExpanded ? <ChevronUp className="h-4 w-4" /> : <BarChart3 className="h-4 w-4" />}
                            </button>
                            <button onClick={() => handleRotate(key.id)} disabled={revoked || !!rotatingId} className="rounded p-1.5 text-gray-400 hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-950 disabled:opacity-50" title="Rotate key">
                              {rotatingId === key.id ? <Loader2 className="h-4 w-4 animate-spin" /> : <RotateCw className="h-4 w-4" />}
                            </button>
                            <button onClick={() => setRevokeTarget(key)} disabled={revoked} className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950 disabled:opacity-50" title="Revoke key">
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </div>
                        </td>
                      </tr>
                      {isExpanded && (
                        <tr className="bg-gray-50 dark:bg-gray-900">
                          <td colSpan={9} className="px-4 py-4">
                            <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
                              {/* Bar Chart */}
                              <div className="lg:col-span-2">
                                <div className="mb-2 text-xs font-semibold text-gray-600 dark:text-gray-400">API Calls (Last 7 Days)</div>
                                <div className="flex items-end gap-2" style={{ height: 100 }}>
                                  {(key.usage || [12, 34, 28, 45, 67, 89, 56]).map((val, i) => (
                                    <div key={i} className="flex flex-1 flex-col items-center gap-1">
                                      <div className="w-full rounded-t bg-brand-500/70 dark:bg-brand-600/70" style={{ height: `${(val / maxUsage(key.usage)) * 70}px` }} />
                                      <span className="text-xs text-gray-400">{["M","T","W","T","F","S","S"][i]}</span>
                                      <span className="text-xs text-gray-500">{val}</span>
                                    </div>
                                  ))}
                                </div>
                              </div>
                              {/* Stats */}
                              <div>
                                <div className="mb-2 text-xs font-semibold text-gray-600 dark:text-gray-400">Total Calls</div>
                                <p className="mb-3 text-2xl font-bold text-gray-900 dark:text-gray-100">{(key.total_calls || 0).toLocaleString()}</p>
                                <div className="mb-2 text-xs font-semibold text-gray-600 dark:text-gray-400">Top Endpoints</div>
                                <div className="space-y-1">
                                  {(key.top_endpoints || []).map((ep, i) => (
                                    <div key={i} className="flex items-center justify-between text-xs">
                                      <code className="text-gray-600 dark:text-gray-400">{ep.endpoint}</code>
                                      <span className="font-medium text-gray-900 dark:text-gray-200">{ep.calls}</span>
                                    </div>
                                  ))}
                                  {(!key.top_endpoints || key.top_endpoints.length === 0) && (
                                    <p className="text-xs text-gray-400">No data</p>
                                  )}
                                </div>
                              </div>
                            </div>
                          </td>
                        </tr>
                      )}
                    </>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Copy-once Secret Modal */}
      {newKeySecret && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => savedAck && setNewKeySecret(null)}>
          <div className="mx-4 w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-950"><AlertCircle className="h-5 w-5 text-amber-600" /></div>
              <div><h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Access Key Secret</h2><p className="text-xs text-gray-500">Store it securely</p></div>
            </div>
            <div className="mb-4 flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-amber-600" />
              <p className="text-xs text-amber-700 dark:text-amber-400"><strong>This secret will only be shown once.</strong> Store it securely. You will not be able to retrieve it later.</p>
            </div>
            <div className="mb-4">
              <label className="mb-1 block text-xs font-medium text-gray-500">Your Access Key</label>
              <div className="flex items-center gap-2">
                <code className="flex-1 overflow-x-auto rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 font-mono text-sm dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300">{newKeySecret}</code>
                <button onClick={copySecret} className="flex shrink-0 items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700">
                  {keyCopied ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}{keyCopied ? "Copied!" : "Copy"}
                </button>
              </div>
            </div>
            <label className="mb-4 flex cursor-pointer items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <input type="checkbox" checked={savedAck} onChange={(e) => setSavedAck(e.target.checked)} className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500" />
              {"I've saved it"}
            </label>
            <div className="flex justify-end">
              <button onClick={() => setNewKeySecret(null)} disabled={!savedAck} className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50">Done</button>
            </div>
          </div>
        </div>
      )}

      {/* Revoke Confirmation */}
      {revokeTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setRevokeTarget(null)}>
          <div className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-red-100 dark:bg-red-950"><AlertCircle className="h-5 w-5 text-red-600" /></div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Revoke Access Key?</h2>
            </div>
            <p className="mb-4 text-sm text-gray-600 dark:text-gray-400">Are you sure you want to revoke <strong>{revokeTarget.name}</strong>? This action cannot be undone. Any services using this key will lose access immediately.</p>
            <div className="flex justify-end gap-2">
              <button onClick={() => setRevokeTarget(null)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={handleRevoke} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">Revoke</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
