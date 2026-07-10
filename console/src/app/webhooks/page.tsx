"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import { Webhook, Plus, X, Trash2, Send, CheckCircle2, Clock, RotateCw, History, Shield, ChevronDown, ChevronUp, Copy } from "lucide-react";

interface WebhookEntry {
  id: string;
  url: string;
  events: string[];
  secret: string;
  active: boolean;
  created_at: string;
}

interface DeliveryRecord {
  id: string;
  event: string;
  status: number;
  delivered: boolean;
  attempts: number;
  error?: string;
  timestamp: string;
}

export default function WebhooksPage() {
  const { apiFetch } = useApi();
  const [webhooks, setWebhooks] = useState<WebhookEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [testing, setTesting] = useState<string | null>(null);
  const [retrying, setRetrying] = useState<string | null>(null);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [deliveries, setDeliveries] = useState<Record<string, DeliveryRecord[]>>({});
  const [copiedId, setCopiedId] = useState<string | null>(null);
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

  const handleRetry = async (id: string) => {
    setRetrying(id);
    try {
      await apiFetch(`/api/v1/webhooks/${id}/retry`, { method: "POST" });
      setMsg("Retry triggered");
      loadDeliveries(id);
    } catch {
      setMsg("Retry endpoint not available");
    } finally {
      setRetrying(null);
    }
  };

  const loadDeliveries = async (id: string) => {
    try {
      const data = await apiFetch<{ deliveries?: DeliveryRecord[] }>(`/api/v1/webhooks/${id}/deliveries`);
      setDeliveries(prev => ({ ...prev, [id]: data.deliveries || [] }));
    } catch {
      // Delivery history endpoint may not exist yet
      setDeliveries(prev => ({ ...prev, [id]: [] }));
    }
  };

  const toggleExpand = (id: string) => {
    if (expandedId === id) {
      setExpandedId(null);
    } else {
      setExpandedId(id);
      if (!deliveries[id]) loadDeliveries(id);
    }
  };

  const copySecret = (id: string, secret: string) => {
    navigator.clipboard?.writeText(secret);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
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
                <p className="mt-1 text-xs text-gray-400">HMAC-SHA256 signature via X-Signature header</p>
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
            <div key={wh.id} className="rounded-xl border border-gray-200 bg-white shadow-sm">
              <div className="p-4">
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
                      onClick={() => toggleExpand(wh.id)}
                      className="flex items-center gap-1 rounded-lg border border-gray-300 px-2 py-1 text-xs font-medium hover:bg-gray-50"
                    >
                      <History className="h-3.5 w-3.5" />
                      History
                      {expandedId === wh.id ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
                    </button>
                    <button
                      onClick={() => handleRetry(wh.id)}
                      disabled={retrying === wh.id}
                      className="flex items-center gap-1 rounded-lg border border-gray-300 px-2 py-1 text-xs font-medium hover:bg-gray-50 disabled:opacity-50"
                    >
                      <RotateCw className="h-3.5 w-3.5" />
                      {retrying === wh.id ? "..." : "Retry"}
                    </button>
                    <button
                      onClick={() => handleTest(wh.id)}
                      disabled={testing === wh.id}
                      className="flex items-center gap-1 rounded-lg border border-gray-300 px-2 py-1 text-xs font-medium hover:bg-gray-50 disabled:opacity-50"
                    >
                      <Send className="h-3.5 w-3.5" />
                      {testing === wh.id ? "..." : "Test"}
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
                {wh.secret && (
                  <div className="mt-2 flex items-center gap-2 rounded-lg bg-gray-50 px-3 py-1.5">
                    <Shield className="h-3.5 w-3.5 text-gray-400" />
                    <span className="text-xs text-gray-400">HMAC Secret:</span>
                    <code className="text-xs font-mono text-gray-600">{wh.secret.slice(0, 8)}...{wh.secret.slice(-4)}</code>
                    <button
                      onClick={() => copySecret(wh.id, wh.secret)}
                      className="text-gray-400 hover:text-gray-600"
                    >
                      <Copy className="h-3 w-3" />
                      {copiedId === wh.id && <span className="ml-1 text-xs text-green-600">Copied!</span>}
                    </button>
                  </div>
                )}
              </div>

              {expandedId === wh.id && (
                <div className="border-t border-gray-100 bg-gray-50 p-4">
                  <h4 className="mb-3 text-xs font-semibold text-gray-600">Delivery History</h4>
                  {deliveries[wh.id]?.length === 0 || !deliveries[wh.id] ? (
                    <p className="text-xs text-gray-400">No delivery records yet. Send a test event to see history.</p>
                  ) : (
                    <div className="space-y-2">
                      {deliveries[wh.id].map((d) => (
                        <div key={d.id} className="flex items-center gap-3 rounded-lg bg-white px-3 py-2 text-xs">
                          <span className={`flex h-6 w-6 items-center justify-center rounded-full ${d.delivered ? "bg-green-100" : "bg-red-100"}`}>
                            {d.delivered ? <CheckCircle2 className="h-3.5 w-3.5 text-green-600" /> : <X className="h-3.5 w-3.5 text-red-500" />}
                          </span>
                          <div className="flex-1">
                            <div className="flex items-center gap-2">
                              <span className="font-mono text-gray-700">{d.event}</span>
                              <span className={`rounded px-1.5 py-0.5 text-[10px] ${d.status >= 200 && d.status < 300 ? "bg-green-50 text-green-600" : "bg-red-50 text-red-600"}`}>
                                {d.status}
                              </span>
                              {d.attempts > 1 && (
                                <span className="text-[10px] text-gray-400">{d.attempts} attempts</span>
                              )}
                            </div>
                            {d.error && <p className="mt-0.5 text-[11px] text-red-400">{d.error}</p>}
                          </div>
                          <span className="text-[10px] text-gray-400">{new Date(d.timestamp).toLocaleString()}</span>
                          {!d.delivered && (
                            <button
                              onClick={() => handleRetry(wh.id)}
                              className="flex items-center gap-1 rounded border border-gray-300 px-1.5 py-0.5 text-[10px] hover:bg-gray-50"
                            >
                              <RotateCw className="h-2.5 w-2.5" /> Retry
                            </button>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
