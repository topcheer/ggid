"use client";
import { useState, useEffect, useCallback } from "react";
import { FolderCheck, Upload, AlertTriangle, CheckCircle, Clock } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface Control { control_id: string; description: string; evidence_required: boolean; evidence_type: "doc" | "screenshot" | "log" | "config"; collection_status: "collected" | "pending" | "overdue"; last_collected: string | null; reviewer: string | null; }
interface FrameworkData { framework: string; controls: Control[]; }
const statusConfig: Record<string, { color: string; icon: typeof CheckCircle }> = { collected: { color: "text-green-600", icon: CheckCircle }, pending: { color: "text-yellow-600", icon: Clock }, overdue: { color: "text-red-600", icon: AlertTriangle } };
const frameworks = ["SOC 2", "ISO 27001", "GDPR", "HIPAA"];
export default function EvidenceCollectionPage() {
  const t = useTranslations();

  const [framework, setFramework] = useState("SOC 2");
  const [data, setData] = useState<FrameworkData | null>(null);
  const [loading, setLoading] = useState(false);
  const fetchData = useCallback(async () => { setLoading(true); try { const res = await fetch("/api/v1/audit/evidence-collection?framework=" + encodeURIComponent(framework), { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); } catch { /* noop */ } finally { setLoading(false); } }, [framework]);
  useEffect(() => { fetchData(); }, [fetchData]);
  const upload = async (controlId: string) => { try { await fetch("/api/v1/audit/evidence-collection/" + controlId + "/upload", { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); fetchData(); } catch { /* noop */ } };
  const collected = data?.controls.filter((c) => c.collection_status === "collected").length || 0;
  const pending = data?.controls.filter((c) => c.collection_status === "pending").length || 0;
  const overdue = data?.controls.filter((c) => c.collection_status === "overdue").length || 0;
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><FolderCheck className="w-6 h-6 text-blue-500" /> {t("evidenceCollection.title")}</h1><p className="text-sm text-gray-500 mt-1">Track compliance evidence collection per control.</p></div>
      <div className="flex items-center gap-3"><select value={framework} onChange={(e) => setFramework(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">{frameworks.map((f) => <option key={f} value={f}>{f}</option>)}</select>{data && <span className="text-sm text-gray-500">{collected}/{data.controls.length} collected, {pending} pending, {overdue} overdue</span>}</div>
      {data && overdue > 0 && <div className="rounded-lg border border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-3 flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-red-500" /><span className="font-semibold text-red-700 dark:text-red-400">{overdue} controls overdue for evidence collection</span></div>}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Control</th><th className="px-4 py-3 text-left font-medium">Evidence Type</th><th className="px-4 py-3 text-left font-medium">Status</th><th className="px-4 py-3 text-left font-medium">Last Collected</th><th className="px-4 py-3 text-left font-medium">Reviewer</th><th className="px-4 py-3 text-left font-medium">Action</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{data?.controls.map((c) => { const cfg = statusConfig[c.collection_status]; const Icon = cfg.icon; return (<tr key={c.control_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><span className="font-mono text-xs font-medium">{c.control_id}</span><p className="text-xs text-gray-400">{c.description}</p></td><td className="px-4 py-3"><span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{c.evidence_type}</span></td><td className="px-4 py-3"><span className={"flex items-center gap-1 text-xs " + cfg.color}><Icon className="w-3.5 h-3.5" /> {c.collection_status}</span></td><td className="px-4 py-3 text-xs text-gray-500">{c.last_collected || "-"}</td><td className="px-4 py-3 text-xs">{c.reviewer || "-"}</td><td className="px-4 py-3"><button onClick={() => upload(c.control_id)} aria-label={`Upload evidence for ${c.control_id}`} className="text-xs text-blue-600 hover:underline flex items-center gap-1"><Upload className="w-3 h-3" /> Upload</button></td></tr>); })}</tbody></table></div>
    </div>
  );
}
