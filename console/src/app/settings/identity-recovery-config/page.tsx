"use client";
import { useEffect, useState } from "react";
import { useIdentityRecoveryConfig, IdentityRecoveryConfig } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function IdentityRecoveryConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig } = useIdentityRecoveryConfig();
  const [form, setForm] = useState<IdentityRecoveryConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  if (loading && !form) return <div className="p-8">{t("big1.identityRecoveryConfig.loading")}</div>;
  if (error) return <div className="p-8 text-red-600">{t("big1.identityRecoveryConfig.error")}{error}</div>;
  if (!form) return <div className="p-8">{t("big1.identityRecoveryConfig.noData")}</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("big1.identityRecoveryConfig.title")}</h1>
      <p className="text-gray-600">{t("big1.identityRecoveryConfig.configureAccountTakeoverResponseMassResetAndForensicsCollection")}</p>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">{t("big1.identityRecoveryConfig.takeoverResponseChecklist")}</h2><div className="space-y-1">{form.takeover_response_checklist.map((item: string, i: number) => (<div key={i} className="flex items-center gap-3 border-b py-1"><input aria-label="Toggle" type="checkbox" readOnly className="w-4 h-4" /><span className="text-sm">{item}</span></div>))}</div></div>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">{t("big1.identityRecoveryConfig.recoveryActions")}</h2><div><label className="block text-sm font-medium mb-1">{t("big1.identityRecoveryConfig.sessionInvalidationScope")}</label><select aria-label="Select option" value={form.session_invalidation_scope} onChange={(e) => setForm({ ...form, session_invalidation_scope: e.target.value as IdentityRecoveryConfig["session_invalidation_scope"] })} className="border rounded px-3 py-2"><option value="affected">{t("big1.identityRecoveryConfig.affectedUsers")}</option><option value="tenant">{t("big1.identityRecoveryConfig.entireTenant")}</option><option value="global">{t("big1.identityRecoveryConfig.global")}</option></select></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.forensics_auto_collect} onChange={(e) => setForm({ ...form, forensics_auto_collect: e.target.checked })} className="w-4 h-4" /><label>{t("big1.identityRecoveryConfig.forensicsAutoCollection")}</label></div><div><label className="block text-sm font-medium mb-1">{t("big1.identityRecoveryConfig.notificationChannels")}</label><div className="text-sm text-gray-600">{form.notification_channels.join(", ")}</div></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">{t("big1.identityRecoveryConfig.massResetTemplate")}</h2><pre className="bg-gray-50 rounded p-4 text-xs overflow-x-auto whitespace-pre-wrap">{form.mass_reset_template}</pre></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">{t("big1.identityRecoveryConfig.userCommunicationTemplate")}</h2><pre className="bg-gray-50 rounded p-4 text-xs overflow-x-auto whitespace-pre-wrap">{form.user_communication_template}</pre></div>
      <button aria-label="action" onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? t("big1.identityRecoveryConfig.saving") : t("big1.identityRecoveryConfig.saveChanges")}</button>
    </div>
  );
}
