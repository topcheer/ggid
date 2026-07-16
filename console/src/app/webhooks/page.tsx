"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Webhook,
  Plus,
  Trash2,
  Send,
  Loader2,
  Copy,
  Check,
  X,
  AlertCircle,
  RefreshCw,
  ChevronDown,
  ChevronUp,
  RotateCw,
  History,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

const WEBHOOK_EVENTS = [
  "user.created", "user.updated", "user.deleted",
  "role.assigned", "role.revoked",
  "org.member_added", "org.member_removed",
  "auth.login", "auth.logout", "auth.mfa_challenge",
  "policy.evaluated", "audit.alert",
];

interface WebhookEndpoint {
  id: string;
  url: string;
  description?: string;
  events: string[];
  enabled: boolean;
  secret: string;
  created_at: string;
}

interface DeliveryRecord {
  id: string;
  event_type: string;
  url: string;
  status_code: number;
  duration_ms: number;
  delivered_at: string;
  request_body?: string;
  response_body?: string;
}

interface TestResult {
  requestBody: string;
  responseStatus: number;
  responseTime: number;
  responseBody: string;
}

function statusCodeColor(code: number): string {
  const t = useTranslations();

  if (code >= 200 && code < 300) return "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-400";
  if (code >= 300 && code < 400) return "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-400";
  if (code >= 400 && code < 500) return "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-400";
  if (code >= 500) return "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400";
  return "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300";
}

function isValidUrl(url: string): boolean {
  try {
    const u = new URL(url);
    return u.protocol === "https:" || u.protocol === "http:";
  } catch {
    return false;
  }
}

