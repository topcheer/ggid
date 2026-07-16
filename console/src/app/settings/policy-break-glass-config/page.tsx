"use client";
import { useEffect, useState } from "react";
import { usePolicyBreakGlassConfig, PolicyBreakGlassConfig, BreakGlassRole } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function PolicyBreakGlassConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = usePolicyBreakGlassConfig();
  const [form, setForm] = useState<PolicyBreakGlassConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Policy Break-Glass Configuration</h1>
      <p className="text-gray-600">Configure emergency break-glass access roles and safeguards.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Break-Glass Roles</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Role</th><th scope="col">Justification Required</th><th>Auto-Expire (min)</th><th>Notify on Use</th></tr></thead><tbody>
          {form.break_glass_roles.map((r: BreakGlassRole, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-medium">{r.role}</td><td>{r.justification_required ? "Yes" : "No"}</td><td>{r.auto_expire_minutes}</td><td>{r.notify_on_use ? "Yes" : "No"}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Safeguards</h2>
        <div><label className="block text-sm font-medium mb-1">Cooldown Period (minutes)</label><input aria-label="form" type="number" value={form.cooldown_period_minutes} onChange={(e) => setForm({ ...form, cooldown_period_minutes: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
        <div><label className="block text-sm font-medium mb-1">Max Concurrent Break-Glass Sessions</label><input aria-label="form" type="number" value={form.max_concurrent} onChange={(e) => setForm({ ...form, max_concurrent: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.auto_revert} onChange={(e) => setForm({ ...form, auto_revert: e.target.checked })} className="w-4 h-4" /><label>Auto-Revert After Expiry</label></div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
