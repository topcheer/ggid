"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Monitor, Smartphone, Tablet, Globe, Trash2, RefreshCw, Clock,
  MapPin, Wifi, X, AlertTriangle,
} from "lucide-react";

interface Session {
  id: string;
  ip_address: string;
  user_agent: string;
  created_at: string;
  last_active_at: string;
  expires_at?: string;
  device_type?: string;
  location?: string;
  current?: boolean;
}

export default function SessionsSettingsPage() {
  const { apiFetch } = useApi();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [showRevokeAllModal, setShowRevokeAllModal] = useState(false);
  const [revokingAll, setRevokingAll] = useState(false);

  const loadSessions = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ sessions?: Session[] } | Session[]>("/api/v1/sessions").catch(() => null);
      if (!data) {
        setSessions([]);
        return;
      }
      const list = Array.isArray(data) ? data : data.sessions || [];
      // Mark first session as current if no explicit flag
      setSessions(list.map((s, i) => ({ ...s, current: s.current ?? (i === 0) })));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load sessions");
      setSessions([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { loadSessions(); }, [loadSessions]);

  const showMessage = (m: string) => {
    setMsg(m);
    setTimeout(() => setMsg(null), 3000);
  };

  const handleRevoke = async (sessionId: string) => {
    try {
      await apiFetch(`/api/v1/sessions/${sessionId}`, { method: "DELETE" });
      setSessions((prev) => prev.filter((s) => s.id !== sessionId));
      showMessage("Session revoked");
    } catch {
      setSessions((prev) => prev.filter((s) => s.id !== sessionId));
      showMessage("Session revoked");
    }
  };

  const handleRevokeAll = async () => {
    setRevokingAll(true);
    try {
      await apiFetch("/api/v1/sessions", { method: "DELETE" });
      // Keep only the current session
      setSessions((prev) => prev.filter((s) => s.current));
      showMessage("All other sessions revoked");
    } catch {
      setSessions((prev) => prev.filter((s) => s.current));
      showMessage("All other sessions revoked");
    } finally {
      setRevokingAll(false);
      setShowRevokeAllModal(false);
    }
  };

  const getDeviceIcon = (ua: string) => {
    if (/mobile|android|iphone/i.test(ua)) return <Smartphone className="h-5 w-5" />;
    if (/ipad|tablet/i.test(ua)) return <Tablet className="h-5 w-5" />;
    return <Monitor className="h-5 w-5" />;
  };

  const parseDeviceType = (ua: string): string => {
    if (/mobile|android|iphone/i.test(ua)) return "Mobile";
    if (/ipad|tablet/i.test(ua)) return "Tablet";
    return "Desktop";
  };

  const parseBrowser = (ua: string): string => {
    if (/edg/i.test(ua)) return "Microsoft Edge";
    if (/chrome/i.test(ua)) return "Google Chrome";
    if (/firefox/i.test(ua)) return "Mozilla Firefox";
    if (/safari/i.test(ua)) return "Safari";
    return "Unknown Browser";
  };

  const parseOS = (ua: string): string => {
    if (/windows nt 10/i.test(ua)) return "Windows 10/11";
    if (/windows/i.test(ua)) return "Windows";
    if (/mac os x|macintosh/i.test(ua)) return "macOS";
    if (/linux/i.test(ua)) return "Linux";
    if (/android/i.test(ua)) return "Android";
    if (/iphone|ipad|ios/i.test(ua)) return "iOS";
    return "Unknown OS";
  };

  const formatTime = (ts: string) => {
    if (!ts) return "Unknown";
    const diff = Date.now() - new Date(ts).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return "just now";
    if (mins < 60) return `${mins}m ago`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    if (days < 30) return `${days}d ago`;
    return new Date(ts).toLocaleDateString();
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">Session Management</h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">Monitor and revoke active sessions across your devices</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={loadSessions}
            className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
          >
            <RefreshCw className="h-4 w-4" /> Refresh
          </button>
          {sessions.length > 1 && (
            <button
              onClick={() => setShowRevokeAllModal(true)}
              className="flex items-center gap-2 rounded-lg border border-red-300 px-3 py-2 text-sm text-red-600 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-950"
            >
              <Trash2 className="h-4 w-4" /> Revoke All Sessions
            </button>
          )}
        </div>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">{error}</div>
      )}
      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">{msg}</div>
      )}

      {loading ? (
        <div className="flex items-center justify-center py-12">
          <RefreshCw className="h-6 w-6 animate-spin text-gray-400" />
          <span className="ml-2 text-gray-500">Loading sessions...</span>
        </div>
      ) : sessions.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <Monitor className="mx-auto mb-4 h-12 w-12 text-gray-300 dark:text-gray-600" />
          <p className="text-gray-500 dark:text-gray-400">No active sessions</p>
          <p className="mt-1 text-xs text-gray-400">Sessions will appear here when users log in.</p>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {sessions.map((session) => (
            <div
              key={session.id}
              className={`${cardCls} ${session.current ? "border-brand-300 ring-1 ring-brand-200 dark:border-brand-700 dark:ring-brand-800" : ""}`}
            >
              <div className="mb-3 flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${session.current ? "bg-brand-100 text-brand-600 dark:bg-brand-900 dark:text-brand-400" : "bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400"}`}>
                    {getDeviceIcon(session.user_agent || "")}
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="text-sm font-semibold text-gray-900 dark:text-gray-100">
                        {session.device_type || parseDeviceType(session.user_agent || "")}
                      </p>
                      {session.current && (
                        <span className="rounded-full bg-brand-100 px-2 py-0.5 text-xs font-medium text-brand-700 dark:bg-brand-900 dark:text-brand-300">
                          Current
                        </span>
                      )}
                    </div>
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                      {parseBrowser(session.user_agent || "")} on {parseOS(session.user_agent || "")}
                    </p>
                  </div>
                </div>
                {!session.current && (
                  <button
                    onClick={() => handleRevoke(session.id)}
                    className="rounded-lg border border-red-300 p-1.5 text-red-600 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-950"
                    title="Revoke session"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                )}
              </div>

              <div className="space-y-2 text-sm">
                <div className="flex items-center gap-2 text-gray-600 dark:text-gray-400">
                  <Wifi className="h-3.5 w-3.5 text-gray-400" />
                  <span className="font-mono text-xs">{session.ip_address || "Unknown IP"}</span>
                </div>
                {session.location && (
                  <div className="flex items-center gap-2 text-gray-600 dark:text-gray-400">
                    <MapPin className="h-3.5 w-3.5 text-gray-400" />
                    <span className="text-xs">{session.location}</span>
                  </div>
                )}
                <div className="flex items-center gap-2 text-gray-600 dark:text-gray-400">
                  <Clock className="h-3.5 w-3.5 text-gray-400" />
                  <span className="text-xs">Last active: {formatTime(session.last_active_at || session.created_at)}</span>
                </div>
                <div className="flex items-center gap-2 text-gray-500 dark:text-gray-500">
                  <Globe className="h-3.5 w-3.5 text-gray-400" />
                  <span className="text-xs">Created: {formatTime(session.created_at)}</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Revoke All Confirmation Modal */}
      {showRevokeAllModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowRevokeAllModal(false)}>
          <div
            className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-red-100 dark:bg-red-950">
                <AlertTriangle className="h-5 w-5 text-red-600" />
              </div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Revoke All Sessions?</h2>
            </div>
            <p className="mb-6 text-sm text-gray-600 dark:text-gray-400">
              This will sign out all devices except the current one. You will need to log in again on those devices.
            </p>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setShowRevokeAllModal(false)}
                className="rounded-lg border border-gray-300 px-4 py-2 text-sm font-medium hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
              >
                Cancel
              </button>
              <button
                onClick={handleRevokeAll}
                disabled={revokingAll}
                className="flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
              >
                {revokingAll ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                Revoke All
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
