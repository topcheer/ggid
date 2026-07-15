"use client";

import { useState, useEffect, useCallback } from "react";
import { Users, GitBranch, Clock, ShieldCheck } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface GroupInfo {
  id: string;
  name: string;
  member_count: number;
  sub_groups: number;
  parent_groups: number;
  nested_depth: number;
  membership_trend_30d: { day: string; count: number }[];
  inactive_members: { user_id: string; username: string; last_active: string }[];
  role_assignments: number;
  access_review_status: "current" | "overdue" | "scheduled";
}

const reviewColors: Record<string, string> = {
  current: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  overdue: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  scheduled: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
};

export default function GroupAnalyticsPage() {
  const t = useTranslations();

  const [groups, setGroups] = useState<GroupInfo[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try { const res = await fetch("/api/v1/org/group-analytics", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); setGroups(d.groups || d || []); } }
    catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const maxTrend = Math.max(...(groups.flatMap((g) => g.membership_trend_30d.map((t) => t.count)) || [1]), 1);

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Users className="w-6 h-6 text-purple-500" /> {t("groupAnalytics.title")}</h1><p className="text-sm text-gray-500 mt-1">Group membership analytics with trends and review status.</p></div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {groups.map((g) => (
          <div key={g.id} className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
            <div className="flex items-center justify-between"><div><span className="font-semibold">{g.name}</span><p className="text-xs text-gray-400 font-mono">{g.id}</p></div><span className={"px-2 py-1 rounded text-xs " + reviewColors[g.access_review_status]}>{g.access_review_status}</span></div>
            <div className="grid grid-cols-2 gap-2 text-sm">
              <div className="flex items-center gap-1"><Users className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">Members</span><span className="font-bold ml-auto">{g.member_count}</span></div>
              <div className="flex items-center gap-1"><GitBranch className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">Depth</span><span className="font-bold ml-auto">{g.nested_depth}</span></div>
              <div className="flex items-center gap-1"><span className="text-gray-500">Sub-groups</span><span className="font-bold ml-auto">{g.sub_groups}</span></div>
              <div className="flex items-center gap-1"><ShieldCheck className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">Roles</span><span className="font-bold ml-auto">{g.role_assignments}</span></div>
            </div>
            <div><span className="text-xs text-gray-500">Trend (30d)</span><div className="flex items-end gap-0.5 h-8 mt-1">{g.membership_trend_30d.map((t, i) => <div key={i} className="flex-1 bg-purple-400 dark:bg-purple-500 rounded-t" style={{ height: (t.count / maxTrend) * 100 + "%", minHeight: "1px" }} />)}</div></div>
            {g.inactive_members.length > 0 && <div className="border-t dark:border-gray-800 pt-2"><span className="text-xs text-orange-600 flex items-center gap-1"><Clock className="w-3 h-3" /> {g.inactive_members.length} inactive members</span><div className="mt-1 space-y-0.5">{g.inactive_members.slice(0, 3).map((m) => <div key={m.user_id} className="text-xs text-gray-400">{m.username} - last active {m.last_active}</div>)}{g.inactive_members.length > 3 && <span className="text-xs text-gray-400">+{g.inactive_members.length - 3} more</span>}</div></div>}
          </div>
        ))}
        {groups.length === 0 && !loading && <div className="col-span-full text-center text-gray-500 py-8">No groups found.</div>}
      </div>
    </div>
  );
}
