'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface RiskFactor {
  name: string;
  score: number;
  weight: number;
  enabled: boolean;
}

interface RiskEvent {
  id: string;
  user: string;
  score: number;
  factors: string[];
  action: string;
  timestamp: string;
}

export default function RiskEngineDashboardPage() {
  const t = useTranslations();
  const [factors, setFactors] = useState<RiskFactor[]>([]);
  const [events, setEvents] = useState<RiskEvent[]>([]);
  const [thresholds, setThresholds] = useState([] as { level: string; minScore: number; maxScore: number; action: string }[]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/v1/auth/risk/aggregate', {
      headers: { ...authHeader(), 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.factors) setFactors(data.factors);
          if (data.events) setEvents(data.events);
          if (data.thresholds) setThresholds(data.thresholds);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  const currentScore = factors.filter(f => f.enabled).reduce((sum, f) => sum + f.score, 0);
  const gaugeColor = currentScore >= 85 ? 'text-red-600' : currentScore >= 60 ? 'text-amber-600' : currentScore >= 30 ? 'text-yellow-600' : 'text-green-600';

  const toggleFactor = (idx: number) => {
    setFactors(prev => prev.map((f: any, i: number) => i === idx ? { ...f, enabled: !f.enabled } : f));
  };

  const updateAction = (idx: number, action: string) => {
    setThresholds(prev => prev.map((t: any, i: number) => i === idx ? { ...t, action } : t));
  };

  const actionColor = (a: string): string =>
    a === 'block' ? 'bg-red-100 text-red-700' : a === 'challenge-mfa' ? 'bg-amber-100 text-amber-700' : a === 'step-up' ? 'bg-blue-100 text-blue-700' : 'bg-green-100 text-green-700';

  if (loading) return <div className="p-6"><p>{t("common.loading")}</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("riskEngine.title")}</h1>
        <p className="text-gray-600">{t("riskEngine.subtitle")}</p>
      </div>

      <div className="grid grid-cols-3 gap-6">
        <section className="bg-white rounded-lg shadow p-6 text-center">
          <h2 className="text-lg font-semibold mb-4">{t("riskEngine.currentScore")}</h2>
          <div className={`text-6xl font-bold ${gaugeColor}`}>{currentScore}</div>
          <div className="text-sm text-gray-500 mt-2">{t("riskEngine.outOf")}</div>
          <div className="mt-4 h-3 bg-gray-200 rounded-full overflow-hidden">
            <div className={`h-3 rounded-full ${currentScore >= 85 ? 'bg-red-500' : currentScore >= 60 ? 'bg-amber-500' : currentScore >= 30 ? 'bg-yellow-500' : 'bg-green-500'}`} style={{ width: `${currentScore}%` }} />
          </div>
        </section>

        <section className="col-span-2 bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">{t("riskEngine.factorBreakdown")}</h2>
          <div className="space-y-3">
            {factors.map((f: any, idx: number) => (
              <div key={f.name} className="flex items-center gap-4">
                <label className="flex items-center gap-2 w-40">
                  <input aria-label="F" type="checkbox" checked={f.enabled} onChange={() => toggleFactor(idx)} className="rounded" />
                  <span className="text-sm">{f.name}</span>
                </label>
                <div className="flex-1 bg-gray-200 rounded-full h-4 overflow-hidden">
                  <div className="h-4 bg-blue-500 rounded-full" style={{ width: `${f.score}%` }} />
                </div>
                <span className="text-sm font-mono w-10 text-right">{f.score}</span>
                <span className="text-xs text-gray-400 w-12">(w:{f.weight})</span>
              </div>
            ))}
          </div>
        </section>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("riskEngine.thresholdConfig")}</h2>
        <div className="space-y-2">
          {thresholds.map((t: any, idx: number) => (
            <div key={t.level} className="flex items-center gap-4">
              <span className={`px-2 py-0.5 rounded text-xs capitalize ${t.level === 'critical' ? 'bg-red-100 text-red-700' : t.level === 'high' ? 'bg-amber-100 text-amber-700' : t.level === 'medium' ? 'bg-yellow-100 text-yellow-700' : 'bg-green-100 text-green-700'}`}>{t.level}</span>
              <span className="text-sm text-gray-500 w-20">{t.minScore}-{t.maxScore}</span>
              <span className="text-gray-300">{'->'}</span>
              <select aria-label="Select option" value={t.action} onChange={e => updateAction(idx, e.target.value)} className="border rounded px-2 py-1 text-sm">
                <option value="allow">Allow</option>
                <option value="step-up">Step-up Auth</option>
                <option value="challenge-mfa">Challenge MFA</option>
                <option value="block">Block</option>
              </select>
            </div>
          ))}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("riskEngine.riskDistribution")}</h2>
        <div className="flex items-end gap-2 h-32">
          {[
            { range: '0-20', count: 45 },
            { range: '20-40', count: 30 },
            { range: '40-60', count: 15 },
            { range: '60-80', count: 8 },
            { range: '80-100', count: 2 },
          ].map(h => (
            <div key={h.range} className="flex-1 flex flex-col items-center">
              <div className="w-full bg-blue-500 rounded-t" style={{ height: `${h.count * 2}px` }} />
              <div className="text-xs text-gray-500 mt-1">{h.range}</div>
              <div className="text-xs font-mono">{h.count}</div>
            </div>
          ))}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("riskEngine.recentEvents")}</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th scope="col" className="p-3">User</th>
              <th scope="col" className="p-3">Score</th>
              <th scope="col" className="p-3">Factors</th>
              <th scope="col" className="p-3">Action</th>
              <th scope="col" className="p-3">Time</th>
            </tr>
          </thead>
          <tbody>
            {events.map(e => (
              <tr key={e.id} className="border-b">
                <td className="p-3 font-medium">{e.user}</td>
                <td className="p-3"><span className={`font-mono font-bold ${e.score >= 85 ? 'text-red-600' : e.score >= 60 ? 'text-amber-600' : 'text-yellow-600'}`}>{e.score}</span></td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{e.factors.map(f => <span key={f} className="px-1.5 py-0.5 bg-gray-100 rounded text-xs">{f}</span>)}</div></td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${actionColor(e.action)}`}>{e.action}</span></td>
                <td className="p-3 text-gray-500 text-xs">{e.timestamp}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
