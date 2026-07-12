"use client";
import { useEffect, useState } from "react";
import { useAuditExportScheduleConfig, AuditExportScheduleConfig, ScheduledExportJob } from "@ggid/sdk-react";

export default function AuditExportScheduleConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useAuditExportScheduleConfig();
  const [form, setForm] = useState<AuditExportScheduleConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Audit Export Schedule Configuration</h1>
      <p className="text-gray-600">Configure scheduled audit log export jobs.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-lg font-semibold">Scheduled Jobs</h2>
          <button className="px-4 py-1 bg-green-600 text-white rounded text-sm hover:bg-green-700">+ Add Job</button>
        </div>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">Name</th><th>Cron</th><th>Format</th><th>Retention</th><th>Destination</th><th>Last Run</th></tr></thead>
          <tbody>
            {form.scheduled_jobs.map((j: ScheduledExportJob, i: number) => (
              <tr key={i} className="border-b">
                <td className="py-2">{j.name}</td><td className="font-mono text-xs">{j.cron}</td><td>{j.format}</td>
                <td>{j.retention_days}d</td><td className="break-all max-w-[150px] truncate">{j.destination}</td>
                <td className="text-xs text-gray-500">{j.last_run}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Global Settings</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Max Concurrent Jobs</label>
          <input type="number" value={form.max_concurrent}
            onChange={(e) => setForm({ ...form, max_concurrent: parseInt(e.target.value) || 0 })}
            className="border rounded px-3 py-2 w-32" />
        </div>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">Max Retry Attempts</label>
            <input type="number" value={form.retry_policy.max_attempts}
              onChange={(e) => setForm({ ...form, retry_policy: { ...form.retry_policy, max_attempts: parseInt(e.target.value) || 0 } })}
              className="border rounded px-3 py-2 w-32" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Backoff (seconds)</label>
            <input type="number" value={form.retry_policy.backoff_seconds}
              onChange={(e) => setForm({ ...form, retry_policy: { ...form.retry_policy, backoff_seconds: parseInt(e.target.value) || 0 } })}
              className="border rounded px-3 py-2 w-32" />
          </div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-3">
        <h2 className="text-lg font-semibold">Notification on Complete</h2>
        <div className="flex items-center gap-3">
          <input type="checkbox" checked={form.notification_on_complete.enabled}
            onChange={(e) => setForm({ ...form, notification_on_complete: { ...form.notification_on_complete, enabled: e.target.checked } })}
            className="w-4 h-4" />
          <label>Enabled</label>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Webhook URL</label>
          <input type="text" value={form.notification_on_complete.webhook_url}
            onChange={(e) => setForm({ ...form, notification_on_complete: { ...form.notification_on_complete, webhook_url: e.target.value } })}
            className="border rounded px-3 py-2 w-full" />
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
