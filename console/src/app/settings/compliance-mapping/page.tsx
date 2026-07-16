"use client";

import { useState } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  GitBranch, Loader2, AlertCircle, X, CheckCircle, XCircle, Clock, FileText, Activity,
} from "lucide-react";

interface ControlMapping {
  id: string;
  framework: string;
  control_id: string;
  requirement: string;
  category: string;
  status: "compliant" | "partial" | "missing" | "not_applicable";
  evidence_count: number;
  mapped_policies: string[];
  gaps: string[];
}

const statusIcons: Record<string, React.ReactNode> = {
  compliant: <CheckCircle className="h-4 w-4 text-green-500" />,
  partial: <Clock className="h-4 w-4 text-yellow-500" />,
  missing: <XCircle className="h-4 w-4 text-red-500" />,
  not_applicable: <Activity className="h-4 w-4 text-gray-400" />,
};

const statusColors: Record<string, string> = {
  compliant: "text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400",
  partial: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400",
  missing: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
  not_applicable: "text-gray-600 bg-gray-100 dark:bg-gray-700 dark:text-gray-400",
};

const frameworks = ["SOC2", "GDPR", "HIPAA", "ISO27001", "PCI-DSS"];

export default function ComplianceMappingPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [mappings, setMappings] = useState<ControlMapping[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [framework, setFramework] = useState("SOC2");

  const load = async (fw: string) => {
    setLoading(true);
    try { setMappings(await apiFetch<ControlMapping[]>(`/api/v1/audit/compliance-mapping?framework=${fw}`).catch(() => [])); }
    catch { setError("Failed to load mappings"); }
    finally { setLoading(false); }
  };

  useState(() => { load(framework); });

  const handleStatusChange = async (id: string, status: ControlMapping["status"]) => {
    try { await apiFetch(`/api/v1/audit/compliance-mapping/${id}`, { method: "PUT", body: JSON.stringify({ status }) }); setMappings((p) => p.map((m) => m.id === id ? { ...m, status } : m)); }
    catch { setError("Update failed"); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const stats = { compliant: mappings.filter((m) => m.status === "compliant").length, partial: mappings.filter((m) => m.status === "partial").length, missing: mappings.filter((m) => m.status === "missing").length };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><GitBranch className="h-6 w-6 text-teal-600" />{t("complianceMapping.title")}</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Map security controls to compliance framework requirements.</p>
      </div>

      {/* Framework selector */}
      <div className="flex gap-2">
        {frameworks.map((fw) => (
          <button key={fw} onClick={() => { setFramework(fw); load(fw); }} className={`rounded-lg px-4 py-2 text-sm font-medium ${framework === fw ? "bg-teal-600 text-white" : "bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-300"}`}>{fw}</button>
        ))}
      </div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-teal-600" /></div>
      : (
        <>
          {/* Stats */}
          <div className="grid grid-cols-3 gap-4">
            <div className={cardCls}><div className="flex items-center gap-2"><CheckCircle className="h-4 w-4 text-green-500" /><span className="text-xs font-semibold uppercase text-gray-400">Compliant</span></div><p className="mt-2 text-2xl font-bold text-green-600">{stats.compliant}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><Clock className="h-4 w-4 text-yellow-500" /><span className="text-xs font-semibold uppercase text-gray-400">Partial</span></div><p className="mt-2 text-2xl font-bold text-yellow-600">{stats.partial}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><XCircle className="h-4 w-4 text-red-500" /><span className="text-xs font-semibold uppercase text-gray-400">Missing</span></div><p className="mt-2 text-2xl font-bold text-red-600">{stats.missing}</p></div>
          </div>

          {/* Mapping table */}
          {mappings.length === 0 ? (
            <div className={cardCls}><div className="py-12 text-center"><GitBranch className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No control mappings for {framework}.</p></div></div>
          ) : (
            <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
              <table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-800"><tr>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Control</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Requirement</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Category</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Status</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Evidence</th>
                  <th className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Policies</th>
                </tr></thead>
                <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                  {mappings.map((m) => (
                    <tr key={m.id} className="bg-white dark:bg-gray-900">
                      <td className="px-4 py-3"><span className="font-mono text-xs font-semibold text-gray-900 dark:text-white">{m.control_id}</span></td>
                      <td className="px-4 py-3 text-gray-600 dark:text-gray-300">{m.requirement}</td>
                      <td className="px-4 py-3 text-gray-400">{m.category}</td>
                      <td className="px-4 py-3"><select value={m.status} onChange={(e) => handleStatusChange(m.id, e.target.value as ControlMapping["status"])} className={`rounded-full px-2 py-0.5 text-xs font-medium border-0 ${statusColors[m.status] || ""}`}>{statusIcons[m.status]}<option value="compliant">compliant</option><option value="partial">partial</option><option value="missing">missing</option><option value="not_applicable">not_applicable</option></select></td>
                      <td className="px-4 py-3"><span className="flex items-center gap-1 text-gray-500"><FileText className="h-3 w-3" />{m.evidence_count}</span></td>
                      <td className="px-4 py-3"><div className="flex flex-wrap gap-1">{m.mapped_policies.slice(0, 3).map((p) => <span key={p} className="rounded bg-indigo-100 px-1 py-0.5 text-xs text-indigo-600 dark:bg-indigo-900/30">{p}</span>)}{m.mapped_policies.length > 3 && <span className="text-xs text-gray-400">+{m.mapped_policies.length - 3}</span>}</div></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  );
}
