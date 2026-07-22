"use client";

import { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  ShieldCheck,
  Download,
  Loader2,
  FileText,
  Calendar,
  CheckCircle2,
  XCircle,
  AlertTriangle,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ComplianceReport {
  id: string;
  framework: string;
  status: "compliant" | "partial" | "non-compliant";
  score: number;
  lastAssessed: string;
  controlsTotal: number;
  controlsPassed: number;
  controlsFailed: number;
  summary: string;
}

interface ControlItem {
  id: string;
  name: string;
  description: string;
  status: "pass" | "fail" | "warning";
  evidence: string;
}

const FRAMEWORKS = [
  { value: "soc2", label: "SOC 2 Type II" },
  { value: "hipaa", label: "HIPAA" },
  { value: "gdpr", label: "GDPR" },
  { value: "iso27001", label: "ISO 27001" },
  { value: "pci", label: "PCI DSS" },
];

const MOCK_REPORTS: ComplianceReport[] = [];

const MOCK_CONTROLS: ControlItem[] = [];

export default function ComplianceReportPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [reports, setReports] = useState<ComplianceReport[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [selectedFramework, setSelectedFramework] = useState("all");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [loading, setLoading] = useState(true);
  const [downloading, setDownloading] = useState<string | null>(null);
  const [expandedReport, setExpandedReport] = useState<string | null>(null);
  const [controls, setControls] = useState<ControlItem[]>([]);

  useEffect(() => {
    // Try fetching from API, fall back to mock data
    const load = async () => {
      try {
        const data = await apiFetch<{ reports?: ComplianceReport[] }>("/api/v1/audit/compliance/reports");
        setReports(data.reports ?? MOCK_REPORTS);
      } catch {
        setReports(MOCK_REPORTS);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const handleDownload = async (reportId: string, format: "pdf" | "csv") => {
    setDownloading(`${reportId}-${format}`);
    try {
      // Use native fetch for blob download (apiFetch returns parsed JSON)
      const apiUrl = `/api/v1/audit/compliance/reports/${reportId}/download?format=${format}`;
      const dlRes = await fetch(apiUrl, {
        headers: {
          "Content-Type": "application/json",
          "X-Tenant-ID": localStorage.getItem("ggid_tenant_id") || "",
        },
      });
      if (!dlRes.ok) throw new Error("download failed");
      const blob = await dlRes.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `${reportId}.${format}`;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      // Generate a CSV locally as fallback
      if (format === "csv") {
        const rpt = reports.find((r: any) => r.id === reportId);
        if (rpt) {
          const csv = [
            "Framework,Status,Score,Controls Total,Controls Passed,Controls Failed,Last Assessed",
            `${rpt.framework},${rpt.status},${rpt.score}%,${rpt.controlsTotal},${rpt.controlsPassed},${rpt.controlsFailed},${rpt.lastAssessed}`,
          ].join("\n");
          const blob = new Blob([csv], { type: "text/csv" });
          const url = URL.createObjectURL(blob);
          const a = document.createElement("a");
          a.href = url;
          a.download = `${reportId}.csv`;
          a.click();
          URL.revokeObjectURL(url);
        }
      }
    } finally {
      setDownloading(null);
    }
  };

  const toggleReport = (reportId: string) => {
    if (expandedReport === reportId) {
      setExpandedReport(null);
    } else {
      setExpandedReport(reportId);
      setControls(MOCK_CONTROLS);
    }
  };

  const filtered = reports.filter(
    (r) => selectedFramework === "all" || r.framework === selectedFramework
  );

  const inputCls =
    "rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200";
  const cardCls =
    "rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const statusBadge = (status: string) => {
    const map: Record<string, { cls: string; icon: typeof CheckCircle2; label: string }> = {
      compliant: { cls: "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400", icon: CheckCircle2, label: "Compliant" },
      partial: { cls: "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400", icon: AlertTriangle, label: "Partial" },
      "non-compliant": { cls: "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400", icon: XCircle, label: "Non-Compliant" },
    };
    const cfg = map[status] ?? map["partial"];
    const Icon = cfg.icon;
    return (
      <span className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium ${cfg.cls}`}>
        <Icon className="h-3 w-3" />
        {cfg.label}
      </span>
    );
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <ShieldCheck className="h-7 w-7 text-indigo-600" />
            Compliance Reports
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            View and download compliance assessment reports for SOC 2, HIPAA, GDPR, and more.
          </p>
        </div>
      </div>

      {/* Filters */}
      <div className={`${cardCls} flex flex-wrap items-end gap-4`}>
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
            Framework
          </label>
          <select
            className={inputCls}
            value={selectedFramework}
            onChange={(e) => setSelectedFramework(e.target.value)}
          >
            <option value="all">All Frameworks</option>
            {FRAMEWORKS.map((f: any) => (
              <option key={f.value} value={f.label}>
                {f.label}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
            <Calendar className="mr-1 inline h-3 w-3" />From
          </label>
          <input
            type="date"
            className={inputCls}
            value={dateFrom}
            onChange={(e) => setDateFrom(e.target.value)}
          />
        </div>
        <div>
          <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
            <Calendar className="mr-1 inline h-3 w-3" />To
          </label>
          <input
            type="date"
            className={inputCls}
            value={dateTo}
            onChange={(e) => setDateTo(e.target.value)}
          />
        </div>
      </div>

      {/* Reports list */}
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
        </div>
      ) : (
        <div className="space-y-4">
          {filtered.map((report: any) => (
            <div key={report.id} className={cardCls}>
              {/* Report header */}
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3">
                    <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                      {report.framework}
                    </h3>
                    {statusBadge(report.status)}
                  </div>
                  <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    {report.summary}
                  </p>
                  <div className="mt-3 flex flex-wrap gap-4 text-sm">
                    <span className="text-gray-600 dark:text-gray-300">
                      Score: <span className="font-semibold text-indigo-600">{report.score}%</span>
                    </span>
                    <span className="text-gray-600 dark:text-gray-300">
                      Controls: {report.controlsPassed}/{report.controlsTotal} passed
                    </span>
                    <span className="text-gray-600 dark:text-gray-300">
                      Last assessed: {report.lastAssessed}
                    </span>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => toggleReport(report.id)}
                    className="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700"
                  >
                    <FileText className="mr-1 inline h-3.5 w-3.5" />
                    Details
                  </button>
                  <button
                    onClick={() => handleDownload(report.id, "pdf")}
                    disabled={downloading === `${report.id}-pdf`}
                    className="rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
                  >
                    {downloading === `${report.id}-pdf` ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    ) : (
                      <Download className="mr-1 inline h-3.5 w-3.5" />
                    )}
                    PDF
                  </button>
                  <button
                    onClick={() => handleDownload(report.id, "csv")}
                    disabled={downloading === `${report.id}-csv`}
                    className="rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-700 disabled:opacity-50"
                  >
                    {downloading === `${report.id}-csv` ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    ) : (
                      <Download className="mr-1 inline h-3.5 w-3.5" />
                    )}
                    CSV
                  </button>
                </div>
              </div>

              {/* Score bar */}
              <div className="mt-4">
                <div className="h-2 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                  <div
                    className={`h-full rounded-full ${
                      report.score >= 90
                        ? "bg-green-500"
                        : report.score >= 70
                        ? "bg-yellow-500"
                        : "bg-red-500"
                    }`}
                    style={{ width: `${report.score}%` }}
                  />
                </div>
              </div>

              {/* Expanded controls */}
              {expandedReport === report.id && (
                <div className="mt-6 border-t border-gray-200 pt-4 dark:border-gray-700">
                  <h4 className="mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">
                    Control Details
                  </h4>
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b border-gray-200 text-left text-xs uppercase text-gray-400 dark:border-gray-700">
                          <th scope="col" className="pb-2 pr-4">ID</th>
                          <th scope="col" className="pb-2 pr-4">Control</th>
                          <th scope="col" className="pb-2 pr-4">Status</th>
                          <th scope="col" className="pb-2 pr-4">Evidence</th>
                        </tr>
                      </thead>
                      <tbody>
                        {controls.map((ctrl: any) => (
                          <tr
                            key={ctrl.id}
                            className="border-b border-gray-100 dark:border-gray-700/50"
                          >
                            <td className="py-2 pr-4 font-mono text-xs text-gray-500 dark:text-gray-400">
                              {ctrl.id}
                            </td>
                            <td className="py-2 pr-4">
                              <div className="font-medium text-gray-800 dark:text-gray-200">
                                {ctrl.name}
                              </div>
                              <div className="text-xs text-gray-400">
                                {ctrl.description}
                              </div>
                            </td>
                            <td className="py-2 pr-4">
                              {ctrl.status === "pass" && (
                                <span className="inline-flex items-center gap-1 text-xs text-green-600">
                                  <CheckCircle2 className="h-3 w-3" /> Pass
                                </span>
                              )}
                              {ctrl.status === "fail" && (
                                <span className="inline-flex items-center gap-1 text-xs text-red-600">
                                  <XCircle className="h-3 w-3" /> Fail
                                </span>
                              )}
                              {ctrl.status === "warning" && (
                                <span className="inline-flex items-center gap-1 text-xs text-yellow-600">
                                  <AlertTriangle className="h-3 w-3" /> Warning
                                </span>
                              )}
                            </td>
                            <td className="py-2 pr-4 text-xs text-gray-400">
                              {ctrl.evidence}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
