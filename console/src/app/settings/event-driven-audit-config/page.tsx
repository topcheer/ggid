"use client";
import { useEffect, useState } from "react";
import { useEventDrivenAuditConfig, EventDrivenAuditConfig } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function EventDrivenAuditConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useEventDrivenAuditConfig();
  const [form, setForm] = useState<EventDrivenAuditConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Event-Driven Audit Configuration</h1>
      <p className="text-gray-600">Configure NATS JetStream audit event streaming.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-3">
        <h2 className="text-lg font-semibold">Stream Configuration</h2>
        <div><span className="text-gray-500 text-sm">Name:</span> <span className="font-medium">{form.stream_config.name}</span></div>
        <div><span className="text-gray-500 text-sm">Subjects:</span> <span className="font-mono text-sm">{form.stream_config.subjects.join(", ")}</span></div>
        <div><span className="text-gray-500 text-sm">Retention:</span> <span className="font-medium">{form.stream_config.retention}</span></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Consumer & Ordering</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="block text-sm font-medium mb-1">Consumer Pattern</label><select aria-label="Select option" value={form.consumer_pattern} onChange={(e) => setForm({ ...form, consumer_pattern: e.target.value as EventDrivenAuditConfig["consumer_pattern"] })} className="border rounded px-3 py-2"><option value="competing">Competing</option><option value="shared">Shared</option><option value="fanout">Fanout</option></select></div>
          <div><label className="block text-sm font-medium mb-1">Ordering</label><select aria-label="form" value={form.ordering} onChange={(e) => setForm({ ...form, ordering: e.target.value as EventDrivenAuditConfig["ordering"] })} className="border rounded px-3 py-2"><option value="per_tenant">Per Tenant</option><option value="global">Global</option></select></div>
        </div>
        <div><label className="block text-sm font-medium mb-1">Deduplication Window (ms)</label><input aria-label="form" type="number" value={form.deduplication_window_ms} onChange={(e) => setForm({ ...form, deduplication_window_ms: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
        <div><label className="block text-sm font-medium mb-1">Backpressure Strategy</label><select aria-label="form" value={form.backpressure_strategy} onChange={(e) => setForm({ ...form, backpressure_strategy: e.target.value as EventDrivenAuditConfig["backpressure_strategy"] })} className="border rounded px-3 py-2"><option value="block">Block</option><option value="drop">Drop</option><option value="buffer">Buffer</option></select></div>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.replay_enabled} onChange={(e) => setForm({ ...form, replay_enabled: e.target.checked })} className="w-4 h-4" /><label>Replay Enabled</label></div>
      </div>

      <button aria-label="action" onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
