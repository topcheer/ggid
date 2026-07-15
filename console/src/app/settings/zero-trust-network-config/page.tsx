"use client";
import { useEffect, useState } from "react";
import { useZeroTrustNetworkConfig, ZeroTrustNetworkConfig, DeviceTrustSignal, NetworkAccessPolicy } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function ZeroTrustNetworkConfigPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig } = useZeroTrustNetworkConfig();
  const [form, setForm] = useState<ZeroTrustNetworkConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Zero Trust Network Configuration</h1>
      <p className="text-gray-600">Configure identity-aware proxy, continuous verification, and microsegmentation.</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">Core Settings</h2><div className="flex items-center gap-3"><input type="checkbox" checked={form.identity_aware_proxy} onChange={(e) => setForm({ ...form, identity_aware_proxy: e.target.checked })} className="w-4 h-4" /><label>Identity-Aware Proxy</label></div><div><label className="block text-sm font-medium mb-1">Continuous Verification Interval (s)</label><input type="number" value={form.continuous_verification_interval} onChange={(e) => setForm({ ...form, continuous_verification_interval: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Device Trust Signals</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Signal</th><th>Source</th><th>Weight</th></tr></thead><tbody>{form.device_trust_signals.map((s: DeviceTrustSignal, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{s.signal}</td><td>{s.source}</td><td>{s.weight}</td></tr>))}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Network Access Policy</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Resource</th><th>Required Trust</th><th>Condition</th></tr></thead><tbody>{form.network_access_policy.map((p: NetworkAccessPolicy, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{p.resource}</td><td><span className={`px-2 py-1 rounded text-xs ${p.required_trust_level === "high" ? "bg-red-100 text-red-700" : p.required_trust_level === "medium" ? "bg-yellow-100 text-yellow-700" : "bg-gray-100 text-gray-500"}`}>{p.required_trust_level}</span></td><td className="text-xs">{p.condition}</td></tr>))}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">Microsegmentation Rules</h2><div className="space-y-1">{form.microsegmentation_rules.map((r: string, i: number) => (<div key={i} className="border-b py-1 font-mono text-xs">{r}</div>))}</div></div>
      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
