"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Bell, Mail, MessageSquare, Smartphone, Plus, Trash2, X,
  AlertCircle, Loader2, Send, Check, Pencil, Eye,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

type Channel = "email" | "sms" | "push";

interface ProviderConfig {
  enabled: boolean;
  provider: string;
  from?: string;
  webhook_url?: string;
  api_key_set?: boolean;
}

interface NotificationTemplate {
  id: string;
  name: string;
  channel: Channel;
  subject: string;
  body: string;
  variables: string[];
  enabled: boolean;
}

interface NotificationSettings {
  providers: Record<Channel, ProviderConfig>;
  templates: NotificationTemplate[];
}

const CHANNEL_ICON: Record<Channel, typeof Mail> = {
  email: Mail,
  sms: MessageSquare,
  push: Smartphone,
};

const CHANNEL_LABEL: Record<Channel, string> = {
  email: "Email",
  sms: "SMS",
  push: "Push",
};

export default function NotificationPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [settings, setSettings] = useState<NotificationSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editTemplate, setEditTemplate] = useState<NotificationTemplate | null>(null);
  const [showTemplate, setShowTemplate] = useState(false);
  const [testing, setTesting] = useState<Channel | null>(null);
  const [testResult, setTestResult] = useState<string | null>(null);

  // Template form
  const [tmplForm, setTmplForm] = useState({
    name: "",
    channel: "email" as Channel,
    subject: "",
    body: "",
  });

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<NotificationSettings>("/api/v1/settings/notifications").catch(() => null);
      if (data) setSettings(data);
    } catch {
      setError("Failed to load notification settings");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const toggleProvider = async (ch: Channel) => {
    if (!settings) return;
    const current = settings.providers[ch];
    try {
      await apiFetch(`/api/v1/settings/notifications/providers/${ch}`, {
        method: "PATCH", body: JSON.stringify({ enabled: !current.enabled }),
      });
      setSettings({
        ...settings,
        providers: { ...settings.providers, [ch]: { ...current, enabled: !current.enabled } },
      });
    } catch {
      setError("Failed to toggle provider");
    }
  };

  const handleTestSend = async (ch: Channel) => {
    setTesting(ch);
    setTestResult(null);
    try {
      await apiFetch(`/api/v1/settings/notifications/test`, {
        method: "POST", body: JSON.stringify({ channel: ch }),
      });
      setTestResult(`${CHANNEL_LABEL[ch]} test message sent successfully.`);
    } catch {
      setTestResult(`${CHANNEL_LABEL[ch]} test failed. Check provider configuration.`);
    } finally {
      setTesting(null);
      setTimeout(() => setTestResult(null), 5000);
    }
  };

  const handleSaveTemplate = async () => {
    if (!editTemplate || !tmplForm.name.trim()) return;
    try {
      await apiFetch(`/api/v1/settings/notifications/templates/${editTemplate.id}`, {
        method: "PUT", body: JSON.stringify(tmplForm),
      });
      setEditTemplate(null);
      await load();
    } catch {
      setError("Failed to save template");
    }
  };

  const handleCreateTemplate = async () => {
    if (!tmplForm.name.trim()) return;
    try {
      await apiFetch("/api/v1/settings/notifications/templates", {
        method: "POST", body: JSON.stringify(tmplForm),
      });
      setTmplForm({ name: "", channel: "email", subject: "", body: "" });
      setShowTemplate(false);
      await load();
    } catch {
      setError("Failed to create template");
    }
  };

  const handleDeleteTemplate = async (id: string) => {
    try {
      await apiFetch(`/api/v1/settings/notifications/templates/${id}`, { method: "DELETE" });
      await load();
    } catch {
      setError("Failed to delete template");
    }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  if (loading) {
    return <div className="flex justify-center py-24"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Bell className="h-6 w-6 text-indigo-600" /> Notifications
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Configure notification channels and message templates.
          </p>
        </div>
        <button onClick={() => setShowTemplate(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700">
          <Plus className="h-4 w-4" /> New Template
        </button>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}
      {testResult && (
        <div className={`flex items-center gap-2 rounded-lg px-4 py-3 text-sm ${testResult.includes("failed") ? "bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-400" : "bg-green-50 text-green-700 dark:bg-green-900/20 dark:text-green-400"}`}>
          {testResult.includes("failed") ? <AlertCircle className="h-4 w-4" /> : <Check className="h-4 w-4" />}
          {testResult}
        </div>
      )}

      {/* Provider channels */}
      <div className="grid gap-4 md:grid-cols-3">
        {settings && (Object.entries(settings.providers) as [Channel, ProviderConfig][]).map(([ch, cfg]) => {
          const Icon = CHANNEL_ICON[ch];
          return (
            <div key={ch} className={cardCls}>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className={`rounded-lg p-2 ${cfg.enabled ? "bg-indigo-100 dark:bg-indigo-900/30" : "bg-gray-100 dark:bg-gray-700"}`}>
                    <Icon className={`h-5 w-5 ${cfg.enabled ? "text-indigo-600" : "text-gray-400"}`} />
                  </div>
                  <div>
                    <h3 className="font-semibold text-gray-800 dark:text-gray-200">{CHANNEL_LABEL[ch]}</h3>
                    <p className="text-xs text-gray-400">{cfg.provider || "Not configured"}</p>
                  </div>
                </div>
                <label className="relative inline-flex cursor-pointer items-center">
                  <input type="checkbox" checked={cfg.enabled} onChange={() => toggleProvider(ch)} className="peer sr-only" />
                  <div className="h-5 w-9 rounded-full bg-gray-200 after:absolute after:left-[2px] after:top-[2px] after:h-4 after:w-4 after:rounded-full after:border after:transition-all peer-checked:bg-indigo-600 peer-checked:after:translate-x-full dark:bg-gray-700" />
                </label>
              </div>
              {cfg.from && <p className="mt-2 text-xs text-gray-400">From: {cfg.from}</p>}
              <button
                onClick={() => handleTestSend(ch)}
                disabled={!cfg.enabled || testing === ch}
                className="mt-3 flex items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                {testing === ch ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Send className="h-3.5 w-3.5" />}
                Send Test
              </button>
            </div>
          );
        })}
      </div>

      {/* Templates */}
      <div>
        <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">Message Templates</h2>
        {!settings?.templates || settings.templates.length === 0 ? (
          <div className={cardCls}>
            <div className="py-12 text-center">
              <Bell className="mx-auto h-12 w-12 text-gray-300" />
              <p className="mt-4 text-sm text-gray-400">No notification templates yet.</p>
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            {settings.templates.map((t) => {
              const Icon = CHANNEL_ICON[t.channel];
              return (
                <div key={t.id} className={cardCls}>
                  <div className="flex items-start justify-between">
                    <div className="flex items-start gap-3">
                      <div className="rounded-lg bg-gray-100 p-2 dark:bg-gray-700">
                        <Icon className="h-4 w-4 text-gray-500" />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-gray-800 dark:text-gray-200">{t.name}</span>
                          <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">{CHANNEL_LABEL[t.channel]}</span>
                          {!t.enabled && <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-gray-700">Disabled</span>}
                        </div>
                        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t.subject}</p>
                        <p className="mt-1 line-clamp-2 text-xs text-gray-400">{t.body}</p>
                        {t.variables.length > 0 && (
                          <div className="mt-2 flex flex-wrap gap-1">
                            {t.variables.map((v) => (
                              <code key={v} className="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-500 dark:bg-gray-700">{`{{${v}}}`}</code>
                            ))}
                          </div>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-1">
                      <button
                        onClick={() => { setEditTemplate(t); setTmplForm({ name: t.name, channel: t.channel, subject: t.subject, body: t.body }); }}
                        className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"
                      >
                        <Pencil className="h-4 w-4" />
                      </button>
                      <button onClick={() => handleDeleteTemplate(t.id)} className="rounded-lg p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20">
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Create/Edit template modal */}
      {(showTemplate || editTemplate) && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => { setShowTemplate(false); setEditTemplate(null); }}>
          <div className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-white">{editTemplate ? "Edit Template" : "New Template"}</h2>
              <button onClick={() => { setShowTemplate(false); setEditTemplate(null); }}><X className="h-5 w-5 text-gray-400" /></button>
            </div>
            <div className="mt-4 space-y-4">
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Template Name</label>
                <input value={tmplForm.name} onChange={(e) => setTmplForm((p) => ({ ...p, name: e.target.value }))} placeholder="Password Reset" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Channel</label>
                <div className="mt-2 flex gap-2">
                  {(["email", "sms", "push"] as const).map((ch) => {
                    const Icon = CHANNEL_ICON[ch];
                    return (
                      <button key={ch} onClick={() => setTmplForm((p) => ({ ...p, channel: ch }))}
                        className={`flex items-center gap-1.5 rounded-lg border px-3 py-2 text-sm ${tmplForm.channel === ch ? "border-indigo-500 bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400" : "border-gray-300 text-gray-500 dark:border-gray-600"}`}>
                        <Icon className="h-4 w-4" /> {CHANNEL_LABEL[ch]}
                      </button>
                    );
                  })}
                </div>
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Subject</label>
                <input value={tmplForm.subject} onChange={(e) => setTmplForm((p) => ({ ...p, subject: e.target.value }))} placeholder="[GGID] Your verification code" className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
              </div>
              <div>
                <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Body (use {`{{variables}}`})</label>
                <textarea value={tmplForm.body} onChange={(e) => setTmplForm((p) => ({ ...p, body: e.target.value }))} rows={5} placeholder={"Hello {{name}},\n\nYour verification code is: {{code}}\n\nThis code expires in {{expiry}} minutes."} className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 font-mono text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
              </div>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button onClick={() => { setShowTemplate(false); setEditTemplate(null); }} className="rounded-lg px-4 py-2 text-sm text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700">Cancel</button>
              <button
                onClick={() => editTemplate ? handleSaveTemplate() : handleCreateTemplate()}
                disabled={!tmplForm.name.trim()}
                className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
              >
                <Check className="h-4 w-4" /> {editTemplate ? "Save" : "Create"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
