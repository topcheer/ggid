"use client";
import { useState, useEffect, useCallback } from "react";
import { ShieldCheck, Clock } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface VerifConfig { methods: { document: boolean; face: boolean; kba: boolean; phone: boolean; email: boolean }; required_factors: number; confidence_threshold: number; risk_matrix: { level: string; factors: number }[]; }
interface VerifEvent { id: string; user: string; method: string; status: string; confidence: number; timestamp: string; }

export default function IdentityVerificationPage() {
  const [config, setConfig] = useState<VerifConfig | null>(null);
  const [history, setHistory] = useState<VerifEvent[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/identity-verification", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setConfig(d.config || d); setHistory(d.history || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  const t = useTranslations();

  if (!config) return <p className="text-sm text-gray-500 text-center py-8">{t("idVerification.loading")}</p>;

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><ShieldCheck className="w-6 h-6 text-green-500" /> Identity Verification</h1><p className="text-sm text-gray-500 mt-1">{t("idVerification.subtitle")}</p></div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">{t("idVerification.title")}</h3><div className="grid grid-cols-2 md:grid-cols-5 gap-2">{Object.entries(config.methods).map(([key, val]) => (<label key={key} className="flex items-center gap-2 text-sm rounded-lg border dark:border-gray-700 p-2"><input aria-label="Val" type="checkbox" checked={val} onChange={(e) => setConfig({ ...config, methods: { ...config.methods, [key]: e.target.checked } })} className="rounded" /> <span className="capitalize">{key}</span></label>))}</div></div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="rounded-lg border dark:border-gray-800 p-4"><label className="text-sm font-medium">{t("idVerification.requiredFactors")}</label><input aria-label="config" type="range" min={1} max={5} value={config.required_factors} onChange={(e) => setConfig({ ...config, required_factors: parseInt(e.target.value) })} className="w-full mt-2" /><span className="text-lg font-bold">{config.required_factors}</span></div>
        <div className="rounded-lg border dark:border-gray-800 p-4"><label className="text-sm font-medium">{t("idVerification.confidenceThreshold")}</label><input aria-label="config" type="range" min={50} max={100} value={config.confidence_threshold} onChange={(e) => setConfig({ ...config, confidence_threshold: parseInt(e.target.value) })} className="w-full mt-2" /><span className="text-lg font-bold">{config.confidence_threshold}%</span></div>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">{t("idVerification.perRisk")}</h3><div className="grid grid-cols-4 gap-2">{config.risk_matrix.map((r) => (<div key={r.level} className="rounded-lg border dark:border-gray-700 p-3 text-center"><div className={"text-sm font-medium capitalize " + (r.level === "critical" ? "text-red-600" : r.level === "high" ? "text-orange-600" : r.level === "medium" ? "text-yellow-600" : "text-green-600")}>{r.level}</div><div className="text-2xl font-bold mt-1">{r.factors}</div><div className="text-xs text-gray-500">factors</div></div>))}</div></div>

      <div className="overflow-x-auto rounded-lg border dark:border-gray-800"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">{t("idVerification.user")}</th><th className="px-4 py-3 text-left font-medium">{t("idVerification.method")}</th><th className="px-4 py-3 text-left font-medium">{t("idVerification.confidence")}</th><th className="px-4 py-3 text-left font-medium">{t("idVerification.status")}</th><th className="px-4 py-3 text-left font-medium">{t("idVerification.time")}</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{history.map((e) => (<tr key={e.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-medium">{e.user}</td><td className="px-4 py-3 text-xs">{e.method}</td><td className="px-4 py-3"><span className={"text-xs font-bold " + (e.confidence >= config.confidence_threshold ? "text-green-600" : "text-red-600")}>{e.confidence}%</span></td><td className="px-4 py-3"><span className={"text-xs " + (e.status === "verified" ? "text-green-600" : "text-red-600")}>{e.status}</span></td><td className="px-4 py-3 text-xs text-gray-400">{e.timestamp}</td></tr>))}{history.length === 0 && <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">{t("idVerification.noEvents")}</td></tr>}</tbody></table></div>
    </div>
  );
}
