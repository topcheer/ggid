"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  ShieldCheck, TrendingUp, Play, Loader2, Check, AlertTriangle,
  X, Download, FileJson, FileText, RefreshCw, Clock, Target,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
type TabId = "controls" | "history" | "actions";

interface Control {
  id: string; framework: string; control_id: string; name: string;
  status: "pass" | "warn" | "fail"; metric: number; threshold: number;
  metric_label: string; last_checked: string;
}

const FRAMEWORKS = ["DORA", "HIPAA", "SOX", "ISO 27001", "NIST CSF"];

const statusConfig: Record<string, { color: string; icon: typeof Check; label: string }> = {
  pass: { color: "bg-green-100 text-green-700 dark:bg-green-950 dark:text-green-300", icon: Check, label: "statusPass" },
  warn: { color: "bg-yellow-100 text-yellow-700 dark:bg-yellow-950 dark:text-yellow-300", icon: AlertTriangle, label: "statusWarn" },
  fail: { color: "bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300", icon: X, label: "statusFail" },
};

export default function CCMPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<TabId>("controls");
  const [controls, setControls] = useState<Control[]>([]);
  const [loading, setLoading] = useState(true);
  const [frameworkFilter, setFrameworkFilter] = useState("all");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/audit/ccm/results`, { headers: { ...authHeader() } });
      if (res.ok) { const d = await res.json(); setControls(d.controls || d || []); return; }
    } catch { /* mock */ }
    setControls([
      { id: "c1", framework: "DORA", control_id: "DORA-ICT-3.1", name: "Multi-factor authentication coverage", status: "pass", metric: 88, threshold: 80, metric_label: "% users with MFA", last_checked: "2025-07-18T09:00:00Z" },
      { id: "c2", framework: "DORA", control_id: "DORA-ICT-4.2", name: "Privileged access reviews", status: "warn", metric: 67, threshold: 90, metric_label: "% reviews completed", last_checked: "2025-07-18T09:00:00Z" },
      { id: "c3", framework: "HIPAA", control_id: "HIPAA-164.312(a)(1)", name: "Access control mechanism", status: "pass", metric: 100, threshold: 100, metric_label: "% resources with ACL", last_checked: "2025-07-18T08:30:00Z" },
      { id: "c4", framework: "HIPAA", control_id: "HIPAA-164.312(b)", name: "Audit controls", status: "pass", metric: 95, threshold: 90, metric_label: "% actions audited", last_checked: "2025-07-18T09:00:00Z" },
      { id: "c5", framework: "SOX", control_id: "SOX-404", name: "Segregation of duties enforcement", status: "fail", metric: 72, threshold: 95, metric_label: "% SoD rules enforced", last_checked: "2025-07-18T09:00:00Z" },
      { id: "c6", framework: "SOX", control_id: "SOX-302", name: "Periodic access recertification", status: "warn", metric: 60, threshold: 85, metric_label: "% certifications current", last_checked: "2025-07-18T08:45:00Z" },
      { id: "c7", framework: "ISO 27001", control_id: "A.9.2.1", name: "User registration and de-registration", status: "pass", metric: 92, threshold: 90, metric_label: "% accounts with owner", last_checked: "2025-07-18T09:00:00Z" },
      { id: "c8", framework: "ISO 27001", control_id: "A.9.2.6", name: "Removal of access rights", status: "pass", metric: 98, threshold: 95, metric_label: "% access removed within SLA", last_checked: "2025-07-18T09:00:00Z" },
      { id: "c9", framework: "ISO 27001", control_id: "A.12.4.1", name: "Event logging", status: "pass", metric: 100, threshold: 100, metric_label: "% critical events logged", last_checked: "2025-07-18T09:00:00Z" },
      { id: "c10", framework: "NIST CSF", control_id: "PR.AC-1", name: "Identity verification", status: "pass", metric: 95, threshold: 90, metric_label: "% users identity-verified", last_checked: "2025-07-18T09:00:00Z" },
      { id: "c11", framework: "NIST CSF", control_id: "PR.AC-4", name: "Access permission management", status: "warn", metric: 78, threshold: 90, metric_label: "% least-privilege compliant", last_checked: "2025-07-18T08:30:00Z" },
      { id: "c12", framework: "NIST CSF", control_id: "DE.CM-1", name: "Network monitored", status: "pass", metric: 100, threshold: 100, metric_label: "% networks monitored", last_checked: "2025-07-18T09:00:00Z" },
      { id: "c13", framework: "DORA", control_id: "DORA-ICT-5.1", name: "Encryption at rest", status: "pass", metric: 100, threshold: 100, metric_label: "% data encrypted", last_checked: "2025-07-18T09:00:00Z" },
      { id: "c14", framework: "HIPAA", control_id: "HIPAA-164.308(a)(3)", name: "Workforce security training", status: "fail", metric: 65, threshold: 90, metric_label: "% workforce trained", last_checked: "2025-07-18T08:00:00Z" },
      { id: "c15", framework: "ISO 27001", control_id: "A.6.1.2", name: "Segregation of duties", status: "warn", metric: 80, threshold: 95, metric_label: "% critical functions SoD-compliant", last_checked: "2025-07-18T09:00:00Z" },
    ]);
  }, []);

  useEffect(() => { load(); }, [load]);

  const filtered = frameworkFilter === "all" ? controls : controls.filter((c) => c.framework === frameworkFilter);
  const passCount = controls.filter((c) => c.status === "pass").length;
  const warnCount = controls.filter((c) => c.status === "warn").length;
  const failCount = controls.filter((c) => c.status === "fail").length;

  const tabs: { id: TabId; label: string; icon: typeof ShieldCheck }[] = [
    { id: "controls", label: t("ccm.tabs.controls"), icon: ShieldCheck },
    { id: "history", label: t("ccm.tabs.history"), icon: TrendingUp },
    { id: "actions", label: t("ccm.tabs.actions"), icon: Play },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-5xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <ShieldCheck className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("ccm.title")}</h1>
          </div>
          <p className="text-gray-600 dark:text-gray-400 text-sm">{t("ccm.description")}</p>
        </div>

        {/* Summary bar */}
        {!loading && (
          <div className="flex gap-4 mb-4">
            <div className="flex items-center gap-2 px-4 py-2 bg-green-50 dark:bg-green-950/30 rounded-lg">
              <Check className="w-4 h-4 text-green-600" />
              <span className="text-sm font-medium text-green-700 dark:text-green-300">{passCount} {t("ccm.controls.statusPass")}</span>
            </div>
            <div className="flex items-center gap-2 px-4 py-2 bg-yellow-50 dark:bg-yellow-950/30 rounded-lg">
              <AlertTriangle className="w-4 h-4 text-yellow-600" />
              <span className="text-sm font-medium text-yellow-700 dark:text-yellow-300">{warnCount} {t("ccm.controls.statusWarn")}</span>
            </div>
            <div className="flex items-center gap-2 px-4 py-2 bg-red-50 dark:bg-red-950/30 rounded-lg">
              <X className="w-4 h-4 text-red-600" />
              <span className="text-sm font-medium text-red-700 dark:text-red-300">{failCount} {t("ccm.controls.statusFail")}</span>
            </div>
          </div>
        )}

        <div className="flex gap-1 mb-6 bg-gray-200 dark:bg-gray-800 rounded-lg p-1">
          {tabs.map(({ id, label, icon: Icon }) => (
            <button key={id} onClick={() => setTab(id)}
              className={`flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                tab === id ? "bg-white dark:bg-gray-700 text-blue-600 dark:text-blue-400 shadow-sm" : "text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white"
              }`}>
              <Icon className="w-4 h-4" />{label}
            </button>
          ))}
        </div>

        {tab === "controls" && <ControlsTab controls={filtered} loading={loading} framework={frameworkFilter} setFramework={setFrameworkFilter} />}
        {tab === "history" && <HistoryTab />}
        {tab === "actions" && <ActionsTab onRun={load} />}
      </div>
    </div>
  );
}

// ============ Controls Tab ============

function ControlsTab({ controls, loading, framework, setFramework }: {
  controls: Control[]; loading: boolean; framework: string; setFramework: (v: string) => void;
}) {
  const t = useTranslations();

  if (loading) return <Spinner />;

  return (
    <div className="space-y-3">
      <div className="flex gap-2">
        <button onClick={() => setFramework("all")} className={`px-3 py-1.5 rounded-lg text-xs font-medium ${framework === "all" ? "bg-blue-600 text-white" : "bg-white dark:bg-gray-800 text-gray-600 border border-gray-200 dark:border-gray-700"}`}>{t("ccm.controls.frameworkAll")}</button>
        {FRAMEWORKS.map((f) => (
          <button key={f} onClick={() => setFramework(f)} className={`px-3 py-1.5 rounded-lg text-xs font-medium ${framework === f ? "bg-blue-600 text-white" : "bg-white dark:bg-gray-800 text-gray-600 border border-gray-200 dark:border-gray-700"}`}>{f}</button>
        ))}
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead><tr className="border-b border-gray-200 dark:border-gray-800 text-left bg-gray-50 dark:bg-gray-800/50">
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("ccm.controls.framework")}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("ccm.controls.control")}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("ccm.controls.status")}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("ccm.controls.metric")}</th>
              <th className="py-2 px-4 font-medium text-gray-600 dark:text-gray-400">{t("ccm.controls.lastChecked")}</th>
            </tr></thead>
            <tbody>
              {controls.map((c) => {
                const cfg = statusConfig[c.status];
                const Icon = cfg.icon;
                return (
                  <tr key={c.id} className="border-b border-gray-100 dark:border-gray-800/50">
                    <td className="py-3 px-4"><span className="text-xs px-2 py-0.5 bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded">{c.framework}</span></td>
                    <td className="py-3 px-4">
                      <div className="font-mono text-xs text-gray-400">{c.control_id}</div>
                      <div className="text-sm text-gray-900 dark:text-white">{c.name}</div>
                    </td>
                    <td className="py-3 px-4">
                      <span className={`flex items-center gap-1 px-2 py-0.5 text-xs rounded-full w-fit ${cfg.color}`}>
                        <Icon className="w-3 h-3" />{t(`ccm.controls.${cfg.label}`)}
                      </span>
                    </td>
                    <td className="py-3 px-4">
                      <div className="flex items-center gap-2">
                        <div className="w-16 h-1.5 bg-gray-200 dark:bg-gray-800 rounded-full overflow-hidden">
                          <div className={`h-full rounded-full ${c.metric >= c.threshold ? "bg-green-500" : c.metric >= c.threshold * 0.8 ? "bg-yellow-500" : "bg-red-500"}`} style={{ width: `${c.metric}%` }} />
                        </div>
                        <span className="text-xs font-medium text-gray-900 dark:text-white">{c.metric}%</span>
                        <span className="text-xs text-gray-400">/ {c.threshold}%</span>
                      </div>
                      <div className="text-xs text-gray-400 mt-0.5">{c.metric_label}</div>
                    </td>
                    <td className="py-3 px-4 text-xs text-gray-500">{new Date(c.last_checked).toLocaleString()}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

// ============ History Tab ============

function HistoryTab() {
  const t = useTranslations();
  const [range, setRange] = useState<"30d" | "90d">("30d");

  // Mock trend data
  const days = range === "30d" ? 30 : 90;
  const data = Array.from({ length: Math.floor(days / 3) }, (_, i) => {
    const date = new Date(); date.setDate(date.getDate() - (Math.floor(days / 3) - 1 - i) * 3);
    return {
      day: date.toISOString().split("T")[0],
      DORA: Math.round(85 + Math.random() * 10 - i * 0.2),
      HIPAA: Math.round(90 + Math.random() * 8 - i * 0.1),
      SOX: Math.round(70 + Math.random() * 15 + i * 0.3),
      ISO: Math.round(88 + Math.random() * 8 - i * 0.05),
    };
  });

  const colors: Record<string, string> = { DORA: "#3b82f6", HIPAA: "#22c55e", SOX: "#ef4444", ISO: "#8b5cf6" };
  const chartW = 700, chartH = 200, pad = 35;
  const maxVal = 100;
  const xStep = data.length > 1 ? (chartW - pad * 2) / (data.length - 1) : 0;
  const yScale = (v: number) => chartH - pad - (v / maxVal) * (chartH - pad * 2);
  const linePath = (key: string) => data.map((d, i) => `${i === 0 ? "M" : "L"} ${pad + i * xStep} ${yScale((d as Record<string, unknown>)[key] as number)}`).join(" ");

  return (
    <div className="space-y-4">
      <div className="flex gap-2">
        {(["30d", "90d"] as const).map((r) => (
          <button key={r} onClick={() => setRange(r)} className={`px-3 py-1.5 rounded-lg text-xs font-medium ${range === r ? "bg-blue-600 text-white" : "bg-white dark:bg-gray-800 text-gray-600 border border-gray-200 dark:border-gray-700"}`}>
            {r === "30d" ? t("ccm.history.days30") : t("ccm.history.days90")}
          </button>
        ))}
      </div>
      <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-4">{t("ccm.history.title")}</h3>
        <p className="text-xs text-gray-500 dark:text-gray-400 mb-3">{t("ccm.history.description")}</p>
        <svg viewBox={`0 0 ${chartW} ${chartH}`} className="w-full h-48">
          {[0, 25, 50, 75, 100].map((p) => (
            <g key={p}>
              <line x1={pad} y1={yScale(p)} x2={chartW - pad} y2={yScale(p)} stroke="currentColor" className="text-gray-100 dark:text-gray-800" strokeWidth={1} />
              <text x={pad - 5} y={yScale(p) + 3} textAnchor="end" className="fill-gray-400 text-xs">{p}</text>
            </g>
          ))}
          {Object.keys(colors).map((fw) => (
            <path key={fw} d={linePath(fw)} fill="none" stroke={colors[fw]} strokeWidth={2} />
          ))}
        </svg>
        <div className="flex items-center justify-center gap-4 mt-2">
          {Object.entries(colors).map(([fw, color]) => (
            <div key={fw} className="flex items-center gap-1.5"><div className="w-3 h-3 rounded" style={{ backgroundColor: color }} /><span className="text-xs text-gray-600 dark:text-gray-400">{fw}</span></div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ============ Actions Tab ============

function ActionsTab({ onRun }: { onRun: () => void }) {
  const t = useTranslations();
  const [running, setRunning] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const runNow = async () => {
    setRunning(true);
    try {
      await fetch(`${API_BASE}/api/v1/audit/ccm/run`, { method: "POST", headers: { ...authHeader() } });
    } catch { /* ok */ }
    setRunning(false);
    onRun();
    setMsg(t("ccm.actions.runSuccess"));
    setTimeout(() => setMsg(null), 3000);
  };

  const exportReport = (format: "csv" | "json") => {
    const blob = new Blob([format === "json" ? "{}" : "framework,control,status,metric,threshold\n"], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url; a.download = `ccm-report.${format}`; a.click();
  };

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 space-y-5">
      <div>
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white">{t("ccm.actions.title")}</h3>
        <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{t("ccm.actions.description")}</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="p-4 rounded-lg bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-900">
          <div className="flex items-center gap-2 mb-3"><RefreshCw className="w-5 h-5 text-blue-600" /><h4 className="text-sm font-medium text-gray-900 dark:text-white">{t("ccm.actions.runNow")}</h4></div>
          <p className="text-xs text-gray-500 mb-3">{t("ccm.actions.lastRun")}: 2025-07-18 09:00</p>
          <p className="text-xs text-gray-500 mb-3">{t("ccm.actions.nextRun")}: 2025-07-18 21:00 ({t("ccm.actions.schedule")} 12h)</p>
          <button onClick={runNow} disabled={running} className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium">
            {running ? <Loader2 className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
            {running ? t("ccm.actions.running") : t("ccm.actions.runNow")}
          </button>
        </div>

        <div className="p-4 rounded-lg bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-900">
          <div className="flex items-center gap-2 mb-3"><Download className="w-5 h-5 text-green-600" /><h4 className="text-sm font-medium text-gray-900 dark:text-white">{t("ccm.actions.exportReport")}</h4></div>
          <div className="flex gap-2">
            <button onClick={() => exportReport("csv")} className="flex items-center gap-1.5 px-3 py-1.5 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-lg text-sm hover:bg-gray-50">
              <FileText className="w-4 h-4 text-green-600" />{t("ccm.actions.exportCsv")}
            </button>
            <button onClick={() => exportReport("json")} className="flex items-center gap-1.5 px-3 py-1.5 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-700 rounded-lg text-sm hover:bg-gray-50">
              <FileJson className="w-4 h-4 text-blue-600" />{t("ccm.actions.exportJson")}
            </button>
          </div>
        </div>
      </div>

      {msg && <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300 text-sm"><Check className="w-4 h-4" />{msg}</div>}
    </div>
  );
}

// ============ Shared ============

function Spinner() { return <div className="flex justify-center py-20"><Loader2 className="w-8 h-8 animate-spin text-blue-600" /></div>; }
