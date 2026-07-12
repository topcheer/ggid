'use client';
import { useState } from 'react';

interface Destination {
  id: string;
  name: string;
  type: string;
  status: string;
  lastForward: string;
  latency: string;
  circuitBreaker: string;
}

export default function SiemForwarderDashboardPage() {
  const [destinations] = useState<Destination[]>([
    { id: 'd1', name: 'Splunk Prod', type: 'Splunk', status: 'healthy', lastForward: '2026-07-12 14:32', latency: '45ms', circuitBreaker: 'closed' },
    { id: 'd2', name: 'ELK Cluster', type: 'ELK', status: 'healthy', lastForward: '2026-07-12 14:32', latency: '120ms', circuitBreaker: 'closed' },
    { id: 'd3', name: 'Datadog', type: 'Datadog', status: 'degraded', lastForward: '2026-07-12 14:30', latency: '850ms', circuitBreaker: 'half-open' },
    { id: 'd4', name: 'Webhook', type: 'HTTP', status: 'down', lastForward: '2026-07-12 13:15', latency: '-', circuitBreaker: 'open' },
  ]);

  const [stats] = useState({ eventsPerSec: 142, batchSize: 50, successRate: 98.5, retryQueueDepth: 23 });
  const [showFilterPreview, setShowFilterPreview] = useState(false);
  const [testTarget, setTestTarget] = useState('');

  const statusColor = (s: string): string =>
    s === 'healthy' ? 'bg-green-100 text-green-700' : s === 'degraded' ? 'bg-amber-100 text-amber-700' : 'bg-red-100 text-red-700';

  const cbColor = (s: string): string =>
    s === 'closed' ? 'bg-green-100 text-green-700' : s === 'half-open' ? 'bg-amber-100 text-amber-700' : 'bg-red-100 text-red-700';

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">SIEM Forwarder Dashboard</h1>
        <p className="text-gray-600">Monitor forwarding health, throughput, and circuit breaker status.</p>
      </div>

      <div className="grid grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{stats.eventsPerSec}</div>
          <div className="text-sm text-gray-500">Events/sec</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{stats.batchSize}</div>
          <div className="text-sm text-gray-500">Batch Size</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-green-600">{stats.successRate}%</div>
          <div className="text-sm text-gray-500">Success Rate</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-amber-600">{stats.retryQueueDepth}</div>
          <div className="text-sm text-gray-500">Retry Queue</div>
        </div>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Destination Health</h2>
        <div className="grid grid-cols-2 gap-4">
          {destinations.map(d => (
            <div key={d.id} className="border rounded p-4 space-y-2">
              <div className="flex items-center justify-between">
                <div>
                  <div className="font-medium text-sm">{d.name}</div>
                  <div className="text-xs text-gray-500">{d.type}</div>
                </div>
                <span className={`px-2 py-0.5 rounded text-xs ${statusColor(d.status)}`}>{d.status}</span>
              </div>
              <div className="flex items-center justify-between text-xs">
                <span className="text-gray-500">Latency: {d.latency}</span>
                <span className={`px-2 py-0.5 rounded text-xs ${cbColor(d.circuitBreaker)}`}>CB: {d.circuitBreaker}</span>
              </div>
              <div className="text-xs text-gray-400">Last forward: {d.lastForward}</div>
              <button onClick={() => setTestTarget(d.name)} className="text-blue-600 text-xs hover:underline">Test Forward</button>
            </div>
          ))}
        </div>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Event Filter Preview</h2>
          <button onClick={() => setShowFilterPreview(!showFilterPreview)} className="text-sm text-blue-600">{showFilterPreview ? 'Hide' : 'Show'}</button>
        </div>
        {showFilterPreview && (
          <div className="space-y-2 text-sm">
            <div className="flex gap-2"><span className="px-2 py-0.5 bg-red-100 text-red-700 rounded text-xs">severity {'>='} high</span><span className="text-gray-500">42 events/min</span></div>
            <div className="flex gap-2"><span className="px-2 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">source_type = auth</span><span className="text-gray-500">85 events/min</span></div>
            <div className="flex gap-2"><span className="px-2 py-0.5 bg-green-100 text-green-700 rounded text-xs">tenant = default</span><span className="text-gray-500">120 events/min</span></div>
          </div>
        )}
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">TLS Certificate Status</h2>
        <div className="space-y-2">
          {destinations.filter(d => d.type !== 'HTTP').map(d => (
            <div key={d.id} className="flex items-center justify-between text-sm border-b pb-2">
              <span className="font-medium">{d.name}</span>
              <span className="text-gray-500">Expires: 2026-11-15</span>
              <span className="px-2 py-0.5 bg-green-100 text-green-700 rounded text-xs">126 days left</span>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
