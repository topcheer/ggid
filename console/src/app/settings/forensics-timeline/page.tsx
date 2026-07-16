"use client";

import { useState, useEffect, useCallback } from "react";
import { Fingerprint, ShieldCheck, AlertTriangle, ShieldAlert } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ForensicsData {
  hash_chain_verified: boolean;
  integrity_score: number;
  total_events: number;
  verified_events: number;
  tamper_evidence: { event_id: string; timestamp: string; type: string; description: string }[];
  insertion_gaps: { after_event: string; gap_duration: string; expected_events: number; actual_events: number }[];
  reorder_detected: { event_id: string; expected_seq: number; actual_seq: number }[];
}

export default function ForensicsTimelinePage() {
  const t = useTranslations();

  const [data, setData] = useState<ForensicsData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/audit/forensics-timeline", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const scoreColor = data ? (data.integrity_score >= 95 ? "#10b981" : data.integrity_score >= 80 ? "#f59e0b" : "#ef4444") : "#3b82f6";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Fingerprint className="w-6 h-6 text-purple-500" /> {t("forensicsTimeline.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Audit log integrity verification with hash chain and tamper detection.</p>
      </div>

      {data && (
        <>
          <div className="flex items-center gap-6">
            <div className="relative w-28 h-28"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={scoreColor} strokeWidth={6} strokeDasharray={`${(data.integrity_score / 100) * 176} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-2xl font-bold" style={{ color: scoreColor }}>{data.integrity_score.toFixed(1)}</span><span className="text-[10px] text-gray-400">integrity</span></div></div>
            <div className="space-y-2">
              <div className="flex items-center gap-2 text-sm">{data.hash_chain_verified ? <ShieldCheck className="w-5 h-5 text-green-500" /> : <ShieldAlert className="w-5 h-5 text-red-500" />}<span className={data.hash_chain_verified ? "text-green-600 font-medium" : "text-red-600 font-medium"}>Hash Chain {data.hash_chain_verified ? "Verified" : "BROKEN"}</span></div>
              <div className="text-sm text-gray-500">{data.verified_events} / {data.total_events} events verified</div>
            </div>
          </div>

          {data.tamper_evidence.length > 0 && (
            <div className="rounded-lg border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-4"><h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><AlertTriangle className="w-4 h-4 text-red-500" /> Tamper Evidence Detected</h3><div className="space-y-1">{data.tamper_evidence.map((t) => (<div key={t.event_id} className="flex items-center gap-2 text-sm"><span className="text-xs text-gray-400">{t.timestamp}</span><span className="font-mono text-xs text-red-600">{t.event_id}</span><span>{t.description}</span></div>))}</div></div>
          )}

          {data.insertion_gaps.length > 0 && (
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><AlertTriangle className="w-4 h-4 text-yellow-500" /> Insertion Gaps</h3><div className="space-y-2">{data.insertion_gaps.map((g, i) => (<div key={i} className="flex items-center gap-3 text-sm border dark:border-gray-800 rounded p-2"><AlertTriangle className="w-4 h-4 text-yellow-500" /><span className="text-gray-500 text-xs">After:</span><span className="font-mono text-xs">{g.after_event}</span><span className="text-gray-500 text-xs ml-auto">Gap: {g.gap_duration}</span><span className="px-2 py-0.5 rounded text-xs bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400">Expected: {g.expected_events} / Actual: {g.actual_events}</span></div>))}</div></div>
          )}

          {data.reorder_detected.length > 0 && (
            <div className="rounded-lg border border-orange-200 dark:border-orange-800 bg-orange-50 dark:bg-orange-900/20 p-4"><h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><AlertTriangle className="w-4 h-4 text-orange-500" /> Event Reorder Detected</h3><div className="space-y-1">{data.reorder_detected.map((r, i) => (<div key={i} className="flex items-center gap-2 text-sm"><span className="font-mono text-xs">{r.event_id}</span><span className="text-gray-500 text-xs">expected seq: {r.expected_seq}</span><span className="text-orange-600 text-xs font-bold">actual: {r.actual_seq}</span></div>))}</div></div>
          )}

          {data.tamper_evidence.length === 0 && data.insertion_gaps.length === 0 && data.reorder_detected.length === 0 && data.hash_chain_verified && (
            <div className="rounded-lg border border-green-300 dark:border-green-800 bg-green-50 dark:bg-green-900/20 p-4 flex items-center gap-3"><ShieldCheck className="w-8 h-8 text-green-500" /><span className="font-semibold text-green-700 dark:text-green-400">All integrity checks passed. No anomalies detected.</span></div>
          )}
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
