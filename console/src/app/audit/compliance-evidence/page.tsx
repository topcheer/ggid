"use client";
import { useState, useCallback, useEffect } from "react";
import {
  ShieldCheck, Loader2, AlertCircle, X, RefreshCw, FileText,
  CheckCircle2, XCircle, AlertTriangle, ChevronRight, Download,
  Clock, Upload, Hash,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "";

interface Framework { id: string; name: string; controls_total: number; controls_pass: number; controls_fail: number; last_assessed: string; coverage: number; }
interface EvidenceItem { id: string; control_id: string; framework: string; artifact_type: string; collected_by: string; collected_at: string; hash: string; notes: string; }
interface Gap { id: string; control_id: string; framework: string; description: string; remediation: string; assignee: string; due_date: string; status: "open" | "in_progress" | "resolved"; }

type Tab = "frameworks" | "evidence" | "gaps";

const FRAMEWORKS: Framework[] = [
  { id: "soc2", name: "SOC 2 Type II", controls_total: 64, controls_pass: 58, controls_fail: 6, last_assessed: "2025-01-10", coverage: 91 },
  { id: "iso27001", name: "ISO 27001", controls_total: 93, controls_pass: 79, controls_fail: 14, last_assessed: "2024-12-15", coverage: 85 },
  { id: "nist", name: "NIST 800-53", controls_total: 246, controls_pass: 201, controls_fail: 45, last_assessed: "2024-11-20", coverage: 82 },
  { id: "nis2", name: "NIS2 Directive", controls_total: 38, controls_pass: 33, controls_fail: 5, last_assessed: "2025-01-12", coverage: 87 },
];

const SAMPLE_EVIDENCE: EvidenceItem[] = [
  { id: "ev-001", control_id: "CC6.1", framework: "soc2", artifact_type: "screenshot", collected_by: "admin@company.com", collected_at: "2025-01-10T14:00:00Z", hash: "sha256:a1b2c3...", notes: "MFA enrollment dashboard screenshot" },
  { id: "ev-002", control_id: "A.9.1.1", framework: "iso27001", artifact_type: "api_export", collected_by: "system:collector", collected_at: "2025-01-09T08:00:00Z", hash: "sha256:d4e5f6...", notes: "Access control policy JSON export" },
  { id: "ev-003", control_id: "AC-2", framework: "nist", artifact_type: "policy_diff", collected_by: "compliance@company.com", collected_at: "2025-01-08T16:30:00Z", hash: "sha256:g7h8i9...", notes: "Account management policy v2.1 diff" },
];

const SAMPLE_GAPS: Gap[] = [
  { id: "gap-001", control_id: "CC6.5", framework: "soc2", description: "Data at rest encryption not enabled for all databases", remediation: "Enable AES-256 on all PostgreSQL instances", assignee: "devops", due_date: "2025-02-15", status: "in_progress" },
  { id: "gap-002", control_id: "A.10.1.1", framework: "iso27001", description: "Network segmentation incomplete between prod and dev", remediation: "Complete VLAN separation and firewall rules", assignee: "infra", due_date: "2025-03-01", status: "open" },
  { id: "gap-003", control_id: "AC-17", framework: "nist", description: "Remote access logging gaps in 2 sub-systems", remediation: "Deploy centralized logging agent to all nodes", assignee: "ops", due_date: "2025-02-28", status: "in_progress" },
  { id: "gap-004", control_id: "CC8.1", framework: "soc2", description: "Change management lacks formal approval workflow", remediation: "Implement GitOps approval gates", assignee: "engineering", due_date: "2025-01-30", status: "resolved" },
];

const GAP_STATUS_CFG: Record<string, { color: string; bg: string }> = {
  resolved: { color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30" },
  in_progress: { color: "text-blue-600", bg: "bg-blue-100 dark:bg-blue-900/30" },
  open: { color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30" },
};

export default function ComplianceEvidencePage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("frameworks");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [evidence, setEvidence] = useState<EvidenceItem[]>([]);
  const [gaps, setGaps] = useState<Gap[]>([]);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [eRes, gRes] = await Promise.all([
        fetch("/api/v1/audit/compliance/evidence", { headers: h }).catch(() => null),
        fetch("/api/v1/audit/compliance/gaps", { headers: h }).catch(() => null),
      ]);
      if (eRes?.ok) { const d = await eRes.json(); setEvidence(d.evidence || d.records || SAMPLE_EVIDENCE); }
      else setEvidence(SAMPLE_EVIDENCE);
      if (gRes?.ok) { const d = await gRes.json(); setGaps(d.gaps || SAMPLE_GAPS); }
      else setGaps(SAMPLE_GAPS);
    } catch { setError(t("complianceEvidence.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const openGaps = gaps.filter(g => g.status === "open").length;
  const inProgressGaps = gaps.filter(g => g.status === "in_progress").length;
  const resolvedGaps = gaps.filter(g => g.status === "resolved").length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldCheck className="h-6 w-6 text-emerald-500" /> {t("complianceEvidence.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("complianceEvidence.subtitle")}</p>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "frameworks" as Tab, label: t("complianceEvidence.frameworks"), icon: ShieldCheck },
          { id: "evidence" as Tab, label: `${t("complianceEvidence.evidenceVault")} (${evidence.length})`, icon: FileText },
          { id: "gaps" as Tab, label: `${t("complianceEvidence.gaps")} (${openGaps})`, icon: AlertTriangle },
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

      {/* FRAMEWORKS */}
      {tab === "frameworks" && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">{FRAMEWORKS.map(f => (
          <div key={f.id} className={card + " hover:shadow-md transition"}>
            <div className="flex items-start justify-between">
              <h3 className="font-semibold text-sm">{f.name}</h3>
              <span className={`px-2 py-0.5 rounded text-sm font-bold ${f.coverage >= 90 ? "text-green-600 bg-green-100 dark:bg-green-900/30" : f.coverage >= 80 ? "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30" : "text-red-600 bg-red-100 dark:bg-red-900/30"}`}>{f.coverage}%</span>
            </div>
            <div className="mt-3 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
              <div className={`h-full rounded-full ${f.coverage >= 90 ? "bg-green-500" : "bg-yellow-500"}`} style={{ width: `${f.coverage}%` }} />
            </div>
            <div className="mt-3 grid grid-cols-3 gap-2 text-center text-xs">
              <div><p className="font-bold text-green-600">{f.controls_pass}</p><p className="text-gray-400">{t("complianceEvidence.passing")}</p></div>
              <div><p className="font-bold text-red-600">{f.controls_fail}</p><p className="text-gray-400">{t("complianceEvidence.failing")}</p></div>
              <div><p className="font-bold">{f.controls_total}</p><p className="text-gray-400">{t("complianceEvidence.total")}</p></div>
            </div>
            <p className="mt-2 text-xs text-gray-400">{t("complianceEvidence.lastAssessed")}: {new Date(f.last_assessed).toLocaleDateString()}</p>
          </div>
        ))}</div>
      )}

      {/* EVIDENCE VAULT */}
      {tab === "evidence" && (
        <div>
          <div className="mb-4"><button className="flex items-center gap-1 rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-emerald-700"><Upload className="h-3 w-3" /> {t("complianceEvidence.uploadEvidence")}</button></div>
          <div className="space-y-2">
            {evidence.map(ev => (
              <div key={ev.id} className={`${card} flex items-center justify-between !p-3`}>
                <div className="flex items-center gap-3">
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-emerald-100 dark:bg-emerald-900/30"><FileText className="h-4 w-4 text-emerald-500" /></div>
                  <div>
                    <div className="flex items-center gap-2">
                      <code className="text-xs font-mono text-emerald-500">{ev.control_id}</code>
                      <span className="px-1.5 py-0.5 rounded bg-gray-100 dark:bg-gray-700 text-xs font-mono">{ev.artifact_type}</span>
                    </div>
                    <p className="text-xs text-gray-400">{ev.notes}</p>
                    <p className="text-xs text-gray-400">{ev.collected_by} · {new Date(ev.collected_at).toLocaleDateString()}</p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <span className="flex items-center gap-1 text-xs text-gray-400"><Hash className="h-3 w-3" />{ev.hash.slice(0, 12)}</span>
                  <button aria-label="Download" className="rounded p-1.5 text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700"><Download className="h-3.5 w-3.5" /></button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* GAPS */}
      {tab === "gaps" && (
        <div className="space-y-6">
          <div className="grid grid-cols-3 gap-4">
            <div className={`${card} text-center`}><AlertTriangle className="mx-auto h-5 w-5 text-red-400" /><p className="mt-2 text-2xl font-bold text-red-600">{openGaps}</p><p className="text-xs text-gray-400">{t("complianceEvidence.openGaps")}</p></div>
            <div className={`${card} text-center`}><Clock className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-2 text-2xl font-bold text-blue-600">{inProgressGaps}</p><p className="text-xs text-gray-400">{t("complianceEvidence.inProgressGaps")}</p></div>
            <div className={`${card} text-center`}><CheckCircle2 className="mx-auto h-5 w-5 text-green-400" /><p className="mt-2 text-2xl font-bold text-green-600">{resolvedGaps}</p><p className="text-xs text-gray-400">{t("complianceEvidence.resolvedGaps")}</p></div>
          </div>
          <div className="space-y-2">
            {gaps.map(g => {
              const cfg = GAP_STATUS_CFG[g.status] || GAP_STATUS_CFG.open;
              return (
                <div key={g.id} className={`${card} flex items-start justify-between`}>
                  <div className="flex items-start gap-3 flex-1">
                    <div className={`flex h-9 w-9 items-center justify-center rounded-lg ${cfg.bg} shrink-0`}>
                      {g.status === "resolved" ? <CheckCircle2 className={`h-4 w-4 ${cfg.color}`} /> : g.status === "in_progress" ? <Clock className={`h-4 w-4 ${cfg.color}`} /> : <AlertTriangle className={`h-4 w-4 ${cfg.color}`} />}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <code className="text-xs font-mono text-emerald-500">{g.control_id}</code>
                        <span className="text-xs text-gray-400 uppercase">{g.framework}</span>
                        <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}>{g.status}</span>
                      </div>
                      <p className="mt-1 text-sm">{g.description}</p>
                      <p className="text-xs text-gray-400">{t("complianceEvidence.remediation")}: {g.remediation} · {t("complianceEvidence.assignee")}: {g.assignee} · {t("complianceEvidence.due")}: {new Date(g.due_date).toLocaleDateString()}</p>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      </>)}
    </div>
  );
}
