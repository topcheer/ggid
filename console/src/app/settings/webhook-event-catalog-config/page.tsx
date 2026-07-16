"use client";
import { useEffect, useState } from "react";
import { useWebhookEventCatalogConfig, WebhookEventCatalogConfig, EventTypeEntry } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function WebhookEventCatalogConfigPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig, testEvent } = useWebhookEventCatalogConfig();
  const [form, setForm] = useState<WebhookEventCatalogConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  const handleTest = async (ev: string) => { setTesting(true); await testEvent(ev); setTesting(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Webhook Event Catalog Configuration</h1>
      <p className="text-gray-600">Configure webhook event types, retry policies, and delivery.</p>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Event Types</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Name</th><th scope="col">Subscribers</th><th>Guarantee</th><th>Test</th></tr></thead><tbody>{form.event_types.map((e: EventTypeEntry, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{e.name}</td><td>{e.subscribers_count}</td><td><span className="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">{e.delivery_guarantee}</span></td><td><button onClick={() => handleTest(e.name)} disabled={testing} className="text-blue-600 hover:text-blue-800 text-xs">Test</button></td></tr>))}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Per-Event Retry Policy</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Event</th><th scope="col">Max Attempts</th><th>Backoff (s)</th></tr></thead><tbody>{form.per_event_retry_policy.map((p: { event: string; max_attempts: number; backoff_seconds: number }, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{p.event}</td><td>{p.max_attempts}</td><td>{p.backoff_seconds}</td></tr>))}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Sample Handlers</h2><div className="space-y-1">{form.sample_handlers.map((h: string, i: number) => (<div key={i} className="border-b py-1 font-mono text-xs">{h}</div>))}</div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Delivery Stats</h2><div className="grid grid-cols-4 gap-4"><div className="text-center"><div className="text-2xl font-bold">{form.delivery_stats.total_sent}</div><div className="text-xs text-gray-500">Total</div></div><div className="text-center"><div className="text-2xl font-bold text-green-600">{form.delivery_stats.delivered}</div><div className="text-xs text-gray-500">Delivered</div></div><div className="text-center"><div className="text-2xl font-bold text-red-600">{form.delivery_stats.failed}</div><div className="text-xs text-gray-500">Failed</div></div><div className="text-center"><div className="text-2xl font-bold text-yellow-600">{form.delivery_stats.retrying}</div><div className="text-xs text-gray-500">Retrying</div></div></div></div>
      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
