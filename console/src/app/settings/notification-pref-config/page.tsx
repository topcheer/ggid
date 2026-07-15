"use client";

import { useNotificationPrefConfig } from "@ggid/sdk-react";
import { Bell } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function NotificationPrefConfigPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useNotificationPrefConfig();
  if (loading) return <div className="p-8 text-gray-400">Loading...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const channels = ["email", "sms", "push", "webhook"];

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">Notification Preferences</h1><p className="text-sm text-gray-400 mt-1">Event-channel delivery matrix</p></div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Save</button>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4 flex items-center gap-2"><Bell className="w-4 h-4 text-blue-400" /> Event × Channel Matrix</h2>
        <table className="w-full text-sm"><thead><tr className="border-b border-gray-800 text-gray-400"><th className="text-left py-2">Event</th>{channels.map((c) => <th key={c} className="text-center py-2 capitalize">{c}</th>)}</tr></thead>
          <tbody>{(data?.matrix ?? []).map((row) => (
            <tr key={row.event} className="border-b border-gray-800"><td className="py-2 text-xs font-medium">{row.event_label}</td>{channels.map((c) => <td key={c} className="text-center py-2"><input type="checkbox" defaultChecked={row.channels.includes(c)} /></td>)}</tr>
          ))}</tbody>
        </table>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="bg-gray-900 rounded-xl p-6"><h2 className="text-sm font-semibold mb-3">Quiet Hours</h2><div className="space-y-2 text-sm"><div><label className="text-xs text-gray-400">Start</label><input type="time" defaultValue={data?.quiet_hours?.start} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg" /></div><div><label className="text-xs text-gray-400">End</label><input type="time" defaultValue={data?.quiet_hours?.end} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg" /></div><div><label className="text-xs text-gray-400">Timezone</label><input type="text" defaultValue={data?.quiet_hours?.timezone} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg" /></div></div></div>
        <div className="bg-gray-900 rounded-xl p-6"><h2 className="text-sm font-semibold mb-3">Digest Frequency</h2><select defaultValue={data?.digest_frequency} className="w-full px-3 py-2 bg-gray-800 rounded-lg text-sm"><option value="realtime">Realtime</option><option value="hourly">Hourly</option><option value="daily">Daily</option><option value="weekly">Weekly</option></select><p className="text-xs text-gray-400 mt-2">Batch non-critical notifications</p></div>
        <div className="bg-gray-900 rounded-xl p-6"><h2 className="text-sm font-semibold mb-3">Emergency Override</h2><label className="flex items-center gap-2 text-sm"><input type="checkbox" defaultChecked={data?.emergency_override} /> Always notify for security alerts</label></div>
      </div>
    </div>
  );
}
