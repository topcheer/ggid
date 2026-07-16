"use client";
import { useTranslations } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { useCibaConfig, CibaConfig, CibaPerClient } from "@ggid/sdk-react";

export default function CibaConfigPage() {
  const t = useTranslations();
  const { config, loading, error, fetchConfig, updateConfig } = useCibaConfig();
  const [form, setForm] = useState<CibaConfig | null>(null);
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
      <h1 className="text-2xl font-bold">CIBA Configuration</h1>
      <p className="text-gray-600">Configure Client-Initiated Backchannel Authentication (CIBA / RFC 9126).</p>

      {/* Enabled toggle */}
      <div className="flex items-center gap-3 bg-white rounded-lg p-4 shadow">
        <input
          type="checkbox"
          checked={form.enabled}
          onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
          className="w-5 h-5"
        />
        <label className="font-medium">Enable CIBA</label>
      </div>

      {/* Binding Message */}
      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Binding Message</h2>
        <div className="flex items-center gap-3">
          <input
            type="checkbox"
            checked={form.binding_message.required}
            onChange={(e) => setForm({ ...form, binding_message: { ...form.binding_message, required: e.target.checked } })}
            className="w-4 h-4"
          />
          <label>Required</label>
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Max Chars</label>
          <input
            type="number"
            value={form.binding_message.max_chars}
            onChange={(e) => setForm({ ...form, binding_message: { ...form.binding_message, max_chars: parseInt(e.target.value) || 0 } })}
            className="border rounded px-3 py-2 w-48"
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Pattern (regex)</label>
          <input
            type="text"
            value={form.binding_message.pattern}
            onChange={(e) => setForm({ ...form, binding_message: { ...form.binding_message, pattern: e.target.value } })}
            className="border rounded px-3 py-2 w-full"
          />
        </div>
      </div>

      {/* Polling & Delivery */}
      <div className="bg-white rounded-lg p-6 shadow space-y-4">
        <h2 className="text-lg font-semibold">Polling & Delivery</h2>
        <div>
          <label className="block text-sm font-medium mb-1">Max Polling Interval (seconds)</label>
          <input
            type="number"
            value={form.max_polling_interval}
            onChange={(e) => setForm({ ...form, max_polling_interval: parseInt(e.target.value) || 0 })}
            className="border rounded px-3 py-2 w-48"
          />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Token Delivery Mode</label>
          <select
            value={form.token_delivery_mode}
            onChange={(e) => setForm({ ...form, token_delivery_mode: e.target.value as CibaConfig["token_delivery_mode"] })}
            className="border rounded px-3 py-2"
          >
            <option value="poll">Poll</option>
            <option value="ping">Ping</option>
            <option value="push">Push</option>
          </select>
        </div>
      </div>

      {/* Per-Client Table */}
      <div className="bg-white rounded-lg p-6 shadow">
        <h2 className="text-lg font-semibold mb-4">Per-Client Configuration</h2>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b text-left">
              <th scope="col" className="py-2">Client ID</th>
              <th scope="col">Client Name</th>
              <th scope="col">Delivery Mode</th>
              <th scope="col">Max Polling Interval</th>
            </tr>
          </thead>
          <tbody>
            {form.per_client.map((c: CibaPerClient, i: number) => (
              <tr key={i} className="border-b">
                <td className="py-2">{c.client_id}</td>
                <td>{c.client_name}</td>
                <td>{c.delivery_mode}</td>
                <td>{c.max_polling_interval}s</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Usage Stats */}
      {form.usage_stats && (
        <div className="bg-white rounded-lg p-6 shadow">
          <h2 className="text-lg font-semibold mb-4">Usage Statistics</h2>
          <div className="grid grid-cols-4 gap-4">
            <div className="text-center"><div className="text-2xl font-bold">{form.usage_stats.total_requests}</div><div className="text-xs text-gray-500">Total</div></div>
            <div className="text-center"><div className="text-2xl font-bold text-green-600">{form.usage_stats.successful}</div><div className="text-xs text-gray-500">Successful</div></div>
            <div className="text-center"><div className="text-2xl font-bold text-yellow-600">{form.usage_stats.expired}</div><div className="text-xs text-gray-500">Expired</div></div>
            <div className="text-center"><div className="text-2xl font-bold text-red-600">{form.usage_stats.denied}</div><div className="text-xs text-gray-500">Denied</div></div>
          </div>
          <div className="mt-4 text-sm text-gray-600">
            By Mode: Poll={form.usage_stats.by_mode.poll}, Ping={form.usage_stats.by_mode.ping}, Push={form.usage_stats.by_mode.push}
          </div>
        </div>
      )}

      <button onClick={handleSave} disabled={saving} className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50">
        {saving ? t("cibaConfig.saving") : t("cibaConfig.saveChanges")}
      </button>
    </div>
  );
}
