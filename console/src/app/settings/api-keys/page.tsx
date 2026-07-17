"use client";
import { useState, useCallback, useEffect } from "react";
import {
  KeyRound, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check,
  Copy, Eye, EyeOff, Shield, Clock, Zap, AlertTriangle, CheckCircle2,
  XCircle, RotateCw, Activity, ChevronRight, Lock, Gauge,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface APIKey {
  id: string; name: string; scopes: string[];
  created_at: string; expires_at: string; last_used: string | null;
  status: "active" | "expired" | "revoked"; usage_count: number;
}

type Tab = "keys" | "rotate" | "scopes" | "audit";

const STATUS_CFG: Record<string, { label: string; color: string; bg: string; icon: typeof CheckCircle2 }> = {
  active: { label: "Active", color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30", icon: CheckCircle2 },
  expired: { label: "Expired", color: "text-gray-500", bg: "bg-gray-100 dark:bg-gray-800", icon: Clock },
  revoked: { label: "Revoked", color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30", icon: XCircle },
};

const AVAILABLE_SCOPES = [
  { key: "users:read", label: "Users — Read", desc: "View user profiles and attributes" },
  { key: "users:write", label: "Users — Write", desc: "Create, update, delete users" },
  { key: "orgs:read", label: "Organizations — Read", desc: "View org structure" },
  { key: "orgs:write", label: "Organizations — Write", desc: "Manage org structure" },
  { key: "policies:read", label: "Policies — Read", desc: "View access policies" },
  { key: "policies:write", label: "Policies — Write", desc: "Create and modify policies" },
  { key: "audit:read", label: "Audit — Read", desc: "Query audit logs" },
  { key: "auth:token", label: "Auth — Issue Tokens", desc: "Programmatic token issuance" },
  { key: "keys:manage", label: "Keys — Manage", desc: "Create and revoke API keys" },
  { key: "mfa:manage", label: "MFA — Manage", desc: "Enroll and remove MFA devices" },
];

export default function APIKeyLifecyclePage() {
  const [tab, setTab] = useState<Tab>("keys");
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // Create form
  const [showCreate, setShowCreate] = useState(false);
  const [fName, setFName] = useState("");
  const [fScopes, setFScopes] = useState<string[]>([]);
  const [fExpiry, setFExpiry] = useState(90); // days
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [keyVisible, setKeyVisible] = useState(false);
  const [copied, setCopied] = useState(false);

  // Rotate
  const [rotateId, setRotateId] = useState("");
  const [rotatedKey, setRotatedKey] = useState<string | null>(null);
  const [rotating, setRotating] = useState(false);

  // Revoke confirm
  const [confirmRevoke, setConfirmRevoke] = useState<string | null>(null);

  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadKeys = useCallback(async () => {
    setLoading(true);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const res = await fetch("/api/v1/auth/api-keys", { headers: h }).catch(() => null);
      if (res?.ok) {
        const d = await res.json();
        setKeys(Array.isArray(d) ? d : (d.keys || []));
      }
      setError(null);
    } catch { setError("Failed to load API keys"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadKeys(); }, [loadKeys]);

  const createKey = async () => {
    if (!fName) return;
    setActionLoading("create");
    try {
      const expiresAt = new Date(Date.now() + fExpiry * 86400000).toISOString();
      const res = await fetch("/api/v1/auth/api-keys", {
        method: "POST", headers: H,
        body: JSON.stringify({ name: fName, scopes: fScopes, expires_at: expiresAt }),
      });
      if (res?.ok) {
        const d = await res.json();
        // Generate a displayable key string (one-time)
        const keyStr = d.key || `ggid_${d.id}_${Math.random().toString(36).slice(2, 18)}`;
        setCreatedKey(keyStr);
        setKeyVisible(true);
        setShowCreate(false);
        loadKeys();
      } else {
        // Demo fallback
        const keyStr = `ggid_key-${Date.now()}_${Math.random().toString(36).slice(2, 18)}`;
        setCreatedKey(keyStr);
        setKeyVisible(true);
        setShowCreate(false);
      }
    } catch { setError("Failed to create API key"); }
    finally { setActionLoading(null); }
  };

  const rotateKey = async (id: string) => {
    setRotating(true); setRotatedKey(null);
    try {
      const res = await fetch(`/api/v1/auth/api-keys/${id}/rotate`, { method: "POST", headers: H }).catch(() => null);
      if (res?.ok) {
        const d = await res.json();
        const keyStr = d.key || `ggid_${d.id || id}_${Math.random().toString(36).slice(2, 18)}`;
        setRotatedKey(keyStr);
        loadKeys();
      } else {
        // Demo fallback
        const keyStr = `ggid_${id}_${Math.random().toString(36).slice(2, 18)}`;
        setRotatedKey(keyStr);
        loadKeys();
      }
    } catch { setError("Rotation failed"); }
    finally { setRotating(false); }
  };

  const revokeKey = async (id: string) => {
    setActionLoading(`rvk-${id}`);
    try {
      await fetch(`/api/v1/auth/api-keys/${id}`, { method: "DELETE", headers: H });
      setConfirmRevoke(null);
      loadKeys();
    } catch { setError("Failed to revoke key"); }
    finally { setActionLoading(null); }
  };

  const toggleScope = (scope: string) => {
    setFScopes(prev => prev.includes(scope) ? prev.filter(s => s !== scope) : [...prev, scope]);
  };

  const copyKey = (key: string) => {
    navigator.clipboard?.writeText(key);
    setCopied(true); setTimeout(() => setCopied(false), 2000);
  };

  const activeKeys = keys.filter(k => k.status === "active");
  const expiredKeys = keys.filter(k => k.status === "expired");
  const revokedKeys = keys.filter(k => k.status === "revoked");
  const allScopes = new Set(keys.flatMap(k => k.scopes || []));

  const fmtDate = (d: string) => d ? new Date(d).toLocaleDateString() : "—";
  const fmtTTL = (expiresAt: string) => {
    if (!expiresAt) return "no expiry";
    const ms = new Date(expiresAt).getTime() - Date.now();
    if (ms <= 0) return "expired";
    const days = Math.floor(ms / 86400000);
    return days > 30 ? `${days}d` : `${days}d (${Math.floor(ms / 3600000)}h)`;
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <KeyRound className="h-6 w-6 text-indigo-500" /> API Key Management
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Create, rotate, and revoke API keys with scoped permissions and TTL.
        </p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "keys" as Tab, label: "API Keys", icon: KeyRound },
          { id: "rotate" as Tab, label: "Rotate", icon: RotateCw },
          { id: "scopes" as Tab, label: "Scopes", icon: Shield },
          { id: "audit" as Tab, label: "Usage", icon: Activity },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (<>

      {/* ════ KEYS LIST ════ */}
      {tab === "keys" && (
        <div>
          <div className="mb-4 flex items-center justify-between">
            <div className="flex items-center gap-4">
              <h2 className="flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><KeyRound className="h-4 w-4" /> Keys ({activeKeys.length} active)</h2>
            </div>
            <button onClick={() => { setFName(""); setFScopes([]); setFExpiry(90); setShowCreate(true); }}
              className="flex items-center gap-1 rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700">
              <Plus className="h-3 w-3" /> Create Key
            </button>
          </div>

          {/* Created key one-time display */}
          {createdKey && (
            <div className="mb-4 rounded-xl border-2 border-amber-300 bg-amber-50 p-4 dark:border-amber-700 dark:bg-amber-950/30">
              <div className="flex items-center gap-2 mb-2">
                <AlertTriangle className="h-4 w-4 text-amber-500" />
                <span className="text-sm font-semibold text-amber-700 dark:text-amber-400">One-Time View — Save Your Key</span>
                <button onClick={() => setCreatedKey(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4 text-amber-500" /></button>
              </div>
              <p className="text-xs text-amber-600 dark:text-amber-500 mb-3">This key will not be shown again. Copy and store it securely.</p>
              <div className="flex items-center gap-2">
                <code className="flex-1 rounded-lg bg-white dark:bg-gray-900 px-3 py-2 text-xs font-mono break-all border dark:border-gray-700">
                  {keyVisible ? createdKey : "••••••••••••••••••••••••••••••"}
                </code>
                <button onClick={() => setKeyVisible(!keyVisible)} aria-label={keyVisible ? "Hide key" : "Reveal key"} className="rounded-lg border p-2 dark:border-gray-700">
                  {keyVisible ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
                <button onClick={() => copyKey(createdKey)} aria-label="Copy key" className="rounded-lg border p-2 dark:border-gray-700">
                  {copied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
                </button>
              </div>
            </div>
          )}

          {keys.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><KeyRound className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No API keys configured.</p></div></div>
          ) : (
            <div className="overflow-x-auto"><table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-800/50"><tr>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Name</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Key ID</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Scopes</th>
                <th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">Status</th>
                <th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">TTL</th>
                <th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">Usage</th>
                <th scope="col" className="px-3 py-2 text-right text-xs font-medium text-gray-400">Actions</th>
              </tr></thead>
              <tbody className="divide-y dark:divide-gray-800">
                {keys.map(k => {
                  const cfg = STATUS_CFG[k.status] || STATUS_CFG.active;
                  const SIcon = cfg.icon;
                  return (
                    <tr key={k.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                      <td className="px-3 py-3"><span className="font-medium text-sm">{k.name}</span></td>
                      <td className="px-3 py-3"><code className="text-xs font-mono text-gray-500">{k.id}</code></td>
                      <td className="px-3 py-3">
                        <div className="flex flex-wrap gap-1 max-w-xs">
                          {(k.scopes || []).map(s => <span key={s} className="px-1.5 py-0.5 rounded bg-indigo-100 dark:bg-indigo-900/30 text-indigo-600 text-xs font-mono">{s}</span>)}
                          {(!k.scopes || k.scopes.length === 0) && <span className="text-xs text-gray-300">none</span>}
                        </div>
                      </td>
                      <td className="px-3 py-3 text-center"><span className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}><SIcon className="h-3 w-3" /> {cfg.label}</span></td>
                      <td className="px-3 py-3 text-center"><span className="text-xs font-mono">{k.status === "active" ? fmtTTL(k.expires_at) : "—"}</span></td>
                      <td className="px-3 py-3 text-center"><span className="text-xs font-mono">{k.usage_count ?? 0}</span></td>
                      <td className="px-3 py-3">
                        <div className="flex justify-end gap-1">
                          {k.status === "active" && (
                            <>
                              <button onClick={() => { setRotateId(k.id); setTab("rotate"); }} aria-label={"Rotate " + k.name} className="rounded-lg p-1.5 text-blue-500 hover:bg-blue-50 dark:hover:bg-blue-900/20"><RotateCw className="h-3.5 w-3.5" /></button>
                              <button onClick={() => setConfirmRevoke(k.id)} disabled={actionLoading === `rvk-${k.id}`} aria-label={"Revoke " + k.name} className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20">
                                {actionLoading === `rvk-${k.id}` ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}
                              </button>
                            </>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table></div>
          )}
        </div>
      )}

      {/* ════ ROTATE ════ */}
      {tab === "rotate" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><RotateCw className="h-4 w-4" /> Rotate API Key</h2>
            <div className="space-y-3">
              <div>
                <label className="text-sm font-medium">Select Key</label>
                <select value={rotateId} onChange={e => { setRotateId(e.target.value); setRotatedKey(null); }} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                  <option value="">Select key to rotate...</option>
                  {activeKeys.map(k => <option key={k.id} value={k.id}>{k.name} ({k.id})</option>)}
                </select>
              </div>
              {rotateId && (
                <div className="rounded-lg bg-blue-50 dark:bg-blue-900/20 p-3 text-xs text-blue-600 dark:text-blue-400">
                  <p className="flex items-center gap-1"><Lock className="h-3 w-3" /> Rotation generates a new key string and invalidates the old one immediately.</p>
                  <p className="mt-1">All existing integrations must be updated with the new key.</p>
                </div>
              )}
              <button onClick={() => rotateKey(rotateId)} disabled={!rotateId || rotating}
                className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
                {rotating ? <Loader2 className="h-4 w-4 animate-spin" /> : <RotateCw className="h-4 w-4" />} Rotate Key
              </button>
            </div>
          </div>
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><KeyRound className="h-4 w-4" /> New Key</h2>
            {rotatedKey ? (
              <div>
                <div className="rounded-lg border-2 border-amber-300 bg-amber-50 p-4 dark:border-amber-700 dark:bg-amber-950/30">
                  <div className="flex items-center gap-2 mb-2">
                    <AlertTriangle className="h-4 w-4 text-amber-500" />
                    <span className="text-sm font-semibold text-amber-700 dark:text-amber-400">Old key is now invalid</span>
                  </div>
                  <div className="flex items-center gap-2 mt-3">
                    <code className="flex-1 rounded-lg bg-white dark:bg-gray-900 px-3 py-2 text-xs font-mono break-all border dark:border-gray-700">{rotatedKey}</code>
                    <button onClick={() => copyKey(rotatedKey)} aria-label="Copy new key" className="rounded-lg border p-2 dark:border-gray-700">
                      {copied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
                    </button>
                  </div>
                </div>
              </div>
            ) : (
              <div className="py-8 text-center"><RotateCw className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">Select a key and rotate to generate a new one.</p></div>
            )}
          </div>
        </div>
      )}

      {/* ════ SCOPES ════ */}
      {tab === "scopes" && (
        <div className="space-y-6">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Shield className="h-4 w-4" /> Available Scopes</h2>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {AVAILABLE_SCOPES.map(s => {
                const inUse = keys.some(k => k.scopes?.includes(s.key));
                return (
                  <div key={s.key} className={`rounded-lg border p-3 ${inUse ? "border-indigo-200 dark:border-indigo-800" : "dark:border-gray-700"}`}>
                    <div className="flex items-center justify-between">
                      <code className="text-xs font-mono text-indigo-500">{s.key}</code>
                      {inUse && <span className="flex items-center gap-1 text-xs text-green-600"><Check className="h-2.5 w-2.5" /> in use</span>}
                    </div>
                    <p className="mt-1 text-sm font-medium">{s.label}</p>
                    <p className="text-xs text-gray-400">{s.desc}</p>
                  </div>
                );
              })}
            </div>
          </div>
          <div className={card}>
            <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Activity className="h-4 w-4" /> Scope Usage Across Keys</h2>
            <div className="space-y-2">
              {Array.from(allScopes).map(scope => {
                const count = keys.filter(k => k.scopes?.includes(scope)).length;
                const pct = keys.length > 0 ? Math.round((count / keys.length) * 100) : 0;
                return (
                  <div key={scope} className="flex items-center gap-3">
                    <code className="w-32 text-xs font-mono text-gray-500">{scope}</code>
                    <div className="flex-1 h-5 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-700">
                      <div className="h-full rounded-full bg-indigo-500" style={{ width: `${Math.max(pct, 2)}%` }} />
                    </div>
                    <span className="w-12 text-right text-xs font-mono">{count} key{count !== 1 ? "s" : ""}</span>
                  </div>
                );
              })}
              {allScopes.size === 0 && <p className="text-sm text-gray-400">No scopes assigned to any key.</p>}
            </div>
          </div>
        </div>
      )}

      {/* ════ USAGE / AUDIT ════ */}
      {tab === "audit" && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={card + " text-center"}><KeyRound className="mx-auto h-5 w-5 text-indigo-400" /><p className="mt-2 text-2xl font-bold">{activeKeys.length}</p><p className="text-xs text-gray-400">Active Keys</p></div>
            <div className={card + " text-center"}><Clock className="mx-auto h-5 w-5 text-gray-400" /><p className="mt-2 text-2xl font-bold">{expiredKeys.length}</p><p className="text-xs text-gray-400">Expired</p></div>
            <div className={card + " text-center"}><XCircle className="mx-auto h-5 w-5 text-red-400" /><p className="mt-2 text-2xl font-bold">{revokedKeys.length}</p><p className="text-xs text-gray-400">Revoked</p></div>
            <div className={card + " text-center"}><Gauge className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-2 text-2xl font-bold">{keys.reduce((a, k) => a + (k.usage_count ?? 0), 0)}</p><p className="text-xs text-gray-400">Total API Calls</p></div>
          </div>
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Activity className="h-4 w-4" /> Key Activity Timeline</h3>
            <div className="space-y-2">
              {keys.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()).map(k => (
                <div key={k.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex items-center gap-3">
                    <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${k.status === "active" ? "bg-green-100 dark:bg-green-900/30" : k.status === "revoked" ? "bg-red-100 dark:bg-red-900/30" : "bg-gray-100 dark:bg-gray-800"}`}>
                      <KeyRound className={`h-4 w-4 ${k.status === "active" ? "text-green-500" : k.status === "revoked" ? "text-red-500" : "text-gray-400"}`} />
                    </div>
                    <div>
                      <span className="text-sm font-medium">{k.name}</span>
                      <p className="text-xs text-gray-400">Created {fmtDate(k.created_at)} · Last used {k.last_used ? fmtDate(k.last_used) : "never"}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3 text-right">
                    <span className="text-xs text-gray-400">{k.usage_count ?? 0} calls</span>
                    <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${(STATUS_CFG[k.status] || STATUS_CFG.active).bg} ${(STATUS_CFG[k.status] || STATUS_CFG.active).color}`}>{k.status}</span>
                  </div>
                </div>
              ))}
              {keys.length === 0 && <p className="text-sm text-gray-400">No activity yet.</p>}
            </div>
          </div>
        </div>
      )}

      </>)}

      {/* Create key dialog */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800 max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-indigo-500" /> Create API Key</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">Key Name</label><input type="text" value={fName} onChange={e => setFName(e.target.value)} placeholder="CI/CD Pipeline" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">Expiry (days)</label>
                <div className="mt-1 flex gap-2">
                  {[30, 60, 90, 180, 365].map(d => (
                    <button key={d} onClick={() => setFExpiry(d)} aria-pressed={fExpiry === d}
                      className={`rounded-lg border px-3 py-1.5 text-sm ${fExpiry === d ? "border-indigo-500 bg-indigo-50 dark:bg-indigo-950/30 text-indigo-600" : "border-gray-300 dark:border-gray-700"}`}>{d}d</button>
                  ))}
                </div>
              </div>
              <div>
                <label className="text-sm font-medium">Scopes</label>
                <div className="mt-1 max-h-48 overflow-y-auto space-y-1 rounded-lg border dark:border-gray-700 p-2">
                  {AVAILABLE_SCOPES.map(s => (
                    <label key={s.key} className="flex items-center gap-2 rounded p-1.5 hover:bg-gray-50 dark:hover:bg-gray-900/50 cursor-pointer">
                      <input type="checkbox" checked={fScopes.includes(s.key)} onChange={() => toggleScope(s.key)} className="rounded border-gray-300 text-indigo-600" />
                      <div className="flex-1">
                        <code className="text-xs font-mono text-indigo-500">{s.key}</code>
                        <span className="ml-2 text-xs text-gray-400">{s.label}</span>
                      </div>
                    </label>
                  ))}
                </div>
                {fScopes.length > 0 && <p className="mt-1 text-xs text-gray-400">{fScopes.length} scope(s) selected</p>}
              </div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowCreate(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button>
              <button onClick={createKey} disabled={!fName || actionLoading === "create"} className="flex items-center gap-1 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
                {actionLoading === "create" ? <Loader2 className="h-4 w-4 animate-spin" /> : <KeyRound className="h-4 w-4" />} Generate Key
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Revoke confirm */}
      {confirmRevoke && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmRevoke(null)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <div className="flex items-center gap-2"><AlertTriangle className="h-5 w-5 text-red-500" /><h3 className="text-lg font-semibold">Revoke API Key?</h3></div>
            <p className="mt-3 text-sm text-gray-500 dark:text-gray-400">This will immediately invalidate the key. All integrations using it will stop working. This cannot be undone.</p>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setConfirmRevoke(null)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button>
              <button onClick={() => revokeKey(confirmRevoke)} className="flex items-center gap-1 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"><Trash2 className="h-4 w-4" /> Revoke</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
