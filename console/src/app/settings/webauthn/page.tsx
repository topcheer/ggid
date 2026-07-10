"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import { Fingerprint, Save, KeyRound, Loader2 } from "lucide-react";

interface WebAuthnConfig {
  rp_id: string;
  rp_name: string;
  origins: string;
  timeout: number;
  attestation: "none" | "indirect" | "direct";
}

interface Credential {
  id: string;
  name: string;
  type: string;
  created_at: string;
}

const STORAGE_KEY = "ggid_webauthn_config";

const defaultConfig: WebAuthnConfig = {
  rp_id: "localhost",
  rp_name: "GGID",
  origins: "http://localhost:3000",
  timeout: 60000,
  attestation: "none",
};

export default function WebAuthnSettingsPage() {
  const { apiFetch } = useApi();
  const [config, setConfig] = useState<WebAuthnConfig>(defaultConfig);
  const [userId, setUserId] = useState("");
  const [credentials, setCredentials] = useState<Credential[]>([]);
  const [credLoading, setCredLoading] = useState(false);
  const [credError, setCredError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  // Load config from localStorage or API
  useEffect(() => {
    const stored = typeof window !== "undefined" ? localStorage.getItem(STORAGE_KEY) : null;
    if (stored) {
      try {
        setConfig({ ...defaultConfig, ...JSON.parse(stored) });
      } catch {
        // ignore parse errors
      }
    }
  }, []);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const handleSave = async () => {
    setSaving(true);
    try {
      await apiFetch("/api/v1/settings/webauthn", {
        method: "POST",
        body: JSON.stringify(config),
      });
      setMsg("WebAuthn settings saved to server");
    } catch {
      // Fallback: save to localStorage
      localStorage.setItem(STORAGE_KEY, JSON.stringify(config));
      setMsg("Endpoint unavailable — saved to localStorage");
    } finally {
      setSaving(false);
    }
  };

  const fetchCredentials = useCallback(async () => {
    if (!userId) return;
    setCredLoading(true);
    setCredError(null);
    try {
      const data = await apiFetch<{ credentials?: Credential[]; items?: Credential[] }>(
        `/api/v1/users/${userId}/credentials`,
      );
      setCredentials(data.credentials || data.items || []);
    } catch (err) {
      setCredError(err instanceof Error ? err.message : "Failed to load credentials");
      setCredentials([]);
    } finally {
      setCredLoading(false);
    }
  }, [apiFetch, userId]);

  useEffect(() => {
    if (userId) fetchCredentials();
  }, [userId, fetchCredentials]);

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
          <Fingerprint className="h-6 w-6 text-brand-600" /> WebAuthn Settings
        </h1>
      </div>

      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}

      <div className="space-y-6">
        {/* Configuration */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold dark:text-gray-100">Relying Party Configuration</h2>
            <button
              onClick={handleSave}
              disabled={saving}
              className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
            >
              {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} Save
            </button>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">RP ID (Domain)</label>
              <input
                value={config.rp_id}
                onChange={(e) => setConfig({ ...config, rp_id: e.target.value })}
                placeholder="example.com"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
              <p className="mt-1 text-xs text-gray-400">The domain name of the relying party</p>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">RP Name</label>
              <input
                value={config.rp_name}
                onChange={(e) => setConfig({ ...config, rp_name: e.target.value })}
                placeholder="GGID"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
              <p className="mt-1 text-xs text-gray-400">Display name shown to users</p>
            </div>
            <div className="sm:col-span-2">
              <label className="mb-1 block text-xs font-medium text-gray-500">Origins (comma-separated)</label>
              <textarea
                value={config.origins}
                onChange={(e) => setConfig({ ...config, origins: e.target.value })}
                rows={2}
                placeholder="https://example.com, https://app.example.com"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
              <p className="mt-1 text-xs text-gray-400">Allowed origins for WebAuthn requests</p>
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Timeout (ms)</label>
              <input
                type="number"
                value={config.timeout}
                onChange={(e) => setConfig({ ...config, timeout: parseInt(e.target.value) || 60000 })}
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Attestation Preference</label>
              <select
                value={config.attestation}
                onChange={(e) =>
                  setConfig({ ...config, attestation: e.target.value as WebAuthnConfig["attestation"] })
                }
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              >
                <option value="none">none</option>
                <option value="indirect">indirect</option>
                <option value="direct">direct</option>
              </select>
              <p className="mt-1 text-xs text-gray-400">
                {config.attestation === "none"
                  ? "No attestation data required (recommended for most deployments)"
                  : config.attestation === "indirect"
                    ? "Anonymized attestation from a trusted CA"
                    : "Full attestation from the authenticator"}
              </p>
            </div>
          </div>
        </div>

        {/* Credentials List */}
        <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold dark:text-gray-100">
            <KeyRound className="h-5 w-5 text-brand-600" /> Registered Credentials
          </h2>
          <div className="mb-4">
            <label className="mb-1 block text-xs font-medium text-gray-500">User ID</label>
            <div className="flex gap-2">
              <input
                value={userId}
                onChange={(e) => setUserId(e.target.value)}
                placeholder="Enter user UUID to list credentials"
                className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200"
              />
              <button
                onClick={fetchCredentials}
                disabled={!userId || credLoading}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700 disabled:opacity-50"
              >
                {credLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : "Fetch"}
              </button>
            </div>
          </div>

          {credError && (
            <div className="mb-3 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">{credError}</div>
          )}

          {credentials.length > 0 ? (
            <div className="overflow-hidden rounded-lg border border-gray-200 dark:border-gray-700">
              <table className="w-full">
                <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-900">
                  <tr>
                    <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">Name</th>
                    <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">Type</th>
                    <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">Created</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                  {credentials.map((cred) => (
                    <tr key={cred.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                      <td className="px-4 py-2 text-sm font-medium text-gray-900 dark:text-gray-200">{cred.name}</td>
                      <td className="px-4 py-2">
                        <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700 dark:bg-blue-900 dark:text-blue-300">
                          {cred.type}
                        </span>
                      </td>
                      <td className="px-4 py-2 text-sm text-gray-500">
                        {cred.created_at ? new Date(cred.created_at).toLocaleDateString() : "-"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            !credLoading && !credError && userId && (
              <p className="py-4 text-center text-sm text-gray-400">No credentials found for this user</p>
            )
          )}

          {!userId && (
            <p className="py-4 text-center text-sm text-gray-400">Enter a user ID to view registered credentials</p>
          )}
        </div>
      </div>
    </div>
  );
}