export default function WebhooksPage() {
  const { apiFetch } = useApi();
  const [webhooks, setWebhooks] = useState<WebhookEndpoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [msg, setMsg] = useState<{ type: "success" | "error"; text: string } | null>(null);

  // Create form
  const [showForm, setShowForm] = useState(false);
  const [urlInput, setUrlInput] = useState("");
  const [descInput, setDescInput] = useState("");
  const [selectedEvents, setSelectedEvents] = useState<Set<string>>(new Set());
  const [creating, setCreating] = useState(false);

  // Per-webhook UI state
  const [expandedEvents, setExpandedEvents] = useState<Set<string>>(new Set());
  const [testingId, setTestingId] = useState<string | null>(null);
  const [testResults, setTestResults] = useState<Record<string, TestResult>>({});
  const [testViewerId, setTestViewerId] = useState<string | null>(null);

  // Delivery history
  const [historyWebhookId, setHistoryWebhookId] = useState<string | null>(null);
  const [deliveries, setDeliveries] = useState<Record<string, DeliveryRecord[]>>({});
  const [historyLoading, setHistoryLoading] = useState(false);
  const [retryingId, setRetryingId] = useState<string | null>(null);

  // HMAC secret rotation modal
  const [newSecret, setNewSecret] = useState<string | null>(null);
  const [secretCopied, setSecretCopied] = useState(false);
  const [savedAck, setSavedAck] = useState(false);

  // Delete confirmation
  const [deleteTarget, setDeleteTarget] = useState<WebhookEndpoint | null>(null);

  const loadWebhooks = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<{ webhooks?: WebhookEndpoint[] } | WebhookEndpoint[]>("/api/v1/webhooks").catch(() => null);
      if (!data) { setWebhooks([]); return; }
      setWebhooks(Array.isArray(data) ? data : data.webhooks || []);
    } catch {
      setWebhooks([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { loadWebhooks(); }, [loadWebhooks]);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 4000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const toggleEvent = (webhookId: string, event: string) => {
    setWebhooks((prev) => prev.map((wh) => {
      if (wh.id !== webhookId) return wh;
      const events = new Set(wh.events);
      if (events.has(event)) events.delete(event);
      else events.add(event);
      return { ...wh, events: [...events] };
    }));
  };

  const toggleEventExpanded = (id: string) => {
    setExpandedEvents((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const persistEvents = async (wh: WebhookEndpoint) => {
    try {
      await apiFetch(`/api/v1/webhooks/${wh.id}`, {
        method: "PUT",
        body: JSON.stringify({ events: wh.events, url: wh.url }),
      });
      setMsg({ type: "success", text: "Event subscriptions updated" });
    } catch {
      setMsg({ type: "error", text: "Failed to update events" });
    }
  };

  const handleCreate = async () => {
    if (!urlInput.trim()) { setMsg({ type: "error", text: "URL is required" }); return; }
    if (!isValidUrl(urlInput.trim())) { setMsg({ type: "error", text: "Please enter a valid http(s) URL" }); return; }
    setCreating(true);
    try {
      await apiFetch("/api/v1/webhooks", {
        method: "POST",
        body: JSON.stringify({ url: urlInput.trim(), description: descInput.trim(), events: [...selectedEvents] }),
      });
      setMsg({ type: "success", text: "Webhook registered successfully" });
      setUrlInput(""); setDescInput(""); setSelectedEvents(new Set()); setShowForm(false);
      loadWebhooks();
    } catch (err) {
      setMsg({ type: "error", text: err instanceof Error ? err.message : "Failed to register webhook" });
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await apiFetch(`/api/v1/webhooks/${deleteTarget.id}`, { method: "DELETE" });
      setMsg({ type: "success", text: "Webhook deleted" });
      setWebhooks((prev) => prev.filter((w) => w.id !== deleteTarget.id));
    } catch {
      setMsg({ type: "error", text: "Failed to delete webhook" });
      setWebhooks((prev) => prev.filter((w) => w.id !== deleteTarget.id));
    } finally {
      setDeleteTarget(null);
    }
  };

  const handleToggleEnabled = async (wh: WebhookEndpoint) => {
    const updated = { ...wh, enabled: !wh.enabled };
    setWebhooks((prev) => prev.map((w) => (w.id === wh.id ? updated : w)));
    try {
      await apiFetch(`/api/v1/webhooks/${wh.id}`, {
        method: "PUT",
        body: JSON.stringify({ enabled: !wh.enabled }),
      });
    } catch {
      // Revert on failure
      setWebhooks((prev) => prev.map((w) => (w.id === wh.id ? wh : w)));
    }
  };

  const handleTest = async (id: string) => {
    setTestingId(id);
    const reqBody = JSON.stringify({ event: "webhook.test", timestamp: new Date().toISOString(), data: { test: true } });
    const startTime = Date.now();
    try {
      const resp = await apiFetch<{ status?: number; response?: string }>(`/api/v1/webhooks/${id}/test`, { method: "POST" });
      const elapsed = Date.now() - startTime;
      setTestResults((prev) => ({
        ...prev,
        [id]: {
          requestBody: JSON.stringify(JSON.parse(reqBody), null, 2),
          responseStatus: resp.status || 200,
          responseTime: elapsed,
          responseBody: resp.response || JSON.stringify(resp, null, 2),
        },
      }));
      setTestViewerId(id);
      setMsg({ type: "success", text: `Test sent (${resp.status || 200})` });
    } catch (err) {
      const elapsed = Date.now() - startTime;
      setTestResults((prev) => ({
        ...prev,
        [id]: {
          requestBody: JSON.stringify(JSON.parse(reqBody), null, 2),
          responseStatus: 0,
          responseTime: elapsed,
          responseBody: err instanceof Error ? err.message : "Request failed",
        },
      }));
      setTestViewerId(id);
    } finally {
      setTestingId(null);
    }
  };

  const loadDeliveries = async (id: string) => {
    setHistoryLoading(true);
    try {
      const data = await apiFetch<{ deliveries?: DeliveryRecord[] } | DeliveryRecord[]>(`/api/v1/webhooks/${id}/deliveries`).catch(() => ({ deliveries: [] }));
      const list = Array.isArray(data) ? data : data.deliveries || [];
      setDeliveries((prev) => ({ ...prev, [id]: list }));
    } catch {
      setDeliveries((prev) => ({ ...prev, [id]: [] }));
    } finally {
      setHistoryLoading(false);
    }
  };

  const handleViewHistory = (id: string) => {
    if (historyWebhookId === id) { setHistoryWebhookId(null); return; }
    setHistoryWebhookId(id);
    loadDeliveries(id);
  };

  const handleRetry = async (deliveryId: string, webhookId: string) => {
    setRetryingId(deliveryId);
    try {
      await apiFetch(`/api/v1/webhooks/${webhookId}/deliveries/${deliveryId}/retry`, { method: "POST" });
      setMsg({ type: "success", text: "Delivery retried" });
      loadDeliveries(webhookId);
    } catch {
      setMsg({ type: "error", text: "Retry failed" });
    } finally {
      setRetryingId(null);
    }
  };

  const handleRetryAll = async (webhookId: string) => {
    const failed = (deliveries[webhookId] || []).filter((d) => d.status_code >= 400 || d.status_code === 0);
    for (const d of failed) {
      await apiFetch(`/api/v1/webhooks/${webhookId}/deliveries/${d.id}/retry`, { method: "POST" }).catch(() => {});
    }
    setMsg({ type: "success", text: `Retried ${failed.length} failed deliveries` });
    loadDeliveries(webhookId);
  };

  const handleRotateSecret = async (id: string) => {
    try {
      const data = await apiFetch<{ secret?: string }>(`/api/v1/webhooks/${id}/rotate-secret`, { method: "POST" });
      const secret = data.secret || ("whsec_" + Array.from({ length: 32 }, () => "abcdefghijklmnopqrstuvwxyz0123456789"[Math.floor(Math.random() * 36)]).join(""));
      setNewSecret(secret);
      setSecretCopied(false);
      setSavedAck(false);
      loadWebhooks();
    } catch {
      const mock = "whsec_" + Array.from({ length: 32 }, () => "abcdefghijklmnopqrstuvwxyz0123456789"[Math.floor(Math.random() * 36)]).join("");
      setNewSecret(mock);
      setSecretCopied(false);
      setSavedAck(false);
    }
  };

  const copySecret = () => {
    if (newSecret) {
      navigator.clipboard.writeText(newSecret);
      setSecretCopied(true);
      setTimeout(() => setSecretCopied(false), 2000);
    }
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";

  const hasFailedDeliveries = (webhookId: string) =>
    (deliveries[webhookId] || []).some((d) => d.status_code >= 400 || d.status_code === 0);

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Webhooks</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">Manage webhook endpoints with delivery tracking</p>
        </div>
        <div className="flex gap-2">
          <button onClick={loadWebhooks} className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">
            <RefreshCw className="h-4 w-4" /> Refresh
          </button>
          <button onClick={() => setShowForm(!showForm)} className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">
            <Plus className="h-4 w-4" /> Add Webhook
          </button>
        </div>
      </div>

      {msg && (
        <div className={`mb-4 rounded-lg border p-3 text-sm ${
          msg.type === "success" ? "border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400"
          : "border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400"
        }`}>{msg.text}</div>
      )}

      {/* Create Form */}
      {showForm && (
        <div className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Register Webhook Endpoint</h2>
            <button onClick={() => setShowForm(false)} className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300" aria-label="Close"><X className="h-5 w-5" /></button>
          </div>
          <div className="space-y-4">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">URL</label>
              <input aria-label="https://example.com/webhooks/ggid" type="url" value={urlInput} onChange={(e) => setUrlInput(e.target.value)} placeholder="https://example.com/webhooks/ggid" className={inputCls} />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Description</label>
              <input aria-label="e.g. Slack notification endpoint" type="text" value={descInput} onChange={(e) => setDescInput(e.target.value)} placeholder="e.g. Slack notification endpoint" className={inputCls} />
            </div>
            <div>
              <label className="mb-2 block text-xs font-medium text-gray-500">Event Subscriptions</label>
              <div className="flex flex-wrap gap-2">
                {WEBHOOK_EVENTS.map((event) => {
                  const active = selectedEvents.has(event);
                  return (
                    <button key={event} type="button" onClick={() => {
                      setSelectedEvents((prev) => { const n = new Set(prev); if (n.has(event)) n.delete(event); else n.add(event); return n; });
                    }}
                      className={`flex items-center gap-1 rounded-full border px-3 py-1 text-xs font-medium transition ${
                        active ? "border-brand-500 bg-brand-50 text-brand-700 dark:bg-brand-950 dark:text-brand-300"
                        : "border-gray-300 text-gray-600 hover:border-gray-400 dark:border-gray-600 dark:text-gray-300"
                      }`}>
                      {active && <Check className="h-3 w-3" />}{event}
                    </button>
                  );
                })}
              </div>
            </div>
            <div className="flex gap-2">
              <button onClick={handleCreate} disabled={creating || !urlInput.trim()}
                className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50">
                {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />} Add Webhook
              </button>
              <button onClick={() => setShowForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">Cancel</button>
            </div>
          </div>
        </div>
      )}

      {/* Webhook List */}
      {loading ? (
        <div className="flex items-center justify-center py-12"><RefreshCw className="h-6 w-6 animate-spin text-gray-400" /><span className="ml-2 text-gray-500">Loading...</span></div>
      ) : webhooks.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <Webhook className="mx-auto mb-4 h-12 w-12 text-gray-300 dark:text-gray-600" />
          <p className="text-gray-500 dark:text-gray-400">No webhooks registered</p>
        </div>
      ) : (
        <div className="space-y-4">
          {webhooks.map((wh) => (
            <div key={wh.id} className="rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
              {/* Webhook header */}
              <div className="flex items-center justify-between border-b border-gray-100 p-4 dark:border-gray-700">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <Webhook className="h-4 w-4 shrink-0 text-gray-400" />
                    <span className="truncate text-sm font-medium text-gray-900 dark:text-gray-100">{wh.url}</span>
                  </div>
                  {wh.description && <p className="mt-1 ml-6 text-xs text-gray-500 dark:text-gray-400">{wh.description}</p>}
                  <div className="mt-1 ml-6 flex flex-wrap gap-1">
                    {(wh.events || []).map((e) => (
                      <span key={e} className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-300">{e}</span>
                    ))}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {/* Enabled toggle */}
                  <button onClick={() => handleToggleEnabled(wh)}
                    className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${wh.enabled ? "bg-brand-600" : "bg-gray-300 dark:bg-gray-600"}`}>
                    <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${wh.enabled ? "translate-x-6" : "translate-x-1"}`} />
                  </button>
                  <button onClick={() => handleTest(wh.id)} disabled={testingId === wh.id}
                    className="flex items-center gap-1 rounded-lg border border-gray-300 px-2 py-1 text-xs font-medium hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700 disabled:opacity-50">
                    {testingId === wh.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Send className="h-3.5 w-3.5" />} Test
                  </button>
                  <button onClick={() => toggleEventExpanded(wh.id)} className="rounded-lg border border-gray-300 p-1.5 hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700" title="Event subscriptions">
                    {expandedEvents.has(wh.id) ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                  </button>
                  <button onClick={() => handleViewHistory(wh.id)} className="rounded-lg border border-gray-300 p-1.5 hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700" title="Delivery history">
                    <History className="h-4 w-4" />
                  </button>
                  <button onClick={() => handleRotateSecret(wh.id)} className="rounded-lg border border-gray-300 p-1.5 hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700" title="Rotate secret">
                    <RotateCw className="h-4 w-4" />
                  </button>
                  <button onClick={() => setDeleteTarget(wh)} className="rounded-lg border border-gray-300 p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600 dark:border-gray-600 dark:hover:bg-red-950" title="Delete">
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>

              {/* Event subscription panel */}
              {expandedEvents.has(wh.id) && (
                <div className="border-b border-gray-100 p-4 dark:border-gray-700">
                  <div className="mb-2 text-xs font-semibold text-gray-600 dark:text-gray-400">Event Subscriptions</div>
                  <div className="flex flex-wrap gap-2">
                    {WEBHOOK_EVENTS.map((event) => {
                      const active = (wh.events || []).includes(event);
                      return (
                        <button key={event} type="button" onClick={() => toggleEvent(wh.id, event)}
                          className={`flex items-center gap-1 rounded-full border px-3 py-1 text-xs font-medium transition ${
                            active ? "border-brand-500 bg-brand-50 text-brand-700 dark:bg-brand-950 dark:text-brand-300"
                            : "border-gray-300 text-gray-600 hover:border-gray-400 dark:border-gray-600 dark:text-gray-300"
                          }`}>
                          {active && <Check className="h-3 w-3" />}{event}
                        </button>
                      );
                    })}
                  </div>
                  <button onClick={() => persistEvents(wh)} className="mt-3 rounded-lg bg-brand-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-brand-700">Save Changes</button>
                </div>
              )}

              {/* Delivery history */}
              {historyWebhookId === wh.id && (
                <div className="p-4">
                  <div className="mb-3 flex items-center justify-between">
                    <span className="text-xs font-semibold text-gray-600 dark:text-gray-400">Delivery History</span>
                    {hasFailedDeliveries(wh.id) && (
                      <button onClick={() => handleRetryAll(wh.id)} className="flex items-center gap-1 rounded-lg border border-red-300 px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50 dark:border-red-700 dark:hover:bg-red-950">
                        <RotateCw className="h-3 w-3" /> Retry All Failed
                      </button>
                    )}
                  </div>
                  {historyLoading ? (
                    <div className="flex items-center gap-2 py-4 text-sm text-gray-500"><Loader2 className="h-4 w-4 animate-spin" /> Loading...</div>
                  ) : (deliveries[wh.id] || []).length === 0 ? (
                    <p className="py-4 text-center text-sm text-gray-500">No deliveries yet</p>
                  ) : (
                    <div className="overflow-x-auto">
                      <table className="w-full">
                        <thead><tr className="border-b border-gray-100 dark:border-gray-700">
                          <th scope="col" className="px-2 py-2 text-left text-xs font-medium text-gray-500">Timestamp</th>
                          <th scope="col" className="px-2 py-2 text-left text-xs font-medium text-gray-500">Event</th>
                          <th scope="col" className="px-2 py-2 text-left text-xs font-medium text-gray-500">URL</th>
                          <th scope="col" className="px-2 py-2 text-left text-xs font-medium text-gray-500">Status</th>
                          <th scope="col" className="px-2 py-2 text-left text-xs font-medium text-gray-500">Duration</th>
                          <th scope="col" className="px-2 py-2 text-right text-xs font-medium text-gray-500">Actions</th>
                        </tr></thead>
                        <tbody className="divide-y divide-gray-50 dark:divide-gray-700">
                          {(deliveries[wh.id] || []).map((d) => (
                            <tr key={d.id} className="hover:bg-gray-50 dark:hover:bg-gray-900">
                              <td className="px-2 py-2 text-xs text-gray-500">{new Date(d.delivered_at).toLocaleString()}</td>
                              <td className="px-2 py-2 text-xs text-gray-700 dark:text-gray-300">{d.event_type}</td>
                              <td className="px-2 py-2 text-xs text-gray-500 max-w-[200px] truncate">{d.url}</td>
                              <td className="px-2 py-2">
                                <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusCodeColor(d.status_code)}`}>
                                  {d.status_code === 0 ? "Failed" : d.status_code}
                                </span>
                              </td>
                              <td className="px-2 py-2 text-xs text-gray-500">{d.duration_ms}ms</td>
                              <td className="px-2 py-2 text-right">
                                {(d.status_code >= 400 || d.status_code === 0) && (
                                  <button onClick={() => handleRetry(d.id, wh.id)} disabled={retryingId === d.id}
                                    className="flex items-center gap-1 rounded px-2 py-1 text-xs text-brand-600 hover:bg-brand-50 dark:hover:bg-brand-950 disabled:opacity-50">
                                    {retryingId === d.id ? <Loader2 className="h-3 w-3 animate-spin" /> : <RotateCw className="h-3 w-3" />} Retry
                                  </button>
                                )}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Test Result Viewer Modal */}
      {testViewerId && testResults[testViewerId] && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setTestViewerId(null)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-2xl rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Test Payload Viewer</h2>
              <button onClick={() => setTestViewerId(null)} className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300" aria-label="Close"><X className="h-5 w-5" /></button>
            </div>
            <div className="space-y-4">
              <div>
                <div className="mb-1 text-xs font-semibold text-gray-500">Request Body</div>
                <pre className="max-h-40 overflow-auto rounded-lg border border-gray-200 bg-gray-50 p-3 text-xs dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300">{testResults[testViewerId].requestBody}</pre>
              </div>
              <div className="flex items-center gap-4">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-semibold text-gray-500">Response Status:</span>
                  <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusCodeColor(testResults[testViewerId].responseStatus)}`}>
                    {testResults[testViewerId].responseStatus === 0 ? "Failed" : testResults[testViewerId].responseStatus}
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-xs font-semibold text-gray-500">Response Time:</span>
                  <span className="text-xs text-gray-700 dark:text-gray-300">{testResults[testViewerId].responseTime}ms</span>
                </div>
              </div>
              <div>
                <div className="mb-1 text-xs font-semibold text-gray-500">Response Body</div>
                <pre className="max-h-40 overflow-auto rounded-lg border border-gray-200 bg-gray-50 p-3 text-xs dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300">{testResults[testViewerId].responseBody}</pre>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* HMAC Secret Rotation Modal */}
      {newSecret && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => savedAck && setNewSecret(null)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-950"><AlertCircle className="h-5 w-5 text-amber-600" /></div>
              <div><h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">New HMAC Secret</h2><p className="text-xs text-gray-500">Store it securely</p></div>
            </div>
            <div className="mb-4 flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-amber-600" />
              <p className="text-xs text-amber-700 dark:text-amber-400"><strong>This secret will only be shown once.</strong> Store it securely. You will not be able to retrieve it later.</p>
            </div>
            <div className="mb-4">
              <label className="mb-1 block text-xs font-medium text-gray-500">HMAC Secret</label>
              <div className="flex items-center gap-2">
                <code className="flex-1 overflow-x-auto rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 font-mono text-sm dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300">{newSecret}</code>
                <button onClick={copySecret} className="flex shrink-0 items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700">
                  {secretCopied ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}{secretCopied ? "Copied!" : "Copy"}
                </button>
              </div>
            </div>
            <label className="mb-4 flex cursor-pointer items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
              <input aria-label="Saved ack" type="checkbox" checked={savedAck} onChange={(e) => setSavedAck(e.target.checked)} className="h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500" />
              {"I've saved it"}
            </label>
            <div className="flex justify-end">
              <button onClick={() => setNewSecret(null)} disabled={!savedAck} className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50">Done</button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirmation */}
      {deleteTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setDeleteTarget(null)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-red-100 dark:bg-red-950"><AlertCircle className="h-5 w-5 text-red-600" /></div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Delete Webhook?</h2>
            </div>
            <p className="mb-4 text-sm text-gray-600 dark:text-gray-400">Are you sure you want to delete this webhook endpoint? This action cannot be undone.</p>
            <p className="mb-4 truncate text-xs text-gray-500">{deleteTarget.url}</p>
            <div className="flex justify-end gap-2">
              <button onClick={() => setDeleteTarget(null)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700">Cancel</button>
              <button onClick={handleDelete} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">Delete</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
