'use client';
import { useState } from 'react';

interface Delegation { id: string; delegator: string; delegatee: string; scopes: string[]; created: string; expires: string; maxDepth: number; }

export default function DelegationManagementPage() {
  const [delegations, setDelegations] = useState<Delegation[]>([
    { id: 'd1', delegator: 'admin@ggid.io', delegatee: 'alice@ggid.io', scopes: ['read:users', 'write:users'], created: '2026-07-10', expires: '2026-07-15', maxDepth: 2 },
    { id: 'd2', delegator: 'alice@ggid.io', delegatee: 'bob@ggid.io', scopes: ['read:users'], created: '2026-07-11', expires: '2026-07-14', maxDepth: 1 },
  ]);
  const [showForm, setShowForm] = useState(false);
  const [newDelegation, setNewDelegation] = useState({ delegatee: '', scopes: '', maxDepth: 2, expiryDays: 7 });
  const [selfError, setSelfError] = useState('');
  const [showChain, setShowChain] = useState(false);

  const create = () => {
    if (newDelegation.delegatee === 'admin@ggid.io') { setSelfError('Cannot delegate to yourself'); return; }
    setSelfError('');
    setDelegations(prev => [...prev, { id: `d${prev.length + 1}`, delegator: 'admin@ggid.io', delegatee: newDelegation.delegatee, scopes: newDelegation.scopes.split(',').map(s => s.trim()).filter(Boolean), created: new Date().toISOString().slice(0, 10), expires: new Date(Date.now() + newDelegation.expiryDays * 86400000).toISOString().slice(0, 10), maxDepth: newDelegation.maxDepth }]);
    setShowForm(false); setNewDelegation({ delegatee: '', scopes: '', maxDepth: 2, expiryDays: 7 });
  };
  const revoke = (id: string) => setDelegations(prev => prev.filter(d => d.id !== id));

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold">Delegation Management</h1><p className="text-gray-600">Manage token delegation chains with depth limits and circular prevention.</p></div>
        <div className="flex gap-2"><button onClick={() => setShowChain(!showChain)} className="px-3 py-1.5 border rounded text-sm">Chain Viewer</button><button onClick={() => setShowForm(!showForm)} className="px-4 py-2 bg-blue-600 text-white rounded text-sm">{showForm ? 'Cancel' : 'Create Delegation'}</button></div>
      </div>

      {showForm && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Create Delegation</h2>
          <div><label className="text-sm font-medium">Delegatee</label><input type="text" placeholder="user@ggid.io" value={newDelegation.delegatee} onChange={e => setNewDelegation(prev => ({ ...prev, delegatee: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1" /></div>
          {selfError && <div className="text-xs text-red-600">{selfError}</div>}
          <div><label className="text-sm font-medium">Scopes (comma-separated)</label><input type="text" placeholder="read:users, write:audit" value={newDelegation.scopes} onChange={e => setNewDelegation(prev => ({ ...prev, scopes: e.target.value }))} className="w-full border rounded px-3 py-2 text-sm mt-1 font-mono" /></div>
          <div className="grid grid-cols-2 gap-4">
            <div><label className="text-sm font-medium">Max Delegation Depth</label><input type="number" min={0} max={10} value={newDelegation.maxDepth} onChange={e => setNewDelegation(prev => ({ ...prev, maxDepth: parseInt(e.target.value) || 0 }))} className="w-20 border rounded px-2 py-1 text-sm mt-1" /></div>
            <div><label className="text-sm font-medium">Expiry (days)</label><input type="number" min={1} max={90} value={newDelegation.expiryDays} onChange={e => setNewDelegation(prev => ({ ...prev, expiryDays: parseInt(e.target.value) || 7 }))} className="w-20 border rounded px-2 py-1 text-sm mt-1" /></div>
          </div>
          <button onClick={create} disabled={!newDelegation.delegatee} className="px-4 py-2 bg-blue-600 text-white rounded text-sm disabled:opacity-50">Create</button>
        </section>
      )}

      <section className="bg-white rounded-lg shadow overflow-hidden">
        <table className="w-full text-sm"><thead className="bg-gray-50"><tr className="text-left"><th className="p-3">Delegator</th><th className="p-3">Delegatee</th><th className="p-3">Scopes</th><th className="p-3">Created</th><th className="p-3">Expires</th><th className="p-3">Max Depth</th><th className="p-3">Action</th></tr></thead>
          <tbody>{delegations.map(d => (
            <tr key={d.id} className="border-b">
              <td className="p-3 font-medium">{d.delegator}</td><td className="p-3">{d.delegatee}</td>
              <td className="p-3"><div className="flex flex-wrap gap-1">{d.scopes.map(s => <span key={s} className="px-1.5 py-0.5 bg-blue-100 text-blue-700 rounded text-xs font-mono">{s}</span>)}</div></td>
              <td className="p-3 text-gray-500">{d.created}</td><td className="p-3 text-gray-500">{d.expires}</td><td className="p-3">{d.maxDepth}</td>
              <td className="p-3"><button onClick={() => revoke(d.id)} className="text-red-600 text-xs hover:underline">Revoke</button></td>
            </tr>))}</tbody></table>
      </section>

      {showChain && (
        <section className="bg-white rounded-lg shadow p-6 space-y-4">
          <h2 className="text-lg font-semibold">Delegation Chain Viewer</h2>
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm"><span className="px-3 py-1 bg-purple-100 text-purple-700 rounded">admin@ggid.io</span><span className="text-gray-400">{'->'}</span><span className="px-3 py-1 bg-blue-100 text-blue-700 rounded">alice@ggid.io</span><span className="text-gray-400">{'->'}</span><span className="px-3 py-1 bg-green-100 text-green-700 rounded">bob@ggid.io</span></div>
            <div className="text-xs text-gray-400">Chain depth: 2 (max: 2) | Scopes inherited: read:users | Circular: no</div>
          </div>
        </section>
      )}
    </div>
  );
}