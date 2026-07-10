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
  History,
  Pencil,
  Save,
  ChevronDown,
  ChevronUp,
  Clock,
} from "lucide-react";

/* ── Categorized events ── */
const EVENT_CATEGORIES: { label: string; events: string[] }[] = [
  {
    label: "User Events",
    events: ["user.created", "user.updated", "user.deleted", "user.login", "user.logout"],
  },
  {
    label: "Auth Events",
    events: ["auth.login_failed", "auth.mfa_enrolled", "auth.password_changed"],
  },
  {
    label: "Org Events",
    events: ["org.created", "org.updated", "member.added", "member.removed"],
  },
  {
    label: "Policy Events",
    events: ["policy.created", "policy.updated", "policy.evaluated"],
  },
];

/* Legacy events kept for backwards compatibility */
const LEGACY_EVENTS = ["role.created", "role.deleted"];

interface WebhookEndpoint {
  id: string;
  url: string;
  events: string[];
  enabled: boolean;
  secret: string;
  created_at: string;
}

interface DeliveryRecord {
  id: string;
  event_type: string;
  status_code: number;
  response_time_ms: number;
  retry_count: number;
  delivered_at: string;
}

function isValidUrl(url: string): boolean {
  try {
    const u = new URL(url);
    return u.protocol === "http:" || u.protocol === "https:";
  } catch {
    return false;
  }
}

