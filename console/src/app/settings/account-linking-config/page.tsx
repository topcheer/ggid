'use client';
import { useState, useEffect } from 'react';

interface LinkedAccount {
  id: string;
  user: string;
  provider: string;
  externalId: string;
  linkedAt: string;
  lastSync: string;
  isDuplicate: boolean;
}

export default function AccountLinkingConfigPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [accounts, setAccounts] = useState<LinkedAccount[]>([]);

  const [showForm, setShowForm] = useState(false);
  const [unlinkTarget, setUnlinkTarget] = useState<LinkedAccount | null>(null);
  const [auditLog, setAuditLog] = useState<{ timestamp: string; action: string; account: string; actor: string }[]>([]);

  useEffect(() => {
    fetch("/api/v1/identity/account-linking/config", {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" },
    })
      .then(res => { if (!res.ok) throw new Error(`HTTP ${res.status}`); return res.json(); })
      .then(data => {
        setAccounts(Array.isArray(data) ? data : (data.accounts || data.items || []));
        setAuditLog(data.auditLog || []);
        setLoading(false);
      })
      .catch(err => { setError(err.message); setLoading(false); });
  }, []);
  const [newLink, setNewLink] = useState({ user: '', provider: 'Google', externalId: '' });

  const providers = ['Google', 'Microsoft', 'GitHub', 'Apple', 'SAML'];

  const providerColor = (p: string): string =>
    p === 'Google' ? 'bg-red-100 text-red-700' :
    p === 'Microsoft' ? 'bg-blue-100 text-blue-700' :
    p === 'GitHub' ? 'bg-gray-200 text-gray-700' :
    p === 'Apple' ? 'bg-gray-100 text-gray-700' :
    'bg-purple-100 text-purple-700';

  const addLink = () => {
    const isDup = accounts.some(a => a.provider === newLink.provider && a.externalId === newLink.externalId);
    setAccounts(prev => [...prev, {
      id: `la${prev.length + 1}`,
      user: newLink.user || 'unknown@ggid.io',
      provider: newLink.provider,
      externalId: newLink.externalId || `ext-${prev.length + 1}`,
      linkedAt: new Date().toISOString().slice(0, 10),
      lastSync: new Date().toISOString().slice(0, 10),
      isDuplicate: isDup,
    }]);
    setAuditLog(prev => [{ timestamp: new Date().toISOString().slice(0, 16).replace('T', ' '), action: 'Linked', account: `${newLink.user} / ${newLink.provider}`, actor: 'current-user@ggid.io' }, ...prev]);
    setShowForm(false);
    setNewLink({ user: '', provider: 'Google', externalId: '' });
  };

  const syncNow = (id: string) => {
    const now = new Date().toISOString().slice(0, 10);
    setAccounts(prev => prev.map(a => a.id === id ? { ...a, lastSync: now } : a));
  };

  const confirmUnlink = () => {
    if (unlinkTarget) {
      setAccounts(prev => prev.filter(a => a.id !== unlinkTarget.id));
      setAuditLog(prev => [{ timestamp: new Date().toISOString().slice(0, 16).replace('T', ' '), action: 'Unlinked', account: `${unlinkTarget.user} / ${unlinkTarget.provider}`, actor: 'current-user@ggid.io' }, ...prev]);
    }
    setUnlinkTarget(null);
  };

  if (loading) return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Account Linking Configuration</h1>
      <p>Loading...</p>
    </div>
  );
  if (error) return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-4">Account Linking Configuration</h1>
      <p className="text-red-600">Error: {error}</p>
    </div>
  );
  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Account Linking Configuration</h1>
          <p className="text-gray-600">Manage linked external accounts and identity provider associations.</p>
        </div>
        <button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">
          {showForm ? 'Cancel' : 'Link Account'}
        </button>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Link External Account</h2>
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="text-sm font-medium">User</label>
              <input type="text" placeholder="user@ggid.io" value={newLink.user} onChange={e => setNewLink(prev => ({ ...prev, user: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" />
            </div>
            <div>
              <label className="text-sm font-medium">Provider</label>
              <select value={newLink.provider} onChange={e => setNewLink(prev => ({ ...prev, provider: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1">
                {providers.map(p => <option key={p} value={p}>{p}</option>)}
              </select>
            </div>
            <div>
              <label className="text-sm font-medium">External ID</label>
              <input type="text" placeholder="external-id" value={newLink.externalId} onChange={e => setNewLink(prev => ({ ...prev, externalId: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" />
            </div>
          </div>
          <button onClick={addLink} disabled={!newLink.user} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Link Account</button>
        </section>
      )}

      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{accounts.length}</div>
          <div className="text-sm text-gray-500">Linked Accounts</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold">{new Set(accounts.map(a => a.provider)).size}</div>
          <div className="text-sm text-gray-500">Providers</div>
        </div>
        <div className="bg-white rounded-lg shadow p-4 text-center">
          <div className="text-2xl font-bold text-red-600">{accounts.filter(a => a.isDuplicate).length}</div>
          <div className="text-sm text-gray-500">Duplicates</div>
        </div>
      </div>

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50">
            <tr className="text-left">
              <th className="p-3">User</th>
              <th className="p-3">Provider</th>
              <th className="p-3">External ID</th>
              <th className="p-3">Linked At</th>
              <th className="p-3">Last Sync</th>
              <th className="p-3">Status</th>
              <th className="p-3">Actions</th>
            </tr>
          </thead>
          <tbody>
            {accounts.map(a => (
              <tr key={a.id} className="border-b hover:bg-gray-50">
                <td className="p-3 font-medium">{a.user}</td>
                <td className="p-3"><span className={`px-2 py-0.5 rounded text-xs ${providerColor(a.provider)}`}>{a.provider}</span></td>
                <td className="p-3 font-mono text-xs text-gray-500">{a.externalId}</td>
                <td className="p-3 text-gray-500">{a.linkedAt}</td>
                <td className="p-3 text-gray-500">{a.lastSync}</td>
                <td className="p-3">
                  {a.isDuplicate && <span className="px-2 py-0.5 bg-red-100 text-red-700 rounded text-xs">Duplicate</span>}
                  {!a.isDuplicate && <span className="px-2 py-0.5 bg-green-100 text-green-700 rounded text-xs">OK</span>}
                </td>
                <td className="p-3">
                  <div className="flex gap-2">
                    <button onClick={() => syncNow(a.id)} className="text-blue-600 text-xs hover:underline">Sync</button>
                    <button onClick={() => setUnlinkTarget(a)} className="text-red-600 text-xs hover:underline">Unlink</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="bg-white rounded-lg shadow p-6 space-y-4">
        <h2 className="text-lg font-semibold">Audit Log</h2>
        <div className="space-y-2">
          {auditLog.map((entry, idx) => (
            <div key={idx} className="flex items-center gap-3 text-sm border-b pb-2">
              <span className="text-xs text-gray-400 w-32">{entry.timestamp}</span>
              <span className="px-2 py-0.5 bg-gray-100 rounded text-xs">{entry.action}</span>
              <span className="font-mono text-xs flex-1">{entry.account}</span>
              <span className="text-xs text-gray-500">{entry.actor}</span>
            </div>
          ))}
        </div>
      </section>

      {unlinkTarget && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4 space-y-4">
            <h2 className="text-lg font-semibold">Unlink Account</h2>
            <p className="text-sm text-gray-600">Unlink <strong>{unlinkTarget.provider}</strong> account for <strong>{unlinkTarget.user}</strong>? The user will lose access via this provider.</p>
            <div className="flex justify-end gap-3">
              <button onClick={() => setUnlinkTarget(null)} className="px-4 py-2 border rounded text-sm">Cancel</button>
              <button onClick={confirmUnlink} className="px-4 py-2 bg-red-600 text-white rounded text-sm">Confirm Unlink</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}