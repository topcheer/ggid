"use client";

import { useState, useEffect, useCallback } from "react";
import { UserX, Calendar, Trash2, Archive, Ban, Play, X, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface InactiveUser {
  user_id: string;
  username: string;
  email: string;
  last_login: string | null;
  days_inactive: number;
  status: string;
}

interface CleanupSchedule {
  inactive_threshold_days: number;
  action: "disable" | "archive" | "delete";
  schedule_date: string;
  exclude_roles: string[];
}

const actionConfig: Record<string, { icon: typeof Ban; label: string; color: string }> = {
  disable: { icon: Ban, label: "Disable", color: "text-yellow-600" },
  archive: { icon: Archive, label: "Archive", color: "text-blue-600" },
  delete: { icon: Trash2, label: "Delete", color: "text-red-600" },
};

export default function InactiveCleanupPage() {
  const t = useTranslations();

  const [users, setUsers] = useState<InactiveUser[]>([]);
  const [loading, setLoading] = useState(false);
  const [threshold, setThreshold] = useState(90);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [action, setAction] = useState<"disable" | "archive" | "delete">("disable");
  const [scheduleDate, setScheduleDate] = useState("");
  const [scheduling, setScheduling] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);

  const fetchUsers = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/identity/inactive-users?threshold=${threshold}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setUsers(data.users || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [threshold]);

  useEffect(() => { fetchUsers(); }, [fetchUsers]);

  const toggleSelect = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const scheduleCleanup = async () => {
    setScheduling(true);
    try {
      await fetch("/api/v1/identity/inactive-cleanup/schedule", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({
          user_ids: [...selectedIds],
          action,
          schedule_date: scheduleDate || undefined,
        }),
      });
      setShowConfirm(false);
      setSelectedIds(new Set());
    } catch { /* noop */ }
    finally { setScheduling(false); }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><UserX className="w-6 h-6 text-gray-500" /> {t("big1.inactiveCleanup.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">{t("big1.inactiveCleanup.identifyInactiveUsersAndScheduleCleanupActions")}</p>
      </div>

      {/* Controls */}
      <div className="flex items-center gap-3 flex-wrap">
        <div className="flex items-center gap-2">
          <label className="text-sm font-medium">{t("big1.inactiveCleanup.inactiveThreshold")}</label>
          <input type="number" value={threshold} onChange={(e) => setThreshold(parseInt(e.target.value) || 90)} min={1} className="w-20 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-800 text-sm" />
          <span className="text-sm text-gray-400">{t("big1.inactiveCleanup.days")}</span>
        </div>
        <button onClick={fetchUsers} disabled={loading} className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50">{loading ? t("big1.inactiveCleanup.loading") : t("big1.inactiveCleanup.refresh")}</button>
        {selectedIds.size > 0 && (
          <button onClick={() => setShowConfirm(true)} className="px-3 py-1.5 rounded-lg bg-orange-600 text-white text-sm font-medium hover:bg-orange-700 flex items-center gap-2"><Play className="w-4 h-4" />{t("big1.inactiveCleanup.scheduleCleanup")}{selectedIds.size})</button>
        )}
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("big1.inactiveCleanup.inactiveUsers")}</span><p className="text-2xl font-bold mt-1">{users.length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("big1.inactiveCleanup.90Days")}</span><p className="text-2xl font-bold mt-1 text-yellow-600">{users.filter((u) => u.days_inactive >= 90).length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("big1.inactiveCleanup.180Days")}</span><p className="text-2xl font-bold mt-1 text-orange-600">{users.filter((u) => u.days_inactive >= 180).length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("big1.inactiveCleanup.neverLoggedIn")}</span><p className="text-2xl font-bold mt-1 text-red-600">{users.filter((u) => !u.last_login).length}</p></div>
      </div>

      {/* Table */}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50">
            <tr>
              <th className="px-4 py-3 text-left font-medium w-8"><input type="checkbox" checked={selectedIds.size === users.length && users.length > 0} onChange={(e) => setSelectedIds(e.target.checked ? new Set(users.map((u) => u.user_id)) : new Set())} className="rounded" /></th>
              <th className="px-4 py-3 text-left font-medium">{t("big1.inactiveCleanup.user")}</th>
              <th className="px-4 py-3 text-left font-medium">{t("big1.inactiveCleanup.lastLogin")}</th>
              <th className="px-4 py-3 text-left font-medium">{t("big1.inactiveCleanup.daysInactive")}</th>
              <th className="px-4 py-3 text-left font-medium">{t("big1.inactiveCleanup.status")}</th>
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {users.map((u) => {
              const ActionIcon = u.days_inactive >= 180 ? actionConfig.delete.icon : u.days_inactive >= 90 ? actionConfig.archive.icon : actionConfig.disable.icon;
              return (
                <tr key={u.user_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                  <td className="px-4 py-3"><input type="checkbox" checked={selectedIds.has(u.user_id)} onChange={() => toggleSelect(u.user_id)} className="rounded" /></td>
                  <td className="px-4 py-3"><span className="font-medium">{u.username}</span><p className="text-xs text-gray-400">{u.email}</p></td>
                  <td className="px-4 py-3 text-gray-500">{u.last_login || "Never"}</td>
                  <td className="px-4 py-3"><span className={`font-bold ${u.days_inactive >= 180 ? "text-red-600" : u.days_inactive >= 90 ? "text-orange-600" : "text-yellow-600"}`}>{u.days_inactive}</span></td>
                  <td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800">{u.status}</span></td>
                </tr>
              );
            })}
            {users.length === 0 && !loading && <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">{t("big1.inactiveCleanup.noInactiveUsersFound")}</td></tr>}
          </tbody>
        </table>
      </div>

      {/* Schedule confirmation modal */}
      {showConfirm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowConfirm(false)}>
          <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-orange-500" />{t("big1.inactiveCleanup.scheduleCleanup")}</h3>
              <button onClick={() => setShowConfirm(false)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 space-y-4 text-sm">
              <div>
                <label className="font-medium">{t("big1.inactiveCleanup.action")}</label>
                <div className="flex gap-2 mt-1">
                  {(["disable", "archive", "delete"] as const).map((a) => {
                    const cfg = actionConfig[a];
                    const Icon = cfg.icon;
                    return (
                      <button key={a} onClick={() => setAction(a)} className={`flex-1 px-3 py-2 rounded-lg border text-sm flex items-center justify-center gap-2 ${action === a ? "border-blue-500 bg-blue-50 dark:bg-blue-900/20" : "dark:border-gray-700"}`}>
                        <Icon className={`w-4 h-4 ${cfg.color}`} /> {cfg.label}
                      </button>
                    );
                  })}
                </div>
              </div>
              <div>
                <label className="font-medium">{t("big1.inactiveCleanup.scheduleDateOptional")}</label>
                <input type="date" value={scheduleDate} onChange={(e) => setScheduleDate(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm" />
                <p className="text-xs text-gray-400 mt-1">{t("big1.inactiveCleanup.leaveEmptyToExecuteImmediately")}</p>
              </div>
              <div className="text-gray-500">
                <p>{selectedIds.size}{t("big1.inactiveCleanup.user")}{selectedIds.size > 1 ? t("big1.inactiveCleanup.s") : ""}{t("big1.inactiveCleanup.willBe")}{action === t("big1.inactiveCleanup.delete") ? t("big1.inactiveCleanup.permanentlyDeleted") : action + t("big1.inactiveCleanup.d")}.</p>
                {action === t("big1.inactiveCleanup.delete") && <p className="text-red-600 mt-1">{t("big1.inactiveCleanup.thisActionCannotBeUndone")}</p>}
              </div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setShowConfirm(false)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">{t("big1.inactiveCleanup.cancel")}</button>
              <button onClick={scheduleCleanup} disabled={scheduling} className="px-4 py-2 rounded-lg bg-orange-600 text-white text-sm font-medium hover:bg-orange-700 disabled:opacity-50">{scheduling ? t("big1.inactiveCleanup.scheduling") : t("big1.inactiveCleanup.scheduleCleanup")}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
