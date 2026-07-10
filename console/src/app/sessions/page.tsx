"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import { Monitor, Smartphone, Globe, Trash2, RefreshCw, Clock } from "lucide-react";

interface Session {
  id: string;
  user_id: string;
  user_email?: string;
  ip_address: string;
  user_agent: string;
  created_at: string;
  last_active_at: string;
  expires_at?: string;
  device_type?: string;
  location?: string;
}

export default function SessionsPage() {
  const { apiFetch } = useApi();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);

  const loadSessions = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ sessions?: Session[] }>("/api/v1/auth/sessions").catch(() => null);
      setSessions(data?.sessions || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load sessions");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { loadSessions(); }, [loadSessions]);

  const handleRevoke = async (sessionId: string) => {
    try {
      await apiFetch(`/api/v1/auth/sessions/${sessionId}`, { method: "DELETE" });
      setSessions(sessions.filter((s) => s.id !== sessionId));
      setMsg("Session revoked");
      setTimeout(() => setMsg(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to revoke session");
    }
  };

  const handleRevokeAll = async () => {
    try {
      await apiFetch("/api/v1/auth/sessions", { method: "DELETE" });
      setSessions([]);
      setMsg("All sessions revoked");
      setTimeout(() => setMsg(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to revoke all sessions");
    }
  };

  const getDeviceIcon = (ua: string) => {
    if (/mobile|android|iphone|ipad/i.test(ua)) return <Smartphone className="h-4 w-4" />;
    return <Monitor className="h-4 w-4" />;
  };

  const parseDevice = (ua: string): string => {
    if (/mobile|android|iphone/i.test(ua)) return "Mobile";
    if (/ipad|tablet/i.test(ua)) return "Tablet";
    return "Desktop";
  };

  const parseBrowser = (ua: string): string => {
    if (/chrome/i.test(ua)) return "Chrome";
    if (/firefox/i.test(ua)) return "Firefox";
    if (/safari/i.test(ua)) return "Safari";
    if (/edge/i.test(ua)) return "Edge";
    return "Unknown";
  };

  const formatTime = (ts: string) => {
    if (!ts) return "-";
    const diff = Date.now() - new Date(ts).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return "just now";
    if (mins < 60) return `${mins}m ago`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}h ago`;
    return new Date(ts).toLocaleDateString();
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Active Sessions</h1>
          <p className="text-sm text-gray-500">Manage active user sessions and devices</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={loadSessions}
            className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50"
          >
            <RefreshCw className="h-4 w-4" /> Refresh
          </button>
          {sessions.length > 0 && (
            <button
              onClick={handleRevokeAll}
              className="flex items-center gap-2 rounded-lg border border-red-300 px-3 py-2 text-sm text-red-600 hover:bg-red-50"
            >
              <Trash2 className="h-4 w-4" /> Revoke All
            </button>
          )}
        </div>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</div>
      )}
      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">{msg}</div>
      )}

      {loading ? (
        <p className="text-gray-500">Loading...</p>
      ) : sessions.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm">
          <Monitor className="mx-auto mb-4 h-12 w-12 text-gray-300" />
          <p className="text-gray-500">No active sessions</p>
          <p className="mt-1 text-xs text-gray-400">Sessions will appear here when users log in.</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm">
          <table className="w-full">
            <thead className="border-b border-gray-200 bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Device</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">IP Address</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Location</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Last Active</th>
                <th className="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500">Created</th>
                <th className="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {sessions.map((session) => (
                <tr key={session.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-3">
                      <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gray-100 text-gray-500">
                        {getDeviceIcon(session.user_agent || "")}
                      </div>
                      <div>
                        <p className="text-sm font-medium">{parseDevice(session.user_agent || "")}</p>
                        <p className="text-xs text-gray-500">{parseBrowser(session.user_agent || "")}</p>
                      </div>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-sm font-mono text-gray-600">
                    {session.ip_address || "-"}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">
                    {session.location || "Unknown"}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">
                    <span className="flex items-center gap-1">
                      <Clock className="h-3.5 w-3.5 text-gray-400" />
                      {formatTime(session.last_active_at || session.created_at)}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500">
                    {formatTime(session.created_at)}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <button
                      onClick={() => handleRevoke(session.id)}
                      className="text-red-500 hover:text-red-700"
                      title="Revoke session"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
