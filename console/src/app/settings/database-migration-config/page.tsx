"use client";
import { useEffect, useState } from "react";
import { useDatabaseMigrationConfig, DatabaseMigrationConfig } from "@ggid/sdk-react";

export default function DatabaseMigrationConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useDatabaseMigrationConfig();
  const [form, setForm] = useState<DatabaseMigrationConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Database Migration Configuration</h1>
      <p className="text-gray-600">Configure zero-downtime schema migration strategy and safety limits.</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Strategy</h2><div><label className="block text-sm font-medium mb-1">Migration Strategy</label><select value={form.migration_strategy} onChange={(e) => setForm({ ...form, migration_strategy: e.target.value as DatabaseMigrationConfig["migration_strategy"] })} className="border rounded px-3 py-2"><option value="expand_contract">Expand & Contract</option><option value="big_bang">Big Bang</option><option value="shadow">Shadow</option></select></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.dry_run} onChange={(e) => setForm({ ...form, dry_run: e.target.checked })} className="w-4 h-4" /><label>Dry Run Mode</label></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Safety Limits</h2><div className="grid grid-cols-2 gap-4"><div><label className="block text-sm font-medium mb-1">Max Lock Duration (ms)</label><input type="number" value={form.max_lock_duration_ms} onChange={(e) => setForm({ ...form, max_lock_duration_ms: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">Batch Size</label><input type="number" value={form.batch_size} onChange={(e) => setForm({ ...form, batch_size: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">Parallel Workers</label><input type="number" value={form.parallel_workers} onChange={(e) => setForm({ ...form, parallel_workers: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div><div><label className="block text-sm font-medium mb-1">Rollback Timeout (s)</label><input type="number" value={form.rollback_timeout_seconds} onChange={(e) => setForm({ ...form, rollback_timeout_seconds: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div></div><div><label className="block text-sm font-medium mb-1">Backward Compatibility Window (days)</label><input type="number" value={form.backward_compat_window_days} onChange={(e) => setForm({ ...form, backward_compat_window_days: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div></div>
      <button onClick={handleSave} disabled={saving} aria-label="Save database migration configuration" className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
