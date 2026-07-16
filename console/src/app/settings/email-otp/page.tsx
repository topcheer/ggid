"use client";

import React, { useState } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  Mail, Loader2, AlertCircle, X, Send, Save, Plus, Trash2,
} from "lucide-react";

interface EmailOTPConfig {
  enabled: boolean;
  otp_length: number;
  expiry_seconds: number;
  rate_limit_per_hour: number;
  allowed_domains: string[];
  issuer_name: string;
}

export default function EmailOTPPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [config, setConfig] = useState<EmailOTPConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [domainInput, setDomainInput] = useState("");
  const [testEmail, setTestEmail] = useState("");
  const [sending, setSending] = useState(false);
  const [testResult, setTestResult] = useState<string | null>(null);

  useState(() => {
    (async () => {
      try { setConfig(await apiFetch<EmailOTPConfig>("/api/v1/auth/email-otp/config").catch(() => null)); }
      catch { setError("Failed to load config"); }
      finally { setLoading(false); }
    })();
  });

  const handleSave = async () => {
    if (!config) return;
    setSaving(true);
    try { await apiFetch("/api/v1/auth/email-otp/config", { method: "PUT", body: JSON.stringify(config) }); }
    catch { setError("Save failed"); }
    finally { setSaving(false); }
  };

  const handleTestSend = async () => {
    if (!testEmail) return;
    setSending(true); setTestResult(null);
    try { await apiFetch("/api/v1/auth/email-otp/send", { method: "POST", body: JSON.stringify({ email: testEmail }) }); setTestResult("OTP sent successfully"); }
    catch { setTestResult("Send failed"); }
    finally { setSending(false); }
  };

  const addDomain = () => {
    if (!config || !domainInput.trim()) return;
    setConfig({ ...config, allowed_domains: [...config.allowed_domains, domainInput.trim()] });
    setDomainInput("");
  };

  const removeDomain = (idx: number) => {
    if (!config) return;
    setConfig({ ...config, allowed_domains: config.allowed_domains.filter((_, i) => i !== idx) });
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Mail className="h-6 w-6 text-blue-600" />{t("emailOtp.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Email-based one-time password configuration and testing.</p>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-600" /></div>
      : config ? (
        <>
          {/* Config card */}
          <div className={cardCls}>
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300">Configuration</h3>
              <button onClick={() => setConfig({ ...config, enabled: !config.enabled })} className={`flex items-center gap-2 rounded-lg px-3 py-1.5 text-sm font-medium ${config.enabled ? "bg-green-100 text-green-700 dark:bg-green-900/30" : "bg-gray-100 text-gray-500 dark:bg-gray-700"}`}>{config.enabled ? "Enabled" : "Disabled"}</button>
            </div>
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">OTP Length</label><input type="number" value={config.otp_length} onChange={(e) => setConfig({ ...config, otp_length: parseInt(e.target.value) || 6 })} min={4} max={10} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Expiry (seconds)</label><input type="number" value={config.expiry_seconds} onChange={(e) => setConfig({ ...config, expiry_seconds: parseInt(e.target.value) || 300 })} min={30} max={3600} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Rate Limit (per hour)</label><input type="number" value={config.rate_limit_per_hour} onChange={(e) => setConfig({ ...config, rate_limit_per_hour: parseInt(e.target.value) || 5 })} min={1} max={100} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Issuer Name</label><input value={config.issuer_name} onChange={(e) => setConfig({ ...config, issuer_name: e.target.value })} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              </div>
              {/* Allowed domains */}
              <div>
                <label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Allowed Domains</label>
                <div className="flex gap-2">
                  <input value={domainInput} onChange={(e) => setDomainInput(e.target.value)} onKeyDown={(e) => e.key === "Enter" && addDomain()} placeholder="example.com" className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" />
                  <button onClick={addDomain} className="rounded-lg bg-gray-100 px-3 text-sm text-gray-600 dark:bg-gray-700 dark:text-gray-300"><Plus className="h-4 w-4" /></button>
                </div>
                {config.allowed_domains.length > 0 && (
                  <div className="mt-2 flex flex-wrap gap-2">
                    {config.allowed_domains.map((d, i) => (
                      <span key={i} className="flex items-center gap-1 rounded-full bg-blue-100 px-2 py-1 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">{d}<button onClick={() => removeDomain(i)}><X className="h-3 w-3" /></button></span>
                    ))}
                  </div>
                )}
              </div>
              <button onClick={handleSave} disabled={saving} className="flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}Save Config</button>
            </div>
          </div>

          {/* Test send */}
          <div className={cardCls}>
            <h3 className="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">Test OTP Send</h3>
            <div className="flex items-center gap-2">
              <input aria-label="test@example.com" value={testEmail} onChange={(e) => setTestEmail(e.target.value)} placeholder="test@example.com" className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" />
              <button onClick={handleTestSend} disabled={!testEmail || sending} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{sending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />} Send Test</button>
            </div>
            {testResult && <p className={`mt-2 text-sm ${testResult.includes("success") ? "text-green-600" : "text-red-600"}`}>{testResult}</p>}
          </div>
        </>
      ) : <div className={cardCls}><div className="py-12 text-center"><Mail className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No OTP configuration found.</p></div></div>}
    </div>
  );
}
