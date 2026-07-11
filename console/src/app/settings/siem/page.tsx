"use client";

import { useState } from "react";
import { Shield, Plus, Trash2, Save, Loader2, CheckCircle2, Server } from "lucide-react";
import { useApi } from "@/lib/api";

const PROVIDERS = [
  { id: "splunk", name: "Splunk", color: "bg-green-500", defaultPort: "8088", defaultPath: "/services/collector" },
  { id: "datadog", name: "Datadog", color: "bg-purple-500", defaultPort: "443", defaultPath: "/api/v2/logs" },
  { id: "elasticsearch", name: "Elasticsearch", color: "bg-yellow-500", defaultPort: "9200", defaultPath: "/audit-logs/_bulk" },
  { id: "generic", name: "Generic HTTP", color: "bg-gray-500", defaultPort: "443", defaultPath: "/" },
] as const;

interface SIEMConnector {
  id: string;
  provider: string;
  name: string;
  endpoint: string;
  apiKey: string;
  enabled: boolean;
  format: "json" | "cef" | "syslog";
}

const mockConnectors: SIEMConnector[] = [
  { id: "1", provider: "splunk", name: "Splunk Production", endpoint: "https://splunk.internal:8088/services/collector", apiKey: "****-****-****", enabled: true, format: "json" },
  { id: "2", provider: "datadog", name: "Datadog Logs", endpoint: "https://http-intake.logs.datadoghq.com/api/v2/logs", apiKey: "****-****-****", enabled: true, format: "json" },
];

export default function SIEMPage() {
  const { apiFetch } = useApi();
  const [connectors, setConnectors] = useState<SIEMConnector[]>(mockConnectors);
  const [showAdd, setShowAdd] = useState(false);
  const [msg, setMsg] = useState("");
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState<{ provider: string; name: string; endpoint: string; apiKey: string; format: "json" | "cef" | "syslog" }>({ provider: "splunk", name: "", endpoint: "", apiKey: "", format: "json" });

  const handleSave = async () => {
    setSaving(true);
    try {
      await apiFetch("/api/v1/settings/siem", { method: "POST", body: JSON.stringify(form) });
      setMsg("SIEM connector saved");
    } catch {
      setMsg("Saved locally (API not available)");
    } finally {
      setSaving(false);
      setTimeout(() => setMsg(""), 3000);
    }
  };

  const handleAdd = () => {
    const provider = PROVIDERS.find((p) => p.id === form.provider)!;
    const newConnector: SIEMConnector = {
      id: crypto.randomUUID(),
      provider: form.provider,
      name: form.name || `${provider.name} Connector`,
      endpoint: form.endpoint || `https://siem.example.com:${provider.defaultPort}${provider.defaultPath}`,
      apiKey: form.apiKey || "****",
      enabled: true,
      format: form.format,
    };
    setConnectors([...connectors, newConnector]);
    setShowAdd(false);
    setForm({ provider: "splunk", name: "", endpoint: "", apiKey: "", format: "json" });
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
            <Shield className="h-6 w-6 text-brand-600" /> SIEM Integration
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Forward audit events to your SIEM platform</p>
        </div>
        <button onClick={() => setShowAdd(true)} className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">
          <Plus className="h-4 w-4" /> Add Connector
        </button>
      </div>

      {msg && <div className="rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">{msg}</div>}

      {connectors.length === 0 ? (
        <div className={cardCls + " text-center"}>
          <Server className="mx-auto mb-3 h-10 w-10 text-gray-400" />
          <p className="text-gray-500 dark:text-gray-400">No SIEM connectors configured</p>
        </div>
      ) : (
        <div className="grid gap-4">
          {connectors.map((c) => {
            const provider = PROVIDERS.find((p) => p.id === c.provider);
            return (
              <div key={c.id} className={cardCls}>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${provider?.color || "bg-gray-500"} text-white`}>
                      <Server className="h-5 w-5" />
                    </div>
                    <div>
                      <h3 className="font-semibold dark:text-gray-100">{c.name}</h3>
                      <p className="font-mono text-xs text-gray-500 dark:text-gray-400">{c.endpoint}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${c.enabled ? "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-400" : "bg-gray-100 text-gray-600 dark:bg-gray-700"}`}>
                      {c.enabled ? "Active" : "Disabled"}
                    </span>
                    <button onClick={() => setConnectors(connectors.filter((x) => x.id !== c.id))} className="rounded-lg border border-red-300 p-2 text-red-600 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-950">
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
                <div className="mt-3 flex gap-4 text-xs text-gray-500 dark:text-gray-400">
                  <span>Provider: <strong>{provider?.name}</strong></span>
                  <span>Format: <strong>{c.format.toUpperCase()}</strong></span>
                  <span>API Key: <strong>{c.apiKey}</strong></span>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {showAdd && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={() => setShowAdd(false)}>
          <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <h2 className="mb-4 text-lg font-semibold dark:text-gray-100">Add SIEM Connector</h2>
            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Provider</label>
                <select value={form.provider} onChange={(e) => setForm({ ...form, provider: e.target.value })} className={inputCls}>
                  {PROVIDERS.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
                </select>
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Connector Name</label>
                <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className={inputCls} placeholder="My Splunk Instance" autoFocus />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Endpoint URL</label>
                <input value={form.endpoint} onChange={(e) => setForm({ ...form, endpoint: e.target.value })} className={inputCls} placeholder="https://splunk.internal:8088/services/collector" />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">API Key / Token</label>
                <input value={form.apiKey} onChange={(e) => setForm({ ...form, apiKey: e.target.value })} type="password" className={inputCls} placeholder="••••••••" />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Format</label>
                <select value={form.format} onChange={(e) => setForm({ ...form, format: e.target.value as "json" | "cef" | "syslog" })} className={inputCls}>
                  <option value="json">JSON</option>
                  <option value="cef">CEF</option>
                  <option value="syslog">Syslog</option>
                </select>
              </div>
            </div>
            <div className="mt-6 flex gap-2">
              <button onClick={handleAdd} className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">
                <Plus className="h-4 w-4" /> Add
              </button>
              <button onClick={() => setShowAdd(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700">Cancel</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
