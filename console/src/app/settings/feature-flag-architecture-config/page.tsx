"use client";
import { useEffect, useState } from "react";
import { useFeatureFlagArchitectureConfig, FeatureFlagArchitectureConfig, KillSwitch, PerTenantFlag } from "@ggid/sdk-react";

export default function FeatureFlagArchitectureConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useFeatureFlagArchitectureConfig();
  const [form, setForm] = useState<FeatureFlagArchitectureConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Feature Flag Architecture</h1>
      <p className="text-gray-600">Configure flag types, evaluation engine, rollout strategies, and kill switches.</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Engine</h2><div><label className="block text-sm font-medium mb-1">Evaluation Engine</label><select value={form.evaluation_engine} onChange={(e) => setForm({ ...form, evaluation_engine: e.target.value as FeatureFlagArchitectureConfig["evaluation_engine"] })} className="border rounded px-3 py-2"><option value="local">Local</option><option value="remote">Remote</option><option value="hybrid">Hybrid</option></select></div><div><label className="block text-sm font-medium mb-1">Flag Types</label><div className="text-sm text-gray-600">{form.flag_types.join(", ")}</div></div><div><label className="block text-sm font-medium mb-1">Rollout Strategies</label><div className="text-sm text-gray-600">{form.rollout_strategies.join(", ")}</div></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Kill Switches</h2><div className="space-y-2">{form.kill_switches.map((k: KillSwitch, i: number) => (<div key={i} className="flex items-center justify-between border-b py-2"><div><span className="font-medium">{k.name}</span><div className="text-xs text-gray-400">{k.description}</div></div><span className={`px-2 py-1 rounded text-xs ${k.enabled ? "bg-red-100 text-red-700" : "bg-gray-100 text-gray-500"}`}>{k.enabled ? "Active" : "Inactive"}</span></div>))}</div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Per-Tenant Flags</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Tenant</th><th>Flags</th></tr></thead><tbody>{form.per_tenant_flags.map((t: PerTenantFlag, i: number) => { const activeFlags: string[] = Object.entries(t.flags).filter(([, v]) => v === true).map(([k]) => k); return (<tr key={i} className="border-b"><td className="py-2 font-medium">{t.tenant_name}</td><td className="text-xs font-mono">{activeFlags.length > 0 ? activeFlags.join(", ") : "none active"}</td></tr>); })}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow space-y-3"><h2 className="text-lg font-semibold">A/B Testing</h2><div className="flex items-center gap-3"><input type="checkbox" checked={form.a_b_test_config.enabled} onChange={(e) => setForm({ ...form, a_b_test_config: { ...form.a_b_test_config, enabled: e.target.checked } })} className="w-4 h-4" /><label>Enabled</label></div><div className="text-sm text-gray-600">Variants: {form.a_b_test_config.variants.join(", ")}</div></div>
      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
