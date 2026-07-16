"use client";
import { useState, useEffect, useCallback } from "react";
import { Bell, Save, Moon } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface PrefData { matrix: Record<string, { email: boolean; sms: boolean; push: boolean; webhook: boolean }>; quiet_hours: { enabled: boolean; start: string; end: string; timezone: string }; digest_frequency: string; per_user_override: { user_id: string; username: string; overrides: Record<string, string[]> }[]; }
const events = ["user.created", "user.deactivated", "role.assigned", "role.revoked", "policy.changed", "security.alert", "access.expired", "audit.anomaly"];
const channels = ["email", "sms", "push", "webhook"] as const;
export default function NotificationPreferencesPage() {
  const t = useTranslations();

  const [data, setData] = useState<PrefData>({ matrix: Object.fromEntries(events.map((e) => [e, { email: true, sms: false, push: false, webhook: false }])), quiet_hours: { enabled: false, start: "22:00", end: "07:00", timezone: "UTC" }, digest_frequency: "daily", per_user_override: [] });
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const loadData = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch("/api/v1/notification/preferences", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); if (d) setData(prev => ({ ...prev, ...d })); } }
    catch (err) { setError(err instanceof Error ? err.message : "An error occurred"); } finally { setLoading(false); }
  }, []);
  useEffect(() => { loadData(); }, [loadData]);
  const save = async () => { setSaving(true); try { await fetch("/api/v1/notification/preferences", { method: "PUT", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(data) }); } catch { /* noop */ } finally { setSaving(false); } };
  const toggle = (event: string, channel: string) => { setData({ ...data, matrix: { ...data.matrix, [event]: { ...data.matrix[event], [channel]: !data.matrix[event][channel as keyof typeof data.matrix[string]] } } }); };
  if (loading) return (<div className="p-8 flex items-center justify-center"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" /></div>);
  if (error) return (<div className="p-8"><div className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-950 dark:border-red-800 p-4"><p className="text-red-700 dark:text-red-400 text-sm font-medium">Error: {error}</p><button onClick={loadData} className="mt-2 px-4 py-1.5 rounded-lg bg-red-600 text-white text-sm hover:bg-red-700">Retry</button></div></div>);
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between"><div><h1 className="text-2xl font-bold flex items-center gap-2"><Bell className="w-6 h-6 text-blue-500" /> {t("notificationPreferences.title")}</h1><p className="text-sm text-gray-500 mt-1">Configure event-channel routing, quiet hours, and digest settings.</p></div><button onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium flex items-center gap-2"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save"}</button></div>
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Event</th>{channels.map((ch) => <th key={ch} className="px-4 py-3 text-center font-medium capitalize">{ch}</th>)}</tr></thead><tbody className="divide-y dark:divide-gray-800">{events.map((event) => (<tr key={event} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs font-medium">{event}</td>{channels.map((ch) => <td key={ch} className="px-4 py-3 text-center"><input aria-label="Toggle" type="checkbox" checked={data.matrix[event][ch]} onChange={() => toggle(event, ch)} className="rounded" /></td>)}</tr>))}</tbody></table></div>
      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3"><div className="flex items-center gap-3"><Moon className="w-5 h-5 text-gray-400" /><h3 className="text-sm font-semibold">Quiet Hours</h3><label className="flex items-center gap-2 text-sm ml-auto"><input type="checkbox" checked={data.quiet_hours.enabled} onChange={(e) => setData({ ...data, quiet_hours: { ...data.quiet_hours, enabled: e.target.checked } })} className="rounded" /> Enabled</label></div>{data.quiet_hours.enabled && <div className="grid grid-cols-3 gap-3"><div><label className="text-xs text-gray-500">Start</label><input type="time" value={data.quiet_hours.start} onChange={(e) => setData({ ...data, quiet_hours: { ...data.quiet_hours, start: e.target.value } })} className="w-full mt-1 px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div><div><label className="text-xs text-gray-500">End</label><input type="time" value={data.quiet_hours.end} onChange={(e) => setData({ ...data, quiet_hours: { ...data.quiet_hours, end: e.target.value } })} className="w-full mt-1 px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div><div><label className="text-xs text-gray-500">Timezone</label><input type="text" value={data.quiet_hours.timezone} onChange={(e) => setData({ ...data, quiet_hours: { ...data.quiet_hours, timezone: e.target.value } })} className="w-full mt-1 px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div></div>}</div>
      <div className="rounded-lg border dark:border-gray-800 p-4"><label className="text-sm font-medium">Digest Frequency</label><select value={data.digest_frequency} onChange={(e) => setData({ ...data, digest_frequency: e.target.value })} className="ml-3 px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="none">None (real-time)</option><option value="hourly">Hourly</option><option value="daily">Daily</option><option value="weekly">Weekly</option></select></div>
      {data.per_user_override.length > 0 && <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">Per-User Overrides</h3><div className="space-y-1">{data.per_user_override.map((o) => (<div key={o.user_id} className="text-xs"><span className="font-medium">{o.username}</span><span className="text-gray-400 ml-2">{JSON.stringify(o.overrides)}</span></div>))}</div></div>}
    </div>
  );
}
