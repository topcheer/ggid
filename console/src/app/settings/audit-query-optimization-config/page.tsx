"use client";
import { useEffect, useState } from "react";
import { useAuditQueryOptimizationConfig, AuditQueryOptimizationConfig, IndexConfig } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function AuditQueryOptimizationConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useAuditQueryOptimizationConfig();
  const [form, setForm] = useState<AuditQueryOptimizationConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Audit Query Optimization Configuration</h1>
      <p className="text-gray-600">Configure audit log partitioning, indexing, and pagination.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Partitioning & Pagination</h2>
        <div className="grid grid-cols-3 gap-4"><div><label className="block text-sm font-medium mb-1">Partition Strategy</label><select aria-label="Select option" value={form.partition_strategy} onChange={(e) => setForm({ ...form, partition_strategy: e.target.value as AuditQueryOptimizationConfig["partition_strategy"] })} className="border rounded px-3 py-2"><option value="daily">Daily</option><option value="monthly">Monthly</option></select></div><div><label className="block text-sm font-medium mb-1">Cursor Page Size</label><input type="number" value={form.cursor_pagination_size} onChange={(e) => setForm({ ...form, cursor_pagination_size: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">Slow Query Threshold (ms)</label><input type="number" value={form.slow_query_threshold_ms} onChange={(e) => setForm({ ...form, slow_query_threshold_ms: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div></div>
        <div><label className="block text-sm font-medium mb-1">Materialized View Refresh (s)</label><input type="number" value={form.materialized_view_refresh_interval} onChange={(e) => setForm({ ...form, materialized_view_refresh_interval: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Index Configuration</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Table</th><th scope="col">Columns</th><th>Type</th></tr></thead><tbody>{form.index_config.map((idx: IndexConfig, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-mono">{idx.table}</td><td className="text-xs">{idx.columns.join(", ")}</td><td><span className="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">{idx.type}</span></td></tr>))}</tbody></table></div>

      <div className="bg-white rounded-lg p-6 shadow space-y-3"><h2 className="text-lg font-semibold">Auto Vacuum</h2><div className="flex items-center gap-3"><input type="checkbox" checked={form.auto_vacuum_config.enabled} onChange={(e) => setForm({ ...form, auto_vacuum_config: { ...form.auto_vacuum_config, enabled: e.target.checked } })} className="w-4 h-4" /><label>Enabled</label></div><div className="grid grid-cols-2 gap-4"><div><label className="block text-sm font-medium mb-1">Threshold (%)</label><input type="number" value={form.auto_vacuum_config.threshold_pct} onChange={(e) => setForm({ ...form, auto_vacuum_config: { ...form.auto_vacuum_config, threshold_pct: parseInt(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">Scale Factor</label><input type="number" step="0.1" value={form.auto_vacuum_config.scale_factor} onChange={(e) => setForm({ ...form, auto_vacuum_config: { ...form.auto_vacuum_config, scale_factor: parseFloat(e.target.value) || 0 } })} className="border rounded px-3 py-2 w-full" /></div></div></div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
