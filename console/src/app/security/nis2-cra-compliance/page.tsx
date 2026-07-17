"use client";
import { useState, useCallback, useEffect } from "react";
import {
  ShieldCheck, Loader2, AlertCircle, X, RefreshCw, Check,
  TrendingUp, FileText, AlertTriangle, ChevronRight, Clock,
  CheckCircle2, XCircle, Hash, Download, Ban, Cpu, Zap, Activity,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface FrameworkStat { framework: string; controls_total: number; covered: number; partial: number; gap: number; coverage_pct: number; }
interface ComplianceGap { id: string; framework: string; control_id: string; gap_description: string; remediation_plan: string; owner: string; due_date?: string; status: "open" | "in_progress" | "resolved"; }
interface RemediationItem { id: string; control: string; title: string; status: string; assigned_to: string; resolved_at?: string; started_at?: string; est_completion?: string; resolution_days?: number; }
interface EvidenceRecord { id: string; framework: string; control_id: string; status: string; artifacts: string[]; collected_by: string; collected_at: string; }

type Tab = "dashboard" | "gaps" | "remediation" | "evidence" | "reports";

const STATUS_CFG: Record<string, { label: string; color: string; bg: string; icon: typeof CheckCircle2 }> = {
  resolved: { label: "Resolved", color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30", icon: CheckCircle2 },
  in_progress: { label: "In Progress", color: "text-blue-600", bg: "bg-blue-100 dark:bg-blue-900/30", icon: Activity },
  open: { label: "Open", color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30", icon: XCircle },
  compliant: { label: "Compliant", color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30", icon: CheckCircle2 },
  non_compliant: { label: "Non-Compliant", color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30", icon: XCircle },
  in_progress_evidence: { label: "Collecting", color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30", icon: Clock },
};

export default function NIS2CRACompliancePage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("dashboard");
  const [dashboard, setDashboard] = useState<{ frameworks: FrameworkStat[]; overall_coverage: number; total_controls: number; total_covered: number } | null>(null);
  const [gaps, setGaps] = useState<ComplianceGap[]>([]);
  const [remediation, setRemediation] = useState<RemediationItem[]>([]);
  const [evidence, setEvidence] = useState<EvidenceRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Report generator
  const [rFramework, setRFramework] = useState("soc2");
  const [rFrom, setRFrom] = useState("");
  const [rTo, setRTo] = useState("");
  const [generating, setGenerating] = useState(false);
  const [reportResult, setReportResult] = useState<{ regulation: string; findings: { id: string; severity: string; description: string }[] } | null>(null);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [dRes, gRes, rRes, eRes] = await Promise.all([
        fetch("/api/v1/audit/compliance/dashboard", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/compliance/gaps", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/compliance/remediation-progress?framework=soc2", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/compliance/evidence", { headers: h }).catch(() => null),
      ]);
      if (dRes?.ok) setDashboard(await dRes.json());
      if (gRes?.ok) { const d = await gRes.json(); setGaps(d.gaps || []); }
      if (rRes?.ok) { const d = await rRes.json(); setRemediation(d.gaps || d.items || []); }
      if (eRes?.ok) { const d = await eRes.json(); setEvidence(d.evidence || d.records || []); }
      setError(null);
    } catch { setError(t("compliance.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const generateReport = async () => {
    setGenerating(true); setReportResult(null);
    try {
      const res = await fetch("/api/v1/audit/regulatory/report", {
        method: "POST", headers: H,
        body: JSON.stringify({ regulation: rFramework.toUpperCase(), period_from: rFrom, period_to: rTo }),
      }).catch(() => null);
      if (res?.ok) setReportResult(await res.json());
    } catch { setError(t("compliance.reportError")); }
    finally { setGenerating(false); }
  };

  const openGaps = gaps.filter(g => g.status === "open");
  const inProgressGaps = gaps.filter(g => g.status === "in_progress");
  const resolvedGaps = gaps.filter(g => g.status === "resolved");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <ShieldCheck className="h-6 w-6 text-emerald-500" /> {t("compliance.title")}
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("compliance.subtitle")}</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "dashboard" as Tab, label: t("compliance.dashboard"), icon: Activity },
          { id: "gaps" as Tab, label: `${t("compliance.gaps")} (${openGaps.length})`, icon: AlertTriangle },
          { id: "remediation" as Tab, label: t("compliance.remediation"), icon: CheckCircle2 },
          { id: "evidence" as Tab, label: t("compliance.evidence"), icon: FileText },
          { id: "reports" as Tab, label: t("compliance.reports"), icon: Download },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-emerald-600 text-emerald-600 dark:text-emerald-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-emerald-500" /></div> : (<>

      {/* ════ DASHBOARD ════ */}
      {tab === "dashboard" && (
        <div className="space-y-6">
          {/* Overall score */}
          <div className={card}>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs text-gray-400">{t("compliance.overallCoverage")}</p>
                <p className="mt-1 text-4xl font-bold text-emerald-600">{dashboard?.overall_coverage?.toFixed(1) ?? 0}%</p>
                <p className="mt-1 text-xs text-gray-400">{dashboard?.total_covered ?? 0} / {dashboard?.total_controls ?? 0} {t("compliance.controlsCovered")}</p>
              </div>
              <div className="relative h-24 w-24">
                <svg viewBox="0 0 100 100" className="-rotate-90">
                  <circle cx="50" cy="50" r="40" fill="none" stroke="currentColor" strokeWidth="10" className="text-gray-200 dark:text-gray-700" />
                  <circle cx="50" cy="50" r="40" fill="none" stroke="#10b981" strokeWidth="10" strokeLinecap="round"
                    strokeDasharray={`${(dashboard?.overall_coverage ?? 0) * 2.51} ${251.2}`} />
                </svg>
                <div className="absolute inset-0 flex items-center justify-center"><span className="text-xl font-bold">{dashboard?.overall_coverage?.toFixed(0) ?? 0}%</span></div>
              </div>
            </div>
          </div>

          {/* Framework cards */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {dashboard?.frameworks?.map(f => (
              <div key={f.framework} className={card}>
                <div className="flex items-center justify-between mb-2">
                  <h3 className="font-semibold text-sm uppercase">{f.framework}</h3>
                  <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${f.coverage_pct >= 80 ? "bg-green-100 dark:bg-green-900/30 text-green-600" : f.coverage_pct >= 60 ? "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600" : "bg-red-100 dark:bg-red-900/30 text-red-600"}`}>{f.coverage_pct.toFixed(0)}%</span>
                </div>
                <div className="h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700 mb-2">
                  <div className={`h-full rounded-full ${f.coverage_pct >= 80 ? "bg-green-500" : f.coverage_pct >= 60 ? "bg-yellow-500" : "bg-red-500"}`} style={{ width: `${f.coverage_pct}%` }} />
                </div>
                <div className="grid grid-cols-3 gap-1 text-center text-xs">
                  <div><p className="font-bold text-green-600">{f.covered}</p><p className="text-gray-400">{t("compliance.covered")}</p></div>
                  <div><p className="font-bold text-yellow-600">{f.partial}</p><p className="text-gray-400">{t("compliance.partial")}</p></div>
                  <div><p className="font-bold text-red-600">{f.gap}</p><p className="text-gray-400">{t("compliance.gapsShort")}</p></div>
                </div>
              </div>
            ))}
          </div>

          {/* Quick stats */}
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={card + " text-center"}><AlertTriangle className="mx-auto h-5 w-5 text-red-400" /><p className="mt-2 text-2xl font-bold">{openGaps.length}</p><p className="text-xs text-gray-400">{t("compliance.openGaps")}</p></div>
            <div className={card + " text-center"}><Activity className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-2 text-2xl font-bold">{inProgressGaps.length}</p><p className="text-xs text-gray-400">{t("compliance.inProgress")}</p></div>
            <div className={card + " text-center"}><CheckCircle2 className="mx-auto h-5 w-5 text-green-400" /><p className="mt-2 text-2xl font-bold">{resolvedGaps.length}</p><p className="text-xs text-gray-400">{t("compliance.resolved")}</p></div>
            <div className={card + " text-center"}><FileText className="mx-auto h-5 w-5 text-purple-400" /><p className="mt-2 text-2xl font-bold">{evidence.length}</p><p className="text-xs text-gray-400">{t("compliance.evidenceCollected")}</p></div>
          </div>
        </div>
      )}

      {/* ════ GAPS ════ */}
      {tab === "gaps" && (
        <div className="space-y-4">
          {gaps.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><CheckCircle2 className="mx-auto h-12 w-12 text-green-300" /><p className="mt-4 text-sm text-gray-400">{t("compliance.noGaps")}</p></div></div>
          ) : (
            <div className="space-y-2">
              {gaps.map(g => {
                const cfg = STATUS_CFG[g.status] || STATUS_CFG.open;
                const SIcon = cfg.icon;
                return (
                  <div key={g.id} className={`${card} flex items-start justify-between`}>
                    <div className="flex items-start gap-3 flex-1">
                      <div className={`flex h-9 w-9 items-center justify-center rounded-lg ${cfg.bg} shrink-0`}><SIcon className={`h-4 w-4 ${cfg.color}`} /></div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="px-1.5 py-0.5 rounded bg-emerald-100 dark:bg-emerald-900/30 text-emerald-600 text-xs font-mono">{g.control_id}</span>
                          <span className="text-xs text-gray-400 uppercase">{g.framework}</span>
                          <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}>{cfg.label}</span>
                        </div>
                        <p className="mt-1 text-sm">{g.gap_description}</p>
                        <div className="mt-1 flex items-center gap-3 text-xs text-gray-400">
                          <span>{t("compliance.remediation")}: {g.remediation_plan}</span>
                          {g.owner && <span>· {t("compliance.owner")}: {g.owner}</span>}
                        </div>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* ════ REMEDIATION ════ */}
      {tab === "remediation" && (
        <div className={card}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><CheckCircle2 className="h-4 w-4" /> {t("compliance.remediationTracking")}</h2>
          {remediation.length === 0 ? (
            <div className="py-8 text-center"><CheckCircle2 className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("compliance.noRemediation")}</p></div>
          ) : (
            <div className="overflow-x-auto"><table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-800/50"><tr>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">{t("compliance.control")}</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">{t("compliance.title_")}</th>
                <th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">{t("compliance.status")}</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">{t("compliance.assignedTo")}</th>
                <th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">{t("compliance.timeline")}</th>
              </tr></thead>
              <tbody className="divide-y dark:divide-gray-800">
                {remediation.map(r => {
                  const cfg = STATUS_CFG[r.status] || STATUS_CFG.open;
                  const SIcon = cfg.icon;
                  return (
                    <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                      <td className="px-3 py-3"><code className="text-xs font-mono text-emerald-500">{r.control}</code></td>
                      <td className="px-3 py-3 text-xs">{r.title}</td>
                      <td className="px-3 py-3 text-center"><span className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs ${cfg.bg} ${cfg.color}`}><SIcon className="h-3 w-3" /> {cfg.label}</span></td>
                      <td className="px-3 py-3 text-xs font-mono text-gray-500">{r.assigned_to || "—"}</td>
                      <td className="px-3 py-3 text-xs text-gray-400">
                        {r.resolved_at ? `${t("compliance.resolved")} ${new Date(r.resolved_at).toLocaleDateString()} (${r.resolution_days}d)` : r.est_completion ? `${t("compliance.estCompletion")} ${r.est_completion}` : r.started_at ? `${t("compliance.started")} ${new Date(r.started_at).toLocaleDateString()}` : "—"}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table></div>
          )}
        </div>
      )}

      {/* ════ EVIDENCE ════ */}
      {tab === "evidence" && (
        <div className={card}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><FileText className="h-4 w-4" /> {t("compliance.evidenceCollection")}</h2>
          {evidence.length === 0 ? (
            <div className="py-8 text-center"><FileText className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("compliance.noEvidence")}</p></div>
          ) : (
            <div className="space-y-2">
              {evidence.map(ev => {
                const cfg = STATUS_CFG[ev.status] || STATUS_CFG.in_progress_evidence;
                const SIcon = cfg.icon;
                return (
                  <div key={ev.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                    <div className="flex items-center gap-3">
                      <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${cfg.bg}`}><SIcon className={`h-4 w-4 ${cfg.color}`} /></div>
                      <div>
                        <div className="flex items-center gap-2">
                          <code className="text-xs font-mono text-emerald-500">{ev.control_id}</code>
                          <span className="text-xs text-gray-400 uppercase">{ev.framework}</span>
                          <span className={`px-1.5 py-0.5 rounded text-xs ${cfg.bg} ${cfg.color}`}>{cfg.label}</span>
                        </div>
                        <p className="text-xs text-gray-400">{t("compliance.collectedBy")}: {ev.collected_by || "—"} · {new Date(ev.collected_at).toLocaleDateString()}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-1">
                      {(ev.artifacts || []).map(a => <span key={a} className="px-1.5 py-0.5 rounded bg-gray-100 dark:bg-gray-700 text-xs font-mono">{a}</span>)}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* ════ REPORTS ════ */}
      {tab === "reports" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Download className="h-4 w-4" /> {t("compliance.generateReport")}</h2>
            <div className="space-y-3">
              <div><label className="text-sm font-medium">{t("compliance.regulation")}</label>
                <select value={rFramework} onChange={e => setRFramework(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                  <option value="nis2">NIS2 Directive</option>
                  <option value="cra">Cyber Resilience Act (CRA)</option>
                  <option value="gdpr">GDPR</option>
                  <option value="soc2">SOC 2 Type II</option>
                  <option value="iso27001">ISO 27001</option>
                  <option value="hipaa">HIPAA</option>
                </select>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium">{t("compliance.from")}</label><input type="date" value={rFrom} onChange={e => setRFrom(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
                <div><label className="text-sm font-medium">{t("compliance.to")}</label><input type="date" value={rTo} onChange={e => setRTo(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              </div>
              <button onClick={generateReport} disabled={generating} className="flex items-center gap-2 rounded-lg bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700 disabled:opacity-50">
                {generating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Download className="h-4 w-4" />} {t("compliance.generate")}
              </button>
            </div>
          </div>
          <div className={card}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><FileText className="h-4 w-4" /> {t("compliance.reportResult")}</h2>
            {reportResult ? (
              <div className="space-y-2">
                <p className="text-sm font-medium">{reportResult.regulation} {t("compliance.findings")}</p>
                {reportResult.findings?.map((f, i) => (
                  <div key={i} className={`rounded-lg border p-3 dark:border-gray-700 ${f.severity === "pass" ? "border-green-200 dark:border-green-800" : f.severity === "warning" ? "border-yellow-200 dark:border-yellow-800" : "border-red-200 dark:border-red-800"}`}>
                    <div className="flex items-center gap-2">
                      <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${f.severity === "pass" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : f.severity === "warning" ? "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600" : "bg-blue-100 dark:bg-blue-900/30 text-blue-600"}`}>{f.severity}</span>
                      <code className="text-xs font-mono text-gray-500">{f.id}</code>
                    </div>
                    <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">{f.description}</p>
                  </div>
                ))}
              </div>
            ) : (
              <div className="py-8 text-center"><Download className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("compliance.noReportGenerated")}</p></div>
            )}
          </div>
        </div>
      )}

      </>)}
    </div>
  );
}
