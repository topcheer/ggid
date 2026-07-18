"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { TrendingUp, TrendingDown, Award, Calendar } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface ScoreEntry {
  month: string;
  score: number;
  gap_count: number;
  delta: number;
}

interface FrameworkScore {
  framework: string;
  current_score: number;
  history: ScoreEntry[];
}

const frameworks = ["SOC 2", "ISO 27001", "GDPR", "HIPAA", "PCI DSS", "NIST CSF"];

export default function ComplianceScorePage() {
  const t = useTranslations();
  const [selectedFramework, setSelectedFramework] = useState("SOC 2");
  const [data, setData] = useState<FrameworkScore | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchScore = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/audit/compliance-score?framework=${encodeURIComponent(selectedFramework)}`, { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch {
      /* noop */
    } finally {
      setLoading(false);
    }
  }, [selectedFramework]);

  useEffect(() => {
    fetchScore();
  }, [fetchScore]);

  // Build chart coordinates
  const chartWidth = 600;
  const chartHeight = 200;
  const padding = 40;
  const scores = data?.history.map((h) => h.score) || [];
  const minScore = Math.min(...scores, 0);
  const maxScore = Math.max(...scores, 100);
  const xStep = data && data.history.length > 1 ? (chartWidth - padding * 2) / (data.history.length - 1) : 0;

  const getY = (score: number) => chartHeight - padding - ((score - minScore) / (maxScore - minScore || 1)) * (chartHeight - padding * 2);
  const getX = (i: number) => padding + i * xStep;

  const linePath = data ? data.history.map((h: any, i: number) => `${i === 0 ? "M" : "L"}${getX(i)},${getY(h.score)}`).join(" ") : "";
  const areaPath = data ? `${linePath} L${getX(data.history.length - 1)},${chartHeight - padding} L${getX(0)},${chartHeight - padding} Z` : "";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Award className="w-6 h-6 text-blue-500" />{t("complianceScore.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Track compliance score trends and gap reduction over time.</p>
      </div>

      {/* Framework selector */}
      <div className="flex items-center gap-3">
        <select aria-label="Selected framework" value={selectedFramework} onChange={(e) => setSelectedFramework(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
          {frameworks.map((f) => <option key={f} value={f}>{f}</option>)}
        </select>
        {data && (
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-500">Current Score:</span>
            <span className={`text-2xl font-bold ${data.current_score >= 80 ? "text-green-600" : data.current_score >= 60 ? "text-yellow-600" : "text-red-600"}`}>{data.current_score}</span>
            <span className="text-sm text-gray-400">/100</span>
          </div>
        )}
      </div>

      {data && (
        <>
          {/* Line chart */}
          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="font-semibold mb-4">Score Trend ({data.history.length} months)</h3>
            <svg viewBox={`0 0 ${chartWidth} ${chartHeight}`} className="w-full h-48">
              {/* Grid lines */}
              {[0, 25, 50, 75, 100].map((y) => (
                <g key={y}>
                  <line x1={padding} y1={getY(y)} x2={chartWidth - padding} y2={getY(y)} stroke="currentColor" strokeWidth={0.5} className="text-gray-200 dark:text-gray-800" />
                  <text x={padding - 8} y={getY(y) + 4} textAnchor="end" className="text-[10px] fill-gray-400">{y}</text>
                </g>
              ))}
              {/* Area */}
              <path d={areaPath} fill="currentColor" className="text-blue-500" opacity={0.1} />
              {/* Line */}
              <path d={linePath} fill="none" stroke="currentColor" strokeWidth={2} className="text-blue-500" />
              {/* Points + gap count badges */}
              {data.history.map((h: any, i: number) => (
                <g key={i}>
                  <circle cx={getX(i)} cy={getY(h.score)} r={4} fill="currentColor" className="text-blue-500" />
                  <text x={getX(i)} y={getY(h.score) - 10} textAnchor="middle" className="text-[10px] fill-gray-500 font-medium">{h.score}</text>
                  {h.gap_count > 0 && (
                    <circle cx={getX(i)} cy={getY(h.score)} r={8} fill="none" stroke="currentColor" strokeWidth={1} className="text-orange-400" opacity={0.5} />
                  )}
                </g>
              ))}
              {/* X-axis labels */}
              {data.history.map((h: any, i: number) => (
                <text key={i} x={getX(i)} y={chartHeight - padding + 16} textAnchor="middle" className="text-[10px] fill-gray-400">{h.month}</text>
              ))}
            </svg>
            <div className="flex items-center gap-4 mt-2 text-xs text-gray-500">
              <span className="flex items-center gap-1"><span className="w-3 h-3 rounded-full bg-blue-500" /> Score</span>
              <span className="flex items-center gap-1"><span className="w-3 h-3 rounded-full border border-orange-400" /> Has gaps</span>
            </div>
          </div>

          {/* Monthly breakdown table */}
          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-900/50">
                <tr>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Month</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Score</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Gap Count</th>
                  <th scope="col" className="px-4 py-3 text-left font-medium">Delta</th>
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-gray-800">
                {data.history.slice().reverse().map((h: any, i: number) => (
                  <tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                    <td className="px-4 py-3 flex items-center gap-1"><Calendar className="w-3 h-3 text-gray-400" /> {h.month}</td>
                    <td className="px-4 py-3"><span className={`font-bold ${h.score >= 80 ? "text-green-600" : h.score >= 60 ? "text-yellow-600" : "text-red-600"}`}>{h.score}</span></td>
                    <td className="px-4 py-3">{h.gap_count > 0 ? <span className="px-2 py-0.5 rounded text-xs bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400">{h.gap_count} gaps</span> : <span className="text-green-600 text-xs">No gaps</span>}</td>
                    <td className="px-4 py-3">
                      {h.delta > 0 ? <span className="flex items-center gap-1 text-green-600 text-xs"><TrendingUp className="w-3 h-3" /> +{h.delta}</span> :
                       h.delta < 0 ? <span className="flex items-center gap-1 text-red-600 text-xs"><TrendingDown className="w-3 h-3" /> {h.delta}</span> :
                       <span className="text-gray-400 text-xs">-</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}

      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
