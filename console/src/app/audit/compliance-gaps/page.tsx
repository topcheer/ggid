"use client";

import { useState, useEffect, useCallback } from "react";
import { ShieldCheck, X, AlertCircle, Calendar, User } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ComplianceGap {
  id: string;
  control_id: string;
  framework: string;
  description: string;
  remediation_plan: string;
  owner: string;
  due_date: string;
  status: "open" | "in_progress" | "remediated" | "accepted_risk";
  severity: "low" | "medium" | "high" | "critical";
}

const statusColors: Record<string, string> = {
  open: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  in_progress: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  remediated: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  accepted_risk: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
};

const severityColors: Record<string, string> = {
  low: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function ComplianceGapsPage() {
  const t = useTranslations();

  const [gaps, setGaps] = useState<ComplianceGap[]>([]);
  const [loading, setLoading] = useState(false);
  const [filterStatus, setFilterStatus] = useState("all");
  const [updateGap, setUpdateGap] = useState<ComplianceGap | null>(null);
  const [newStatus, setNewStatus] = useState("");

  const fetchGaps = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/audit/compliance-gaps", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setGaps(data.gaps || data || []);
      }
    } catch {
      /* noop */
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchGaps();
  }, [fetchGaps]);

  const handleUpdate = async () => {
    if (!updateGap || !newStatus) return;
    try {
      await fetch(`/api/v1/audit/compliance-gaps/${updateGap.id}`, {
        method: "PATCH",
        headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
        body: JSON.stringify({ status: newStatus }),
      });
      setGaps((prev) => prev.map((g) => g.id === updateGap.id ? { ...g, status: newStatus as ComplianceGap["status"] } : g));
      setUpdateGap(null);
      setNewStatus("");
    } catch {
      /* noop */
    }
  };

  const filtered = filterStatus === "all" ? gaps : gaps.filter((g) => g.status === filterStatus);

  const summary = {
    open: gaps.filter((g) => g.status === "open").length,
    in_progress: gaps.filter((g) => g.status === "in_progress").length,
    remediated: gaps.filter((g) => g.status === "remediated").length,
    accepted_risk: gaps.filter((g) => g.status === "accepted_risk").length,
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><ShieldCheck className="w-6 h-6 text-green-500" /> {t("auditComplianceGaps.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Track and remediate compliance control gaps across frameworks.</p>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {(["open", "in_progress", "remediated", "accepted_risk"] as const).map((s) => (
          <div key={s} className="rounded-lg border p-4 dark:border-gray-800 cursor-pointer hover:border-blue-400" onClick={() => setFilterStatus(filterStatus === s ? "all" : s)}>
            <span className="text-sm text-gray-500 capitalize">{s.replace("_", " ")}</span>
            <p className={`text-2xl font-bold mt-1`}>
              <span className={`inline-block px-2 py-0.5 rounded ${statusColors[s]}`}>{summary[s]}</span>
            </p>
          </div>
        ))}
      </div>

      {/* Filter */}
      <div className="flex items-center gap-3">
        <select aria-label="Filter" value={filterStatus} onChange={(e) => setFilterStatus(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
          <option value="all">All Statuses</option>
          <option value="open">Open</option>
          <option value="in_progress">In Progress</option>
          <option value="remediated">Remediated</option>
          <option value="accepted_risk">Accepted Risk</option>
        </select>
      </div>

      {/* Gap table */}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-900/50">
            <tr>
              <th scope="col" className="px-4 py-3 text-left font-medium">Control ID</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Framework</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Description</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Severity</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Owner</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Due Date</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Status</th>
              <th scope="col" className="px-4 py-3 text-left font-medium">Action</th>
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-gray-800">
            {filtered.map((gap) => (
              <tr key={gap.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3 font-mono text-xs">{gap.control_id}</td>
                <td className="px-4 py-3">{gap.framework}</td>
                <td className="px-4 py-3 max-w-xs truncate" title={gap.description}>{gap.description}</td>
                <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${severityColors[gap.severity]}`}>{gap.severity}</span></td>
                <td className="px-4 py-3 flex items-center gap-1"><User className="w-3 h-3 text-gray-400" />{gap.owner}</td>
                <td className="px-4 py-3 flex items-center gap-1"><Calendar className="w-3 h-3 text-gray-400" />{gap.due_date}</td>
                <td className="px-4 py-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColors[gap.status]}`}>{gap.status.replace("_", " ")}</span></td>
                <td className="px-4 py-3">
                  <button onClick={() => { setUpdateGap(gap); setNewStatus(gap.status); }} className="text-blue-600 hover:underline text-xs font-medium">Update</button>
                </td>
              </tr>
            ))}
            {filtered.length === 0 && !loading && (
              <tr><td colSpan={8} className="px-4 py-8 text-center text-gray-500">No compliance gaps found.</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Update modal */}
      {updateGap && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setUpdateGap(null)}>
          <div role="dialog" aria-modal="true" className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-lg w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-6 py-4 border-b dark:border-gray-800">
              <h3 className="font-semibold flex items-center gap-2"><AlertCircle className="w-5 h-5 text-blue-500" /> Update Gap Status</h3>
              <button onClick={() => setUpdateGap(null)} aria-label="Close"><X className="w-5 h-5 text-gray-400" /></button>
            </div>
            <div className="px-6 py-4 space-y-4">
              <div>
                <label className="text-sm font-medium">Control ID</label>
                <p className="text-sm text-gray-500 mt-1 font-mono">{updateGap.control_id}</p>
              </div>
              <div>
                <label className="text-sm font-medium">Description</label>
                <p className="text-sm text-gray-500 mt-1">{updateGap.description}</p>
              </div>
              <div>
                <label className="text-sm font-medium">Remediation Plan</label>
                <p className="text-sm text-gray-500 mt-1">{updateGap.remediation_plan}</p>
              </div>
              <div>
                <label className="text-sm font-medium">New Status</label>
                <select aria-label="new Status" value={newStatus} onChange={(e) => setNewStatus(e.target.value)} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-800 text-sm">
                  <option value="open">Open</option>
                  <option value="in_progress">In Progress</option>
                  <option value="remediated">Remediated</option>
                  <option value="accepted_risk">Accepted Risk</option>
                </select>
              </div>
            </div>
            <div className="flex justify-end gap-2 px-6 py-4 border-t dark:border-gray-800">
              <button onClick={() => setUpdateGap(null)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">Cancel</button>
              <button onClick={handleUpdate} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700">Save</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
