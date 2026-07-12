"use client";
import { useEffect, useState } from "react";
import { useConnectionPoolTuning, ConnectionPoolTuning, PoolConfig } from "@ggid/sdk-react";

export default function ConnectionPoolTuningPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useConnectionPoolTuning();
  const [form, setForm] = useState<ConnectionPoolTuning | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Connection Pool Tuning</h1>
      <p className="text-gray-600">Monitor and tune connection pools across services.</p>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Pool Configurations</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Service</th><th>Type</th><th>Max</th><th>Min</th><th>Idle</th><th>Utilization</th><th>Leaks</th></tr></thead><tbody>{form.pool_configs.map((p: PoolConfig, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{p.service}</td><td><span className="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs">{p.type}</span></td><td>{p.max}</td><td>{p.min}</td><td>{p.idle}</td><td><div className="flex items-center gap-2"><div className="w-20 bg-gray-200 rounded-full h-2"><div className="bg-blue-600 h-2 rounded-full" style={{ width: `${Math.min(p.current_utilization, 100)}%` }} /></div><span className="text-xs">{p.current_utilization}%</span></div></td><td>{p.leak_count > 0 ? <span className="text-red-600 font-medium">{p.leak_count}</span> : "0"}</td></tr>))}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Settings</h2><div><label className="block text-sm font-medium mb-1">Sizing Recommendation</label><p className="text-sm text-gray-600">{form.sizing_recommendation}</p></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.leak_detection} onChange={(e) => setForm({ ...form, leak_detection: e.target.checked })} className="w-4 h-4" /><label>Leak Detection</label></div></div>
      {form.benchmark_results && (<div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Benchmark Results</h2><div className="grid grid-cols-2 gap-4"><div className="text-center"><div className="text-2xl font-bold">{form.benchmark_results.throughput_rps}</div><div className="text-xs text-gray-500">Throughput (rps)</div></div><div className="text-center"><div className="text-2xl font-bold">{form.benchmark_results.avg_latency_ms}ms</div><div className="text-xs text-gray-500">Avg Latency</div></div></div></div>)}
      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
