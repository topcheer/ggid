"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  TrendingUp, PieChart as PieIcon, AlertTriangle, Users, Activity,
  Loader2, Shield, Eye, KeyRound, Smartphone, Globe, Fingerprint,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

type TabId = "growth" | "methods" | "risk";

export default function IdentityAnalyticsPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("growth");

  const tabs: { id: TabId; label: string; icon: typeof TrendingUp }[] = [
    { id: "growth", label: t("identityAnalytics.tabs.growth"), icon: TrendingUp },
    { id: "methods", label: t("identityAnalytics.tabs.methods"), icon: PieIcon },
    { id: "risk", label: t("identityAnalytics.tabs.risk"), icon: AlertTriangle },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Activity className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("identityAnalytics.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("identityAnalytics.description")}</p>
        </div>

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {tabs.map(({ id, label, icon: Icon }) => (
            <button key={id} onClick={() => setTab(id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                tab === id ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm" : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
              }`}>
              <Icon className="w-4 h-4" />
              {label}
            </button>
          ))}
        </div>

        {tab === "growth" && <GrowthTab />}
        {tab === "methods" && <MethodsTab />}
        {tab === "risk" && <RiskTab />}
      </div>
    </div>
  );
}

// ============ Growth Tab ============

function GrowthTab() {
  const t = useTranslations();
  const [data, setData] = useState<{ day: string; registered: number; active: number; dormant: number }[]>([]);
  const [loading, setLoading] = useState(true);
  const [range, setRange] = useState<"7d" | "30d">("30d");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/identity/dashboard/stats?range=${range}`, { headers: { ...authHeader() } });
      if (res.ok) {
        const d = await res.json();
        if (d.growth) { setData(d.growth); return; }
      }
    } catch { /* mock */ }
    // Generate mock trend
    const days = range === "7d" ? 7 : 30;
    setData(Array.from({ length: days }, (_, i) => {
      const date = new Date(); date.setDate(date.getDate() - (days - 1 - i));
      const base = 800 + i * 5;
      return {
        day: date.toISOString().split("T")[0],
        registered: base + Math.floor(Math.random() * 20),
        active: Math.floor(base * 0.65) + Math.floor(Math.random() * 30),
        dormant: Math.floor(base * 0.15) + Math.floor(Math.random() * 10),
      };
    }));
  }, [range]);

  useEffect(() => { load(); }, [load]);

  if (loading) return <Spinner />;

  const latest = data[data.length - 1] || { registered: 0, active: 0, dormant: 0 };
  const activeRate = latest.registered > 0 ? Math.round((latest.active / latest.registered) * 100) : 0;

  // SVG line chart
  const maxVal = Math.max(...data.map((d: any) => d.registered), 1);
  const chartW = 700, chartH = 200, pad = 30;
  const xStep = data.length > 1 ? (chartW - pad * 2) / (data.length - 1) : 0;
  const yScale = (v: number) => chartH - pad - (v / maxVal) * (chartH - pad * 2);

  const linePath = (key: "registered" | "active" | "dormant") =>
    data.map((d: any, i: any) => `${i === 0 ? "M" : "L"} ${pad + i * xStep} ${yScale(d[key])}`).join(" ");

  return (
    <div className="space-y-4">
      {/* Range selector */}
      <div className="flex gap-2">
        {(["7d", "30d"] as const).map((r: any) => (
          <button key={r} onClick={() => setRange(r)}
            className={`px-3 py-1.5 rounded-lg text-xs font-medium ${range === r ? "bg-blue-600 text-white" : "bg-white dark:bg-gray-800 text-gray-600 dark:text-gray-400 border border-gray-200 dark:border-gray-700"}`}>
            {r === "7d" ? t("identityAnalytics.growth.last7Days") : t("identityAnalytics.growth.last30Days")}
          </button>
        ))}
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard icon={Users} label={t("identityAnalytics.growth.totalUsers")} value={latest.registered} color="blue" />
        <StatCard icon={Activity} label={t("identityAnalytics.growth.activeRate")} value={`${activeRate}%`} color="green" />
        <StatCard icon={TrendingUp} label={t("identityAnalytics.growth.newThisWeek")} value={`+${data.slice(-7).reduce((s, d) => s + (d.registered - (data[data.indexOf(d) - 1]?.registered || d.registered)), 0)}`} color="blue" />
        <StatCard icon={Shield} label={t("identityAnalytics.growth.dormantRate")} value={latest.registered > 0 ? `${Math.round((latest.dormant / latest.registered) * 100)}%` : "0%"} color="orange" />
      </div>

      {/* Line Chart */}
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">{t("identityAnalytics.growth.title")}</h3>
        <svg viewBox={`0 0 ${chartW} ${chartH}`} className="w-full h-48">
          {/* Grid */}
          {[0, 0.25, 0.5, 0.75, 1].map((p: any) => (
            <line key={p} x1={pad} y1={pad + p * (chartH - pad * 2)} x2={chartW - pad} y2={pad + p * (chartH - pad * 2)}
              stroke="currentColor" className="text-gray-100 dark:text-gray-800" strokeWidth={1} />
          ))}
          {/* Lines */}
          <path d={linePath("registered")} fill="none" stroke="#3b82f6" strokeWidth={2} />
          <path d={linePath("active")} fill="none" stroke="#22c55e" strokeWidth={2} />
          <path d={linePath("dormant")} fill="none" stroke="#f97316" strokeWidth={2} strokeDasharray="4 2" />
        </svg>
        <div className="flex items-center justify-center gap-4 mt-2">
          <Legend color="bg-blue-500" label={t("identityAnalytics.growth.registered")} />
          <Legend color="bg-green-500" label={t("identityAnalytics.growth.active")} />
          <Legend color="bg-orange-500" label={t("identityAnalytics.growth.dormant")} />
        </div>
      </div>
    </div>
  );
}

// ============ Methods Tab ============

function MethodsTab() {
  const t = useTranslations();
  const [methods, setMethods] = useState<{ method: string; count: number }[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/passwordless/stats`, { headers: { ...authHeader() } });
      if (res.ok) {
        const d = await res.json();
        if (d.methods) { setMethods(d.methods); return; }
      }
    } catch { /* mock */ }
    setMethods([
      { method: "password", count: 850 },
      { method: "passkey", count: 320 },
      { method: "webauthn", count: 180 },
      { method: "social", count: 145 },
      { method: "totp", count: 210 },
      { method: "saml", count: 65 },
    ]);
  }, []);

  useEffect(() => { load(); }, [load]);

  if (loading) return <Spinner />;

  const total = methods.reduce((s, m) => s + m.count, 0);
  const passwordless = methods.filter((m: any) => m.method !== "password").reduce((s, m) => s + m.count, 0);
  const passwordlessRate = total > 0 ? Math.round((passwordless / total) * 100) : 0;

  // Pie chart
  const colors = ["#ef4444", "#3b82f6", "#8b5cf6", "#f59e0b", "#06b6d4", "#10b981"];
  const radius = 70, cx = 90, cy = 90;
  let cumulative = 0;

  const methodIcons: Record<string, typeof KeyRound> = {
    password: KeyRound, passkey: Fingerprint, webauthn: Shield,
    social: Globe, totp: Smartphone, saml: Users,
  };

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <StatCard icon={KeyRound} label={t("identityAnalytics.methods.totalAuths")} value={total} color="blue" />
        <StatCard icon={Fingerprint} label={t("identityAnalytics.methods.passwordlessRate")} value={`${passwordlessRate}%`} color="green" />
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">{t("identityAnalytics.methods.title")}</h3>
        <div className="flex flex-col md:flex-row items-center gap-8">
          {/* Pie */}
          <svg width="180" height="180" viewBox="0 0 180 180" className="flex-shrink-0">
            {methods.map((m: any, i: any) => {
              const pct = total > 0 ? m.count / total : 0;
              const startAngle = (cumulative / total) * 2 * Math.PI - Math.PI / 2;
              cumulative += m.count;
              const endAngle = (cumulative / total) * 2 * Math.PI - Math.PI / 2;
              const x1 = cx + radius * Math.cos(startAngle);
              const y1 = cy + radius * Math.sin(startAngle);
              const x2 = cx + radius * Math.cos(endAngle);
              const y2 = cy + radius * Math.sin(endAngle);
              const largeArc = pct > 0.5 ? 1 : 0;
              return (
                <path key={m.method}
                  d={`M ${cx} ${cy} L ${x1} ${y1} A ${radius} ${radius} 0 ${largeArc} 1 ${x2} ${y2} Z`}
                  fill={colors[i % colors.length]} opacity={0.85} />
              );
            })}
            <circle cx={cx} cy={cy} r={30} className="fill-white dark:fill-gray-900" />
            <text x={cx} y={cy - 4} textAnchor="middle" className="fill-gray-900 dark:fill-white text-sm font-bold">{total}</text>
            <text x={cx} y={cy + 12} textAnchor="middle" className="fill-gray-400 text-xs">total</text>
          </svg>

          {/* Legend + bars */}
          <div className="flex-1 space-y-2 w-full">
            {methods.map((m: any, i: any) => {
              const pct = total > 0 ? Math.round((m.count / total) * 100) : 0;
              const Icon = methodIcons[m.method] || KeyRound;
              return (
                <div key={m.method} className="flex items-center gap-3">
                  <div className="w-3 h-3 rounded-full flex-shrink-0" style={{ backgroundColor: colors[i % colors.length] }} />
                  <Icon className="w-4 h-4 text-gray-400" />
                  <span className="text-sm text-gray-700 dark:text-gray-300 flex-1">
                    {t(`identityAnalytics.methods.${m.method}`)}
                  </span>
                  <div className="w-24 h-1.5 bg-gray-200 dark:bg-gray-800 rounded-full overflow-hidden">
                    <div className="h-full rounded-full" style={{ width: `${pct}%`, backgroundColor: colors[i % colors.length] }} />
                  </div>
                  <span className="text-xs font-medium text-gray-900 dark:text-white w-12 text-right">{m.count}</span>
                  <span className="text-xs text-gray-400 w-8 text-right">{pct}%</span>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}

// ============ Risk Tab ============

function RiskTab() {
  const t = useTranslations();
  const [users, setUsers] = useState<RiskUser[]>([]);
  const [loading, setLoading] = useState(true);

  interface RiskUser {
    user_id: string; email: string; display_name: string;
    risk_score: number; risk_level: string; last_activity: string; factors: string[];
  }

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/audit/stats?type=risk_users`, { headers: { ...authHeader() } });
      if (res.ok) {
        const d = await res.json();
        if (d.users) { setUsers(d.users); return; }
      }
    } catch { /* mock */ }
    setUsers([
      { user_id: "1", email: "suspicious@unknown.com", display_name: "Unknown Actor", risk_score: 92, risk_level: "critical", last_activity: "2025-07-18T03:00:00Z", factors: ["impossible_travel", "new_device", "multiple_failed_logins"] },
      { user_id: "2", email: "temp@tempmail.com", display_name: "Temp User", risk_score: 85, risk_level: "high", last_activity: "2025-07-17T22:30:00Z", factors: ["disposable_email", "tor_exit_node"] },
      { user_id: "3", email: "admin@company.com", display_name: "Admin User", risk_score: 67, risk_level: "high", last_activity: "2025-07-18T08:00:00Z", factors: ["privileged_account", "off_hours_access"] },
      { user_id: "4", email: "user@company.com", display_name: "Regular User", risk_score: 45, risk_level: "medium", last_activity: "2025-07-16T14:00:00Z", factors: ["new_location"] },
    ]);
  }, []);

  useEffect(() => { load(); }, [load]);

  if (loading) return <Spinner />;

  const avgScore = users.length > 0 ? Math.round(users.reduce((s, u) => s + u.risk_score, 0) / users.length) : 0;
  const levelColors: Record<string, string> = {
    critical: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300",
    high: "bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300",
    medium: "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300",
    low: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300",
  };

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <StatCard icon={AlertTriangle} label={t("identityAnalytics.risk.totalHighRisk")} value={users.filter((u: any) => u.risk_level === "high" || u.risk_level === "critical").length} color="red" />
        <StatCard icon={Shield} label={t("identityAnalytics.risk.avgRiskScore")} value={avgScore} color="orange" />
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-1">{t("identityAnalytics.risk.title")}</h3>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-4">{t("identityAnalytics.risk.description")}</p>

        {users.length === 0 ? (
          <div className="text-center py-12">
            <Shield className="w-12 h-12 mx-auto mb-2 text-green-500" />
            <p className="text-sm text-gray-500">{t("identityAnalytics.risk.noRiskUsers")}</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 dark:border-gray-800 text-left">
                  <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("identityAnalytics.risk.user")}</th>
                  <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("identityAnalytics.risk.riskScore")}</th>
                  <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("identityAnalytics.risk.riskLevel")}</th>
                  <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("identityAnalytics.risk.factors")}</th>
                  <th className="py-2 px-3 font-medium text-gray-600 dark:text-gray-400">{t("identityAnalytics.risk.lastActivity")}</th>
                </tr>
              </thead>
              <tbody>
                {users.map((u: any) => (
                  <tr key={u.user_id} className="border-b border-gray-100 dark:border-gray-800/50">
                    <td className="py-3 px-3">
                      <div className="font-medium text-gray-900 dark:text-white">{u.display_name}</div>
                      <div className="text-xs text-gray-400">{u.email}</div>
                    </td>
                    <td className="py-3 px-3">
                      <div className="flex items-center gap-2">
                        <div className="w-16 h-1.5 bg-gray-200 dark:bg-gray-800 rounded-full overflow-hidden">
                          <div className={`h-full rounded-full ${u.risk_score >= 80 ? "bg-red-500" : u.risk_score >= 50 ? "bg-orange-500" : "bg-yellow-500"}`} style={{ width: `${u.risk_score}%` }} />
                        </div>
                        <span className="text-xs font-medium text-gray-900 dark:text-white">{u.risk_score}</span>
                      </div>
                    </td>
                    <td className="py-3 px-3">
                      <span className={`px-2 py-0.5 text-xs rounded-full ${levelColors[u.risk_level] || levelColors.low}`}>
                        {t(`identityAnalytics.risk.level${u.risk_level.replace(/^./, (m) => m.toUpperCase())}`)}
                      </span>
                    </td>
                    <td className="py-3 px-3">
                      <div className="flex flex-wrap gap-1">
                        {u.factors.map((f: any) => (
                          <span key={f} className="px-1.5 py-0.5 text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded">{f.replace(/_/g, " ")}</span>
                        ))}
                      </div>
                    </td>
                    <td className="py-3 px-3 text-xs text-gray-500">
                      {new Date(u.last_activity).toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

// ============ Shared ============

function Spinner() { return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>; }

function StatCard({ icon: Icon, label, value, color }: { icon: typeof Users; label: string; value: string | number; color: string }) {
  const colors: Record<string, string> = {
    blue: "text-blue-600", green: "text-green-600", orange: "text-orange-500", red: "text-red-500",
  };
  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
      <div className="flex items-center gap-2 mb-2">
        <Icon className={`w-5 h-5 ${colors[color]}`} />
        <span className="text-xs text-gray-500 dark:text-gray-400">{label}</span>
      </div>
      <div className="text-2xl font-bold text-gray-900 dark:text-white">{value}</div>
    </div>
  );
}

function Legend({ color, label }: { color: string; label: string }) {
  return (
    <div className="flex items-center gap-1.5">
      <div className={`w-3 h-3 rounded-full ${color}`} />
      <span className="text-xs text-gray-600 dark:text-gray-400">{label}</span>
    </div>
  );
}
