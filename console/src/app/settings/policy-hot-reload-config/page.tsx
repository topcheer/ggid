"use client";
import { useEffect, useState } from "react";
import { usePolicyHotReloadConfig, PolicyHotReloadConfig } from "@ggid/sdk-react";

export default function PolicyHotReloadConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = usePolicyHotReloadConfig();
  const [form, setForm] = useState<PolicyHotReloadConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Policy Hot-Reload Configuration</h1>
      <p className="text-gray-600">Configure zero-downtime policy reload with atomic swap and rollback.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Reload Settings</h2>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.watch_enabled} onChange={(e) => setForm({ ...form, watch_enabled: e.target.checked })} className="w-4 h-4" /><label>Watch Enabled</label></div>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.atomic_swap} onChange={(e) => setForm({ ...form, atomic_swap: e.target.checked })} className="w-4 h-4" /><label>Atomic Swap</label></div>
        <div><label className="block text-sm font-medium mb-1">Cache Invalidation Strategy</label><select value={form.cache_invalidation_strategy} onChange={(e) => setForm({ ...form, cache_invalidation_strategy: e.target.value as PolicyHotReloadConfig["cache_invalidation_strategy"] })} className="border rounded px-3 py-2"><option value="all">All</option><option value="lazy">Lazy</option><option value="versioned">Versioned</option></select></div>
        <div className="grid grid-cols-2 gap-4"><div><label className="block text-sm font-medium mb-1">Version Check Interval (ms)</label><input type="number" value={form.version_check_interval_ms} onChange={(e) => setForm({ ...form, version_check_interval_ms: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">Max Reload Concurrency</label><input type="number" value={form.max_reload_concurrency} onChange={(e) => setForm({ ...form, max_reload_concurrency: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div></div>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.rollback_on_error} onChange={(e) => setForm({ ...form, rollback_on_error: e.target.checked })} className="w-4 h-4" /><label>Rollback on Error</label></div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
