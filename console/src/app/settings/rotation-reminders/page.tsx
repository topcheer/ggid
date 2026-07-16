"use client";

import { useState, useEffect, useCallback } from "react";
import { Clock, Send, AlertTriangle, KeyRound, Shield } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface RotationItem {
  id: string;
  credential_type: string;
  user_id: string;
  username: string;
  last_rotated: string;
  rotation_period_days: number;
  days_overdue: number;
  severity: "low" | "medium" | "high" | "critical";
}

const sevColors: Record<string, string> = {
  low: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function RotationRemindersPage() {
  const t = useTranslations();

  const [items, setItems] = useState<RotationItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [sendingId, setSendingId] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/rotation-reminders", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); setItems(data.items || data || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const sendReminder = async (id: string) => {
    setSendingId(id);
    try { await fetch(`/api/v1/auth/rotation-reminders/${id}/send`, { method: "POST", headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); }
    catch { /* noop */ }
    finally { setSendingId(null); }
  };

  const critical = items.filter((i) => i.severity === "critical").length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Clock className="w-6 h-6 text-orange-500" /> {t("rotationReminders.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Track overdue credential rotations and send reminders.</p>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Overdue</span><p className="text-2xl font-bold mt-1">{items.length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Critical</span><p className="text-2xl font-bold mt-1 text-red-600">{critical}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Unique Users</span><p className="text-2xl font-bold mt-1">{new Set(items.map((i) => i.user_id)).size}</p></div>
      </div>

      {critical > 0 && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-4 flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-red-500" /><span className="font-semibold text-red-700 dark:text-red-400">{critical} credentials critically overdue for rotation</span></div>}

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Type</th><th className="px-4 py-3 text-left font-medium">User</th><th className="px-4 py-3 text-left font-medium">Last Rotated</th><th className="px-4 py-3 text-left font-medium">Period</th><th className="px-4 py-3 text-left font-medium">Overdue</th><th className="px-4 py-3 text-left font-medium">Severity</th><th className="px-4 py-3 text-left font-medium">Action</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">
            {items.map((item) => (
              <tr key={item.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3"><span className="flex items-center gap-1 text-xs font-medium"><KeyRound className="w-3 h-3 text-gray-400" />{item.credential_type}</span></td>
                <td className="px-4 py-3 font-medium">{item.username}</td>
                <td className="px-4 py-3 text-gray-500">{item.last_rotated}</td>
                <td className="px-4 py-3 text-xs text-gray-500">{item.rotation_period_days}d</td>
                <td className="px-4 py-3"><span className="font-bold text-orange-600">{item.days_overdue}d</span></td>
                <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${sevColors[item.severity]}`}>{item.severity}</span></td>
                <td className="px-4 py-3"><button onClick={() => sendReminder(item.id)} disabled={sendingId === item.id} className="text-xs font-medium text-blue-600 hover:underline disabled:opacity-50 flex items-center gap-1"><Send className="w-3 h-3" /> {sendingId === item.id ? "Sending..." : "Remind"}</button></td>
              </tr>
            ))}
            {items.length === 0 && !loading && <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-500">No overdue rotations.</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}
