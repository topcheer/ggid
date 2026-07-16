"use client";
import { useEffect, useState } from "react";
import { useDataSovereigntyConfig, DataSovereigntyConfig, ResidencyRegion, CrossBorderRule } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

interface LocalSovereigntyViolation {
  region: string;
  description: string;
  timestamp: string;
  severity: "high" | "medium" | "low";
}

interface LocalDataSovereigntyConfig extends DataSovereigntyConfig {
  sovereignty_violations: LocalSovereigntyViolation[];
}

export default function DataSovereigntyConfigPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig } = useDataSovereigntyConfig();
  const [form, setForm] = useState<LocalDataSovereigntyConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config as unknown as LocalDataSovereigntyConfig); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form as unknown as Parameters<typeof updateConfig>[0]); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Data Sovereignty Configuration</h1>
      <p className="text-gray-600">Configure data residency, cross-border transfer, and GDPR compliance.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Residency Regions</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Region</th><th scope="col">Allowed</th><th>Encryption Required</th></tr></thead><tbody>
          {form.residency_regions.map((r: ResidencyRegion, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-medium">{r.region}</td><td>{r.allowed ? "Yes" : "No"}</td><td>{r.encryption_required ? "Yes" : "No"}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Cross-Border Transfer Rules</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">From</th><th scope="col">To</th><th>Allowed</th><th>Condition</th></tr></thead><tbody>
          {form.cross_border_transfer_rules.map((r: CrossBorderRule, i: number) => (
            <tr key={i} className="border-b"><td className="py-2">{r.from_region}</td><td>{r.to_region}</td><td><span className={`px-2 py-1 rounded text-xs ${r.allowed ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}>{r.allowed ? "Yes" : "No"}</span></td><td className="text-xs">{r.condition}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-3">
        <h2 className="text-lg font-semibold">GDPR Compliance</h2>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.gdpr_article_45_compliant} onChange={(e) => setForm({ ...form, gdpr_article_45_compliant: e.target.checked })} className="w-4 h-4" /><label>Article 45 (Adequacy Decision) Compliant</label></div>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.gdpr_article_49_compliant} onChange={(e) => setForm({ ...form, gdpr_article_49_compliant: e.target.checked })} className="w-4 h-4" /><label>Article 49 (Derogations) Compliant</label></div>
        <div className="text-sm text-gray-500">Data Localization Status: {form.data_localization_status}</div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Sovereignty Violations</h2>
        <div className="space-y-2">
          {form.sovereignty_violations.map((v, i) => (
            <div key={i} className="flex items-center justify-between border-b py-2">
              <div><span className="font-medium">{v.region}</span><span className="ml-2 text-sm text-gray-500">{v.description}</span></div>
              <div className="flex items-center gap-3">
                <span className="text-xs text-gray-400">{v.timestamp}</span>
                <span className={`px-2 py-1 rounded text-xs ${v.severity === "high" ? "bg-red-100 text-red-700" : v.severity === "medium" ? "bg-yellow-100 text-yellow-700" : "bg-gray-100 text-gray-500"}`}>{v.severity}</span>
              </div>
            </div>
          ))}
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} aria-label="Save data sovereignty config" className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
