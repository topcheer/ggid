"use client";
import { useState, useEffect } from "react";
import { ArrowRightLeft, Loader2, Save, Plus, Trash2, Check, X } from "lucide-react";
import { useApi } from "@/lib/api";

interface SCIMConfig {
  enabled: boolean;
  endpoint_url: string;
  auth_token: string;
  sync_interval: number;
  user_filter: string;
  group_filter: string;
  mapping: Record<string, string>;
}

// Re-export of SCIM provisioning page at /settings/scim-provisioning
// to fix 404 from settings grid navigation.
export default function SCIMProvisioningPage() {
  const { apiFetch, TENANT_ID } = useApi();
  const [config, setConfig] = useState<SCIMConfig>({
    enabled: false,
    endpoint_url: "",
    auth_token: "",
    sync_interval: 300,
    user_filter: "",
    group_filter: "",
    mapping: { userName: "username", emails: "email", displayName: "name" },
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);
  const [testStatus, setTestStatus] = useState<"idle" | "testing" | "ok" | "fail">("idle");

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  useEffect(() => {
    apiFetch<SCIMConfig>(`/api/v1/tenants/${TENANT_ID}/scim-config`)
      .then((data) => {
        if (data) setConfig({ ...config, ...data });
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const handleSave = async () => {
    setSaving(true);
    setMsg(null);
    try {
      await apiFetch(`/api/v1/tenants/${TENANT_ID}/scim-config`, {
        method: "PUT",
        body: JSON.stringify(config),
      });
      setMsg("SCIM configuration saved");
    } catch {
      setMsg("Save failed — API unavailable");
    }
    setSaving(false);
    setTimeout(() => setMsg(null), 3000);
  };

  const testConnection = async () => {
    setTestStatus("testing");
    try {
      const resp = await apiFetch<{ status?: string }>(`/api/v1/tenants/${TENANT_ID}/scim-config/test`, {
        method: "POST",
      });
      setTestStatus(resp?.status === "ok" ? "ok" : "fail");
    } catch {
      setTestStatus("fail");
    }
    setTimeout(() => setTestStatus("idle"), 3000);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-brand-500" />
      </div>
    );
  }

  return (
    <div className="max-w-3xl space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white dark:text-white">
          <ArrowRightLeft className="h-6 w-6 text-cyan-500" /> SCIM Provisioning
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Configure SCIM 2.0 user provisioning for automatic identity sync.
        </p>
      </div>

      <div className={card}>
        <div className="space-y-4">
          <label className="flex items-center justify-between">
            <div>
              <span className="text-sm font-medium">Enable SCIM Provisioning</span>
              <p className="text-xs text-gray-400">Automatically sync users and groups from your IdP</p>
            </div>
            <button
              type="button"
              onClick={() => setConfig({ ...config, enabled: !config.enabled })}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition ${config.enabled ? "bg-cyan-600" : "bg-gray-300 dark:bg-gray-600"}`}
            >
              <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition ${config.enabled ? "translate-x-6" : "translate-x-1"}`} />
            </button>
          </label>

          <div>
            <label className="text-sm font-medium">SCIM Endpoint URL</label>
            <input
              type="text"
              value={config.endpoint_url}
              onChange={e => setConfig({ ...config, endpoint_url: e.target.value })}
              placeholder="https://idp.example.com/scim/v2"
              className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"
            />
          </div>

          <div>
            <label className="text-sm font-medium">Bearer Token</label>
            <input
              type="password"
              value={config.auth_token}
              onChange={e => setConfig({ ...config, auth_token: e.target.value })}
              placeholder="••••••••••••"
              className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"
            />
          </div>

          <div>
            <label className="text-sm font-medium">Sync Interval (seconds)</label>
            <input
              type="number"
              value={config.sync_interval}
              onChange={e => setConfig({ ...config, sync_interval: parseInt(e.target.value) || 300 })}
              className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"
            />
          </div>

          <div className="flex gap-3">
            <button onClick={handleSave} disabled={saving} className="flex items-center gap-2 rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white hover:bg-cyan-700 disabled:opacity-50">
              {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} Save Configuration
            </button>
            <button onClick={testConnection} disabled={testStatus === "testing"} className="flex items-center gap-2 rounded-lg border border-gray-300 dark:border-gray-600 px-4 py-2 text-sm font-medium dark:border-gray-700">
              {testStatus === "testing" ? <Loader2 className="h-4 w-4 animate-spin" /> : testStatus === "ok" ? <Check className="h-4 w-4 text-green-500" /> : testStatus === "fail" ? <X className="h-4 w-4 text-red-500" /> : <ArrowRightLeft className="h-4 w-4" />}
              Test Connection
            </button>
          </div>
          {msg && <p className="text-sm text-green-600">{msg}</p>}
        </div>
      </div>
    </div>
  );
}