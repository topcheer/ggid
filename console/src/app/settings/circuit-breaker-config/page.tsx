'use client';
import { useState, useEffect } from 'react';
import { Loader2 } from 'lucide-react';
import { useTranslations } from "@/lib/i18n";

export default function CircuitBreakerConfigPage() {

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const t = useTranslations();

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/metrics", {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) return null;
        const json = await res.json();
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const [failureThreshold, setFailureThreshold] = useState(5);
  const [successThreshold, setSuccessThreshold] = useState(3);
  const [timeoutDuration, setTimeoutDuration] = useState(30);
  const [halfOpenMaxCalls, setHalfOpenMaxCalls] = useState(2);
  const [status, setStatus] = useState('closed');
  const [lastFailure, setLastFailure] = useState('never');
  const [autoRestore, setAutoRestore] = useState(true);
  const [excludeClientErrors, setExcludeClientErrors] = useState(false);
  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  const [rules, setRules] = useState([
    { service: 'identity-service', threshold: 5, enabled: true },
    { service: 'policy-service', threshold: 8, enabled: true },
    { service: 'audit-service', threshold: 10, enabled: false },
  ]);

  const statusColor = {
    closed: 'bg-green-500',
    open: 'bg-red-500',
    halfOpen: 'bg-yellow-500',
  }[status] || 'bg-gray-400';

  const toggleRule = (idx: number) => {
    setRules(prev => prev.map((r, i) => i === idx ? { ...r, enabled: !r.enabled } : r));
  };

  const updateThreshold = (idx: number, val: string) => {
    setRules(prev => prev.map((r, i) => i === idx ? { ...r, threshold: parseInt(val) || 0 } : r));
  };

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t("backend2.circuitBreakerConfig.title")}</h1>
      <p className="text-gray-600">Configure failure thresholds and recovery behavior for downstream services.</p>

      <section className="bg-white rounded-lg shadow p-6 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <div className={`w-4 h-4 rounded-full ${statusColor}`} />
          <div>
            <div className="text-sm font-medium">{"Status"}</div>
            <div className="text-2xl font-bold">
              {status === 'closed' ? "Closed" :
               status === 'open' ? "Open" :
               "Half Open"}
            </div>
          </div>
        </div>
        <div className="text-sm text-gray-500">
          {"Last Failure"}: {lastFailure}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Global Thresholds</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{t("backend2.circuitBreakerConfig.failureThreshold")}</label>
            <input
              type="number"
              min={1}
              max={100}
              value={failureThreshold}
              onChange={e => setFailureThreshold(parseInt(e.target.value) || 0)}
              className="w-full border rounded px-3 py-2 text-sm"
            />
          </div>
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{"Success Threshold"}</label>
            <input
              type="number"
              min={1}
              max={100}
              value={successThreshold}
              onChange={e => setSuccessThreshold(parseInt(e.target.value) || 0)}
              className="w-full border rounded px-3 py-2 text-sm"
            />
          </div>
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{"Timeout Duration"}</label>
            <input
              type="number"
              min={1}
              max={300}
              value={timeoutDuration}
              onChange={e => setTimeoutDuration(parseInt(e.target.value) || 0)}
              className="w-full border rounded px-3 py-2 text-sm"
            />
          </div>
          <div className="space-y-1">
            <label className="text-sm text-gray-600">{"Half Open Max Calls"}</label>
            <input
              type="number"
              min={1}
              max={100}
              value={halfOpenMaxCalls}
              onChange={e => setHalfOpenMaxCalls(parseInt(e.target.value) || 0)}
              className="w-full border rounded px-3 py-2 text-sm"
            />
          </div>
        </div>

        <div className="space-y-2">
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={autoRestore} onChange={e => setAutoRestore(e.target.checked)} />
            Auto-restore to half-open after timeout
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={excludeClientErrors} onChange={e => setExcludeClientErrors(e.target.checked)} />
            Exclude 4xx client errors from failure count
          </label>
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Per-Service Rules</h2>
        <div className="space-y-2">
          {rules.map((rule, idx) => (
            <div key={rule.service} className="flex items-center gap-4 p-3 border rounded">
              <input
                type="checkbox"
                checked={rule.enabled}
                onChange={() => toggleRule(idx)}
                className="w-4 h-4"
              />
              <span className="flex-1 font-mono text-sm">{rule.service}</span>
              <div className="flex items-center gap-2">
                <span className="text-sm text-gray-500">Threshold:</span>
                <input
                  type="number"
                  min={1}
                  value={rule.threshold}
                  onChange={e => updateThreshold(idx, e.target.value)}
                  className="w-20 border rounded px-2 py-1 text-sm"
                />
              </div>
            </div>
          ))}
        </div>
      </section>

      <div className="flex justify-end gap-3">
        <button className="px-4 py-2 border rounded text-sm">{t("backend2.circuitBreakerConfig.reset")}</button>
        <button className="px-4 py-2 bg-blue-600 text-white rounded text-sm">Save Configuration</button>
      </div>
    </div>
  );
}
