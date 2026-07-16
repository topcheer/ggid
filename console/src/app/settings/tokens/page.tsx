"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  KeyRound, Trash2, X, AlertCircle, Loader2, Save,
  Clock, ShieldCheck,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ActiveToken {
  id: string;
  user_id: string;
  user_name: string;
  client_id: string;
  client_name: string;
  scopes: string[];
  token_type: string;
  issued_at: string;
  expires_at: string;
  last_used?: string;
}

interface TokenConfig {
  access_token_lifetime_minutes: number;
  refresh_token_lifetime_days: number;
  issuer: string;
  jwks_rotation_days: number;
}

export default function TokensPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [tokens, setTokens] = useState<ActiveToken[]>([]);
  const [config, setConfig] = useState<TokenConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editConfig, setEditConfig] = useState(false);
  const [draftConfig, setDraftConfig] = useState<TokenConfig>({ access_token_lifetime_minutes: 60, refresh_token_lifetime_days: 30, issuer: "", jwks_rotation_days: 30 });
  const [confirmRevoke, setConfirmRevoke] = useState<ActiveToken | null>(null);
  const [revoking, setRevoking] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [tokRes, cfgRes] = await Promise.all([
        apiFetch<{ tokens?: ActiveToken[]; items?: ActiveToken[] }>("/api/v1/oauth/tokens/active").catch(() => ({ tokens: [] as ActiveToken[], items: [] as ActiveToken[] })),
        apiFetch<TokenConfig>("/api/v1/oauth/token-config").catch(() => null),
      ]);
      setTokens(tokRes.tokens ?? tokRes.items ?? []);
      if (cfgRes) { setConfig(cfgRes); setDraftConfig(cfgRes); }
    } catch {
      setError("Failed to load token data");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleRevoke = async (id: string) => {
    setRevoking(id);
    try {
      await apiFetch(`/api/v1/oauth/tokens/${id}`, { method: "DELETE" });
      setConfirmRevoke(null);
      setTokens((prev) => prev.filter((t) => t.id !== id));
    } catch {
      setError("Failed to revoke token");
    } finally {
      setRevoking(null);
    }
  };

  const handleSaveConfig = async () => {
    try {
      await apiFetch("/api/v1/oauth/token-config", { method: "PUT", body: JSON.stringify(draftConfig) });
      setConfig(draftConfig);
      setEditConfig(false);
    } catch {
      setError("Failed to save token config");
    }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <KeyRound className="h-6 w-6 text-indigo-600" /> Token Management
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Active session tokens, lifetime configuration, and revocation.</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* Token config */}
      {config && (
        <div className={cardCls}>
          <div className="flex items-center justify-between">
            <h3 className="flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300"><ShieldCheck className="h-4 w-4" /> Token Lifetime Configuration</h3>
            {editConfig ? (
              <div className="flex gap-2">
                <button onClick={handleSaveConfig} className="flex items-center gap-1 rounded-lg bg-green-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-green-700"><Save className="h-3.5 w-3.5" />Save</button>
                <button onClick={() => { setEditConfig(false); setDraftConfig(config); }} className="rounded-lg px-3 py-1.5 text-xs text-gray-500">Cancel</button>
              </div>
            ) : (
              <button onClick={() => setEditConfig(true)} className="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-500 hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700">Edit</button>
            )}
          </div>
          <div className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div>
              <label className="text-xs text-gray-400">Access Token (min)</label>
              {editConfig ? <input aria-label="draft Config" type="number" value={draftConfig.access_token_lifetime_minutes} onChange={(e) => setDraftConfig((p) => ({ ...p, access_token_lifetime_minutes: Number(e.target.value) }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /> : <p className="mt-1 text-lg font-bold text-indigo-600">{config.access_token_lifetime_minutes}</p>}
            </div>
            <div>
              <label className="text-xs text-gray-400">Refresh Token (days)</label>
              {editConfig ? <input aria-label="draft Config" type="number" value={draftConfig.refresh_token_lifetime_days} onChange={(e) => setDraftConfig((p) => ({ ...p, refresh_token_lifetime_days: Number(e.target.value) }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /> : <p className="mt-1 text-lg font-bold text-indigo-600">{config.refresh_token_lifetime_days}</p>}
            </div>
            <div>
              <label className="text-xs text-gray-400">Issuer</label>
              {editConfig ? <input aria-label="draft Config" value={draftConfig.issuer} onChange={(e) => setDraftConfig((p) => ({ ...p, issuer: e.target.value }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /> : <p className="mt-1 truncate text-sm font-mono text-gray-600 dark:text-gray-300">{config.issuer || "—"}</p>}
            </div>
            <div>
              <label className="text-xs text-gray-400">JWKS Rotation (days)</label>
              {editConfig ? <input aria-label="draft Config" type="number" value={draftConfig.jwks_rotation_days} onChange={(e) => setDraftConfig((p) => ({ ...p, jwks_rotation_days: Number(e.target.value) }))} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /> : <p className="mt-1 text-lg font-bold text-indigo-600">{config.jwks_rotation_days}</p>}
            </div>
          </div>
        </div>
      )}

      {/* Active tokens */}
      <div>
        <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-500"><KeyRound className="h-4 w-4" /> Active Tokens ({tokens.length})</h2>
        {loading ? (
          <div className="flex justify-center py-8"><Loader2 className="h-6 w-6 animate-spin text-indigo-600" /></div>
        ) : tokens.length === 0 ? (
          <div className={cardCls}><div className="py-12 text-center"><KeyRound className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No active tokens.</p></div></div>
        ) : (
          <div className="hidden overflow-hidden rounded-xl border border-gray-200 shadow-sm md:block dark:border-gray-700">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-800"><tr className="text-left text-xs font-semibold uppercase text-gray-500">
                <th scope="col" className="px-4 py-3">User</th><th className="px-4 py-3">Client</th><th className="px-4 py-3">Scopes</th><th className="px-4 py-3">Type</th><th className="px-4 py-3">Issued</th><th className="px-4 py-3">Expires</th><th className="px-4 py-3">Last Used</th><th className="px-4 py-3 text-right">Action</th>
              </tr></thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {tokens.map((t) => (
                  <tr key={t.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                    <td className="px-4 py-3"><div className="font-medium text-gray-800 dark:text-gray-200">{t.user_name}</div></td>
                    <td className="px-4 py-3 text-gray-500">{t.client_name}</td>
                    <td className="px-4 py-3"><div className="flex flex-wrap gap-1">{t.scopes.map((s) => <span key={s} className="rounded-full bg-indigo-100 px-2 py-0.5 text-xs text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">{s}</span>)}</div></td>
                    <td className="px-4 py-3"><span className="rounded bg-blue-100 px-1.5 py-0.5 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">{t.token_type}</span></td>
                    <td className="px-4 py-3 text-gray-500">{new Date(t.issued_at).toLocaleDateString()}</td>
                    <td className="px-4 py-3 text-gray-500"><span className="flex items-center gap-1"><Clock className="h-3 w-3" />{new Date(t.expires_at).toLocaleDateString()}</span></td>
                    <td className="px-4 py-3 text-gray-400">{t.last_used ? new Date(t.last_used).toLocaleDateString() : "—"}</td>
                    <td className="px-4 py-3 text-right"><button onClick={() => setConfirmRevoke(t)} className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20"><Trash2 className="h-4 w-4" /></button></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Mobile cards */}
      {!loading && tokens.length > 0 && (
        <div className="space-y-3 md:hidden">
          {tokens.map((t) => (
            <div key={t.id} className={cardCls}>
              <div className="flex items-center justify-between">
                <span className="font-medium text-gray-800 dark:text-gray-200">{t.user_name}</span>
                <button onClick={() => setConfirmRevoke(t)} className="text-red-500"><Trash2 className="h-4 w-4" /></button>
              </div>
              <p className="mt-1 text-xs text-gray-400">{t.client_name} · {t.token_type}</p>
              <div className="mt-1 flex flex-wrap gap-1">{t.scopes.map((s) => <span key={s} className="rounded-full bg-indigo-100 px-2 py-0.5 text-xs text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">{s}</span>)}</div>
              <p className="mt-1 text-xs text-gray-400">Expires: {new Date(t.expires_at).toLocaleString()}</p>
            </div>
          ))}
        </div>
      )}

      {/* Revoke confirm */}
      {confirmRevoke && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmRevoke(null)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3"><div className="rounded-full bg-red-100 p-2 dark:bg-red-900/30"><Trash2 className="h-5 w-5 text-red-600" /></div><div><h2 className="font-semibold text-gray-900 dark:text-white">Revoke Token?</h2><p className="text-sm text-gray-500">Token for <strong>{confirmRevoke.user_name}</strong> ({confirmRevoke.client_name}) will be invalidated immediately.</p></div></div>
            <div className="mt-5 flex justify-end gap-2"><button onClick={() => setConfirmRevoke(null)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button><button onClick={() => handleRevoke(confirmRevoke.id)} disabled={revoking === confirmRevoke.id} className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">{revoking === confirmRevoke.id ? <Loader2 className="h-4 w-4 animate-spin" /> : null}Revoke</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
