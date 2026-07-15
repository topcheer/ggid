"use client";
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect, useCallback } from "react";
import { Calendar, Clock, CheckCircle } from "lucide-react";
interface Milestone { id: string; name: string; status: "completed" | "in_progress" | "pending" | "overdue"; due_date: string; responsible_party: string; }
interface FrameworkData { framework: string; milestones: Milestone[]; }
const frameworks = ["SOC 2", "HIPAA", "ISO 27001", "GDPR"];
const statusColors: Record<string, string> = { completed: "text-green-600", in_progress: "text-blue-600", pending: "text-gray-500", overdue: "text-red-600" };
const statusBg: Record<string, string> = { completed: "bg-green-500", in_progress: "bg-blue-500", pending: "bg-gray-400", overdue: "bg-red-500" };
export default function ComplianceTimelinePage() {
  const t = useTranslations();
  const [tab, setTab] = useState("SOC 2");
  const [data, setData] = useState<FrameworkData | null>(null);
  const [loading, setLoading] = useState(false);
  const fetchData = useCallback(async () => {
    setLoading(true);
    try { const res = await fetch("/api/v1/audit/compliance-timeline?framework=" + encodeURIComponent(tab), { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); }
    catch { /* noop */ } finally { setLoading(false); }
  }, [tab]);
  useEffect(() => { fetchData(); }, [fetchData]);
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Calendar className="w-6 h-6 text-green-500" />{t("complianceTimeline.title")}</h1><p className="text-sm text-gray-500 mt-1">Track compliance milestones and deadlines across frameworks.</p></div>
      <div className="flex gap-2">{frameworks.map((f) => <button key={f} onClick={() => setTab(f)} className={"px-4 py-2 rounded-lg text-sm font-medium " + (tab === f ? "bg-green-600 text-white" : "border dark:border-gray-700")}>{f}</button>)}</div>
      {data && (<>
        <div className="relative pl-8"><div className="absolute left-3 top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-800" /><div className="space-y-4">{data.milestones.map((m) => (<div key={m.id} className="relative"><div className={"absolute -left-5 w-4 h-4 rounded-full border-2 border-white " + statusBg[m.status]} /><div className="rounded-lg border dark:border-gray-800 p-3 ml-2"><div className="flex items-center justify-between"><span className="font-medium text-sm">{m.name}</span><span className={"text-xs font-medium " + statusColors[m.status]}>{m.status.replace("_", " ")}</span></div><div className="flex items-center gap-3 mt-1 text-xs text-gray-500"><span className="flex items-center gap-1"><Clock className="w-3 h-3" />{m.due_date}</span><span>Owner: {m.responsible_party}</span></div></div></div>))}</div></div>
      </>)}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
