"use client";

import React, { useEffect, useState } from "react";
import { useApi } from "@/lib/api";
import { useTranslations } from "@/lib/i18n";
import {
  FileCheck, Loader2, AlertCircle, X, Download, Upload, Trash2, CheckCircle, XCircle, Clock,
} from "lucide-react";

interface ComplianceControl {
  id: string;
  framework: string;
  control_id: string;
  description: string;
  category: string;
  required: boolean;
  evidence_count: number;
  status: "compliant" | "partial" | "missing";
}

interface EvidenceArtifact {
  id: string;
  control_id: string;
  framework: string;
  name: string;
  type: string;
  status: "pending" | "collected" | "verified" | "expired";
  collected_at: string;
  expires_at: string;
  file_url: string;
}

const statusIcons: Record<string, React.ReactNode> = {
  compliant: <CheckCircle className="h-4 w-4 text-green-500" />,
  partial: <Clock className="h-4 w-4 text-yellow-500" />,
  missing: <XCircle className="h-4 w-4 text-red-500" />,
};

const artifactStatusColors: Record<string, string> = {
  pending: "text-gray-600 bg-gray-100 dark:bg-gray-700 dark:text-gray-400",
  collected: "text-blue-600 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400",
  verified: "text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400",
  expired: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
};

const frameworks = ["SOC2", "GDPR", "HIPAA", "ISO27001", "PCI-DSS"];

