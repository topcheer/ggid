"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Plus, Pencil, Trash2, RefreshCw, Copy, Eye, EyeOff, RotateCw, X, Key, Check,
} from "lucide-react";

interface OAuthClient {
  id: string;
  client_id: string;
  client_secret?: string;
  name: string;
  redirect_uris?: string[];
  grant_types?: string[];
  scopes?: string[];
  active?: boolean;
  created_at?: string;
  access_token_ttl?: number;
  refresh_token_ttl?: number;
}

const GRANT_TYPES = ["authorization_code", "implicit", "password", "client_credentials", "refresh_token", "device_code"];
const STANDARD_SCOPES = ["openid", "profile", "email", "offline_access"];

export default function OAuthClientsPage() {
  const { apiFetch } = useApi();
  const [clients, setClients] = useState<OAuthClient[]>([]);
  const [loading, setLoading] = useState(true);
  const [msg, setMsg] = useState<string | null>(null);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editingClient, setEditingClient] = useState<OAuthClient | null>(null);
  const [revealSecret, setRevealSecret] = useState<Record<string, boolean>>({});
  const [deleteTarget, setDeleteTarget] = useState<OAuthClient | null>(null);
  const [rotateTarget, setRotateTarget] = useState<OAuthClient | null>(null);

  const loadClients = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<{ clients?: OAuthClient[] } | OAuthClient[]>("/api/v1/oauth/clients").catch(() => null);
      if (!data) { setClients([]); return; }
      setClients(Array.isArray(data) ? data : data.clients || []);
    } catch {
      setClients([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { loadClients(); }, [loadClients]);
  useEffect(() => { if (msg) { const t = setTimeout(() => setMsg(null), 3000); return () => clearTimeout(t); } }, [msg]);

  const showMessage = (m: string) => setMsg(m);

  // Form state
  const [form, setForm] = useState({
    name: "",
    redirectUris: "",
    grantTypes: ["authorization_code", "refresh_token"] as string[],
    scopes: ["openid", "profile", "email"] as string[],
    customScope: "",
    accessTokenTtl: 3600,
    refreshTokenTtl: 2592000,
  });

  const openCreateForm = () => {
    setEditingClient(null);
    setForm({ name: "", redirectUris: "", grantTypes: ["authorization_code", "refresh_token"], scopes: ["openid", "profile", "email"], customScope: "", accessTokenTtl: 3600, refreshTokenTtl: 2592000 });
    setShowForm(true);
  };

  const openEditForm = (c: OAuthClient) => {
    setEditingClient(c);
    setForm({
      name: c.name,
      redirectUris: (c.redirect_uris || []).join("\n"),
      grantTypes: c.grant_types || [],
      scopes: c.scopes || [],
      customScope: "",
      accessTokenTtl: c.access_token_ttl || 3600,
      refreshTokenTtl: c.refresh_token_ttl || 2592000,
    });
    setShowForm(true);
  };

  const toggleGrant = (g: string) => {
    setForm(prev => ({ ...prev, grantTypes: prev.grantTypes.includes(g) ? prev.grantTypes.filter(x => x !== g) : [...prev.grantTypes, g] }));
  };
  const toggleScope = (s: string) => {
    setForm(prev => ({ ...prev, scopes: prev.scopes.includes(s) ? prev.scopes.filter(x => x !== s) : [...prev.scopes, s] }));
  };
  const addCustomScope = () => {
    const s = form.customScope.trim();
    if (s && !form.scopes.includes(s)) { setForm(prev => ({ ...prev, scopes: [...prev.scopes, s], customScope: "" })); }
  };

  const handleCreateOrUpdate = async () => {
    const uris = form.redirectUris.split("\n").map(u => u.trim()).filter(Boolean);
    const body = {
      name: form.name,
      redirect_uris: uris,
      grant_types: form.grantTypes,
      scopes: form.scopes,
      access_token_ttl: form.accessTokenTtl,
      refresh_token_ttl: form.refreshTokenTtl,
    };
    try {
      if (editingClient) {
        await apiFetch(`/api/v1/oauth/clients/${editingClient.client_id}`, { method: "PUT", body: JSON.stringify(body) });
        showMessage("Client updated successfully");
      } else {
        await apiFetch("/api/v1/oauth/clients", { method: "POST", body: JSON.stringify(body) });
        showMessage("Client created successfully");
      }
      setShowForm(false);
      loadClients();
    } catch {
      showMessage(editingClient ? "Failed to update client" : "Failed to create client");
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await apiFetch(`/api/v1/oauth/clients/${deleteTarget.client_id}`, { method: "DELETE" });
      setClients(prev => prev.filter(c => c.id !== deleteTarget.id));
      showMessage("Client deleted");
    } catch {
      showMessage("Failed to delete client");
    } finally {
      setDeleteTarget(null);
    }
  };

  const handleRotate = async () => {
    if (!rotateTarget) return;
    try {
      const data = await apiFetch<OAuthClient>(`/api/v1/oauth/clients/${rotateTarget.client_id}/rotate-secret`, { method: "POST" });
      setClients(prev => prev.map(c => c.id === rotateTarget.id ? { ...c, client_secret: data.client_secret } : c));
      showMessage("Secret rotated successfully");
    } catch {
      showMessage("Failed to rotate secret");
    } finally {
      setRotateTarget(null);
    }
  };

  const copyToClipboard = (text: string) => {
    if (typeof navigator !== "undefined" && navigator.clipboard) {
      navigator.clipboard.writeText(text);
      showMessage("Copied to clipboard");
    }
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const labelCls = "mb-1 block text-xs font-medium text-gray-500";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const badge = (text: string, color = "brand") => (
    <span className={`mr-1 mb-1 inline-block rounded-full px-2 py-0.5 text-xs font-medium ${color === "brand" ? "bg-brand-100 text-brand-700 dark:bg-brand-900 dark:text-brand-300" : "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400"}`}>{text}</span>
  );

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-gray-100">
            <Key className="h-7 w-7 text-brand-600" /> OAuth Client Registry
          </h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">Manage OAuth 2.0 client applications</p>
        </div>
        <div className="flex gap-2">
          <button onClick={loadClients} className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">
            <RefreshCw className="h-4 w-4" /> Refresh
          </button>
          <button onClick={openCreateForm} className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">
            <Plus className="h-4 w-4" /> Create Client
          </button>
        </div>
      </div>

      {msg && <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">{msg}</div>}

      {loading ? (
        <div className="flex items-center justify-center py-12"><RefreshCw className="h-6 w-6 animate-spin text-gray-400" /><span className="ml-2 text-gray-500">Loading clients...</span></div>
      ) : clients.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <Key className="mx-auto mb-4 h-12 w-12 text-gray-300 dark:text-gray-600" />
          <p className="text-gray-500 dark:text-gray-400">No OAuth clients registered</p>
          <button onClick={openCreateForm} className="mt-4 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">Create your first client</button>
        </div>
      ) : (
        <div className={cardCls}>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 text-left text-xs uppercase tracking-wider text-gray-500 dark:border-gray-700">
                  <th className="pb-3 pr-4">Client Name</th>
                  <th className="pb-3 pr-4">Client ID</th>
                  <th className="pb-3 pr-4">Grant Types</th>
                  <th className="pb-3 pr-4">Scopes</th>
                  <th className="pb-3 pr-4">Status</th>
                  <th className="pb-3 pr-4">Created</th>
                  <th className="pb-3"></th>
                </tr>
              </thead>
              <tbody>
                {clients.map(c => (
                  <>
                    <tr key={c.id} className="border-b border-gray-100 cursor-pointer hover:bg-gray-50 dark:border-gray-800 dark:hover:bg-gray-700/50" onClick={() => setExpandedId(expandedId === c.id ? null : c.id)}>
                      <td className="py-3 pr-4 font-medium text-gray-900 dark:text-gray-100">{c.name}</td>
                      <td className="py-3 pr-4 font-mono text-xs text-gray-500">{c.client_id ? `${c.client_id.slice(0, 12)}...` : "—"}</td>
                      <td className="py-3 pr-4">{(c.grant_types || []).slice(0, 3).map(g => badge(g, "gray"))}{(c.grant_types || []).length > 3 && <span className="text-xs text-gray-400">+{(c.grant_types || []).length - 3}</span>}</td>
                      <td className="py-3 pr-4">{(c.scopes || []).slice(0, 3).map(s => badge(s, "brand"))}{(c.scopes || []).length > 3 && <span className="text-xs text-gray-400">+{(c.scopes || []).length - 3}</span>}</td>
                      <td className="py-3 pr-4"><span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${c.active !== false ? "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-400" : "bg-gray-100 text-gray-500 dark:bg-gray-700"}`}><span className={`h-1.5 w-1.5 rounded-full ${c.active !== false ? "bg-green-500" : "bg-gray-400"}`} />{c.active !== false ? "Active" : "Inactive"}</span></td>
                      <td className="py-3 pr-4 text-xs text-gray-500">{c.created_at ? new Date(c.created_at).toLocaleDateString() : "—"}</td>
                      <td className="py-3"><div className="flex items-center gap-1" onClick={e => e.stopPropagation()}>
                        <button onClick={() => openEditForm(c)} className="rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-brand-600 dark:hover:bg-gray-700" title="Edit"><Pencil className="h-4 w-4" /></button>
                        <button onClick={() => setDeleteTarget(c)} className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950" title="Delete"><Trash2 className="h-4 w-4" /></button>
                      </div></td>
                    </tr>
                    {expandedId === c.id && (
                      <tr className="bg-gray-50 dark:bg-gray-800/50">
                        <td colSpan={7} className="p-4">
                          <div className="grid gap-4 sm:grid-cols-2">
                            <div>
                              <p className="mb-1 text-xs font-medium text-gray-500">Redirect URIs</p>
                              {(c.redirect_uris || []).length > 0 ? (
                                <ul className="space-y-1">{c.redirect_uris!.map((uri, i) => <li key={i} className="font-mono text-xs text-gray-600 dark:text-gray-400">{uri}</li>)}</ul>
                              ) : <p className="text-xs text-gray-400">None configured</p>}
                            </div>
                            <div>
                              <p className="mb-1 text-xs font-medium text-gray-500">Client Secret</p>
                              <div className="flex items-center gap-2">
                                <code className="flex-1 rounded bg-gray-200 px-2 py-1 font-mono text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-400">
                                  {c.client_secret ? (revealSecret[c.id] ? c.client_secret : "••••••••••••••••") : "—"}
                                </code>
                                {c.client_secret && <>
                                  <button onClick={() => setRevealSecret(prev => ({ ...prev, [c.id]: !prev[c.id] }))} className="rounded p-1.5 text-gray-400 hover:text-brand-600" title={revealSecret[c.id] ? "Hide" : "Reveal"}>{revealSecret[c.id] ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}</button>
                                  <button onClick={() => copyToClipboard(c.client_secret!)} className="rounded p-1.5 text-gray-400 hover:text-brand-600" title="Copy"><Copy className="h-4 w-4" /></button>
                                  <button onClick={() => setRotateTarget(c)} className="rounded p-1.5 text-gray-400 hover:text-amber-600" title="Rotate Secret"><RotateCw className="h-4 w-4" /></button>
                                </>}
                              </div>
                            </div>
                            <div><p className="mb-1 text-xs font-medium text-gray-500">Access Token TTL</p><p className="text-xs text-gray-600 dark:text-gray-400">{c.access_token_ttl || 3600}s</p></div>
                            <div><p className="mb-1 text-xs font-medium text-gray-500">Refresh Token TTL</p><p className="text-xs text-gray-600 dark:text-gray-400">{c.refresh_token_ttl || 2592000}s</p></div>
                          </div>
                        </td>
                      </tr>
                    )}
                  </>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Create/Edit Modal */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowForm(false)}>
          <div className="mx-4 max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <div className="mb-6 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{editingClient ? "Edit Client" : "Create OAuth Client"}</h2>
              <button onClick={() => setShowForm(false)} className="rounded-lg p-1 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"><X className="h-5 w-5" /></button>
            </div>

            <div className="space-y-5">
              {/* Name */}
              <div><label className={labelCls}>Client Name</label><input value={form.name} onChange={e => setForm({ ...form, name: e.target.value })} className={inputCls} placeholder="My Application" /></div>

              {/* Redirect URIs */}
              <div>
                <label className={labelCls}>Redirect URIs (one per line)</label>
                <textarea value={form.redirectUris} onChange={e => setForm({ ...form, redirectUris: e.target.value })} className={`${inputCls} min-h-[80px] font-mono text-xs`} placeholder={"https://app.example.com/callback\nhttps://staging.example.com/callback"} />
              </div>

              {/* Grant Types */}
              <div>
                <label className={labelCls}>Grant Types</label>
                <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                  {GRANT_TYPES.map(g => (
                    <label key={g} className={`flex cursor-pointer items-center gap-2 rounded-lg border p-2 text-xs transition-colors ${form.grantTypes.includes(g) ? "border-brand-400 bg-brand-50 dark:border-brand-700 dark:bg-brand-900/20" : "border-gray-200 dark:border-gray-700"}`}>
                      <input type="checkbox" checked={form.grantTypes.includes(g)} onChange={() => toggleGrant(g)} className="h-3.5 w-3.5" />
                      <span className="text-gray-700 dark:text-gray-300">{g}</span>
                    </label>
                  ))}
                </div>
              </div>

              {/* Scopes */}
              <div>
                <label className={labelCls}>Scopes</label>
                <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                  {STANDARD_SCOPES.map(s => (
                    <label key={s} className={`flex cursor-pointer items-center gap-2 rounded-lg border p-2 text-xs transition-colors ${form.scopes.includes(s) ? "border-brand-400 bg-brand-50 dark:border-brand-700 dark:bg-brand-900/20" : "border-gray-200 dark:border-gray-700"}`}>
                      <input type="checkbox" checked={form.scopes.includes(s)} onChange={() => toggleScope(s)} className="h-3.5 w-3.5" />
                      <span className="text-gray-700 dark:text-gray-300">{s}</span>
                    </label>
                  ))}
                </div>
                <div className="mt-2 flex gap-2">
                  <input value={form.customScope} onChange={e => setForm({ ...form, customScope: e.target.value })} onKeyDown={e => { if (e.key === "Enter") { e.preventDefault(); addCustomScope(); } }} className={`${inputCls} flex-1`} placeholder="custom:scope" />
                  <button onClick={addCustomScope} className="rounded-lg border border-brand-600 px-3 py-2 text-sm text-brand-600 hover:bg-brand-50 dark:hover:bg-brand-900/30">Add</button>
                </div>
                {form.scopes.filter(s => !STANDARD_SCOPES.includes(s)).length > 0 && (
                  <div className="mt-2 flex flex-wrap gap-1">
                    {form.scopes.filter(s => !STANDARD_SCOPES.includes(s)).map(s => (
                      <span key={s} className="flex items-center gap-1 rounded-full bg-brand-100 px-2 py-0.5 text-xs text-brand-700 dark:bg-brand-900 dark:text-brand-300">{s}<button onClick={() => toggleScope(s)}><X className="h-3 w-3" /></button></span>
                    ))}
                  </div>
                )}
              </div>

              {/* Token Lifetimes */}
              <div className="grid gap-4 sm:grid-cols-2">
                <div><label className={labelCls}>Access Token TTL (seconds)</label><input type="number" min={60} value={form.accessTokenTtl} onChange={e => setForm({ ...form, accessTokenTtl: Number(e.target.value) || 3600 })} className={inputCls} /></div>
                <div><label className={labelCls}>Refresh Token TTL (seconds)</label><input type="number" min={60} value={form.refreshTokenTtl} onChange={e => setForm({ ...form, refreshTokenTtl: Number(e.target.value) || 2592000 })} className={inputCls} /></div>
              </div>
            </div>

            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => setShowForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={handleCreateOrUpdate} className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">
                <Check className="h-4 w-4" /> {editingClient ? "Update" : "Create"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirmation */}
      {deleteTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setDeleteTarget(null)}>
          <div className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <div className="mb-4 flex items-center gap-3"><div className="flex h-10 w-10 items-center justify-center rounded-full bg-red-100 dark:bg-red-950"><Trash2 className="h-5 w-5 text-red-600" /></div><h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Delete Client?</h2></div>
            <p className="mb-6 text-sm text-gray-600 dark:text-gray-400">Are you sure you want to delete <span className="font-semibold">{deleteTarget.name}</span>? This action cannot be undone.</p>
            <div className="flex justify-end gap-2">
              <button onClick={() => setDeleteTarget(null)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={handleDelete} className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"><Trash2 className="h-4 w-4" /> Delete</button>
            </div>
          </div>
        </div>
      )}

      {/* Rotate Confirmation */}
      {rotateTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setRotateTarget(null)}>
          <div className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <div className="mb-4 flex items-center gap-3"><div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-950"><RotateCw className="h-5 w-5 text-amber-600" /></div><h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Rotate Secret?</h2></div>
            <p className="mb-6 text-sm text-gray-600 dark:text-gray-400">This will generate a new secret for <span className="font-semibold">{rotateTarget.name}</span>. The old secret will stop working immediately.</p>
            <div className="flex justify-end gap-2">
              <button onClick={() => setRotateTarget(null)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={handleRotate} className="flex items-center gap-2 rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white hover:bg-amber-700"><RotateCw className="h-4 w-4" /> Rotate</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
