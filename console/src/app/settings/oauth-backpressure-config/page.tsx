"use client";
import { useEffect, useState } from "react";
import { useOAuthBackpressureConfig, OAuthBackpressureConfig, DegradationRule } from "@ggid/sdk-react";

export default function OAuthBackpressureConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useOAuthBackpressureConfig();
  const [form, setForm] = useState<OAuthBackpressureConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">OAuth Backpressure Configuration</h1>
      <p className="text-gray-600">Configure OAuth request queueing, circuit breaking, and rate limiting.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Queue & Concurrency</h2>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.per_client_fair_queueing} onChange={(e) => setForm({ ...form, per_client_fair_queueing: e.target.checked })} className="w-4 h-4" /><label>Per-Client Fair Queueing</label></div>
        <div className="grid grid-cols-2 gap-4"><div><label className="block text-sm font-medium mb-1">Max Concurrent Token Requests</label><input type="number" value={form.max_concurrent_token_requests} onChange={(e) => setForm({ ...form, max_concurrent_token_requests: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">Circuit Breaker Threshold</label><input type="number" value={form.circuit_breaker_threshold} onChange={(e) => setForm({ ...form, circuit_breaker_threshold: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div></div>
        <div><label className="block text-sm font-medium mb-1">Queue Overflow Action</label><select value={form.queue_overflow_action} onChange={(e) => setForm({ ...form, queue_overflow_action: e.target.value as OAuthBackpressureConfig["queue_overflow_action"] })} className="border rounded px-3 py-2"><option value="reject">Reject</option><option value="defer">Defer</option></select></div>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.rate_limit_headers} onChange={(e) => setForm({ ...form, rate_limit_headers: e.target.checked })} className="w-4 h-4" /><label>Rate Limit Headers</label></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Graceful Degradation Rules</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Metric</th><th>Threshold</th><th>Action</th></tr></thead><tbody>{form.graceful_degradation_rules.map((r: DegradationRule, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{r.metric}</td><td>{r.threshold}</td><td className="text-xs">{r.action}</td></tr>))}</tbody></table></div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