function statusCodeColor(code: number): string {
  if (code >= 200 && code < 300) return "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-400";
  if (code >= 400 && code < 500) return "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-400";
  if (code >= 500) return "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-400";
  return "bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300";
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

  // Delivery history state
  const [historyWebhookId, setHistoryWebhookId] = useState<string | null>(null);
  const [deliveries, setDeliveries] = useState<Record<string, DeliveryRecord[]>>({});
  const [historyLoading, setHistoryLoading] = useState<string | null>(null);

  // Edit state
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editUrl, setEditUrl] = useState("");
  const [editEvents, setEditEvents] = useState<Set<string>>(new Set());
  const [editError, setEditError] = useState<string | null>(null);
  const [savingEdit, setSavingEdit] = useState(false);

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

  /* ── Event toggle for create form ── */
  const toggleEvent = (event: string) => {
    setSelectedEvents((prev) => {
      const next = new Set(prev);
      if (next.has(event)) next.delete(event);
      else next.add(event);
      return next;
    });
  };

  /* ── Event toggle for edit form ── */
  const toggleEditEvent = (event: string) => {
    setEditEvents((prev) => {
      const next = new Set(prev);
      if (next.has(event)) next.delete(event);
      else next.add(event);
      return next;
    });
  };

  /* ── Toggle all events in a category (create) ── */
  const toggleCategory = (events: string[], allSelected: boolean) => {
    setSelectedEvents((prev) => {
      const next = new Set(prev);
      if (allSelected) {
        events.forEach((e) => next.delete(e));
      } else {
        events.forEach((e) => next.add(e));
      }
      return next;
    });
  };

  /* ── Toggle all events in a category (edit) ── */
  const toggleEditCategory = (events: string[], allSelected: boolean) => {
    setEditEvents((prev) => {
      const next = new Set(prev);
      if (allSelected) {
        events.forEach((e) => next.delete(e));
      } else {
        events.forEach((e) => next.add(e));
      }
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

  /* ── Delivery history ── */
  const handleViewHistory = async (id: string) => {
    if (historyWebhookId === id) {
      setHistoryWebhookId(null);
      return;
    }
    setHistoryWebhookId(id);

    // Fetch if not cached
    if (!deliveries[id]) {
      setHistoryLoading(id);
      try {
        const data = await apiFetch<{ deliveries?: DeliveryRecord[] } | DeliveryRecord[]>(
          `/api/v1/webhooks/${id}/deliveries?page_size=20`,
        ).catch(() => ({ deliveries: [] }));
        const list = Array.isArray(data) ? data : data.deliveries || [];
        setDeliveries((prev) => ({ ...prev, [id]: list }));
      } catch {
        setDeliveries((prev) => ({ ...prev, [id]: [] }));
      } finally {
        setHistoryLoading(null);
      }
    }
  };

  /* ── Inline edit ── */
  const startEdit = (wh: WebhookEndpoint) => {
    setEditingId(wh.id);
    setEditUrl(wh.url);
    setEditEvents(new Set(wh.events || []));
    setEditError(null);
  };

  const cancelEdit = () => {
    setEditingId(null);
    setEditUrl("");
    setEditEvents(new Set());
    setEditError(null);
  };

  const handleSaveEdit = async (id: string) => {
    setEditError(null);
    if (!editUrl.trim()) {
      setEditError("URL is required");
      return;
    }
    if (!isValidUrl(editUrl.trim())) {
      setEditError("Please enter a valid http(s) URL");
      return;
    }
    if (editEvents.size === 0) {
      setEditError("Select at least one event");
      return;
    }

    setSavingEdit(true);
    try {
      await apiFetch(`/api/v1/webhooks/${id}`, {
        method: "PUT",
        body: JSON.stringify({
          url: editUrl.trim(),
          events: [...editEvents],
        }),
      });
      setMsg({ type: "success", text: "Webhook updated successfully" });
      cancelEdit();
      fetchWebhooks();
    } catch (err) {
      setEditError(err instanceof Error ? err.message : "Failed to update webhook");
    } finally {
      setSavingEdit(false);
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

  /* ── Categorized event picker component ── */
  const renderEventCategories = (
    selected: Set<string>,
    onToggle: (e: string) => void,
    onToggleCat: (events: string[], all: boolean) => void,
  ) => (
    <div className="space-y-3">
      {EVENT_CATEGORIES.map((cat) => {
        const allSelected = cat.events.every((e) => selected.has(e));
        return (
          <div key={cat.label} className="rounded-lg border border-gray-200 p-3 dark:border-gray-600">
            <div className="mb-2 flex items-center justify-between">
              <span className="text-xs font-semibold text-gray-700 dark:text-gray-300">{cat.label}</span>
              <button
                type="button"
                onClick={() => onToggleCat(cat.events, allSelected)}
                className="text-xs font-medium text-brand-600 hover:underline"
              >
                {allSelected ? "Clear all" : "Select all"}
              </button>
            </div>
            <div className="flex flex-wrap gap-2">
              {cat.events.map((event) => {
                const active = selected.has(event);
                return (
                  <button
                    key={event}
                    type="button"
                    onClick={() => onToggle(event)}
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
        );
      })}
      {/* Legacy events (not in categories but may exist on existing webhooks) */}
      {LEGACY_EVENTS.some((e) => selected.has(e)) && (
        <div className="flex flex-wrap gap-2">
          {LEGACY_EVENTS.map((event) => {
            const active = selected.has(event);
            return (
              <button
                key={event}
                type="button"
                onClick={() => onToggle(event)}
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
      )}
    </div>
  );

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

            {/* Categorized event picker */}
            <div>
              <label className="mb-2 block text-xs font-medium text-gray-500">
                Event Subscriptions ({selectedEvents.size} selected)
              </label>
              {renderEventCategories(selectedEvents, toggleEvent, toggleCategory)}
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
        <div className="space-y-4">
          {webhooks.map((wh) => {
            const revealed = revealedSecrets.has(wh.id);
            const testResult = testResults[wh.id];
            const isEditing = editingId === wh.id;
            const showHistory = historyWebhookId === wh.id;
            const whDeliveries = deliveries[wh.id] || [];
            const isLoadingHistory = historyLoading === wh.id;

            return (
              <div
                key={wh.id}
                className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800"
              >
                {/* Main row */}
                <div className="flex flex-wrap items-start justify-between gap-4 p-4">
                  {isEditing ? (
                    /* ── Inline edit mode ── */
                    <div className="w-full space-y-3">
                      <div className="flex items-center gap-2">
                        <Pencil className="h-4 w-4 text-brand-600" />
                        <span className="text-sm font-semibold text-gray-700 dark:text-gray-200">Edit Webhook</span>
                      </div>
                      <div>
                        <label className="mb-1 block text-xs font-medium text-gray-500">Endpoint URL</label>
                        <input
                          value={editUrl}
                          onChange={(e) => {
                            setEditUrl(e.target.value);
                            setEditError(null);
                          }}
                          className={`w-full rounded-lg border px-3 py-2 text-sm dark:bg-gray-700 dark:text-gray-200 ${
                            editError
                              ? "border-red-400 dark:border-red-600"
                              : "border-gray-300 dark:border-gray-600"
                          }`}
                        />
                      </div>
                      <div>
                        <label className="mb-2 block text-xs font-medium text-gray-500">
                          Events ({editEvents.size} selected)
                        </label>
                        {renderEventCategories(editEvents, toggleEditEvent, toggleEditCategory)}
                      </div>
                      {editError && <p className="text-xs text-red-500">{editError}</p>}
                      <div className="flex gap-2">
                        <button
                          onClick={() => handleSaveEdit(wh.id)}
                          disabled={savingEdit}
                          className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
                        >
                          {savingEdit ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                          Save
                        </button>
                        <button
                          onClick={cancelEdit}
                          className="flex items-center gap-1.5 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300"
                        >
                          <X className="h-4 w-4" /> Cancel
                        </button>
                      </div>
                    </div>
                  ) : (
                    /* ── Display mode ── */
                    <>
                      <div className="min-w-0 flex-1">
                        <div className="mb-2 flex items-center gap-2">
                          <span className="truncate font-mono text-xs text-gray-700 dark:text-gray-300">
                            {wh.url}
                          </span>
                          {testResult ? (
                            testResult.ok ? (
                              <span className="flex items-center gap-0.5 text-xs text-green-600">
                                <CheckCircle2 className="h-3.5 w-3.5" /> {testResult.status}
                              </span>
                            ) : (
                              <span className="flex items-center gap-0.5 text-xs text-red-500">
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
                        </div>
                        <div className="flex flex-wrap gap-1">
                          {(wh.events || []).map((ev) => (
                            <span
                              key={ev}
                              className="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-300"
                            >
                              {ev}
                            </span>
                          ))}
                        </div>
                        <div className="mt-2 flex items-center gap-1">
                          <span className="font-mono text-xs text-gray-600 dark:text-gray-400">
                            {revealed ? wh.secret || "—" : maskSecret(wh.secret)}
                          </span>
                          <button
                            onClick={() => toggleRevealSecret(wh.id)}
                            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                            title={revealed ? "Hide" : "Reveal"}
                          >
                            {revealed ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
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
                          <span className="ml-3 text-xs text-gray-400">
                            {wh.created_at ? new Date(wh.created_at).toLocaleDateString() : "—"}
                          </span>
                        </div>
                      </div>

                      {/* Actions */}
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => handleViewHistory(wh.id)}
                          className={`flex items-center gap-1 rounded border px-2 py-1 text-xs font-medium transition ${
                            showHistory
                              ? "border-brand-500 bg-brand-50 text-brand-700 dark:bg-brand-950 dark:text-brand-300"
                              : "border-gray-300 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                          }`}
                        >
                          {showHistory ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
                          <History className="h-3 w-3" /> History
                        </button>
                        <button
                          onClick={() => startEdit(wh)}
                          className="flex items-center gap-1 rounded border border-gray-300 px-2 py-1 text-xs text-gray-600 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                        >
                          <Pencil className="h-3 w-3" /> Edit
                        </button>
                        <button
                          onClick={() => handleTest(wh.id)}
                          disabled={testingId === wh.id}
                          className="flex items-center gap-1 rounded border border-gray-300 px-2 py-1 text-xs text-gray-600 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                        >
                          {testingId === wh.id ? <Loader2 className="h-3 w-3 animate-spin" /> : <Send className="h-3 w-3" />}
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
                    </>
                  )}
                </div>

                {/* Delivery history (expandable) */}
                {showHistory && (
                  <div className="border-t border-gray-100 bg-gray-50 px-4 py-3 dark:border-gray-700 dark:bg-gray-900/50">
                    {isLoadingHistory ? (
                      <div className="flex items-center justify-center py-6">
                        <Loader2 className="h-4 w-4 animate-spin text-brand-600" />
                        <span className="ml-2 text-xs text-gray-500">Loading delivery history...</span>
                      </div>
                    ) : whDeliveries.length === 0 ? (
                      <div className="py-6 text-center">
                        <Clock className="mx-auto mb-2 h-6 w-6 text-gray-300 dark:text-gray-600" />
                        <p className="text-xs text-gray-400">No delivery history yet</p>
                      </div>
                    ) : (
                      <div className="overflow-x-auto">
                        <table className="w-full text-left text-xs">
                          <thead>
                            <tr className="border-b border-gray-200 dark:border-gray-700">
                              <th className="py-2 pr-3 font-medium text-gray-500">Timestamp</th>
                              <th className="py-2 pr-3 font-medium text-gray-500">Event</th>
                              <th className="py-2 pr-3 font-medium text-gray-500">Status</th>
                              <th className="py-2 pr-3 font-medium text-gray-500">Response Time</th>
                              <th className="py-2 pr-3 font-medium text-gray-500">Retries</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                            {whDeliveries.slice(0, 20).map((d) => (
                              <tr key={d.id}>
                                <td className="py-1.5 pr-3 text-gray-500">
                                  {d.delivered_at ? new Date(d.delivered_at).toLocaleString() : "—"}
                                </td>
                                <td className="py-1.5 pr-3">
                                  <span className="rounded bg-gray-100 px-1.5 py-0.5 text-gray-600 dark:bg-gray-700 dark:text-gray-300">
                                    {d.event_type}
                                  </span>
                                </td>
                                <td className="py-1.5 pr-3">
                                  <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${statusCodeColor(d.status_code)}`}>
                                    {d.status_code || "—"}
                                  </span>
                                </td>
                                <td className="py-1.5 pr-3 text-gray-500">
                                  {d.response_time_ms != null ? `${d.response_time_ms}ms` : "—"}
                                </td>
                                <td className="py-1.5 pr-3 text-gray-500">
                                  {d.retry_count > 0 ? (
                                    <span className="text-amber-600">{d.retry_count}</span>
                                  ) : (
                                    "0"
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
            );
          })}
        </div>
      )}
    </div>
  );
}
