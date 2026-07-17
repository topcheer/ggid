"use client";
import { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { useHealthCheckDesignConfig, HealthCheckDesignConfig, DependencyCheck } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

interface LocalDegradationRule {
  condition: string;
  action: string;
}

interface LocalHealthCheckDesignConfig extends HealthCheckDesignConfig {
  degradation_rules: LocalDegradationRule[];
}

export default function HealthCheckDesignConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useHealthCheckDesignConfig();
  const [form, setForm] = useState<LocalHealthCheckDesignConfig | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => { fetchConfig(); }, [fetchConfig]); useEffect(() => { if (config) setForm(config as unknown as LocalHealthCheckDesignConfig); }, [config]);
  const t = useTranslations();
  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form as unknown as Parameters<typeof updateConfig>[0]); setSaving(false); };
  if (loading && !form) return <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8 text-gray-400">No data</div>;
  const statusColors: Record<string, string> = { healthy: "bg-green-100 text-green-700", degraded: "bg-yellow-100 text-yellow-700", down: "bg-red-100 text-red-700" };

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">{t("healthCheckDesign.title")}</h1>
      <p className="text-gray-600">{t("healthCheckDesign.subtitle")}</p>
      <div className="bg-white rounded-lg p-6 shadow space-y-4"><h2 className="text-lg font-semibold">{t("healthCheckDesign.checkTypes")}</h2><div className="text-sm text-gray-600">{form.check_types.join(", ")}</div><div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.circuit_breaker_integration} onChange={(e) => setForm({ ...form, circuit_breaker_integration: e.target.checked })} className="w-4 h-4" /><label>{t("healthCheckDesign.circuitBreaker")}</label></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.auto_healing} onChange={(e) => setForm({ ...form, auto_healing: e.target.checked })} className="w-4 h-4" /><label>{t("healthCheckDesign.autoHealing")}</label></div><div className="flex items-center gap-3"><input type="checkbox" checked={form.lb_integration} onChange={(e) => setForm({ ...form, lb_integration: e.target.checked })} className="w-4 h-4" /><label>{t("healthCheckDesign.loadBalancer")}</label></div></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">{t("healthCheckDesign.dependencyChecks")}</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("healthCheckDesign.dependency")}</th><th scope="col">{t("healthCheckDesign.status")}</th><th>{t("healthCheckDesign.latency")}</th></tr></thead><tbody>{form.dependency_checks.map((d: DependencyCheck, i: number) => (<tr key={i} className="border-b"><td className="py-2 font-medium">{d.name}</td><td><span className={`px-2 py-1 rounded text-xs ${statusColors[d.status] || ""}`}>{d.status}</span></td><td>{d.latency_ms}ms</td></tr>))}</tbody></table></div>
      <div className="bg-white rounded-lg p-6 shadow"><h2 className="text-lg font-semibold mb-4">{t("healthCheckDesign.degradationRules")}</h2><table className="w-full text-sm"><thead><tr className="border-b text-left"><th className="py-2">{t("healthCheckDesign.condition")}</th><th scope="col">{t("healthCheckDesign.action")}</th></tr></thead><tbody>{form.degradation_rules.map((r, i) => (<tr key={i} className="border-b"><td className="py-2">{r.condition}</td><td className="text-xs">{r.action}</td></tr>))}</tbody></table></div>
      <button onClick={handleSave} disabled={saving} aria-label={t("healthCheckDesign.save")} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? t("healthCheckDesign.saving") : t("healthCheckDesign.saveChanges")}</button>
    </div>
  );
}
