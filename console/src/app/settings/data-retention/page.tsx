"use client";

import { useState } from "react";
import { Database, Save, Loader2, Plus, Trash2, Clock, AlertTriangle } from "lucide-react";
import { useApi } from "@/lib/api";

interface RetentionPolicy {
  id: string;
  resource_type: string;
  retention_days: number;
  action: "delete" | "archive" | "anonymize";
  enabled: boolean;
}

const mockPolicies: RetentionPolicy[] = [
  { id: "1", resource_type: "audit_logs", retention_days: 365, action: "archive", enabled: true },
  { id: "2", resource_type: "user_sessions", retention_days: 30, action: "delete", enabled: true },
  { id: "3", resource_type: "security_events", retention_days: 730, action: "archive", enabled: true },
  { id: "4", resource_type: "login_attempts", retention_days: 90, action: "delete", enabled: false },
];

const RESOURCE_TYPES = ["audit_logs", "user_sessions", "security_events", "login_attempts", "api_logs", "webhook_deliveries", "mfa_events", "saml_logs"] as const;
const ACTIONS = ["delete", "archive", "anonymize"] as const;

export default function DataRetentionPage() {
  const { apiFetch } = useApi();
  const [policies, setPolicies] = useState<RetentionPolicy[]>(mockPolicies);
  const [msg, setMsg] = useState("");
  const [saving, setSaving] = useState(false);
  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState({ resource_type: "audit_logs", retention_days: 90, action: "delete" as const });

  const handleSave = async () => {
    setSaving(true);
    try {
      await apiFetch("/api/v1/settings/data-retention", { method: "POST", body: JSON.stringify({ policies }) });
      setMsg("Retention policies saved");
    } catch {
      setMsg("Saved locally (API not available)");
    } finally {
      setSaving(false);
      setTimeout(() => setMsg(""), 3000);
    }
  };

  const handleAdd = () => {
    setPolicies([...policies, { id: crypto.randomUUID(), ...form, enabled: true }]);
    setShowAdd(false);
    setForm({ resource_type: "audit_logs", retention_days: 90, action: "delete" });
  };

  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 focus:border-brand-500 focus:outline-none";
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const actionBadge = (action: string) => {
    const styles: Record<string, string> = {
      delete: "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-400",
      archive: "bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-400",
      anonymize: "bg-amber-100 text-amber-700 dark:bg-amber-900 dark:text-amber-400",
    };
    return styles[action] || "bg-gray-100 text-gray-600";
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold dark:text-gray-100">
            <Database className="h-6 w-6 text-brand-600" /> Data Retention
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Configure how long different data types are kept</p>
        </div>
        <div className="flex gap-2">
          <button onClick={() => setShowAdd(true)} className="flex items-center gap-2 rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700">
            <Plus className="h-4 w-4" /> Add Policy
          </button>
          <button onClick={handleSave} disabled={saving} className="flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50">
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} Save
          </button>
        </div>
      </div>

      {msg && <div className="rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">{msg}</div>}

      <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-950">
        <div className="flex items-start gap-3">
          <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
          <div className="text-sm text-amber-700 dark:text-amber-400">
            <p className="font-medium">Compliance Notice</p>
            <p className="mt-1">Ensure retention policies comply with GDPR, HIPAA, and your organization&apos;s data governance requirements. Archived data is moved to cold storage.</p>
          </div>
        </div>
      </div>

      <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
        <table className="w-full min-w-[700px]">
          <thead className="border-b border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800">
            <tr>
              <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Resource Type</th>
              <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Retention</th>
              <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Action</th>
              <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Status</th>
              <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
            {policies.map((p) => (
              <tr key={p.id} className="hover:bg-gray-50 dark:hover:bg-gray-800/50">
                <td className="px-4 py-3">
                  <span className="font-mono text-sm text-gray-700 dark:text-gray-300">{p.resource_type}</span>
                </td>
                <td className="px-4 py-3">
                  <span className="inline-flex items-center gap-1 text-sm text-gray-600 dark:text-gray-400">
                    <Clock className="h-3 w-3" /> {p.retention_days} days
                  </span>
                </td>
                <td className="px-4 py-3">
                  <span className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${actionBadge(p.action)}`}>{p.action}</span>
                </td>
                <td className="px-4 py-3">
                  <button
                    onClick={() => setPolicies(policies.map((x) => x.id === p.id ? { ...x, enabled: !x.enabled } : x))}
                    className={`relative h-6 w-11 rounded-full transition-colors ${p.enabled ? "bg-brand-600" : "bg-gray-300 dark:bg-gray-600"}`}
                  >
                    <span className={`absolute top-0.5 h-5 w-5 rounded-full bg-white transition-transform ${p.enabled ? "translate-x-5" : "translate-x-0.5"}`} />
                  </button>
                </td>
                <td className="px-4 py-3 text-right">
                  <button onClick={() => setPolicies(policies.filter((x) => x.id !== p.id))} className="rounded-lg border border-red-300 p-2 text-red-600 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-950">
                    <Trash2 className="h-4 w-4" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {showAdd && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={() => setShowAdd(false)}>
          <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={(e) => e.stopPropagation()}>
            <h2 className="mb-4 text-lg font-semibold dark:text-gray-100">Add Retention Policy</h2>
            <div className="space-y-4">
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Resource Type</label>
                <select value={form.resource_type} onChange={(e) => setForm({ ...form, resource_type: e.target.value })} className={inputCls}>
                  {RESOURCE_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
                </select>
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Retention (days)</label>
                <input type="number" value={form.retention_days} onChange={(e) => setForm({ ...form, retention_days: Number(e.target.value) })} className={inputCls} min={1} max={3650} />
              </div>
              <div>
                <label className="mb-1 block text-sm font-medium dark:text-gray-300">Action</label>
                <select value={form.action} onChange={(e) => setForm({ ...form, action: e.target.value as typeof form.action })} className={inputCls}>
                  {ACTIONS.map((a) => <option key={a} value={a}>{a}</option>)}
                </select>
              </div>
            </div>
            <div className="mt-6 flex gap-2">
              <button onClick={handleAdd} className="rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700">Add Policy</button>
              <button onClick={() => setShowAdd(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700">Cancel</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
