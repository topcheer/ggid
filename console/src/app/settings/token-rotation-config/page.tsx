"use client";
import { useEffect, useState } from "react";
import { useTokenRotationConfig, TokenRotationConfig, TokenRotationEntry, UpcomingRotation } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";

export default function TokenRotationConfigPage() {
  const t = useTranslations();

  const { config, loading, error, fetchConfig, updateConfig, bulkUpdate } = useTokenRotationConfig();
  const [form, setForm] = useState<TokenRotationConfig | null>(null);
  const [saving, setSaving] = useState(false);
  const [bulkDays, setBulkDays] = useState(90);
  const [bulkAuto, setBulkAuto] = useState(true);

  useEffect(() => { fetchConfig(); }, [fetchConfig]);
  useEffect(() => { if (config) setForm(config); }, [config]);

  const handleSave = async () => { if (!form) return; setSaving(true); await updateConfig(form); setSaving(false); };
  const handleBulk = async () => { setSaving(true); await bulkUpdate(bulkDays, bulkAuto); setSaving(false); };

  if (loading && !form) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-600">Error: {error}</div>;
  if (!form) return <div className="p-8">No data</div>;

  return (
    <div className="p-8 space-y-6 max-w-4xl">
      <h1 className="text-2xl font-bold">Token Rotation Configuration</h1>
      <p className="text-gray-600">Configure per-client token rotation policies and grace periods.</p>

      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Bulk Update</h2>
        <div className="flex items-center gap-4">
          <div>
            <label className="block text-sm font-medium mb-1">Rotation Interval (days)</label>
            <input aria-label="bulk Days" type="number" value={bulkDays} onChange={(e) => setBulkDays(parseInt(e.target.value) || 0)} className="border rounded px-3 py-2 w-32" />
          </div>
          <div className="flex items-center gap-2 pt-6">
            <input aria-label="Bulk auto" type="checkbox" checked={bulkAuto} onChange={(e) => setBulkAuto(e.target.checked)} className="w-4 h-4" />
            <label>Auto Rotate</label>
          </div>
          <button onClick={handleBulk} disabled={saving} className="px-4 py-2 bg-purple-600 text-white rounded text-sm hover:bg-purple-700 disabled:opacity-50 ml-4 mt-4">Apply to All</button>
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow space-y-3">
        <h2 className="text-lg font-semibold">Grace Period</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Grace Period (hours)</label>
          <input aria-label="form" type="number" value={form.grace_period_hours}
            onChange={(e) => setForm({ ...form, grace_period_hours: parseInt(e.target.value) || 0 })}
            className="border rounded px-3 py-2 w-32" />
        </div>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Per-Client Rotation</h2>
        <table className="w-full text-sm">
          <thead><tr className="border-b text-left"><th className="py-2">Client</th><th scope="col">Interval (days)</th><th>Max Age (days)</th><th>Notify Before</th><th>Auto</th><th>Last Rotated</th></tr></thead>
          <tbody>
            {form.per_client.map((c: TokenRotationEntry, i: number) => (
              <tr key={i} className="border-b">
                <td className="py-2"><span className="font-medium">{c.client_name}</span><div className="text-xs text-gray-400">{c.client_id}</div></td>
                <td>{c.rotation_interval_days}</td><td>{c.max_age_days}</td><td>{c.notify_before_days}d</td>
                <td>{c.auto_rotate ? "Yes" : "No"}</td><td className="text-xs text-gray-500">{c.last_rotated}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Upcoming Rotations</h2>
        <div className="space-y-2">
          {form.upcoming_rotations.map((u: UpcomingRotation, i: number) => (
            <div key={i} className="flex items-center justify-between border-b py-2">
              <div><span className="font-medium">{u.client_name}</span><span className="ml-2 text-gray-500">{u.client_id}</span></div>
              <div className="text-sm">
                <span className="text-gray-500">Due: {u.rotation_due}</span>
                <span className={`ml-3 px-2 py-0.5 rounded text-xs ${u.days_until <= 7 ? "bg-red-100 text-red-700" : "bg-yellow-100 text-yellow-700"}`}>{u.days_until} days</span>
              </div>
            </div>
          ))}
        </div>
      </div>

      <button aria-label="action" onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">{saving ? "Saving..." : "Save Changes"}</button>
    </div>
  );
}
