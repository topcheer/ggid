"use client";

import React, { useState } from "react";
import { useApi } from "@/lib/api";
import {
  KeyRound, Loader2, AlertCircle, X, Save, Send, RotateCcw, ShieldCheck,
} from "lucide-react";

interface ResetConfig {
  token_expiry_minutes: number;
  max_attempts: number;
  check_password_history: boolean;
  history_count: number;
  require_mfa: boolean;
  notify_on_reset: boolean;
  cooldown_minutes: number;
}

export default function PasswordResetPage() {
  const { apiFetch } = useApi();
  const [config, setConfig] = useState<ResetConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [testEmail, setTestEmail] = useState("");
  const [sending, setSending] = useState(false);
  const [testResult, setTestResult] = useState<string | null>(null);

  useState(() => {
    (async () => {
      try { setConfig(await apiFetch<ResetConfig>("/api/v1/auth/password-reset/config").catch(() => ({ token_expiry_minutes: 30, max_attempts: 5, check_password_history: true, history_count: 5, require_mfa: false, notify_on_reset: true, cooldown_minutes: 5 }))); }
      catch { setError("Failed to load config"); }
      finally { setLoading(false); }
    })();
  });

  const handleSave = async () => {
    if (!config) return;
    setSaving(true);
    try { await apiFetch("/api/v1/auth/password-reset/config", { method: "PUT", body: JSON.stringify(config) }); }
    catch { setError("Save failed"); }
    finally { setSaving(false); }
  };

  const handleTestInitiate = async () => {
    if (!testEmail) return;
    setSending(true); setTestResult(null);
    try { await apiFetch("/api/v1/auth/password-reset/initiate", { method: "POST", body: JSON.stringify({ email: testEmail }) }); setTestResult("Reset email sent successfully"); }
    catch { setTestResult("Initiate failed"); }
    finally { setSending(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><KeyRound className="h-6 w-6 text-orange-600" /> Password Reset</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Configure password reset flow: token expiry, rate limiting, and history checking.</p>
      </div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-orange-600" /></div>
      : config ? (
        <>
          <div className={cardCls}>
            <h3 className="mb-4 text-sm font-semibold text-gray-700 dark:text-gray-300">Configuration</h3>
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Token Expiry (minutes)</label><input type="number" value={config.token_expiry_minutes} onChange={(e) => setConfig({ ...config, token_expiry_minutes: parseInt(e.target.value) || 30 })} min={5} max={1440} className={inputCls} /></div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Max Attempts</label><input type="number" value={config.max_attempts} onChange={(e) => setConfig({ ...config, max_attempts: parseInt(e.target.value) || 5 })} min={1} max={20} className={inputCls} /></div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Cooldown (minutes)</label><input type="number" value={config.cooldown_minutes} onChange={(e) => setConfig({ ...config, cooldown_minutes: parseInt(e.target.value) || 5 })} min={1} max={60} className={inputCls} /></div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Password History Count</label><input type="number" value={config.history_count} onChange={(e) => setConfig({ ...config, history_count: parseInt(e.target.value) || 5 })} min={0} max={24} disabled={!config.check_password_history} className={`${inputCls} disabled:opacity-50`} /></div>
              </div>
              <div className="flex flex-wrap gap-4">
                <label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300"><input type="checkbox" checked={config.check_password_history} onChange={(e) => setConfig({ ...config, check_password_history: e.target.checked })} />Check password history</label>
                <label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300"><input type="checkbox" checked={config.require_mfa} onChange={(e) => setConfig({ ...config, require_mfa: e.target.checked })} />Require MFA for reset</label>
                <label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300"><input type="checkbox" checked={config.notify_on_reset} onChange={(e) => setConfig({ ...config, notify_on_reset: e.target.checked })} />Notify on successful reset</label>
              </div>
              <button onClick={handleSave} disabled={saving} className="flex items-center gap-2 rounded-lg bg-orange-600 px-4 py-2 text-sm font-medium text-white hover:bg-orange-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}Save Config</button>
            </div>
          </div>

          {/* Test initiate */}
          <div className={cardCls}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300"><Send className="h-4 w-4" /> Test Reset Initiate</h3>
            <div className="flex items-center gap-2">
              <input value={testEmail} onChange={(e) => setTestEmail(e.target.value)} placeholder="user@example.com" className={inputCls} />
              <button onClick={handleTestInitiate} disabled={!testEmail || sending} className="flex items-center gap-2 whitespace-nowrap rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{sending ? <Loader2 className="h-4 w-4 animate-spin" /> : <RotateCcw className="h-4 w-4" />} Send</button>
            </div>
            {testResult && <p className={`mt-2 flex items-center gap-1 text-sm ${testResult.includes("success") ? "text-green-600" : "text-red-600"}`}><ShieldCheck className="h-3 w-3" />{testResult}</p>}
          </div>
        </>
      ) : null}
    </div>
  );
}
