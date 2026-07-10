"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Webhook,
  Plus,
  Trash2,
  Send,
  Loader2,
  Copy,
  Eye,
  EyeOff,
  Check,
  X,
  CheckCircle2,
  XCircle,
} from "lucide-react";

const AVAILABLE_EVENTS = [
  "user.created",
  "user.login",
  "user.logout",
  "user.deleted",
  "role.created",
  "role.deleted",
  "org.created",
  "member.added",
  "member.removed",
];

interface WebhookEndpoint {
  id: string;
  url: string;
  events: string[];
  enabled: boolean;
  secret: string;
  created_at: string;
}

function isValidUrl(url: string): boolean {
  try {
    const u = new URL(url);
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}

export default function WebhooksPage() {
  const { apiFetch } = useApi();
  const [webhooks, setWebhooks] = useState<WebhookEndpoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [msg, setMsg] = useState<{ type: "success" | "error"; text: string } | null>(null);

  // Create form state
  const [showForm, setShowForm] = useState(false);
  const [urlInput, setUrlInput] = useState("");
  const [urlError, setUrlError] = useState<string | null>(null);
  const [selectedEvents, setSelectedEvents] = useState<Set<string>>(new Set());
  const [enabledToggle, setEnabledToggle] = useState(true);
  const [creating, setCreating] = useState(false);

  // Per-webhook UI state
  const [revealedSecrets, setRevealedSecrets] = useState<Set<string>>(new Set());
  const [copiedSecret, setCopiedSecret] = useState<string | null>(null);
  const [testingId, setTestingId] = useState<string | null>(null);
  const [testResults, setTestResults] = useState<Record<string, { ok: boolean; status: number }>>({});

  const fetchWebhooks = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetch<{ webhooks?: WebhookEndpoint[] } | WebhookEndpoint[]>(
        "/api/v1/webhooks",
      );
      const list = Array.isArray(data) ? data : data.webhooks || [];
      setWebhooks(list);
    } catch {
      setWebhooks([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    fetchWebhooks();
  }, [fetchWebhooks]);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 4000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const toggleEvent = (event: string) => {
    setSelectedEvents((prev) => {
      const next = new Set(prev);
      if (next.has(event)) next.delete(event);
      else next.add(event);
      return next;
    });
  };

  const handleCreate = async () => {
    setUrlError(null);
    if (!urlInput.trim()) {
      setUrlError("URL is required");
      return;
    }
    if (!isValidUrl(urlInput.trim())) {
      setUrlError("Please enter a valid http(s) URL");
      return;
    }
    if (selectedEvents.size === 0) {
      setUrlError("Select at least one event");
      return;
    }

    setCreating(true);
    try {
      await apiFetch("/api/v1/webhooks", {
        method: "POST",
        body: JSON.stringify({
          url: urlInput.trim(),
          events: [...selectedEvents],
          enabled: enabledToggle,
        }),
      });
      setMsg({ type: "success", text: "Webhook created successfully" });
      setUrlInput("");
      setSelectedEvents(new Set());
      setEnabledToggle(true);
      setShowForm(false);
      fetchWebhooks();
    } catch (err) {
      setMsg({ type: "error", text: err instanceof Error ? err.message : "Failed to create webhook" });
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Delete this webhook endpoint?")) return;
    try {
      await apiFetch(`/api/v1/webhooks/${id}`, { method: "DELETE" });
      setMsg({ type: "success", text: "Webhook deleted" });
      fetchWebhooks();
    } catch (err) {
      setMsg({ type: "error", text: err instanceof Error ? err.message : "Failed to delete" });
    }
  };

  const handleTest = async (id: string) => {
    setTestingId(id);
    try {
      const resp = await apiFetch<{ status?: number; delivered?: boolean }>(
        `/api/v1/webhooks/${id}/test`,
        { method: "POST" },
      );
      setTestResults((prev) => ({
        ...prev,
        [id]: { ok: true, status: resp.status || 200 },
      }));
      setMsg({ type: "success", text: `Test sent (status ${resp.status || 200})` });
    } catch (err) {
      setTestResults((prev) => ({
        ...prev,
        [id]: { ok: false, status: 0 },
      }));
      setMsg({ type: "error", text: err instanceof Error ? err.message : "Test failed" });
    } finally {
      setTestingId(null);
    }
  };

  const toggleRevealSecret = (id: string) => {
    setRevealedSecrets((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const copySecret = async (id: string, secret: string) => {
    try {
      await navigator.clipboard.writeText(secret);
      setCopiedSecret(id);
      setTimeout(() => setCopiedSecret(null), 2000);
    } catch {
      // clipboard unavailable
    }
  };

  const maskSecret = (secret: string) => {
    if (!secret) return "—";
    const visible = secret.slice(0, 10);
    return `${visible}••••••`;
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
          <Webhook className="h-6 w-6 text-brand-600" /> Webhooks
        </h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
        >
          <Plus className="h-4 w-4" /> New Webhook
        </button>
      </div>

      {msg && (
        <div
          className={`mb-4 rounded-lg border p-3 text-sm ${
            msg.type === "success"
              ? "border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400"
              : "border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400"
          }`}
        >
          {msg.text}
        </div>
      )}

      {/* Create Form */}
      {showForm && (
        <div className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold dark:text-gray-100">Create Webhook</h2>
            <button onClick={() => setShowForm(false)}>
              <X className="h-4 w-4 text-gray-400" />
            </button>
          </div>

          <div className="space-y-4">
            {/* URL input */}
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Endpoint URL</label>
              <input
                value={urlInput}
                onChange={(e) => {
                  setUrlInput(e.target.value);
                  setUrlError(null);
                }}
                placeholder="https://api.example.com/webhooks/ggid"
                className={`w-full rounded-lg border px-3 py-2 text-sm dark:bg-gray-700 dark:text-gray-200 ${
                  urlError
                    ? "border-red-400 dark:border-red-600"
                    : "border-gray-300 dark:border-gray-600"
                }`}
              />
              {urlError && <p className="mt-1 text-xs text-red-500">{urlError}</p>}
            </div>

            {/* Event multi-select */}
            <div>
              <label className="mb-2 block text-xs font-medium text-gray-500">
                Events to subscribe ({selectedEvents.size} selected)
              </label>
              <div className="flex flex-wrap gap-2">
                {AVAILABLE_EVENTS.map((event) => {
                  const active = selectedEvents.has(event);
                  return (
                    <button
                      key={event}
                      type="button"
                      onClick={() => toggleEvent(event)}
                      className={`flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs font-medium transition ${
                        active
                          ? "border-brand-500 bg-brand-50 text-brand-700 dark:bg-brand-950 dark:text-brand-300"
                          : "border-gray-300 text-gray-600 hover:border-gray-400 dark:border-gray-600 dark:text-gray-300"
                      }`}
                    >
                      {active && <Check className="h-3 w-3" />}
                      {event}
                    </button>
                  );
                })}
              </div>
            </div>

            {/* Enable/disable toggle */}
            <div className="flex items-center gap-3">
              <label className="text-xs font-medium text-gray-500">Status</label>
              <button
                type="button"
                onClick={() => setEnabledToggle(!enabledToggle)}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition ${
                  enabledToggle ? "bg-brand-600" : "bg-gray-300 dark:bg-gray-600"
                }`}
              >
                <span
                  className={`inline-block h-4 w-4 transform rounded-full bg-white transition ${
                    enabledToggle ? "translate-x-6" : "translate-x-1"
                  }`}
                />
              </button>
              <span className="text-sm text-gray-600 dark:text-gray-400">
                {enabledToggle ? "Enabled" : "Disabled"}
              </span>
            </div>

            {/* Create button */}
            <button
              onClick={handleCreate}
              disabled={creating}
              className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
            >
              {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
              Create Webhook
            </button>
          </div>
        </div>
      )}

      {/* Webhooks List */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-brand-600" />
        </div>
      ) : webhooks.length === 0 ? (
        <div className="rounded-xl border border-dashed border-gray-300 py-16 text-center dark:border-gray-600">
          <Webhook className="mx-auto mb-3 h-10 w-10 text-gray-300 dark:text-gray-600" />
          <p className="text-sm text-gray-500">No webhooks configured yet</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-gray-200 shadow-sm dark:border-gray-700">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800">
              <tr>
                <th className="px-4 py-3 font-semibold text-gray-600 dark:text-gray-300">URL</th>
                <th className="px-4 py-3 font-semibold text-gray-600 dark:text-gray-300">Events</th>
                <th className="px-4 py-3 font-semibold text-gray-600 dark:text-gray-300">Status</th>
                <th className="px-4 py-3 font-semibold text-gray-600 dark:text-gray-300">Created</th>
                <th className="px-4 py-3 font-semibold text-gray-600 dark:text-gray-300">Secret</th>
                <th className="px-4 py-3 font-semibold text-gray-600 dark:text-gray-300">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
              {webhooks.map((wh) => {
                const revealed = revealedSecrets.has(wh.id);
                const testResult = testResults[wh.id];
                return (
                  <tr key={wh.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                    {/* URL */}
                    <td className="px-4 py-3">
                      <span className="font-mono text-xs text-gray-700 dark:text-gray-300">
                        {wh.url}
                      </span>
                    </td>
                    {/* Events */}
                    <td className="px-4 py-3">
                      <div className="flex max-w-xs flex-wrap gap-1">
                        {(wh.events || []).map((ev) => (
                          <span
                            key={ev}
                            className="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-300"
                          >
                            {ev}
                          </span>
                        ))}
                      </div>
                    </td>
                    {/* Status */}
                    <td className="px-4 py-3">
                      {testResult ? (
                        testResult.ok ? (
                          <span className="flex items-center gap-1 text-xs text-green-600">
                            <CheckCircle2 className="h-3.5 w-3.5" /> {testResult.status}
                          </span>
                        ) : (
                          <span className="flex items-center gap-1 text-xs text-red-500">
                            <XCircle className="h-3.5 w-3.5" /> Failed
                          </span>
                        )
                      ) : wh.enabled ? (
                        <span className="inline-flex items-center gap-1 rounded-full bg-green-100 px-2 py-0.5 text-xs text-green-700 dark:bg-green-950 dark:text-green-400">
                          Active
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-gray-700">
                          Disabled
                        </span>
                      )}
                    </td>
                    {/* Created */}
                    <td className="px-4 py-3 text-xs text-gray-500">
                      {wh.created_at
                        ? new Date(wh.created_at).toLocaleDateString()
                        : "—"}
                    </td>
                    {/* Secret */}
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-1">
                        <span className="font-mono text-xs text-gray-600 dark:text-gray-400">
                          {revealed ? wh.secret || "—" : maskSecret(wh.secret)}
                        </span>
                        <button
                          onClick={() => toggleRevealSecret(wh.id)}
                          className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                          title={revealed ? "Hide" : "Reveal"}
                        >
                          {revealed ? (
                            <EyeOff className="h-3.5 w-3.5" />
                          ) : (
                            <Eye className="h-3.5 w-3.5" />
                          )}
                        </button>
                        {wh.secret && (
                          <button
                            onClick={() => copySecret(wh.id, wh.secret)}
                            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                            title="Copy"
                          >
                            {copiedSecret === wh.id ? (
                              <Check className="h-3.5 w-3.5 text-green-500" />
                            ) : (
                              <Copy className="h-3.5 w-3.5" />
                            )}
                          </button>
                        )}
                      </div>
                    </td>
                    {/* Actions */}
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => handleTest(wh.id)}
                          disabled={testingId === wh.id}
                          className="flex items-center gap-1 rounded border border-gray-300 px-2 py-1 text-xs text-gray-600 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                        >
                          {testingId === wh.id ? (
                            <Loader2 className="h-3 w-3 animate-spin" />
                          ) : (
                            <Send className="h-3 w-3" />
                          )}
                          Test
                        </button>
                        <button
                          onClick={() => handleDelete(wh.id)}
                          className="text-red-500 hover:text-red-700"
                          title="Delete"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
