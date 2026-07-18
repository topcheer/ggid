"use client";
import { useState, useCallback } from "react";
import { Globe, Search, ShieldAlert, CheckCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface IpInfo { ip: string; reputation_score: number; threat_tags: string[]; first_seen: string; last_seen: string; country: string; city: string; isp: string; associated_events: number; blacklisted: boolean; }

export default function IpReputationPage() {
  const t = useTranslations();

  const [query, setQuery] = useState("");
  const [info, setInfo] = useState<IpInfo | null>(null);
  const [loading, setLoading] = useState(false);

  const search = useCallback(async () => {
    if (!query) return;
    setLoading(true);
    try {
      const res = await fetch("/api/v1/audit/ip-reputation?ip=" + encodeURIComponent(query), { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setInfo(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [query]);

  const scoreColor = info ? (info.reputation_score <= 30 ? "#10b981" : info.reputation_score <= 60 ? "#f59e0b" : "#ef4444") : "#3b82f6";

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Globe className="w-6 h-6 text-blue-500" /> {t("big1.ipReputation.title")}</h1><p className="text-sm text-gray-500 mt-1">{t("big1.ipReputation.lookUpReputationScoresAndThreatIntelligenceForIPAddresses")}</p></div>

      <div className="flex gap-2"><div className="relative flex-1 max-w-md"><Search className="absolute left-2 top-2.5 w-4 h-4 text-gray-400" /><input aria-label="192.168.1.1" type="text" value={query} onChange={(e) => setQuery(e.target.value)} onKeyDown={(e) => e.key === "Enter" && search()} placeholder="192.168.1.1" className="w-full pl-8 pr-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div><button onClick={search} disabled={loading || !query} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium disabled:opacity-50" aria-label="Action">{t("big1.ipReputation.search")}</button></div>

      {info && (<>
        {info.blacklisted && <div className="rounded-lg border border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-3 flex items-center gap-2"><ShieldAlert className="w-5 h-5 text-red-500" /><span className="font-semibold text-red-700 dark:text-red-400">{t("big1.ipReputation.thisIpIsBlacklisted")}</span></div>}
        <div className="flex items-center gap-6">
          <div className="relative w-28 h-28"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={scoreColor} strokeWidth={6} strokeDasharray={(info.reputation_score / 100) * 176 + " 176"} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-2xl font-bold" style={{ color: scoreColor }}>{info.reputation_score}</span><span className="text-[10px] text-gray-400">{t("big1.ipReputation.risk")}</span></div></div>
          <div className="space-y-1 text-sm"><div className="flex items-center gap-2"><span className="text-gray-500">{t("big1.ipReputation.ip")}</span><span className="font-mono font-medium">{info.ip}</span></div><div className="flex items-center gap-2"><span className="text-gray-500">{t("big1.ipReputation.location")}</span><span>{info.city}, {info.country}</span></div><div className="flex items-center gap-2"><span className="text-gray-500">{t("big1.ipReputation.isp")}</span><span>{info.isp}</span></div><div className="flex items-center gap-2"><span className="text-gray-500">{t("big1.ipReputation.events")}</span><span className="font-bold">{info.associated_events}</span></div><div className="flex items-center gap-2"><span className="text-gray-500">{t("big1.ipReputation.firstSeen")}</span><span className="text-xs">{info.first_seen}</span></div></div>
        </div>
        {info.threat_tags.length > 0 && (<div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">{t("big1.ipReputation.threatTags")}</h3><div className="flex flex-wrap gap-2">{info.threat_tags.map((t: any) => (<span key={t} className="px-2 py-1 rounded text-xs bg-red-100 dark:bg-red-900/30 dark:text-red-400 font-medium">{t}</span>))}</div></div>)}
        {!info.blacklisted && info.threat_tags.length === 0 && (<div className="rounded-lg border border-green-300 dark:border-green-800 bg-green-50 dark:bg-green-900/20 p-3 flex items-center gap-2"><CheckCircle className="w-5 h-5 text-green-500" /><span className="text-sm text-green-700 dark:text-green-400">{t("big1.ipReputation.noThreatsDetectedForThisIp")}</span></div>)}
      </>)}
    </div>
  );
}
