"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import { Archive, Save, Loader2, Plus, Trash2, Clock } from "lucide-react";

interface RetentionPolicy {
  id: string;
  name: string;
  dataType: string;
  retentionDays: number;
  action: string;
  enabled: boolean;
}

const DATA_TYPES = [
  { value: "audit_events", label: "Audit Events" },
  { value: "user_sessions", label: "User Sessions" },
  { value: "login_logs", label: "Login Logs" },
  { value: "api_logs", label: "API Logs" },
  { value: "failed_auth", label: "Failed Auth Attempts" },
];

const ACTIONS = [
  { value: "delete", label: "Delete" },
  { value: "archive", label: "Archive (cold storage)" },
  { value: "anonymize", label: "Anonymize PII" },
];

export default function DataRetentionPage() {
  const { apiFetch } = useApi();
  const t = useTranslations();
  const [policies, setPolicies] = useState<RetentionPolicy[]>([
    { id: "1", name: "Default Audit Events", dataType: "audit_events", retentionDays: 365, action: "archive", enabled: true },
    { id: "2", name: "Session Data", dataType: "user_sessions", retentionDays: 30, action: "delete", enabled: true },
    { id: "3", name: "Failed Login Logs", dataType: "failed_auth", retentionDays: 90, action: "delete", enabled: true },
  ]);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState("");

  const handleSave = async () => {
    setSaving(true);
    try {
      await apiFetch("/api/v1/settings/data-retention", {
        method: "POST",
        body: JSON.stringify({ policies }),
      });
      setMsg("Retention policies saved");
    } catch {
      localStorage.setItem("ggid_retention_policies", JSON.stringify(policies));
      setMsg("Retention policies saved (offline mode)");
    } finally {
      setSaving(false);
      setTimeout(() => setMsg(""), 4000);
    }
  };

  const addPolicy = () => {
    setPolicies([
      ...policies,
      { id: crypto.randomUUID(), name: "New Policy", dataType: "audit_events", retentionDays: 90, action: "delete", enabled: true },
    ]);
  };

  const removePolicy = (id: string) => {
    setPolicies(policies.filter((p) => p.id !== id));
  };

  const updatePolicy = (id: string, updates: Partial<RetentionPolicy>) => {
    setPolicies(policies.map((p) => (p.id === id ? { ...p, ...updates } : p)));
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
            <Archive className="h-6 w-6 text-brand-600" /> Data Retention
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Configure how long different data types are retained.
          </p>
        </div>
        <button
          onClick={addPolicy}
          className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700"
        >
          <Plus className="h-4 w-4" /> Add Policy
        </button>
      </div>

      {msg && (
        <div className="rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}

      {/* Compliance summary */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div className={cardCls}>
          <p className="text-xs font-medium uppercase text-gray-500">GDPR Retention</p>
          <p className="mt-2 text-2xl font-bold dark:text-gray-100">{policies.filter((p) => p.enabled).length}</p>
          <p className="text-xs text-gray-400">active policies</p>
        </div>
        <div className={cardCls}>
          <p className="text-xs font-medium uppercase text-gray-500">Longest Retention</p>
          <p className="mt-2 text-2xl font-bold dark:text-gray-100">
            {Math.max(0, ...policies.filter((p) => p.enabled).map((p) => p.retentionDays))}
          </p>
          <p className="text-xs text-gray-400">days</p>
        </div>
        <div className={cardCls}>
          <p className="text-xs font-medium uppercase text-gray-500">Auto-Delete</p>
          <p className="mt-2 text-2xl font-bold dark:text-gray-100">
            {policies.filter((p) => p.enabled && p.action === "delete").length}
          </p>
          <p className="text-xs text-gray-400">policies</p>
        </div>
      </div>

      {/* Policy table */}
      <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
        <table className="w-full min-w-[800px]">
          <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800">
            <tr>
              <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Policy Name</th>
              <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Data Type</th>
              <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Retention</th>
              <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Action</th>
              <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Enabled</th>
              <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
            {policies.map((policy) => (
              <tr key={policy.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                <td className="px-4 py-3">
                  <input
                    value={policy.name}
                    onChange={(e) => updatePolicy(policy.id, { name: e.target.value })}
                    className={inputCls}
                  />
                </td>
                <td className="px-4 py-3">
                  <select
                    value={policy.dataType}
                    onChange={(e) => updatePolicy(policy.id, { dataType: e.target.value })}
                    className={inputCls}
                  >
                    {DATA_TYPES.map((dt) => (
                      <option key={dt.value} value={dt.value}>{dt.label}</option>
                    ))}
                  </select>
                </td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-2">
                    <input
                      type="number"
                      value={policy.retentionDays}
                      onChange={(e) => updatePolicy(policy.id, { retentionDays: Number(e.target.value) })}
                      className={`${inputCls} w-20`}
                      min={1}
                    />
                    <Clock className="h-4 w-4 text-gray-400" />
                    <span className="text-xs text-gray-500">days</span>
                  </div>
                </td>
                <td className="px-4 py-3">
                  <select
                    value={policy.action}
                    onChange={(e) => updatePolicy(policy.id, { action: e.target.value })}
                    className={inputCls}
                  >
                    {ACTIONS.map((a) => (
                      <option key={a.value} value={a.value}>{a.label}</option>
                    ))}
                  </select>
                </td>
                <td className="px-4 py-3">
                  <input
                    type="checkbox"
                    checked={policy.enabled}
                    onChange={(e) => updatePolicy(policy.id, { enabled: e.target.checked })}
                    className="h-4 w-4 rounded border-gray-300 text-brand-600"
                  />
                </td>
                <td className="px-4 py-3 text-right">
                  <button
                    onClick={() => removePolicy(policy.id)}
                    title="Delete policy"
                    className="rounded-lg p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="flex justify-end">
        <button
          onClick={handleSave}
          disabled={saving}
          className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
        >
          {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} Save Policies
        </button>
      </div>
    </div>
  );
}
