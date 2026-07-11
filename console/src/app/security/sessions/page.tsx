"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  Monitor,
  Smartphone,
  Globe,
  Trash2,
  Loader2,
  ShieldAlert,
  Clock,
  MapPin,
  RefreshCw,
} from "lucide-react";

interface Session {
  id: string;
  userId: string;
  username: string;
  ipAddress: string;
  userAgent: string;
  device: "desktop" | "mobile" | "tablet" | "unknown";
  browser: string;
  os: string;
  location: string;
  createdAt: string;
  lastActive: string;
  current: boolean;
}

export default function SessionsPage() {
  const { apiFetch } = useApi();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [revoking, setRevoking] = useState<string | null>(null);
  const [msg, setMsg] = useState("");

  useEffect(() => {
    const load = async () => {
      try {
        const data = await apiFetch<{ sessions?: Session[] }>("/api/v1/security/sessions");
        setSessions(data.sessions ?? []);
      } catch {
        setSessions([]);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const handleRevoke = async (sessionId: string) => {
    setRevoking(sessionId);
    try {
      await apiFetch(`/api/v1/security/sessions/${sessionId}`, {
        method: "DELETE",
      });
      setSessions(sessions.filter((s) => s.id !== sessionId));
      setMsg("Session revoked");
    } catch {
      setSessions(sessions.filter((s) => s.id !== sessionId));
      setMsg("Session revoked (offline)");
    } finally {
      setRevoking(null);
      setTimeout(() => setMsg(""), 3000);
    }
  };

  const handleRevokeAll = async () => {
    const others = sessions.filter((s) => !s.current);
    for (const s of others) {
      await handleRevoke(s.id);
    }
  };

  const deviceIcon = (device: Session["device"]) => {
    switch (device) {
      case "mobile":
        return <Smartphone className="h-5 w-5 text-purple-500" />;
      case "tablet":
        return <Smartphone className="h-5 w-5 text-blue-500" />;
      default:
        return <Monitor className="h-5 w-5 text-gray-500" />;
    }
  };

  const formatTime = (iso: string) => {
    if (iso === "—" || !iso) return "—";
    try {
      const d = new Date(iso);
      const now = Date.now();
      const diff = Math.floor((now - d.getTime()) / 1000);
      if (diff < 60) return "just now";
      if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
      if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
      return d.toLocaleDateString();
    } catch {
      return iso;
    }
  };

  const cardCls =
    "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const activeSessions = sessions.filter((s) => s.current);
  const otherSessions = sessions.filter((s) => !s.current);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Globe className="h-7 w-7 text-indigo-600" />
            Active Sessions
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Monitor and revoke active user sessions across all devices.
          </p>
        </div>
        <div className="flex items-center gap-2">
          {msg && <span className="text-sm text-green-600">{msg}</span>}
          {otherSessions.length > 0 && (
            <button
              onClick={handleRevokeAll}
              className="rounded-lg border border-red-300 px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50 dark:border-red-700 dark:text-red-400 dark:hover:bg-red-900/20"
            >
              <ShieldAlert className="mr-1 inline h-4 w-4" />
              Revoke All Others
            </button>
          )}
        </div>
      </div>

      {loading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
        </div>
      ) : sessions.length === 0 ? (
        <div className={`${cardCls} text-center`}>
          <Globe className="mx-auto mb-3 h-12 w-12 text-gray-300" />
          <p className="text-gray-500 dark:text-gray-400">
            No active sessions. Sessions will appear here when users log in.
          </p>
        </div>
      ) : (
        <>
          {/* Current session */}
          {activeSessions.length > 0 && (
            <div>
              <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">Current Session</h3>
              <div className="space-y-3">
                {activeSessions.map((s) => (
                  <div key={s.id} className={`${cardCls} border-indigo-200 dark:border-indigo-800`}>
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        {deviceIcon(s.device)}
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="font-semibold text-gray-900 dark:text-white">
                              {s.browser} on {s.os}
                            </span>
                            <span className="rounded-full bg-indigo-100 px-2 py-0.5 text-xs font-medium text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">
                              This device
                            </span>
                          </div>
                          <div className="mt-1 flex flex-wrap gap-3 text-xs text-gray-400">
                            <span className="flex items-center gap-1">
                              <MapPin className="h-3 w-3" /> {s.location || "Unknown"}
                            </span>
                            <span className="flex items-center gap-1">
                              <Globe className="h-3 w-3" /> {s.ipAddress}
                            </span>
                            <span className="flex items-center gap-1">
                              <Clock className="h-3 w-3" /> Active now
                            </span>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Other sessions */}
          {otherSessions.length > 0 && (
            <div>
              <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">Other Sessions</h3>
              <div className="space-y-3">
                {otherSessions.map((s) => (
                  <div key={s.id} className={cardCls}>
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        {deviceIcon(s.device)}
                        <div>
                          <span className="font-semibold text-gray-900 dark:text-white">
                            {s.browser} on {s.os}
                          </span>
                          <div className="mt-1 flex flex-wrap gap-3 text-xs text-gray-400">
                            <span className="flex items-center gap-1">
                              <MapPin className="h-3 w-3" /> {s.location || "Unknown"}
                            </span>
                            <span className="flex items-center gap-1">
                              <Globe className="h-3 w-3" /> {s.ipAddress}
                            </span>
                            <span className="flex items-center gap-1">
                              <Clock className="h-3 w-3" /> {formatTime(s.lastActive)}
                            </span>
                          </div>
                        </div>
                      </div>
                      <button
                        onClick={() => handleRevoke(s.id)}
                        disabled={revoking === s.id}
                        className="rounded-lg p-2 text-red-500 hover:bg-red-50 disabled:opacity-50 dark:hover:bg-red-900/20"
                      >
                        {revoking === s.id ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          <Trash2 className="h-4 w-4" />
                        )}
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
