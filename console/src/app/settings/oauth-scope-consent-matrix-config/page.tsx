"use client";
import { useEffect, useState } from "react";
import { useOAuthScopeConsentMatrixConfig, OAuthScopeConsentMatrixConfig, ScopeConsentEntry } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function OAuthScopeConsentMatrixConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useOAuthScopeConsentMatrixConfig();
  const [form, setForm] = useState<OAuthScopeConsentMatrixConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  const levelColors: Record<string, string> = { none: "bg-gray-100 text-gray-600", implicit: "bg-blue-100 text-blue-700", explicit: "bg-yellow-100 text-yellow-700", admin_required: "bg-red-100 text-red-700" };
  const riskColors: Record<string, string> = { low: "bg-green-100 text-green-700", medium: "bg-yellow-100 text-yellow-700", high: "bg-red-100 text-red-700" };

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">OAuth Scope Consent Matrix</h1>
      <p className="text-gray-600">Configure consent levels and risk per scope.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">Scope Matrix</h2>
          <div className="text-sm">Compliance: <span className="font-bold text-blue-600">{form.compliance_summary_pct}%</span></div>
        </div>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">Scope</th><th>Consent Level</th><th>Risk Level</th></tr></thead><tbody>
          {form.matrix.map((e: ScopeConsentEntry, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-mono">{e.scope}</td><td><span className={`px-2 py-1 rounded text-xs ${levelColors[e.consent_level] || ""}`}>{e.consent_level}</span></td><td><span className={`px-2 py-1 rounded text-xs ${riskColors[e.risk_level] || ""}`}>{e.risk_level}</span></td></tr>
          ))}
        </tbody></table>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
