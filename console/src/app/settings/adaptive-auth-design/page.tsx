"use client";
import { useEffect, useState } from "react";
import { useAdaptiveAuthDesign, AdaptiveAuthDesign, SignalConfig } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function AdaptiveAuthDesignPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useAdaptiveAuthDesign();
  const [form, setForm] = useState<AdaptiveAuthDesign | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Adaptive Authentication Design</h1>
      <p className="text-gray-600">Configure risk scoring, signal collection, and threshold tuning.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Risk Scoring Model</h2>
        <div><label className="block text-sm font-medium mb-1">Model</label><input aria-label="form" type="text" value={form.risk_scoring_model} onChange={(e) => setForm({ ...form, risk_scoring_model: e.target.value })} className="border rounded px-3 py-2 w-full" /></div>
        <div>
          <label className="block text-sm font-medium mb-1">Engine Type</label>
          <select aria-label="form" value={form.ml_vs_rule_based} onChange={(e) => setForm({ ...form, ml_vs_rule_based: e.target.value as AdaptiveAuthDesign["ml_vs_rule_based"] })} className="border rounded px-3 py-2">
            <option value="rule">Rule-Based</option><option value="ml">ML-Based</option><option value="hybrid">Hybrid</option>
          </select>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Signal Collection</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Signal</th><th scope="col">Source</th><th>Latency (ms)</th><th>Weight</th></tr></thead><tbody>
          {form.signal_collection.map((s: SignalConfig, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-medium">{s.signal}</td><td>{s.source}</td><td>{s.latency_ms}</td><td>{s.weight}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Threshold Tuning</h2>
        <div><label className="block text-sm font-medium mb-1">Low: {form.threshold_tuning.low}</label><input aria-label="form" type="range" min={0} max={100} value={form.threshold_tuning.low} onChange={(e) => setForm({ ...form, threshold_tuning: { ...form.threshold_tuning, low: parseInt(e.target.value) } })} className="w-full" /></div>
        <div><label className="block text-sm font-medium mb-1">Medium: {form.threshold_tuning.medium}</label><input aria-label="form" type="range" min={0} max={100} value={form.threshold_tuning.medium} onChange={(e) => setForm({ ...form, threshold_tuning: { ...form.threshold_tuning, medium: parseInt(e.target.value) } })} className="w-full" /></div>
        <div><label className="block text-sm font-medium mb-1">High: {form.threshold_tuning.high}</label><input aria-label="form" type="range" min={0} max={100} value={form.threshold_tuning.high} onChange={(e) => setForm({ ...form, threshold_tuning: { ...form.threshold_tuning, high: parseInt(e.target.value) } })} className="w-full" /></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-3">
        <h2 className="text-lg font-semibold">A/B Testing</h2>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.a_b_test.enabled} onChange={(e) => setForm({ ...form, a_b_test: { ...form.a_b_test, enabled: e.target.checked } })} className="w-4 h-4" /><label>Enabled</label></div>
        <div><label className="block text-sm font-medium mb-1">Variant A (%): {form.a_b_test.variant_a_pct}</label><input aria-label="form" type="range" min={0} max={100} value={form.a_b_test.variant_a_pct} onChange={(e) => setForm({ ...form, a_b_test: { ...form.a_b_test, variant_a_pct: parseInt(e.target.value) } })} className="w-full" /></div>
        <div><label className="block text-sm font-medium mb-1">Variant B Label</label><input aria-label="form" type="text" value={form.a_b_test.variant_b_label} onChange={(e) => setForm({ ...form, a_b_test: { ...form.a_b_test, variant_b_label: e.target.value } })} className="border rounded px-3 py-2 w-full" /></div>
      </div>

      <button aria-label="action" onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
