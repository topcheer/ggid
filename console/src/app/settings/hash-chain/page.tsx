"use client";
import { useState, useEffect, useCallback } from "react";
import { Link2, ShieldCheck, AlertTriangle, RefreshCw } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface TamperAlert { block_num: number; expected_hash: string; actual_hash: string; detected_at: string; }
interface VerifyEntry { timestamp: string; status: "ok" | "failed"; blocks_checked: number; }
interface HashChainData { chain_status: "intact" | "broken"; last_verified_at: string; total_blocks: number; integrity_score: number; tamper_alerts: TamperAlert[]; verify_log: VerifyEntry[]; }
export default function HashChainPage() {
  const t = useTranslations();

  const [data, setData] = useState<HashChainData | null>(null);
  const [loading, setLoading] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const fetchData = useCallback(async () => { setLoading(true); try { const res = await fetch("/api/v1/audit/hash-chain", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); } catch { /* noop */ } finally { setLoading(false); } }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  const verify = async () => { setVerifying(true); try { await fetch("/api/v1/audit/hash-chain/verify", { method: "POST", headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); fetchData(); } catch { /* noop */ } finally { setVerifying(false); } };
  const scoreColor = data ? (data.integrity_score >= 99 ? "#10b981" : data.integrity_score >= 90 ? "#f59e0b" : "#ef4444") : "#3b82f6";
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between"><div><h1 className="text-2xl font-bold flex items-center gap-2"><Link2 className="w-6 h-6 text-blue-500" /> {t("big1.hashChain.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("big1.hashChain.auditLogIntegrityVerificationViaHashChain")}</p></div><button onClick={verify} disabled={verifying} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium flex items-center gap-2"><RefreshCw className={"w-4 h-4 " + (verifying ? "animate-spin" : "")} /> {verifying ? t("big1.hashChain.verifying") : t("big1.hashChain.reVerify")}</button></div>
      {data && (<>
        <div className={"rounded-lg border-2 p-4 flex items-center gap-3 " + (data.chain_status === "intact" ? "border-green-300 dark:border-green-800 bg-green-50 dark:bg-green-900/20" : "border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20")}>{data.chain_status === t("big1.hashChain.intact") ? <ShieldCheck className="w-8 h-8 text-green-500" /> : <AlertTriangle className="w-8 h-8 text-red-500" />}<div><span className="text-sm text-gray-500">{t("big1.hashChain.chainStatus")}</span><p className={"text-lg font-bold " + (data.chain_status === "intact" ? "text-green-600" : "text-red-600")}>{data.chain_status === t("big1.hashChain.intact") ? t("big1.hashChain.intact") : t("big1.hashChain.broken")}</p></div><div className="ml-auto text-right"><span className="text-xs text-gray-500">{t("big1.hashChain.lastVerified")}</span><p className="text-xs text-gray-400">{data.last_verified_at}</p></div></div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-4"><div className="relative w-24 h-24"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={scoreColor} strokeWidth={6} strokeDasharray={(data.integrity_score / 100) * 176 + " 176"} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-2xl font-bold" style={{ color: scoreColor }}>{data.integrity_score}%</span><span className="text-[10px] text-gray-400">{t("big1.hashChain.integrity")}</span></div></div><div><span className="text-sm text-gray-500">{t("big1.hashChain.totalBlocks")}</span><p className="text-xl font-bold">{data.total_blocks.toLocaleString()}</p></div></div>
          <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">{t("big1.hashChain.verificationLog")}</h3><div className="space-y-1 max-h-32 overflow-y-auto">{data.verify_log.map((v, i) => (<div key={i} className="text-xs flex items-center gap-2"><span className={v.status === "ok" ? "text-green-600" : "text-red-600"}>{v.status}</span><span className="text-gray-400">{v.blocks_checked}{t("big1.hashChain.blocks")}</span><span className="text-gray-400 ml-auto">{v.timestamp}</span></div>))}</div></div>
        </div>
        {data.tamper_alerts.length > 0 && <div className="rounded-lg border border-red-300 dark:border-red-800 p-4"><h3 className="text-sm font-semibold text-red-700 dark:text-red-400 mb-2 flex items-center gap-2"><AlertTriangle className="w-4 h-4" />{t("big1.hashChain.tamperAlerts")}{data.tamper_alerts.length})</h3><div className="space-y-2">{data.tamper_alerts.map((a, i) => (<div key={i} className="rounded border dark:border-gray-800 p-2 text-xs"><div className="flex items-center justify-between"><span className="font-bold">{t("big1.hashChain.block")}{a.block_num}</span><span className="text-gray-400">{a.detected_at}</span></div><div className="mt-1 font-mono text-gray-500"><span className="text-red-500">{t("big1.hashChain.expected")}{a.expected_hash}</span></div><div className="font-mono text-gray-500"><span className="text-red-500">{t("big1.hashChain.actual")}{a.actual_hash}</span></div></div>))}</div></div>}
      </>)}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">{t("big1.hashChain.loading")}</p>}
    </div>
  );
}