export default function ComplianceEvidencePage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [controls, setControls] = useState<ComplianceControl[]>([]);
  const [artifacts, setArtifacts] = useState<EvidenceArtifact[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [framework, setFramework] = useState("SOC2");
  const [selectedControl, setSelectedControl] = useState<string | null>(null);
  const [uploadName, setUploadName] = useState("");
  const [uploading, setUploading] = useState(false);

  const loadData = async (fw: string) => {
    setLoading(true);
    try {
      const [c, a] = await Promise.all([
        apiFetch<ComplianceControl[]>(`/api/v1/audit/compliance-evidence/controls?framework=${fw}`).catch(() => []),
        apiFetch<EvidenceArtifact[]>(`/api/v1/audit/compliance-evidence/artifacts?framework=${fw}`).catch(() => []),
      ]);
      setControls(c); setArtifacts(a);
    } catch { setError("Failed to load evidence data"); }
    finally { setLoading(false); }
  };

  useEffect(() => { loadData(framework); });

  const handleFrameworkChange = (fw: string) => { setFramework(fw); setSelectedControl(null); loadData(fw); };

  const handleUpload = async () => {
    if (!selectedControl || !uploadName) return;
    setUploading(true);
    try {
      await apiFetch("/api/v1/audit/compliance-evidence/artifacts", { method: "POST", body: JSON.stringify({ control_id: selectedControl, name: uploadName, data: btoa(uploadName) }) });
      setUploadName("");
      await loadData(framework);
    } catch { setError("Upload failed"); }
    finally { setUploading(false); }
  };

  const handleDelete = async (id: string) => {
    try { await apiFetch(`/api/v1/audit/compliance-evidence/artifacts/${id}`, { method: "DELETE" }); await loadData(framework); }
    catch { setError("Delete failed"); }
  };

  const handleExport = async () => {
    try {
      const resp = await fetch(`/api/v1/audit/compliance-evidence/export?framework=${framework}`, { headers: { Authorization: `Bearer ${localStorage.getItem("ggid_access_token") || ""}` } });
      if (!resp.ok) throw new Error("Export failed");
      const blob = await resp.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a"); a.href = url; a.download = `evidence-${framework}.zip`; a.click(); URL.revokeObjectURL(url);
    } catch { setError("Export failed"); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const fwControls = selectedControl ? controls.filter((c) => c.id === selectedControl) : controls;
  const fwArtifacts = selectedControl ? artifacts.filter((a) => a.control_id === selectedControl) : artifacts;
  const stats = { compliant: controls.filter((c) => c.status === "compliant").length, partial: controls.filter((c) => c.status === "partial").length, missing: controls.filter((c) => c.status === "missing").length };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><FileCheck className="h-6 w-6 text-emerald-600" />{t("complianceEvidence.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Evidence collection and artifact management for compliance frameworks.</p>
        </div>
        <button onClick={handleExport} className="flex items-center gap-2 rounded-lg bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700"><Download className="h-4 w-4" /> Export</button>
      </div>

      {/* Framework selector */}
      <div className="flex gap-2">
        {frameworks.map((fw) => (
          <button key={fw} onClick={() => handleFrameworkChange(fw)} className={`rounded-lg px-4 py-2 text-sm font-medium ${framework === fw ? "bg-indigo-600 text-white" : "bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-300"}`}>{fw}</button>
        ))}
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-emerald-600" /></div>
      : (
        <>
          {/* Stats */}
          <div className="grid grid-cols-3 gap-4">
            <div className={cardCls}><div className="flex items-center gap-2"><CheckCircle className="h-4 w-4 text-green-500" /><span className="text-xs font-semibold uppercase text-gray-400">Compliant</span></div><p className="mt-2 text-2xl font-bold text-green-600">{stats.compliant}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><Clock className="h-4 w-4 text-yellow-500" /><span className="text-xs font-semibold uppercase text-gray-400">Partial</span></div><p className="mt-2 text-2xl font-bold text-yellow-600">{stats.partial}</p></div>
            <div className={cardCls}><div className="flex items-center gap-2"><XCircle className="h-4 w-4 text-red-500" /><span className="text-xs font-semibold uppercase text-gray-400">Missing</span></div><p className="mt-2 text-2xl font-bold text-red-600">{stats.missing}</p></div>
          </div>

          {/* Control checklist */}
          <div>
            <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">Control Checklist — {framework}</h2>
            <div className="space-y-2">
              {fwControls.map((c) => (
                <div key={c.id} className={`${cardCls} cursor-pointer`} onClick={() => setSelectedControl(selectedControl === c.id ? null : c.id)}>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      {statusIcons[c.status]}
                      <div>
                        <span className="font-mono text-sm font-semibold text-gray-900 dark:text-white">{c.control_id}</span>
                        <span className="ml-2 text-sm text-gray-500">{c.description}</span>
                      </div>
                    </div>
                    <div className="flex items-center gap-3">
                      {c.required && <span className="rounded bg-red-100 px-1.5 py-0.5 text-xs text-red-600 dark:bg-red-900/30">Required</span>}
                      <span className="text-xs text-gray-400">{c.evidence_count} artifacts</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Upload section */}
          {selectedControl && (
            <div className="flex items-center gap-2">
              <input aria-label="Artifact name" value={uploadName} onChange={(e) => setUploadName(e.target.value)} placeholder="Artifact name" className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200" />
              <button onClick={handleUpload} disabled={!uploadName || uploading} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{uploading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />} Upload</button>
            </div>
          )}

          {/* Artifacts table */}
          <div>
            <h2 className="mb-3 text-sm font-semibold uppercase text-gray-500">Evidence Artifacts {selectedControl && "(filtered)"}</h2>
            {fwArtifacts.length === 0 ? (
              <div className={cardCls}><div className="py-8 text-center"><FileCheck className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No artifacts collected.</p></div></div>
            ) : (
              <div className="overflow-x-auto rounded-xl border border-gray-200 dark:border-gray-700">
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 dark:bg-gray-800"><tr>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Name</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Control</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Type</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Status</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Collected</th>
                    <th scope="col" className="px-4 py-3 text-left font-semibold text-gray-600 dark:text-gray-300">Expires</th>
                    <th scope="col" className="px-4 py-3 text-right font-semibold text-gray-600 dark:text-gray-300">Actions</th>
                  </tr></thead>
                  <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                    {fwArtifacts.map((a) => (
                      <tr key={a.id} className="bg-white dark:bg-gray-900">
                        <td className="px-4 py-3 font-medium text-gray-900 dark:text-white">{a.name}</td>
                        <td className="px-4 py-3 font-mono text-xs text-gray-500">{a.control_id}</td>
                        <td className="px-4 py-3 text-gray-500">{a.type}</td>
                        <td className="px-4 py-3"><span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${artifactStatusColors[a.status] || ""}`}>{a.status}</span></td>
                        <td className="px-4 py-3 text-gray-400">{a.collected_at ? new Date(a.collected_at).toLocaleDateString() : "—"}</td>
                        <td className="px-4 py-3 text-gray-400">{a.expires_at ? new Date(a.expires_at).toLocaleDateString() : "—"}</td>
                        <td className="px-4 py-3 text-right"><button onClick={() => handleDelete(a.id)} className="text-red-400 hover:text-red-600"><Trash2 className="h-4 w-4" /></button></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
