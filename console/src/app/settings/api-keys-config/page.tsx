'use client';
import { useState, useEffect } from 'react';
import { useTranslations } from "@/lib/i18n";

interface ApiKey {
  id: string;
  keyId: string;
  name: string;
  created: string;
  scopes: string[];
  expires: string;
  rateLimit: number;
  ipRestriction: string;
}

export default function ApiKeysConfigPage() {
  const t = useTranslations();

  const [keys, setKeys] = useState<ApiKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [newKey, setNewKey] = useState({ name: '', scopes: [] as string[], expires: '', rateLimit: 1000, ipRestriction: 'any' });
  const [auditLog, setAuditLog] = useState<{ time: string; action: string; key: string; endpoint: string }[]>([]);

  useEffect(() => {
    fetch("/api/v1/identity/api-keys", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) throw new Error(`HTTP ${res.status}`); return res.json(); })
      .then(data => {
        const items = data.keys || data.items || [];
        setKeys(items);
        setAuditLog(data.audit_log || []);
        setLoading(false);
      })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);

  const allScopes = ['read:users', 'write:users', 'read:orgs', 'write:orgs', 'read:audit', 'write:audit', 'read:policy', 'admin:all'];
  const toggleScope = (s: string) => setNewKey(prev => ({ ...prev, scopes: prev.scopes.includes(s) ? prev.scopes.filter(x => x !== s) : [...prev.scopes, s] }));
  const createKey = () => {
    const id = `gak_${Math.random().toString(36).slice(2, 8)}`;
    setKeys(prev => [...prev, { id: `k${prev.length + 1}`, keyId: id, name: newKey.name, created: new Date().toISOString().slice(0, 10), scopes: newKey.scopes, expires: newKey.expires || '2027-01-01', rateLimit: newKey.rateLimit, ipRestriction: newKey.ipRestriction }]);
    setShowForm(false); setNewKey({ name: '', scopes: [], expires: '', rateLimit: 1000, ipRestriction: 'any' });
  };
  const revoke = (id: string) => setKeys(prev => prev.filter(k => k.id !== id));

  if (loading) return <div className="p-6"><h1 className="text-2xl font-bold">API Keys Configuration</h1><p className="text-gray-600 mt-2">Loading...</p></div>;
  if (error) return <div className="p-6"><h1 className="text-2xl font-bold">API Keys Configuration</h1><p className="text-red-600 mt-2">Error: {error}</p></div>;

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold">API Keys Configuration</h1><p className="text-gray-600">Manage API keys with scoped access, rate limits, and IP restrictions.</p></div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Create Key'}</button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Create API Key</h2>
          <div><label className="text-sm font-medium">Name</label><input type="text" value={newKey.name} onChange={e => setNewKey(prev => ({ ...prev, name: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          <div><label className="text-sm font-medium">Scopes</label><div className="flex flex-wrap gap-2 mt-2">{allScopes.map(s => <label key={s} className="flex items-center gap-1 text-sm"><input type="checkbox" checked={newKey.scopes.includes(s)} onChange={() => toggleScope(s)} className="rounded" />{s}</label>)}</div></div>
          <div className="grid grid-cols-3 gap-4">
            <div><label className="text-sm font-medium">Expires</label><input type="date" value={newKey.expires} onChange={e => setNewKey(prev => ({ ...prev, expires: e.target.value }))} className="w-full border rounded px-2 py-1 text-sm mt-1" /></div>
            <div><label className="text-sm font-medium">Rate Limit (req/min)</label><input type="number" min={1} value={newKey.rateLimit} onChange={e => setNewKey(prev => ({ ...prev, rateLimit: parseInt(e.target.value) || 1000 }))} className="w-full border rounded px-2 py-1 text-sm mt-1" /></div>
            <div><label className="text-sm font-medium">IP Restriction</label><input type="text" value={newKey.ipRestriction} onChange={e => setNewKey(prev => ({ ...prev, ipRestriction: e.target.value }))} className="w-full border rounded px-2 py-1 text-sm mt-1" /></div>
          </div>
          <button onClick={createKey} disabled={!newKey.name} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Create</button>
        </section>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50"><tr className="text-left"><th className="p-3">Key ID</th><th className="p-3">Name</th><th className="p-3">Scopes</th><th className="p-3">Expires</th><th className="p-3">Rate</th><th className="p-3">IP</th><th className="p-3">Action</th></tr></thead>
          <tbody>
            {keys.length === 0 ? <tr><td colSpan={7} className="p-6 text-center text-gray-500">No API keys configured.</td></tr> :
            keys.map(k => (
              <tr key={k.id} className="border-b">
                <td className="p-3 font-mono text-xs">{k.keyId}</td>
                <td className="p-3 font-medium">{k.name}</td>
                <td className="p-3"><div className="flex flex-wrap gap-1">{k.scopes.map(s => <span key={s} className="px-1.5 py-0.5 bg-blue-100 text-blue-700 rounded text-xs">{s}</span>)}</div></td>
                <td className="p-3 text-gray-500">{k.expires}</td>
                <td className="p-3 text-xs">{k.rateLimit}/min</td>
                <td className="p-3 font-mono text-xs">{k.ipRestriction}</td>
                <td className="p-3"><button onClick={() => revoke(k.id)} className="text-red-600 text-xs hover:underline">Revoke</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Audit Log</h2>
        <div className="space-y-1">{auditLog.length === 0 ? <p className="text-gray-500 text-sm">No audit entries.</p> : auditLog.map((l, i) => (
          <div key={i} className="text-sm flex items-center gap-3 border-b pb-1"><span className="text-gray-500 text-xs">{l.time}</span><span className={`px-2 py-0.5 rounded text-xs ${l.action === 'Rate limited' ? 'bg-amber-100 text-amber-700' : l.action === 'Created' ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-600'}`}>{l.action}</span><span className="font-mono text-xs">{l.key}</span><span className="text-gray-500 text-xs">{l.endpoint}</span></div>
        ))}</div>
      </section>
    </div>
  );
}
