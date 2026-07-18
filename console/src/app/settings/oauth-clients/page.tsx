"use client";

import { useState, useCallback, useEffect } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  KeyRound,
  Loader2,
  Plus,
  X,
  Trash2,
  Copy,
  Check,
  Eye,
  EyeOff,
  Pencil,
  RefreshCw,
  AlertTriangle,
} from "lucide-react";

const ALL_GRANT_TYPES = [
  "authorization_code",
  "client_credentials",
  "refresh_token",
  "implicit",
];

interface OAuthClient {
  id: string;
  client_id: string;
  client_secret?: string;
  name: string;
  type?: string;
  grant_types: string[];
  response_types?: string[];
  redirect_uris: string[];
  scopes: string[];
  created_at: string;
}

interface ClientForm {
  name: string;
  redirect_uris: string;
  grant_types: Set<string>;
  scopes: string;
}

const emptyForm: ClientForm = {
  name: "",
  redirect_uris: "",
  grant_types: new Set(["authorization_code", "refresh_token"]),
  scopes: "openid,profile,email",
};

export default function OAuthClientsSettingsPage() {
  const { apiFetch } = useApi();
  const t = useTranslations();
  const [clients, setClients] = useState<OAuthClient[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [showEdit, setShowEdit] = useState(false);
  const [editClient, setEditClient] = useState<OAuthClient | null>(null);
  const [form, setForm] = useState<ClientForm>(emptyForm);
  const [newSecret, setNewSecret] = useState<{ id: string; secret: string } | null>(null);
  const [showSecret, setShowSecret] = useState(true);
  const [copied, setCopied] = useState(false);
  const [rotating, setRotating] = useState<string | null>(null);

  const loadClients = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<{ clients?: OAuthClient[]; items?: OAuthClient[] }>(
        "/api/v1/oauth/clients",
      ).catch(() => ({ clients: [], items: [] }));
      setClients(data.clients || data.items || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("settings.failedLoad"));
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadClients();
  }, [loadClients]);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const toggleGrantType = (gt: string, target: ClientForm) => {
    const next = new Set(target.grant_types);
    if (next.has(gt)) next.delete(gt);
    else next.add(gt);
    return { ...target, grant_types: next };
  };

  const handleCreate = async () => {
    setCreating(true);
    try {
      const result = await apiFetch<OAuthClient>("/api/v1/oauth/clients", {
        method: "POST",
        body: JSON.stringify({
          name: form.name,
          grant_types: [...form.grant_types],
          redirect_uris: form.redirect_uris.split("\n").map((s: any) => s.trim()).filter(Boolean),
          scopes: form.scopes.split(",").map((s: any) => s.trim()).filter(Boolean),
          response_types: ["code"],
        }),
      });
      setShowCreate(false);
      setForm(emptyForm);
      if (result.client_secret) {
        setNewSecret({ id: result.client_id, secret: result.client_secret });
        setShowSecret(true);
      }
      setMsg(t("oauth.created"));
      loadClients();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("settings.failedCreate"));
    } finally {
      setCreating(false);
    }
  };

  const handleEdit = (client: OAuthClient) => {
    setEditClient(client);
    setForm({
      name: client.name,
      redirect_uris: (client.redirect_uris || []).join("\n"),
      grant_types: new Set(client.grant_types || []),
      scopes: (client.scopes || []).join(","),
    });
    setShowEdit(true);
  };

  const handleUpdate = async () => {
    if (!editClient) return;
    try {
      await apiFetch(`/api/v1/oauth/clients/${editClient.client_id}`, {
        method: "PUT",
        body: JSON.stringify({
          name: form.name,
          grant_types: [...form.grant_types],
          redirect_uris: form.redirect_uris.split("\n").map((s: any) => s.trim()).filter(Boolean),
          scopes: form.scopes.split(",").map((s: any) => s.trim()).filter(Boolean),
        }),
      });
      setShowEdit(false);
      setEditClient(null);
      setForm(emptyForm);
      setMsg(t("oauth.updated"));
      loadClients();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("settings.failedUpdate"));
    }
  };

  const handleDelete = async (clientId: string, name: string) => {
    if (!confirm(`${t("oauth.deleteConfirm").replace("{name}", name)}`)) return;
    try {
      await apiFetch(`/api/v1/oauth/clients/${clientId}`, { method: "DELETE" });
      setMsg(t("oauth.deleted"));
      loadClients();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("settings.failedDelete"));
    }
  };

  const handleRotateSecret = async (clientId: string) => {
    if (!confirm(t("oauth.rotateConfirm"))) return;
    setRotating(clientId);
    try {
      const result = await apiFetch<{ client_secret?: string }>(
        `/api/v1/oauth/clients/${clientId}/rotate-secret`,
        { method: "POST" },
      ).catch(async () => {
        // Fallback: some APIs use PUT to regenerate
        return apiFetch<OAuthClient>(`/api/v1/oauth/clients/${clientId}`, {
          method: "PUT",
          body: JSON.stringify({ rotate_secret: true }),
        });
      });
      if (result.client_secret) {
        setNewSecret({ id: clientId, secret: result.client_secret });
        setShowSecret(true);
        setMsg(t("oauth.secretRotated"));
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : t("settings.failedRotate"));
    } finally {
      setRotating(null);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const GrantCheckboxes = ({ target, onChange }: { target: ClientForm; onChange: (f: ClientForm) => void }) => (
    <div className="flex flex-wrap gap-3">
      {ALL_GRANT_TYPES.map((gt: any) => (
        <label key={gt} className="flex items-center gap-1.5 text-sm">
          <input
            type="checkbox"
            checked={target.grant_types.has(gt)}
            onChange={() => onChange(toggleGrantType(gt, target))}
            className="rounded"
          />
          <span className="font-mono text-xs">{gt}</span>
        </label>
      ))}
    </div>
  );

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold dark:text-gray-100">{t("oauth.title")}</h1>
          <p className="mt-1 text-sm text-gray-500">{t("oauth.subtitle")}</p>
        </div>
        <button
          onClick={() => { setShowCreate(!showCreate); setError(null); setForm(emptyForm); }}
          className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
        >
          <Plus className="h-4 w-4" /> {t("oauth.registerClient")}
        </button>
      </div>

      {msg && (
        <div role="status" className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">{msg}</div>
      )}
      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div>
      )}

      {/* Secret reveal modal */}
      {newSecret && (
        <div className="mb-4 rounded-xl border-2 border-amber-400 bg-amber-50 p-5 shadow-sm">
          <div className="mb-3 flex items-center justify-between">
            <h3 className="flex items-center gap-2 text-sm font-bold text-amber-800">
              <AlertTriangle className="h-5 w-5" /> {t("oauth.secretRevealed")}
            </h3>
            <button onClick={() => setNewSecret(null)} aria-label="Close">
              <X className="h-4 w-4 text-gray-400" />
            </button>
          </div>
          <p className="mb-3 text-xs font-medium text-amber-700">
            {t("oauth.secretWarning")}
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded-lg bg-white px-3 py-2 font-mono text-sm break-all">
              {showSecret ? newSecret.secret : "••••••••••••••••••••••••••••"}
            </code>
            <button onClick={() => setShowSecret(!showSecret)} aria-label="Toggle secret visibility" className="rounded-lg border p-2" title={t("common.toggleVisibility")}>
              {showSecret ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
            <button onClick={() => copyToClipboard(newSecret.secret)} aria-label="Copy secret" className="rounded-lg border p-2" title={t("common.copy")}>
              {copied ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
            </button>
          </div>
          <div className="mt-2 text-xs text-gray-500">{t("oauth.clientId")}: <code className="font-mono">{newSecret.id}</code></div>
        </div>
      )}

      {/* Create form */}
      {showCreate && (
        <div className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-4 flex items-center justify-between">
            <h3 className="text-sm font-semibold dark:text-gray-100">{t("oauth.registerNew")}</h3>
            <button onClick={() => setShowCreate(false)} aria-label="Close" className="text-gray-400 hover:text-gray-600">
              <X className="h-5 w-5" />
            </button>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">{t("oauth.clientNameRequired")}</label>
              <input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder={t("oauth.clientNamePlaceholder")}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">{t("oauth.scopesComma")}</label>
              <input
                value={form.scopes}
                onChange={(e) => setForm({ ...form, scopes: e.target.value })}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
            </div>
            <div className="sm:col-span-2">
              <label className="mb-1 block text-xs font-medium text-gray-500">{t("oauth.redirectUrisHint")}</label>
              <textarea
                value={form.redirect_uris}
                onChange={(e) => setForm({ ...form, redirect_uris: e.target.value })}
                placeholder={"https://example.com/callback\nhttps://example.com/oauth/callback"}
                rows={4}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
            </div>
            <div className="sm:col-span-2">
              <label className="mb-1 block text-xs font-medium text-gray-500">{t("oauth.grantTypes")}</label>
              <GrantCheckboxes target={form} onChange={setForm} />
            </div>
          </div>
          <div className="mt-4 flex gap-2">
            <button
              onClick={handleCreate}
              disabled={!form.name || creating}
              aria-label="Create OAuth client"
              className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
            >
              {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : null} {t("oauth.createClient")}
            </button>
            <button
              onClick={() => { setShowCreate(false); setForm(emptyForm); }}
              className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
            >
              {t("common.cancel")}
            </button>
          </div>
        </div>
      )}

      {/* Edit modal */}
      {showEdit && editClient && (
        <div className="mb-6 rounded-xl border-2 border-brand-300 bg-white p-6 shadow-sm dark:border-brand-700 dark:bg-gray-800">
          <div className="mb-4 flex items-center justify-between">
            <h3 className="text-sm font-semibold dark:text-gray-100">
              {t("oauth.editClient")}: {editClient.name}
            </h3>
            <button onClick={() => { setShowEdit(false); setEditClient(null); }} aria-label="Close" className="text-gray-400 hover:text-gray-600">
              <X className="h-5 w-5" />
            </button>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">{t("oauth.clientName")}</label>
              <input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">{t("oauth.scopesComma")}</label>
              <input
                value={form.scopes}
                onChange={(e) => setForm({ ...form, scopes: e.target.value })}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
            </div>
            <div className="sm:col-span-2">
              <label className="mb-1 block text-xs font-medium text-gray-500">{t("oauth.redirectUrisHint")}</label>
              <textarea
                value={form.redirect_uris}
                onChange={(e) => setForm({ ...form, redirect_uris: e.target.value })}
                rows={4}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
            </div>
            <div className="sm:col-span-2">
              <label className="mb-1 block text-xs font-medium text-gray-500">{t("oauth.grantTypes")}</label>
              <GrantCheckboxes target={form} onChange={setForm} />
            </div>
          </div>
          <div className="mt-4 flex gap-2">
            <button
              onClick={handleUpdate}
              className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
             aria-label="Action">
              {t("common.saveChanges")}
            </button>
            <button
              onClick={() => { setShowEdit(false); setEditClient(null); }}
              className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
            >
              {t("common.cancel")}
            </button>
          </div>
        </div>
      )}

      {/* Clients table */}
      {loading ? (
        <p className="text-gray-500">{t("common.loading")}</p>
      ) : clients.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <KeyRound className="mx-auto mb-4 h-12 w-12 text-gray-300" />
          <p className="text-gray-500">{t("oauth.noClients")}</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <table className="w-full">
            <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-700/50">
              <tr>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("common.name")}</th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("oauth.clientId")}</th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("oauth.grantTypes")}</th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("oauth.redirectUris")}</th>
                <th scope="col" className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("common.created")}</th>
                <th scope="col" className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">{t("common.actions")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
              {clients.map((client: any) => (
                <tr key={client.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-brand-100">
                        <KeyRound className="h-4 w-4 text-brand-600" />
                      </div>
                      <span className="text-sm font-medium dark:text-gray-100">{client.name || t("common.unnamed")}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <code className="font-mono text-xs text-gray-500">{client.client_id}</code>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {(client.grant_types || []).map((gt: any) => (
                        <span key={gt} className="rounded-full bg-blue-50 px-2 py-0.5 font-mono text-xs text-blue-700">
                          {gt}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500">
                    {client.redirect_uris?.length || 0} {t("common.uris")}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500">
                    {client.created_at ? new Date(client.created_at).toLocaleDateString() : "—"}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-end gap-1">
                      <button
                        onClick={() => handleEdit(client)}
                        aria-label={"Edit " + client.name}
                        title={t("oauth.edit")}
                        className="rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-700"
                      >
                        <Pencil className="h-4 w-4" />
                      </button>
                      <button
                        onClick={() => handleRotateSecret(client.client_id)}
                        disabled={rotating === client.client_id}
                        aria-label={"Rotate secret for " + client.name}
                        title={t("oauth.rotateSecret")}
                        className="rounded p-1.5 text-gray-400 hover:bg-amber-50 hover:text-amber-600 disabled:opacity-50"
                      >
                        <RefreshCw className={`h-4 w-4 ${rotating === client.client_id ? "animate-spin" : ""}`} />
                      </button>
                      <button
                        onClick={() => handleDelete(client.client_id, client.name)}
                        aria-label={"Delete " + client.name}
                        title={t("oauth.delete")}
                        className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
