"use client";
import { useEffect, useState } from "react";
import { useLogAggregationConfig, LogAggregationConfig, LogLevel, RedactionRule } from "@ggid/sdk-react";

export default function LogAggregationConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useLogAggregationConfig();
  const [form, setForm] = useState<LogAggregationConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;
  const levelColors: Record<string, string> = { debug: "bg-gray-100 text-gray-600", info: "bg-blue-100 text-blue-700", warn: "bg-yellow-100 text-yellow-700", error: "bg-red-100 text-red-700" };

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Log Aggregation Configuration</h1>
      <p className="text-gray-600">Configure structured logging, correlation, redaction, and routing.</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Core</h2><div className="grid grid-cols-2 gap-4"><div><label className="block text-sm font-medium mb-1">Log Format</label><select value={form.log_format} onChange={(e) => setForm({ ...form, log_format: e.target.value as LogAggregationConfig["log_format"] })} className="border rounded px-3 py-2"><option value="JSON">JSON</option><option value="structured">Structured</option></select></div><div><label className="block text-sm font-medium mb-1">Routing</label><select value={form.log_routing} onChange={(e) => setForm({ ...form, log_routing: e.target.value as LogAggregationConfig["log_routing"] })} className="border rounded px-3 py-2"><option value="Loki">Loki</option><option value="ELK">ELK</option></select></div></div><div><label className="block text-sm font-medium mb-1">Retention (days)</label><input type="number" value={form.retention_days} onChange={(e) => setForm({ ...form, retention_days: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.correlation_id_enabled} onChange={(e) => setForm({ ...form, correlation_id_enabled: e.target.checked })} className="w-4 h-4" /><label>Correlation ID</label></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.cost_optimization} onChange={(e) => setForm({ ...form, cost_optimization: e.target.checked })} className="w-4 h-4" /><label>Cost Optimization (drop debug logs in prod)</label></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Per-Service Log Levels</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Service</th><th>Level</th></tr></thead><tbody>{form.level_per_service.map((l: LogLevel, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{l.service}</td><td><span className={`px-2 py-1 rounded text-xs ${levelColors[l.level] || ""}`}>{l.level}</span></td></tr>))}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Sensitive Data Redaction Rules</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Pattern</th><th>Replacement</th></tr></thead><tbody>{form.sensitive_data_redaction_rules.map((r: RedactionRule, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-mono text-xs">{r.pattern}</td><td className="font-mono text-xs">{r.replacement}</td></tr>))}</tbody></table></div>
      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
