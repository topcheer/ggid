"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useApi } from "@/lib/api";
import { ArrowLeft, Trash2, Plus, X, KeyRound, Save } from "lucide-react";
import { CopyButton } from "@/components/ui/copy-button";

interface Client {
  id: string;
  client_id: string;
  name: string;
  type: string;
  grant_types: string[];
  response_types: string[];
  redirect_uris: string[];
  scopes: string[];
  created_at: string;
}

export default function OAuthClientDetailPage({ params }: { params: { id: string } }) {
  const { apiFetch } = useApi();
  const router = useRouter();
  const [client, setClient] = useState<Client | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [editingURIs, setEditingURIs] = useState(false);
  const [uriText, setUriText] = useState("");
  const [newSecret, setNewSecret] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<Client>(`/api/v1/oauth/clients/${params.id}`);
      setClient(data);
      setUriText((data.redirect_uris || []).join("\n"));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load");
    } finally {
      setLoading(false);
    }
  }, [apiFetch, params.id]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const handleSaveURIs = async () => {
    if (!client) return;
    try {
      await apiFetch(`/api/v1/oauth/clients/${client.client_id}`, {
        method: "PUT",
        body: JSON.stringify({
          redirect_uris: uriText.split("\n").map((s) => s.trim()).filter(Boolean),
        }),
      });
      setEditingURIs(false);
      setMsg("Redirect URIs updated");
      load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update");
    }
  };

  const handleRotateSecret = async () => {
    if (!client || !confirm("Rotate client secret? The old secret will stop working immediately.")) return;
    try {
      const result = await apiFetch<{ client_secret?: string }>(
        `/api/v1/oauth/clients/${client.client_id}/secret`,
        { method: "POST" },
      );
      if (result.client_secret) {
        setNewSecret(result.client_secret);
        setMsg("Secret rotated");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to rotate");
    }
  };

  const handleDelete = async () => {
    if (!client || !confirm("Delete this OAuth client? This cannot be undone.")) return;
    try {
      await apiFetch(`/api/v1/oauth/clients/${client.client_id}`, { method: "DELETE" });
      router.push("/oauth-clients");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete");
    }
  };

  if (loading) return <p className="text-gray-500">Loading...</p>;
  if (error) return (
    <div>
      <button onClick={() => router.push("/oauth-clients")} className="mb-4 flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700">
        <ArrowLeft className="h-4 w-4" /> Back
      </button>
      <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">{error}</div>
    </div>
  );
  if (!client) return null;

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button onClick={() => router.push("/oauth-clients")} className="text-gray-400 hover:text-gray-600">
            <ArrowLeft className="h-5 w-5" />
          </button>
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-brand-100">
            <KeyRound className="h-5 w-5 text-brand-600" />
          </div>
          <div>
            <h1 className="text-2xl font-bold">{client.name}</h1>
            <p className="font-mono text-xs text-gray-500">{client.client_id}</p>
          </div>
        </div>
        <div className="flex gap-2">
          <button onClick={handleRotateSecret} className="rounded-lg border border-amber-300 px-3 py-2 text-sm text-amber-700 hover:bg-amber-50">
            Rotate Secret
          </button>
          <button onClick={handleDelete} className="flex items-center gap-1 rounded-lg border border-red-300 px-3 py-2 text-sm text-red-600 hover:bg-red-50">
            <Trash2 className="h-4 w-4" /> Delete
          </button>
        </div>
      </div>

      {msg && <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">{msg}</div>}

      {newSecret && (
        <div className="mb-4 rounded-xl border border-amber-300 bg-amber-50 p-5 dark:border-amber-800 dark:bg-amber-950">
          <div className="mb-2 flex items-center justify-between">
            <h3 className="text-sm font-semibold text-amber-800 dark:text-amber-400">New Client Secret (save now!)</h3>
            <CopyButton value={newSecret} label="Copy Secret" variant="button" title="Copy client secret" />
          </div>
          <code className="block rounded-lg bg-white px-3 py-2 font-mono text-sm dark:bg-gray-900 dark:text-gray-300">{newSecret}</code>
        </div>
      )}

      <div className="space-y-4">
        {/* Client info */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h3 className="mb-4 text-sm font-semibold">Client Information</h3>
          <div className="grid gap-3 sm:grid-cols-2">
            <InfoRow label="Type" value={client.type} />
            <InfoRow label="Created" value={client.created_at ? new Date(client.created_at).toLocaleString() : "-"} />
            <InfoRow label="Grant Types" value={(client.grant_types || []).join(", ")} />
            <InfoRow label="Response Types" value={(client.response_types || []).join(", ")} />
          </div>
        </div>

        {/* Scopes */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <h3 className="mb-3 text-sm font-semibold">Scopes</h3>
          <div className="flex flex-wrap gap-2">
            {(client.scopes || []).map((scope, i) => (
              <span key={i} className="rounded-full bg-purple-50 px-3 py-1 text-xs text-purple-700">{scope}</span>
            ))}
          </div>
        </div>

        {/* Redirect URIs */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <div className="mb-3 flex items-center justify-between">
            <h3 className="text-sm font-semibold">Redirect URIs</h3>
            {!editingURIs ? (
              <button onClick={() => setEditingURIs(true)} className="text-xs text-brand-600 hover:underline">Edit</button>
            ) : (
              <div className="flex gap-2">
                <button onClick={() => setEditingURIs(false)} className="text-xs text-gray-500">Cancel</button>
                <button onClick={handleSaveURIs} className="flex items-center gap-1 text-xs text-green-600">
                  <Save className="h-3 w-3" /> Save
                </button>
              </div>
            )}
          </div>
          {editingURIs ? (
            <textarea
              value={uriText}
              onChange={(e) => setUriText(e.target.value)}
              rows={4}
              className="w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-sm"
              placeholder={"https://example.com/callback\nhttps://example.com/oauth/callback"}
            />
          ) : (
            <div className="space-y-1">
              {(client.redirect_uris || []).map((uri, i) => (
                <div key={i} className="rounded-lg bg-blue-50 px-3 py-1.5 font-mono text-sm text-blue-700">{uri}</div>
              ))}
              {(!client.redirect_uris || client.redirect_uris.length === 0) && (
                <p className="text-sm text-gray-400">No redirect URIs configured</p>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs font-medium text-gray-500">{label}</p>
      <p className="text-sm">{value || "-"}</p>
    </div>
  );
}
