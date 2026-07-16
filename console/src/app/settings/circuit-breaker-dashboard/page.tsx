'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface BreakerRow {
  service: string;
  state: 'closed' | 'open' | 'half-open';
  failures: number;
  last1m: number;
  last5m: number;
  last1h: number;
  slow: number;
}

const defaultData: BreakerRow[] = [
  { service: 'identity-service', state: 'closed', failures: 0, last1m: 12, last5m: 58, last1h: 1200, slow: 3 },
  { service: 'policy-service', state: 'closed', failures: 1, last1m: 8, last5m: 45, last1h: 900, slow: 1 },
  { service: 'audit-service', state: 'half-open', failures: 3, last1m: 0, last5m: 2, last1h: 110, slow: 0 },
  { service: 'notification-service', state: 'open', failures: 14, last1m: 0, last5m: 0, last1h: 45, slow: 5 },
  { service: 'org-service', state: 'closed', failures: 0, last1m: 5, last5m: 32, last1h: 600, slow: 0 },
];

export default function CircuitBreakerDashboardPage() {
  const [rows, setRows] = useState<BreakerRow[]>(defaultData);
  const [loading, setLoading] = useState(true);

  const t = useTranslations();

  useEffect(() => {
    const timer = setTimeout(() => setLoading(false), 600);
    return () => clearTimeout(timer);
  }, []);

  const resetService = (service: string) => {
    setRows(prev => prev.map(r => r.service === service ? { ...r, state: 'closed' as const, failures: 0 } : r));
  };

  const stateClass = (state: string) => {
    switch (state) {
      case 'closed': return 'bg-green-100 text-green-700';
      case 'open': return 'bg-red-100 text-red-700';
      case 'half-open': return 'bg-yellow-100 text-yellow-700';
      default: return 'bg-gray-100 text-gray-700';
    }
  };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.circuitBreakerDash.title")}</h1>
      <p className="text-gray-600">Real-time view of circuit breaker state across services.</p>

      {loading ? (
        <div className="p-8 text-center text-gray-500">Loading...</div>
      ) : (
        <div className="bg-white rounded-lg shadow overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50">
              <tr className="text-left border-b">
                <th scope="col" className="px-4 py-3">{t("backend2.circuitBreakerDash.service")}</th>
                <th scope="col" className="px-4 py-3">{t("backend2.circuitBreakerDash.state")}</th>
                <th scope="col" className="px-4 py-3">{"Failures"}</th>
                <th scope="col" className="px-4 py-3">{"Last Minute"}</th>
                <th scope="col" className="px-4 py-3">{"Last 5 Minutes"}</th>
                <th scope="col" className="px-4 py-3">{"Last Hour"}</th>
                <th scope="col" className="px-4 py-3">Slow</th>
                <th scope="col" className="px-4 py-3">{t("backend2.circuitBreakerDash.action")}</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {rows.map(row => (
                <tr key={row.service} className="hover:bg-gray-50">
                  <td className="px-4 py-3 font-mono">{row.service}</td>
                  <td className="px-4 py-3">
                    <span className={`px-2 py-1 rounded text-xs font-medium ${stateClass(row.state)}`}>{row.state}</span>
                  </td>
                  <td className="px-4 py-3">{row.failures}</td>
                  <td className="px-4 py-3">{row.last1m}</td>
                  <td className="px-4 py-3">{row.last5m}</td>
                  <td className="px-4 py-3">{row.last1h}</td>
                  <td className="px-4 py-3">{row.slow}</td>
                  <td className="px-4 py-3">
                    <button
                      onClick={() => resetService(row.service)}
                      className="px-2 py-1 text-xs border rounded hover:bg-gray-100"
                    >
                      {"Reset Service"}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold mb-3">{"Per Service"}</h2>
        <div className="space-y-3">
          {rows.map(row => (
            <div key={row.service} className="flex items-center gap-4">
              <span className="font-mono text-sm w-40">{row.service}</span>
              <div className="flex-1 bg-gray-100 rounded-full h-3 overflow-hidden">
                <div
                  className={`h-full ${row.state === 'open' ? 'bg-red-500' : row.state === 'half-open' ? 'bg-yellow-500' : 'bg-green-500'}`}
                  style={{ width: `${Math.min((row.failures / 20) * 100, 100)}%` }}
                />
              </div>
              <span className="text-xs text-gray-500 w-12 text-right">{row.failures}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
