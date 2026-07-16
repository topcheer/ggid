'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface CacheStats {
  name: string;
  hitRate: number;
  misses: number;
  size: number;
  memory: number;
  healthy: boolean;
}

const defaultStats: CacheStats[] = [
  { name: 'session', hitRate: 94.2, misses: 124, size: 12400, memory: 18, healthy: true },
  { name: 'policy', hitRate: 88.7, misses: 340, size: 5600, memory: 9, healthy: true },
  { name: 'token', hitRate: 99.1, misses: 12, size: 89000, memory: 64, healthy: true },
  { name: 'audit', hitRate: 72.4, misses: 2100, size: 3400, memory: 5, healthy: false },
];

export default function CacheHealthPage() {
  const [stats, setStats] = useState<CacheStats[]>(defaultStats);
  const [loading, setLoading] = useState(true);

  const t = useTranslations();

  useEffect(() => {
    const timer = setTimeout(() => setLoading(false), 500);
    return () => clearTimeout(timer);
  }, []);

  const refresh = (name: string) => {
    setStats(prev => prev.map(s => s.name === name ? { ...s, hitRate: Math.min(s.hitRate + 0.5, 100), misses: Math.max(0, s.misses - 10) } : s));
  };

  const clear = (name: string) => {
    setStats(prev => prev.map(s => s.name === name ? { ...s, hitRate: 0, misses: 0, size: 0, memory: 0 } : s));
  };

  if (loading) return <div className="p-8 text-center">Loading...</div>;
  if (!stats.length) return <div className="p-8 text-center text-gray-500">{"No Data"}</div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.cacheHealth.title")}</h1>
      <p className="text-gray-600">View cache hit rates, miss counts, and memory footprint per cache.</p>

      <div className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left border-b">
              <th scope="col" className="px-4 py-3">{"Cache Type"}</th>
              <th scope="col" className="px-4 py-3">{t("backend2.cacheHealth.hitRate")}</th>
              <th scope="col" className="px-4 py-3">{"Misses"}</th>
              <th scope="col" className="px-4 py-3">{"Size"}</th>
              <th scope="col" className="px-4 py-3">{t("backend2.cacheHealth.memory")}</th>
              <th scope="col" className="px-4 py-3">Healthy</th>
              <th scope="col" className="px-4 py-3">{"Actions"}</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {stats.map(s => (
              <tr key={s.name} className="hover:bg-gray-50">
                <td className="px-4 py-3 font-mono font-medium">{s.name}</td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-2">
                    <div className="w-24 bg-gray-200 rounded-full h-2">
                      <div className={`h-2 rounded-full ${s.hitRate > 90 ? 'bg-green-500' : s.hitRate > 75 ? 'bg-yellow-500' : 'bg-red-500'}`} style={{ width: `${s.hitRate}%` }} />
                    </div>
                    <span className="text-xs">{s.hitRate.toFixed(1)}%</span>
                  </div>
                </td>
                <td className="px-4 py-3">{s.misses.toLocaleString()}</td>
                <td className="px-4 py-3">{s.size.toLocaleString()}</td>
                <td className="px-4 py-3">{s.memory} MB</td>
                <td className="px-4 py-3">
                  <span className={`px-2 py-1 rounded text-xs ${s.healthy ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>{s.healthy ? 'Yes' : 'No'}</span>
                </td>
                <td className="px-4 py-3 flex gap-2">
                  <button onClick={() => refresh(s.name)} className="px-2 py-1 text-xs border rounded hover:bg-gray-100">{"Refresh"}</button>
                  <button onClick={() => clear(s.name)} className="px-2 py-1 text-xs border rounded text-red-600 hover:bg-red-50">{"Clear"}</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
