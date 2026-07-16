"use client";
import { useState, useEffect, useCallback } from "react";
import { RefreshCw, Save, Clock } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface RotationEntry { client_id: string; client_name: string; interval_days: number; max_age_hours: number; notify_before_hours: number; auto_rotate: boolean; last_rotated: string; }
interface PolicyData { clients: RotationEntry[]; upcoming: { client_name: string; scheduled_at: string; overdue: boolean }[]; compliance_pct: number; }
export default function TokenRotationPolicyPage() {
  const t = useTranslations();

  const [data, setData] = useState<PolicyData | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const fetchData = useCallback(async () => { setLoading(true); try { const res = await fetch("/api/v1/oauth/token-rotation-policy", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); } catch { /* noop */ } finally { setLoading(false); } }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  const updateClient = (id: string, field: string, value: string | number | boolean) => { if (!data) return; setData({ ...data, clients: data.clients.map((c) => c.client_id === id ? { ...c, [field]: value } : c) }); };
  const save = async () => { if (!data) return; setSaving(true); try { await fetch("/api/v1/oauth/token-rotation-policy", { method: "PUT", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(data.clients) }); } catch { /* noop */ } finally { setSaving(false); } };
  const gaugeColor = data ? (data.compliance_pct >= 90 ? "#10b981" : data.compliance_pct >= 70 ? "#f59e0b" : "#ef4444") : "#3b82f6";
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between"><div><h1 className="text-2xl font-bold flex items-center gap-2"><RefreshCw className="w-6 h-6 text-blue-500" /> {t("tokenRotationPolicy.title")}</h1><p className="text-sm text-gray-500 mt-1">Configure per-client token rotation policies.</p></div><button onClick={save} disabled={saving || !data} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium flex items-center gap-2"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save"}</button></div>
      {data && (<>
        <div className="flex items-center gap-4"><div className="relative w-20 h-20"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={gaugeColor} strokeWidth={6} strokeDasharray={(data.compliance_pct / 100) * 176 + " 176"} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex items-center justify-center"><span className="text-lg font-bold" style={{ color: gaugeColor }}>{data.compliance_pct}%</span></div></div><span className="text-sm text-gray-500">Compliance</span></div>
        <div className="overflow-x-auto rounded-lg border dark:border-gray-800"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Client</th><th className="px-4 py-3 text-left font-medium">Interval (days)</th><th className="px-4 py-3 text-left font-medium">Max Age (h)</th><th className="px-4 py-3 text-left font-medium">Notify Before (h)</th><th className="px-4 py-3 text-left font-medium">Auto</th><th className="px-4 py-3 text-left font-medium">Last Rotated</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{data.clients.map((c) => (<tr key={c.client_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-medium">{c.client_name}</td><td className="px-4 py-3"><input aria-label="Toggle" type="number" value={c.interval_days} onChange={(e) => updateClient(c.client_id, "interval_days", parseInt(e.target.value) || 0)} className="w-16 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs" /></td><td className="px-4 py-3"><input type="number" value={c.max_age_hours} onChange={(e) => updateClient(c.client_id, "max_age_hours", parseInt(e.target.value) || 0)} className="w-16 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs" /></td><td className="px-4 py-3"><input type="number" value={c.notify_before_hours} onChange={(e) => updateClient(c.client_id, "notify_before_hours", parseInt(e.target.value) || 0)} className="w-16 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs" /></td><td className="px-4 py-3"><input type="checkbox" checked={c.auto_rotate} onChange={(e) => updateClient(c.client_id, "auto_rotate", e.target.checked)} className="rounded" /></td><td className="px-4 py-3 text-xs text-gray-500">{c.last_rotated}</td></tr>))}</tbody></table></div>
        {data.upcoming.length > 0 && <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">Upcoming Rotations</h3><div className="space-y-1">{data.upcoming.map((u, i) => (<div key={i} className="flex items-center gap-2 text-sm"><Clock className={"w-3.5 h-3.5 " + (u.overdue ? "text-red-500" : "text-gray-400")} /><span className="font-medium">{u.client_name}</span><span className="text-xs text-gray-400">{u.scheduled_at}</span>{u.overdue && <span className="px-2 py-0.5 rounded text-xs bg-red-100 dark:bg-red-900/30 dark:text-red-400">Overdue</span>}</div>))}</div></div>}
      </>)}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
