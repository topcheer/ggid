'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface Endpoint { id: string; path: string; method: string; status: string; latency: string; uptime: string; errorRate: string; }

export default function ApiHealthMonitorPage() {

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

  const t = useTranslations();
  if (loading) return <div className="p-8">{t("apiHealthMonitor.loading")}</div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  if (!data || data.length === 0) return <div className="p-8 text-gray-500">{t("apiHealthMonitor.noData")}</div>;
  const [endpoints] = useState<Endpoint[]>([
    { id: 'e1', path: '/api/v1/auth/login', method: 'POST', status: 'healthy', latency: '45ms', uptime: '99.98%', errorRate: '0.02%' },
    { id: 'e2', path: '/api/v1/users', method: 'GET', status: 'healthy', latency: '32ms', uptime: '99.99%', errorRate: '0.01%' },
    { id: 'e3', path: '/api/v1/policy/evaluate', method: 'POST', status: 'degraded', latency: '280ms', uptime: '99.5%', errorRate: '0.5%' },
    { id: 'e4', path: '/api/v1/audit/events', method: 'GET', status: 'healthy', latency: '67ms', uptime: '99.95%', errorRate: '0.05%' },
    { id: 'e5', path: '/api/v1/orgs', method: 'POST', status: 'down', latency: '-', uptime: '98.2%', errorRate: '1.8%' },
  ]);
  const [deps] = useState([
    { name: 'PostgreSQL', status: 'healthy', latency: '2ms' },
    { name: 'Redis', status: 'healthy', latency: '1ms' },
    { name: 'NATS JetStream', status: 'healthy', latency: '5ms' },
    { name: 'SMTP Server', status: 'degraded', latency: '450ms' },
  ]);
  const [alerts] = useState([
    { time: '14:30', msg: 'Policy evaluate latency p99 > 500ms', level: 'warn' },
    { time: '13:15', msg: 'Org create endpoint returning 500', level: 'critical' },
  ]);

  const statusColor = (s: string) => s === 'healthy' ? 'bg-green-100 text-green-700' : s === 'degraded' ? 'bg-amber-100 text-amber-700' : 'bg-red-100 text-red-700';
  const depColor = (s: string) => s === 'healthy' ? 'text-green-600' : 'text-amber-600';

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div><h1 className="text-2xl font-bold">{t("apiHealthMonitor.title")}</h1><p className="text-gray-600">{t("apiHealthMonitor.subtitle")}</p></div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <h2 className="text-lg font-semibold p-6 pb-4">{t("apiHealthMonitor.endpointHealth")}</h2>
        <table className="w-full text-sm"><thead className="bg-gray-50"><tr className="text-left"><th className="p-3">{t("apiHealthMonitor.path")}</th><th className="p-3">{t("apiHealthMonitor.method")}</th><th className="p-3">{t("apiHealthMonitor.status")}</th><th className="p-3">{t("apiHealthMonitor.latency")}</th><th className="p-3">{t("apiHealthMonitor.uptime")}</th><th className="p-3">{t("apiHealthMonitor.errorRate")}</th></tr></thead>
          <tbody>{endpoints.map(e => (
            <tr key={e.id} className="border-b"><td className="p-3 font-mono text-xs">{e.path}</td><td className="p-3"><span className="px-2 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">{e.method}</span></td><td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(e.status)}`}>{e.status}</span></td><td className="p-3 text-gray-500">{e.latency}</td><td className="p-3 text-gray-500">{e.uptime}</td><td className="p-3 text-gray-500">{e.errorRate}</td></tr>
          ))}</tbody></table>
      </section>

      <div className="grid grid-cols-2 gap-6">
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">{t("apiHealthMonitor.dependencyHealth")}</h2>
          <div className="space-y-2">{deps.map(d => <div key={d.name} className="flex items-center justify-between text-sm border-b pb-2"><span className="font-medium">{d.name}</span><div className="flex items-center gap-3"><span className={`text-xs ${depColor(d.status)}`}>{d.status}</span><span className="text-xs text-gray-400">{d.latency}</span></div></div>)}</div>
        </section>
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">{t("apiHealthMonitor.latencyDistribution")}</h2>
          <div className="space-y-3">
            <div className="flex items-center gap-3"><span className="text-sm w-12">p50</span><div className="flex-1 bg-gray-200 rounded-full h-3 overflow-hidden"><div className="h-3 bg-green-500 rounded-full" style={{ width: '20%' }} /></div><span className="text-xs text-gray-500">45ms</span></div>
            <div className="flex items-center gap-3"><span className="text-sm w-12">p95</span><div className="flex-1 bg-gray-200 rounded-full h-3 overflow-hidden"><div className="h-3 bg-amber-500 rounded-full" style={{ width: '55%' }} /></div><span className="text-xs text-gray-500">180ms</span></div>
            <div className="flex items-center gap-3"><span className="text-sm w-12">p99</span><div className="flex-1 bg-gray-200 rounded-full h-3 overflow-hidden"><div className="h-3 bg-red-500 rounded-full" style={{ width: '85%' }} /></div><span className="text-xs text-gray-500">520ms</span></div>
          </div>
        </section>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("apiHealthMonitor.alertTimeline")}</h2>
        <div className="space-y-2">{alerts.map((a, i) => <div key={i} className="flex items-center gap-3 text-sm border-b pb-2"><span className="text-xs text-gray-500">{a.time}</span><span className={`px-2 py-0.5 rounded text-xs ${a.level === 'critical' ? 'bg-red-100 text-red-700' : 'bg-amber-100 text-amber-700'}`}>{a.level}</span><span>{a.msg}</span></div>)}</div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("apiHealthMonitor.statusCodeDist")}</h2>
        <div className="flex items-end gap-4 h-32">
          {[{'code':'2xx','count':15200,'color':'bg-green-500'},{'code':'3xx','count':230,'color':'bg-blue-500'},{'code':'4xx','count':180,'color':'bg-amber-500'},{'code':'5xx','count':42,'color':'bg-red-500'}].map(s => (
            <div key={s.code} className="flex-1 flex flex-col items-center"><div className={`w-full rounded-t ${s.color}`} style={{ height: `${Math.min(s.count / 100, 100)}px` }} /><div className="text-xs text-gray-500 mt-1">{s.code}</div><div className="text-xs font-mono">{s.count.toLocaleString()}</div></div>
          ))}
        </div>
      </section>
    </div>
  );
}