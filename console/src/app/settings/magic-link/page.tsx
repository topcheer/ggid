"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Mail, Plus, X, AlertCircle, Loader2, Check, Save, Send,
  ToggleLeft, ToggleRight, Trash2, Clock, Globe,
} from "lucide-react";

interface MagicLinkConfig {
  enabled: boolean;
  expiry_minutes: number;
  single_use: boolean;
  allowed_domains: string[];
  require_https: boolean;
  redirect_url: string;
}

export default function MagicLinkPage() {
  const { apiFetch } = useApi();
  const [config, setConfig] = useState<MagicLinkConfig | null>(null);
  const [draft, setDraft] = useState<MagicLinkConfig>({ enabled: false, expiry_minutes: 15, single_use: true, allowed_domains: [], require_https: true, redirect_url: "" });
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [testEmail, setTestEmail] = useState("");
  const [sending, setSending] = useState(false);
  const [testResult, setTestResult] = useState<string | null>(null);
  const [newDomain, setNewDomain] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<MagicLinkConfig>("/api/v1/auth/magic-link/config").catch(() => null);
      if (data) { setConfig(data); setDraft(data); }
    } catch {
      setError("Failed to load magic link config");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); }, [load]);

  const handleSave = async () => {
    setSaving(true);
    try {
      await apiFetch("/api/v1/auth/magic-link/config", { method: "PUT", body: JSON.stringify(draft) });
      setConfig(draft);
      setEditing(false);
    } catch {
      setError("Failed to save config");
    } finally {
      setSaving(false);
    }
  };

  const handleTestSend = async () => {
    if (!testEmail.trim()) return;
    setSending(true);
    setTestResult(null);
    try {
      await apiFetch("/api/v1/auth/magic-link/test", { method: "POST", body: JSON.stringify({ email: testEmail }) });
      setTestResult(`Magic link sent to ${testEmail}`);
      setTestEmail("");
    } catch {
      setTestResult("Failed to send test magic link");
    } finally {
      setSending(false);
    }
  };

  const addDomain = () => {
    const d = newDomain.trim().replace(/^@/, "");
    if (d && !draft.allowed_domains.includes(d)) {
      setDraft((p) => ({ ...p, allowed_domains: [...p.allowed_domains, d] }));
    }
    setNewDomain("");
  };

  const removeDomain = (d: string) => setDraft((p) => ({ ...p, allowed_domains: p.allowed_domains.filter((x) => x !== d) }));

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  if (loading) return <div className="flex justify-center py-24"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>;

  const cfg = editing ? draft : config ?? draft;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Mail className="h-6 w-6 text-indigo-600" /> Magic Link Configuration
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Passwordless email-based authentication with domain restrictions.</p>
      </div>

      {error && (
        <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Config card */}
        <div className="lg:col-span-2">
          <div className={cardCls}>
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300">Settings</h3>
              {editing ? (
                <div className="flex gap-2">
                  <button onClick={handleSave} disabled={saving} className="flex items-center gap-1 rounded-lg bg-green-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-green-700"><Save className="h-3.5 w-3.5" />Save</button>
                  <button onClick={() => { setEditing(false); setDraft(config ?? draft); }} className="rounded-lg px-3 py-1.5 text-xs text-gray-500">Cancel</button>
                </div>
              ) : (
                <button onClick={() => setEditing(true)} className="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-500 hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700">Edit</button>
              )}
            </div>

            {/* Enabled toggle */}
            <div className="flex items-center justify-between border-b border-gray-100 py-3 dark:border-gray-700">
              <span className="text-sm text-gray-700 dark:text-gray-300">Enable Magic Link Login</span>
              {editing ? (
                <button onClick={() => setDraft((p) => ({ ...p, enabled: !p.enabled }))}>{cfg.enabled ? <ToggleRight className="h-6 w-6 text-indigo-600" /> : <ToggleLeft className="h-6 w-6 text-gray-300" />}</button>
              ) : <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${cfg.enabled ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : "bg-gray-100 text-gray-500"}`}>{cfg.enabled ? "Enabled" : "Disabled"}</span>}
            </div>

            {/* Single use toggle */}
            <div className="flex items-center justify-between border-b border-gray-100 py-3 dark:border-gray-700">
              <span className="text-sm text-gray-700 dark:text-gray-300">Single-Use Only</span>
              {editing ? (
                <button onClick={() => setDraft((p) => ({ ...p, single_use: !p.single_use }))}>{cfg.single_use ? <ToggleRight className="h-6 w-6 text-indigo-600" /> : <ToggleLeft className="h-6 w-6 text-gray-300" />}</button>
              ) : <span className="text-xs text-gray-500">{cfg.single_use ? "Yes" : "No"}</span>}
            </div>

            {/* Expiry */}
            <div className="flex items-center justify-between border-b border-gray-100 py-3 dark:border-gray-700">
              <span className="flex items-center gap-1 text-sm text-gray-700 dark:text-gray-300"><Clock className="h-4 w-4 text-gray-400" />Link Expiry (minutes)</span>
              {editing ? <input type="number" value={cfg.expiry_minutes} onChange={(e) => setDraft((p) => ({ ...p, expiry_minutes: Number(e.target.value) }))} className="w-20 rounded-lg border border-gray-300 px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /> : <span className="font-medium text-indigo-600">{cfg.expiry_minutes} min</span>}
            </div>

            {/* Redirect URL */}
            <div className="flex items-center justify-between border-b border-gray-100 py-3 dark:border-gray-700">
              <span className="text-sm text-gray-700 dark:text-gray-300">Redirect URL</span>
              {editing ? <input value={cfg.redirect_url} onChange={(e) => setDraft((p) => ({ ...p, redirect_url: e.target.value }))} placeholder="/dashboard" className="w-48 rounded-lg border border-gray-300 px-2 py-1 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" /> : <span className="font-mono text-xs text-gray-500">{cfg.redirect_url || "/dashboard"}</span>}
            </div>

            {/* Allowed domains */}
            <div className="py-3">
              <span className="flex items-center gap-1 text-sm text-gray-700 dark:text-gray-300"><Globe className="h-4 w-4 text-gray-400" />Allowed Domains</span>
              <div className="mt-2 flex flex-wrap gap-2">
                {cfg.allowed_domains.map((d) => (
                  <span key={d} className="flex items-center gap-1 rounded-lg bg-indigo-50 px-2 py-1 text-xs text-indigo-700 dark:bg-indigo-900/20 dark:text-indigo-400">
                    @{d}
                    {editing && <button onClick={() => removeDomain(d)}><X className="h-3 w-3" /></button>}
                  </span>
                ))}
                {editing && (
                  <div className="flex items-center gap-1">
                    <input value={newDomain} onChange={(e) => setNewDomain(e.target.value)} onKeyDown={(e) => e.key === "Enter" && addDomain()} placeholder="company.com" className="w-32 rounded border border-gray-300 px-2 py-1 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
                    <button onClick={addDomain} className="rounded p-1 text-indigo-600 hover:bg-indigo-50"><Plus className="h-3.5 w-3.5" /></button>
                  </div>
                )}
                {!editing && cfg.allowed_domains.length === 0 && <span className="text-xs text-gray-400">All domains allowed</span>}
              </div>
            </div>
          </div>
        </div>

        {/* Test send */}
        <div>
          <div className={cardCls}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300"><Send className="h-4 w-4" /> Test Send</h3>
            <input value={testEmail} onChange={(e) => setTestEmail(e.target.value)} placeholder="user@company.com" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-white" />
            <button onClick={handleTestSend} disabled={!testEmail.trim() || sending} className="mt-3 flex w-full items-center justify-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
              {sending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Mail className="h-4 w-4" />}Send Magic Link
            </button>
            {testResult && <p className="mt-3 flex items-center gap-1 text-xs text-gray-500"><Check className="h-3 w-3 text-green-500" />{testResult}</p>}
          </div>
        </div>
      </div>
    </div>
  );
}
