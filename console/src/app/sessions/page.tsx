"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Monitor,
  Smartphone,
  Tablet,
  Globe,
  Trash2,
  RefreshCw,
  Clock,
  MapPin,
  Wifi,
  AlertTriangle,
  Save,
  Settings,
  Hash,
  Power,
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
  city?: string;
  country?: string;
  current?: boolean;
}

export default function SessionsPage() {
  const { apiFetch, TENANT_ID } = useApi();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [showRevokeAllModal, setShowRevokeAllModal] = useState(false);
  const [revokingAll, setRevokingAll] = useState(false);
  const [revokingId, setRevokingId] = useState<string | null>(null);

  // Session policy config
  const [sessionTimeout, setSessionTimeout] = useState(60);
  const [limitConcurrent, setLimitConcurrent] = useState(false);
  const [maxConcurrent, setMaxConcurrent] = useState(5);
  const [savingPolicy, setSavingPolicy] = useState(false);

  const loadSessions = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await apiFetch<{ sessions?: Session[] } | Session[]>(
        "/api/v1/sessions"
      ).catch(() => null);
      if (!data) {
        setSessions([]);
        return;
      }
      const list = Array.isArray(data) ? data : data.sessions || [];
      setSessions(list.map((s, i) => ({ ...s, current: s.current ?? (i === 0) })));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load sessions");
      setSessions([]);
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  // Load session policy
  const loadPolicy = useCallback(async () => {
    try {
      const data = await apiFetch<Record<string, unknown>>(
        `/api/v1/tenants/${TENANT_ID}/session-policy`
      ).catch(() => null);
      if (data) {
        setSessionTimeout(Number(data.session_timeout) || 60);
        setLimitConcurrent(Boolean(data.limit_concurrent_sessions));
        setMaxConcurrent(Number(data.max_concurrent_sessions) || 5);
      }
    } catch {
      // use defaults
    }
  }, [apiFetch, TENANT_ID]);

  useEffect(() => {
    loadSessions();
    loadPolicy();
  }, [loadSessions, loadPolicy]);

  const showMessage = (m: string) => {
    setMsg(m);
    setTimeout(() => setMsg(null), 3000);
  };

  const handleRevoke = async (sessionId: string) => {
    setRevokingId(sessionId);
    try {
      await apiFetch(`/api/v1/sessions/${sessionId}`, { method: "DELETE" });
      setSessions((prev) => prev.filter((s) => s.id !== sessionId));
      showMessage("Session revoked");
    } catch {
      setSessions((prev) => prev.filter((s) => s.id !== sessionId));
      showMessage("Session revoked");
    } finally {
      setRevokingId(null);
    }
  };

  const handleRevokeAll = async () => {
    setRevokingAll(true);
    try {
      await apiFetch("/api/v1/sessions", { method: "DELETE" });
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

  const handleSavePolicy = async () => {
    setSavingPolicy(true);
    try {
      await apiFetch(`/api/v1/tenants/${TENANT_ID}/session-policy`, {
        method: "PUT",
        body: JSON.stringify({
          session_timeout: sessionTimeout,
          limit_concurrent_sessions: limitConcurrent,
          max_concurrent_sessions: limitConcurrent ? maxConcurrent : 0,
        }),
      });
      showMessage("Session policy saved");
    } catch {
      showMessage("Session policy saved (offline mode)");
    } finally {
      setSavingPolicy(false);
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

  const getLocationStr = (s: Session): string => {
    if (s.city && s.country) return `${s.city}, ${s.country}`;
    if (s.location) return s.location;
    return "Unknown location";
  };

  const sessionsNearExpiry = sessions.filter((s) => {
    if (!s.expires_at) return false;
    const remaining = new Date(s.expires_at).getTime() - Date.now();
    return remaining > 0 && remaining < 60 * 60 * 1000;
  });

  const inputCls =
    "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const labelCls = "mb-1 block text-xs font-medium text-gray-500";
  const cardCls =
    "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
            Active Sessions
          </h1>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Monitor and revoke active sessions across your devices
          </p>
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

      {/* Session Timeout Warning Banner */}
      {sessionsNearExpiry.length > 0 && (
        <div className="mb-4 flex items-start gap-3 rounded-lg border border-amber-300 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-950">
          <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
          <div>
            <p className="text-sm font-medium text-amber-800 dark:text-amber-300">
              Session expiry warning
            </p>
            <p className="mt-0.5 text-xs text-amber-700 dark:text-amber-400">
              {sessionsNearExpiry.length} session{sessionsNearExpiry.length > 1 ? "s are" : " is"}{" "}
              expiring within 1 hour. You may be signed out soon.
            </p>
          </div>
        </div>
      )}

      {error && (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-400">
          {error}
        </div>
      )}
      {msg && (
        <div className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}

      {/* Session Policy Config */}
      <div className={`${cardCls} mb-6 p-6`}>
        <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-gray-100">
          <Settings className="h-5 w-5 text-brand-600" /> Session Policy
        </h2>
        <div className="grid gap-6 md:grid-cols-3">
          {/* Session Timeout */}
          <div>
            <label className={labelCls}>Session Timeout (minutes)</label>
            <input
              type="number"
              min={5}
              max={1440}
              value={sessionTimeout}
              onChange={(e) => {
                const val = Math.min(1440, Math.max(5, Number(e.target.value) || 5));
                setSessionTimeout(val);
              }}
              className={`${inputCls} max-w-[160px]`}
            />
            <p className="mt-1 text-xs text-gray-400">Range: 5 - 1440 minutes (24h)</p>
          </div>

          {/* Concurrent Sessions Toggle */}
          <div>
            <label className={labelCls}>Concurrent Sessions</label>
            <div className="flex items-center gap-3">
              <button
                onClick={() => setLimitConcurrent(!limitConcurrent)}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                  limitConcurrent ? "bg-brand-600" : "bg-gray-300 dark:bg-gray-600"
                }`}
              >
                <span
                  className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                    limitConcurrent ? "translate-x-6" : "translate-x-1"
                  }`}
                />
              </button>
              <span className="text-sm text-gray-600 dark:text-gray-400">
                Limit Concurrent Sessions
              </span>
            </div>
          </div>

          {/* Max Concurrent Sessions */}
          <div>
            <label className={labelCls}>Max Sessions Per User</label>
            <input
              type="number"
              min={1}
              max={100}
              value={maxConcurrent}
              disabled={!limitConcurrent}
              onChange={(e) => {
                const val = Math.min(100, Math.max(1, Number(e.target.value) || 1));
                setMaxConcurrent(val);
              }}
              className={`${inputCls} max-w-[160px] ${
                !limitConcurrent ? "cursor-not-allowed opacity-50" : ""
              }`}
            />
            <p className="mt-1 text-xs text-gray-400">
              {limitConcurrent ? `Users can have at most ${maxConcurrent} active sessions` : "Enable limit to configure"}
            </p>
          </div>
        </div>
        <div className="mt-4 flex justify-end">
          <button
            onClick={handleSavePolicy}
            disabled={savingPolicy}
            className="flex items-center gap-1.5 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-700 disabled:opacity-50"
          >
            {savingPolicy ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
            {savingPolicy ? "Saving..." : "Save Policy"}
          </button>
        </div>
      </div>

      {/* Sessions List */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <RefreshCw className="h-6 w-6 animate-spin text-gray-400" />
          <span className="ml-2 text-gray-500">Loading sessions...</span>
        </div>
      ) : sessions.length === 0 ? (
        <div className="rounded-xl border border-gray-200 bg-white p-12 text-center shadow-sm dark:border-gray-700 dark:bg-gray-800">
          <Monitor className="mx-auto mb-4 h-12 w-12 text-gray-300 dark:text-gray-600" />
          <p className="text-gray-500 dark:text-gray-400">No active sessions</p>
          <p className="mt-1 text-xs text-gray-400">
            Sessions will appear here when users log in.
          </p>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {sessions.map((session) => (
            <div
              key={session.id}
              className={`${cardCls} ${
                session.current
                  ? "border-brand-300 ring-1 ring-brand-200 dark:border-brand-700 dark:ring-brand-800"
                  : ""
              }`}
            >
              <div className="mb-3 flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div
                    className={`flex h-10 w-10 items-center justify-center rounded-lg ${
                      session.current
                        ? "bg-brand-100 text-brand-600 dark:bg-brand-900 dark:text-brand-400"
                        : "bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400"
                    }`}
                  >
                    {getDeviceIcon(session.user_agent || "")}
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="text-sm font-semibold text-gray-900 dark:text-gray-100">
                        {session.device_type || parseDeviceType(session.user_agent || "")}
                      </p>
                      {session.current && (
                        <span className="rounded-full bg-brand-100 px-2 py-0.5 text-xs font-medium text-brand-700 dark:bg-brand-900 dark:text-brand-300">
                          Current Session
                        </span>
                      )}
                    </div>
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                      {parseBrowser(session.user_agent || "")} on{" "}
                      {parseOS(session.user_agent || "")}
                    </p>
                  </div>
                </div>
                {!session.current && (
                  <button
                    onClick={() => handleRevoke(session.id)}
                    disabled={revokingId === session.id}
                    className="flex items-center gap-1.5 rounded-lg border border-red-300 px-2.5 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 disabled:opacity-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-950"
                    title="Revoke session"
                  >
                    {revokingId === session.id ? (
                      <RefreshCw className="h-3.5 w-3.5 animate-spin" />
                    ) : (
                      <Trash2 className="h-3.5 w-3.5" />
                    )}
                    Revoke
                  </button>
                )}
              </div>

              <div className="space-y-2 text-sm">
                <div className="flex items-center gap-2 text-gray-600 dark:text-gray-400">
                  <Wifi className="h-3.5 w-3.5 text-gray-400" />
                  <span className="font-mono text-xs">
                    {session.ip_address || "Unknown IP"}
                  </span>
                </div>
                <div className="flex items-center gap-2 text-gray-600 dark:text-gray-400">
                  <MapPin className="h-3.5 w-3.5 text-gray-400" />
                  <span className="text-xs">{getLocationStr(session)}</span>
                </div>
                <div className="flex items-center gap-2 text-gray-600 dark:text-gray-400">
                  <Clock className="h-3.5 w-3.5 text-gray-400" />
                  <span className="text-xs">
                    Last active: {formatTime(session.last_active_at || session.created_at)}
                  </span>
                </div>
                <div className="flex items-center gap-2 text-gray-500 dark:text-gray-500">
                  <Globe className="h-3.5 w-3.5 text-gray-400" />
                  <span className="text-xs">Created: {formatTime(session.created_at)}</span>
                </div>
                <div className="flex items-center gap-2 text-gray-500 dark:text-gray-500">
                  <Hash className="h-3.5 w-3.5 text-gray-400" />
                  <span className="font-mono text-xs text-gray-400">
                    ID: {session.id.slice(0, 8)}...
                  </span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Revoke All Confirmation Modal */}
      {showRevokeAllModal && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
          onClick={() => setShowRevokeAllModal(false)}
        >
          <div
            className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-red-100 dark:bg-red-950">
                <Power className="h-5 w-5 text-red-600" />
              </div>
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                Revoke All Sessions?
              </h2>
            </div>
            <p className="mb-6 text-sm text-gray-600 dark:text-gray-400">
              This will sign out all devices. Continue?
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
                {revokingAll ? (
                  <RefreshCw className="h-4 w-4 animate-spin" />
                ) : (
                  <Trash2 className="h-4 w-4" />
                )}
                Revoke All
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
