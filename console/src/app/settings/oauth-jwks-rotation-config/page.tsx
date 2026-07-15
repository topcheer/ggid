"use client";
import { useEffect, useState } from "react";
import { useOAuthJwksRotationConfig, OAuthJwksRotationConfig } from "@ggid/sdk-react";

interface LocalRotationHistoryEntry {
  kid: string;
  rotated_at: string;
  algorithm: string;
}

interface LocalOAuthJwksRotationConfig extends OAuthJwksRotationConfig {
  rotation_history: LocalRotationHistoryEntry[];
}

export default function OAuthJwksRotationConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig, rotateNow } = useOAuthJwksRotationConfig();
  const [form, setForm] = useState<LocalOAuthJwksRotationConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [rotating, setRotating] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config as unknown as LocalOAuthJwksRotationConfig); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form as unknown as Parameters<typeof updateConfig>[0]); setSaving(false); };
  const handleRotate = async () => { if (!confirm("Rotate JWKS keys now?")) return; setRotating(true); await rotateNow(); setRotating(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">OAuth JWKS Rotation Configuration</h1>
      <p className="text-gray-600">Configure automatic JWKS key rotation strategy.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Rotation Settings</h2>
        <div className="grid grid-cols-2 gap-4">
          <div><label className="block text-sm font-medium mb-1">Auto Rotation Interval (days)</label><input type="number" value={form.auto_rotation_interval_days} onChange={(e) => setForm({ ...form, auto_rotation_interval_days: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Key Overlap Period (days)</label><input type="number" value={form.key_overlap_period_days} onChange={(e) => setForm({ ...form, key_overlap_period_days: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
        </div>
        <div className="grid grid-cols-3 gap-4">
          <div><label className="block text-sm font-medium mb-1">Signing Algorithm</label><input type="text" value={form.signing_alg} onChange={(e) => setForm({ ...form, signing_alg: e.target.value })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">KID Strategy</label><input type="text" value={form.kid_strategy} onChange={(e) => setForm({ ...form, kid_strategy: e.target.value })} className="border rounded px-3 py-2 w-full" /></div>
          <div><label className="block text-sm font-medium mb-1">Max Active Keys</label><input type="number" value={form.max_active_keys} onChange={(e) => setForm({ ...form, max_active_keys: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-full" /></div>
        </div>
      </div>

      <div className="flex gap-4">
        <button onClick={handleSave} disabled={saving} aria-label="Save JWKS rotation config" className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
        <button onClick={handleRotate} disabled={rotating} aria-label="Rotate JWKS keys now" className="px-6 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 disabled:opacity-50">{rotating ? "Rotating..." : "Rotate Now"}</button>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Rotation History</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">KID</th><th>Rotated At</th><th>Algorithm</th></tr></thead><tbody>
          {form.rotation_history.map((h, i) => (
            <tr key={i} className="border-b"><td className="py-2 font-mono text-xs">{h.kid}</td><td className="text-xs text-gray-500">{h.rotated_at}</td><td>{h.algorithm}</td></tr>
          ))}
        </tbody></table>
      </div>
    </div>
  );
}
