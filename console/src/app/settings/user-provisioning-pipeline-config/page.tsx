"use client";
import { useEffect, useState } from "react";
import { useUserProvisioningPipelineConfig, UserProvisioningPipelineConfig, PipelineStage } from "@ggid/sdk-react";

export default function UserProvisioningPipelineConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useUserProvisioningPipelineConfig();
  const [form, setForm] = useState<UserProvisioningPipelineConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  const configEntries = Object.entries(form.error_retry_policy);

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">User Provisioning Pipeline Configuration</h1>
      <p className="text-gray-600">Configure user provisioning pipeline stages, webhooks, and retry.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Pipeline Stages</h2>
        <div className="space-y-2">
          {form.pipeline_stages.map((s: PipelineStage, i: number) => (
            <div key={i} className="flex items-center gap-3 border-b py-2">
              <span className="w-8 h-8 rounded-full bg-blue-100 text-blue-700 flex items-center justify-center text-sm font-bold">{i + 1}</span>
              <div className="flex-1">
                <div className="font-medium">{s.stage}</div>
                <div className="text-sm text-gray-500">{s.description}</div>
              </div>
              <span className={`px-2 py-1 rounded text-xs ${s.enabled ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-500"}`}>{s.enabled ? "Enabled" : "Disabled"}</span>
            </div>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-3">
        <h2 className="text-lg font-semibold">Webhook Notifications</h2>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.webhook_notifications.enabled} onChange={(e) => setForm({ ...form, webhook_notifications: { ...form.webhook_notifications, enabled: e.target.checked } })} className="w-4 h-4" /><label>Enabled</label></div>
        <div><label className="block text-sm font-medium mb-1">Webhook URL</label><input type="text" value={form.webhook_notifications.url} onChange={(e) => setForm({ ...form, webhook_notifications: { ...form.webhook_notifications, url: e.target.value } })} className="border rounded px-3 py-2 w-full" /></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Error Retry Policy</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="block text-sm font-medium mb-1">Max Attempts</label><input type="number" value={form.error_retry_policy.max_attempts} onChange={(e) => setForm({ ...form, error_retry_policy: { ...form.error_retry_policy, max_attempts: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Backoff (s)</label><input type="number" value={form.error_retry_policy.backoff_seconds} onChange={(e) => setForm({ ...form, error_retry_policy: { ...form.error_retry_policy, backoff_seconds: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-full" /></div>
        </div>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.error_retry_policy.dead_letter_queue} onChange={(e) => setForm({ ...form, error_retry_policy: { ...form.error_retry_policy, dead_letter_queue: e.target.checked } })} className="w-4 h-4" /><label>Dead Letter Queue</label></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <label className="block text-sm font-medium mb-1">Throughput Target (users/min)</label>
        <input type="number" value={form.throughput_target} onChange={(e) => setForm({ ...form, throughput_target: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" />
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
