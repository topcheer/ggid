"use client";

import React, { useState } from "react";
import { useApi } from "@/lib/api";
import {
  Clock, Loader2, AlertCircle, X, LogIn, LogOut, KeyRound, Shield, Users2, RefreshCw, Filter,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface TimelineEvent {
  id: string;
  event_type: string;
  description: string;
  actor: string;
  ip_address: string;
  user_agent: string;
  metadata: Record<string, string>;
  created_at: string;
}

const typeIcons: Record<string, React.ReactNode> = {
  login: <LogIn className="h-4 w-4 text-green-500" />,
  logout: <LogOut className="h-4 w-4 text-gray-400" />,
  password_change: <KeyRound className="h-4 w-4 text-orange-500" />,
  mfa_enroll: <Shield className="h-4 w-4 text-blue-500" />,
  mfa_remove: <Shield className="h-4 w-4 text-red-500" />,
  role_change: <Users2 className="h-4 w-4 text-purple-500" />,
};

const filterTypes = ["all", "login", "logout", "password_change", "mfa_enroll", "mfa_remove", "role_change", "permission_grant", "permission_revoke", "session_revoke", "api_key_create"];

export default function UserTimelinePage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [userId, setUserId] = useState("");
  const [events, setEvents] = useState<TimelineEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState("all");

  const handleLoad = async (uid?: string) => {
    const id = uid || userId;
    if (!id.trim()) return;
    setLoading(true); setError(null);
    try {
      const q = filter !== "all" ? `&type=${filter}` : "";
      setEvents(await apiFetch<TimelineEvent[]>(`/api/v1/users/${id}/timeline?${q}`));
    } catch { setError("Failed to load timeline"); }
    finally { setLoading(false); }
  };

  const handleFilterChange = (f: string) => { setFilter(f); if (events.length > 0) handleLoad(); };

  const filtered = filter === "all" ? events : events.filter((e) => e.event_type === filter);
  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Clock className="h-6 w-6 text-indigo-600" /> {t("usersTimeline.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Chronological activity log: logins, password changes, MFA, role changes.</p>
      </div>

      <div className="flex items-center gap-2">
        <input aria-label="User ID or username" value={userId} onChange={(e) => setUserId(e.target.value)} onKeyDown={(e) => e.key === "Enter" && handleLoad()} placeholder="User ID or username" className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" />
        <button onClick={() => handleLoad()} disabled={!userId.trim() || loading} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Filter className="h-4 w-4" />} Load</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Filter tabs */}
      {events.length > 0 && (
        <div className="flex flex-wrap gap-1">{filterTypes.map((f) => <button key={f} onClick={() => handleFilterChange(f)} className={`rounded px-2 py-1 text-xs font-medium ${filter === f ? "bg-indigo-600 text-white" : "bg-gray-100 text-gray-500 dark:bg-gray-800"}`}>{f.replace(/_/g, " ")}</button>)}</div>
      )}

      {/* Timeline */}
      {events.length === 0 && !loading ? (
        <div className={cardCls}><div className="py-12 text-center"><Clock className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">Enter a user ID to view their activity timeline.</p></div></div>
      ) : loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      : (
        <div className="relative">
          <div className="absolute left-5 top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-700" />
          <div className="space-y-4">
            {filtered.map((e) => (
              <div key={e.id} className="relative flex gap-4 pl-2">
                <div className="z-10 flex h-8 w-8 items-center justify-center rounded-full border-2 border-white bg-white dark:border-gray-800 dark:bg-gray-800">{typeIcons[e.event_type] || <Clock className="h-4 w-4 text-gray-400" />}</div>
                <div className="flex-1 rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-gray-700 dark:bg-gray-800">
                  <div className="flex items-center justify-between"><span className="text-sm font-medium text-gray-900 dark:text-white">{e.event_type.replace(/_/g, " ")}</span><span className="text-xs text-gray-400">{new Date(e.created_at).toLocaleString()}</span></div>
                  <p className="mt-1 text-sm text-gray-500">{e.description}</p>
                  <div className="mt-1 flex flex-wrap items-center gap-3 text-xs text-gray-400"><span>Actor: {e.actor.slice(0, 12)}</span>{e.ip_address && <span className="font-mono">IP: {e.ip_address}</span>}{e.user_agent && <span className="truncate max-w-[200px]">{e.user_agent}</span>}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
