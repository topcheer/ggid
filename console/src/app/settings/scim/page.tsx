"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  RefreshCw,
  Copy,
  Check,
  Users,
  Group,
  Loader2,
  Server,
  AlertTriangle,
  CheckCircle2,
} from "lucide-react";

interface SCIMConfig {
  endpoint: string;
  bearerToken: string;
  enabled: boolean;
}

interface SyncStatus {
  resourceType: "users" | "groups";
  lastSync: string;
  totalRecords: number;
  syncedRecords: number;
  failedRecords: number;
  status: "idle" | "syncing" | "success" | "error";
  errorMessage?: string;
}

export default function SCIMPage() {
  const { apiFetch } = useApi();
  const [config, setConfig] = useState<SCIMConfig>({
    endpoint: "",
    bearerToken: "",
    enabled: false,
  });
  const [syncStatus, setSyncStatus] = useState<SyncStatus[]>([
    { resourceType: "users", lastSync: "—", totalRecords: 0, syncedRecords: 0, failedRecords: 0, status: "idle" },
    { resourceType: "groups", lastSync: "—", totalRecords: 0, syncedRecords: 0, failedRecords: 0, status: "idle" },
  ]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState("");
  const [copied, setCopied] = useState(false);
  const [syncing, setSyncing] = useState<string | null>(null);

  useEffect(() => {
    const load = async () => {
      try {
        const data = await apiFetch<SCIMConfig & { syncStatus?: SyncStatus[] }>("/api/v1/settings/scim");
        setConfig({
          endpoint: data.endpoint || `${window.location.origin}/api/v1/scim/v2`,
          bearerToken: data.bearerToken || "",
          enabled: data.enabled ?? false,
        });
        if (data.syncStatus) setSyncStatus(data.syncStatus);
      } catch {
        setConfig((prev) => ({
          ...prev,
          endpoint: `${window.location.origin}/api/v1/scim/v2`,
        }));
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const handleSave = async () => {
    setSaving(true);
    try {
      await apiFetch("/api/v1/settings/scim", {
        method: "POST",
        body: JSON.stringify(config),
      });
      setMsg("SCIM configuration saved");
    } catch {
      localStorage.setItem("ggid_scim_config", JSON.stringify(config));
      setMsg("SCIM configuration saved (offline mode)");
    } finally {
      setSaving(false);
      setTimeout(() => setMsg(""), 4000);
    }
  };

  const handleSync = async (resourceType: "users" | "groups") => {
    setSyncing(resourceType);
    setSyncStatus((prev) =>
      prev.map((s) => (s.resourceType === resourceType ? { ...s, status: "syncing" } : s))
    );
    try {
      const data = await apiFetch<SyncStatus>(`/api/v1/settings/scim/sync`, {
        method: "POST",
        body: JSON.stringify({ resourceType }),
      });
      setSyncStatus((prev) =>
        prev.map((s) => (s.resourceType === resourceType ? { ...data, status: "success" } : s))
      );
      setMsg(`${resourceType} sync completed`);
    } catch {
      setSyncStatus((prev) =>
        prev.map((s) =>
          s.resourceType === resourceType
            ? { ...s, status: "error", errorMessage: "Sync failed — endpoint unavailable" }
            : s
        )
      );
      setMsg(`${resourceType} sync failed`);
    } finally {
      setSyncing(null);
      setTimeout(() => setMsg(""), 4000);
    }
  };

  const handleCopy = () => {
    navigator.clipboard.writeText(config.endpoint);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleRegenerateToken = () => {
    const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
    let token = "";
    for (let i = 0; i < 48; i++) token += chars[Math.floor(Math.random() * chars.length)];
    setConfig({ ...config, bearerToken: token });
  };

  const inputCls =
    "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const cardCls =
    "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const statusIcon = (status: SyncStatus["status"]) => {
    switch (status) {
      case "success":
        return <CheckCircle2 className="h-4 w-4 text-green-500" />;
      case "error":
        return <AlertTriangle className="h-4 w-4 text-red-500" />;
      case "syncing":
        return <Loader2 className="h-4 w-4 animate-spin text-indigo-500" />;
      default:
        return <div className="h-4 w-4 rounded-full border-2 border-gray-300" />;
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Server className="h-7 w-7 text-indigo-600" />
          SCIM Provisioning
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Configure SCIM 2.0 endpoint for automated user and group provisioning.
        </p>
      </div>

      {loading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
        </div>
      ) : (
        <>
          {/* Endpoint info */}
          <div className={cardCls}>
            <h3 className="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">
              SCIM Endpoint
            </h3>
            <div className="flex items-center gap-2">
              <code className="flex-1 truncate rounded-lg bg-gray-100 px-3 py-2 text-sm text-gray-700 dark:bg-gray-900 dark:text-gray-300">
                {config.endpoint}
              </code>
              <button
                onClick={handleCopy}
                className="rounded-lg border border-gray-300 p-2 text-gray-500 hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
              >
                {copied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
              </button>
            </div>
            <p className="mt-2 text-xs text-gray-400">
              Configure your IdP (Okta, Azure AD, Google Workspace) to send SCIM requests to this endpoint.
            </p>
          </div>

          {/* Auth config */}
          <div className={cardCls}>
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300">Authentication</h3>
              <button
                onClick={() => setConfig({ ...config, enabled: !config.enabled })}
                className={`flex h-6 w-11 items-center rounded-full transition-colors ${
                  config.enabled ? "bg-indigo-600" : "bg-gray-300 dark:bg-gray-600"
                }`}
              >
                <span
                  className={`h-5 w-5 transform rounded-full bg-white shadow transition-transform ${
                    config.enabled ? "translate-x-5" : "translate-x-0.5"
                  }`}
                />
              </button>
            </div>
            <div className="space-y-3">
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">Bearer Token</label>
                <div className="flex gap-2">
                  <input
                    className={inputCls}
                    type="password"
                    placeholder="SCIM bearer token"
                    value={config.bearerToken}
                    onChange={(e) => setConfig({ ...config, bearerToken: e.target.value })}
                  />
                  <button
                    onClick={handleRegenerateToken}
                    className="shrink-0 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                  >
                    <RefreshCw className="h-4 w-4" />
                  </button>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
                >
                  {saving ? <Loader2 className="mr-1 inline h-4 w-4 animate-spin" /> : null}
                  Save Configuration
                </button>
                {msg && <span className="text-sm text-green-600">{msg}</span>}
              </div>
            </div>
          </div>

          {/* Sync status */}
          <div className={cardCls}>
            <h3 className="mb-4 text-sm font-semibold text-gray-700 dark:text-gray-300">Sync Status</h3>
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              {syncStatus.map((sync) => (
                <div
                  key={sync.resourceType}
                  className="rounded-lg border border-gray-200 p-4 dark:border-gray-700"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      {sync.resourceType === "users" ? (
                        <Users className="h-5 w-5 text-indigo-500" />
                      ) : (
                        <Group className="h-5 w-5 text-indigo-500" />
                      )}
                      <span className="font-medium capitalize text-gray-800 dark:text-gray-200">
                        {sync.resourceType}
                      </span>
                    </div>
                    {statusIcon(sync.status)}
                  </div>
                  <div className="mt-3 space-y-1 text-sm">
                    <div className="flex justify-between">
                      <span className="text-gray-400">Total:</span>
                      <span className="text-gray-700 dark:text-gray-300">{sync.totalRecords}</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-gray-400">Synced:</span>
                      <span className="text-green-600">{sync.syncedRecords}</span>
                    </div>
                    {sync.failedRecords > 0 && (
                      <div className="flex justify-between">
                        <span className="text-gray-400">Failed:</span>
                        <span className="text-red-600">{sync.failedRecords}</span>
                      </div>
                    )}
                    <div className="flex justify-between">
                      <span className="text-gray-400">Last sync:</span>
                      <span className="text-gray-700 dark:text-gray-300">{sync.lastSync}</span>
                    </div>
                    {sync.errorMessage && (
                      <p className="mt-1 text-xs text-red-500">{sync.errorMessage}</p>
                    )}
                  </div>
                  <button
                    onClick={() => handleSync(sync.resourceType)}
                    disabled={syncing === sync.resourceType}
                    className="mt-3 flex w-full items-center justify-center gap-1 rounded-lg border border-gray-300 py-2 text-sm font-medium text-gray-600 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                  >
                    {syncing === sync.resourceType ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <RefreshCw className="h-4 w-4" />
                    )}
                    Sync Now
                  </button>
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
