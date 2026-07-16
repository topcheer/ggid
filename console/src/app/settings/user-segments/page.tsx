"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import {
  Users, Loader2, AlertCircle, X, Download, Layers,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface UserSegment {
  id: string;
  name: string;
  criteria: { type: "role" | "activity" | "risk" | "department"; value: string }[];
  user_count: number;
  users: { user_id: string; username: string; email: string }[];
}

export default function UserSegmentsPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [segments, setSegments] = useState<UserSegment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);

  useState(() => {
    (async () => {
      try { setSegments(await apiFetch<UserSegment[]>("/api/v1/users/segments").catch(() => [])); }
      catch { setError("Failed to load segments"); }
      finally { setLoading(false); }
    })();
  });

  const handleExport = (seg: UserSegment) => {
    const rows = [["user_id", "username", "email"], ...seg.users.map((u) => [u.user_id, u.username, u.email])];
    const csv = rows.map((r) => r.map((c) => `"${c}"`).join(",")).join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a"); a.href = url; a.download = `segment-${seg.name}.csv`; a.click(); URL.revokeObjectURL(url);
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const criteriaColors: Record<string, string> = { role: "bg-purple-100 text-purple-600 dark:bg-purple-900/30", activity: "bg-blue-100 text-blue-600 dark:bg-blue-900/30", risk: "bg-red-100 text-red-600 dark:bg-red-900/30", department: "bg-cyan-100 text-cyan-600 dark:bg-cyan-900/30" };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Layers className="h-6 w-6 text-cyan-600" /> {t("userSegments.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Segment users by role, activity, risk, or department. Export to CSV.</p>
      </div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-cyan-600" /></div>
      : segments.length === 0 ? (
        <div className={cardCls}><div className="py-12 text-center"><Layers className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No segments defined.</p></div></div>
      ) : (
        <div className="space-y-3">
          {segments.map((seg) => (
            <div key={seg.id} className={cardCls}>
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-2"><span className="font-semibold text-gray-900 dark:text-white">{seg.name}</span><span className="rounded bg-indigo-100 px-1.5 py-0.5 text-xs text-indigo-600 dark:bg-indigo-900/30">{seg.user_count} users</span></div>
                  <div className="mt-2 flex flex-wrap gap-1">{seg.criteria.map((c, i) => <span key={i} className={`rounded px-1.5 py-0.5 text-xs ${criteriaColors[c.type] || "bg-gray-100 text-gray-500"}`}>{c.type}: {c.value}</span>)}</div>
                </div>
                <div className="flex gap-2">
                  <button onClick={() => setExpanded(expanded === seg.id ? null : seg.id)} className="text-xs text-indigo-600 hover:underline">{expanded === seg.id ? "Hide" : "Users"}</button>
                  <button onClick={() => handleExport(seg)} className="flex items-center gap-1 text-xs text-gray-500 hover:text-cyan-600"><Download className="h-3 w-3" /> CSV</button>
                </div>
              </div>
              {expanded === seg.id && seg.users.length > 0 && (
                <div className="mt-3 overflow-x-auto rounded-lg border border-gray-200 dark:border-gray-700">
                  <table className="w-full text-sm">
                    <thead className="bg-gray-50 dark:bg-gray-800"><tr><th className="px-3 py-2 text-left text-xs font-semibold text-gray-500">User</th><th className="px-3 py-2 text-left text-xs font-semibold text-gray-500">Email</th></tr></thead>
                    <tbody className="divide-y divide-gray-200 dark:divide-gray-700">{seg.users.slice(0, 20).map((u) => (<tr key={u.user_id} className="bg-white dark:bg-gray-900"><td className="px-3 py-2 font-medium text-gray-900 dark:text-white">{u.username}</td><td className="px-3 py-2 text-gray-400">{u.email}</td></tr>))}{seg.users.length > 20 && <tr><td colSpan={2} className="px-3 py-2 text-center text-xs text-gray-400">+{seg.users.length - 20} more</td></tr>}</tbody>
                  </table>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
