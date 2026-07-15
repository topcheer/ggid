"use client";
import { useEffect, useState } from "react";
import { useCanaryDeploymentConfig, CanaryDeploymentConfig, PerTenantCanary, PromotionCriteria } from "@ggid/sdk-react";

export default function CanaryDeploymentConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useCanaryDeploymentConfig();
  const [form, setForm] = useState<CanaryDeploymentConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Canary Deployment Configuration</h1>
      <p className="text-gray-600">Configure progressive delivery, traffic splitting, and auto-rollback.</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Traffic Split</h2><div><label className="block text-sm font-medium mb-1">Canary Percentage ({form.canary_percentage}%)</label><input type="range" min="0" max="100" value={form.canary_percentage} onChange={(e) => setForm({ ...form, canary_percentage: parseInt(e.target.value) })} className="w-full" /></div><div><label className="block text-sm font-medium mb-1">Traffic Split Method</label><select value={form.traffic_split_method} onChange={(e) => setForm({ ...form, traffic_split_method: e.target.value as CanaryDeploymentConfig["traffic_split_method"] })} className="border rounded px-3 py-2"><option value="header">Header</option><option value="weight">Weight</option><option value="sticky">Sticky</option></select></div><div><label className="block text-sm font-medium mb-1">Auto-Rollback on Error Rate Threshold (%)</label><input type="number" value={form.auto_rollback_on_error_rate} onChange={(e) => setForm({ ...form, auto_rollback_on_error_rate: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Promotion Criteria</h2><div className="space-y-1">{form.promotion_criteria.map((c: PromotionCriteria, i: number) => (<div key={i} className="flex items-center justify-between border-b py-2"><span className="text-sm">{c.criterion} <span className="text-gray-400">({c.threshold})</span></span><span className={`px-2 py-1 rounded text-xs ${c.met ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-500"}`}>{c.met ? "Met" : "Pending"}</span></div>))}</div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Per-Tenant Canary</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Tenant</th><th>Canary Enabled</th></tr></thead><tbody>{form.per_tenant_canary.map((t: PerTenantCanary, i: number) => (<tr key={i} className="border-b"><td className="py-2">{t.tenant_name}</td><td>{t.canary_enabled ? "Yes" : "No"}</td></tr>))}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Monitoring Checkpoints</h2><div className="space-y-1">{form.monitoring_checkpoints.map((c: string, i: number) => (<div key={i} className="border-b py-1 text-sm font-mono">{c}</div>))}</div></div>
      <button onClick={handleSave} disabled={saving} aria-label="Save canary deployment configuration" className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
