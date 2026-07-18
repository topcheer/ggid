"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Shield, Loader2, AlertCircle, X, RefreshCw, ChevronRight,
  AlertTriangle, Activity, CheckCircle2, XCircle, Zap,
  Rocket, Crosshair, Search, TrendingUp, Clock,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

export default function SecurityOverviewPage() {
  const t = useTranslations();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try { await Promise.all([fetch("/api/v1/audit/itdr/stats", { headers: h }).catch(() => null), fetch("/api/v1/auth/risk/aggregate?group_by=user", { headers: h }).catch(() => null), fetch("/api/v1/audit/compliance/dashboard", { headers: h }).catch(() => null)]); setError(null); }
    catch { setError(t("secOverview.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  // Demo composite scores
  const subScores = [
    { label: t("secOverview.itdrCoverage"), score: 88, color: "#22c55e" },
    { label: t("secOverview.riskAvg"), score: 34, color: "#eab308" },
    { label: t("secOverview.dlpCompliance"), score: 94, color: "#22c55e" },
    { label: t("secOverview.postureCompliance"), score: 76, color: "#eab308" },
    { label: t("secOverview.mitreCoverage"), score: 82, color: "#22c55e" },
  ];
  const overallScore = Math.round(subScores.reduce((a: any, s: any) => a + s.score, 0) / subScores.length);

  const activeThreats = [{ sev: "critical", count: 2 }, { sev: "high", count: 5 }, { sev: "medium", count: 11 }, { sev: "low", count: 23 }];
  const riskDist = [{ label: "Low", value: 287, color: "#22c55e" }, { label: "Medium", value: 42, color: "#eab308" }, { label: "High", value: 12, color: "#f97316" }, { label: "Critical", value: 3, color: "#ef4444" }];
  const incidents = [
    { id: "INC-001", title: "MFA Fatigue Attack", severity: "critical", status: "investigating", responder: "soc-team", time: "2h ago" },
    { id: "INC-002", title: "Impossible Travel — user:bob", severity: "high", status: "contained", responder: "auto-response", time: "5h ago" },
    { id: "INC-003", title: "Credential Stuffing Burst", severity: "high", status: "resolved", responder: "soc-team", time: "8h ago" },
    { id: "INC-004", title: "Anomalous API Usage", severity: "medium", status: "investigating", responder: "ops", time: "1d ago" },
    { id: "INC-005", title: "Token Theft Detected", severity: "critical", status: "contained", responder: "auto-response", time: "1d ago" },
  ];
  const compliance = [{ name: "SOC2", pct: 91 }, { name: "ISO27001", pct: 85 }, { name: "NIS2", pct: 87 }, { name: "NIST", pct: 82 }];
  const totalRisk = riskDist.reduce((a: any, r: any) => a + r.value, 0) || 1;

  const sevCfg: Record<string, string> = { critical: "text-red-600 bg-red-100 dark:bg-red-900/30", high: "text-orange-600 bg-orange-100 dark:bg-orange-900/30", medium: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30", low: "text-blue-600 bg-blue-100 dark:bg-blue-900/30" };
  const quickActions = [
    { label: t("secOverview.soaPlaybooks"), icon: Rocket, href: "/security/soar", color: "text-pink-500" },
    { label: t("secOverview.mitreMatrix"), icon: Crosshair, href: "/security/itdr-mitre", color: "text-orange-500" },
    { label: t("secOverview.dlpScanner"), icon: Search, href: "/security/dlp-egress", color: "text-red-500" },
    { label: t("secOverview.riskEngine"), icon: Zap, href: "/security/risk-engine", color: "text-yellow-500" },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Shield className="h-6 w-6 text-indigo-500" /> {t("secOverview.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("secOverview.subtitle")}</p></div>
        <button onClick={loadData} aria-label="Refresh" className="rounded-lg border border-gray-300 p-2 dark:border-gray-700"><RefreshCw className="h-4 w-4" /></button>
      </div>

      {error && (<div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>)}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (
        <div className="space-y-6">
          {/* Row 1: Score gauge + Active threats */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
            {/* Score gauge */}
            <div className={`${card} flex items-center gap-6`}>
              <svg width={100} height={100} viewBox="0 0 100 100">
                <circle cx="50" cy="50" r="42" fill="none" stroke="currentColor" strokeWidth="8" className="text-gray-200 dark:text-gray-700" />
                <circle cx="50" cy="50" r="42" fill="none" stroke={overallScore >= 80 ? "#22c55e" : overallScore >= 60 ? "#eab308" : "#ef4444"} strokeWidth="8" strokeLinecap="round" strokeDasharray={`${(overallScore / 100) * 263.9} 263.9`} transform="rotate(-90 50 50)" />
                <text x="50" y="50" textAnchor="middle" dominantBaseline="central" className="fill-gray-900 dark:fill-white text-2xl font-bold">{overallScore}</text>
              </svg>
              <div className="flex-1 space-y-1">
                <p className="text-xs font-semibold uppercase text-gray-400">{t("secOverview.securityScore")}</p>
                {subScores.map(s => (
                  <div key={s.label} className="flex items-center gap-2"><span className="text-xs text-gray-400 w-28">{s.label}</span><div className="flex-1 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-700"><div className="h-full rounded-full" style={{ width: `${s.score}%`, backgroundColor: s.color }} /></div><span className="text-xs font-mono w-8">{s.score}</span></div>
                ))}
              </div>
            </div>

            {/* Active threats */}
            <div className={card}>
              <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><AlertTriangle className="h-4 w-4 text-red-500" /> {t("secOverview.activeThreats")}</h3>
              <div className="grid grid-cols-4 gap-2 text-center">
                {activeThreats.map(th => (
                  <div key={th.sev}><p className={`text-2xl font-bold ${sevCfg[th.sev]?.split(" ")[0]}`}>{th.count}</p><p className="text-xs text-gray-400 capitalize">{th.sev}</p></div>
                ))}
              </div>
              <div className="mt-4 flex items-center justify-between rounded-lg bg-red-50 dark:bg-red-900/20 p-3"><div className="flex items-center gap-2"><Activity className="h-4 w-4 text-red-500" /><span className="text-sm font-medium">{activeThreats.reduce((a: any, t: any) => a + t.count, 0)} {t("secOverview.totalThreats24h")}</span></div></div>
            </div>

            {/* Quick actions */}
            <div className={card}>
              <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("secOverview.quickActions")}</h3>
              <div className="grid grid-cols-2 gap-2">
                {quickActions.map(a => { const Icon = a.icon; return (
                  <a key={a.href} href={a.href} className="flex flex-col items-center gap-2 rounded-lg border p-3 transition hover:shadow-md dark:border-gray-700">
                    <Icon className={`h-5 w-5 ${a.color}`} /><span className="text-xs font-medium text-center">{a.label}</span>
                  </a>
                );})}
              </div>
            </div>
          </div>

          {/* Row 2: Risk distribution + Incidents */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            {/* Risk pie */}
            <div className={card}>
              <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><TrendingUp className="h-4 w-4" /> {t("secOverview.riskDistribution")}</h3>
              <div className="flex items-center gap-6">
                <svg width={120} height={120} viewBox="0 0 120 120">
                  {(() => { let offset = 0; return riskDist.map(r => { const frac = r.value / totalRisk; const dash = frac * 263.9; const el = <circle key={r.label} cx="60" cy="60" r="42" fill="none" stroke={r.color} strokeWidth="16" strokeDasharray={`${dash} ${263.9 - dash}`} strokeDashoffset={-offset} transform="rotate(-90 60 60)" />; offset += dash; return el; }); })()}
                  <text x="60" y="56" textAnchor="middle" className="fill-gray-900 dark:fill-white text-xl font-bold">{totalRisk}</text>
                  <text x="60" y="74" textAnchor="middle" className="fill-gray-400 text-xs">{t("secOverview.users")}</text>
                </svg>
                <div className="flex-1 space-y-2">
                  {riskDist.map(r => (
                    <div key={r.label} className="flex items-center gap-2"><span className="h-3 w-3 rounded-full" style={{ backgroundColor: r.color }} /><span className="text-sm flex-1">{r.label}</span><span className="text-sm font-mono">{r.value}</span><span className="text-xs text-gray-400 w-10">{Math.round(r.value / totalRisk * 100)}%</span></div>
                  ))}
                </div>
              </div>
            </div>

            {/* Incidents */}
            <div className={card}>
              <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><AlertTriangle className="h-4 w-4" /> {t("secOverview.recentIncidents")}</h3>
              <div className="space-y-2">
                {incidents.map(inc => (
                  <div key={inc.id} className="flex items-center justify-between rounded-lg border p-2 dark:border-gray-700">
                    <div className="flex items-center gap-3 min-w-0">
                      <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${sevCfg[inc.severity]}`}>{inc.severity}</span>
                      <div className="min-w-0"><span className="text-xs font-medium truncate block">{inc.title}</span><p className="text-xs text-gray-400">{inc.responder} · {inc.time}</p></div>
                    </div>
                    <span className={`px-1.5 py-0.5 rounded text-xs ${inc.status === "resolved" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : inc.status === "contained" ? "bg-blue-100 dark:bg-blue-900/30 text-blue-600" : "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600"}`}>{inc.status}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Row 3: Compliance badges */}
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><CheckCircle2 className="h-4 w-4" /> {t("secOverview.complianceStatus")}</h3>
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              {compliance.map(c => (
                <div key={c.name} className="flex items-center gap-3 rounded-lg border p-3 dark:border-gray-700">
                  <div className="relative h-10 w-10"><svg width={40} height={40} viewBox="0 0 40 40"><circle cx="20" cy="20" r="16" fill="none" stroke="currentColor" strokeWidth="3" className="text-gray-200 dark:text-gray-700" /><circle cx="20" cy="20" r="16" fill="none" stroke={c.pct >= 90 ? "#22c55e" : "#eab308"} strokeWidth="3" strokeLinecap="round" strokeDasharray={`${(c.pct / 100) * 100.5} 100.5`} transform="rotate(-90 20 20)" /><text x="20" y="23" textAnchor="middle" className="fill-gray-900 dark:fill-white text-[9px] font-bold">{c.pct}%</text></svg></div>
                  <div><p className="text-sm font-semibold">{c.name}</p></div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
