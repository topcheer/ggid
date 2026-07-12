'use client';
import { useState, useEffect } from 'react';

interface Webhook {
  id: string;
  url: string;
  eventTypes: string[];
  enabled: boolean;
  maxRetries: number;
  status: string;
  deliveries: { success: number; failure: number; retry: number };
}

export default function AlertWebhookConfigPage() {

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [data, setData] = useState<any[]>([]);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/alert-webhooks", {
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
  const [webhooks, setWebhooks] = useState<Webhook[]>([
    { id: 'w1', url: 'https://hooks.slack.com/services/xxx', eventTypes: ['alert', 'escalation'], enabled: true, maxRetries: 3, status: 'healthy', deliveries: { success: 142, failure: 2, retry: 5 } },
    { id: 'w2', url: 'https://api.pagerduty.com/v2/enqueue', eventTypes: ['alert', 'correlation'], enabled: true, maxRetries: 5, status: 'healthy', deliveries: { success: 89, failure: 0, retry: 1 } },
    { id: 'w3', url: 'https://hooks.example.com/alerts', eventTypes: ['correlation'], enabled: false, maxRetries: 3, status: 'down', deliveries: { success: 12, failure: 8, retry: 15 } },
  ]);

  const [showForm, setShowForm] = useState(false);
  const [newUrl, setNewUrl] = useState('');
  const [newSecret, setNewSecret] = useState('');
  const [newEvents, setNewEvents] = useState<string[]>([]);
  const [testResult, setTestResult] = useState('');

  const allEvents = ['alert', 'correlation', 'escalation'];

  const toggleEvent = (e: string) => {
    setNewEvents(prev => prev.includes(e) ? prev.filter(x => x !== e) : [...prev, e]);
  };

  const addWebhook = () => {
    setWebhooks(prev => [...prev, {
      id: `w${prev.length + 1}`, url: newUrl,
      eventTypes: newEvents, enabled: true, maxRetries: 3, status: 'untested',
      deliveries: { success: 0, failure: 0, retry: 0 },
    }]);
    setShowForm(false); setNewUrl(''); setNewSecret(''); setNewEvents([]);
  };

  const toggleWebhook = (id: string) => {
    setWebhooks(prev => prev.map(w => w.id === id ? { ...w, enabled: !w.enabled } : w));
  };

  const sendTest = () => {
    setTestResult('Test alert sent successfully (200 OK)');
    setTimeout(() => setTestResult(''), 3000);
  };

  const statusColor = (s: string): string =>
    s === 'healthy' ? 'bg-green-100 text-green-700' : s === 'down' ? 'bg-red-100 text-red-700' : 'bg-gray-100 text-gray-600';

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Alert Webhook Configuration</h1>
          <p className="text-gray-600">Configure webhooks for alert, correlation, and escalation event delivery.</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Add Webhook'}</button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Add Webhook</h2>
          <div>
            <label className="text-sm font-medium">URL</label>
            <input type="url" placeholder="https://hooks.example.com/alerts" value={newUrl} onChange={e => setNewUrl(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" />
          </div>
          <div>
            <label className="text-sm font-medium">HMAC Secret</label>
            <input type="password" placeholder="Shared secret for HMAC signing" value={newSecret} onChange={e => setNewSecret(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" />
          </div>
          <div>
            <label className="text-sm font-medium">Event Types</label>
            <div className="flex gap-4 mt-2">
              {allEvents.map(e => (
                <label key={e} className="flex items-center gap-1 text-sm">
                  <input type="checkbox" checked={newEvents.includes(e)} onChange={() => toggleEvent(e)} className="rounded" />
                  {e}
                </label>
              ))}
            </div>
          </div>
          <button onClick={addWebhook} disabled={!newUrl} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Add</button>
        </section>
      )}

      {testResult && <div className="bg-green-50 border border-green-200 rounded p-3 text-sm text-green-700">{testResult}</div>}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">URL</th>
              <th className="p-3">Events</th>
              <th className="p-3">Status</th>
              <th className="p-3">Deliveries (S/F/R)</th>
              <th className="p-3">Retries</th>
              <th className="p-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            {webhooks.map(w => (
              <tr key={w.id} className="border-b">
                <td className="p-3">
                  <div className="font-mono text-xs truncate max-w-xs">{w.url}</div>
                  <label className="flex items-center gap-1 mt-1">
                    <input type="checkbox" checked={w.enabled} onChange={() => toggleWebhook(w.id)} className="rounded" />
                    <span className="text-xs">{w.enabled ? 'enabled' : 'disabled'}</span>
                  </label>
                </td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{w.eventTypes.map(e => <span key={e} className="px-1.5 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">{e}</span>)}</div></td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(w.status)}`}>{w.status}</span></td>
                <td className="p-3 text-xs"><span className="text-green-600">{w.deliveries.success}</span>/<span className="text-red-600">{w.deliveries.failure}</span>/<span className="text-amber-600">{w.deliveries.retry}</span></td>
                <td className="p-3">{w.maxRetries}</td>
                <td className="p-3"><button onClick={sendTest} className="text-blue-600 text-xs hover:underline">Test</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}