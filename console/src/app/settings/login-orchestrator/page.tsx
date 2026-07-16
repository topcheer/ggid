"use client";

import React, { useState } from "react";
import { useApi } from "@/lib/api";
import {
  Shuffle, Loader2, AlertCircle, X, ChevronUp, ChevronDown, ToggleLeft, ToggleRight, Play, CheckCircle,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface AuthMethod {
  id: string;
  type: string;
  label: string;
  enabled: boolean;
  priority: number;
  provider_count: number;
}

interface Provider {
  id: string;
  name: string;
  type: string;
  enabled: boolean;
  domains: string[];
}

export default function LoginOrchestratorPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [methods, setMethods] = useState<AuthMethod[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [testIdentifier, setTestIdentifier] = useState("");
  const [testResult, setTestResult] = useState<{ method: string; provider: string } | null>(null);
  const [testing, setTesting] = useState(false);

  useState(() => {
    (async () => {
      try {
        const [m, p] = await Promise.all([
          apiFetch<AuthMethod[]>("/api/v1/auth/orchestrator/methods").catch(() => [
            { id: "1", type: "password", label: "Password", enabled: true, priority: 1, provider_count: 1 },
            { id: "2", type: "otp", label: "Email OTP", enabled: true, priority: 2, provider_count: 1 },
            { id: "3", type: "webauthn", label: "WebAuthn", enabled: false, priority: 3, provider_count: 0 },
            { id: "4", type: "saml", label: "SAML SSO", enabled: false, priority: 4, provider_count: 0 },
          ]),
          apiFetch<Provider[]>("/api/v1/auth/orchestrator/providers").catch(() => []),
        ]);
        setMethods(m); setProviders(p);
      } catch { setError("Failed to load orchestrator config"); }
      finally { setLoading(false); }
    })();
  });

  const moveMethod = (idx: number, dir: -1 | 1) => {
    const newIdx = idx + dir;
    if (newIdx < 0 || newIdx >= methods.length) return;
    const updated = [...methods];
    [updated[idx], updated[newIdx]] = [updated[newIdx], updated[idx]];
    setMethods(updated.map((m, i) => ({ ...m, priority: i + 1 })));
  };

  const toggleMethod = (id: string) => setMethods((p) => p.map((m) => m.id === id ? { ...m, enabled: !m.enabled } : m));
  const toggleProvider = (id: string) => setProviders((p) => p.map((pr) => pr.id === id ? { ...pr, enabled: !pr.enabled } : pr));

  const handleSave = async () => {
    setSaving(true);
    try { await apiFetch("/api/v1/auth/orchestrator/methods", { method: "PUT", body: JSON.stringify(methods) }); }
    catch { setError("Save failed"); }
    finally { setSaving(false); }
  };

  const handleTest = async () => {
    if (!testIdentifier.trim()) return;
    setTesting(true); setTestResult(null);
    try {
      const result = await apiFetch<{ method: string; provider: string }>("/api/v1/auth/orchestrator/resolve", { method: "POST", body: JSON.stringify({ identifier: testIdentifier }) });
      setTestResult(result);
    } catch { setTestResult({ method: "error", provider: "Resolution failed" }); }
    finally { setTesting(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Shuffle className="h-6 w-6 text-violet-600" /> {t("loginOrchestrator.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Configure authentication method priority, provider enablement, and identifier resolution.</p>
      </div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-violet-600" /></div>
      : (
        <>
          {/* Method priority */}
          <div className={cardCls}>
            <h3 className="mb-4 text-sm font-semibold text-gray-700 dark:text-gray-300">Auth Method Priority</h3>
            <div className="space-y-2">
              {methods.map((m, idx) => (
                <div key={m.id} className={`flex items-center justify-between rounded-lg border px-4 py-3 ${m.enabled ? "border-gray-200 dark:border-gray-700" : "border-gray-200 opacity-60 dark:border-gray-700"}`}>
                  <div className="flex items-center gap-3">
                    <div className="flex flex-col">
                      <button onClick={() => moveMethod(idx, -1)} disabled={idx === 0} className="text-gray-400 hover:text-violet-600 disabled:opacity-30"><ChevronUp className="h-3 w-3" /></button>
                      <button onClick={() => moveMethod(idx, 1)} disabled={idx === methods.length - 1} className="text-gray-400 hover:text-violet-600 disabled:opacity-30"><ChevronDown className="h-3 w-3" /></button>
                    </div>
                    <span className="flex h-7 w-7 items-center justify-center rounded-full bg-violet-100 text-xs font-bold text-violet-700 dark:bg-violet-900/30 dark:text-violet-400">{m.priority}</span>
                    <div><span className="text-sm font-medium text-gray-900 dark:text-white">{m.label}</span><span className="ml-2 text-xs text-gray-400">{m.provider_count} provider{m.provider_count !== 1 ? "s" : ""}</span></div>
                  </div>
                  <button onClick={() => toggleMethod(m.id)}>{m.enabled ? <ToggleRight className="h-6 w-6 text-green-600" /> : <ToggleLeft className="h-6 w-6 text-gray-300" />}</button>
                </div>
              ))}
            </div>
            <button onClick={handleSave} disabled={saving} className="mt-4 flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Shuffle className="h-4 w-4" />}Save Priority</button>
          </div>

          {/* Providers */}
          {providers.length > 0 && (
            <div className={cardCls}>
              <h3 className="mb-4 text-sm font-semibold text-gray-700 dark:text-gray-300">Identity Providers</h3>
              <div className="space-y-2">
                {providers.map((p) => (
                  <div key={p.id} className="flex items-center justify-between rounded-lg border border-gray-200 px-4 py-3 dark:border-gray-700">
                    <div><span className="text-sm font-medium text-gray-900 dark:text-white">{p.name}</span><span className="ml-2 rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-500 dark:bg-gray-700">{p.type}</span>{p.domains.length > 0 && <span className="ml-2 text-xs text-gray-400">{p.domains.join(", ")}</span>}</div>
                    <button onClick={() => toggleProvider(p.id)}>{p.enabled ? <ToggleRight className="h-6 w-6 text-green-600" /> : <ToggleLeft className="h-6 w-6 text-gray-300" />}</button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Identifier resolver test */}
          <div className={cardCls}>
            <h3 className="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">Test Identifier Resolver</h3>
            <div className="flex items-center gap-2">
              <input value={testIdentifier} onChange={(e) => setTestIdentifier(e.target.value)} onKeyDown={(e) => e.key === "Enter" && handleTest()} placeholder="user@example.com or username" className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" />
              <button onClick={handleTest} disabled={!testIdentifier.trim() || testing} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{testing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />} Resolve</button>
            </div>
            {testResult && (
              <div className="mt-3 flex items-center gap-2 rounded-lg bg-gray-50 px-3 py-2 text-sm dark:bg-gray-900">
                <CheckCircle className="h-4 w-4 text-green-500" />
                <span className="text-gray-600 dark:text-gray-300">Method: <span className="font-medium text-gray-900 dark:text-white">{testResult.method}</span> → Provider: <span className="font-medium text-gray-900 dark:text-white">{testResult.provider}</span></span>
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
