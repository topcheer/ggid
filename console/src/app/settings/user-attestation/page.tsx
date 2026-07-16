"use client";

import React, { useState } from "react";
import { useApi } from "@/lib/api";
import {
  FileText, Loader2, AlertCircle, X, Download, Trash2, FileDown, Calendar, FileCheck, Plus, Save,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface AttestationRecord {
  id: string;
  user_id: string;
  username: string;
  pending_fields: string[];
  sent_at: string;
  expires_at: string;
  status: "pending" | "completed" | "expired";
}

interface AttestationConfig {
  enabled: boolean;
  frequency_days: number;
  required_fields: string[];
  reminder_days_before: number;
}

const statusColors: Record<string, string> = {
  pending: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400",
  completed: "text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400",
  expired: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
};

export default function UserAttestationPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [records, setRecords] = useState<AttestationRecord[]>([]);
  const [config, setConfig] = useState<AttestationConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [reminding, setReminding] = useState<string | null>(null);
  const [fieldInput, setFieldInput] = useState("");

  useState(() => {
    (async () => {
      try {
        const [r, c] = await Promise.all([
          apiFetch<AttestationRecord[]>("/api/v1/users/attestation/pending").catch(() => []),
          apiFetch<AttestationConfig>("/api/v1/users/attestation/config").catch(() => null),
        ]);
        setRecords(r); setConfig(c);
      } catch { setError("Failed to load attestation data"); }
      finally { setLoading(false); }
    })();
  });

  const handleSave = async () => {
    if (!config) return;
    setSaving(true);
    try { await apiFetch("/api/v1/users/attestation/config", { method: "PUT", body: JSON.stringify(config) }); }
    catch { setError("Save failed"); }
    finally { setSaving(false); }
  };

  const handleReminder = async (id: string) => {
    setReminding(id);
    try { await apiFetch(`/api/v1/users/attestation/${id}/remind`, { method: "POST" }); }
    catch { setError("Reminder failed"); }
    finally { setReminding(null); }
  };

  const addField = () => {
    if (!config || !fieldInput.trim()) return;
    setConfig({ ...config, required_fields: [...config.required_fields, fieldInput.trim()] });
    setFieldInput("");
  };

  const removeField = (idx: number) => {
    if (!config) return;
    setConfig({ ...config, required_fields: config.required_fields.filter((_, i) => i !== idx) });
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const pending = records.filter((r) => r.status === "pending");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><FileCheck className="h-6 w-6 text-teal-600" /> {t("userAttestation.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Periodic user data attestation with reminders and configurable required fields.</p>
      </div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-teal-600" /></div>
      : (
        <>
          {/* Config */}
          {config && (
            <div className={cardCls}>
              <div className="mb-4 flex items-center justify-between">
                <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300">Configuration</h3>
                <button onClick={() => setConfig({ ...config, enabled: !config.enabled })} className={`flex items-center gap-2 rounded-lg px-3 py-1.5 text-sm font-medium ${config.enabled ? "bg-green-100 text-green-700 dark:bg-green-900/30" : "bg-gray-100 text-gray-500 dark:bg-gray-700"}`}>{config.enabled ? "Enabled" : "Disabled"}</button>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Frequency (days)</label><input type="number" value={config.frequency_days} onChange={(e) => setConfig({ ...config, frequency_days: parseInt(e.target.value) || 90 })} min={7} max={365} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
                <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Reminder (days before expiry)</label><input type="number" value={config.reminder_days_before} onChange={(e) => setConfig({ ...config, reminder_days_before: parseInt(e.target.value) || 7 })} min={1} max={30} className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" /></div>
              </div>
              <div className="mt-4">
                <label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Required Fields</label>
                <div className="flex gap-2">
                  <input value={fieldInput} onChange={(e) => setFieldInput(e.target.value)} onKeyDown={(e) => e.key === "Enter" && addField()} placeholder="e.g. phone, department" className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" />
                  <button onClick={addField} className="rounded-lg bg-gray-100 px-3 text-sm text-gray-600 dark:bg-gray-700 dark:text-gray-300"><Plus className="h-4 w-4" /></button>
                </div>
                {config.required_fields.length > 0 && (
                  <div className="mt-2 flex flex-wrap gap-2">
                    {config.required_fields.map((f, i) => (
                      <span key={i} className="flex items-center gap-1 rounded-full bg-teal-100 px-2 py-1 text-xs text-teal-700 dark:bg-teal-900/30 dark:text-teal-400">{f}<button onClick={() => removeField(i)}><X className="h-3 w-3" /></button></span>
                    ))}
                  </div>
                )}
              </div>
              <button onClick={handleSave} disabled={saving} className="mt-4 flex items-center gap-2 rounded-lg bg-teal-600 px-4 py-2 text-sm font-medium text-white hover:bg-teal-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}Save Config</button>
            </div>
          )}

          {/* Stats */}
          <div className="grid grid-cols-3 gap-4">
            <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">Pending</div><p className="mt-2 text-2xl font-bold text-yellow-600">{pending.length}</p></div>
            <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">Completed</div><p className="mt-2 text-2xl font-bold text-green-600">{records.filter((r) => r.status === "completed").length}</p></div>
            <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">Expired</div><p className="mt-2 text-2xl font-bold text-red-600">{records.filter((r) => r.status === "expired").length}</p></div>
          </div>

          {/* Pending table */}
          <div>
            <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">Pending Attestations</h2>
            {records.length === 0 ? (
              <div className={cardCls}><div className="py-12 text-center"><FileCheck className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No attestation records.</p></div></div>
            ) : (
              <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 dark:bg-gray-800"><tr>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">User</th>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Pending Fields</th>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Sent</th>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Expires</th>
                    <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Status</th>
                    <th className="px-4 py-3 text-right font-semibold text-gray-600 dark:text-gray-300">Actions</th>
                  </tr></thead>
                  <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                    {records.map((r) => (
                      <tr key={r.id} className="bg-white dark:bg-gray-900">
                        <td className="px-4 py-3"><div className="font-medium text-gray-900 dark:text-white">{r.username}</div><div className="text-xs text-gray-400 font-mono">{r.user_id.slice(0, 16)}</div></td>
                        <td className="px-4 py-3"><div className="flex flex-wrap gap-1">{r.pending_fields.map((f) => <span key={f} className="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600 dark:bg-gray-700 dark:text-gray-300">{f}</span>)}</div></td>
                        <td className="px-4 py-3 text-gray-400">{new Date(r.sent_at).toLocaleDateString()}</td>
                        <td className="px-4 py-3 text-gray-400">{r.expires_at ? new Date(r.expires_at).toLocaleDateString() : "—"}</td>
                        <td className="px-4 py-3"><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColors[r.status] || ""}`}>{r.status}</span></td>
                        <td className="px-4 py-3 text-right">{r.status === "pending" && <button onClick={() => handleReminder(r.id)} disabled={reminding === r.id} className="text-xs text-indigo-600 hover:underline">{reminding === r.id ? <Loader2 className="inline h-3 w-3 animate-spin" /> : "Remind"}</button>}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
