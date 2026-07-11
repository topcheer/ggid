"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import { KeyRound, Plus, X, Trash2, Copy, Check, Eye, EyeOff } from "lucide-react";

interface OAuthClient {
  id: string;
  client_id: string;
  client_secret?: string;
  name: string;
  type: string;
  grant_types: string[];
  response_types: string[];
  redirect_uris: string[];
  scopes: string[];
  created_at: string;
}

export default function OAuthClientsPage() {
  const { apiFetch } = useApi();
  const [clients, setClients] = useState<OAuthClient[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newSecret, setNewSecret] = useState<{ id: string; secret: string } | null>(null);
  const [showSecret, setShowSecret] = useState(true);
  const [copied, setCopied] = useState(false);

  const [form, setForm] = useState({
    name: "",
    type: "confidential",
    grant_types: "authorization_code,refresh_token",
    redirect_uris: "",
    scopes: "openid,profile,email",
  });

  const loadClients = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<{ clients?: OAuthClient[] }>("/api/v1/oauth/clients");
      setClients(data.clients || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadClients();
  }, [loadClients]);

  const handleCreate = async () => {
    try {
      const result = await apiFetch<OAuthClient>("/api/v1/oauth/clients", {
        method: "POST",
        body: JSON.stringify({
          name: form.name,
          type: form.type,
          grant_types: form.grant_types.split(",").map((s) => s.trim()).filter(Boolean),
          redirect_uris: form.redirect_uris.split("\n").map((s) => s.trim()).filter(Boolean),
          scopes: form.scopes.split(",").map((s) => s.trim()).filter(Boolean),
          response_types: ["code"],
        }),
      });
      setShowCreate(false);
      setForm({ name: "", type: "confidential", grant_types: "authorization_code,refresh_token", redirect_uris: "", scopes: "openid,profile,email" });
      if (result.client_secret) {
        setNewSecret({ id: result.client_id, secret: result.client_secret });
        setShowSecret(true);
      }
      setMsg("Client created");
      loadClients();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create");
    }
  };

  const handleDelete = async (clientId: string) => {
    if (!confirm("Delete this OAuth client?")) return;
    try {
      await apiFetch(`/api/v1/oauth/clients/${clientId}`, { method: "DELETE" });
      setMsg("Client deleted");
      loadClients();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete");
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">OAuth Clients</h1>
        <button
          onClick={() => { setShowCreate(!showCreate); setError(null); }}
          className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
        >
          <Plus className="h-4 w-4" /> Register Client
        </button>
      </div>

      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">{msg}</div>
      )}
      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div>
      )}

      {/* Secret reveal modal */}
      {newSecret && (
        <div className="mb-4 rounded-xl border border-amber-300 bg-amber-50 p-5 shadow-sm">
          <div className="mb-2 flex items-center justify-between">
            <h3 className="flex items-center gap-2 text-sm font-semibold text-amber-800">
              <KeyRound className="h-4 w-4" /> Client Secret (show only once!)
            </h3>
            <button onClick={() => setNewSecret(null)} aria-label="Close">
              <X className="h-4 w-4 text-gray-400" />
            </button>
          </div>
          <p className="mb-3 text-xs text-amber-700">
            Copy this secret now. For security, it will not be shown again.
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded-lg bg-white px-3 py-2 font-mono text-sm">
              {showSecret ? newSecret.secret : "••••••••••••••••••••••••"}
            </code>
            <button onClick={() => setShowSecret(!showSecret)} className="rounded-lg border p-2">
              {showSecret ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
            <button onClick={() => copyToClipboard(newSecret.secret)} className="rounded-lg border p-2">
              {copied ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
            </button>
          </div>
          <div className="mt-2 text-xs text-gray-500">Client ID: <code className="font-mono">{newSecret.id}</code></div>
        </div>
      )}

      {/* Create form */}
      {showCreate && (
        <div className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <div className="mb-4 flex items-center justify-between">
            <h3 className="text-sm font-semibold">New OAuth Client</h3>
            <button onClick={() => setShowCreate(false)} aria-label="Close">
              <X className="h-4 w-4 text-gray-400" />
            </button>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Name *</label>
              <input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="e.g. My Web App"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Type</label>
              <select
                value={form.type}
                onChange={(e) => setForm({ ...form, type: e.target.value })}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
              >
                <option value="confidential">Confidential (server-side)</option>
                <option value="public">Public (SPA/Mobile)</option>
              </select>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Grant Types (comma-separated)</label>
              <input
                value={form.grant_types}
                onChange={(e) => setForm({ ...form, grant_types: e.target.value })}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Scopes (comma-separated)</label>
              <input
                value={form.scopes}
                onChange={(e) => setForm({ ...form, scopes: e.target.value })}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
              />
            </div>
            <div className="sm:col-span-2">
              <label className="mb-1 block text-xs font-medium text-gray-500">Redirect URIs (one per line)</label>
              <textarea
                value={form.redirect_uris}
                onChange={(e) => setForm({ ...form, redirect_uris: e.target.value })}
                placeholder={"https://example.com/callback\nhttps://example.com/oauth/callback"}
                rows={3}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono"
              />
            </div>
          </div>
          <button
            onClick={handleCreate}
            disabled={!form.name}
            className="mt-4 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            Create Client
          </button>
        </div>
      )}

      {/* Client list */}
      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : clients.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm">
          <KeyRound className="mx-auto mb-4 h-12 w-12 text-gray-300" />
          <p className="text-gray-500">No OAuth clients registered</p>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {clients.map((client) => (
            <div key={client.id} className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
              <div className="mb-3 flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-100">
                    <KeyRound className="h-5 w-5 text-brand-600" />
                  </div>
                  <div>
                    <h3 className="font-semibold">{client.name || "Unnamed"}</h3>
                    <p className="font-mono text-xs text-gray-500">{client.client_id}</p>
                  </div>
                </div>
                <button onClick={() => handleDelete(client.client_id)} className="text-gray-400 hover:text-red-500">
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
              <div className="space-y-2 text-sm">
                <div>
                  <span className="text-xs text-gray-500">Type: </span>
                  <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs">{client.type}</span>
                </div>
                {client.redirect_uris?.length > 0 && (
                  <div>
                    <p className="text-xs text-gray-500">Redirect URIs:</p>
                    <div className="mt-1 flex flex-wrap gap-1">
                      {client.redirect_uris.map((uri, i) => (
                        <span key={i} className="rounded bg-blue-50 px-1.5 py-0.5 font-mono text-xs text-blue-700">
                          {uri}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
                {client.scopes?.length > 0 && (
                  <div>
                    <p className="text-xs text-gray-500">Scopes:</p>
                    <div className="mt-1 flex flex-wrap gap-1">
                      {client.scopes.map((scope, i) => (
                        <span key={i} className="rounded bg-purple-50 px-1.5 py-0.5 text-xs text-purple-700">
                          {scope}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
