"use client";
import { useEffect, useState } from "react";
import { useTranslations } from "@/lib/i18n";
import { useTenantIsolationConfig, TenantIsolationConfig, IsolationTestResult } from "@ggid/sdk-react";

export default function TenantIsolationConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig, testIsolation } = useTenantIsolationConfig();
  const [form, setForm] = useState<TenantIsolationConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const t = useTranslations();
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config); }, [config]);
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  const handleTest = async () => { setTesting(true); await testIsolation(); setTesting(false); };
  if (loading && !form) return <div className="p-8">{t("tenantIsolationConfig.loading")}</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">{t("tenantIsolationConfig.noData")}</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("tenantIsolationConfig.title")}</h1>
      <p className="text-gray-600">{t("tenantIsolationConfig.subtitle")}</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">{t("tenantIsolationConfig.isolationSettings")}</h2><div><label className="block text-sm font-medium mb-1">{t("tenantIsolationConfig.isolationMode")}</label><select aria-label="Select option" value={form.isolation_mode} onChange={(e) => setForm({ ...form, isolation_mode: e.target.value as TenantIsolationConfig["isolation_mode"] })} className="border rounded px-3 py-2"><option value="rls">{t("tenantIsolationConfig.rowLevelSecurity")}</option><option value="schema">{t("tenantIsolationConfig.schemaPerTenant")}</option><option value="hybrid">{t("tenantIsolationConfig.hybrid")}</option></select></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.per_tenant_redis_namespace} onChange={(e) => setForm({ ...form, per_tenant_redis_namespace: e.target.checked })} className="w-4 h-4" /><label>{t("tenantIsolationConfig.redisNamespace")}</label></div><div><label className="block text-sm font-medium mb-1">{t("tenantIsolationConfig.natsSubjectPrefix")}</label><input type="text" value={form.nats_subject_prefix} onChange={(e) => setForm({ ...form, nats_subject_prefix: e.target.value })} className="border rounded px-3 py-2 w-full" /></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.file_storage_isolation} onChange={(e) => setForm({ ...form, file_storage_isolation: e.target.checked })} className="w-4 h-4" /><label>{t("tenantIsolationConfig.fileStorageIsolation")}</label></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">{t("tenantIsolationConfig.testResults")}</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("tenantIsolationConfig.test")}</th><th scope="col">{t("tenantIsolationConfig.passed")}</th><th>{t("tenantIsolationConfig.detail")}</th></tr></thead><tbody>{form.isolation_test_results.map((r: IsolationTestResult, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{r.test}</td><td><span className={`px-2 py-1 rounded text-xs ${r.passed ? "bg-green-100 text-green-700" : "bg-red-100 text-red-700"}`}>{r.passed ? t("tenantIsolationConfig.pass") : t("tenantIsolationConfig.fail")}</span></td><td className="text-xs">{r.detail}</td></tr>))}</tbody></table></div>
      <div className="flex gap-4"><button aria-label="action" onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? t("tenantIsolationConfig.saving") : t("tenantIsolationConfig.save")}</button><button onClick={handleTest} disabled={testing} className="px-6 py-2 border rounded-lg hover:bg-gray-50 disabled:opacity-50" aria-label="Action">{testing ? "Testing..." : "Test Isolation"}</button></div>
    </div>
  );
}
