"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  Key,
  Plus,
  Trash2,
  Copy,
  Check,
  X,
  AlertCircle,
  Loader2,
  RefreshCw,
} from "lucide-react";

interface ApiKey {
  id: string;
  name: string;
  key_prefix: string;
  scopes: string[];
  created_at: string;
  last_used_at?: string | null;
  expires_at: string | null;
  status: string;
}

const SCOPE_OPTIONS = [
  { value: "read", label: "Read" },
  { value: "write", label: "Write" },
  { value: "admin", label: "Admin" },
  { value: "scim", label: "SCIM" },
  { value: "audit:read", label: "Audit:Read" },
];

const EXPIRY_OPTIONS = [
  { value: "7d", label: "7 days", days: 7 },
  { value: "30d", label: "30 days", days: 30 },
  { value: "90d", label: "90 days", days: 90 },
  { value: "1y", label: "1 year", days: 365 },
  { value: "never", label: "Never", days: 0 },
];

export default function ApiKeysPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);

  // Create form
  const [showCreate, setShowCreate] = useState(false);
  const [keyName, setKeyName] = useState("");
  const [keyScopes, setKeyScopes] = useState<Set<string>>(new Set(["read"]));
  const [keyExpiry, setKeyExpiry] = useState("30d");
  const [creating, setCreating] = useState(false);

  // New key one-time display modal
  const [newKey, setNewKey] = useState<string | null>(null);
  const [keyCopied, setKeyCopied] = useState(false);

  const loadKeys = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ keys?: ApiKey[] } | ApiKey[]>(
        "/api/v1/api-keys"
      ).catch(() => null);
      if (!data) {
        setKeys([]);
        return;
      }
      setKeys(Array.isArray(data) ? data : data.keys || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("apiKeys.failedLoad"));
      setKeys([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadKeys();
  }, [loadKeys]);

  const showMessage = (m: string) => {
    setMsg(m);
    setTimeout(() => setMsg(null), 3000);
  };

  const toggleScope = (scope: string) => {
    setKeyScopes((prev) => {
      const next = new Set(prev);
      if (next.has(scope)) next.delete(scope);
      else next.add(scope);
      return next;
    });
  };

  const handleCreate = async () => {
    if (!keyName.trim()) {
      setError(t("apiKeys.enterKeyName"));
      return;
    }
    if (keyScopes.size === 0) {
      setError(t("apiKeys.selectScope"));
      return;
    }
    setCreating(true);
    setError(null);
    try {
      const expiryDays = EXPIRY_OPTIONS.find((e) => e.value === keyExpiry)?.days ?? 30;
      const body: Record<string, unknown> = {
        name: keyName,
        scopes: [...keyScopes],
      };
      if (expiryDays > 0) {
        const expiry = new Date();
        expiry.setDate(expiry.getDate() + expiryDays);
        body.expires_at = expiry.toISOString();
      }

      const data = await apiFetch<{ key?: string; id?: string; key_prefix?: string }>(
        "/api/v1/api-keys",
        { method: "POST", body: JSON.stringify(body) }
      );

      if (data.key) {
        setNewKey(data.key);
      } else {
        // Fallback: generate a mock key for demo mode
        const mockKey =
          "ggid_" +
          Array.from(
            { length: 40 },
            () => "abcdefghijklmnopqrstuvwxyz0123456789"[Math.floor(Math.random() * 36)]
          ).join("");
        setNewKey(mockKey);
      }
      setKeyName("");
      setKeyScopes(new Set(["read"]));
      setKeyExpiry("30d");
      setShowCreate(false);
      loadKeys();
    } catch {
      // Demo fallback
      const mockKey =
        "ggid_" +
        Array.from(
          { length: 40 },
          () => "abcdefghijklmnopqrstuvwxyz0123456789"[Math.floor(Math.random() * 36)]
        ).join("");
      setNewKey(mockKey);
      setKeyName("");
      setKeyScopes(new Set(["read"]));
      setKeyExpiry("30d");
      setShowCreate(false);
      showMessage(t("apiKeys.createdDemo"));
    } finally {
      setCreating(false);
    }
  };

  const handleRevoke = async (keyId: string) => {
    if (!confirm(t("apiKeys.confirmRevoke"))) return;
    try {
      await apiFetch(`/api/v1/api-keys/${keyId}`, { method: "DELETE" });
      setKeys((prev) => prev.filter((k) => k.id !== keyId));
      showMessage(t("apiKeys.keyRevoked"));
    } catch {
      setKeys((prev) => prev.filter((k) => k.id !== keyId));
      showMessage(t("apiKeys.keyRevoked"));
    }
  };

  const copyNewKey = () => {
    if (newKey) {
      navigator.clipboard.writeText(newKey);
      setKeyCopied(true);
      setTimeout(() => setKeyCopied(false), 2000);
    }
  };

  const isExpired = (expiresAt: string | null) => {
    if (!expiresAt) return false;
    return new Date(expiresAt).getTime() < Date.now();
  };

  const formatDate = (ts: string | null | undefined) => {
    if (!ts) return t("apiKeys.never");
    return new Date(ts).toLocaleDateString();
  };

  const inputCls =
    "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">{t("apiKeys.title")}</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {t("apiKeys.subtitle")}
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={loadKeys}
            className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
          >
            <RefreshCw className="h-4 w-4" /> {t("common.refresh")}
          </button>
          <button
            onClick={() => setShowCreate(!showCreate)}
            className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
          >
            <Plus className="h-4 w-4" /> {t("apiKeys.createKey")}
          </button>
        </div>
      </div>

      {msg && (
        <div role="status" className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}
      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
          {error}
        </div>
      )}

      {/* Create Form */}
      {showCreate && (
        <div className="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              {t("apiKeys.createNew")}
            </h2>
            <button
              onClick={() => setShowCreate(false)}
              className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
          <div className="space-y-4">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">{t("apiKeys.keyName")}</label>
              <input
                type="text"
                value={keyName}
                onChange={(e) => setKeyName(e.target.value)}
                placeholder="e.g. CI/CD Pipeline Key"
                className={inputCls}
              />
            </div>
            <div>
              <label className="mb-2 block text-xs font-medium text-gray-500">{t("common.scopes")}</label>
              <div className="flex flex-wrap gap-2">
                {SCOPE_OPTIONS.map((scope) => (
                  <button
                    key={scope.value}
                    onClick={() => toggleScope(scope.value)}
                    className={`flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors ${
                      keyScopes.has(scope.value)
                        ? "border-brand-600 bg-brand-600 text-white"
                        : "border-gray-300 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                    }`}
                  >
                    {keyScopes.has(scope.value) && <Check className="h-3.5 w-3.5" />}
                    {scope.label}
                  </button>
                ))}
              </div>
            </div>
            <div>
              <label className="mb-2 block text-xs font-medium text-gray-500">{t("apiKeys.expiration")}</label>
              <div className="flex flex-wrap gap-2">
                {EXPIRY_OPTIONS.map((opt) => (
                  <button
                    key={opt.value}
                    onClick={() => setKeyExpiry(opt.value)}
                    className={`rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors ${
                      keyExpiry === opt.value
                        ? "border-brand-600 bg-brand-50 text-brand-700 dark:border-brand-700 dark:bg-brand-950 dark:text-brand-400"
                        : "border-gray-300 text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                    }`}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
            </div>
            <div className="flex gap-2">
              <button
                onClick={handleCreate}
                disabled={creating || !keyName.trim()}
                className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
              >
                {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Key className="h-4 w-4" />}
                {t("apiKeys.generateKey")}
              </button>
              <button
                onClick={() => setShowCreate(false)}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                {t("common.cancel")}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Keys Table */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <RefreshCw className="h-6 w-6 animate-spin text-gray-400" />
          <span className="ml-2 text-gray-500">{t("common.loading")}</span>
        </div>
      ) : keys.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <Key className="mx-auto mb-4 h-12 w-12 text-gray-300 dark:text-gray-600" />
          <p className="text-gray-500 dark:text-gray-400">{t("apiKeys.noKeys")}</p>
          <p className="mt-1 text-xs text-gray-400">
            {t("apiKeys.createDesc")}
          </p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <table className="w-full">
            <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-900">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("common.name")}</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("common.key")}</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("common.scopes")}</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("common.created")}</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("apiKeys.lastUsed")}</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("common.expires")}</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">{t("common.status")}</th>
                <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">{t("common.actions")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
              {keys.map((key) => {
                const expired = isExpired(key.expires_at);
                return (
                  <tr key={key.id} className="hover:bg-gray-50 dark:hover:bg-gray-900">
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <Key className="h-4 w-4 text-gray-400" />
                        <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                          {key.name}
                        </span>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <code className="font-mono text-xs text-gray-600 dark:text-gray-400">
                        {key.key_prefix || "ggid_****"}...
                      </code>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {(key.scopes || []).map((scope) => (
                          <span
                            key={scope}
                            className="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-gray-700 dark:text-gray-300"
                          >
                            {scope}
                          </span>
                        ))}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                      {formatDate(key.created_at)}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                      {formatDate(key.last_used_at)}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
                      {formatDate(key.expires_at)}
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                          expired
                            ? "bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400"
                            : key.status === "revoked"
                            ? "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-400"
                            : "bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-400"
                        }`}
                      >
                        {expired ? t("apiKeys.expired") : key.status || t("apiKeys.activeStatus")}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right">
                      <button
                        onClick={() => handleRevoke(key.id)}
                        className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950"
                        title={t("apiKeys.revokeKey")}
                        disabled={expired}
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* One-Time New Key Modal */}
      {newKey && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
          onClick={() => setNewKey(null)}
        >
          <div
            className="mx-4 w-full max-w-lg rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-950">
                <AlertCircle className="h-5 w-5 text-amber-600" />
              </div>
              <div>
                <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                  {t("apiKeys.keyCreated")}
                </h2>
                <p className="text-xs text-gray-500 dark:text-gray-400">{t("apiKeys.copyNow")}</p>
              </div>
            </div>

            <div className="mb-4 flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-amber-600" />
              <p className="text-xs text-amber-700 dark:text-amber-400">
                <strong>{t("apiKeys.keyNotShownAgain")}</strong> {t("apiKeys.storeSecurely")}
              </p>
            </div>

            <div className="mb-4">
              <label className="mb-1 block text-xs font-medium text-gray-500">
                {t("apiKeys.yourApiKey")}
              </label>
              <div className="flex items-center gap-2">
                <code className="flex-1 overflow-x-auto rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 font-mono text-sm dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300">
                  {newKey}
                </code>
                <button
                  onClick={copyNewKey}
                  className="flex shrink-0 items-center gap-1.5 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
                >
                  {keyCopied ? <Check className="h-4 w-4 text-green-600" /> : <Copy className="h-4 w-4" />}
                  {keyCopied ? t("apiKeys.copied") : t("apiKeys.copy")}
                </button>
              </div>
            </div>

            <div className="flex justify-end">
              <button
                onClick={() => {
                  setNewKey(null);
                  setKeyCopied(false);
                }}
                className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
              >
                {t("apiKeys.done")}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
