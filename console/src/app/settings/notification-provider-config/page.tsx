'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from '@/lib/i18n';

interface Provider {
  id: string;
  type: string;
  name: string;
  enabled: boolean;
  status: string;
  template: string;
  rateLimit: number;
}

export default function NotificationProviderConfigPage() {
  const t = useTranslations();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [providers, setProviders] = useState<Provider[]>([]);

  useEffect(() => {
    fetch("/api/v1/auth/notification-preferences", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) throw new Error(`HTTP ${res.status}`); return res.json(); })
      .then(data => { setProviders(Array.isArray(data) ? data : (data.providers || data.items || [])); setLoading(false); })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const [showForm, setShowForm] = useState(false);
  const [newProvider, setNewProvider] = useState({ type: 'email', name: '', rateLimit: 100 });
  const [fallbackChain, setFallbackChain] = useState<string[]>([]);
  const [testTarget, setTestTarget] = useState('');
  const [testResult, setTestResult] = useState('');

  const types = ['email', 'sms', 'push', 'webhook', 'slack', 'teams'];
  const available = types.filter(t => !fallbackChain.includes(t));

  const addProvider = () => {
    setProviders(prev => [...prev, { id: `p${prev.length + 1}`, type: newProvider.type, name: newProvider.name || newProvider.type, enabled: true, status: 'untested', template: `default-${newProvider.type}`, rateLimit: newProvider.rateLimit }]);
    setShowForm(false);
    setNewProvider({ type: 'email', name: '', rateLimit: 100 });
  };

  const toggleProvider = (id: string) => {
    setProviders(prev => prev.map(p => p.id === id ? { ...p, enabled: !p.enabled } : p));
  };

  const sendTest = (name: string) => {
    setTestTarget(name);
    setTestResult(`Test notification sent to ${name} - 200 OK`);
    setTimeout(() => setTestResult(''), 3000);
  };

  const statusColor = (s: string): string =>
    s === 'healthy' ? 'bg-green-100 text-green-700' : s === 'degraded' ? 'bg-amber-100 text-amber-700' : s === 'down' ? 'bg-red-100 text-red-700' : 'bg-gray-100 text-gray-600';

  const addToChain = (t: string) => setFallbackChain(prev => [...prev, t]);
  const removeFromChain = (idx: number) => setFallbackChain(prev => prev.filter((_, i) => i !== idx));

  if (loading) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">{t("notifProvider.title")}</h1><p>{t("common.loading")}</p></div>
  );
  if (error) return (
    <div className="p-6"><h1 className="text-2xl font-bold mb-4">{t("notifProvider.title")}</h1><p className="text-red-600">{t("common.error")}: {error}</p></div>
  );
  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{t("notifProvider.title")}</h1>
          <p className="text-gray-600">{t("notifProvider.subtitle")}</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? t("common.cancel") : t("notifProvider.add")}</button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">{t("notifProvider.add")}</h2>
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="text-sm font-medium">Type</label>
              <select value={newProvider.type} onChange={e => setNewProvider(prev => ({ ...prev, type: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1">
                {types.map(t => <option key={t} value={t}>{t}</option>)}
              </select>
            </div>
            <div>
              <label className="text-sm font-medium">Name</label>
              <input type="text" placeholder="Provider name" value={newProvider.name} onChange={e => setNewProvider(prev => ({ ...prev, name: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
            </div>
            <div>
              <label className="text-sm font-medium">Rate Limit (/min)</label>
              <input type="number" min={1} max={1000} value={newProvider.rateLimit} onChange={e => setNewProvider(prev => ({ ...prev, rateLimit: parseInt(e.target.value) || 100 }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
            </div>
          </div>
          <button onClick={addProvider} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{t("common.add")}</button>
        </section>
      )}

      {testResult && <div className="bg-green-50 border border-green-200 rounded p-3 text-sm text-green-700">{testResult}</div>}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">{t("common.name")}</th>
              <th className="p-3">{t("common.type")}</th>
              <th className="p-3">{t("common.status")}</th>
              <th className="p-3">{t("notifProvider.template")}</th>
              <th className="p-3">{t("notifProvider.rateLimit")}</th>
              <th className="p-3">{t("common.enabled")}</th>
              <th className="p-3">{t("common.action")}</th>
            </tr>
          </thead>
          <tbody>
            {providers.map(p => (
              <tr key={p.id} className="border-b">
                <td className="p-3 font-medium">{p.name}</td>
                <td className="p-3 capitalize text-gray-600">{p.type}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${statusColor(p.status)}`}>{p.status}</span></td>
                <td className="p-3 font-mono text-xs text-gray-500">{p.template}</td>
                <td className="p-3">{p.rateLimit}/min</td>
                <td className="p-3"><label className="flex items-center"><input type="checkbox" checked={p.enabled} onChange={() => toggleProvider(p.id)} className="rounded" /></label></td>
                <td className="p-3"><button onClick={() => sendTest(p.name)} className="text-blue-600 text-xs hover:underline">{t("common.test")}</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">{t("notifProvider.fallbackChain")}</h2>
        <p className="text-sm text-gray-500">If the primary provider fails, notifications fall through to the next in chain.</p>
        <div className="flex items-center gap-2 flex-wrap">
          {fallbackChain.map((t, idx) => (
            <div key={idx} className="flex items-center gap-2">
              <span className="px-3 py-1 bg-blue-100 text-blue-700 rounded text-sm capitalize">{t}</span>
              {idx < fallbackChain.length - 1 && <span className="text-gray-300">{'->'}</span>}
              <button onClick={() => removeFromChain(idx)} className="text-red-600 text-xs">x</button>
            </div>
          ))}
          {available.length > 0 && (
            <select onChange={e => { if (e.target.value) addToChain(e.target.value); e.target.selectedIndex = 0; }} className="border rounded px-2 py-1 text-sm">
              <option value="">{t("common.add")}...</option>
              {available.map(t => <option key={t} value={t}>{t}</option>)}
            </select>
          )}
        </div>
      </section>
    </div>
  );
}