"use client";
import { useEffect, useState } from "react";
import { useTenantProvisioningConfig, TenantProvisioningConfig, ProvisioningStep, OnboardingChecklistItem } from "@ggid/sdk-react";

export default function TenantProvisioningConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useTenantProvisioningConfig();
  const [form, setForm] = useState<TenantProvisioningConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Tenant Provisioning Configuration</h1>
      <p className="text-gray-600">Configure automated tenant provisioning, quotas, and onboarding.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">General Settings</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Default Quota Template</label>
          <select value={form.default_quota_template}
            onChange={(e) => setForm({ ...form, default_quota_template: e.target.value })}
            className="border rounded px-3 py-2">
            <option value="free">Free</option>
            <option value="starter">Starter</option>
            <option value="business">Business</option>
            <option value="enterprise">Enterprise</option>
          </select>
        </div>
        <div className="flex items-center gap-3">
          <input type="checkbox" checked={form.auto_approve_new_tenants}
            onChange={(e) => setForm({ ...form, auto_approve_new_tenants: e.target.checked })}
            className="w-4 h-4" />
          <label>Auto Approve New Tenants</label>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Trial Period (days)</label>
          <input type="number" value={form.trial_period_days}
            onChange={(e) => setForm({ ...form, trial_period_days: parseInt(e.target.value) || 0 })}
            className="border rounded px-3 py-2 w-32" />
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Provisioning Steps</h2>
        <div className="space-y-2">
          {form.provisioning_steps.map((s: ProvisioningStep, i: number) => (
            <div key={i} className="flex items-center justify-between border-b py-2">
              <div>
                <span className="font-medium">{s.step}</span>
                <span className="ml-2 text-gray-500">{s.description}</span>
              </div>
              <span className={`px-2 py-1 rounded text-xs ${s.enabled ? "bg-green-100 text-green-700" : "bg-gray-100 text-gray-500"}`}>{s.enabled ? "Enabled" : "Disabled"}</span>
            </div>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Onboarding Checklist</h2>
        <div className="space-y-2">
          {form.onboarding_checklist.map((item: OnboardingChecklistItem, i: number) => (
            <div key={i} className="flex items-center gap-3 border-b py-2">
              <input type="checkbox" checked={item.completed} readOnly className="w-4 h-4" />
              <span className={item.completed ? "text-gray-400 line-through" : ""}>{item.item}</span>
            </div>
          ))}
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
