"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import { Webhook, Plus, X, Trash2, Send, CheckCircle2, Clock } from "lucide-react";

interface WebhookEntry {
  id: string;
  url: string;
  events: string[];
  secret: string;
  active: boolean;
  created_at: string;
}

export default function WebhooksPage() {
  const { apiFetch } = useApi();
  const [webhooks, setWebhooks] = useState<WebhookEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [testing, setTesting] = useState<string | null>(null);
  const [form, setForm] = useState({ url: "", events: "user.login,user.register", secret: "" });

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<{ webhooks?: WebhookEntry[] }>("/api/v1/webhooks");
      setWebhooks(data.webhooks || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleCreate = async () => {
    if (!form.url) return;
    try {
      await apiFetch("/api/v1/webhooks", {
        method: "POST",
        body: JSON.stringify({
          url: form.url,
          events: form.events.split(",").map((s) => s.trim()).filter(Boolean),
          secret: form.secret || undefined,
        }),
      });
      setShowCreate(false);
      setForm({ url: "", events: "user.login,user.register", secret: "" });
      setMsg("Webhook registered");
      loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create");
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Delete this webhook?")) return;
    try {
      await apiFetch(`/api/v1/webhooks/${id}`, { method: "DELETE" });
      setMsg("Webhook deleted");
      loadData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete");
    }
  };

  const handleTest = async (id: string) => {
    setTesting(id);
    try {
      const result = await apiFetch<{ delivered?: boolean; error?: string }>(
        `/api/v1/webhooks/${id}/test`,
        { method: "POST" },
      );
      setMsg(result.delivered ? "Test delivery sent" : `Delivery failed: ${result.error || "unknown"}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Test failed");
    } finally {
      setTesting(null);
    }
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
        <h1 className="text-2xl font-bold">Webhooks</h1>
        <button
          onClick={() => { setShowCreate(!showCreate); setError(null); }}
          className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
        >
          <Plus className="h-4 w-4" /> Add Webhook
        </button>
      </div>

      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">{msg}</div>
      )}
      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div>
      )}

      {showCreate && (
        <div className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
          <div className="mb-4 flex items-center justify-between">
            <h3 className="text-sm font-semibold">New Webhook</h3>
            <button onClick={() => setShowCreate(false)}>
              <X className="h-4 w-4 text-gray-400" />
            </button>
          </div>
          <div className="grid gap-4">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Callback URL *</label>
              <input
                value={form.url}
                onChange={(e) => setForm({ ...form, url: e.target.value })}
                placeholder="https://example.com/webhooks/ggid"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono"
              />
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Events (comma-separated)</label>
                <input
                  value={form.events}
                  onChange={(e) => setForm({ ...form, events: e.target.value })}
                  placeholder="user.login,user.register,*"
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
                />
                <p className="mt-1 text-xs text-gray-400">Use * for all events</p>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Signing Secret (optional)</label>
                <input
                  value={form.secret}
                  onChange={(e) => setForm({ ...form, secret: e.target.value })}
                  placeholder="Auto-generated if blank"
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono"
                />
              </div>
            </div>
          </div>
          <button
            onClick={handleCreate}
            disabled={!form.url}
            className="mt-4 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            Register Webhook
          </button>
        </div>
      )}

      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : webhooks.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm">
          <Webhook className="mx-auto mb-4 h-12 w-12 text-gray-300" />
          <p className="text-gray-500">No webhooks registered</p>
        </div>
      ) : (
        <div className="space-y-3">
          {webhooks.map((wh) => (
            <div key={wh.id} className="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${wh.active ? "bg-green-100" : "bg-gray-100"}`}>
                    <Webhook className={`h-5 w-5 ${wh.active ? "text-green-600" : "text-gray-400"}`} />
                  </div>
                  <div>
                    <p className="font-mono text-sm font-medium">{wh.url}</p>
                    <p className="text-xs text-gray-400">ID: {wh.id.slice(0, 12)}</p>
                  </div>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => handleTest(wh.id)}
                    disabled={testing === wh.id}
                    className="flex items-center gap-1 rounded-lg border border-gray-300 px-2 py-1 text-xs font-medium hover:bg-gray-50 disabled:opacity-50"
                  >
                    <Send className="h-3.5 w-3.5" />
                    {testing === wh.id ? "Sending..." : "Test"}
                  </button>
                  <button onClick={() => handleDelete(wh.id)} className="text-gray-400 hover:text-red-500">
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
              <div className="mt-3 flex flex-wrap items-center gap-2">
                <span className={`flex items-center gap-1 rounded-full px-2 py-0.5 text-xs ${wh.active ? "bg-green-50 text-green-700" : "bg-gray-50 text-gray-500"}`}>
                  {wh.active ? <CheckCircle2 className="h-3 w-3" /> : <Clock className="h-3 w-3" />}
                  {wh.active ? "Active" : "Inactive"}
                </span>
                {wh.events?.map((event, i) => (
                  <span key={i} className="rounded-full bg-blue-50 px-2 py-0.5 text-xs text-blue-700">
                    {event}
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
