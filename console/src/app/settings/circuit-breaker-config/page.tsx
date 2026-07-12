'use client';
import { useState, useEffect } from 'react';

interface ServiceCB {
  id: string;
  name: string;
  state: string;
  failureThreshold: number;
  recoveryTimeout: number;
  halfOpenMax: number;
  failureCount: number;
}

export default function CircuitBreakerConfigPage() {

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<any[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/healthz", {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        setData(Array.isArray(json) ? json : [json]);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  if (loading) return <div className="p-8">Loading...</div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  if (!data || data.length === 0) return <div className="p-8 text-gray-500">No data available</div>;
  const [services, setServices] = useState<ServiceCB[]>([
    { id: 'auth', name: 'Auth Service', state: 'closed', failureThreshold: 5, recoveryTimeout: 30, halfOpenMax: 3, failureCount: 0 },
    { id: 'identity', name: 'Identity Service', state: 'closed', failureThreshold: 5, recoveryTimeout: 30, halfOpenMax: 3, failureCount: 0 },
    { id: 'oauth', name: 'OAuth Service', state: 'open', failureThreshold: 3, recoveryTimeout: 60, halfOpenMax: 2, failureCount: 7 },
    { id: 'policy', name: 'Policy Service', state: 'closed', failureThreshold: 10, recoveryTimeout: 15, halfOpenMax: 5, failureCount: 0 },
    { id: 'org', name: 'Org Service', state: 'half-open', failureThreshold: 5, recoveryTimeout: 30, halfOpenMax: 3, failureCount: 2 },
    { id: 'audit', name: 'Audit Service', state: 'closed', failureThreshold: 5, recoveryTimeout: 30, halfOpenMax: 3, failureCount: 0 },
  ]);

  const [globalThreshold, setGlobalThreshold] = useState(5);
  const [globalRecovery, setGlobalRecovery] = useState(30);
  const [globalHalfOpen, setGlobalHalfOpen] = useState(3);
  const [history] = useState([
    { time: '14:30', service: 'oauth', event: 'opened', failures: 3 },
    { time: '14:15', service: 'org', event: 'half-opened', failures: 5 },
    { time: '13:45', service: 'org', event: 'opened', failures: 5 },
    { time: '13:20', service: 'oauth', event: 'closed', failures: 0 },
  ]);

  const stateColor = (s: string): string =>
    s === 'closed' ? 'bg-green-100 text-green-700' : s === 'open' ? 'bg-red-100 text-red-700' : 'bg-amber-100 text-amber-700';

  const reset = (id: string) => {
    setServices(prev => prev.map(s => s.id === id ? { ...s, state: 'closed', failureCount: 0 } : s));
  };

  const updateService = (id: string, field: keyof ServiceCB, value: number) => {
    setServices(prev => prev.map(s => s.id === id ? { ...s, [field]: value } : s));
  };

  const applyGlobal = () => {
    setServices(prev => prev.map(s => ({ ...s, failureThreshold: globalThreshold, recoveryTimeout: globalRecovery, halfOpenMax: globalHalfOpen })));
  };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Circuit Breaker Configuration</h1>
        <p className="text-gray-600">Per-service circuit breaker settings, state monitoring, and failure history.</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Global Defaults</h2>
        <div className="grid grid-cols-4 gap-4 items-end">
          <div><label className="text-sm font-medium">Failure Threshold</label><input type="number" min={1} value={globalThreshold} onChange={e => setGlobalThreshold(parseInt(e.target.value) || 5)} className="w-full border rounded px-2 py-1 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">Recovery Timeout (s)</label><input type="number" min={1} value={globalRecovery} onChange={e => setGlobalRecovery(parseInt(e.target.value) || 30)} className="w-full border rounded px-2 py-1 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">Half-Open Max Requests</label><input type="number" min={1} value={globalHalfOpen} onChange={e => setGlobalHalfOpen(parseInt(e.target.value) || 3)} className="w-full border rounded px-2 py-1 text-sm mt-1" /></div>
          <button onClick={applyGlobal} className="px-3 py-1.5 bg-blue-600 text-white rounded text-sm">Apply to All</button>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">Service</th>
              <th className="p-3">State</th>
              <th className="p-3">Failures</th>
              <th className="p-3">Threshold</th>
              <th className="p-3">Recovery (s)</th>
              <th className="p-3">Half-Open Max</th>
              <th className="p-3">Action</th>
            </tr>
          </thead>
          <tbody>
            {services.map(s => (
              <tr key={s.id} className="border-b">
                <td className="p-3 font-medium">{s.name}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${stateColor(s.state)}`}>{s.state}</span></td>
                <td className="p-3">{s.failureCount}</td>
                <td className="p-3"><input type="number" min={1} value={s.failureThreshold} onChange={e => updateService(s.id, 'failureThreshold', parseInt(e.target.value) || 5)} className="w-16 border rounded px-1 py-0.5 text-sm" /></td>
                <td className="p-3"><input type="number" min={1} value={s.recoveryTimeout} onChange={e => updateService(s.id, 'recoveryTimeout', parseInt(e.target.value) || 30)} className="w-16 border rounded px-1 py-0.5 text-sm" /></td>
                <td className="p-3"><input type="number" min={1} value={s.halfOpenMax} onChange={e => updateService(s.id, 'halfOpenMax', parseInt(e.target.value) || 3)} className="w-16 border rounded px-1 py-0.5 text-sm" /></td>
                <td className="p-3"><button onClick={() => reset(s.id)} className="text-blue-600 text-xs hover:underline">Reset</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Failure History</h2>
        <div className="flex items-end gap-2 h-32">
          {[3, 7, 5, 2, 8, 4, 1].map((v, i) => (
            <div key={i} className="flex-1 flex flex-col items-center">
              <div className={`w-full rounded-t ${v > 5 ? 'bg-red-500' : v > 2 ? 'bg-amber-500' : 'bg-green-500'}`} style={{ height: `${v * 20}px` }} />
              <div className="text-xs text-gray-500 mt-1">{i + 1}h ago</div>
            </div>
          ))}
        </div>
        <div className="space-y-1">
          {history.map((h, idx) => (
            <div key={idx} className="text-sm flex items-center gap-3 border-b pb-1">
              <span className="text-gray-500 text-xs">{h.time}</span>
              <span className="font-medium">{h.service}</span>
              <span className={`px-2 py-0.5 rounded text-xs ${h.event === 'opened' ? 'bg-red-100 text-red-700' : h.event === 'half-opened' ? 'bg-amber-100 text-amber-700' : 'bg-green-100 text-green-700'}`}>{h.event}</span>
              <span className="text-xs text-gray-500">{h.failures} failures</span>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}