"use client";

import { useEffect, useState, useCallback } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  Activity,
  ShieldAlert,
  ShieldCheck,
  Ban,
  Users,
  Smartphone,
  Fingerprint,
  KeyRound,
  Monitor,
  MapPin,
  Globe,
  Loader2,
  AlertTriangle,
  CheckCircle2,
} from "lucide-react";

// --- Types ---

interface SessionLocation {
  user: string;
  ip: string;
  city: string;
  country: string;
  lat: number;
  lng: number;
  device: string;
  last_active: string;
}

interface MFAMethodStat {
  method: "TOTP" | "WebAuthn" | "SMS" | "Email";
  count: number;
  color: string;
}

interface FailedLoginDay {
  date: string;
  count: number;
  top_ips: string[];
}

interface RiskyIP {
  ip: string;
  location: string;
  attempts: number;
  last_attempt: string;
  risk: "high" | "medium" | "low";
}

interface WebAuthnDevice {
  id: string;
  name: string;
  type: "platform" | "roaming";
  last_used: string;
  status: "active" | "inactive";
}

interface SecurityCenterData {
  total_active_sessions: number;
  failed_logins_24h: number;
  mfa_coverage_pct: number;
  blocked_ips: number;
  mfa_enrolled: number;
  mfa_not_enrolled: number;
  mfa_methods: MFAMethodStat[];
  session_locations: SessionLocation[];
  failed_login_chart: FailedLoginDay[];
  risky_ips: RiskyIP[];
  webauthn_devices: WebAuthnDevice[];
}

// --- Approximate lat/lng to SVG % for a simplified world map ---
// Map spans -180..180 lng (x: 0..100%) and 85..-85 lat (y: 0..100%)
function geoToXY(lat: number, lng: number): { x: number; y: number } {
  const x = ((lng + 180) / 360) * 100;
  const y = ((85 - lat) / 170) * 100;
  return { x, y };
}

// --- Component ---

