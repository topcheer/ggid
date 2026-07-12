"use client";
import { useEffect, useState } from "react";
import { useIdentityProofingConfig, IdentityProofingConfig, VerificationMethod, RiskLevelConfig } from "@ggid/sdk-react";

export default function IdentityProofingConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useIdentityProofingConfig();
  const [form, setForm] = useState<IdentityProofingConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  const completionPct = form.completion_rate.total > 0 ? Math.round((form.completion_rate.completed / form.completion_rate.total) * 100) : 0;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Identity Proofing Configuration</h1>
      <p className="text-gray-600">Configure identity verification methods and confidence thresholds.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Verification Methods</h2>
        <div className="space-y-2">
          {form.verification_methods.map((m: VerificationMethod, i: number) => (
            <div key={i} className="flex items-center gap-3 border-b py-2">
              <input type="checkbox" checked={m.enabled} readOnly className="w-4 h-4" />
              <span className="font-medium">{m.method}</span>
            </div>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Requirements</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Required Factors: {form.required_factors}</label>
          <input type="range" min={1} max={5} value={form.required_factors} onChange={(e) => setForm({ ...form, required_factors: parseInt(e.target.value) })} className="w-full" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Confidence Threshold: {form.confidence_threshold}%</label>
          <input type="range" min={50} max={100} value={form.confidence_threshold} onChange={(e) => setForm({ ...form, confidence_threshold: parseInt(e.target.value) })} className="w-full" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Verification Provider</label>
          <select value={form.verification_provider} onChange={(e) => setForm({ ...form, verification_provider: e.target.value })} className="border rounded px-3 py-2">
            <option value="onfido">Onfido</option><option value="jumio">Jumio</option><option value="idology">IDology</option><option value="internal">Internal</option>
          </select>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Per Risk Level Matrix</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Risk Level</th><th>Required Factors</th><th>Methods</th></tr></thead><tbody>
          {form.per_risk_level.map((r: RiskLevelConfig, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-medium">{r.level}</td><td>{r.required_factors}</td><td>{r.methods.join(", ")}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Completion Rate</h2>
        <div className="grid grid-cols-4 gap-4">
          <div className="text-center"><div className="text-2xl font-bold">{form.completion_rate.total}</div><div className="text-xs text-gray-500">Total</div></div>
          <div className="text-center"><div className="text-2xl font-bold text-green-600">{form.completion_rate.completed}</div><div className="text-xs text-gray-500">Completed</div></div>
          <div className="text-center"><div className="text-2xl font-bold text-red-600">{form.completion_rate.failed}</div><div className="text-xs text-gray-500">Failed</div></div>
          <div className="text-center"><div className="text-2xl font-bold text-blue-600">{completionPct}%</div><div className="text-xs text-gray-500">Rate</div></div>
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
