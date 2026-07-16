"use client";
import { useEffect, useState } from "react";
import { useRefreshTokenRotationConfig, RefreshTokenRotationConfig, RefreshTokenClientOverride } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function RefreshTokenRotationConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useRefreshTokenRotationConfig();
  const [form, setForm] = useState<RefreshTokenRotationConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Refresh Token Rotation Configuration</h1>
      <p className="text-gray-600">Configure refresh token rotation, reuse detection, and family revocation.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Rotation Policy</h2>
        <div><label className="block text-sm font-medium mb-1">Rotation Mode</label>
          <select aria-label="form" value={form.rotation_mode} onChange={(e) => setForm({ ...form, rotation_mode: e.target.value as RefreshTokenRotationConfig["rotation_mode"] })} className="border rounded px-3 py-2">
            <option value="rotate">Rotate</option><option value="reuse">Reuse</option>
          </select>
        </div>
        <div className="flex items-center gap-3">
          <input aria-label="Form" type="checkbox" checked={form.reuse_detection} onChange={(e) => setForm({ ...form, reuse_detection: e.target.checked })} className="w-4 h-4" />
          <label>Reuse Detection</label>
        </div>
        <div className="flex items-center gap-3">
          <input aria-label="Form" type="checkbox" checked={form.family_revocation_on_reuse} onChange={(e) => setForm({ ...form, family_revocation_on_reuse: e.target.checked })} className="w-4 h-4" />
          <label>Family Revocation on Reuse</label>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Timing</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="block text-sm font-medium mb-1">Grace Period (s)</label><input aria-label="form" type="number" value={form.grace_period_seconds} onChange={(e) => setForm({ ...form, grace_period_seconds: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Backward Compat Duration (s)</label><input aria-label="form" type="number" value={form.backward_compat_duration} onChange={(e) => setForm({ ...form, backward_compat_duration: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Per-Client Overrides</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Client</th><th scope="col">Rotation Mode</th><th>Grace Period (s)</th></tr></thead><tbody>
          {form.per_client_override.map((c: RefreshTokenClientOverride, i: number) => (
            <tr key={i} className="border-b"><td className="py-2"><span className="font-medium">{c.client_name}</span><div className="text-xs text-gray-400">{c.client_id}</div></td><td>{c.rotation_mode}</td><td>{c.grace_period_seconds}</td></tr>
          ))}
        </tbody></table>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
