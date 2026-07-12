"use client";
import { useEffect, useState } from "react";
import { useDistributedTracingConfig, DistributedTracingConfig, PerServiceSpan } from "@ggid/sdk-react";

export default function DistributedTracingConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useDistributedTracingConfig();
  const [form, setForm] = useState<DistributedTracingConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Distributed Tracing Configuration</h1>
      <p className="text-gray-600">Configure OpenTelemetry tracing, sampling, and audit correlation.</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Core Settings</h2><div><label className="block text-sm font-medium mb-1">OTel Endpoint</label><input type="text" value={form.otel_endpoint} onChange={(e) => setForm({ ...form, otel_endpoint: e.target.value })} className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">Sampling Rate ({form.sampling_rate}%)</label><input type="range" min="0" max="100" value={form.sampling_rate} onChange={(e) => setForm({ ...form, sampling_rate: parseInt(e.target.value) })} className="w-full" /></div><div className="grid grid-cols-2 gap-4"><div><label className="block text-sm font-medium mb-1">Propagation Format</label><select value={form.propagation_format} onChange={(e) => setForm({ ...form, propagation_format: e.target.value as DistributedTracingConfig["propagation_format"] })} className="border rounded px-3 py-2"><option value="W3C">W3C</option><option value="Jaeger">Jaeger</option></select></div><div><label className="block text-sm font-medium mb-1">Backend</label><select value={form.backend} onChange={(e) => setForm({ ...form, backend: e.target.value as DistributedTracingConfig["backend"] })} className="border rounded px-3 py-2"><option value="Jaeger">Jaeger</option><option value="Tempo">Tempo</option></select></div></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.trace_correlation_with_audit} onChange={(e) => setForm({ ...form, trace_correlation_with_audit: e.target.checked })} className="w-4 h-4" /><label>Correlate traces with audit events</label></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Per-Service Span Configuration</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Service</th><th>Sample Rate</th><th>Max Spans</th></tr></thead><tbody>{form.per_service_span_config.map((s: PerServiceSpan, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{s.service}</td><td>{s.sample_rate}%</td><td>{s.max_spans}</td></tr>))}</tbody></table></div>
      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
