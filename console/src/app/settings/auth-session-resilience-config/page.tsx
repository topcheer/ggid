"use client";
import { useEffect, useState } from "react";
import { useAuthSessionResilienceConfig, AuthSessionResilienceConfig } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function AuthSessionResilienceConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig, testRecovery } = useAuthSessionResilienceConfig();
  const [form, setForm] = useState<AuthSessionResilienceConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  const handleTest = async () => { setTesting(true); await testRecovery(); setTesting(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Auth Session Resilience Configuration</h1>
      <p className="text-gray-600">Configure session failover, degraded mode, and recovery.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Failover Configuration</h2>
        <div><label className="block text-sm font-medium mb-1">Primary Redis</label><input aria-label="form" type="text" value={form.failover_config.primary_redis} onChange={(e) => setForm({ ...form, failover_config: { ...form.failover_config, primary_redis: e.target.value } })} className="border rounded px-3 py-2 w-full" /></div>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.failover_config.fallback_memory} onChange={(e) => setForm({ ...form, failover_config: { ...form.failover_config, fallback_memory: e.target.checked } })} className="w-4 h-4" /><label>Fallback to In-Memory</label></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Resilience Settings</h2>
        <div><label className="block text-sm font-medium mb-1">Grace Period During Outage (s)</label><input aria-label="form" type="number" value={form.grace_period_during_outage_seconds} onChange={(e) => setForm({ ...form, grace_period_during_outage_seconds: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.offline_token_validation} onChange={(e) => setForm({ ...form, offline_token_validation: e.target.checked })} className="w-4 h-4" /><label>Offline Token Validation</label></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Degraded Mode Indicators</h2>
        <div className="space-y-1">
          {form.degraded_mode_indicators.map((ind: string, i: number) => (
            <div key={i} className="border-b py-1 text-sm">{ind}</div>
          ))}
        </div>
      </div>

      <div className="flex gap-4">
        <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
        <button onClick={handleTest} disabled={testing} className="px-6 py-2 border rounded-lg hover:bg-gray-50 disabled:opacity-50">{testing ? "Testing..." : "Test Recovery"}</button>
      </div>
    </div>
  );
}
