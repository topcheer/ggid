'use client';
import { useState, useEffect } from 'react';

interface Delivery { id: string; webhook: string; event: string; status: string; attempts: number; latency: string; nextRetry: string; }

export default function WebhookDeliveryMonitorPage() {

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<any[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/webhooks/delivery-status", {
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
  const [deliveries] = useState<Delivery[]>([
    { id: 'd1', webhook: 'Splunk Prod', event: 'user.login', status: 'success', attempts: 1, latency: '45ms', nextRetry: '-' },
    { id: 'd2', webhook: 'ELK Cluster', event: 'policy.update', status: 'success', attempts: 1, latency: '120ms', nextRetry: '-' },
    { id: 'd3', webhook: 'Datadog', event: 'alert.triggered', status: 'failed', attempts: 3, latency: '850ms', nextRetry: '14:35' },
    { id: 'd4', webhook: 'Webhook', event: 'user.created', status: 'dead_letter', attempts: 5, latency: '-', nextRetry: '-' },
  ]);
  const [statusFilter, setStatusFilter] = useState('all');
  const [selected, setSelected] = useState<Delivery | null>(null);

  const filtered = deliveries.filter(d => statusFilter === 'all' || d.status === statusFilter);
  const statusColor = (s: string) => s === 'success' ? 'bg-green-100 text-green-700' : s === 'failed' ? 'bg-red-100 text-red-700' : 'bg-gray-200 text-gray-600';
  const stats = { total: 15420, success: 98.5, failed: 2, deadLetter: 23 };

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div><h1 className="text-2xl font-bold">Webhook Delivery Monitor</h1><p className="text-gray-600">Track webhook delivery status, retry failures, and dead letter queue.</p></div>

      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold">{stats.total.toLocaleString()}</div><div className="text-sm text-gray-500">Total (24h)</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold text-green-600">{stats.success}%</div><div className="text-sm text-gray-500">Success Rate</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold text-red-600">{stats.failed}</div><div className="text-sm text-gray-500">Failed</div></div>
        <div className="bg-white rounded-lg shadow p-4 text-center"><div className="text-2xl font-bold text-amber-600">{stats.deadLetter}</div><div className="text-sm text-gray-500">Dead Letter</div></div>
      </div>

      <div className="flex gap-3"><select value={statusFilter} onChange={e => setStatusFilter(e.target.value)} className="border rounded px-3 py-2 text-sm"><option value="all">All Statuses</option><option value="success">Success</option><option value="failed">Failed</option><option value="dead_letter">Dead Letter</option></select></div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm"><thead className="bg-gray-50"><tr className="text-left"><th className="p-3">Webhook</th><th className="p-3">Event</th><th className="p-3">Status</th><th className="p-3">Attempts</th><th className="p-3">Latency</th><th className="p-3">Next Retry</th><th className="p-3">Action</th></tr></thead>
          <tbody>{filtered.map(d => (
            <tr key={d.id} onClick={() => setSelected(d)} className="border-b cursor-pointer hover:bg-gray-50">
              <td className="p-3 font-medium">{d.webhook}</td><td className="p-3 font-mono text-xs">{d.event}</td><td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(d.status)}`}>{d.status}</span></td><td className="p-3">{d.attempts}</td><td className="p-3 text-gray-500">{d.latency}</td><td className="p-3 text-gray-500 text-xs">{d.nextRetry}</td><td className="p-3">{d.status === 'failed' && <button className="text-blue-600 text-xs hover:underline">Retry</button>}</td>
            </tr>))}</tbody></table>
      </section>

      {selected && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-lg w-full mx-4 space-y-4">
            <div className="flex items-center justify-between"><h2 className="text-lg font-semibold">Delivery Detail</h2><button onClick={() => setSelected(null)} className="text-gray-400">x</button></div>
            <div className="space-y-2 text-sm"><div><span className="text-gray-500">Webhook:</span> {selected.webhook}</div><div><span className="text-gray-500">Event:</span> {selected.event}</div><div><span className="text-gray-500">Status:</span> {selected.status}</div><div><span className="text-gray-500">Attempts:</span> {selected.attempts}</div></div>
            <div><div className="text-xs text-gray-500 mb-1">Request:</div><pre className="bg-gray-900 text-green-400 rounded p-3 text-xs overflow-x-auto">{`POST /webhook\nContent-Type: application/json\nX-GGID-Signature: sha256=abc123\n\n{"event":"${selected.event}","timestamp":"2026-07-12T14:30Z"}`}</pre></div>
            <div><div className="text-xs text-gray-500 mb-1">Response:</div><pre className="bg-gray-900 text-green-400 rounded p-3 text-xs overflow-x-auto">{`HTTP/1.1 ${selected.status === 'success' ? '200 OK' : '503 Service Unavailable'}\n\n{"status":"${selected.status}"}`}</pre></div>
          </div>
        </div>
      )}
    </div>
  );
}