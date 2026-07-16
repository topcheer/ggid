"use client";
import { useState, useEffect, useCallback } from "react";
import { Activity, Sliders } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
interface Signal { name: string; weight: number; }
interface ScoreEntry { username: string; score: number; top_signal: string; last_event: string; }
interface ModelStats { accuracy: number; precision: number; recall: number; false_positive_rate: number; }
interface Data { signals: Signal[]; thresholds: { low: number; medium: number; high: number; critical: number }; distribution: { bucket: string; count: number }[]; top_users: ScoreEntry[]; model_stats: ModelStats; }
export default function AnomalyScoringPage() {
  const t = useTranslations();

  const [data, setData] = useState<Data | null>(null);
  const [loading, setLoading] = useState(false);
  const fetchData = useCallback(async () => { setLoading(true); try { const res = await fetch("/api/v1/auth/anomaly-scoring", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); } catch { /* noop */ } finally { setLoading(false); } }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  const maxBucket = Math.max(...(data?.distribution.map((d) => d.count) || [1]), 1);
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Activity className="w-6 h-6 text-orange-500" /> {t("anomalyScoring.title")}</h1><p className="text-sm text-gray-500 mt-1">Configure risk-based anomaly detection scoring model.</p></div>
      {data && (<>
        <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><Sliders className="w-4 h-4 text-gray-400" /> Signal Weights</h3><div className="space-y-3">{data.signals.map((s, i) => (<div key={s.name} className="flex items-center gap-3"><span className="text-sm w-28 capitalize">{s.name}</span><input aria-label="Input field" type="range" min={0} max={50} defaultValue={s.weight} className="flex-1" /><span className="text-sm font-bold w-10 text-right">{s.weight}</span></div>))}</div></div>
        <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Thresholds</h3><div className="grid grid-cols-4 gap-3">{(["low", "medium", "high", "critical"] as const).map((level) => (<div key={level} className="text-center"><label className="text-xs text-gray-500 capitalize">{level}</label><div className="mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 text-sm font-bold">{data.thresholds[level]}</div></div>))}</div></div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Score Distribution</h3><div className="flex items-end gap-1 h-24">{data.distribution.map((d, i) => (<div key={i} className="flex-1 bg-orange-400 dark:bg-orange-500 rounded-t" style={{ height: (d.count / maxBucket) * 100 + "%", minHeight: "2px" }} title={d.bucket + ": " + d.count} />))}</div></div>
          <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Model Accuracy</h3><div className="grid grid-cols-2 gap-2 text-sm"><div><span className="text-xs text-gray-500">Accuracy</span><p className="font-bold">{data.model_stats.accuracy.toFixed(1)}%</p></div><div><span className="text-xs text-gray-500">Precision</span><p className="font-bold">{data.model_stats.precision.toFixed(1)}%</p></div><div><span className="text-xs text-gray-500">Recall</span><p className="font-bold">{data.model_stats.recall.toFixed(1)}%</p></div><div><span className="text-xs text-gray-500">False Positive Rate</span><p className="font-bold text-red-600">{data.model_stats.false_positive_rate.toFixed(1)}%</p></div></div></div>
        </div>
        <div className="overflow-x-auto rounded-lg border dark:border-gray-800"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">User</th><th className="px-4 py-3 text-left font-medium">Score</th><th className="px-4 py-3 text-left font-medium">Top Signal</th><th className="px-4 py-3 text-left font-medium">Last Event</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{data.top_users.map((u, i) => (<tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-medium">{u.username}</td><td className="px-4 py-3"><span className={"font-bold " + (u.score >= 70 ? "text-red-600" : u.score >= 40 ? "text-orange-600" : "text-yellow-600")}>{u.score}</span></td><td className="px-4 py-3 text-xs font-mono">{u.top_signal}</td><td className="px-4 py-3 text-xs text-gray-400">{u.last_event}</td></tr>))}</tbody></table></div>
      </>)}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
