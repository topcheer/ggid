"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  AppWindow, Plus, Trash2, Copy, Check, X, AlertCircle, Loader2,
  RefreshCw, Eye, EyeOff, Shield, ExternalLink,
} from "lucide-react";

interface OAuthClient {
  id: string;
  client_id: string;
  client_name: string;
  redirect_uris: string[];
  grant_types: string[];
  scopes: string[];
  created_at: string;
  last_used?: string;
  status: string;
}

const GRANT_TYPES = ["authorization_code", "client_credentials", "refresh_token", "implicit"];
const DEFAULT_SCOPES = ["openid", "profile", "email", "offline_access"];

export default function OAuthClientsPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [clients, setClients] = useState<OAuthClient[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newSecret, setNewSecret] = useState<{ id: string; secret: string } | null>(null);
  const [copiedField, setCopiedField] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<OAuthClient | null>(null);
  const [editClient, setEditClient] = useState<OAuthClient | null>(null);
  const [showSecrets, setShowSecrets] = useState<Record<string, boolean>>({});

  // Create form
  const [form, setForm] = useState({
    client_name: "",
    redirect_uris: "",
    grant_types: ["authorization_code", "refresh_token"],
    scopes: ["openid", "profile", "email"],
  });
  const [creating, setCreating] = useState(false);

  // Edit form
  const [editUris, setEditUris] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ clients?: OAuthClient[]; items?: OAuthClient[] }>("/api/v1/oauth/clients").catch(() => null);
      setClients(data?.clients ?? data?.items ?? []);
    } catch {
      setError(t("oauth.failedLoadClients"));
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    if (!form.client_name.trim()) return;
    setCreating(true);
    try {
      const body = {
        client_name: form.client_name,
        redirect_uris: form.redirect_uris.split("\n").map((u) => u.trim()).filter(Boolean),
        grant_types: form.grant_types,
        scopes: form.scopes,
      };
      const resp = await apiFetch<{ client?: OAuthClient; client_secret?: string; client_id?: string }>("/api/v1/oauth/clients", {
        method: "POST", body: JSON.stringify(body),
      });
      if (resp.client_secret) {
        setNewSecret({ id: resp.client?.client_id ?? resp.client_id ?? "", secret: resp.client_secret });
      }
      setForm({ client_name: "", redirect_uris: "", grant_types: ["authorization_code", "refresh_token"], scopes: ["openid", "profile", "email"] });
      setShowCreate(false);
      await load();
    } catch {
      setError(t("oauth.failedCreateClient"));
    } finally {
      setCreating(false);
    }
  };

  const handleSaveEdit = async () => {
    if (!editClient) return;
    try {
      const uris = editUris.split("\n").map((u) => u.trim()).filter(Boolean);
      await apiFetch(`/api/v1/oauth/clients/${editClient.client_id}`, {
        method: "PATCH", body: JSON.stringify({ redirect_uris: uris }),
      });
      setEditClient(null);
      await load();
    } catch {
      setError(t("oauth.failedUpdateClient"));
    }
  };

  const handleDelete = async (clientId: string) => {
    try {
      await apiFetch(`/api/v1/oauth/clients/${clientId}`, { method: "DELETE" });
      setConfirmDelete(null);
      await load();
    } catch {
      setError(t("oauth.failedDeleteClient"));
    }
  };

  const toggleGrant = (g: string) => {
    setForm((p) => ({
      ...p,
      grant_types: p.grant_types.includes(g) ? p.grant_types.filter((x) => x !== g) : [...p.grant_types, g],
    }));
  };

  const toggleScope = (s: string) => {
    setForm((p) => ({
      ...p,
      scopes: p.scopes.includes(s) ? p.scopes.filter((x) => x !== s) : [...p.scopes, s],
    }));
  };

  const copy = (text: string, field: string) => {
    navigator.clipboard.writeText(text);
    setCopiedField(field);
    setTimeout(() => setCopiedField(null), 2000);
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <AppWindow className="h-6 w-6 text-indigo-600" /> {t("oauth.title")}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {t("oauth.subtitle2")}
          </p>
        </div>
        <button onClick={() => setShowCreate(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700">
          <Plus className="h-4 w-4" /> {t("oauth.registerClient")}
        </button>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {/* New secret reveal */}
      {newSecret && (
        <div className="rounded-xl border border-green-300 bg-green-50 p-5 dark:border-green-700 dark:bg-green-900/20">
          <div className="flex items-center gap-2 text-sm font-semibold text-green-800 dark:text-green-400">
            <Check className="h-5 w-5" /> {t("oauth.clientSecretGenerated")}
          </div>
          <div className="mt-3 flex items-center gap-2">
            <code className="flex-1 truncate rounded-lg bg-white px-3 py-2 font-mono text-sm dark:bg-gray-900">{newSecret.secret}</code>
            <button onClick={() => copy(newSecret.secret, "secret")} className="rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700">
              {copiedField === "secret" ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
            </button>
            <button onClick={() => setNewSecret(null)} className="text-sm text-gray-500">{t("oauth.dismiss")}</button>
          </div>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      ) : clients.length === 0 ? (
        <div className={cardCls}>
          <div className="py-12 text-center">
            <AppWindow className="mx-auto h-12 w-12 text-gray-300" />
            <p className="mt-4 text-sm text-gray-400">{t("oauth.noClients2")}</p>
          </div>
        </div>
      ) : (
        <>
          {/* Desktop table */}
          <div className="hidden overflow-hidden rounded-xl border border-gray-200 shadow-sm md:block dark:border-gray-700">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-800">
                <tr className="text-left text-xs font-semibold uppercase text-gray-500">
                  <th className="px-4 py-3">Client Name</th>
                  <th className="px-4 py-3">Client ID</th>
                  <th className="px-4 py-3">Redirect URIs</th>
                  <th className="px-4 py-3">Grants</th>
                  <th className="px-4 py-3">Created</th>
                  <th className="px-4 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {clients.map((c) => (
                  <tr key={c.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                    <td className="px-4 py-3">
                      <div className="font-medium text-gray-800 dark:text-gray-200">{c.client_name}</div>
                      <div className="text-xs text-gray-400">{c.status}</div>
                    </td>
                    <td className="px-4 py-3">
                      <button onClick={() => copy(c.client_id, `cid-${c.id}`)} className="flex items-center gap-1 font-mono text-xs text-indigo-600 hover:underline">
                        {c.client_id.substring(0, 16)}...
                        {copiedField === `cid-${c.id}` ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
                      </button>
                    </td>
                    <td className="px-4 py-3">
                      <div className="max-w-xs truncate text-xs text-gray-500" title={c.redirect_uris.join(", ")}>
                        {c.redirect_uris.length} URI{c.redirect_uris.length !== 1 ? "s" : ""}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {c.grant_types.slice(0, 2).map((g) => (
                          <span key={g} className="rounded-full bg-blue-100 px-2 py-0.5 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">{g.replace("_", " ")}</span>
                        ))}
                        {c.grant_types.length > 2 && <span className="text-xs text-gray-400">+{c.grant_types.length - 2}</span>}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-gray-500">{new Date(c.created_at).toLocaleDateString()}</td>
                    <td className="px-4 py-3">
                      <div className="flex justify-end gap-1">
                        <button onClick={() => { setEditClient(c); setEditUris(c.redirect_uris.join("\n")); }} className="rounded-lg p-1.5 text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">
                          <ExternalLink className="h-4 w-4" />
                        </button>
                        <button onClick={() => setConfirmDelete(c)} className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20">
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Mobile cards */}
          <div className="space-y-3 md:hidden">
            {clients.map((c) => (
              <div key={c.id} className={cardCls}>
                <div className="flex items-center justify-between">
                  <span className="font-medium text-gray-800 dark:text-gray-200">{c.client_name}</span>
                  <div className="flex gap-1">
                    <button onClick={() => { setEditClient(c); setEditUris(c.redirect_uris.join("\n")); }} className="p-1 text-gray-400"><ExternalLink className="h-4 w-4" /></button>
                    <button onClick={() => setConfirmDelete(c)} aria-label="Delete client" className="p-1 text-red-500"><Trash2 className="h-4 w-4" /></button>
                  </div>
                </div>
                <p className="mt-1 font-mono text-xs text-indigo-600">{c.client_id.substring(0, 20)}...</p>
                <div className="mt-2 flex flex-wrap gap-1">
                  {c.grant_types.map((g) => (
                    <span key={g} className="rounded-full bg-blue-100 px-2 py-0.5 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">{g.replace("_", " ")}</span>
                  ))}
                </div>
                <p className="mt-2 text-xs text-gray-400">{c.redirect_uris.length} redirect URI{c.redirect_uris.length !== 1 ? "s" : ""}</p>
              </div>
            ))}
          </div>
        </>
      )}

      {/* Create modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowCreate(false)}>
          <div className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">{t("oauth.registerTitle")}</h2>
              <button onClick={() => setShowCreate(false)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button>
            </div>
            <div className="mt-4 space-y-4">
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t("oauth.clientName")}</label>
                <input value={form.client_name} onChange={(e) => setForm((p) => ({ ...p, client_name: e.target.value }))} placeholder="e.g. My Web App" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t("oauth.redirectUrisHint")}</label>
                <textarea value={form.redirect_uris} onChange={(e) => setForm((p) => ({ ...p, redirect_uris: e.target.value }))} placeholder={"https://app.example.com/callback\nhttps://app.example.com/auth/callback"} rows={3} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t("oauth.grantTypes")}</label>
                <div className="mt-2 flex flex-wrap gap-2">
                  {GRANT_TYPES.map((g) => (
                    <button key={g} onClick={() => toggleGrant(g)} className={`rounded-lg border px-3 py-1.5 text-xs font-medium ${form.grant_types.includes(g) ? "border-indigo-500 bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400" : "border-gray-300 text-gray-500 dark:border-gray-600"}`}>
                      {g.replace("_", " ")}
                    </button>
                  ))}
                </div>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{t("common.scopes")}</label>
                <div className="mt-2 flex flex-wrap gap-2">
                  {DEFAULT_SCOPES.map((s) => (
                    <button key={s} onClick={() => toggleScope(s)} className={`rounded-lg border px-3 py-1.5 text-xs font-medium ${form.scopes.includes(s) ? "border-indigo-500 bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400" : "border-gray-300 text-gray-500 dark:border-gray-600"}`}>
                      {s}
                    </button>
                  ))}
                </div>
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => setShowCreate(false)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">{t("common.cancel")}</button>
              <button onClick={handleCreate} disabled={!form.client_name.trim() || creating} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
                {creating && <Loader2 className="h-4 w-4 animate-spin" />} {t("oauth.register")}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Edit redirect URIs modal */}
      {editClient && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setEditClient(null)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">{t("oauth.editRedirectUris")}</h2>
              <button onClick={() => setEditClient(null)} aria-label="Close"><X className="h-5 w-5 text-gray-400" /></button>
            </div>
            <p className="mt-1 text-sm text-gray-500">{editClient.client_name}</p>
            <textarea value={editUris} onChange={(e) => setEditUris(e.target.value)} rows={5} className="mt-3 w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setEditClient(null)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">{t("common.cancel")}</button>
              <button onClick={handleSaveEdit} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"><RefreshCw className="h-4 w-4" /> {t("common.save")}</button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirmation */}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setConfirmDelete(null)}>
          <div role="dialog" aria-modal="true" className="w-full max-w-sm rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center gap-3">
              <div className="rounded-full bg-red-100 p-2 dark:bg-red-900/30"><Trash2 className="h-5 w-5 text-red-600" /></div>
              <div>
                <h2 className="font-semibold text-gray-900 dark:text-white">{t("oauth.deleteConfirmTitle")}</h2>
                <p className="text-sm text-gray-500"><strong>{confirmDelete.client_name}</strong> {t("oauth.deleteWarning")}</p>
              </div>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button onClick={() => setConfirmDelete(null)} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">{t("common.cancel")}</button>
              <button onClick={() => handleDelete(confirmDelete.client_id)} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">{t("common.delete")}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
