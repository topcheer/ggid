"use client";
import { useEffect, useState } from "react";
import { useTranslations } from "@/lib/i18n";
import { useTenantProvisioningConfig, TenantProvisioningConfig, ProvisioningStep, OnboardingChecklistItem } from "@ggid/sdk-react";

export default function TenantProvisioningConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useTenantProvisioningConfig();
  const [form, setForm] = useState<TenantProvisioningConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const t = useTranslations();

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">{t("tenantProvisioning.loading")}</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">{t("tenantProvisioning.noData")}</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("tenantProvisioning.title")}</h1>
      <p className="text-gray-600">{t("tenantProvisioning.subtitle")}</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">{t("tenantProvisioning.generalSettings")}</h2>
        <div>
          <label className="block text-sm font-medium mb-1">{t("tenantProvisioning.defaultQuota")}</label>
          <select aria-label="form" value={form.default_quota_template}
            onChange={(e) => setForm({ ...form, default_quota_template: e.target.value })}
            className="border rounded px-3 py-2">
            <option value="free">{t("tenantProvisioning.free")}</option>
            <option value="starter">{t("tenantProvisioning.starter")}</option>
            <option value="business">{t("tenantProvisioning.business")}</option>
            <option value="enterprise">{t("tenantProvisioning.enterprise")}</option>
          </select>
        </div>
        <div className="flex items-center gap-3">
          <input aria-label="Toggle" type="checkbox" checked={form.auto_approve_new_tenants}
            onChange={(e) => setForm({ ...form, auto_approve_new_tenants: e.target.checked })}
            className="w-4 h-4" />
          <label>{t("tenantProvisioning.autoApprove")}</label>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">{t("tenantProvisioning.trialPeriod")}</label>
          <input aria-label="form" type="number" value={form.trial_period_days}
            onChange={(e) => setForm({ ...form, trial_period_days: parseInt(e.target.value) || 0 })}
            className="border rounded px-3 py-2 w-32" />
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("tenantProvisioning.steps")}</h2>
        <div className="space-y-2">
          {form.provisioning_steps.map((s: ProvisioningStep, i: number) => (
            <div key={i} className="flex items-center justify-between border-b py-2">
              <div>
                <span className="font-medium">{s.step}</span>
                <span className="ml-2 text-gray-500">{s.description}</span>
              </div>
              <span className={`px-2 py-1 rounded text-xs ${s.enabled ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-500"}`}>{s.enabled ? t("tenantProvisioning.enabled") : t("tenantProvisioning.disabled")}</span>
            </div>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">{t("tenantProvisioning.onboardingChecklist")}</h2>
        <div className="space-y-2">
          {form.onboarding_checklist.map((item: OnboardingChecklistItem, i: number) => (
            <div key={i} className="flex items-center gap-3 border-b py-2">
              <input aria-label="Toggle" type="checkbox" checked={item.completed} readOnly className="w-4 h-4" />
              <span className={item.completed ? "text-gray-400 line-through" : ""}>{item.item}</span>
            </div>
          ))}
        </div>
      </div>

      <button aria-label="action" onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? t("tenantProvisioning.saving") : t("tenantProvisioning.saveChanges")}</button>
    </div>
  );
}
