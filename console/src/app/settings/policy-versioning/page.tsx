"use client";
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect, useCallback } from "react";
import { GitBranch, RotateCcw, Clock, ArrowRight } from "lucide-react";
interface Version { version_num: number; author: string; timestamp: string; change_summary: string; is_current: boolean; }
interface PolicyVersionData { policy_id: string; policy_name: string; versions: Version[]; }
export default function PolicyVersioningPage() {
  const [policies] = useState([{ id: "p1", name: "Data Access Policy" }, { id: "p2", name: "Admin Access Policy" }]);
  const [policyId, setPolicyId] = useState("");
  const [data, setData] = useState<PolicyVersionData | null>(null);
  const [loading, setLoading] = useState(false);
  const [compareA, setCompareA] = useState("");
  const [compareB, setCompareB] = useState("");
  const [showRollback, setShowRollback] = useState<number | null>(null);
  const t = useTranslations();
  const fetchData = useCallback(async () => { if (!policyId) return; setLoading(true); try { const res = await fetch("/api/v1/policy/" + policyId + "/versions", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); } catch { /* noop */ } finally { setLoading(false); } }, [policyId]);
  useEffect(() => { fetchData(); }, [fetchData]);
  const activate = async (versionNum: number) => { try { await fetch("/api/v1/policy/" + policyId + "/versions/" + versionNum + "/activate", { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); fetchData(); } catch { /* noop */ } };
  const rollback = async (versionNum: number) => { try { await fetch("/api/v1/policy/" + policyId + "/versions/" + versionNum + "/rollback", { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); setShowRollback(null); fetchData(); } catch { /* noop */ } };
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><GitBranch className="w-6 h-6 text-purple-500" /> {t("policyVersioning.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("policyVersioning.subtitle")}</p></div>
      <select value={policyId} onChange={(e) => setPolicyId(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">{t("policyVersioning.selectPolicy")}</option>{policies.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}</select>
      {data && (<>
        <div className="relative pl-8"><div className="absolute left-3 top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-800" /><div className="space-y-3">{data.versions.map((v) => (<div key={v.version_num} className="relative"><div className={"absolute -left-5 w-4 h-4 rounded-full border-2 " + (v.is_current ? "bg-green-500 border-green-200" : "bg-gray-300 border-gray-100 dark:bg-gray-700 dark:border-gray-800")} /><div className={"rounded-lg border p-3 ml-2 " + (v.is_current ? "border-green-300 dark:border-green-800" : "dark:border-gray-800")}><div className="flex items-center justify-between"><div className="flex items-center gap-2"><span className="font-bold text-sm">v{v.version_num}</span>{v.is_current && <span className="px-2 py-0.5 rounded text-xs bg-green-100 dark:bg-green-900/30 dark:text-green-400">{t("policyVersioning.current")}</span>}</div><span className="text-xs text-gray-400">{v.timestamp}</span></div><p className="text-sm text-gray-500 mt-1">{v.change_summary}</p><div className="flex items-center gap-3 mt-2"><span className="text-xs text-gray-400">by {v.author}</span>{!v.is_current && <><button onClick={() => activate(v.version_num)} className="text-xs text-blue-600 hover:underline">{t("policyVersioning.activate")}</button><button onClick={() => setShowRollback(v.version_num)} className="text-xs text-orange-600 hover:underline flex items-center gap-1"><RotateCcw className="w-3 h-3" /> {t("policyVersioning.rollback")}</button></>}</div></div></div>))}</div></div>
        {data.versions.length >= 2 && (<div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">{t("policyVersioning.compareVersions")}</h3><div className="flex items-center gap-2 mb-3"><select value={compareA} onChange={(e) => setCompareA(e.target.value)} className="px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs">{data.versions.map((v) => <option key={v.version_num} value={v.version_num}>v{v.version_num}</option>)}</select><ArrowRight className="w-4 h-4 text-gray-400" /><select value={compareB} onChange={(e) => setCompareB(e.target.value)} className="px-2 py-1.5 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs">{data.versions.map((v) => <option key={v.version_num} value={v.version_num}>v{v.version_num}</option>)}</select></div>{compareA && compareB && compareA !== compareB && <div className="text-sm space-y-1"><div className="flex items-center gap-2"><span className="text-red-500 line-through">Old: rule was deny</span></div><div className="flex items-center gap-2"><span className="text-green-500">New: rule is allow</span></div></div>}</div>)}
        {showRollback !== null && (<div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowRollback(null)}><div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-sm w-full mx-4" onClick={(e) => e.stopPropagation()}><div className="px-6 py-4"><h3 className="font-semibold flex items-center gap-2"><RotateCcw className="w-5 h-5 text-orange-500" /> {t("policyVersioning.confirmRollback")}</h3><p className="text-sm text-gray-500 mt-2">Rollback to v{showRollback}? This will make v{showRollback} the current version.</p><div className="flex justify-end gap-2 mt-4"><button onClick={() => setShowRollback(null)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm">{t("policyVersioning.cancel")}</button><button onClick={() => rollback(showRollback)} className="px-4 py-2 rounded-lg bg-orange-600 text-white text-sm font-medium">{t("policyVersioning.rollback")}</button></div></div></div></div>)}
      </>)}
      {!data && !loading && policyId && <p className="text-sm text-gray-500 text-center py-8">{t("policyVersioning.loading")}</p>}
    </div>
  );
}
