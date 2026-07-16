"use client";
import { useEffect, useState } from "react";
import { useWebauthnRecoveryConfig, WebauthnRecoveryConfig, ReEnrollmentStep } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function WebauthnRecoveryConfigPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig } = useWebauthnRecoveryConfig();
  const [form, setForm] = useState<WebauthnRecoveryConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">WebAuthn Recovery Configuration</h1>
      <p className="text-gray-600">Configure backup authenticators, recovery codes, and re-enrollment.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Device & Recovery Settings</h2>
        <div className="flex items-center gap-3"><input aria-label="Form" type="checkbox" checked={form.backup_authenticator_required} onChange={(e) => setForm({ ...form, backup_authenticator_required: e.target.checked })} className="w-4 h-4" /><label>Backup Authenticator Required</label></div>
        <div><label className="block text-sm font-medium mb-1">Max Devices Per User</label><input aria-label="form" type="number" value={form.max_devices_per_user} onChange={(e) => setForm({ ...form, max_devices_per_user: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
        <div><label className="block text-sm font-medium mb-1">Recovery Codes Count</label><input aria-label="form" type="number" value={form.recovery_codes_count} onChange={(e) => setForm({ ...form, recovery_codes_count: parseInt(e.target.value) || 0 })} className="border rounded px-3 py-2 w-32" /></div>
        <div><label className="block text-sm font-medium mb-1">Recovery Code Format</label><select aria-label="form" value={form.recovery_code_format} onChange={(e) => setForm({ ...form, recovery_code_format: e.target.value as WebauthnRecoveryConfig["recovery_code_format"] })} className="border rounded px-3 py-2"><option value="numeric">Numeric</option><option value="alphanumeric">Alphanumeric</option><option value="hex">Hex</option></select></div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Re-Enrollment Flow</h2>
        <div className="space-y-2">
          {form.re_enrollment_flow.map((s: ReEnrollmentStep, i: number) => (
            <div key={i} className="flex items-center justify-between border-b py-2">
              <div><span className="font-medium">{s.step}</span><span className="ml-2 text-sm text-gray-500">{s.description}</span></div>
              <span className={`px-2 py-1 rounded text-xs ${s.required ? "bg-blue-100 text-blue-700" : "bg-gray-100 text-gray-500"}`}>{s.required ? "Required" : "Optional"}</span>
            </div>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <div className="flex items-center gap-3">
          <input aria-label="Form" type="checkbox" checked={form.admin_assisted_recovery} onChange={(e) => setForm({ ...form, admin_assisted_recovery: e.target.checked })} className="w-4 h-4" />
          <label className="font-medium">Admin Assisted Recovery</label>
        </div>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
