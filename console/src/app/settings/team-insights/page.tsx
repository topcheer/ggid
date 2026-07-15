"use client";

import { useState, useEffect, useCallback } from "react";
import { Users, AlertTriangle, TrendingDown } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface TeamInsights {
  cohesion_score: number;
  collaboration_patterns: { team_a: string; team_b: string; frequency: number }[];
  silo_detection: { team: string; isolation_pct: number }[];
  cross_team_deps: { from: string; to: string; type: string }[];
  expertise_distribution: { team: string; skill: string; level: number }[];
  attrition_risk: { team: string; risk_level: "low" | "medium" | "high" }[];
}

const riskColors: Record<string, string> = {
  low: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400",
  medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
};

export default function TeamInsightsPage() {
  const t = useTranslations();

  const [data, setData] = useState<TeamInsights | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/identity/team-insights", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const scoreColor = data ? (data.cohesion_score >= 70 ? "#10b981" : data.cohesion_score >= 40 ? "#f59e0b" : "#ef4444") : "#3b82f6";
  const teams = data ? [...new Set(data.expertise_distribution.map((e) => e.team))] : [];
  const skills = data ? [...new Set(data.expertise_distribution.map((e) => e.skill))] : [];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Users className="w-6 h-6 text-teal-500" /> {t("teamInsights.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Organizational collaboration patterns, silos, and attrition risk analysis.</p>
      </div>

      {data && (
        <>
          <div className="flex items-center gap-6">
            <div className="relative w-28 h-28"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={scoreColor} strokeWidth={6} strokeDasharray={`${(data.cohesion_score / 100) * 176} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-2xl font-bold" style={{ color: scoreColor }}>{data.cohesion_score.toFixed(0)}</span><span className="text-[10px] text-gray-400">cohesion</span></div></div>
            <div className="flex-1 space-y-1">{data.silo_detection.filter((s) => s.isolation_pct > 50).map((s) => (<div key={s.team} className="flex items-center gap-2 text-sm"><AlertTriangle className="w-4 h-4 text-red-500" /><span className="font-medium">{s.team}</span><span className="text-red-600">{s.isolation_pct}% isolated</span></div>))}</div>
          </div>

          <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Collaboration Heatmap</h3><div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-2">{data.collaboration_patterns.map((c, i) => { const intensity = Math.min(c.frequency / 100, 1); return (<div key={i} className="rounded p-2 text-xs" style={{ background: `rgba(16, 185, 129, ${intensity * 0.8 + 0.1})` }}><span className="font-medium block">{c.team_a} / {c.team_b}</span><span className="text-gray-600 dark:text-gray-300">{c.frequency} interactions</span></div>); })}</div></div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Cross-Team Dependencies</h3><div className="space-y-1">{data.cross_team_deps.map((d, i) => (<div key={i} className="flex items-center gap-2 text-sm"><span className="font-mono text-xs">{d.from}</span><span className="text-gray-400">{"->"}</span><span className="font-mono text-xs">{d.to}</span><span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800">{d.type}</span></div>))}{data.cross_team_deps.length === 0 && <p className="text-xs text-gray-400">None detected.</p>}</div></div>

            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><TrendingDown className="w-4 h-4 text-red-500" /> Attrition Risk</h3><div className="space-y-1">{data.attrition_risk.map((a) => (<div key={a.team} className="flex items-center gap-2 text-sm"><span className="flex-1">{a.team}</span><span className={`px-2 py-0.5 rounded text-xs ${riskColors[a.risk_level]}`}>{a.risk_level}</span></div>))}</div></div>
          </div>

          {skills.length > 0 && teams.length > 0 && (
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Expertise Distribution</h3><div className="space-y-2">{skills.map((skill) => (<div key={skill} className="flex items-center gap-2"><span className="text-xs text-gray-500 w-24">{skill}</span><div className="flex-1 flex items-center gap-1">{teams.map((team) => { const exp = data.expertise_distribution.find((e) => e.team === team && e.skill === skill); return <div key={team} className="flex-1"><div className="w-full bg-gray-100 dark:bg-gray-800 rounded-full h-4 overflow-hidden"><div className="h-full bg-teal-500 rounded-full" style={{ width: `${exp?.level ?? 0}%` }} /></div><span className="text-[9px] text-gray-400 block text-center mt-0.5 truncate">{team}</span></div>; })}</div></div>))}</div></div>
          )}
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
