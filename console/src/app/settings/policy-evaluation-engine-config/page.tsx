"use client";
import { useEffect, useState } from "react";
import { usePolicyEvaluationEngineConfig, PolicyEvaluationEngineConfig } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function PolicyEvaluationEngineConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = usePolicyEvaluationEngineConfig();
  const [form, setForm] = useState<PolicyEvaluationEngineConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Policy Evaluation Engine Configuration</h1>
      <p className="text-gray-600">Configure policy evaluation performance, caching, and optimization.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Evaluation Settings</h2>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.rbac_fast_path} onChange={(e) => setForm({ ...form, rbac_fast_path: e.target.checked })} className="w-4 h-4" /><label>RBAC Fast Path</label></div>
        <div><label className="block text-sm font-medium mb-1">ABAC CEL Timeout (ms)</label><input aria-label="form" type="number" value={form.abac_cel_timeout_ms} onChange={(e) => setForm({ ...form, abac_cel_timeout_ms: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.decision_tree_optimization} onChange={(e) => setForm({ ...form, decision_tree_optimization: e.target.checked })} className="w-4 h-4" /><label>Decision Tree Optimization</label></div>
        <div><label className="block text-sm font-medium mb-1">Hot Path Threshold</label><input aria-label="form" type="number" value={form.hot_path_threshold} onChange={(e) => setForm({ ...form, hot_path_threshold: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Cache Settings</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="block text-sm font-medium mb-1">Cache TTL (s)</label><input aria-label="form" type="number" value={form.cache_ttl_seconds} onChange={(e) => setForm({ ...form, cache_ttl_seconds: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Max Cache Entries</label><input aria-label="form" type="number" value={form.max_cache_entries} onChange={(e) => setForm({ ...form, max_cache_entries: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Benchmark Results</h2>
        <div className="grid grid-cols-4 gap-4">
          <div className="text-center"><div className="text-2xl font-bold">{form.benchmark_results.total_evaluations}</div><div className="text-xs text-gray-500">Total</div></div>
          <div className="text-center"><div className="text-2xl font-bold text-green-600">{form.benchmark_results.avg_latency_ms}ms</div><div className="text-xs text-gray-500">Avg Latency</div></div>
          <div className="text-center"><div className="text-2xl font-bold text-yellow-600">{form.benchmark_results.p99_latency_ms}ms</div><div className="text-xs text-gray-500">P99 Latency</div></div>
          <div className="text-center"><div className="text-2xl font-bold text-blue-600">{form.benchmark_results.cache_hit_rate_pct}%</div><div className="text-xs text-gray-500">Cache Hit Rate</div></div>
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
