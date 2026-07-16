"use client";
import { useTranslations } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { useUserLifecycleConfig, UserLifecycleConfig, DormantDetectionRule, StageTransitionRule, PerRoleOverride } from "@ggid/sdk-react";

export default function UserLifecycleConfigPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig } = useUserLifecycleConfig();
  const [form, setForm] = useState<UserLifecycleConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">User Lifecycle Configuration</h1>
      <p className="text-gray-600">Configure automatic deactivation, dormant detection, and stage transitions.</p>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Auto Deactivation</h2>
        <label className="block text-sm font-medium mb-2">Auto Deactivate After (days): {form.auto_deactivate_after_days}</label>
        <input type="range" min={30} max={365} value={form.auto_deactivate_after_days}
          onChange={(e) => setForm({ ...form, auto_deactivate_after_days: parseInt(e.target.value) })}
          className="w-full" />
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Dormant Detection Rules</h2>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">Metric</th><th scope="col">Threshold (days)</th><th>Enabled</th></tr></thead>
          <tbody>
            {form.dormant_detection_rules.map((r: DormantDetectionRule, i: number) => (
              <tr key={i} className="border-b">
                <td className="py-2">{r.metric}</td><td>{r.threshold_days}</td>
                <td><input aria-label="Toggle" type="checkbox" checked={r.enabled} readOnly className="w-4 h-4" /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Stage Transition Rules</h2>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">From</th><th scope="col">To</th><th>Condition</th><th>Auto</th></tr></thead>
          <tbody>
            {form.stage_transition_rules.map((r: StageTransitionRule, i: number) => (
              <tr key={i} className="border-b"><td className="py-2">{r.from_stage}</td><td>{r.to_stage}</td><td>{r.condition}</td><td>{r.auto ? "Yes" : "No"}</td></tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-3">
        <h2 className="text-lg font-semibold">Notification Before Deactivation</h2>
        <div className="flex items-center gap-3">
          <input aria-label="Toggle" type="checkbox" checked={form.notification_before_deactivate.enabled}
            onChange={(e) => setForm({ ...form, notification_before_deactivate: { ...form.notification_before_deactivate, enabled: e.target.checked } })}
            className="w-4 h-4" />
          <label>Enabled</label>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Days Before</label>
          <input type="number" value={form.notification_before_deactivate.days_before}
            onChange={(e) => setForm({ ...form, notification_before_deactivate: { ...form.notification_before_deactivate, days_before: parseInt(e.target.value) || 0 } })}
            className="border rounded px-3 py-2 w-32" />
        </div>
        <div className="text-sm text-gray-500">Channels: {form.notification_before_deactivate.channels.join(", ")}</div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Per-Role Overrides</h2>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">Role</th><th scope="col">Deactivate After</th><th>Notify Before</th></tr></thead>
          <tbody>
            {form.per_role_override.map((o: PerRoleOverride, i: number) => (
              <tr key={i} className="border-b"><td className="py-2">{o.role}</td><td>{o.deactivate_after_days} days</td><td>{o.notify_before_days} days</td></tr>
            ))}
          </tbody>
        </table>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
