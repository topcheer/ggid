"use client";
import { useEffect, useState } from "react";
import { useJwtClaimValidationConfig, JwtClaimValidationConfig, CustomClaim, RequiredClaim } from "@ggid/sdk-react";

export default function JwtClaimValidationConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useJwtClaimValidationConfig();
  const [form, setForm] = useState<JwtClaimValidationConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">JWT Claim Validation Configuration</h1>
      <p className="text-gray-600">Configure JWT token claim validation rules.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">Required Claims</h2>
          <div className="flex items-center gap-3">
            <input type="checkbox" checked={form.strict_mode}
              onChange={(e) => setForm({ ...form, strict_mode: e.target.checked })}
              className="w-4 h-4" />
            <label className="text-sm font-medium">Strict Mode</label>
          </div>
        </div>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">Claim</th><th>Enabled</th></tr></thead>
          <tbody>
            {form.required_claims.map((rc: RequiredClaim, i: number) => (
              <tr key={i} className="border-b">
                <td className="py-2 font-mono">{rc.claim}</td>
                <td><input type="checkbox" checked={rc.enabled} readOnly className="w-4 h-4" /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Validation Settings</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Clock Skew (seconds)</label>
          <input type="number" value={form.clock_skew_seconds}
            onChange={(e) => setForm({ ...form, clock_skew_seconds: parseInt(e.target.value) || 0 })}
            className="border rounded px-3 py-2 w-32" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Validation Order</label>
          <div className="text-sm text-gray-600">{form.validation_order.join(" -> ")}</div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Custom Claims</h2>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">Name</th><th>Type</th><th>Required</th><th>Validator</th></tr></thead>
          <tbody>
            {form.custom_claims.map((c: CustomClaim, i: number) => (
              <tr key={i} className="border-b"><td className="py-2 font-mono">{c.name}</td><td>{c.type}</td><td>{c.required ? "Yes" : "No"}</td><td className="text-xs">{c.validator}</td></tr>
            ))}
          </tbody>
        </table>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
