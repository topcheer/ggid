"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Activity, PieChart as PieIcon, ShieldCheck, TrendingUp,
  Loader2, Clock, Zap, Users, AlertTriangle, CheckCircle2,
  KeyRound, Fingerprint, UserX, CalendarClock, RefreshCw,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
type TabId = "mttd" | "coverage" | "hygiene" | "incidents";

export default function IAMMetricsPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("mttd");

  const tabs: { id: TabId; label: string; icon: typeof Activity }[] = [
    { id: "mttd", label: t("iamMetrics.tabs.mttd"), icon: Clock },
    { id: "coverage", label: t("iamMetrics.tabs.coverage"), icon: PieIcon },
    { id: "hygiene", label: t("iamMetrics.tabs.hygiene"), icon: ShieldCheck },
    { id: "incidents", label: t("iamMetrics.tabs.incidents"), icon: TrendingUp },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Activity className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("iamMetrics.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("iamMetrics.description")}</p>
        </div>

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1 overflow-x-auto">
          {tabs.map(({ id, label, icon: Icon }) => (
            <button key={id} onClick={() => setTab(id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors whitespace-nowrap ${
                tab === id ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm" : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
              }`}>
              <Icon className="w-4 h-4" />{label}
            </button>
          ))}
        </div>

        {tab === "mttd" && <MTTDTab />}
        {tab === "coverage" && <CoverageTab />}
        {tab === "hygiene" && <HygieneTab />}
        {tab === "incidents" && <IncidentsTab />}
      </div>
    </div>
  );
}

// ============ MTTD/MTTR Tab ============

function MTTDTab() {
  const t = useTranslations();
  const [data, setData] = useState<{ day: string; mttd: number; mttr: number }[]>([]);
  const [loading, setLoading] = useState(true);
  const [range, setRange] = useState<"7d" | "30d">("30d");

  useEffect(() => {
    const days = range === "7d" ? 7 : 30;
    setData(Array.from({ length: days }, (_, i) => {
      const date = new Date(); date.setDate(date.getDate() - (days - 1 - i));
      return {
        day: date.toISOString().split("T")[0],
        mttd: Math.round((8 + Math.random() * 6 - i * 0.1) * 10) / 10,
        mttr: Math.round((25 + Math.random() * 15 - i * 0.2) * 10) / 10,
      };
    }));
    setLoading(false);
  }, [range]);

  if (loading) return <Spinner />;

  const latest = data[data.length - 1];
  const prev = data[0];
  const mttdChange = prev.mttd > 0 ? Math.round(((latest.mttd - prev.mttd) / prev.mttd) * 100) : 0;
  const mttrChange = prev.mttr > 0 ? Math.round(((latest.mttr - prev.mttr) / prev.mttr) * 100) : 0;

  // SVG dual-line chart
  const maxVal = Math.max(...data.map((d: any) => Math.max(d.mttd, d.mttr)), 1);
  const chartW = 700, chartH = 200, pad = 35;
  const xStep = data.length > 1 ? (chartW - pad * 2) / (data.length - 1) : 0;
  const yScale = (v: number) => chartH - pad - (v / maxVal) * (chartH - pad * 2);
  const linePath = (key: "mttd" | "mttr") =>
    data.map((d: any, i: any) => `${i === 0 ? "M" : "L"} ${pad + i * xStep} ${yScale(d[key])}`).join(" ");

  return (
    <div className="space-y-4">
      <div className="flex gap-2">
        {(["7d", "30d"] as const).map((r: any) => (
          <button key={r} onClick={() => setRange(r)}
            className={`px-3 py-1.5 rounded-lg text-xs font-medium ${range === r ? "bg-blue-600 text-white" : "bg-white dark:bg-gray-800 text-gray-600 dark:text-gray-400 border border-gray-200 dark:border-gray-700"}`}>
            {r === "7d" ? t("iamMetrics.mttd.trend7d") : t("iamMetrics.mttd.trend30d")}
          </button>
        ))}
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
          <div className="flex items-center gap-2 mb-2">
            <Zap className="w-5 h-5 text-blue-600" />
            <span className="text-xs text-gray-500">{t("iamMetrics.mttd.currentMTTD")}</span>
          </div>
          <div className="text-3xl font-bold text-gray-900 dark:text-white">{latest.mttd} <span className="text-sm text-gray-400">{t("iamMetrics.mttd.minutes")}</span></div>
          <div className={`text-xs mt-1 ${mttdChange < 0 ? "text-green-600" : "text-red-500"}`}>
            {mttdChange < 0 ? "↓" : "↑"} {Math.abs(mttdChange)}% {t("iamMetrics.mttd.improvement")}
          </div>
        </div>
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
          <div className="flex items-center gap-2 mb-2">
            <Clock className="w-5 h-5 text-orange-500" />
            <span className="text-xs text-gray-500">{t("iamMetrics.mttd.currentMTTR")}</span>
          </div>
          <div className="text-3xl font-bold text-gray-900 dark:text-white">{latest.mttr} <span className="text-sm text-gray-400">{t("iamMetrics.mttd.minutes")}</span></div>
          <div className={`text-xs mt-1 ${mttrChange < 0 ? "text-green-600" : "text-red-500"}`}>
            {mttrChange < 0 ? "↓" : "↑"} {Math.abs(mttrChange)}% {t("iamMetrics.mttd.improvement")}
          </div>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">{t("iamMetrics.mttd.title")}</h3>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">{t("iamMetrics.mttd.description")}</p>
        <svg viewBox={`0 0 ${chartW} ${chartH}`} className="w-full h-48">
          {[0, 0.25, 0.5, 0.75, 1].map((p: any) => (
            <line key={p} x1={pad} y1={pad + p * (chartH - pad * 2)} x2={chartW - pad} y2={pad + p * (chartH - pad * 2)}
              stroke="currentColor" className="text-gray-100 dark:text-gray-800" strokeWidth={1} />
          ))}
          <path d={linePath("mttd")} fill="none" stroke="#3b82f6" strokeWidth={2} />
          <path d={linePath("mttr")} fill="none" stroke="#f97316" strokeWidth={2} />
        </svg>
        <div className="flex items-center justify-center gap-4 mt-2">
          <Legend color="bg-blue-500" label={t("iamMetrics.mttd.mttd")} />
          <Legend color="bg-orange-500" label={t("iamMetrics.mttd.mttr")} />
        </div>
      </div>
    </div>
  );
}

// ============ Coverage Tab ============

function CoverageTab() {
  const t = useTranslations();
  const [coverage, setCoverage] = useState<{ label: string; pct: number; total: number; covered: number; color: string; icon: typeof ShieldCheck }[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setCoverage([
      { label: t("iamMetrics.coverage.mfaCoverage"), pct: 88, total: 500, covered: 440, color: "#22c55e", icon: ShieldCheck },
      { label: t("iamMetrics.coverage.passkeyCoverage"), pct: 42, total: 500, covered: 210, color: "#3b82f6", icon: KeyRound },
      { label: t("iamMetrics.coverage.itdrCoverage"), pct: 95, total: 500, covered: 475, color: "#8b5cf6", icon: Fingerprint },
      { label: t("iamMetrics.coverage.caeCoverage"), pct: 76, total: 500, covered: 380, color: "#f59e0b", icon: Activity },
    ]);
    setLoading(false);
  }, [t]);

  if (loading) return <Spinner />;

  const radius = 70, cx = 90, cy = 90;

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {coverage.map((c: any) => {
          const Icon = c.icon;
          return (
            <div key={c.label} className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
              <div className="flex items-center gap-2 mb-2"><Icon className="w-5 h-5" style={{ color: c.color }} /><span className="text-xs text-gray-500">{c.label}</span></div>
              <div className="text-3xl font-bold text-gray-900 dark:text-white">{c.pct}%</div>
              <div className="text-xs text-gray-400 mt-0.5">{c.covered}/{c.total} {t("iamMetrics.coverage.protectedUsers")}</div>
            </div>
          );
        })}
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">{t("iamMetrics.coverage.title")}</h3>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-4">{t("iamMetrics.coverage.description")}</p>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {coverage.map((c: any) => {
            const covered = (c.pct / 100) * 2 * Math.PI * radius;
            const circumference = 2 * Math.PI * radius;
            return (
              <div key={c.label} className="flex flex-col items-center">
                <svg width="120" height="120" viewBox="0 0 180 180" className="-rotate-90">
                  <circle cx={cx} cy={cy} r={radius} fill="none" stroke="currentColor" className="text-gray-100 dark:text-gray-800" strokeWidth="14" />
                  <circle cx={cx} cy={cy} r={radius} fill="none" stroke={c.color} strokeWidth="14" strokeLinecap="round"
                    strokeDasharray={`${covered} ${circumference}`} />
                </svg>
                <div className="text-center -mt-[88px] mb-[40px] relative pointer-events-none">
                  <div className="text-2xl font-bold text-gray-900 dark:text-white">{c.pct}%</div>
                </div>
                <span className="text-xs text-gray-600 dark:text-gray-400 mt-2 text-center">{c.label}</span>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

// ============ Hygiene Tab ============

function HygieneTab() {
  const t = useTranslations();
  const [issues, setIssues] = useState<{ key: string; count: number; icon: typeof UserX; desc: string; color: string }[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setIssues([
      { key: "orphanAccounts", count: 12, icon: UserX, desc: t("iamMetrics.hygiene.orphanAccountsDesc"), color: "red" },
      { key: "expiredPermissions", count: 34, icon: CalendarClock, desc: t("iamMetrics.hygiene.expiredPermissionsDesc"), color: "orange" },
      { key: "pendingAccessReviews", count: 5, icon: AlertTriangle, desc: t("iamMetrics.hygiene.pendingAccessReviewsDesc"), color: "yellow" },
      { key: "unrotatedServiceAccounts", count: 8, icon: KeyRound, desc: t("iamMetrics.hygiene.unrotatedServiceAccountsDesc"), color: "red" },
    ]);
    setLoading(false);
  }, [t]);

  if (loading) return <Spinner />;

  const total = issues.reduce((s, i) => s + i.count, 0);
  const healthScore = Math.max(0, 100 - total * 2);

  const colorMap: Record<string, { bg: string; text: string; icon: string }> = {
    red: { bg: "bg-red-50 dark:bg-red-950/30", text: "text-red-700 dark:text-red-300", icon: "text-red-500" },
    orange: { bg: "bg-orange-50 dark:bg-orange-950/30", text: "text-orange-700 dark:text-orange-300", icon: "text-orange-500" },
    yellow: { bg: "bg-yellow-50 dark:bg-yellow-950/30", text: "text-yellow-700 dark:text-yellow-300", icon: "text-yellow-500" },
  };

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
          <div className="flex items-center gap-2 mb-2"><AlertTriangle className="w-5 h-5 text-red-500" /><span className="text-xs text-gray-500">{t("iamMetrics.hygiene.totalIssues")}</span></div>
          <div className="text-3xl font-bold text-gray-900 dark:text-white">{total}</div>
        </div>
        <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
          <div className="flex items-center gap-2 mb-2"><ShieldCheck className="w-5 h-5 text-blue-600" /><span className="text-xs text-gray-500">{t("iamMetrics.hygiene.healthScore")}</span></div>
          <div className="text-3xl font-bold text-gray-900 dark:text-white">{healthScore}<span className="text-sm text-gray-400">/100</span></div>
          <div className="mt-2 h-1.5 bg-gray-200 dark:bg-gray-800 rounded-full overflow-hidden">
            <div className={`h-full rounded-full ${healthScore >= 80 ? "bg-green-500" : healthScore >= 60 ? "bg-yellow-500" : "bg-red-500"}`} style={{ width: `${healthScore}%` }} />
          </div>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">{t("iamMetrics.hygiene.title")}</h3>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-4">{t("iamMetrics.hygiene.description")}</p>
        <div className="space-y-3">
          {issues.map((issue: any) => {
            const Icon = issue.icon;
            const colors = colorMap[issue.color];
            return (
              <div key={issue.key} className={`flex items-center gap-3 p-3 rounded-lg ${colors.bg}`}>
                <Icon className={`w-5 h-5 ${colors.icon}`} />
                <div className="flex-1">
                  <span className="text-sm font-medium text-gray-900 dark:text-white">{t(`iamMetrics.hygiene.${issue.key}`)}</span>
                  <p className="text-xs text-gray-500 dark:text-gray-400">{issue.desc}</p>
                </div>
                <span className={`text-2xl font-bold ${colors.text}`}>{issue.count}</span>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}

// ============ Incidents Tab ============

function IncidentsTab() {
  const t = useTranslations();
  const [data, setData] = useState<{ month: string; itdr: number; soar: number; manual: number }[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setData([
      { month: "Jan", itdr: 12, soar: 10, manual: 3 },
      { month: "Feb", itdr: 15, soar: 13, manual: 5 },
      { month: "Mar", itdr: 8, soar: 7, manual: 2 },
      { month: "Apr", itdr: 20, soar: 18, manual: 4 },
      { month: "May", itdr: 25, soar: 23, manual: 3 },
      { month: "Jun", itdr: 18, soar: 16, manual: 2 },
      { month: "Jul", itdr: 14, soar: 13, manual: 1 },
    ]);
    setLoading(false);
  }, []);

  if (loading) return <Spinner />;

  const totalITDR = data.reduce((s, d) => s + d.itdr, 0);
  const totalSOAR = data.reduce((s, d) => s + d.soar, 0);
  const totalManual = data.reduce((s, d) => s + d.manual, 0);
  const total = totalITDR + totalSOAR + totalManual;

  // SVG stacked bar chart
  const maxVal = Math.max(...data.map((d: any) => d.itdr + d.soar + d.manual), 1);
  const barW = data.length > 0 ? (700 - 60) / data.length : 0;
  const colors = { itdr: "#8b5cf6", soar: "#3b82f6", manual: "#ef4444" };

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-3 gap-4">
        <StatCard icon={Fingerprint} label={t("iamMetrics.incidents.itdrDetected")} value={totalITDR} color="text-purple-600" />
        <StatCard icon={Zap} label={t("iamMetrics.incidents.soarResponded")} value={totalSOAR} color="text-blue-600" />
        <StatCard icon={Users} label={t("iamMetrics.incidents.manualIntervention")} value={totalManual} color="text-red-500" />
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">{t("iamMetrics.incidents.title")}</h3>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-4">{t("iamMetrics.incidents.description")}</p>

        <svg viewBox="0 0 700 240" className="w-full h-56">
          {data.map((d: any, i: any) => {
            const total_h = ((d.itdr + d.soar + d.manual) / maxVal) * 180;
            const itdrH = (d.itdr / (d.itdr + d.soar + d.manual)) * total_h;
            const soarH = (d.soar / (d.itdr + d.soar + d.manual)) * total_h;
            const manualH = total_h - itdrH - soarH;
            const x = 30 + i * barW;
            return (
              <g key={i}>
                <rect x={x + 2} y={210 - total_h} width={barW - 4} height={itdrH} fill={colors.itdr} rx={2} />
                <rect x={x + 2} y={210 - total_h + itdrH} width={barW - 4} height={soarH} fill={colors.soar} />
                <rect x={x + 2} y={210 - manualH} width={barW - 4} height={manualH} fill={colors.manual} rx={2} />
                <text x={x + barW / 2} y={225} textAnchor="middle" className="fill-gray-400 text-xs">{d.month}</text>
              </g>
            );
          })}
          <line x1="30" y1="210" x2="690" y2="210" stroke="currentColor" className="text-gray-200 dark:text-gray-700" />
        </svg>
        <div className="flex items-center justify-center gap-4 mt-2">
          <Legend color="bg-purple-500" label={t("iamMetrics.incidents.itdrDetected")} />
          <Legend color="bg-blue-500" label={t("iamMetrics.incidents.soarResponded")} />
          <Legend color="bg-red-500" label={t("iamMetrics.incidents.manualIntervention")} />
        </div>

        <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-800 flex items-center justify-between text-sm">
          <span className="text-gray-500">{t("iamMetrics.incidents.total")}: <strong className="text-gray-900 dark:text-white">{total}</strong></span>
          <span className="text-gray-500">{t("iamMetrics.incidents.resolved")}: <strong className="text-green-600">{totalITDR + totalSOAR}</strong></span>
        </div>
      </div>
    </div>
  );
}

// ============ Shared ============

function Spinner() { return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>; }

function StatCard({ icon: Icon, label, value, color }: { icon: typeof Activity; label: string; value: number; color: string }) {
  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
      <div className="flex items-center gap-2 mb-2"><Icon className={`w-5 h-5 ${color}`} /><span className="text-xs text-gray-500">{label}</span></div>
      <div className="text-2xl font-bold text-gray-900 dark:text-white">{value}</div>
    </div>
  );
}

function Legend({ color, label }: { color: string; label: string }) {
  return (
    <div className="flex items-center gap-1.5">
      <div className={`w-3 h-3 rounded ${color}`} />
      <span className="text-xs text-gray-600 dark:text-gray-400">{label}</span>
    </div>
  );
}
