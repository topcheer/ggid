'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface Subscription {
  id: string;
  eventType: string;
  endpointUrl: string;
  enabled: boolean;
  retryCount: number;
  maxRetries: number;
}

interface Delivery {
  id: string;
  subscription: string;
  status: string;
  timestamp: string;
  statusCode: number;
}

export default function WebhookSubscriptionConfigPage() {
  const t = useTranslations();

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<any[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/alerts/config", {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) return null;
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
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([
    { id: 'sub1', eventType: 'user.created', endpointUrl: 'https://app.example.com/hooks/user', enabled: true, retryCount: 0, maxRetries: 3 },
    { id: 'sub2', eventType: 'user.deleted', endpointUrl: 'https://app.example.com/hooks/user', enabled: true, retryCount: 2, maxRetries: 3 },
    { id: 'sub3', eventType: 'role.assigned', endpointUrl: 'https://hr.example.com/webhooks', enabled: true, retryCount: 0, maxRetries: 5 },
    { id: 'sub4', eventType: 'org.created', endpointUrl: 'https://billing.example.com/hooks', enabled: false, retryCount: 5, maxRetries: 3 },
  ]);

  const [showForm, setShowForm] = useState(false);
  const [newSub, setNewSub] = useState({ eventType: '', endpointUrl: '', secret: '', maxRetries: 3 });
  const [deliveries] = useState<Delivery[]>([
    { id: 'd1', subscription: 'user.created', status: 'success', timestamp: '2026-07-12 14:30', statusCode: 200 },
    { id: 'd2', subscription: 'user.deleted', status: 'retry', timestamp: '2026-07-12 14:15', statusCode: 500 },
    { id: 'd3', subscription: 'role.assigned', status: 'success', timestamp: '2026-07-12 13:45', statusCode: 200 },
    { id: 'd4', subscription: 'org.created', status: 'failed', timestamp: '2026-07-12 12:00', statusCode: 0 },
  ]);

  const eventCatalog = ['user.created', 'user.updated', 'user.deleted', 'role.assigned', 'role.revoked', 'org.created', 'org.updated', 'audit.high_risk', 'auth.login', 'auth.failed'];

  const addSub = () => {
    setSubscriptions(prev => [...prev, { id: `sub${prev.length + 1}`, eventType: newSub.eventType, endpointUrl: newSub.endpointUrl, enabled: true, retryCount: 0, maxRetries: newSub.maxRetries }]);
    setShowForm(false);
    setNewSub({ eventType: '', endpointUrl: '', secret: '', maxRetries: 3 });
  };

  const toggleSub = (id: string) => setSubscriptions(prev => prev.map(s => s.id === id ? { ...s, enabled: !s.enabled } : s));

  const statusColor = (s: string): string =>
    s === 'success' ? 'bg-green-100 text-green-700' : s === 'retry' ? 'bg-amber-100 text-amber-700' : 'bg-red-100 text-red-700';

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Webhook Subscription Configuration</h1>
          <p className="text-gray-600">Manage event subscriptions, retry policies, and delivery monitoring.</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Add Subscription'}</button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Add Subscription</h2>
          <div>
            <label className="text-sm font-medium">Event Type</label>
            <select value={newSub.eventType} onChange={e => setNewSub(prev => ({ ...prev, eventType: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1">
              <option value="">Select event...</option>
              {eventCatalog.map(e => <option key={e} value={e}>{e}</option>)}
            </select>
          </div>
          <div>
            <label className="text-sm font-medium">Endpoint URL</label>
            <input type="url" placeholder="https://app.example.com/webhook" value={newSub.endpointUrl} onChange={e => setNewSub(prev => ({ ...prev, endpointUrl: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
          </div>
          <div>
            <label className="text-sm font-medium">HMAC Secret</label>
            <input type="password" placeholder="Shared secret for payload signing" value={newSub.secret} onChange={e => setNewSub(prev => ({ ...prev, secret: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
          </div>
          <div>
            <label className="text-sm font-medium">Max Retries</label>
            <input type="number" min={0} max={10} value={newSub.maxRetries} onChange={e => setNewSub(prev => ({ ...prev, maxRetries: parseInt(e.target.value) || 3 }))} className="w-24 border rounded px-2 py-1 text-sm mt-1" />
          </div>
          <button onClick={addSub} disabled={!newSub.eventType || !newSub.endpointUrl} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Add</button>
        </section>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">Event Type</th>
              <th className="p-3">Endpoint</th>
              <th className="p-3">Enabled</th>
              <th className="p-3">Retries</th>
              <th className="p-3">Max</th>
              <th className="p-3">Action</th>
            </tr>
          </thead>
          <tbody>
            {subscriptions.map(s => (
              <tr key={s.id} className="border-b">
                <td className="p-3 font-mono text-xs">{s.eventType}</td>
                <td className="p-3 font-mono text-xs text-gray-500 truncate max-w-xs">{s.endpointUrl}</td>
                <td className="p-3"><label className="flex items-center"><input type="checkbox" checked={s.enabled} onChange={() => toggleSub(s.id)} className="rounded" /></label></td>
                <td className="p-3">{s.retryCount > 0 ? <span className="text-amber-600 text-xs">{s.retryCount}</span> : <span className="text-gray-400 text-xs">0</span>}</td>
                <td className="p-3">{s.maxRetries}</td>
                <td className="p-3"><button className="text-blue-600 text-xs hover:underline">Test</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Delivery History</h2>
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left"><th className="p-3">Subscription</th><th className="p-3">Status</th><th className="p-3">Code</th><th className="p-3">Timestamp</th></tr>
          </thead>
          <tbody>
            {deliveries.map(d => (
              <tr key={d.id} className="border-b">
                <td className="p-3 font-mono text-xs">{d.subscription}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(d.status)}`}>{d.status}</span></td>
                <td className="p-3 font-mono text-xs">{d.statusCode || '-'}</td>
                <td className="p-3 text-gray-500">{d.timestamp}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}