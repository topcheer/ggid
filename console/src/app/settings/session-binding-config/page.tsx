"use client";
import { useEffect, useState } from "react";
import { useSessionBindingConfig, SessionBindingConfig, PerAppBinding } from "@ggid/sdk-react";

export default function SessionBindingConfigPage() {
  const { config, loading, error, fetchConfig, updateConfig } = useSessionBindingConfig();
  const [form, setForm] = useState<SessionBindingConfig | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => {
    if (!form) return;
    setSaving(true);
    await updateConfig(form);
    setSaving(false);
  };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Session Binding Configuration</h1>
      <p className="text-gray-600">Configure session binding methods for enhanced security.</p>

      {/* Global Binding Method */}
      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Global Binding Method</h2>
        <select
          value={form.binding_method}
          onChange={(e) => setForm({ ...form, binding_method: e.target.value as SessionBindingConfig["binding_method"] })}
          className="border rounded px-3 py-2"
        >
          <option value="cookie">Cookie</option>
          <option value="bearer">Bearer Token</option>
          <option value="DPoP">DPoP</option>
          <option value="mTLS">mTLS</option>
        </select>
        <div>
          <label className="block text-sm font-medium mb-1">Fallback Method</label>
          <select
            value={form.fallback_method}
            onChange={(e) => setForm({ ...form, fallback_method: e.target.value as SessionBindingConfig["fallback_method"] })}
            className="border rounded px-3 py-2"
          >
            <option value="cookie">Cookie</option>
            <option value="bearer">Bearer Token</option>
          </select>
        </div>
      </div>

      {/* Session Hijack Protection */}
      <div className="flex items-center gap-3 bg-white rounded-lg p-4 shadow">
        <input
          type="checkbox"
          checked={form.session_hijack_protection}
          onChange={(e) => setForm({ ...form, session_hijack_protection: e.target.checked })}
          className="w-5 h-5"
        />
        <label className="font-medium">Session Hijack Protection</label>
      </div>

      {/* Binding Rotation Policy */}
      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Binding Rotation Policy</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Rotation Interval (seconds)</label>
          <input
            type="number"
            value={form.binding_rotation_policy.interval_seconds}
            onChange={(e) => setForm({ ...form, binding_rotation_policy: { ...form.binding_rotation_policy, interval_seconds: parseInt(e.target.value) || 0 } })}
            className="border rounded px-3 py-2 w-48"
          />
        </div>
        <div className="flex items-center gap-3">
          <input
            type="checkbox"
            checked={form.binding_rotation_policy.rotate_on_reauth}
            onChange={(e) => setForm({ ...form, binding_rotation_policy: { ...form.binding_rotation_policy, rotate_on_reauth: e.target.checked } })}
            className="w-4 h-4"
          />
          <label>Rotate on re-authentication</label>
        </div>
      </div>

      {/* Cross-Device Transfer */}
      <div className="bg-white rounded-lg p-6 shadow space-y-3">
        <h2 className="text-lg font-semibold">Cross-Device Transfer</h2>
        <div className="flex items-center gap-3">
          <input
            type="checkbox"
            checked={form.cross_device_transfer.enabled}
            onChange={(e) => setForm({ ...form, cross_device_transfer: { ...form.cross_device_transfer, enabled: e.target.checked } })}
            className="w-4 h-4"
          />
          <label>Enabled</label>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Max Transfer Window (seconds)</label>
          <input
            type="number"
            value={form.cross_device_transfer.max_transfer_window}
            onChange={(e) => setForm({ ...form, cross_device_transfer: { ...form.cross_device_transfer, max_transfer_window: parseInt(e.target.value) || 0 } })}
            className="border rounded px-3 py-2 w-48"
          />
        </div>
      </div>

      {/* Per-Application Binding Table */}
      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Per-Application Binding</h2>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left">
              <th className="py-2">Application ID</th>
              <th>Name</th>
              <th>Binding Method</th>
              <th>Rotation Interval</th>
            </tr>
          </thead>
          <tbody>
            {form.per_application_binding.map((b: PerAppBinding, i: number) => (
              <tr key={i} className="border-b">
                <td className="py-2">{b.application_id}</td>
                <td>{b.application_name}</td>
                <td>{b.binding_method}</td>
                <td>{b.rotation_interval}s</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
        {saving ? "Saving..." : "Save Changes"}
      </button>
    </div>
  );
}
