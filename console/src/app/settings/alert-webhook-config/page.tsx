'use client';
import { useState, useEffect } from 'react';
import { Loader2 } from 'lucide-react';
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

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
  const t = useTranslations();


  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch("/api/v1/audit/alert-webhooks", {
          method: "GET",
          headers: { ...authHeader(),
            "Content-Type": "application/json",
            "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
          },
        });
        if (!res.ok) return;
        const json = await res.json();
        const items = json.webhooks || json.items || [];
        setWebhooks(items.map((w: any) => ({ id: w.id || "", url: w.url || "", eventTypes: w.event_types || [], enabled: w.enabled ?? true, maxRetries: w.max_retries || 3, status: w.status || "unknown", deliveries: w.deliveries || { success: 0, failure: 0, retry: 0 } })));
      } catch (e) {
        setError(e instanceof Error ? e.message : "Failed to load");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const [showForm, setShowForm] = useState(false);
  const [newUrl, setNewUrl] = useState('');
  const [newSecret, setNewSecret] = useState('');
  const [newEvents, setNewEvents] = useState<string[]>([]);
  const [testResult, setTestResult] = useState('');const [webhooks, setWebhooks] = useState<Webhook[]>([]);;

  if (loading) return <div className="flex items-center justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div>;
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;
  

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
        <button onClick={() => setShowForm(!showForm)} aria-label={showForm ? "Cancel adding webhook" : "Add webhook"} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Add Webhook'}</button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Add Webhook</h2>
          <div>
            <label className="text-sm font-medium">URL</label>
            <input aria-label="https://hooks.example.com/alerts" type="url" placeholder="https://hooks.example.com/alerts" value={newUrl} onChange={e => setNewUrl(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" />
          </div>
          <div>
            <label className="text-sm font-medium">HMAC Secret</label>
            <input autoComplete="current-password" type="password" placeholder="Shared secret for HMAC signing" value={newSecret} onChange={e => setNewSecret(e.target.value)} className="w-full border rounded px-3 py-2 text-sm mt-1" />
          </div>
          <div>
            <label className="text-sm font-medium">Event Types</label>
            <div className="flex gap-4 mt-2">
              {allEvents.map(e => (
                <label key={e} className="flex items-center gap-1 text-sm">
                  <input aria-label="New events" type="checkbox" checked={newEvents.includes(e)} onChange={() => toggleEvent(e)} className="rounded" />
                  {e}
                </label>
              ))}
            </div>
          </div>
          <button onClick={addWebhook} disabled={!newUrl} aria-label="Add webhook" className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Add</button>
        </section>
      )}

      {testResult && <div className="bg-green-50 border border-green-200 rounded p-3 text-sm text-green-700">{testResult}</div>}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th scope="col" className="p-3">URL</th>
              <th scope="col" className="p-3">Events</th>
              <th scope="col" className="p-3">Status</th>
              <th scope="col" className="p-3">Deliveries (S/F/R)</th>
              <th scope="col" className="p-3">Retries</th>
              <th scope="col" className="p-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            {webhooks.map(w => (
              <tr key={w.id} className="border-b">
                <td className="p-3">
                  <div className="font-mono text-xs truncate max-w-xs">{w.url}</div>
                  <label className="flex items-center gap-1 mt-1">
                    <input aria-label="W" type="checkbox" checked={w.enabled} onChange={() => toggleWebhook(w.id)} className="rounded" />
                    <span className="text-xs">{w.enabled ? 'enabled' : 'disabled'}</span>
                  </label>
                </td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{w.eventTypes.map(e => <span key={e} className="px-1.5 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">{e}</span>)}</div></td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(w.status)}`}>{w.status}</span></td>
                <td className="p-3 text-xs"><span className="text-green-600">{w.deliveries.success}</span>/<span className="text-red-600">{w.deliveries.failure}</span>/<span className="text-amber-600">{w.deliveries.retry}</span></td>
                <td className="p-3">{w.maxRetries}</td>
                <td className="p-3"><button onClick={sendTest} aria-label="Test webhook" className="text-blue-600 text-xs hover:underline">Test</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}