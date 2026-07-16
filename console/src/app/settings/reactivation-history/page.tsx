"use client";

import { useState, useEffect, useCallback } from "react";
import { History, UserX, UserCheck, Clock } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface TimelineEvent {
  id: string;
  action: "deactivated" | "reactivated";
  timestamp: string;
  reason: string;
  actor: string;
  duration_days: number | null;
}

interface User {
  user_id: string;
  username: string;
}

export default function ReactivationHistoryPage() {
  const t = useTranslations();

  const [users, setUsers] = useState<User[]>([]);
  const [selectedId, setSelectedId] = useState("");
  const [timeline, setTimeline] = useState<TimelineEvent[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchUsers = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/identity/users", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); setUsers(data.users || data || []); }
    } catch { /* noop */ }
  }, []);

  const fetchTimeline = useCallback(async (id: string) => {
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/identity/reactivation-history?user_id=${encodeURIComponent(id)}`, { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); setTimeline(data.events || data || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchUsers(); }, [fetchUsers]);
  useEffect(() => { if (selectedId) fetchTimeline(selectedId); }, [selectedId, fetchTimeline]);

  const totalInactive = timeline.filter((t) => t.action === "deactivated").length;
  const totalReactivated = timeline.filter((t) => t.action === "reactivated").length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><History className="w-6 h-6 text-indigo-500" /> {t("reactivationHistory.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Track user deactivation/reactivation events with reasons and actors.</p>
      </div>

      <select aria-label="Selected id" value={selectedId} onChange={(e) => setSelectedId(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
        <option value="">Select User</option>
        {users.map((u) => <option key={u.user_id} value={u.user_id}>{u.username}</option>)}
      </select>

      {selectedId && timeline.length > 0 && (
        <div className="grid grid-cols-3 gap-4">
          <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><UserX className="w-8 h-8 text-red-500" /><div><span className="text-sm text-gray-500">Deactivations</span><p className="text-xl font-bold text-red-600">{totalInactive}</p></div></div>
          <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><UserCheck className="w-8 h-8 text-green-500" /><div><span className="text-sm text-gray-500">Reactivations</span><p className="text-xl font-bold text-green-600">{totalReactivated}</p></div></div>
          <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Clock className="w-8 h-8 text-blue-500" /><div><span className="text-sm text-gray-500">Total Events</span><p className="text-xl font-bold">{timeline.length}</p></div></div>
        </div>
      )}

      {selectedId && (
        <div className="relative pl-8">
          <div className="absolute left-3 top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-800" />
          <div className="space-y-4">
            {timeline.map((evt) => (
              <div key={evt.id} className="relative">
                <div className={`absolute -left-5 w-4 h-4 rounded-full border-2 ${evt.action === "deactivated" ? "bg-red-500 border-red-200" : "bg-green-500 border-green-200"}`} />
                <div className="rounded-lg border dark:border-gray-800 p-3 ml-2">
                  <div className="flex items-center justify-between">
                    <span className="flex items-center gap-2 text-sm font-medium">{evt.action === "deactivated" ? <UserX className="w-4 h-4 text-red-500" /> : <UserCheck className="w-4 h-4 text-green-500" />}{evt.action === "deactivated" ? "Deactivated" : "Reactivated"}</span>
                    <span className="text-xs text-gray-400">{evt.timestamp}</span>
                  </div>
                  <div className="mt-2 text-sm space-y-1">
                    <div><span className="text-gray-500">Reason: </span><span>{evt.reason}</span></div>
                    <div><span className="text-gray-500">Actor: </span><span className="font-mono text-xs">{evt.actor}</span></div>
                    {evt.duration_days !== null && <div><span className="text-gray-500">Duration: </span><span className="font-bold">{evt.duration_days} days</span></div>}
                  </div>
                </div>
              </div>
            ))}
            {timeline.length === 0 && !loading && <p className="text-sm text-gray-500 py-4">No reactivation history.</p>}
          </div>
        </div>
      )}
      {!selectedId && <p className="text-sm text-gray-500 text-center py-8">Select a user to view history.</p>}
    </div>
  );
}
