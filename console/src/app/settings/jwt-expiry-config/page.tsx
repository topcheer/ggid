'use client';
import { useState, useEffect } from 'react';

interface ClientOverride {
  clientId: string;
  accessTokenExpiry: number;
  refreshTokenExpiry: number;
}

export default function JwtExpiryConfigPage() {
  const [accessExpiry, setAccessExpiry] = useState(15);
  const [refreshExpiry, setRefreshExpiry] = useState(43200);
  const [idTokenExpiry, setIdTokenExpiry] = useState(60);
  const [agentTokenExpiry, setAgentTokenExpiry] = useState(3600);
  const [slidingWindow, setSlidingWindow] = useState(true);
  const [maxRefreshCount, setMaxRefreshCount] = useState(100);
  const [warningThreshold, setWarningThreshold] = useState(300);
  const [overrides, setOverrides] = useState<ClientOverride[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showAdd, setShowAdd] = useState(false);
  const [newOverride, setNewOverride] = useState({ clientId: '', accessTokenExpiry: 15, refreshTokenExpiry: 43200 });

  const addOverride = () => {
    setOverrides(prev => [...prev, newOverride]);
    setShowAdd(false);
    setNewOverride({ clientId: '', accessTokenExpiry: 15, refreshTokenExpiry: 43200 });
  };
  const removeOverride = (clientId: string) => setOverrides(prev => prev.filter(o => o.clientId !== clientId));

  const fmt = (s: number): string => s < 60 ? `${s}s` : s < 3600 ? `${Math.round(s / 60)}min` : s < 86400 ? `${Math.round(s / 3600)}h` : `${Math.round(s / 86400)}d`;

  useEffect(() => {
    fetch('/api/v1/auth/expiry-status', {
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': '00000000-0000-0000-0000-000000000001' },
    })
      .then(r => { if (!r.ok) throw new Error(`HTTP ${r.status}`); return r.json(); })
      .then(data => {
        if (data) {
          if (data.access_expiry) setAccessExpiry(data.access_expiry);
          if (data.refresh_expiry) setRefreshExpiry(data.refresh_expiry);
          if (data.id_token_expiry) setIdTokenExpiry(data.id_token_expiry);
          if (data.agent_token_expiry) setAgentTokenExpiry(data.agent_token_expiry);
          if (data.sliding_window !== undefined) setSlidingWindow(data.sliding_window);
          if (data.max_refresh_count) setMaxRefreshCount(data.max_refresh_count);
          if (data.warning_threshold) setWarningThreshold(data.warning_threshold);
          if (data.overrides) setOverrides(data.overrides);
        }
        setLoading(false);
      })
      .catch(e => { setError(e.message); setLoading(false); });
  }, []);

  if (loading) return <div className="p-6"><p>Loading...</p></div>;
  if (error) return <div className="p-6 text-red-600">Error: {error}</div>;

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      <div>
        <h1 className="text-2xl font-bold">JWT Expiry Configuration</h1>
        <p className="text-gray-600">Configure token lifetimes, sliding windows, and per-client overrides.</p>
      </div>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Token Lifetimes</h2>
        <div className="grid grid-cols-2 gap-6">
          <div>
            <label className="text-sm font-medium">Access Token: {fmt(accessExpiry)}</label>
            <input type="range" min={1} max={3600} value={accessExpiry} onChange={e => setAccessExpiry(parseInt(e.target.value))} className="w-full mt-2" />
            <div className="flex justify-between text-xs text-gray-400"><span>1s</span><span>1h</span></div>
          </div>
          <div>
            <label className="text-sm font-medium">Refresh Token: {fmt(refreshExpiry)}</label>
            <input type="range" min={3600} max={2592000} step={3600} value={refreshExpiry} onChange={e => setRefreshExpiry(parseInt(e.target.value))} className="w-full mt-2" />
            <div className="flex justify-between text-xs text-gray-400"><span>1h</span><span>30d</span></div>
          </div>
          <div>
            <label className="text-sm font-medium">ID Token: {fmt(idTokenExpiry)}</label>
            <input type="range" min={5} max={600} step={5} value={idTokenExpiry} onChange={e => setIdTokenExpiry(parseInt(e.target.value))} className="w-full mt-2" />
            <div className="flex justify-between text-xs text-gray-400"><span>5s</span><span>10min</span></div>
          </div>
          <div>
            <label className="text-sm font-medium">Agent Token: {fmt(agentTokenExpiry)}</label>
            <input type="range" min={300} max={86400} step={300} value={agentTokenExpiry} onChange={e => setAgentTokenExpiry(parseInt(e.target.value))} className="w-full mt-2" />
            <div className="flex justify-between text-xs text-gray-400"><span>5min</span><span>24h</span></div>
          </div>
        </div>
      </section>

      <div className="grid grid-cols-3 gap-4">
        <label className="flex items-center justify-between bg-white rounded-lg shadow p-4">
          <span className="text-sm font-medium">Sliding Window</span>
          <input type="checkbox" checked={slidingWindow} onChange={e => setSlidingWindow(e.target.checked)} className="rounded" />
        </label>
        <div className="bg-white rounded-lg shadow p-4">
          <label className="text-sm font-medium">Max Refresh Count</label>
          <input type="number" min={1} max={10000} value={maxRefreshCount} onChange={e => setMaxRefreshCount(parseInt(e.target.value) || 100)} className="w-full border rounded px-2 py-1 text-sm mt-1" />
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <label className="text-sm font-medium">Expiry Warning (s before)</label>
          <input type="number" min={10} max={3600} value={warningThreshold} onChange={e => setWarningThreshold(parseInt(e.target.value) || 300)} className="w-full border rounded px-2 py-1 text-sm mt-1" />
        </div>
      </div>

      {slidingWindow && <p className="text-xs text-gray-400">Sliding window: each token refresh extends the refresh token lifetime by its original expiry. Max {maxRefreshCount} refreshes allowed.</p>}

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Per-Client Overrides</h2>
          <button onClick={() => setShowAdd(!showAdd)} className="px-3 py-1 bg-blue-600 text-white rounded text-sm">{showAdd ? 'Cancel' : 'Add Override'}</button>
        </div>
        {showAdd && (
          <div className="flex gap-3 border rounded p-3">
            <input type="text" placeholder="client-id" value={newOverride.clientId} onChange={e => setNewOverride(prev => ({ ...prev, clientId: e.target.value }))} className="flex-1 border rounded px-2 py-1 text-sm font-mono" />
            <input type="number" placeholder="access (s)" value={newOverride.accessTokenExpiry} onChange={e => setNewOverride(prev => ({ ...prev, accessTokenExpiry: parseInt(e.target.value) || 15 }))} className="w-28 border rounded px-2 py-1 text-sm" />
            <input type="number" placeholder="refresh (s)" value={newOverride.refreshTokenExpiry} onChange={e => setNewOverride(prev => ({ ...prev, refreshTokenExpiry: parseInt(e.target.value) || 43200 }))} className="w-28 border rounded px-2 py-1 text-sm" />
            <button onClick={addOverride} disabled={!newOverride.clientId} className="px-3 py-1 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Add</button>
          </div>
        )}
        <table className="w-full text-sm">
          <thead className="bg-gray-50"><tr className="text-left"><th className="p-3">Client ID</th><th className="p-3">Access Token</th><th className="p-3">Refresh Token</th><th className="p-3">Action</th></tr></thead>
          <tbody>
            {overrides.map(o => (
              <tr key={o.clientId} className="border-b">
                <td className="p-3 font-mono text-xs">{o.clientId}</td>
                <td className="p-3">{fmt(o.accessTokenExpiry)}</td>
                <td className="p-3">{fmt(o.refreshTokenExpiry)}</td>
                <td className="p-3"><button onClick={() => removeOverride(o.clientId)} className="text-red-600 text-xs hover:underline">Remove</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}