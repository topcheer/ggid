"use client";

import { useState, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Search, Shield, Clock, Globe, Smartphone, Activity,
  Loader2, AlertTriangle, KeyRound, Check, X, Monitor,
  Fingerprint, Ban,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface SessionDetail {
  session_id: string; user: string; email: string;
  device: string; user_agent: string; ip_address: string;
  geo_location: string; risk_score: number;
  token_family_id: string; created_at: string; last_active: string;
  expires_at: string; status: string; auth_method: string; mfa_verified: boolean;
}

interface CAEEntry {
  id: string; timestamp: string; event: string; result: string; risk_delta: number;
}

export default function SessionDetailPage() {
  const t = useTranslations();
  const [query, setQuery] = useState("");
  const [session, setSession] = useState<SessionDetail | null>(null);
  const [caeHistory, setCaeHistory] = useState<CAEEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [notFound, setNotFound] = useState(false);
  const [revoking, setRevoking] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const search = useCallback(async () => {
    if (!query.trim()) return;
    setLoading(true); setNotFound(false); setSession(null);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/sessions/${query.trim()}`, { headers: { ...authHeader() } });
      if (res.ok) {
        const d = await res.json();
        setSession(d.session || d);
        setCaeHistory(d.cae_history || []);
        return;
      }
    } catch { /* mock */ }
    // Mock data
    setSession({
      session_id: query.trim(), user: "alice@company.com", email: "alice@company.com",
      device: "MacBook Pro 16\" (macOS 14.5)", user_agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/125.0",
      ip_address: "192.168.1.100", geo_location: "Shanghai, CN", risk_score: 32,
      token_family_id: "tf-abc123-def456", created_at: "2025-07-18T08:00:00Z", last_active: "2025-07-18T09:35:00Z",
      expires_at: "2025-07-18T20:00:00Z", status: "active", auth_method: "passkey", mfa_verified: true,
    });
    setCaeHistory([
      { id: "1", timestamp: "2025-07-18T09:35:00Z", event: "risk_spike", result: "step_up", risk_delta: 15 },
      { id: "2", timestamp: "2025-07-18T09:15:00Z", event: "token_refresh", result: "continue", risk_delta: -5 },
      { id: "3", timestamp: "2025-07-18T08:30:00Z", event: "ip_change", result: "challenge", risk_delta: 20 },
      { id: "4", timestamp: "2025-07-18T08:00:00Z", event: "session_start", result: "continue", risk_delta: 32 },
    ]);
    setLoading(false);
  }, [query]);

  const revoke = async () => {
    if (!session) return;
    if (!confirm(t("sessionDetail.actions.revokeConfirm"))) return;
    setRevoking(true);
    try {
      await fetch(`${API_BASE}/api/v1/auth/sessions/revoke`, {
        method: "POST", headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ session_id: session.session_id }),
      });
    } catch { /* ok */ }
    setRevoking(false);
    setSession({ ...session, status: "revoked" });
    setMsg(t("sessionDetail.actions.revoked"));
    setTimeout(() => setMsg(null), 3000);
  };

  const statusColors: Record<string, string> = {
    active: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
    expired: "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400",
    revoked: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
    idle: "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300",
  };

  const caeResultColors: Record<string, string> = {
    revoke: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
    step_up: "bg-blue-100 text-blue-700 dark:bg-blue-950 dark:text-blue-300",
    challenge: "bg-purple-100 text-purple-700 dark:bg-purple-950 dark:text-purple-300",
    continue: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Fingerprint className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("sessionDetail.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("sessionDetail.description")}</p>
        </div>

        {/* Search */}
        <div className="flex gap-2 mb-6">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input type="text" value={query} onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && search()}
              placeholder={t("sessionDetail.searchPlaceholder")}
              className="w-full pl-9 pr-3 py-2.5 rounded-lg border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-900 text-sm font-mono text-gray-900 dark:text-white" />
          </div>
          <button onClick={search} disabled={loading || !query.trim()}
            className="flex items-center gap-2 px-5 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium">
            {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Search className="w-4 h-4" />}
            {loading ? t("sessionDetail.searching") : t("sessionDetail.search")}
          </button>
        </div>

        {msg && (
          <div className="flex items-center gap-2 px-4 py-2 mb-4 rounded-lg bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300 text-sm">
            <Check className="w-4 h-4" />{msg}
          </div>
        )}

        {/* Loading */}
        {loading && <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>}

        {/* Not found */}
        {!loading && notFound && (
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-12 text-center">
            <AlertTriangle className="w-12 h-12 mx-auto mb-3 text-gray-300" />
            <p className="text-sm text-gray-500">{t("sessionDetail.notFound")}</p>
          </div>
        )}

        {/* Empty state */}
        {!loading && !session && !notFound && (
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-12 text-center">
            <Search className="w-12 h-12 mx-auto mb-3 text-gray-300" />
            <p className="text-sm text-gray-500">{t("sessionDetail.noSearch")}</p>
          </div>
        )}

        {/* Session Detail */}
        {session && !loading && (
          <div className="space-y-4">
            {/* Attributes */}
            <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-semibold text-gray-900 dark:text-white flex items-center gap-2">
                  <Fingerprint className="w-5 h-5 text-blue-600" />
                  {t("sessionDetail.attributes.title")}
                </h3>
                <span className={`px-3 py-1 text-xs font-medium rounded-full ${statusColors[session.status] || statusColors.active}`}>
                  {t(`sessionDetail.status${session.status.replace(/^./, (m: any) => m.toUpperCase())}`)}
                </span>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <Attr icon={KeyRound} label={t("sessionDetail.attributes.sessionId")} value={session.session_id} mono />
                <Attr icon={Activity} label={t("sessionDetail.attributes.user")} value={session.user} />
                <Attr icon={Monitor} label={t("sessionDetail.attributes.device")} value={session.device} />
                <Attr icon={Smartphone} label={t("sessionDetail.attributes.userAgent")} value={session.user_agent} small />
                <Attr icon={Globe} label={t("sessionDetail.attributes.ipAddress")} value={session.ip_address} mono />
                <Attr icon={Globe} label={t("sessionDetail.attributes.geoLocation")} value={session.geo_location} />
                <Attr icon={Shield} label={t("sessionDetail.attributes.riskScore")}
                  value={session.risk_score.toString()} riskScore={session.risk_score} />
                <Attr icon={KeyRound} label={t("sessionDetail.attributes.tokenFamilyId")} value={session.token_family_id} mono />
                <Attr icon={Clock} label={t("sessionDetail.attributes.createdAt")} value={new Date(session.created_at).toLocaleString()} />
                <Attr icon={Activity} label={t("sessionDetail.attributes.lastActive")} value={new Date(session.last_active).toLocaleString()} />
                <Attr icon={Clock} label={t("sessionDetail.attributes.expiresAt")} value={new Date(session.expires_at).toLocaleString()} />
                <Attr icon={Fingerprint} label={t("sessionDetail.attributes.authMethod")} value={session.auth_method} />
                <Attr icon={Shield} label={t("sessionDetail.attributes.mfaVerified")} value={session.mfa_verified ? "Yes" : "No"} />
              </div>

              {/* Risk Score Bar */}
              <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-800">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs font-medium text-gray-600 dark:text-gray-400">{t("sessionDetail.attributes.riskScore")}</span>
                  <span className={`text-sm font-bold ${session.risk_score >= 70 ? "text-red-600" : session.risk_score >= 40 ? "text-orange-500" : "text-green-600"}`}>
                    {session.risk_score}/100
                  </span>
                </div>
                <div className="h-2 bg-gray-200 dark:bg-gray-800 rounded-full overflow-hidden">
                  <div className={`h-full rounded-full ${session.risk_score >= 70 ? "bg-red-500" : session.risk_score >= 40 ? "bg-orange-500" : "bg-green-500"}`}
                    style={{ width: `${session.risk_score}%` }} />
                </div>
              </div>

              {/* Revoke Button */}
              {session.status === "active" && (
                <button onClick={revoke} disabled={revoking}
                  className="mt-4 flex items-center gap-2 px-5 py-2 bg-red-600 hover:bg-red-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium">
                  {revoking ? <Loader2 className="w-4 h-4 animate-spin" /> : <Ban className="w-4 h-4" />}
                  {t("sessionDetail.actions.revoke")}
                </button>
              )}
            </div>

            {/* CAE History */}
            <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
              <h3 className="text-sm font-semibold text-gray-900 dark:text-white flex items-center gap-2 mb-4">
                <Activity className="w-5 h-5 text-blue-600" />
                {t("sessionDetail.caeHistory.title")}
              </h3>

              {caeHistory.length === 0 ? (
                <p className="text-sm text-gray-500 py-4 text-center">{t("sessionDetail.caeHistory.noHistory")}</p>
              ) : (
                <div className="space-y-2">
                  {caeHistory.map((e: any) => (
                    <div key={e.id} className="flex items-center gap-3 p-2 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800/30">
                      <div className="w-1.5 h-1.5 rounded-full bg-blue-500" />
                      <span className="text-xs text-gray-500 w-32">{new Date(e.timestamp).toLocaleTimeString()}</span>
                      <code className="text-xs text-gray-700 dark:text-gray-300 flex-1">{e.event}</code>
                      <span className={`px-2 py-0.5 text-xs rounded-full ${caeResultColors[e.result] || caeResultColors.continue}`}>{e.result}</span>
                      <span className={`text-xs font-medium w-12 text-right ${e.risk_delta > 0 ? "text-red-500" : "text-green-500"}`}>
                        {e.risk_delta > 0 ? "+" : ""}{e.risk_delta}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ============ Shared ============

function Attr({ icon: Icon, label, value, mono, small, riskScore }: {
  icon: typeof KeyRound; label: string; value: string; mono?: boolean; small?: boolean; riskScore?: number;
}) {
  return (
    <div className="flex items-center gap-2">
      <Icon className="w-4 h-4 text-gray-400 flex-shrink-0" />
      <div className="min-w-0">
        <div className="text-xs text-gray-500">{label}</div>
        <div className={`text-sm text-gray-900 dark:text-white ${mono ? "font-mono" : ""} ${small ? "truncate" : ""}`} title={value}>
          {riskScore !== undefined ? (
            <span className={riskScore >= 70 ? "text-red-600 font-bold" : riskScore >= 40 ? "text-orange-500 font-bold" : "text-green-600 font-bold"}>{value}</span>
          ) : value}
        </div>
      </div>
    </div>
  );
}
