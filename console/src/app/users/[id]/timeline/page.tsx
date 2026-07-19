"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Clock, Shield, AlertTriangle, Check, X, Eye,
  Loader2, Activity, Lock, RefreshCw, KeyRound, UserCog,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

interface TimelineEvent {
  id: string;
  timestamp: string;
  type: string;
  severity: "info" | "warning" | "error" | "critical";
  description: string;
  risk_level?: number;
  details?: Record<string, unknown>;
  is_risk: boolean;
}

const eventIcons: Record<string, typeof Clock> = {
  login: Lock, logout: Lock, token_refresh: RefreshCw,
  password_change: KeyRound, mfa_verify: Shield,
  role_assign: UserCog, itdr: AlertTriangle, cae: Activity, policy_deny: X,
};

const severityColors: Record<string, string> = {
  info: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
  warning: "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300",
  error: "bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300",
  critical: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
};

const riskEventTypes = ["itdr", "cae", "policy_deny"];

export default function UserTimelinePage({ params }: { params: { id: string } }) {
  const t = useTranslations();
  const userId = params.id;
  const [events, setEvents] = useState<TimelineEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [showRiskOnly, setShowRiskOnly] = useState(false);
  const [showRiskOverlay, setShowRiskOverlay] = useState(true);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/users/${userId}/timeline`, { headers: { ...authHeader() } });
      if (res.ok) { const d = await res.json(); setEvents(d.events || d || []); return; }
    } catch { /* mock */ }
    setEvents([
      { id: "1", timestamp: "2025-07-18T09:35:00Z", type: "login", severity: "info", description: "Login via passkey from 192.168.1.100", is_risk: false },
      { id: "2", timestamp: "2025-07-18T09:32:00Z", type: "cae", severity: "warning", description: "CAE: Device posture dropped to 65", risk_level: 65, is_risk: true, details: { prev_score: 85, new_score: 65, action: "step_up_required" } },
      { id: "3", timestamp: "2025-07-18T09:28:00Z", type: "itdr", severity: "critical", description: "ITDR: Impossible travel detected (CN→US in 2min)", risk_level: 92, is_risk: true, details: { detection: "impossible_travel", prev_location: "Shanghai, CN", new_location: "San Francisco, US", time_delta_min: 2 } },
      { id: "4", timestamp: "2025-07-18T09:25:00Z", type: "token_refresh", severity: "info", description: "Access token refreshed", is_risk: false },
      { id: "5", timestamp: "2025-07-18T09:20:00Z", type: "policy_deny", severity: "error", description: "Conditional access policy denied: risk_score > 80", risk_level: 85, is_risk: true, details: { policy: "High Risk MFA Required", condition: "risk_score > 60", action: "require_mfa" } },
      { id: "6", timestamp: "2025-07-18T09:15:00Z", type: "mfa_verify", severity: "info", description: "TOTP verification successful", is_risk: false },
      { id: "7", timestamp: "2025-07-18T09:10:00Z", type: "login", severity: "warning", description: "Failed login attempt (wrong password)", is_risk: false },
      { id: "8", timestamp: "2025-07-17T18:00:00Z", type: "role_assign", severity: "info", description: "Role 'engineer' assigned by admin@company.com", is_risk: false },
      { id: "9", timestamp: "2025-07-17T14:30:00Z", type: "password_change", severity: "info", description: "Password changed successfully", is_risk: false },
      { id: "10", timestamp: "2025-07-17T09:00:00Z", type: "cae", severity: "info", description: "CAE: Session evaluated — continue", risk_level: 20, is_risk: true, details: { action: "continue", risk_score: 20 } },
    ]);
  }, [userId]);

  useEffect(() => { load(); }, [load]);

  const toggleExpand = (id: string) => {
    const next = new Set(expanded);
    if (next.has(id)) next.delete(id); else next.add(id);
    setExpanded(next);
  };

  const filtered = events.filter((e: any) => {
    if (showRiskOnly && !e.is_risk) return false;
    if (!showRiskOverlay && e.is_risk && !riskEventTypes.includes(e.type)) return false;
    return true;
  });

  if (loading) {
    return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>;
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Clock className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">User Timeline</h1>
            <span className="px-2 py-0.5 text-xs bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-300 rounded font-mono">{userId}</span>
          </div>
        </div>

        {/* Risk Overlay Controls */}
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4 mb-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white flex items-center gap-2">
                <AlertTriangle className="w-4 h-4 text-orange-500" />
                {t("userTimeline.riskOverlay")}
              </h3>
              <p className="text-xs text-gray-500 dark:text-gray-400">{t("userTimeline.riskOverlayDesc")}</p>
            </div>
            <div className="flex items-center gap-4">
              <label className="flex items-center gap-2 cursor-pointer">
                <input type="checkbox" checked={showRiskOverlay} onChange={(e) => setShowRiskOverlay(e.target.checked)} className="rounded" />
                <span className="text-xs text-gray-600 dark:text-gray-400">{t("userTimeline.showRiskEvents")}</span>
              </label>
              <button onClick={() => setShowRiskOnly(!showRiskOnly)}
                className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                  showRiskOnly ? "bg-red-600 text-white" : "bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400"
                }`}>
                {showRiskOnly ? t("userTimeline.filterRiskOnly") : t("userTimeline.filterAll")}
              </button>
            </div>
          </div>

          {/* Event type legend */}
          <div className="flex flex-wrap gap-2 mt-3 pt-3 border-t border-gray-100 dark:border-gray-800">
            {Object.entries({
              login: "bg-blue-500", logout: "bg-gray-500", token_refresh: "bg-cyan-500",
              password_change: "bg-purple-500", mfa_verify: "bg-green-500", role_assign: "bg-indigo-500",
              itdr: "bg-red-500", cae: "bg-orange-500", policy_deny: "bg-pink-500",
            }).map(([type, color]: any[]) => (
              <div key={type} className="flex items-center gap-1">
                <div className={`w-2.5 h-2.5 rounded-full ${color}`} />
                <span className="text-xs text-gray-500 dark:text-gray-400">
                  {t(`userTimeline.eventTypes.${type}`)}
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* Timeline */}
        {filtered.length === 0 ? (
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-12 text-center">
            <Clock className="w-12 h-12 mx-auto mb-3 text-gray-300" />
            <p className="text-sm text-gray-500">{t("userTimeline.noEvents")}</p>
          </div>
        ) : (
          <div className="relative">
            {/* Vertical line */}
            <div className="absolute left-4 top-0 bottom-0 w-0.5 bg-gray-200 dark:bg-gray-800" />

            <div className="space-y-3">
              {filtered.map((e: any) => {
                const Icon = eventIcons[e.type] || Clock;
                const isRisk = e.is_risk && riskEventTypes.includes(e.type);
                return (
                  <div key={e.id} className="relative pl-12">
                    {/* Node */}
                    <div className={`absolute left-2 top-3 w-5 h-5 rounded-full flex items-center justify-center ring-4 ring-gray-50 dark:ring-gray-950 ${
                      isRisk ? (e.severity === "critical" ? "bg-red-500" : "bg-orange-500") :
                      e.type === "login" ? "bg-blue-500" :
                      e.type === "mfa_verify" ? "bg-green-500" :
                      "bg-gray-400"
                    }`}>
                      <Icon className="w-3 h-3 text-white" />
                    </div>

                    {/* Card */}
                    <div
                      onClick={() => e.details && toggleExpand(e.id)}
                      className={`bg-white dark:bg-gray-900 rounded-lg border p-3 transition-all ${
                        isRisk ? "border-orange-200 dark:border-orange-900" : "border-gray-200 dark:border-gray-800"
                      } ${e.details ? "cursor-pointer hover:border-gray-300 dark:hover:border-gray-700" : ""}`}
                    >
                      <div className="flex items-start justify-between gap-3">
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <span className={`px-1.5 py-0.5 text-xs rounded ${severityColors[e.severity]}`}>
                              {t(`userTimeline.severity.${e.severity}`)}
                            </span>
                            <span className="text-xs text-gray-400">{new Date(e.timestamp).toLocaleString()}</span>
                            {isRisk && e.risk_level !== undefined && (
                              <span className={`px-1.5 py-0.5 text-xs rounded-full font-medium ${
                                e.risk_level >= 80 ? "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300" :
                                e.risk_level >= 50 ? "bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300" :
                                "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300"
                              }`}>
                                {t("userTimeline.riskLevel")}: {e.risk_level}
                              </span>
                            )}
                          </div>
                          <p className="text-sm text-gray-900 dark:text-white mt-1">{e.description}</p>
                        </div>
                        {e.details && (
                          <div className="flex-shrink-0">
                            {expanded.has(e.id) ? <Eye className="w-4 h-4 text-gray-400" /> : <Eye className="w-4 h-4 text-gray-300" />}
                          </div>
                        )}
                      </div>

                      {/* Expanded details */}
                      {e.details && expanded.has(e.id) && (
                        <div className="mt-3 pt-3 border-t border-gray-100 dark:border-gray-800">
                          <span className="text-xs font-medium text-gray-500 mb-1 block">{t("userTimeline.details")}:</span>
                          <pre className="text-xs p-2 bg-gray-100 dark:bg-gray-800 rounded overflow-x-auto text-gray-700 dark:text-gray-300">
                            {JSON.stringify(e.details, null, 2)}
                          </pre>
                        </div>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
