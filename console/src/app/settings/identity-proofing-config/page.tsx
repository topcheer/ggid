"use client";
import { useEffect, useState } from "react";
import { useIdentityProofingConfig, IdentityProofingConfig, VerificationMethod, RiskLevelConfig } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function IdentityProofingConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useIdentityProofingConfig();
  const [form, setForm] = useState<IdentityProofingConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">{t("big1.identityProofingConfig.loading")}</div>;
  if (error) return <div className="p-8 text-red-600">{t("big1.identityProofingConfig.error")}{error}</div>;
  if (!form) return <div className="p-8">{t("big1.identityProofingConfig.noData")}</div>;

  const completionPct = form.completion_rate.total > 0 ? Math.round((form.completion_rate.completed / form.completion_rate.total) * 100) : 0;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("big1.identityProofingConfig.title")}</h1>
      <p className="text-gray-600">{t("big1.identityProofingConfig.configureIdentityVerificationMethodsAndConfidenceThresholds")}</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">{t("big1.identityProofingConfig.verificationMethods")}</h2>
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
        <h2 className="text-lg font-semibold">{t("big1.identityProofingConfig.requirements")}</h2>
        <div>
          <label className="block text-sm font-medium mb-1">{t("big1.identityProofingConfig.requiredFactors")}{form.required_factors}</label>
          <input type="range" min={1} max={5} value={form.required_factors} onChange={(e) => setForm({ ...form, required_factors: parseInt(e.target.value) })} className="w-full" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t("big1.identityProofingConfig.confidenceThreshold")}{form.confidence_threshold}%</label>
          <input type="range" min={50} max={100} value={form.confidence_threshold} onChange={(e) => setForm({ ...form, confidence_threshold: parseInt(e.target.value) })} className="w-full" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t("big1.identityProofingConfig.verificationProvider")}</label>
          <select value={form.verification_provider} onChange={(e) => setForm({ ...form, verification_provider: e.target.value })} className="border rounded px-3 py-2">
            <option value="onfido">{t("big1.identityProofingConfig.onfido")}</option><option value="jumio">{t("big1.identityProofingConfig.jumio")}</option><option value="idology">{t("big1.identityProofingConfig.idology")}</option><option value="internal">{t("big1.identityProofingConfig.internal")}</option>
          </select>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("big1.identityProofingConfig.perRiskLevelMatrix")}</h2>
        <table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("big1.identityProofingConfig.riskLevel")}</th><th scope="col">{t("big1.identityProofingConfig.requiredFactors")}</th><th>{t("big1.identityProofingConfig.methods")}</th></tr></thead><tbody>
          {form.per_risk_level.map((r: RiskLevelConfig, i: number) => (
            <tr key={i} className="border-b"><td className="py-2 font-medium">{r.level}</td><td>{r.required_factors}</td><td>{r.methods.join(", ")}</td></tr>
          ))}
        </tbody></table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("big1.identityProofingConfig.completionRate")}</h2>
        <div className="grid grid-cols-4 gap-4">
          <div className="text-center"><div className="text-2xl font-bold">{form.completion_rate.total}</div><div className="text-xs text-gray-500">{t("big1.identityProofingConfig.total")}</div></div>
          <div className="text-center"><div className="text-2xl font-bold text-green-600">{form.completion_rate.completed}</div><div className="text-xs text-gray-500">{t("big1.identityProofingConfig.completed")}</div></div>
          <div className="text-center"><div className="text-2xl font-bold text-red-600">{form.completion_rate.failed}</div><div className="text-xs text-gray-500">{t("big1.identityProofingConfig.failed")}</div></div>
          <div className="text-center"><div className="text-2xl font-bold text-blue-600">{completionPct}%</div><div className="text-xs text-gray-500">{t("big1.identityProofingConfig.rate")}</div></div>
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? t("big1.identityProofingConfig.saving") : t("big1.identityProofingConfig.saveChanges")}</button>
    </div>
  );
}
