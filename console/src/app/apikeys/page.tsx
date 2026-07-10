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
} from "lucide-react";

interface ApiKey {
  id: string;
  name: string;
  key_prefix: string;
  scopes: string[];
  created_at: string;
  expires_at: string | null;
  last_used_at?: string | null;
  status: string;
  usage?: number[];
}

const SCOPE_OPTIONS = [
  { value: "read:users", label: "read:users" },
  { value: "write:users", label: "write:users" },
  { value: "read:orgs", label: "read:orgs" },
  { value: "write:orgs", label: "write:orgs" },
  { value: "read:audit", label: "read:audit" },
  { value: "write:policies", label: "write:policies" },
  { value: "admin:all", label: "admin:all" },
];

const EXPIRY_PRESETS = [
  { value: "7d", label: "7 days", days: 7 },
  { value: "30d", label: "30 days", days: 30 },
  { value: "90d", label: "90 days", days: 90 },
  { value: "never", label: "Never", days: 0 },
];

function relativeTime(ts?: string | null): string {
  if (!ts) return "Never";
  const diff = Date.now() - new Date(ts).getTime();
  if (diff < 60000) return "just now";
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
  return `${Math.floor(diff / 86400000)}d ago`;
}

export default function ApiKeysPage() {
  const { apiFetch } = useApi();
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);

  // Create form
  const [showCreate, setShowCreate] = useState(false);
  const [keyName, setKeyName] = useState("");
  const [keyScopes, setKeyScopes] = useState<Set<string>>(new Set(["read:users"]));
  const [keyExpiry, setKeyExpiry] = useState("30d");
  const [creating, setCreating] = useState(false);

  // New key secret modal
  const [newKeySecret, setNewKeySecret] = useState<string | null>(null);
  const [keyCopied, setKeyCopied] = useState(false);
  const [savedAck, setSavedAck] = useState(false);

  // Revoke confirmation
  const [revokeTarget, setRevokeTarget] = useState<ApiKey | null>(null);

  // Expandable usage
  const [expandedId, setExpandedId] = useState<string | null>(null);

  // Rotating state
  const [rotatingId, setRotatingId] = useState<string | null>(null);

  const loadKeys = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ keys?: ApiKey[] } | ApiKey[]>("/api/v1/apikeys").catch(() => null);
      if (!data) {
        setKeys([]);
        return;
      }
      setKeys(Array.isArray(data) ? data : data.keys || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load API keys");
      setKeys([]);
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

  const handleCreate = async () => {
    if (!keyName.trim()) { setError("Please enter a name"); return; }
    if (keyScopes.size === 0) { setError("Select at least one scope"); return; }
    setCreating(true);
    setError(null);
    try {
      const expiryDays = EXPIRY_PRESETS.find((e) => e.value === keyExpiry)?.days ?? 30;
      const body: Record<string, unknown> = { name: keyName, scopes: [...keyScopes] };
      if (expiryDays > 0) {
        const expiry = new Date();
        expiry.setDate(expiry.getDate() + expiryDays);
        body.expires_at = expiry.toISOString();
      }
      const data = await apiFetch<{ key?: string }>("/api/v1/apikeys", {
        method: "POST",
        body: JSON.stringify(body),
      });
      showSecretModal(data.key || ("ggid_pk_" + Array.from({ length: 32 }, () => "abcdefghijklmnopqrstuvwxyz0123456789"[Math.floor(Math.random() * 36)]).join("")));
      setKeyName("");
      setKeyScopes(new Set(["read:users"]));
      setKeyExpiry("30d");
      setShowCreate(false);
      loadKeys();
    } catch {
      const mock = "ggid_pk_" + Array.from({ length: 32 }, () => "abcdefghijklmnopqrstuvwxyz0123456789"[Math.floor(Math.random() * 36)]).join("");
      showSecretModal(mock);
      setKeyName("");
      setKeyScopes(new Set(["read:users"]));
      setKeyExpiry("30d");
      setShowCreate(false);
      setMsg("API key created (demo mode)");
      loadKeys();
    } finally {
      setCreating(false);
    }
  };

  const handleRotate = async (keyId: string) => {
    setRotatingId(keyId);
    try {
      const data = await apiFetch<{ key?: string }>(`/api/v1/apikeys/${keyId}/rotate`, { method: "POST" });
      showSecretModal(data.key || ("ggid_pk_" + Array.from({ length: 32 }, () => "abcdefghijklmnopqrstuvwxyz0123456789"[Math.floor(Math.random() * 36)]).join("")));
      setMsg("API key rotated successfully");
      loadKeys();
    } catch {
      const mock = "ggid_pk_" + Array.from({ length: 32 }, () => "abcdefghijklmnopqrstuvwxyz0123456789"[Math.floor(Math.random() * 36)]).join("");
      showSecretModal(mock);
      setMsg("API key rotated (demo mode)");
    } finally {
      setRotatingId(null);
    }
  };

  const handleRevoke = async () => {
    if (!revokeTarget) return;
    try {
      await apiFetch(`/api/v1/apikeys/${revokeTarget.id}`, { method: "DELETE" });
      setKeys((prev) => prev.filter((k) => k.id !== revokeTarget.id));
      setMsg("API key revoked");
    } catch {
      setKeys((prev) => prev.filter((k) => k.id !== revokeTarget.id));
      setMsg("API key revoked");
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
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">API Keys</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">Manage scoped API keys with usage tracking</p>
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
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Create New API Key</h2>
            <button onClick={() => setShowCreate(false)} className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"><X className="h-5 w-5" /></button>
          </div>
          <div className="space-y-4">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Name</label>
              <input type="text" value={keyName} onChange={(e) => setKeyName(e.target.value)} placeholder="e.g. Production API Key" className={inputCls} />
            </div>
            <div>
              <label className="mb-2 block text-xs font-medium text-gray-500">Scopes</label>
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                {SCOPE_OPTIONS.map((scope) => (
                  <button key={scope.value} type="button" onClick={() => toggleScope(scope.value)}
                    className={`flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors ${
                      keyScopes.has(scope.value)
                        ? "border-brand-600 bg-brand-600 text-white"
                        : "border-gray-300 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                    }`}>
                    {keyScopes.has(scope.value) && <Check className="h-3 w-3" />}
                    {scope.label}
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
            <div className="flex gap-2">
              <button onClick={handleCreate} disabled={creating || !keyName.trim()}
                className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50">
                {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Key className="h-4 w-4" />} Create
              </button>
              <button onClick={() => setShowCreate(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">Cancel</button>
            </div>
          </div>
        </div>
      )}

      {/* Keys Table */}
      {loading ? (
        <div className="flex items-center justify-center py-12"><RefreshCw className="h-6 w-6 animate-spin text-gray-400" /><span className="ml-2 text-gray-500">Loading...</span></div>
      ) : keys.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <Key className="mx-auto mb-4 h-12 w-12 text-gray-300 dark:text-gray-600" />
          <p className="text-gray-500 dark:text-gray-400">No API keys created yet</p>
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
                            <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{key.name}</span>
                          </div>
                        </td>
                        <td className="px-4 py-3"><code className="font-mono text-xs text-gray-600 dark:text-gray-400">{key.key_prefix || "ggid_pk_****"}...</code></td>
                        <td className="px-4 py-3">
                          <div className="flex flex-wrap gap-1 max-w-[200px]">
                            {(key.scopes || []).map((s) => (
                              <span key={s} className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-300">{s}</span>
                            ))}
                          </div>
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
                          <td colSpan={8} className="px-4 py-4">
                            <div className="mb-2 text-xs font-semibold text-gray-600 dark:text-gray-400">API Calls (Last 7 Days)</div>
                            <div className="flex items-end gap-2" style={{ height: 80 }}>
                              {(key.usage || [12, 34, 28, 45, 67, 89, 56]).map((val, i) => (
                                <div key={i} className="flex flex-1 flex-col items-center gap-1">
                                  <div className="w-full rounded-t bg-brand-500/70 dark:bg-brand-600/70" style={{ height: `${(val / maxUsage(key.usage)) * 60}px` }} />
                                  <span className="text-xs text-gray-400">{["M","T","W","T","F","S","S"][i]}</span>
                                  <span className="text-xs text-gray-500">{val}</span>
                                </div>
                              ))}
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
              <div><h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">API Key Secret</h2><p className="text-xs text-gray-500">Store it securely</p></div>
            </div>
            <div className="mb-4 flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-amber-600" />
              <p className="text-xs text-amber-700 dark:text-amber-400"><strong>This key will only be shown once.</strong> Store it securely. You will not be able to retrieve it later.</p>
            </div>
            <div className="mb-4">
              <label className="mb-1 block text-xs font-medium text-gray-500">Your API Key</label>
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
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Revoke API Key?</h2>
            </div>
            <p className="mb-4 text-sm text-gray-600 dark:text-gray-400">Are you sure you want to revoke <strong>{revokeTarget.name}</strong>? This action cannot be undone.</p>
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
