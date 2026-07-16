"use client";
import { useEffect, useState } from "react";
import { useJwtClaimValidationConfig, JwtClaimValidationConfig, CustomClaim, RequiredClaim } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function JwtClaimValidationConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useJwtClaimValidationConfig();
  const [form, setForm] = useState<JwtClaimValidationConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">{t("big1.jwtClaimValidationConfig.loading")}</div>;
  if (error) return <div className="p-8 text-red-600">{t("big1.jwtClaimValidationConfig.error")}{error}</div>;
  if (!form) return <div className="p-8">{t("big1.jwtClaimValidationConfig.noData")}</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("big1.jwtClaimValidationConfig.title")}</h1>
      <p className="text-gray-600">{t("big1.jwtClaimValidationConfig.configureJWTTokenClaimValidationRules")}</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold">{t("big1.jwtClaimValidationConfig.requiredClaims")}</h2>
          <div className="flex items-center gap-3">
            <input type="checkbox" checked={form.strict_mode}
              onChange={(e) => setForm({ ...form, strict_mode: e.target.checked })}
              className="w-4 h-4" />
            <label className="text-sm font-medium">{t("big1.jwtClaimValidationConfig.strictMode")}</label>
          </div>
        </div>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">{t("big1.jwtClaimValidationConfig.claim")}</th><th scope="col">{t("big1.jwtClaimValidationConfig.enabled")}</th></tr></thead>
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
        <h2 className="text-lg font-semibold">{t("big1.jwtClaimValidationConfig.validationSettings")}</h2>
        <div>
          <label className="block text-sm font-medium mb-1">{t("big1.jwtClaimValidationConfig.clockSkewSeconds")}</label>
          <input type="number" value={form.clock_skew_seconds}
            onChange={(e) => setForm({ ...form, clock_skew_seconds: parseInt(e.target.value) || 0 })}
            className="border rounded px-3 py-2 w-32" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t("big1.jwtClaimValidationConfig.validationOrder")}</label>
          <div className="text-sm text-gray-600">{form.validation_order.join(" -> ")}</div>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("big1.jwtClaimValidationConfig.customClaims")}</h2>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">{t("big1.jwtClaimValidationConfig.name")}</th><th scope="col">{t("big1.jwtClaimValidationConfig.type")}</th><th>{t("big1.jwtClaimValidationConfig.required")}</th><th>{t("big1.jwtClaimValidationConfig.validator")}</th></tr></thead>
          <tbody>
            {form.custom_claims.map((c: CustomClaim, i: number) => (
              <tr key={i} className="border-b"><td className="py-2 font-mono">{c.name}</td><td>{c.type}</td><td>{c.required ? "Yes" : "No"}</td><td className="text-xs">{c.validator}</td></tr>
            ))}
          </tbody>
        </table>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? t("big1.jwtClaimValidationConfig.saving") : t("big1.jwtClaimValidationConfig.saveChanges")}</button>
    </div>
  );
}