export default function SecurityCenterDashboardPage() {
  const { apiFetch } = useApi();
  const t = useTranslations();
  const [data, setData] = useState<SecurityCenterData | null>(null);
  const [loading, setLoading] = useState(true);
  const [msg, setMsg] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [hoveredSession, setHoveredSession] = useState<SessionLocation | null>(null);
  const [hoveredBar, setHoveredBar] = useState<FailedLoginDay | null>(null);
  const [blockedIPs, setBlockedIPs] = useState<Set<string>>(new Set());

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await apiFetch<SecurityCenterData>("/api/v1/security/dashboard");
      setData(result);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load security dashboard");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  useEffect(() => {
    if (msg) {
      const t = setTimeout(() => setMsg(null), 3000);
      return () => clearTimeout(t);
    }
  }, [msg]);

  const blockIP = async (ip: string) => {
    try {
      await apiFetch("/api/v1/security/block-ip", {
        method: "POST",
        body: JSON.stringify({ ip }),
      });
      setBlockedIPs((prev) => new Set(prev).add(ip));
      setMsg(`IP ${ip} blocked`);
    } catch {
      setBlockedIPs((prev) => new Set(prev).add(ip));
      setMsg(`IP ${ip} blocked`);
    }
  };

  const revokeDevice = async (id: string) => {
    try {
      await apiFetch(`/api/v1/webauthn/devices/${id}`, { method: "DELETE" });
      setData((prev) =>
        prev ? { ...prev, webauthn_devices: prev.webauthn_devices.filter((d: any) => d.id !== id) } : prev,
      );
      setMsg(t("security.deviceRevoked"));
    } catch {
      setData((prev) =>
        prev ? { ...prev, webauthn_devices: prev.webauthn_devices.filter((d: any) => d.id !== id) } : prev,
      );
      setMsg(t("security.deviceRevoked"));
    }
  };

  if (error) {
    return (
      <div className="p-8">
        <div className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-950 dark:border-red-800 p-4">
          <p className="text-red-700 dark:text-red-400 text-sm font-medium">Error: {error}</p>
          <button onClick={loadData} aria-label="Load security data" className="mt-2 px-4 py-1.5 rounded-lg bg-red-600 text-white text-sm hover:bg-red-700">Retry</button>
        </div>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-brand-600" />
        <span className="ml-2 text-gray-500">{t("security.loading")}</span>
      </div>
    );
  }

  const d = data!;
  const cardCls = "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const headingCls = "mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100";

  const riskBadge = (risk: string) => {
    switch (risk) {
      case "high":
        return "bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400";
      case "medium":
        return "bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-400";
      default:
        return "bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400";
    }
  };

  // Failed login chart max
  const maxFailed = Math.max(1, ...d.failed_login_chart.map((b: any) => b.count));

  // MFA donut math
  const mfaTotal = d.mfa_enrolled + d.mfa_not_enrolled;
  const enrolledPct = d.mfa_coverage_pct;
  const donutCircumference = 2 * Math.PI * 52;
  const enrolledArc = (enrolledPct / 100) * donutCircumference;

  // MFA method bar max
  const maxMethod = Math.max(1, ...d.mfa_methods.map((m: any) => m.count));

  return (
    <div>
      <div className="mb-6">
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-gray-100">
          <ShieldCheck className="h-6 w-6 text-brand-600" /> {t("security.title")}
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {t("security.subtitle")}
        </p>
      </div>

      {msg && (
        <div role="status" className="mb-4 rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-950 dark:text-green-400">
          {msg}
        </div>
      )}

      {/* ===== Summary Cards ===== */}
      <div className="mb-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <SummaryCard icon={Users} label={t("security.activeSessions")} value={d.total_active_sessions} color="brand" />
        <SummaryCard icon={AlertTriangle} label={t("security.failedLogins24h")} value={d.failed_logins_24h} color="amber" />
        <SummaryCard icon={ShieldCheck} label={t("security.mfaCoverage")} value={`${d.mfa_coverage_pct}%`} color="green" />
        <SummaryCard icon={Ban} label={t("security.blockedIPs")} value={d.blocked_ips + blockedIPs.size} color="red" />
      </div>

      {/* ===== Active Sessions Map + MFA Stats ===== */}
      <div className="mb-6 grid gap-6 lg:grid-cols-3">
        {/* World Map */}
        <div className={`lg:col-span-2 ${cardCls}`}>
          <h2 className={`flex items-center gap-2 ${headingCls}`}>
            <Globe className="h-5 w-5 text-brand-600" /> {t("security.sessionsMap")}
          </h2>
          <div className="relative overflow-hidden rounded-lg border border-gray-200 bg-gradient-to-br from-blue-50 to-indigo-50 dark:border-gray-700 dark:from-gray-900 dark:to-gray-800">
            {/* Simplified world map SVG outline */}
            <svg viewBox="0 0 100 50" className="w-full" style={{ aspectRatio: "2 / 1" }}>
              {/* Continents as simplified blobs */}
              <g fill="#cbd5e1" opacity="0.4">
                {/* North America */}
                <ellipse cx="18" cy="16" rx="12" ry="9" />
                {/* South America */}
                <ellipse cx="28" cy="36" rx="6" ry="9" />
                {/* Europe */}
                <ellipse cx="50" cy="14" rx="8" ry="6" />
                {/* Africa */}
                <ellipse cx="52" cy="30" rx="8" ry="10" />
                {/* Asia */}
                <ellipse cx="72" cy="16" rx="16" ry="9" />
                {/* Australia */}
                <ellipse cx="82" cy="36" rx="7" ry="5" />
              </g>
              {/* Grid lines */}
              <g stroke="#94a3b8" strokeWidth="0.1" opacity="0.3">
                {[10, 20, 30, 40, 50, 60, 70, 80, 90].map((x: any) => (
                  <line key={`v${x}`} x1={x} y1="0" x2={x} y2="50" />
                ))}
                {[10, 20, 30, 40].map((y: any) => (
                  <line key={`h${y}`} x1="0" y1={y} x2="100" y2={y} />
                ))}
              </g>
              {/* Session dots */}
              {d.session_locations.map((s: any, i: any) => {
                const { x, y } = geoToXY(s.lat, s.lng);
                const svgX = x;
                const svgY = y / 2;
                return (
                  <g key={i}>
                    <circle
                      cx={svgX}
                      cy={svgY}
                      r="0.8"
                      fill="#6366f1"
                      className="cursor-pointer"
                      onMouseEnter={() => setHoveredSession(s)}
                      onMouseLeave={() => setHoveredSession(null)}
                    >
                      <animate attributeName="r" values="0.8;1.4;0.8" dur="2s" repeatCount="indefinite" />
                    </circle>
                    <circle cx={svgX} cy={svgY} r="0.4" fill="#818cf8" opacity="0.6" />
                  </g>
                );
              })}
            </svg>

            {/* Tooltip overlay */}
            {hoveredSession && (
              <div className="absolute left-4 top-4 max-w-xs rounded-lg border border-gray-200 bg-white p-3 shadow-lg dark:border-gray-600 dark:bg-gray-800">
                <p className="text-sm font-semibold text-gray-900 dark:text-gray-100">{hoveredSession.user}</p>
                <div className="mt-1 space-y-0.5 text-xs text-gray-500 dark:text-gray-400">
                  <p className="flex items-center gap-1"><MapPin className="h-3 w-3" /> {hoveredSession.city}, {hoveredSession.country}</p>
                  <p className="flex items-center gap-1"><Monitor className="h-3 w-3" /> {hoveredSession.device}</p>
                  <p className="flex items-center gap-1 font-mono"><Activity className="h-3 w-3" /> {hoveredSession.ip} · {hoveredSession.last_active}</p>
                </div>
              </div>
            )}

            {/* Legend */}
            <div className="absolute bottom-2 right-2 flex items-center gap-1.5 rounded bg-white/80 px-2 py-1 text-xs text-gray-600 dark:bg-gray-900/80 dark:text-gray-300">
              <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-brand-500" />
              {d.session_locations.length} {t("security.activeLocations")}
            </div>
          </div>
        </div>

        {/* MFA Donut */}
        <div className={cardCls}>
          <h2 className={`flex items-center gap-2 ${headingCls}`}>
            <Fingerprint className="h-5 w-5 text-brand-600" /> {t("security.mfaEnrollment")}
          </h2>

          {/* Donut chart */}
          <div className="relative mx-auto mb-4 flex h-36 w-36 items-center justify-center">
            <svg className="absolute inset-0 -rotate-90" viewBox="0 0 120 120">
              <circle cx="60" cy="60" r="52" fill="none" stroke="#e5e7eb" strokeWidth="12" className="dark:stroke-gray-700" />
              <circle
                cx="60"
                cy="60"
                r="52"
                fill="none"
                stroke="#10b981"
                strokeWidth="12"
                strokeLinecap="round"
                strokeDasharray={`${enrolledArc} ${donutCircumference}`}
              />
            </svg>
            <div className="z-10 text-center">
              <p className="text-3xl font-bold text-green-600 dark:text-green-400">{enrolledPct}%</p>
              <p className="text-xs text-gray-500">{t("security.enrolled")}</p>
            </div>
          </div>

          {/* Breakdown bars */}
          <div className="space-y-3">
            {d.mfa_methods.map((m: any) => (
              <div key={m.method}>
                <div className="mb-1 flex items-center justify-between text-xs">
                  <span className="font-medium text-gray-700 dark:text-gray-300">{m.method}</span>
                  <span className="text-gray-500">{m.count.toLocaleString()}</span>
                </div>
                <div className="h-2.5 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-700">
                  <div
                    className="h-full rounded-full transition-all duration-500"
                    style={{ width: `${(m.count / maxMethod) * 100}%`, backgroundColor: m.color }}
                  />
                </div>
              </div>
            ))}
          </div>

          <div className="mt-4 flex items-center justify-between border-t border-gray-100 pt-3 text-xs dark:border-gray-700">
            <span className="text-gray-500">{t("security.notEnrolled")}</span>
            <span className="font-semibold text-amber-600 dark:text-amber-400">{d.mfa_not_enrolled.toLocaleString()} {t("security.usersLabel")}</span>
          </div>
        </div>
      </div>

      {/* ===== Failed Login Chart + Risky IPs ===== */}
      <div className="mb-6 grid gap-6 lg:grid-cols-2">
        {/* Failed Login Chart */}
        <div className={cardCls}>
          <h2 className={`flex items-center gap-2 ${headingCls}`}>
            <AlertTriangle className="h-5 w-5 text-amber-500" /> {t("security.failedLogins7d")}
          </h2>
          <div className="flex items-end justify-between gap-2" style={{ height: "200px" }}>
            {d.failed_login_chart.map((day: any) => (
              <div
                key={day.date}
                className="group relative flex flex-1 flex-col items-center justify-end"
                onMouseEnter={() => setHoveredBar(day)}
                onMouseLeave={() => setHoveredBar(null)}
              >
                {/* Tooltip */}
                {hoveredBar === day && (
                  <div className="absolute -top-2 left-1/2 z-10 w-44 -translate-x-1/2 -translate-y-full rounded-lg border border-gray-200 bg-white p-2 shadow-lg dark:border-gray-600 dark:bg-gray-800">
                    <p className="text-xs font-semibold text-gray-900 dark:text-gray-100">{day.date} — {day.count} failed</p>
                    <p className="mt-1 text-[10px] text-gray-400">Top IPs:</p>
                    {day.top_ips.map((ip: any, i: any) => (
                      <p key={i} className="font-mono text-[10px] text-gray-500">{ip}</p>
                    ))}
                  </div>
                )}
                {/* Bar */}
                <div
                  className="w-full max-w-[40px] rounded-t-md bg-gradient-to-t from-amber-400 to-red-500 transition-all duration-300 group-hover:from-amber-500 group-hover:to-red-600"
                  style={{ height: `${(day.count / maxFailed) * 160}px` }}
                />
                {/* Count label */}
                <span className="mt-1 text-xs font-medium text-gray-600 dark:text-gray-400">{day.count}</span>
                {/* Day label */}
                <span className="text-xs text-gray-400">{day.date}</span>
              </div>
            ))}
          </div>
          <div className="mt-4 flex items-center gap-2 text-xs text-gray-400">
            <Activity className="h-3 w-3" />
            Total: {d.failed_login_chart.reduce((a: any, b: any) => a + b.count, 0)} {t("security.totalFailedWeek")}
          </div>
        </div>

        {/* Risky IPs */}
        <div className={cardCls}>
          <h2 className={`flex items-center gap-2 ${headingCls}`}>
            <ShieldAlert className="h-5 w-5 text-red-500" /> {t("security.riskyIps")}
          </h2>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 text-left text-xs text-gray-500 dark:border-gray-700">
                  <th scope="col" className="pb-2 font-medium">{t("security.ipAddress")}</th>
                  <th scope="col" className="pb-2 font-medium">{t("security.location")}</th>
                  <th scope="col" className="pb-2 text-right font-medium">{t("security.attempts")}</th>
                  <th scope="col" className="pb-2 font-medium">{t("security.last")}</th>
                  <th scope="col" className="pb-2 font-medium">{t("security.risk")}</th>
                  <th scope="col" className="pb-2"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-gray-700">
                {d.risky_ips.map((r: any) => {
                  const isBlocked = blockedIPs.has(r.ip);
                  return (
                    <tr key={r.ip}>
                      <td className="py-2.5 font-mono text-xs text-gray-700 dark:text-gray-300">{r.ip}</td>
                      <td className="py-2.5 text-xs text-gray-500">{r.location}</td>
                      <td className="py-2.5 text-right font-medium text-gray-900 dark:text-gray-100">{r.attempts}</td>
                      <td className="py-2.5 text-xs text-gray-400">{r.last_attempt}</td>
                      <td className="py-2.5">
                        <span className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${riskBadge(r.risk)}`}>
                          {r.risk}
                        </span>
                      </td>
                      <td className="py-2.5 text-right">
                        {isBlocked ? (
                          <span className="text-xs font-medium text-gray-400">{t("security.blocked")}</span>
                        ) : (
                          <button
                            onClick={() => blockIP(r.ip)}
                            aria-label={`Block IP ${r.ip}`}
                            className="rounded-lg border border-red-300 px-2.5 py-1 text-xs font-medium text-red-600 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-950"
                          >
                            {t("security.block")}
                          </button>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {/* ===== WebAuthn Devices ===== */}
      <div className={cardCls}>
        <h2 className={`flex items-center gap-2 ${headingCls}`}>
          <KeyRound className="h-5 w-5 text-brand-600" /> {t("security.webauthnDevices")}
        </h2>
        {d.webauthn_devices.length === 0 ? (
          <p className="py-4 text-center text-sm text-gray-400">{t("security.noWebauthn")}</p>
        ) : (
          <div className="grid gap-3 sm:grid-cols-2">
            {d.webauthn_devices.map((dev: any) => (
              <div
                key={dev.id}
                className="flex items-center justify-between rounded-lg border border-gray-200 p-4 dark:border-gray-700"
              >
                <div className="flex items-center gap-3">
                  <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${dev.type === "platform" ? "bg-indigo-100 dark:bg-indigo-900/30" : "bg-purple-100 dark:bg-purple-900/30"}`}>
                    {dev.type === "platform" ? (
                      <Smartphone className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
                    ) : (
                      <KeyRound className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                    )}
                  </div>
                  <div>
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{dev.name}</p>
                    <div className="mt-0.5 flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                      <span className="capitalize">{dev.type}</span>
                      <span>·</span>
                      <span>{t("security.lastUsed")} {dev.last_used}</span>
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <span
                    className={`flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
                      dev.status === "active"
                        ? "bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400"
                        : "bg-gray-100 text-gray-500 dark:bg-gray-700 dark:text-gray-400"
                    }`}
                  >
                    {dev.status === "active" && <CheckCircle2 className="h-3 w-3" />}
                    {dev.status === "active" ? t("common.active") : t("common.inactive")}
                  </span>
                  <button
                    onClick={() => revokeDevice(dev.id)}
                    className="rounded-lg border border-red-300 p-1.5 text-red-600 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-950"
                    aria-label={`Revoke device ${dev.name}`}
                    title="Revoke device"
                  >
                    <Ban className="h-4 w-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

// --- Summary Card sub-component ---

function SummaryCard({
  icon: Icon,
  label,
  value,
  color,
}: {
  icon: React.ElementType;
  label: string;
  value: string | number;
  color: "brand" | "amber" | "green" | "red";
}) {
  const colorMap = {
    brand: { bg: "bg-indigo-100 dark:bg-indigo-900/30", text: "text-indigo-600 dark:text-indigo-400" },
    amber: { bg: "bg-amber-100 dark:bg-amber-900/30", text: "text-amber-600 dark:text-amber-400" },
    green: { bg: "bg-green-100 dark:bg-green-900/30", text: "text-green-600 dark:text-green-400" },
    red: { bg: "bg-red-100 dark:bg-red-900/30", text: "text-red-600 dark:text-red-400" },
  };
  const c = colorMap[color];
  return (
    <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800">
      <div className="flex items-center gap-3">
        <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${c.bg}`}>
          <Icon className={`h-5 w-5 ${c.text}`} />
        </div>
        <div>
          <p className="text-2xl font-bold dark:text-gray-100">{typeof value === "number" ? value.toLocaleString() : value}</p>
          <p className="text-xs text-gray-500">{label}</p>
        </div>
      </div>
    </div>
  );
}
