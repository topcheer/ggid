"use client";
import { useEffect, useState } from "react";
import { usePolicyDryRunConfig, PolicyDryRunConfig, ContextValue } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function PolicyDryRunConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = usePolicyDryRunConfig();
  const [form, setForm] = useState<PolicyDryRunConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Policy Dry-Run Configuration</h1>
      <p className="text-gray-600">Configure policy simulation settings for testing before deployment.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Default Context Values</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Key</th><th>Value</th></tr></thead><tbody>
          {form.default_context_values.map((cv: ContextValue, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-mono">{cv.key}</td><td className="font-mono">{cv.value}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Simulation Settings</h2>
        <div><label className="block text-sm font-medium mb-1">Max Simulation Subjects</label><input type="number" value={form.max_simulation_subjects} onChange={(e) => setForm({ ...form, max_simulation_subjects: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
        <div><label className="block text-sm font-medium mb-1">Cache Results (minutes)</label><input type="number" value={form.cache_results_minutes} onChange={(e) => setForm({ ...form, cache_results_minutes: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
        <div><label className="block text-sm font-medium mb-1">Results Retention (days)</label><input type="number" value={form.results_retention_days} onChange={(e) => setForm({ ...form, results_retention_days: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-3">
        <h2 className="text-lg font-semibold">Automation</h2>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.compare_against_current} onChange={(e) => setForm({ ...form, compare_against_current: e.target.checked })} className="w-4 h-4" /><label>Compare Against Current Policy</label></div>
        <div className="flex items-center gap-3"><input type="checkbox" checked={form.auto_run_on_policy_change} onChange={(e) => setForm({ ...form, auto_run_on_policy_change: e.target.checked })} className="w-4 h-4" /><label>Auto-Run on Policy Change</label></div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
