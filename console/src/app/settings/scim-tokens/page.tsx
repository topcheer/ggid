"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Key, Plus, Trash2, Copy, Check, Eye, EyeOff, RefreshCw, Loader2,
  AlertCircle, X, Clock, ShieldCheck,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface SCIMToken {
  id: string;
  name: string;
  token_preview: string;
  created_at: string;
  last_used: string | null;
  expires_at: string | null;
  scopes: string[];
  active: boolean;
}

export default function SCIMTokenPage() {
  const t = useTranslations();
  const [tokens, setTokens] = useState<SCIMToken[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState("");
  const [newExpiry, setNewExpiry] = useState("");
  const [newScopes, setNewScopes] = useState("scim:read,scim:write");
  const [creating, setCreating] = useState(false);
  const [revealedToken, setRevealedToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [revokingId, setRevokingId] = useState<string | null>(null);

  const loadTokens = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/identity/scim/tokens", {
        headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      }).catch(() => null);
      if (res?.ok) {
        const d = await res.json();
        setTokens(d.tokens || d.items || []);
      }
    } catch { setError("Failed to load SCIM tokens"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadTokens(); }, [loadTokens]);

  const createToken = async () => {
    if (!newName) return;
    setCreating(true);
    try {
      const body: Record<string, unknown> = { name: newName, scopes: newScopes.split(",").map(s => s.trim()) };
      if (newExpiry) body.expires_at = newExpiry;
      const res = await fetch("/api/v1/identity/scim/tokens", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify(body),
      });
      if (res.ok) {
        const data = await res.json();
        if (data.token || data.full_token) {
          setRevealedToken(data.token || data.full_token);
        }
        setShowCreate(false); setNewName(""); setNewExpiry(""); setNewScopes("scim:read,scim:write");
        loadTokens();
      } else { setError("Failed to create token"); }
    } catch { setError("Network error"); }
    finally { setCreating(false); }
  };

  const revokeToken = async (id: string) => {
    setRevokingId(id);
    try {
      await fetch(`/api/v1/identity/scim/tokens/${id}`, {
        method: "DELETE",
        headers: { ...authHeader(), "X-Tenant-ID": TENANT_ID },
      });
      setTokens(prev => prev.filter(t => t.id !== id));
    } catch { setError("Failed to revoke token"); }
    finally { setRevokingId(null); }
  };

  const toggleToken = async (id: string, active: boolean) => {
    try {
      await fetch(`/api/v1/identity/scim/tokens/${id}`, {
        method: "PUT",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ active: !active }),
      });
      setTokens(prev => prev.map(t => t.id === id ? { ...t, active: !active } : t));
    } catch { setError("Failed to toggle token"); }
  };

  const copyToken = async () => {
    if (!revealedToken) return;
    try {
      await navigator.clipboard.writeText(revealedToken);
      setCopied(true); setTimeout(() => setCopied(false), 3000);
    } catch { /* noop */ }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Key className="h-6 w-6 text-indigo-500" />
            SCIM Token Management
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Manage bearer tokens for SCIM 2.0 provisioning endpoints. Tokens grant /scim/v2/ API access.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => setShowCreate(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700">
            <Plus className="h-4 w-4" /> Generate Token
          </button>
          <button onClick={loadTokens} disabled={loading} aria-label="Refresh tokens" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
            <RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh
          </button>
        </div>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* New token reveal */}
      {revealedToken && (
        <div role="status" className="rounded-xl border border-green-300 bg-green-50 p-4 dark:border-green-700 dark:bg-green-950/30">
          <div className="flex items-center justify-between">
            <h3 className="flex items-center gap-2 text-sm font-semibold text-green-800 dark:text-green-400"><ShieldCheck className="h-4 w-4" /> Token Created — Copy Now!</h3>
            <button onClick={() => setRevealedToken(null)} aria-label="Dismiss" className="text-green-600"><X className="h-4 w-4" /></button>
          </div>
          <div className="mt-2 flex items-center gap-2">
            <code className="flex-1 truncate rounded-lg bg-white px-3 py-2 font-mono text-xs dark:bg-gray-900">{revealedToken}</code>
            <button onClick={copyToken} aria-label="Copy token" className="rounded-lg bg-green-600 p-2 text-white hover:bg-green-700">{copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}</button>
          </div>
          <p className="mt-2 text-xs text-green-700 dark:text-green-500">This token will not be shown again. Store it securely.</p>
        </div>
      )}

      {/* Stats */}
      <div className="grid grid-cols-3 gap-4">
        <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Total</span><p className="mt-2 text-2xl font-bold">{tokens.length}</p></div>
        <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Active</span><p className="mt-2 text-2xl font-bold text-green-600">{tokens.filter(t => t.active).length}</p></div>
        <div className={cardCls}><span className="text-xs font-semibold uppercase text-gray-400">Expired/Revoked</span><p className="mt-2 text-2xl font-bold text-gray-400">{tokens.filter(t => !t.active).length}</p></div>
      </div>

      {/* Token list */}
      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : tokens.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><Key className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No SCIM tokens configured.</p></div></div>
      ) : (
        <div className="space-y-3">
          {tokens.map(tok => (
            <div key={tok.id} className={cardCls}>
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-gray-900 dark:text-white">{tok.name}</span>
                    <span className={"px-2 py-0.5 rounded text-xs font-medium " + (tok.active ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400")}>{tok.active ? "Active" : "Inactive"}</span>
                  </div>
                  <div className="mt-1 flex flex-wrap items-center gap-3 text-xs text-gray-500">
                    <span className="font-mono">{tok.token_preview || tok.id.substring(0, 16) + "..."}</span>
                    {tok.scopes?.length > 0 && tok.scopes.map(s => <span key={s} className="px-1.5 py-0.5 rounded bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400 font-mono">{s}</span>)}
                  </div>
                  <div className="mt-1 flex items-center gap-3 text-xs text-gray-400">
                    <span className="flex items-center gap-1"><Clock className="h-3 w-3" /> Created: {tok.created_at ? new Date(tok.created_at).toLocaleDateString() : "—"}</span>
                    {tok.last_used && <span>Last used: {new Date(tok.last_used).toLocaleString()}</span>}
                    {tok.expires_at && <span className={new Date(tok.expires_at) < new Date() ? "text-red-500" : ""}>Expires: {new Date(tok.expires_at).toLocaleDateString()}</span>}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <button onClick={() => toggleToken(tok.id, tok.active)} aria-label={`${tok.active ? "Disable" : "Enable"} ${tok.name}`} aria-pressed={tok.active} className="rounded-lg border border-gray-300 px-2 py-1 text-xs dark:border-gray-700">{tok.active ? "Disable" : "Enable"}</button>
                  <button onClick={() => revokeToken(tok.id)} disabled={revokingId === tok.id} aria-label={`Revoke ${tok.name}`} className="rounded-lg bg-red-50 p-1.5 text-red-500 hover:bg-red-100 dark:bg-red-950/20 disabled:opacity-50">{revokingId === tok.id ? <Loader2 className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}</button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Create dialog */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowCreate(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-indigo-500" /> Generate SCIM Token</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">Name *</label><input aria-label="Token name" type="text" value={newName} onChange={e => setNewName(e.target.value)} placeholder="Okta SCIM Connector" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">Scopes</label><input aria-label="Token scopes" type="text" value={newScopes} onChange={e => setNewScopes(e.target.value)} placeholder="scim:read,scim:write" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">Expiry (optional)</label><input aria-label="Token expiry date" type="date" value={newExpiry} onChange={e => setNewExpiry(e.target.value)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowCreate(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button>
              <button onClick={createToken} disabled={!newName || creating} className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{creating ? <Loader2 className="h-4 w-4 animate-spin" /> : "Generate"}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
