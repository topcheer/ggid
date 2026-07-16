"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import { Activity, Save, Loader2, Radio, Server, AlertCircle } from "lucide-react";

interface SIEMConfig {
  provider: string;
  endpoint: string;
  apiKey: string;
  indexName: string;
  batchSize: number;
  flushInterval: number;
  enabled: boolean;
}

const PROVIDERS = [
  { value: "splunk", label: "Splunk (HEC)" },
  { value: "datadog", label: "Datadog Logs" },
  { value: "elasticsearch", label: "Elasticsearch" },
  { value: "generic", label: "Generic HTTP" },
];

export default function SIEMPage() {
  const { apiFetch } = useApi();
  const t = useTranslations();
  const [config, setConfig] = useState<SIEMConfig>({
    provider: "splunk",
    endpoint: "",
    apiKey: "",
    indexName: "audit-logs",
    batchSize: 100,
    flushInterval: 5,
    enabled: false,
  });
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState("");
  const [error, setError] = useState("");
  const [testing, setTesting] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true); setError("");
    const stored = typeof window !== "undefined" ? localStorage.getItem("ggid_siem_config") : null;
    if (stored) {
      try { const parsed = JSON.parse(stored); if (parsed) setConfig(prev => ({ ...prev, ...parsed })); } catch { /* ignore */ }
    }
    fetch("/api/v1/settings/siem", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } })
      .then(async (res) => { if (res.ok) { const data = await res.json(); if (data) setConfig(data); } })
      .catch(() => { /* use stored/local defaults */ })
      .finally(() => setLoading(false));
  }, []);

  const handleSave = async () => {
    setSaving(true); setError(""); setMsg("");
    try {
      await apiFetch("/api/v1/settings/siem", { method: "POST", body: JSON.stringify(config) });
      setMsg("SIEM configuration saved");
    } catch (e) {
      localStorage.setItem("ggid_siem_config", JSON.stringify(config));
      setError(e instanceof Error ? e.message : "Failed to save SIEM configuration");
    } finally { setSaving(false); setTimeout(() => setMsg(""), 4000); }
  };

  const handleTest = async () => {
    setTesting(true); setError(""); setMsg("");
    try {
      await apiFetch("/api/v1/settings/siem/test", { method: "POST", body: JSON.stringify(config) });
      setMsg("Connection test succeeded!");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Connection test failed");
    } finally { setTesting(false); setTimeout(() => setMsg(""), 4000); }
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
          <Radio className="h-6 w-6 text-brand-600" /> SIEM Integration
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Forward audit events to your SIEM platform in real-time.
        </p>
      </div>

      {msg && (
        <div className="rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}
      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400 flex items-center gap-2">
          <AlertCircle className="h-4 w-4" /> {error}
        </div>
      )}
      {loading && <div className="flex items-center gap-2 text-sm text-gray-500"><Loader2 className="h-4 w-4 animate-spin" /> Loading SIEM configuration...</div>}

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Provider Selection */}
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
            <Server className="h-5 w-5 text-brand-600" /> Provider
          </h2>
          <div className="grid grid-cols-2 gap-3">
            {PROVIDERS.map((p) => (
              <button
                key={p.value}
                onClick={() => setConfig({ ...config, provider: p.value })}
                className={`rounded-lg border p-4 text-left transition-colors ${
                  config.provider === p.value
                    ? "border-brand-600 bg-brand-50 dark:border-brand-700 dark:bg-brand-950"
                    : "border-gray-200 hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-800"
                }`}
              >
                <p className="text-sm font-medium dark:text-gray-100">{p.label}</p>
                <p className="text-xs text-gray-500 dark:text-gray-400 capitalize">{p.value}</p>
              </button>
            ))}
          </div>
        </div>

        {/* Connection Config */}
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
            <Activity className="h-5 w-5 text-brand-600" /> Connection
          </h2>
          <div className="space-y-4">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Endpoint URL</label>
              <input
                aria-label="SIEM endpoint URL"
                value={config.endpoint}
                onChange={(e) => setConfig({ ...config, endpoint: e.target.value })}
                className={inputCls}
                placeholder="https://splunk.example.com:8088/services/collector"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">API Key / Token</label>
              <input
                aria-label="SIEM API key"
                type="password"
                value={config.apiKey}
                onChange={(e) => setConfig({ ...config, apiKey: e.target.value })}
                className={inputCls}
                placeholder="••••••••••••"
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Index Name</label>
                <input
                  aria-label="SIEM index name"
                  value={config.indexName}
                  onChange={(e) => setConfig({ ...config, indexName: e.target.value })}
                  className={inputCls}
                  placeholder="audit-logs"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">Batch Size</label>
                <input
                  aria-label="SIEM batch size"
                  type="number"
                  value={config.batchSize}
                  onChange={(e) => setConfig({ ...config, batchSize: Number(e.target.value) })}
                  className={inputCls}
                  min={1}
                  max={1000}
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Enable toggle + actions */}
      <div className="flex items-center justify-between">
        <label className="flex items-center gap-3">
          <input
            aria-label="Enable SIEM forwarding"
            type="checkbox"
            checked={config.enabled}
            onChange={(e) => setConfig({ ...config, enabled: e.target.checked })}
            className="h-4 w-4 rounded border-gray-300 text-brand-600"
          />
          <span className="text-sm font-medium dark:text-gray-200">Enable SIEM forwarding</span>
        </label>
        <div className="flex gap-2">
          <button
            aria-label="Test SIEM connection"
            onClick={handleTest}
            disabled={testing || !config.endpoint}
            className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:hover:bg-gray-700"
          >
            {testing ? <Loader2 className="h-4 w-4 animate-spin" /> : "Test Connection"}
          </button>
          <button
            aria-label="Save SIEM configuration"
            onClick={handleSave}
            disabled={saving || loading}
            className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} Save
          </button>
        </div>
      </div>
    </div>
  );
}
